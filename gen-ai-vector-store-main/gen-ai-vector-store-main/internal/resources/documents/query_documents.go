/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	documents_sql "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents/sql"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/filters"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

func (m *docManager) FindDocuments2(ctx context.Context, docReq *QueryDocumentsRequest) ([]*DocumentQueryResponse, error) {
	vector, _, err := m.Embedder.GetEmbedding(ctx, docReq.Filters.Query)
	if err != nil {
		return nil, err
	}

	tx, rollback, commit, err := m.getTxOrStartNew()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	defer func() {
		_ = rollback()
	}()

	// Set hnsw parameters to improve search recall
	// iterative_scan - https://github.com/pgvector/pgvector/tree/v0.8.0?tab=readme-ov-file#iterative-scan-options
	_, err = m.execTx(ctx, tx, fmt.Sprintf(`
		SET LOCAL hnsw.ef_search = %[1]d;
		SET LOCAL hnsw.iterative_scan = strict_order;
		SET LOCAL hnsw.max_scan_tuples = %[2]d;
		SET LOCAL hnsw.scan_mem_multiplier = %[3]d;
		`,
		helpers.GetEnvOrDefaultInt64(resources.PGVectorHnswEfSearchEnvVarName, resources.DefaultPGVectorHnswEfSearch),
		helpers.GetEnvOrDefaultInt64(resources.PGVectorHnswMaxScanTuplesEnvVarName, resources.DefaultHnswMaxScanTuples),
		helpers.GetEnvOrDefaultInt64(resources.PGVectorHnswScanMemMultiplierEnvVarName, resources.DefaultHnswScanMemMultiplier),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to set hnsw.ef_search parameter: %w", err)
	}

	if helpers.IsEncourageSemSearchIndexUseEnabled() {
		_, err = m.execTx(ctx, tx, `
		SET LOCAL enable_seqscan = off;
		SET LOCAL enable_bitmapscan = off;
		`)
		if err != nil {
			return nil, fmt.Errorf("failed to disabling seqscan and bitmapscan: %w", err)
		}

		m.logger.Debug("Successfully disabled seqscan and bitmapscan for semantic search index use")
	}

	// Compute limit and CTE limit with multiplier
	limit := docReq.Limit
	if limit <= 0 {
		limit = math.MaxInt32
	}

	multiplier := helpers.GetEnvOrDefaultInt64(
		resources.DocumentSemanticSearchMultiplierEnvVarName,
		resources.DefaultDocumentSemanticSearchMultiplier,
	)
	cteLimit := safeMul(limit, int(multiplier))

	// Stage 1: Execute search with multiplied CTE limit
	docs, err := m.executeDocumentSearch2(ctx, tx, docReq, vector, cteLimit, limit)
	if err != nil {
		return nil, err
	}

	// Stage 2: If first search returned fewer documents than requested, and second scan is enabled
	if len(docs) < limit && docReq.EnableSecondScan {
		maxChunks, err := m.getMaxChunksPerDoc(ctx, tx, docReq.Filters.SubFilters)
		if err != nil {
			m.logger.Warn("failed to get max chunks per doc for second scan, using first scan results",
				zap.Error(err))
			return docs, m.reEnableScansAndCommit(ctx, tx, commit)
		}

		newCteLimit := safeMul(limit, maxChunks)
		if newCteLimit > cteLimit {
			m.logger.Debug("executing second scan for document search",
				zap.Int("firstScanResults", len(docs)),
				zap.Int("requestedLimit", limit),
				zap.Int("newCteLimit", newCteLimit),
				zap.Int("maxChunksPerDoc", maxChunks))

			docs, err = m.executeDocumentSearch2(ctx, tx, docReq, vector, newCteLimit, limit)
			if err != nil {
				return nil, err
			}
		}
	}

	return docs, m.reEnableScansAndCommit(ctx, tx, commit)
}

// reEnableScansAndCommit re-enables seqscan/bitmapscan if needed and commits the transaction.
func (m *docManager) reEnableScansAndCommit(ctx context.Context, tx *sql.Tx, commit func() error) error {
	if helpers.IsEncourageSemSearchIndexUseEnabled() {
		_, err := m.execTx(ctx, tx, `
		SET LOCAL enable_seqscan = on;
		SET LOCAL enable_bitmapscan = on;
		`)
		if err != nil {
			return fmt.Errorf("failed enabling back seqscan and bitmapscan: %w", err)
		}
	}
	return commit()
}

// executeDocumentSearch2 builds and executes the legacy document search query with the given CTE and final limits.
func (m *docManager) executeDocumentSearch2(ctx context.Context, tx *sql.Tx, docReq *QueryDocumentsRequest, vector []float32, cteLimit int, limit int) ([]*DocumentQueryResponse, error) {
	reqCopy := *docReq
	reqCopy.Limit = limit

	dbQuery, err := m.buildFindDocumentsSqlQuery(&reqCopy, cteLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to build sql query: %w", err)
	}
	m.logger.Debug("findDocuments with query",
		zap.String("query", dbQuery),
		zap.Int("cteLimit", cteLimit),
		zap.Int("finalLimit", limit),
		zap.Int("filterCount", len(docReq.Filters.SubFilters)))

	rows, err := m.queryTx(ctx, tx, dbQuery, pgvector.NewVector(vector))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*DocumentQueryResponse
	for rows.Next() {
		doc := &DocumentQueryResponse{}

		err = rows.Scan(&doc.DocumentID, &doc.Distance, &doc.Attributes)
		if err != nil {
			return nil, fmt.Errorf("rows.scan error: %s", err)
		}

		if docReq.RetrieveAttributes != nil {
			if len(*docReq.RetrieveAttributes) > 0 {
				doc.Attributes = filters.FilterAttributesToRetrieve(doc.Attributes, *docReq.RetrieveAttributes)
			} else {
				doc.Attributes = nil
			}
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// getMaxChunksPerDoc returns the maximum number of embeddings (chunks) belonging to a single
// document in the embeddings table, considering the given attribute filters (legacy attr_ids path).
// This is used by the second scan stage to determine the exact CTE limit multiplier.
func (m *docManager) getMaxChunksPerDoc(ctx context.Context, tx *sql.Tx, filterAttrs []attributes.AttributeFilter) (int, error) {
	embAttrsFilter, err := m.getAttrsWhereClause(filterAttrs, "attr_ids2")
	if err != nil {
		return 1, fmt.Errorf("failed to build attrs where clause for max chunks query: %w", err)
	}

	query := fmt.Sprintf(
		`SELECT COALESCE(MAX(cnt), 1) FROM (SELECT COUNT(*) as cnt FROM %s.%s_emb EMB %s GROUP BY doc_id) sub`,
		m.schemaName,
		m.prefix,
		embAttrsFilter,
	)

	var maxChunks int
	err = tx.QueryRowContext(ctx, query).Scan(&maxChunks)
	if err != nil {
		return 1, fmt.Errorf("failed to get max chunks per doc: %w", err)
	}
	return maxChunks, nil
}

func (m *docManager) buildFindDocumentsSqlQuery(docReq *QueryDocumentsRequest, cteLimit int) (string, error) {

	limit := docReq.Limit
	distance := docReq.MaxDistance

	embAttrsFilter, err := m.getAttrsWhereClause(docReq.Filters.SubFilters, "attr_ids2")
	if err != nil {
		return "", fmt.Errorf("failed to create WHERE clause from attributes: %w", err)
	}

	if limit <= 0 {
		limit = math.MaxInt32
	}

	distanceStr := "1.0"
	if distance != nil {
		distanceStr = fmt.Sprintf("%f", *distance)
	}

	return fmt.Sprintf(documents_sql.FindDocumentsSqlQueryTemplate,
		m.schemaName,
		m.prefix,
		embAttrsFilter,
		distanceStr,
		limit,
		cteLimit,
	), nil
}
