// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package embedings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

const findChunksJSONB4SqlQueryTemplate = `WITH filtered_embeddings_with_distance AS (
    SELECT EMB.emb_id, emb.embedding <=> $1 as distance
    FROM %[1]s.%[2]s_emb EMB
    %[3]s
    /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
    ORDER BY distance
    LIMIT $%[4]d
)
SELECT
    EMB.emb_id,
    EMB.doc_id,
    content,
    EMB.attributes,
    distance%[5]s
FROM filtered_embeddings_with_distance FDEMB LEFT JOIN %[1]s.%[2]s_emb EMB ON FDEMB.emb_id = EMB.emb_id
WHERE distance <= $%[6]d
ORDER BY distance, emb_id
LIMIT $%[7]d`

// queryParamsCollectorForChunks collects parameters for parameterized queries
type queryParamsCollectorForChunks struct {
	params []interface{}
}

// newQueryParamsCollectorForChunks creates a new parameter collector starting at the specified position
func newQueryParamsCollectorForChunks(initialParams ...interface{}) *queryParamsCollectorForChunks {
	return &queryParamsCollectorForChunks{
		params: initialParams,
	}
}

// add adds a parameter and returns its position (1-indexed)
func (c *queryParamsCollectorForChunks) add(param interface{}) int {
	c.params = append(c.params, param)
	return len(c.params)
}

// getParams returns all collected parameters
func (c *queryParamsCollectorForChunks) getParams() []interface{} {
	return c.params
}

// buildJSONBArrayClauseParamForChunks creates a parameterized JSONB array check for a single value
// Returns: $n = ANY (SELECT jsonb_array_elements_text(attributes->$m->'values'))
func buildJSONBArrayClauseParamForChunks(collector *queryParamsCollectorForChunks, attrName, value string) string {
	valuePos := collector.add(value)
	attrNamePos := collector.add(attrName)

	return fmt.Sprintf("$%d = ANY (\n\t\t    SELECT jsonb_array_elements_text(attributes->$%d->'values')\n\t\t)", valuePos, attrNamePos)
}

// buildJSONBExistsClauseParamForChunks creates a parameterized JSONB EXISTS check for multiple values
// Returns: EXISTS (SELECT 1 FROM jsonb_array_elements_text(attributes->$n->'values') AS x(val) WHERE x.val IN ($m, $p, ...))
func buildJSONBExistsClauseParamForChunks(collector *queryParamsCollectorForChunks, attrName string, values []string) string {
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

// attributeToJSONBSqlClause4ParamForChunks converts a single AttributeFilter to parameterized JSONB SQL condition
// For single value, uses = ANY pattern
// For multiple values, uses EXISTS with IN pattern
func attributeToJSONBSqlClause4ParamForChunks(collector *queryParamsCollectorForChunks, attr attributes.AttributeFilter) string {
	if len(attr.Values) == 0 {
		return ""
	}

	// Single value: use = ANY pattern
	if len(attr.Values) == 1 {
		return buildJSONBArrayClauseParamForChunks(collector, attr.Name, attr.Values[0])
	}

	// Multiple values: use EXISTS with IN pattern
	return buildJSONBExistsClauseParamForChunks(collector, attr.Name, attr.Values)
}

// getJSONBAttrsWhereClause4Param converts array of AttributeFilters to complete parameterized WHERE clause
// Returns empty string if no filters, otherwise returns formatted WHERE clause with proper indentation
func (m *embManager) getJSONBAttrsWhereClause4Param(collector *queryParamsCollectorForChunks, filterAttrs []attributes.AttributeFilter) string {
	if len(filterAttrs) == 0 {
		return ""
	}

	var attrClauses []string
	for _, attr := range filterAttrs {
		clause := attributeToJSONBSqlClause4ParamForChunks(collector, attr)
		if clause != "" {
			attrClauses = append(attrClauses, clause)
		}
	}

	if len(attrClauses) == 0 {
		return ""
	}

	return fmt.Sprintf("    WHERE %s", strings.Join(attrClauses, "\n\t\tAND "))
}

// buildFindChunksJSONB4SqlQuery builds the complete parameterized SQL query using JSONB array operations for filtering
// Returns the query string and the parameters to be used
func (m *embManager) buildFindChunksJSONB4SqlQuery(chReq *QueryChunksRequest, vector []float32) (string, []interface{}) {
	// Start with vector as first parameter
	collector := newQueryParamsCollectorForChunks(pgvector.NewVector(vector))

	limit := chReq.Limit
	if limit <= 0 {
		limit = math.MaxInt32
	}

	distance := float64(1.0)
	if chReq.MaxDistance != nil {
		distance = *chReq.MaxDistance
	}

	// Build WHERE clause with parameters
	embAttrsFilter := m.getJSONBAttrsWhereClause4Param(collector, chReq.Filters.SubFilters)

	// Include embedding column in SELECT when RetrieveVector is requested
	embeddingColumn := ""
	if chReq.RetrieveVector {
		embeddingColumn = ",\n    EMB.embedding"
	}

	// Add limit and distance parameters
	limitPos := collector.add(limit)
	distancePos := collector.add(distance)
	limitPos2 := collector.add(limit)

	query := fmt.Sprintf(findChunksJSONB4SqlQueryTemplate,
		m.schemaName,
		m.tablesPrefix,
		embAttrsFilter,
		limitPos,
		embeddingColumn,
		distancePos,
		limitPos2,
	)

	return query, collector.getParams()
}

// FindChunks4 performs vector similarity search using JSONB 'attributes' column with array operations for filtering
func (m *embManager) FindChunks4(ctx context.Context, chReq *QueryChunksRequest) ([]*Chunk, error) {
	chs := []*Chunk{}

	vector, _, err := m.Embedder.GetEmbedding(ctx, chReq.Filters.Query)
	if err != nil {
		return nil, err
	}

	dbQuery, queryParams := m.buildFindChunksJSONB4SqlQuery(chReq, vector)
	m.logger.Debug("findChunks4 with query",
		zap.String("query", dbQuery),
		zap.Int("filterCount", len(chReq.Filters.SubFilters)))

	// tx is needed to run all queries in one db session
	tx, err := m.database.GetConn().Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %s", err)
	}
	defer func() {
		if err != nil {
			if err1 := tx.Rollback(); err1 != nil && !errors.Is(err, sql.ErrTxDone) {
				m.logger.Error("failed to rollback transaction", zap.Error(err))
			}
		}
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

	rows, err := m.queryTx(ctx, tx, dbQuery, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ch := &Chunk{}
		var attributesV2 attributes.AttributesV2

		if chReq.RetrieveVector {
			err = rows.Scan(&ch.ID, &ch.DocumentID, &ch.Content, &attributesV2, &ch.Distance, &ch.Vector)
			ch.Embedding = ch.Vector.Slice()
		} else {
			err = rows.Scan(&ch.ID, &ch.DocumentID, &ch.Content, &attributesV2, &ch.Distance)
		}
		if err != nil {
			return nil, fmt.Errorf("rows.scan error: %s", err)
		}

		// Convert AttributesV2 (JSONB format) to Attributes (array format)
		// Use filtered conversion if retrieveAttributes is specified for better performance
		if chReq.RetrieveAttributes != nil {
			if len(*chReq.RetrieveAttributes) > 0 {
				// Filter during conversion for optimal performance
				ch.Attributes = attributes.ConvertAttributesV2ToV1WithFilter(attributesV2, *chReq.RetrieveAttributes)
			} else {
				ch.Attributes = nil
			}
		} else {
			// No filter specified, convert all attributes
			ch.Attributes = attributes.ConvertAttributesV2ToV1(attributesV2)
		}
		chs = append(chs, ch)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return chs, nil
}
