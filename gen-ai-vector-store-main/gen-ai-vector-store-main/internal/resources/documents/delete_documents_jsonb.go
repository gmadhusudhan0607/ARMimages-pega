// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package documents

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	documents_sql "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents/sql"
	"go.uber.org/zap"
)

// DeleteDocumentsByFilters3 deletes documents using JSONB doc_attributes column for filtering
// This is the JSONB equivalent of DeleteDocumentsByFilters()
func (m *docManager) DeleteDocumentsByFilters3(ctx context.Context, attrFilters []attributes.AttributeFilter) (int64, error) {
	if len(attrFilters) == 0 {
		return 0, fmt.Errorf("no attributes provided to filter documents")
	}

	dbQuery, err := m.buildDeleteDocuments3SqlQuery(attrFilters)
	if err != nil {
		return 0, fmt.Errorf("failed to build delete documents: %w", err)
	}

	m.logger.Debug("deleteDocuments3 with query", zap.String("dbQuery", dbQuery))

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

// buildDeleteDocuments3SqlQuery builds SQL query using JSONB filtering
func (m *docManager) buildDeleteDocuments3SqlQuery(attrFilters []attributes.AttributeFilter) (string, error) {
	attrWhereClause, err := m.getDocJSONBAttrsProcessingWhereClause(attrFilters, "")
	if err != nil {
		return "", fmt.Errorf("failed to get JSONB attributes where clause: %s", err)
	}

	return fmt.Sprintf(documents_sql.DeleteDocumentsSqlQueryTemplate,
		m.schemaName,
		m.prefix,
		attrWhereClause,
	), nil
}
