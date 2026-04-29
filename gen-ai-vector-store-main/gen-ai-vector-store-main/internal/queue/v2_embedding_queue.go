/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.uber.org/zap"
)

var (
	// MaxRetryCount is the maximum number of retries for a failed embedding queue item
	// Default: 300 retries => 300*300 sec => max 25 hours
	// Can be configured via EMBEDDING_QUEUE_MAX_RETRY_COUNT environment variable
	MaxRetryCount = getMaxRetryCount()
)

func getMaxRetryCount() int {
	envValue := helpers.GetEnvOrDefault("EMBEDDING_QUEUE_MAX_RETRY_COUNT", "300")
	count, err := strconv.Atoi(envValue)
	if err != nil || count <= 0 {
		return 300 // Default value if parsing fails or invalid value
	}
	return count
}

type EmbeddingQueue2 struct {
	queue     queue2
	tableName string
}

func NewEmbeddingQueue2(ctx context.Context, database db.Database) EmbeddingQueue2 {
	return EmbeddingQueue2{
		queue: queue2{
			ctx:      ctx,
			database: database,
		},
		tableName: "vector_store.embedding_queue",
	}
}

func (e *EmbeddingQueueItem) GetEmbPath2() string {
	return fmt.Sprintf("%s/%s/%s/%s", e.IsolationID, e.CollectionID, e.DocumentID, e.EmbeddingID)
}

func (e *EmbeddingQueueItem) GetDocPath2() string {
	return fmt.Sprintf("%s/%s/%s", e.IsolationID, e.CollectionID, e.DocumentID)
}

func (eq *EmbeddingQueue2) Get2() (*EmbeddingQueueItem, error) {
	qItem, err := eq.queue.Get2()
	if err != nil {
		return nil, fmt.Errorf("failed to get item from embedding queue: %w", err)
	}
	eqi := &EmbeddingQueueItem{}

	err = json.Unmarshal([]byte(qItem.Content), &eqi)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal to EmbeddingQueueItem [%s]: %w", qItem.Content, err)
	}

	if !eqi.isValid() {
		return nil, ErrInvalidEntry
	}

	emb, err := eq.getEmbeddingData2(eqi)
	if err != nil {
		logger.Warn("error while reading embedding from DB", zap.Error(err))
		return nil, err
	}
	eqi.Data = emb

	return eqi, nil
}

func (eq *EmbeddingQueue2) Put2(item *EmbeddingQueueItem) error {
	item.Data = nil // Do not store data in queue table
	return eq.queue.Put2(item)
}

func (eq *EmbeddingQueue2) PutPostponed2(item *EmbeddingQueueItem, seconds int) error {
	item.Data = nil // Do not store data in queue table
	return eq.queue.PutPostponed2(item, seconds)
}

func (eq *EmbeddingQueue2) getEmbeddingData2(item *EmbeddingQueueItem) (*EmbeddingQueueItemData, error) {
	// check if isolation exists
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE iso_id=$1", db.GetTableIsolations())
	rows, err := eq.queue.database.GetConn().Query(query, item.IsolationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("error while reading rows from query [%s] when processing %s: %w", query, item.GetEmbPath(), err)
		}
	}
	if count == 0 {
		return nil, ErrIsolationDoesNotExist
	}

	tableEmbProcessing := db.GetTableEmbProcessing(item.IsolationID, item.CollectionID)
	tableDocProcessing := db.GetTableDocProcessing(item.IsolationID, item.CollectionID)
	dbQueryTpl := `SELECT emb.emb_id, emb.doc_id, COALESCE(emb.status, ''),  COALESCE(emb.response_code, 0), COALESCE(emb.error_message, ''), emb.content,emb.metadata,doc.doc_metadata AS doc_metadata FROM %s emb JOIN %s doc ON emb.doc_id = doc.doc_id WHERE emb.emb_id = '%s' `
	query = fmt.Sprintf(dbQueryTpl, tableEmbProcessing, tableDocProcessing, item.EmbeddingID)
	rows, err = eq.queue.database.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emb := &EmbeddingQueueItemData{}
	var embMetadataRaw, docMetadataRaw []byte
	if rows.Next() {
		err = rows.Scan(&emb.ID, &emb.DocumentID, &emb.Status, &emb.ResponseCode, &emb.ErrorMessage, &emb.Content, &embMetadataRaw, &docMetadataRaw)
		if err != nil {
			return nil, fmt.Errorf("error while reading rows from query [%s] when processing %s: %w", query, item.GetEmbPath(), err)
		}
		// Unmarshal emb.metadata
		if len(embMetadataRaw) > 0 {
			if err := json.Unmarshal(embMetadataRaw, &emb.Metadata); err != nil {
				return nil, fmt.Errorf("error while unmarshaling emb.metadata: %w", err)
			}
		}
		// Unmarshal doc.metadata (document-level metadata)
		if len(docMetadataRaw) > 0 {
			if err := json.Unmarshal(docMetadataRaw, &emb.DocMetadata); err != nil {
				return nil, fmt.Errorf("error while unmarshaling doc.metadata: %w", err)
			}
		}
	}
	return emb, nil
}

func (eq *EmbeddingQueue2) DropDocument2(isoID, colID, docID string) (int64, error) {
	dbQuery := `
       DELETE FROM vector_store.embedding_queue
               WHERE (content->'iso_id')::jsonb ? $1
                 AND (content->'col_id')::jsonb ? $2
                 AND (content->'doc_id')::jsonb ? $3
       `
	res, err := eq.queue.database.GetConn().Exec(dbQuery, isoID, colID, docID)
	if err != nil {
		return 0, fmt.Errorf("failed to execute query[%s]: %w", dbQuery, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	logger.Debug("dropped document from embedding_queue", zap.String("isoID", isoID), zap.String("colID", colID), zap.String("docID", docID))
	return rowsAffected, nil
}

// DropEmbedding2 removes a specific embedding from the queue
// This is used when an embedding reaches a terminal ERROR state and should not be retried
func (eq *EmbeddingQueue2) DropEmbedding2(isoID, colID, docID, embID string) error {
	dbQuery := `
       DELETE FROM vector_store.embedding_queue
               WHERE (content->'iso_id')::jsonb ? $1
                 AND (content->'col_id')::jsonb ? $2
                 AND (content->'doc_id')::jsonb ? $3
                 AND (content->'emb_id')::jsonb ? $4
       `
	res, err := eq.queue.database.GetConn().Exec(dbQuery, isoID, colID, docID, embID)
	if err != nil {
		return fmt.Errorf("failed to execute query[%s]: %w", dbQuery, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		logger.Debug("dropped embedding from embedding_queue",
			zap.String("isoID", isoID),
			zap.String("colID", colID),
			zap.String("docID", docID),
			zap.String("embID", embID))
	}
	return nil
}
