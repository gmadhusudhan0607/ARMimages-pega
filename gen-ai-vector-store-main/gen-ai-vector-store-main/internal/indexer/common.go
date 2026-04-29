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
	"slices"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer/processing"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"go.uber.org/zap"
)

// PrepareDocumentData handles common setup operations for both sync and async transactions
// It sets document status, updates metadata and attributes
func (i *indexer) updateDocumentMetadataTx(ctx context.Context, tx *sql.Tx, docID string, docMetadata *documents.DocumentMetadata) (err error) {

	// Update metadata
	tableDocMeta := db.GetTableDocMeta(i.IsolationID, i.CollectionName)
	err = i.dbConn.DeleteDocMetadataTx(ctx, tx, tableDocMeta, docID)
	if err != nil {
		return fmt.Errorf("error while deleting document metadata: %w", err)
	}

	if docMetadata != nil && len(docMetadata.StaticEmbeddingAttributes) > 0 {
		metadataKey := documents.MetadataKeyStaticEmbeddingAttributes
		// Create a deep copy of StaticEmbeddingAttributes to avoid modifying the original slice
		staticEmbeddingAttributes := make([]string, len(docMetadata.StaticEmbeddingAttributes))
		copy(staticEmbeddingAttributes, docMetadata.StaticEmbeddingAttributes)
		metadataValue := strings.Join(slices.Compact(staticEmbeddingAttributes), ",")
		err = i.dbConn.InsertDocMetadataTx(ctx, tx, tableDocMeta, docID, metadataKey, metadataValue)
		if err != nil {
			return fmt.Errorf("error while upserting document metadata: %w", err)
		}
	}
	return nil
}

// prepareDocumentProcessing prepares a transaction for document processing with all common setup steps.
// This function is used by both sync and async document processing flows.
// On error, the transaction is automatically rolled back and nil is returned to prevent misuse.
func (i *indexer) prepareDocumentProcessing(
	ctx context.Context,
	tx *sql.Tx,
	docID string,
	chunks []embedings.Chunk,
	docAttrs []attributes.Attribute,
	docMetadata *documents.DocumentMetadata,
	extraAttributesKinds []string,
) error {

	var docAttrIDs []int64
	var err error

	// Update attributes
	attrMgr := attributes.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)
	docAttrIDs, err = attrMgr.UpsertAttributes2(ctx, docAttrs, extraAttributesKinds)
	if err != nil {
		return fmt.Errorf("error while upserting attributes (docID: '%s', docAttrs: %v): %w", docID, docAttrs, err)
	}

	// Set document status to IN_PROGRESS
	tableDoc := db.GetTableDoc(i.IsolationID, i.CollectionName)
	err = i.dbConn.UpsertDocStatusTx(ctx, tx, tableDoc, docID, resources.StatusInProgress, "")
	if err != nil {
		err = fmt.Errorf("error while setting document '%s' status to IN_PROGRESS: %w", docID, err)
		return err
	}

	err = i.initializeDocumentProcessingTx(ctx, tx, docID, docAttrIDs, docAttrs, docMetadata)
	if err != nil {
		return fmt.Errorf("error while initializing document processing: %w", err)
	}

	// Initialize chunks using the common function
	if err := i.initializeChunkProcessingTx(ctx, tx, docID, docAttrIDs, chunks, docAttrs, extraAttributesKinds); err != nil {
		return fmt.Errorf("error while initializing chunk processing: %w", err)
	}

	return nil
}

func (i *indexer) initializeChunkProcessingTx(ctx context.Context, tx *sql.Tx, docID string, docAttrIDs []int64,
	chunks []embedings.Chunk, docAttrs []attributes.Attribute, extraAttributesKinds []string) error {
	attrMgr := attributes.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)
	procMgr := processing.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)

	// delete existing embeddings for the document
	err := procMgr.DeleteProcessingChunks(ctx, docID)
	if err != nil {
		return fmt.Errorf("error deleting items from database while deleting embedding processing: %s", err)
	}

	// for each chunk
	for _, ch := range chunks {
		// upsert chunk attributes
		embAttrIDs, err := attrMgr.UpsertAttributes2(ctx, ch.Attributes, extraAttributesKinds)
		if err != nil {
			return fmt.Errorf("error while upserting attributes for chunk '%s': %w", ch.ID, err)
		}

		err = procMgr.AddChunkToProcessing(ctx, ch, embAttrIDs, ch.Attributes, docAttrs)
		if err != nil {
			return fmt.Errorf("error while adding chunk '%s' to processing: %w", ch.ID, err)
		}
	}

	return nil
}

func (i *indexer) initializeDocumentProcessingTx(ctx context.Context, tx *sql.Tx, docID string, docAttrIDs []int64, docAttrs []attributes.Attribute, docMetadata *documents.DocumentMetadata) error {
	procMgr := processing.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)

	err := procMgr.AddDocumentToProcessing(ctx, docID, docAttrIDs, docMetadata, docAttrs)
	if err != nil {
		return fmt.Errorf("error while adding document '%s' to processing: %w", docID, err)
	}

	return nil
}

func (i *indexer) setDocumentStatus(doc documents.Document) {
	tableDoc := db.GetTableDoc(i.IsolationID, i.CollectionName)
	i.logger.Debug("setting document status", zap.String("docID", doc.ID), zap.String("status", doc.Status), zap.String("error", doc.Error))
	err := i.genericDbConn.UpsertDocStatus(context.Background(), tableDoc, doc.ID, doc.Status, doc.Error)
	if err != nil {
		i.logger.Error("error while updating document status", zap.String("docID", doc.ID), zap.Error(err))
		return
	}
}

func (i *indexer) handleAsyncFailureAndReschedule(bgCtx context.Context, docID string, errMsg string) {
	if reErr := i.putDocumentIntoQueueForReembedding(bgCtx, docID, 5); reErr != nil {
		i.logger.Error("error putting document into queue for reindexing", zap.Error(reErr))
	}
	i.setDocumentStatus(documents.Document{
		ID:     docID,
		Status: resources.StatusError,
		Error:  errMsg,
	})
}

// MoveDataToPermanentTablesTx moves embeddings and documents from processing tables to regular tables and cleans up processing tables.
func (i *indexer) MoveDataToPermanentTablesTx(ctx context.Context, tx *sql.Tx, docID string) error {
	procMgr := processing.NewManagerTx(tx, i.IsolationID, i.CollectionName, i.logger)
	tableDoc := db.GetTableDoc(i.IsolationID, i.CollectionName)

	docMetadata, docAttrIDs, docAttributesV2, err := procMgr.GetDocumentProcessingData(ctx, docID)
	if err != nil {
		return fmt.Errorf("error while getting document processing data for docID '%s': %w", docID, err)
	}

	if err := i.updateDocumentMetadataTx(ctx, tx, docID, docMetadata); err != nil {
		return err
	}

	err = i.dbConn.UpdateDocAttributesTx(ctx, tx, tableDoc, docID, docAttrIDs, docAttributesV2)
	if err != nil {
		err = fmt.Errorf("error while setting document attributes: %w", err)
		return err
	}

	err = procMgr.ReplaceChunksWithProcessing(ctx, docID)
	if err != nil {
		return fmt.Errorf("error while replacing chunks with processing for docID '%s': %w", docID, err)
	}

	chunksMeta, err := procMgr.GetChunksProcessingMetadata(ctx, docID)
	if err != nil {
		return fmt.Errorf("error while getting chunks processing metadata for docID '%s': %w", docID, err)
	}

	for _, chunk := range chunksMeta {
		if err := i.replaceChunkMetadataTx(ctx, tx, chunk); err != nil {
			return fmt.Errorf("error inserting chunk metadata: %w", err)
		}
	}

	err = procMgr.CleanupProcessing(ctx, docID)
	if err != nil {
		return fmt.Errorf("error while cleaning up processing tables for docID '%s': %w", docID, err)
	}

	i.logger.Debug("deleted processed data from processing tables for document", zap.String("docID", docID))

	return nil
}

// runInTransaction is a helper function that runs the provided function within a database transaction.
// It handles transaction commit and rollback, as well as error propagation.
// The function fn is executed with the provided context and transaction.
// if `fn` returns an error, the transaction is rolled back and the error is returned.
// if `fn` panics, the panic is recovered, the transaction is rolled back, and the panic is re-thrown.
// If `fn` completes successfully, the transaction is committed and the result of `fn` is returned.
func runInTransaction[T any](
	ctx context.Context,
	dbConn db.Database,
	logger *zap.Logger,
	fn func(ctx context.Context, tx *sql.Tx) (T, error),
) (res T, err error) {
	tx, err := dbConn.GetConn().BeginTx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("error while opening transaction; %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			logger.Error("transaction rollback error in inTx due to panic", zap.Any("panic", p))

			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				logger.Error("transaction rollback error in inTx due to panic failed", zap.Error(rollbackErr))
			}

			panic(p)
		}

		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				logger.Error("transaction rollback error in inTx ", zap.Error(rollbackErr))
				err = errors.Join(err, rollbackErr)
			}
		}
	}()

	res, err = fn(ctx, tx)
	if err != nil {
		return res, err
	}

	if commitErr := tx.Commit(); commitErr != nil {
		logger.Error("transaction commit error in inTx", zap.Error(commitErr))
		return res, commitErr
	}

	return res, nil
}

func runInTransactionNoResult(ctx context.Context, dbConn db.Database, logger *zap.Logger,
	fn func(ctx context.Context, tx *sql.Tx) error) error {
	_, err := runInTransaction(ctx, dbConn, logger,
		func(ctx context.Context, tx *sql.Tx) (any, error) {
			return nil, fn(ctx, tx)
		})
	return err
}

func (i *indexer) createDocErrorStatus(ctx context.Context, tx *sql.Tx, err error, docID string) string {
	errorMessage := (err).Error()

	docMgr := documents.NewManagerTx(tx, nil, i.IsolationID, i.CollectionName, i.logger)
	_, msg, statusErr := docMgr.CalculateDocumentStatus2(ctx, docID)
	if statusErr != nil {
		i.logger.Error("status update error", zap.Error(statusErr))
	}
	if statusErr == nil && msg != "" {
		errorMessage = msg
	}

	return errorMessage
}
