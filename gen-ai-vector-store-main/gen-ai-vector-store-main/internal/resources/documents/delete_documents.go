/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	documents_sql "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents/sql"
	"go.uber.org/zap"
)

func (m *docManager) DeleteDocumentsByFilters(ctx context.Context, attrFilters []attributes.AttributeFilter) (int64, error) {
	if len(attrFilters) == 0 {
		return 0, fmt.Errorf("no attributes provided to filter documents")
	}

	dbQuery, err := m.buildDeleteDocumentsSqlQuery(attrFilters)
	if err != nil {
		return 0, fmt.Errorf("failed to build delete documents: %w", err)
	}

	m.logger.Debug("deleteDocuments with query", zap.String("dbQuery", dbQuery))

	res, err := m.exec(ctx, dbQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to delete document: %s", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return rowsAffected, nil
}

func (m *docManager) buildDeleteDocumentsSqlQuery(attrFilters []attributes.AttributeFilter) (string, error) {
	attrWhereClause, err := m.getAttrsProcessingWhereClause(attrFilters, "")
	if err != nil {
		return "", fmt.Errorf("failed to get attributes where clause: %s", err)
	}

	return fmt.Sprintf(documents_sql.DeleteDocumentsSqlQueryTemplate,
		m.schemaName,
		m.prefix,
		attrWhereClause,
	), nil
}

func (m *docManager) DeleteDocument2(ctx context.Context, docID string) (int64, error) {
	dbQuery := fmt.Sprintf(documents_sql.DeleteDocumentSqlQueryTemplate, m.schemaName, m.prefix)
	m.logger.Debug("deleteDocument with query", zap.String("dbQuery", dbQuery))

	res, err := m.exec(ctx, dbQuery, docID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete document: %s", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return rowsAffected, nil
}
