/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
)

func (m *docManager) SetDocumentStatus(ctx context.Context, documentID, status, msg string) (err error) {
	tableDoc := db.GetTableDoc(m.IsolationID, m.CollectionID)
	query := fmt.Sprintf(`
	UPDATE %s
	SET
		status=$1,
		error_message=$2,
		modified_at=CURRENT_TIMESTAMP,
		record_timestamp=CURRENT_TIMESTAMP
	WHERE doc_id=$3
    `, tableDoc)
	result, err := m.exec(ctx, query, status, msg, documentID)
	if err != nil {
		return fmt.Errorf("failed to execute query [%s]: %w", query, err)
	}

	// Check if document exists - if no rows were affected, document doesn't exist
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrDocumentNotFound
	}

	m.logger.Info("updated document status",
		zap.String("documentID", documentID),
		zap.String("status", status),
		zap.String("msg", msg))
	return nil
}

func (m *docManager) CalculateDocumentStatus2(ctx context.Context, documentID string) (status, msg string, err error) {
	tableEmbProc := db.GetTableEmbProcessing(m.IsolationID, m.CollectionID)

	query := fmt.Sprintf(`
		SELECT 
		    (SELECT vector_store.calculate_document_status('%[1]s', $1 ) ) as status,
		    (SELECT vector_store.embedding_statuses_as_json('%[1]s', $1 )::TEXT ) as msg
        `, tableEmbProc, documentID)

	rows, err := m.query(ctx, query, documentID)
	if err != nil {
		return status, msg, fmt.Errorf("failed to execute query [%s]: %w", query, err)
	}

	defer func(rows *sql.Rows) {
		if err := rows.Close(); err != nil {
			m.logger.Error("failed to close rows",
				zap.Error(err))
		}
	}(rows)

	if rows.Next() {
		if err = rows.Scan(&status, &msg); err != nil {
			return status, msg, fmt.Errorf("error while reading rows from query [%s]: %w", query, err)
		}
	}
	return status, msg, nil
}
