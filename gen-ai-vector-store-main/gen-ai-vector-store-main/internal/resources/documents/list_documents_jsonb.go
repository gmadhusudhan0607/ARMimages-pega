// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package documents

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	query2 "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents/sql"
	"go.uber.org/zap"
)

// ListDocuments3 lists documents using JSONB doc_attributes column for filtering
// This is the JSONB equivalent of ListDocuments2()
func (m *docManager) ListDocuments3(ctx context.Context, status string, filters []attributes.AttributeFilter) (docs []Document, err error) {
	dbQuery, err := m.buildListDocuments3SqlQuery(status, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to build list documents sql query: %s", err)
	}

	m.logger.Debug("listDocuments3 with query", zap.String("dbQuery", dbQuery))
	rows, err := m.query(ctx, dbQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		doc := Document{}
		err = rows.Scan(&doc.ID, &doc.Status, &doc.Error)
		if err != nil {
			return nil, fmt.Errorf("rows.scan error: %s", err)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// buildListDocuments3SqlQuery builds SQL query using JSONB filtering
func (m *docManager) buildListDocuments3SqlQuery(status string, attrFilters []attributes.AttributeFilter) (string, error) {

	attrWhereClause, err := m.getDocJSONBAttrsProcessingWhereClause(attrFilters, status)
	if err != nil {
		return "", fmt.Errorf("failed to get JSONB attributes where clause: %s", err)
	}

	if attrWhereClause == "" {
		if status != "" {
			status = fmt.Sprintf(" WHERE status = '%s'", status)
		}

		return fmt.Sprintf(query2.ListDocumentsWithoutFiltersSqlQueryTemplate,
			m.schemaName,
			m.prefix,
			status,
		), nil

	} else {
		if status != "" {
			status = fmt.Sprintf(" AND status = '%s'", status)
		}

		return fmt.Sprintf(query2.ListDocumentsSqlQueryTemplate,
			m.schemaName,
			m.prefix,
			attrWhereClause,
			status,
		), nil
	}
}
