// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package documents

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

const findDocumentsJSONB4SqlQueryTemplate = `WITH filtered_embeddings_with_distance AS (
    SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=> $1 as distance
    FROM %[1]s.%[2]s_emb EMB
    %[3]s
    /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
    ORDER BY distance
    LIMIT $%[4]d
), ranked_filtered_embeddings AS (
    SELECT
        emb_id,
        distance,
        ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
    FROM filtered_embeddings_with_distance
    ORDER BY distance, emb_id
)
SELECT
    DOC.doc_id,
    distance,
    EMB.attributes
FROM ranked_filtered_embeddings RFE
    LEFT JOIN %[1]s.%[2]s_emb EMB ON RFE.emb_id = EMB.emb_id
    LEFT JOIN %[1]s.%[2]s_doc DOC ON DOC.doc_id = EMB.doc_id
WHERE distance <= $%[5]d AND rank = 1
ORDER BY distance, doc_id
LIMIT $%[6]d`

// queryParamsCollector collects parameters for parameterized queries
type queryParamsCollector struct {
	params []interface{}
}

// newQueryParamsCollector creates a new parameter collector starting at the specified position
func newQueryParamsCollector(initialParams ...interface{}) *queryParamsCollector {
	return &queryParamsCollector{
		params: initialParams,
	}
}

// add adds a parameter and returns its position (1-indexed)
func (c *queryParamsCollector) add(param interface{}) int {
	c.params = append(c.params, param)
	return len(c.params)
}

// getParams returns all collected parameters
func (c *queryParamsCollector) getParams() []interface{} {
	return c.params
}

// buildJSONBArrayClauseParam creates a parameterized JSONB array check for a single value
// Returns: $n = ANY (SELECT jsonb_array_elements_text(attributes->$m->'values'))
func buildJSONBArrayClauseParam(collector *queryParamsCollector, attrName, value string) string {
	valuePos := collector.add(value)
	attrNamePos := collector.add(attrName)

	return fmt.Sprintf("$%d = ANY (\n\t\t    SELECT jsonb_array_elements_text(attributes->$%d->'values')\n\t\t)", valuePos, attrNamePos)
}

// buildJSONBExistsClauseParam creates a parameterized JSONB EXISTS check for multiple values
// Returns: EXISTS (SELECT 1 FROM jsonb_array_elements_text(attributes->$n->'values') AS x(val) WHERE x.val IN ($m, $p, ...))
func buildJSONBExistsClauseParam(collector *queryParamsCollector, attrName string, values []string) string {
	attrNamePos := collector.add(attrName)

	var placeholders []string
	for _, val := range values {
		pos := collector.add(val)
		placeholders = append(placeholders, fmt.Sprintf("$%d", pos))
	}

	return fmt.Sprintf("EXISTS (\n\t\t    SELECT 1\n\t\t    FROM jsonb_array_elements_text(attributes->$%d->'values') AS ds(val)\n\t\t    WHERE ds.val IN (\n\t\t        %s\n\t\t    )\n\t\t)",
		attrNamePos,
		strings.Join(placeholders, ","))
}

// attributeToJSONBSqlClause4Param converts a single AttributeFilter to parameterized JSONB SQL condition
// For single value, uses = ANY pattern
// For multiple values, uses EXISTS with IN pattern
func attributeToJSONBSqlClause4Param(collector *queryParamsCollector, attr attributes.AttributeFilter) string {
	if len(attr.Values) == 0 {
		return ""
	}

	// Single value: use = ANY pattern
	if len(attr.Values) == 1 {
		return buildJSONBArrayClauseParam(collector, attr.Name, attr.Values[0])
	}

	// Multiple values: use EXISTS with IN pattern
	return buildJSONBExistsClauseParam(collector, attr.Name, attr.Values)
}

// getJSONBAttrsWhereClause4Param converts array of AttributeFilters to complete parameterized WHERE clause
// Returns empty string if no filters, otherwise returns formatted WHERE clause with proper indentation
func (m *docManager) getJSONBAttrsWhereClause4Param(collector *queryParamsCollector, filterAttrs []attributes.AttributeFilter) string {
	if len(filterAttrs) == 0 {
		return ""
	}

	var attrClauses []string
	for _, attr := range filterAttrs {
		clause := attributeToJSONBSqlClause4Param(collector, attr)
		if clause != "" {
			attrClauses = append(attrClauses, clause)
		}
	}

	if len(attrClauses) == 0 {
		return ""
	}

	return fmt.Sprintf("    WHERE %s", strings.Join(attrClauses, "\n\t\tAND "))
}

// safeMul multiplies a and b, capping at math.MaxInt32 on overflow.
func safeMul(a, b int) int {
	if a <= 0 || b <= 0 {
		return math.MaxInt32
	}
	result := a * b
	if result/b != a || result < 0 || result > math.MaxInt32 {
		return math.MaxInt32
	}
	return result
}

// buildFindDocumentsJSONB4SqlQuery builds the complete parameterized SQL query using JSONB array operations for filtering.
// cteLimit controls the embedding-level scan limit (CTE LIMIT), while docReq.Limit controls the final document-level limit.
// Returns the query string and the parameters to be used.
func (m *docManager) buildFindDocumentsJSONB4SqlQuery(docReq *QueryDocumentsRequest, vector []float32, cteLimit int) (string, []interface{}) {
	// Start with vector as first parameter
	collector := newQueryParamsCollector(pgvector.NewVector(vector))

	limit := docReq.Limit
	if limit <= 0 {
		limit = math.MaxInt32
	}

	distance := float64(1.0)
	if docReq.MaxDistance != nil {
		distance = *docReq.MaxDistance
	}

	// Build WHERE clause with parameters
	embAttrsFilter := m.getJSONBAttrsWhereClause4Param(collector, docReq.Filters.SubFilters)

	// Add CTE limit (embedding scan), distance, and final limit (document level) parameters
	cteLimitPos := collector.add(cteLimit)
	distancePos := collector.add(distance)
	finalLimitPos := collector.add(limit)

	query := fmt.Sprintf(findDocumentsJSONB4SqlQueryTemplate,
		m.schemaName,
		m.prefix,
		embAttrsFilter,
		cteLimitPos,
		distancePos,
		finalLimitPos,
	)

	return query, collector.getParams()
}

// FindDocuments4 performs vector similarity search using JSONB 'attributes' column with array operations for filtering.
// It uses a two-stage approach to maximize the chance of returning the requested number of documents:
// Stage 1: Applies a configurable multiplier to the CTE limit to scan more embeddings than requested documents.
// Stage 2 (opt-in): If Stage 1 returns fewer documents than requested, queries the actual max chunks per document
// and re-executes the search with an exact multiplier.
func (m *docManager) FindDocuments4(ctx context.Context, docReq *QueryDocumentsRequest) ([]*DocumentQueryResponse, error) {
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
	docs, err := m.executeDocumentSearch4(ctx, tx, docReq, vector, cteLimit, limit)
	if err != nil {
		return nil, err
	}

	// Stage 2: If first search returned fewer documents than requested, and second scan is enabled
	if len(docs) < limit && docReq.EnableSecondScan {
		maxChunks, err := m.getMaxChunksPerDocJSONB4(ctx, tx, docReq.Filters.SubFilters)
		if err != nil {
			m.logger.Warn("failed to get max chunks per doc for second scan, using first scan results",
				zap.Error(err))
			return docs, commit()
		}

		newCteLimit := safeMul(limit, maxChunks)
		if newCteLimit > cteLimit {
			m.logger.Debug("executing second scan for document search",
				zap.Int("firstScanResults", len(docs)),
				zap.Int("requestedLimit", limit),
				zap.Int("newCteLimit", newCteLimit),
				zap.Int("maxChunksPerDoc", maxChunks))

			docs, err = m.executeDocumentSearch4(ctx, tx, docReq, vector, newCteLimit, limit)
			if err != nil {
				return nil, err
			}
		}
	}

	return docs, commit()
}

// executeDocumentSearch4 builds and executes the document search query with the given CTE and final limits.
func (m *docManager) executeDocumentSearch4(ctx context.Context, tx *sql.Tx, docReq *QueryDocumentsRequest, vector []float32, cteLimit int, limit int) ([]*DocumentQueryResponse, error) {
	reqCopy := *docReq
	reqCopy.Limit = limit

	dbQuery, queryParams := m.buildFindDocumentsJSONB4SqlQuery(&reqCopy, vector, cteLimit)
	m.logger.Debug("findDocuments4 with query",
		zap.String("query", dbQuery),
		zap.Int("cteLimit", cteLimit),
		zap.Int("finalLimit", limit),
		zap.Int("filterCount", len(docReq.Filters.SubFilters)))

	rows, err := m.queryTx(ctx, tx, dbQuery, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*DocumentQueryResponse
	for rows.Next() {
		doc := &DocumentQueryResponse{}
		var attributesV2 attributes.AttributesV2

		err = rows.Scan(&doc.DocumentID, &doc.Distance, &attributesV2)
		if err != nil {
			return nil, fmt.Errorf("rows.scan error: %s", err)
		}

		// Convert AttributesV2 (JSONB format) to Attributes (array format)
		// Use filtered conversion if retrieveAttributes is specified for better performance
		if docReq.RetrieveAttributes != nil {
			if len(*docReq.RetrieveAttributes) > 0 {
				doc.Attributes = attributes.ConvertAttributesV2ToV1WithFilter(attributesV2, *docReq.RetrieveAttributes)
			} else {
				doc.Attributes = nil
			}
		} else {
			doc.Attributes = attributes.ConvertAttributesV2ToV1(attributesV2)
		}

		docs = append(docs, doc)
	}
	return docs, nil
}

// getMaxChunksPerDocJSONB4 returns the maximum number of embeddings (chunks) belonging to a single
// document in the embeddings table, considering the given attribute filters.
// This is used by the second scan stage to determine the exact CTE limit multiplier.
func (m *docManager) getMaxChunksPerDocJSONB4(ctx context.Context, tx *sql.Tx, filterAttrs []attributes.AttributeFilter) (int, error) {
	collector := newQueryParamsCollector()
	embAttrsFilter := m.getJSONBAttrsWhereClause4Param(collector, filterAttrs)

	query := fmt.Sprintf(
		`SELECT COALESCE(MAX(cnt), 1) FROM (SELECT COUNT(*) as cnt FROM %s.%s_emb EMB %s GROUP BY doc_id) sub`,
		m.schemaName,
		m.prefix,
		embAttrsFilter,
	)

	var maxChunks int
	err := tx.QueryRowContext(ctx, query, collector.getParams()...).Scan(&maxChunks)
	if err != nil {
		return 1, fmt.Errorf("failed to get max chunks per doc: %w", err)
	}
	return maxChunks, nil
}
