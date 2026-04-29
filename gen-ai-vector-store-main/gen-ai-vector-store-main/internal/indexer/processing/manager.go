/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package processing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

// ErrDocumentNotFoundInProcessing indicates that a document was not found in the processing table.
// This typically occurs when a document is deleted while async processing is in progress.
var ErrDocumentNotFoundInProcessing = errors.New("document not found in processing table")

type ProcManager struct {
	db           db.Database
	tx           *sql.Tx
	logger       *zap.Logger
	IsolationID  string
	CollectionID string
}

func NewManager(db db.Database, isolationID, collectionID string, logger *zap.Logger) *ProcManager {
	mgr := &ProcManager{
		db:           db,
		IsolationID:  isolationID,
		CollectionID: collectionID,
		logger:       logger,
	}
	return mgr
}

func NewManagerTx(tx *sql.Tx, isolationID, collectionID string, logger *zap.Logger) *ProcManager {
	mgr := &ProcManager{
		tx:           tx,
		IsolationID:  isolationID,
		CollectionID: collectionID,
		logger:       logger,
	}
	return mgr
}

func (m *ProcManager) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.QueryContext(ctx, query, args...)
	}

	return m.db.GetConn().QueryContext(ctx, query, args...)
}

func (m *ProcManager) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.ExecContext(ctx, query, args...)
	}

	return m.db.GetConn().ExecContext(ctx, query, args...)
}

// SetChunkEmbeddingBatch updates the embeddings and status of a batch of chunks in the database.
// It performs a batch UPDATE using a VALUES clause to efficiently set the embedding, status, and timestamps
// for multiple records identified by their chunk IDs and document IDs.
//
// If the provided chunks slice is empty, the function returns immediately with no error.
//
// Example of the generated SQL query:
//
//	UPDATE emb_processing_table AS t
//	SET embedding = c.embedding,
//	    end_time = CURRENT_TIMESTAMP,
//	    status = $N,
//	    response_code = 200,
//	    record_timestamp = CURRENT_TIMESTAMP
//	FROM (VALUES ($1, $2, $3), ($4, $5, $6), ...) AS c(emb_id, embedding, doc_id)
//	WHERE c.emb_id = t.emb_id AND t.doc_id = c.doc_id
//
// Where $1, $2, $3, ... are parameter placeholders for chunk ID, embedding vector, and document ID,
// and $N is the placeholder for the completed status value.
//
// Returns an error if the database update fails.
func (m *ProcManager) SetChunkEmbeddingBatch(ctx context.Context, chunks []embedings.Chunk) error {
	if len(chunks) == 0 {
		return nil // No chunks to update
	}

	tableEmbProcessing := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)

	// Build VALUES clause for batch update
	valuesClause := ""
	args := []any{}
	argIndex := 1

	for i, chunk := range chunks {
		if i > 0 {
			valuesClause += ", "
		}
		valuesClause += fmt.Sprintf("($%d, $%d, $%d)", argIndex, argIndex+1, argIndex+2)
		args = append(args, chunk.ID, pgvector.NewVector(chunk.Embedding), chunk.DocumentID)
		argIndex += 3
	}

	// status parameter is the next argument after all chunk triplets
	statusParam := fmt.Sprintf("$%d", argIndex)
	args = append(args, resources.StatusCompleted)

	query := fmt.Sprintf(`
		UPDATE %s AS t
		SET embedding = c.embedding::vector,
		    end_time = CURRENT_TIMESTAMP,
		    status = %s,
		    response_code = 200,
		    record_timestamp = CURRENT_TIMESTAMP
		FROM (VALUES %s) AS c(emb_id, embedding, doc_id)
		WHERE c.emb_id = t.emb_id AND t.doc_id = c.doc_id
    `, tableEmbProcessing, statusParam, valuesClause)

	_, err := m.exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute batch update query [%s]: %w", query, err)
	}

	return nil
}

func (m *ProcManager) AddDocumentToProcessing(ctx context.Context, docID string, docAttrIDs []int64, docMetadata *documents.DocumentMetadata, docAttrs []attributes.Attribute) error {
	tableDocProcessing := db.GetTableDocProcessing(m.IsolationID, m.CollectionID)
	query := fmt.Sprintf(`INSERT INTO %[1]s (doc_id, attr_ids, doc_metadata, doc_attributes, created_at, heartbeat, error_message, retry_count)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, '', 0)
		ON CONFLICT (doc_id) DO UPDATE SET
			attr_ids = $2,
			doc_metadata = $3,
			doc_attributes = $4,
			created_at = CURRENT_TIMESTAMP,
			heartbeat = CURRENT_TIMESTAMP,
			record_timestamp = CURRENT_TIMESTAMP,
			error_message = '',
			retry_count = 0
		WHERE %[1]s.doc_id = $1`, tableDocProcessing)

	metadataJSON, err := json.Marshal(docMetadata)
	if err != nil {
		return fmt.Errorf("error while marshalling document metadata: %w", err)
	}

	// Convert attributes from V1 to V2 format before marshaling
	docAttrsV2 := attributes.ConvertAttributesV1ToV2(docAttrs)
	attributesJSON, err := json.Marshal(docAttrsV2)
	if err != nil {
		return fmt.Errorf("error while marshalling document attributes: %w", err)
	}

	_, err = m.exec(ctx, query, docID, docAttrIDs, metadataJSON, attributesJSON)
	if err != nil {
		return fmt.Errorf("error while upserting document processing info for docID '%s': %w", docID, err)
	}

	return nil
}

func (m *ProcManager) DeleteProcessingChunks(ctx context.Context, docID string) error {
	tableEmbProcessing := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)
	query := fmt.Sprintf(`DELETE FROM %s WHERE doc_id = $1`, tableEmbProcessing)

	res, err := m.exec(ctx, query, docID)
	if err != nil {
		return fmt.Errorf("failed to execute query [%s]: %w", query, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected != 0 {
		m.logger.Debug(
			"successfully deleted chunks from embeddings processing table",
			zap.Int64("deleted_count", rowsAffected),
		)
	}

	return nil
}

func (m *ProcManager) AddChunkToProcessing(ctx context.Context, chunk embedings.Chunk, chunkAttrIDs []int64, chunkAttrs []attributes.Attribute, docAttrs []attributes.Attribute) error {
	// insert chunk processing table
	tableEmbProc := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)
	query := fmt.Sprintf(`INSERT INTO %s (
			emb_id, doc_id, content, status, attr_ids, metadata, emb_attributes, attributes, error_message, retry_count, record_timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, CURRENT_TIMESTAMP)`, tableEmbProc)

	// Marshal chunk metadata if available
	metadataJSON, err := json.Marshal(chunk.Metadata)
	if err != nil {
		return fmt.Errorf("error marshalling chunk metadata for chunk %s: %w", chunk.ID, err)
	}

	// Convert chunk attributes to V2 format and marshal
	chunkAttrsV2 := attributes.ConvertAttributesV1ToV2(chunkAttrs)
	embAttributesJSON, err := json.Marshal(chunkAttrsV2)
	if err != nil {
		return fmt.Errorf("error marshalling chunk attributes for chunk %s: %w", chunk.ID, err)
	}

	// Merge document and chunk attributes
	mergedAttrs := attributes.MergeAttributes(docAttrs, chunkAttrs)

	// Convert merged attributes to V2 format and marshal
	mergedAttrsV2 := attributes.ConvertAttributesV1ToV2(mergedAttrs)
	attributesJSON, err := json.Marshal(mergedAttrsV2)
	if err != nil {
		return fmt.Errorf("error marshalling merged attributes for chunk %s: %w", chunk.ID, err)
	}

	// Execute the insert query
	_, err = m.exec(ctx, query,
		chunk.ID,
		chunk.DocumentID,
		chunk.Content,
		resources.StatusInProgress,
		chunkAttrIDs,
		metadataJSON,
		embAttributesJSON,
		attributesJSON,
		"",
		0,
	)

	if err != nil {
		return fmt.Errorf("error inserting chunk processing for chunk %s: %w", chunk.ID, err)
	}

	return nil
}

func (m *ProcManager) GetDocumentProcessingData(ctx context.Context, docID string) (metadata *documents.DocumentMetadata, attrIds []int64, docAttributes attributes.AttributesV2, err error) {
	tableDocProc := db.GetTableDocProcessing(m.IsolationID, m.CollectionID)

	var docMetadata *documents.DocumentMetadata
	var metadataJSON []byte
	var docAttrIDs []int64
	var docAttributesJSON []byte

	query := fmt.Sprintf(`SELECT doc_metadata, attr_ids, doc_attributes FROM %s WHERE doc_id = $1`, tableDocProc)
	rows, err := m.query(ctx, query, docID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error while querying document metadata from processing table: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, nil, nil, fmt.Errorf("error while iterating document metadata rows for docID '%s': %w", docID, err)
		}
		return nil, nil, nil, fmt.Errorf("%w: %s", ErrDocumentNotFoundInProcessing, docID)
	}

	if err := rows.Scan(&metadataJSON, pq.Array(&docAttrIDs), &docAttributesJSON); err != nil {
		return nil, nil, nil, fmt.Errorf("error while scanning document metadata for docID '%s': %w", docID, err)
	}

	if len(metadataJSON) > 0 {
		docMetadata = &documents.DocumentMetadata{}
		if err := json.Unmarshal(metadataJSON, docMetadata); err != nil {
			return nil, nil, nil, fmt.Errorf("error unmarshalling document metadata for docID '%s': %w", docID, err)
		}
	}

	// Unmarshal attributes from JSON to AttributesV2
	var docAttrsV2 attributes.AttributesV2
	if len(docAttributesJSON) > 0 {
		if err := json.Unmarshal(docAttributesJSON, &docAttrsV2); err != nil {
			return nil, nil, nil, fmt.Errorf("error unmarshalling document attributes for docID '%s': %w", docID, err)
		}
	}

	return docMetadata, docAttrIDs, docAttrsV2, nil
}

func (m *ProcManager) ReplaceChunksWithProcessing(ctx context.Context, docID string) error {
	tableEmbProc := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)
	tableEmb := db.GetTableEmb(m.IsolationID, m.CollectionID)
	tableDocProc := db.GetTableDocProcessing(m.IsolationID, m.CollectionID)

	deleteOldChunksQuery := fmt.Sprintf(`DELETE FROM %s WHERE doc_id = $1`, tableEmb)
	if _, err := m.exec(ctx, deleteOldChunksQuery, docID); err != nil {
		return fmt.Errorf("error deleting old chunks from regular table: %w", err)
	}

	insertEmbQuery := fmt.Sprintf(
		`INSERT INTO %[1]s (
			emb_id, doc_id, content, embedding, status, attr_ids, attr_ids2, 
			emb_attributes, attributes, response_code, error_message, modified_at, record_timestamp
		)
		SELECT 
			p.emb_id, p.doc_id, p.content, p.embedding, p.status, p.attr_ids,
			(
				SELECT ARRAY( SELECT DISTINCT e FROM unnest(array_cat(d.attr_ids, p.attr_ids)) as a(e) ) 
				FROM %[2]s d
				WHERE d.doc_id = p.doc_id
			) as attr_ids2,
			p.emb_attributes, p.attributes,
			p.response_code, p.error_message, 
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		FROM %[3]s p
		WHERE p.doc_id = $1`,
		tableEmb, tableDocProc, tableEmbProc,
	)

	if _, err := m.exec(ctx, insertEmbQuery, docID); err != nil {
		return fmt.Errorf("error inserting embeddings into regular table: %w", err)
	}

	return nil
}

func (m *ProcManager) GetChunksProcessingMetadata(ctx context.Context, docID string) ([]embedings.Chunk, error) {
	tableEmbProc := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)

	query := fmt.Sprintf(
		`SELECT emb_id, metadata FROM %s WHERE doc_id = $1`,
		tableEmbProc)
	rows, err := m.query(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("error querying chunk metadata: %w", err)
	}
	defer rows.Close()

	chunksMeta := []embedings.Chunk{}
	for rows.Next() {
		var embID string
		var chunkMetadataJSON []byte
		if err := rows.Scan(&embID, &chunkMetadataJSON); err != nil {
			return nil, fmt.Errorf("error scanning chunk metadata row: %w", err)
		}

		var chunkMetadata embedings.ChunkMetadata
		if len(chunkMetadataJSON) > 0 {
			if err := json.Unmarshal(chunkMetadataJSON, &chunkMetadata); err != nil {
				return nil, fmt.Errorf("error unmarshalling chunk metadata: %w", err)
			}
		}

		chunk := embedings.Chunk{
			ID:         embID,
			DocumentID: docID,
			Metadata:   &chunkMetadata,
		}
		chunksMeta = append(chunksMeta, chunk)
	}

	return chunksMeta, nil
}

func (m *ProcManager) CleanupProcessing(ctx context.Context, docID string) error {
	tableEmbProc := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)
	tableDocProc := db.GetTableDocProcessing(m.IsolationID, m.CollectionID)

	query := fmt.Sprintf("DELETE FROM %s WHERE doc_id = $1", tableEmbProc)
	if _, err := m.exec(ctx, query, docID); err != nil {
		return fmt.Errorf("error deleting from embeddings processing table: %w", err)
	}

	query = fmt.Sprintf("DELETE FROM %s WHERE doc_id = $1", tableDocProc)
	if _, err := m.exec(ctx, query, docID); err != nil {
		return fmt.Errorf("error deleting from document processing table: %w", err)
	}

	return nil
}

func (m *ProcManager) GetProcessingAttributeIDs(ctx context.Context, docID string) ([]int64, error) {
	var docAttrIDs []int64
	tableDocProcessing := db.GetTableDocProcessing(m.IsolationID, m.CollectionID)

	query := fmt.Sprintf("SELECT unnest(attr_ids) FROM %s WHERE doc_id=$1", tableDocProcessing)
	rows, err := m.query(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute IDs for document '%s': %w", docID, err)
	}
	if rows.Next() {
		var attrID int64
		if err = rows.Scan(&attrID); err != nil {
			return nil, fmt.Errorf("failed to scan attribute IDs for document '%s': %w", docID, err)
		}
		docAttrIDs = append(docAttrIDs, attrID)
	}
	return docAttrIDs, nil
}

func (m *ProcManager) GetChunksProcessingContent(ctx context.Context, docID string) ([]embedings.Chunk, error) {
	var chunks []embedings.Chunk
	tableEmbProc := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)

	query := fmt.Sprintf("SELECT emb_id, content FROM %s WHERE doc_id=$1", tableEmbProc)

	rows, err := m.query(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processing chunks via query %q for document '%s': %w", query, docID, err)
	}
	defer rows.Close()

	for rows.Next() {
		chunk := embedings.Chunk{DocumentID: docID}
		if err = rows.Scan(&chunk.ID, &chunk.Content); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (m *ProcManager) GetDocumentProcessingMetadata(ctx context.Context, docID string) (*documents.DocumentMetadata, error) {
	metadataJSON := []byte("null")
	tableDocProcessing := db.GetTableDocProcessing(m.IsolationID, m.CollectionID)

	query := fmt.Sprintf(`SELECT doc_metadata FROM %s WHERE doc_id=$1`, tableDocProcessing)
	rows, err := m.query(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to query document processing metadata for %s: %w", docID, err)
	}

	if rows.Next() {
		err := rows.Scan(&metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document processing metadata for %s: %w", docID, err)
		}
	}

	docMetadata := documents.DocumentMetadata{}
	err = json.Unmarshal(metadataJSON, &docMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal doc metadata %q for %s: %w", string(metadataJSON), docID, err)
	}

	return &docMetadata, nil
}

func (m *ProcManager) SetChunkError(ctx context.Context, docID, chunkID, errorMessage string, responseCode int) error {
	tableEmbProcessing := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)
	query := fmt.Sprintf(`
		UPDATE %s
		SET end_time = CURRENT_TIMESTAMP,
		    status = $1,
		    response_code = $2,
			record_timestamp = CURRENT_TIMESTAMP,
			error_message = $3
		WHERE emb_id = $4 AND doc_id = $5
    `, tableEmbProcessing)
	_, err := m.exec(ctx, query, resources.StatusError, responseCode, errorMessage, chunkID, docID)
	if err != nil {
		return fmt.Errorf("failed to execute query [%s]: %w", query, err)
	}

	return nil
}
