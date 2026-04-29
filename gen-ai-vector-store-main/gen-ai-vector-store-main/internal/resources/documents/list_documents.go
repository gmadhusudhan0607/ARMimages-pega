/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	query2 "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents/sql"
	"go.uber.org/zap"
)

func (m *docManager) ListDocuments2(ctx context.Context, status string, filters []attributes.AttributeFilter) (docs []Document, err error) {
	dbQuery, err := m.buildListDocumentsSqlQuery(status, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to build list documents sql query: %s", err)
	}

	m.logger.Debug("listDocuments with query", zap.String("dbQuery", dbQuery))
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

func (m *docManager) buildListDocumentsSqlQuery(status string, attrFilters []attributes.AttributeFilter) (string, error) {

	attrWhereClause, err := m.getAttrsProcessingWhereClause(attrFilters, status)
	if err != nil {
		return "", fmt.Errorf("failed to get attributes where clause: %s", err)
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
