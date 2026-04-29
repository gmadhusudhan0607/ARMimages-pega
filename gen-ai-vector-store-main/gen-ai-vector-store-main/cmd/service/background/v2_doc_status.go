/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package background

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/queue"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"go.uber.org/zap"
)

// docStatusUpdatePeriod is the period of time to update the status of the documents
var docStatusUpdatePeriod = time.Duration(30) * time.Second

// GetDocStatusUpdaterHandler update the status of the documents if any chunks record_timestamp is more recent than the document record_timestamp
func GetDocStatusUpdaterHandler(ctx context.Context, database db.Database) func() error {
	logger := log.GetNamedLogger("doc-status-updater")

	ms := helpers.GetEnvOrDefaultInt64("DOCUMENT_STATUS_UPDATE_PERIOD_MS", 30000)
	logger.Info("Configured document status update period", zap.Int64("milliseconds", ms))
	docStatusUpdatePeriod = time.Duration(ms) * time.Millisecond
	return func() error {
		for {
			if err := updateDocumentStatusIfEmbeddingChanged(ctx, database, logger); err != nil {
				logger.Error("failed to update document statuses", zap.Error(err))
			}
			time.Sleep(docStatusUpdatePeriod)
		}
	}
}

func updateDocumentStatusIfEmbeddingChanged(ctx context.Context, database db.Database, logger *zap.Logger) (err error) {
	defer func() {
		if err1 := recover(); err1 != nil {
			err = errors.Join(err, fmt.Errorf("failed with panic: %v", err1))
		}
	}()

	schemaMgr, err := schema.NewVsSchemaManager(database, logger).Load(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get db metadata: %w", err)
	}
	for _, c := range schemaMgr.GetCollections() {
		if err := updateDocumentStatusInCollection(ctx, database, c); err != nil {
			return fmt.Errorf("failed to update document status in collection %s / %s: %w", c.IsolationID, c.CollectionID, err)
		}
	}
	return nil
}

type docEmbeddingAggregate struct {
	DocID            string
	Status           string
	MaxEmbRetryCount int64
}

func updateDocumentStatusInCollection(ctx context.Context, database db.Database, c *schema.Collection) error {
	logger := log.GetNamedLogger("doc-status-updater").With(
		zap.String(isolationIDParamName, c.IsolationID),
		zap.String(collectionNameParamName, c.CollectionID),
	)
	tableEmbProc := db.GetTableEmbProcessing(c.IsolationID, c.CollectionID)

	tx, err := database.GetConn().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			logger.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	aggs, err := getDocEmbeddingAggregatesTx(ctx, tx, tableEmbProc)
	if err != nil {
		return fmt.Errorf("failed to get document embedding aggregates: %w", err)
	}

	docIDToStatusToAgg := make(map[string]map[string]*docEmbeddingAggregate)
	for _, item := range aggs {
		if _, exists := docIDToStatusToAgg[item.DocID]; !exists {
			docIDToStatusToAgg[item.DocID] = make(map[string]*docEmbeddingAggregate)
		}
		docIDToStatusToAgg[item.DocID][item.Status] = &item
	}

	if len(aggs) > 0 {
		// too noisy in loop, log only if we found something to process
		logger.Debug("found embedding processing records", zap.Int("count", len(aggs)))
	}

	for docID, statusToAgg := range docIDToStatusToAgg {
		logger := logger.With(zap.String(docIDParamName, docID))

		docMgr := documents.NewManagerTx(tx, nil, c.IsolationID, c.CollectionID, logger)
		indexer := indexer.NewIndexer(database, nil, nil, c.IsolationID, c.CollectionID, logger)

		_, msg, err := docMgr.CalculateDocumentStatus2(ctx, docID)
		if err != nil {
			return fmt.Errorf("error calculating document status for docID '%s': %w", docID, err)
		}

		logger.Debug("found processing records for document", zap.Any("details", statusToAgg))

		if statusToAgg[resources.StatusInProgress] != nil {
			if err = docMgr.SetDocumentStatus(ctx, docID, resources.StatusInProgress, msg); err != nil {
				if errors.Is(err, documents.ErrDocumentNotFound) {
					logger.Warn("document not found during status update, skipping", zap.String(docIDParamName, docID))
					continue
				}
				return fmt.Errorf("error update document status in progress for docID '%s': %w", docID, err)
			}
			logger.Debug("document status updated to in progress", zap.String("msg", msg))
		} else if statusToAgg[resources.StatusError] != nil {
			if statusToAgg[resources.StatusError].MaxEmbRetryCount >= int64(queue.MaxRetryCount) {
				msg = fmt.Sprintf("Max retry count reached: %s", msg)
			}

			if err = docMgr.SetDocumentStatus(ctx, docID, resources.StatusError, msg); err != nil {
				if errors.Is(err, documents.ErrDocumentNotFound) {
					logger.Warn("document not found during status update, skipping", zap.String(docIDParamName, docID))
					continue
				}
				return fmt.Errorf("error update document status when reached max retries for docID '%s': %w", docID, err)
			}
			logger.Debug("document status updated to error", zap.String("msg", msg))
		} else if statusToAgg[resources.StatusError] == nil && statusToAgg[resources.StatusInProgress] == nil {
			err := indexer.MoveDataToPermanentTablesTx(ctx, tx, docID)
			if err != nil {
				return fmt.Errorf("error while moving embeddings from processing to regular table for docID '%s': %w", docID, err)
			}
			logger.Debug("moved embeddings from processing to regular table")

			if err = docMgr.SetDocumentStatus(ctx, docID, resources.StatusCompleted, msg); err != nil {
				if errors.Is(err, documents.ErrDocumentNotFound) {
					logger.Warn("document not found during status update, skipping", zap.String(docIDParamName, docID))
					continue
				}
				return fmt.Errorf("error update document status when completed for docID '%s': %w", docID, err)
			}

			logger.Debug("document status updated to completed", zap.String("msg", msg))
		}
	}

	return tx.Commit()
}

func getDocEmbeddingAggregatesTx(ctx context.Context, tx *sql.Tx, tableEmbProc string) ([]docEmbeddingAggregate, error) {
	query := fmt.Sprintf(`
		WITH locked_rows AS (
			SELECT doc_id, status, retry_count
			FROM %s
			FOR NO KEY UPDATE SKIP LOCKED
		)
		SELECT doc_id, status, MAX(retry_count) AS max_emb_retry_count
		FROM locked_rows
		GROUP BY doc_id, status;
	`, tableEmbProc)

	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query [%s]: %w", query, err)
	}
	defer rows.Close()

	var aggregates []docEmbeddingAggregate
	for rows.Next() {
		var agg docEmbeddingAggregate
		if err := rows.Scan(&agg.DocID, &agg.Status, &agg.MaxEmbRetryCount); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		aggregates = append(aggregates, agg)
	}

	return aggregates, nil
}
