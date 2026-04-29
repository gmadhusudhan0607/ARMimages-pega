/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package indexer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer/processing"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"go.uber.org/zap"
)

func (i *indexer) processSync(
	ctx context.Context,
	docID string,
	chunks []embedings.Chunk,
	docAttrs []attributes.Attribute,
	docMetadata *documents.DocumentMetadata,
	extraAttributesKinds []string) (err error) {

	docErrStatus, txErr := runInTransaction(ctx, i.dbConn, i.logger,
		func(ctx context.Context, tx *sql.Tx) (string, error) {
			err := i.prepareDocumentProcessing(ctx, tx, docID, chunks, docAttrs, docMetadata, extraAttributesKinds)
			if err != nil {
				return i.createDocErrorStatus(ctx, tx, err, docID), err
			}

			contentToEmbed, err := i.precomputeChunkContent(ctx, tx, chunks, docMetadata)
			if err != nil {
				return i.createDocErrorStatus(ctx, tx, err, docID), err
			}

			failures := i.embedChunksInParallel(ctx, chunks, contentToEmbed)

			if len(failures) > 0 {
				i.persistChunkErrorsTx(ctx, tx, chunks, failures)
				err = fmt.Errorf("one or more chunks failed to embed: %w", failures[0].err)
				return i.createDocErrorStatus(ctx, tx, err, docID), err
			}

			procMgr := processing.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)
			err = procMgr.SetChunkEmbeddingBatch(ctx, chunks)
			if err != nil {
				return i.createDocErrorStatus(ctx, tx, err, docID), fmt.Errorf("error while inserting chunk embeddings: %w", err)
			}

			err = i.MoveDataToPermanentTablesTx(ctx, tx, docID)
			if err != nil {
				return i.createDocErrorStatus(ctx, tx, err, docID), fmt.Errorf("error while moving embeddings from processing to regular table: %w", err)
			}

			return "", nil
		},
	)

	if txErr != nil {
		i.logger.Error("transaction error", zap.Error(txErr))
		errStatus := docErrStatus
		if errStatus == "" {
			errStatus = txErr.Error()
		}
		i.setDocumentStatus(documents.Document{
			ID:     docID,
			Status: resources.StatusError,
			Error:  errStatus,
		})

		return txErr
	}

	i.setDocumentStatus(documents.Document{
		ID:     docID,
		Status: resources.StatusCompleted,
		Error:  "",
	})

	return nil
}

func (i *indexer) processAsync(
	ctx context.Context,
	docID string,
	chunks []embedings.Chunk,
	docAttrs []attributes.Attribute,
	docMetadata *documents.DocumentMetadata,
	extraAttributesKinds []string,
) (err error) {
	// Phase 1: Preparation (uses request context with existing HTTP_REQUEST_TIMEOUT)
	i.logger.Debug("starting document preparation",
		zap.String("docID", docID),
		zap.String("isolation", i.IsolationID),
		zap.String("collection", i.CollectionName),
		zap.Int("chunkCount", len(chunks)),
		zap.Int("attributeCount", len(docAttrs)),
	)

	prepStartTime := time.Now()

	// Commit all changes made in the transaction by prepareDocumentProcessing function.
	// The rest will be handled in the goroutine.
	txErr := runInTransactionNoResult(ctx, i.dbConn, i.logger,
		func(ctx context.Context, tx *sql.Tx) error {
			return i.prepareDocumentProcessing(ctx, tx, docID, chunks, docAttrs, docMetadata, extraAttributesKinds)
		},
	)
	if txErr != nil {
		return fmt.Errorf("error during transaction: %w", txErr)
	}

	i.logger.Debug("document preparation completed",
		zap.String("docID", docID),
		zap.Duration("elapsed", time.Since(prepStartTime)),
	)

	// Phase 2: Async processing (separate goroutine with separate HTTP_REQUEST_BACKGROUND_TIMEOUT)
	go func() {
		// Set async processing timeout for the context
		bgTimeout := helpers.GetAsyncProcessingTimeout()
		bgCtx := servicemetrics.WithMetrics(context.Background()) // to prevent log warning about missing metrics during background processing
		bgCtx, cancel := context.WithTimeout(bgCtx, bgTimeout)
		defer cancel()

		// Pre-compute content to embed (short-lived TX)
		contentToEmbed, precomputeErr := runInTransaction(bgCtx, i.dbConn, i.logger,
			func(ctx context.Context, tx *sql.Tx) ([]string, error) {
				return i.precomputeChunkContent(ctx, tx, chunks, docMetadata)
			},
		)
		if precomputeErr != nil {
			i.logger.Error("error pre-computing chunk content", zap.Error(precomputeErr))
			i.handleAsyncFailureAndReschedule(bgCtx, docID, precomputeErr.Error())
			return
		}

		// Embed in parallel (NO DB connection held)
		failures := i.embedChunksInParallel(bgCtx, chunks, contentToEmbed)

		// Persist results (short-lived TX)
		docErrStatus, bgTxErr := runInTransaction(bgCtx, i.dbConn, i.logger,
			func(ctx context.Context, tx *sql.Tx) (string, error) {
				// Persist any per-chunk embedding failures.
				// Note: persistChunkErrorsTx writes are rolled back because the callback
				// returns an error.  This is acceptable — createDocErrorStatus captures the
				// error message before rollback, and the document is rescheduled via the
				// re-embedding queue by handleAsyncFailureAndReschedule.
				if len(failures) > 0 {
					i.persistChunkErrorsTx(ctx, tx, chunks, failures)
					err := fmt.Errorf("one or more chunks failed to embed: %w", failures[0].err)
					return i.createDocErrorStatus(ctx, tx, err, docID), err
				}

				procMgr := processing.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)
				err := procMgr.SetChunkEmbeddingBatch(ctx, chunks)
				if err != nil {
					return i.createDocErrorStatus(ctx, tx, err, docID), fmt.Errorf("error while inserting chunk embeddings: %w", err)
				}

				err = i.MoveDataToPermanentTablesTx(ctx, tx, docID)
				if err != nil {
					return i.createDocErrorStatus(ctx, tx, err, docID), fmt.Errorf("error while moving embeddings from processing to regular table: %w", err)
				}

				return "", nil
			},
		)

		if errors.Is(bgTxErr, processing.ErrDocumentNotFoundInProcessing) {
			i.logger.Info("document was deleted during processing, skipping finalization",
				zap.String("docID", docID))
			return
		}
		if bgTxErr != nil {
			i.logger.Error("transaction error", zap.Error(bgTxErr))
			errStatus := docErrStatus
			if errStatus == "" {
				errStatus = bgTxErr.Error()
			}
			i.handleAsyncFailureAndReschedule(bgCtx, docID, errStatus)
			return
		}

		i.setDocumentStatus(documents.Document{
			ID:     docID,
			Status: resources.StatusCompleted,
			Error:  "",
		})

	}()

	return nil
}
