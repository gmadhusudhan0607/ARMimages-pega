// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package documents

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	query2 "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents/sql"
)

// GetDocumentStatuses3 gets document statuses using JSONB doc_attributes column for filtering
// This is the JSONB equivalent of GetDocumentStatuses()
func (m *docManager) GetDocumentStatuses3(ctx context.Context, status string, fields []string, filter attributes.Filter, cursor string, limit int) (documentStatuses []DocumentStatus, itemsTotal int, itemsLeft int, err error) {
	// Setting the maximum postgres integer value for limit if it is less than 1
	// This is to prevent the default value of 0 from being used, which would cause an error
	if limit < 1 {
		limit = MAX_POSTGRES_INT_VALUE
	}

	// Get total number of documents with the given status
	itemsTotal, err = m.getTotalDocumentsByStatus3(ctx, status, filter.Items)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get total documents with status '%s': %w", status, err)
	}
	if itemsTotal == 0 {
		return []DocumentStatus{}, 0, 0, nil
	}

	// Get documents with pagination and status filtering
	documentStatuses, err = m.getDocumentsByStatusPaginated3(ctx, status, filter.Items, cursor, limit)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get documents with status '%s': %w", status, err)
	}

	itemsLeft = itemsTotal - len(documentStatuses)
	return documentStatuses, itemsTotal, itemsLeft, nil
}

// getTotalDocumentsByStatus3 gets total count using JSONB filtering
func (m *docManager) getTotalDocumentsByStatus3(ctx context.Context, status string, attrFilters []attributes.AttributeFilter) (int, error) {
	whereClause, err := m.buildWhereClauseForTotalDocuments3(status, attrFilters)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(query2.CountDocumentsByStatusSqlQueryTemplate, m.tableDoc, whereClause)
	return m.executeCountQuery(ctx, query)
}

// buildWhereClauseForTotalDocuments3 builds WHERE clause using JSONB filtering
func (m *docManager) buildWhereClauseForTotalDocuments3(status string, attrFilters []attributes.AttributeFilter) (string, error) {
	attrWhereClause, err := m.getDocJSONBAttrsProcessingWhereClause(attrFilters, status)
	if err != nil {
		return "", fmt.Errorf("failed to get JSONB attributes where clause: %s", err)
	}

	if attrWhereClause == "" {
		if status != "" {
			return fmt.Sprintf(" WHERE DOC.status = '%s'", status), nil
		}
		return "", nil
	}

	if status != "" {
		return attrWhereClause + fmt.Sprintf(" AND DOC.status = '%s'", status), nil
	}
	return attrWhereClause, nil
}

// getDocumentsByStatusPaginated3 gets paginated documents using JSONB filtering
func (m *docManager) getDocumentsByStatusPaginated3(ctx context.Context, status string, attrFilters []attributes.AttributeFilter, cursor string, limit int) ([]DocumentStatus, error) {
	query, err := m.buildPaginatedQuery3(status, attrFilters, cursor, limit)
	if err != nil {
		return nil, err
	}

	rows, err := m.query(ctx, query, m.IsolationID, m.CollectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query[%s]: %w", query, err)
	}
	defer rows.Close()

	docs := []DocumentStatus{}
	for rows.Next() {
		var (
			docID, status, ingestionStart, lastSuccessIngestion, errorMessage string
			attrs, attrsProcessing                                            attributes.Attributes
			completed, pending, errored                                       int
		)

		if err := rows.Scan(
			&docID, &status, &ingestionStart, &lastSuccessIngestion, &errorMessage,
			&attrs, &attrsProcessing,
			&completed, &pending, &errored,
		); err != nil {
			return nil, fmt.Errorf("rows.scan error: %s", err)
		}

		doc := DocumentStatus{
			DocumentID:              docID,
			Status:                  status,
			IngestionStart:          ingestionStart,
			LastSuccessfulIngestion: lastSuccessIngestion,
			ErrorMessage:            errorMessage,
			ChunkStatus: map[string]int{
				resources.StatusInProgress: pending,
				resources.StatusCompleted:  completed,
				resources.StatusError:      errored,
			},
		}

		if doc.Status == resources.StatusCompleted {
			doc.DocumentAttributes = attrs
		} else {
			doc.DocumentAttributes = attrsProcessing
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// buildPaginatedQuery3 builds paginated query using JSONB filtering
func (m *docManager) buildPaginatedQuery3(status string, attrFilters []attributes.AttributeFilter, cursor string, limit int) (string, error) {
	attrWhereClause, err := m.getDocJSONBAttrsProcessingWhereClause(attrFilters, status)
	if err != nil {
		return "", fmt.Errorf("failed to get JSONB attributes where clause: %s", err)
	}

	var whereClause string
	var limitClause string

	if attrWhereClause == "" {
		if status == "" && cursor == "" {
			whereClause = ""
		} else if status == "" && cursor != "" {
			whereClause = fmt.Sprintf(" WHERE DOC.doc_id > '%s'", cursor)
		} else if status != "" && cursor != "" {
			whereClause = fmt.Sprintf(" WHERE DOC.status = '%s' AND DOC.doc_id > '%s'", status, cursor)
		}
	} else {
		whereClause = attrWhereClause
		if status != "" {
			whereClause += fmt.Sprintf(" AND DOC.status = '%s'", status)
		}
		if cursor != "" {
			whereClause += fmt.Sprintf(" AND DOC.doc_id > '%s'", cursor)
		}
	}

	limitClause = fmt.Sprintf(" LIMIT %d", limit)
	query := fmt.Sprintf(query2.ListDocumentsPaginatedSqlQueryTemplate, m.schemaName, m.prefix, whereClause, limitClause)
	m.logger.Debug("Built paginated query (JSONB)", zap.String("query", query))
	return query, nil
}
