/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package embedings

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	embeddings_sql "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings/sql"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/filters"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

func attributeToSqlClause(attr attributes.AttributeFilter) string {
	var valuesHashes []string
	for _, val := range attr.Values {
		hash := md5.Sum([]byte(val))
		valuesHashes = append(valuesHashes, fmt.Sprintf("'%s'", hex.EncodeToString(hash[:])))
	}
	// TODO : refactor this later (default must not be set on this level)
	attrType := attr.Type
	if attrType == "" {
		attrType = "string"
	}
	return fmt.Sprintf("( name = '%s' AND type = '%s' AND value_hash IN ( %s ) )", attr.Name, attrType, strings.Join(valuesHashes, ", "))
}

func (m *embManager) FindChunks2(ctx context.Context, chReq *QueryChunksRequest) ([]*Chunk, error) {
	chs := []*Chunk{}

	vector, _, err := m.Embedder.GetEmbedding(ctx, chReq.Filters.Query)
	if err != nil {
		return nil, err
	}

	dbQuery := m.buildFindChunksSqlQuery(chReq)
	m.logger.Debug("findChunks with query", zap.String("query", dbQuery))

	// tx is needed to run all queries in one db session
	tx, err := m.database.GetConn().Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %s", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
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

	rows, err := m.queryTx(ctx, tx, dbQuery, pgvector.NewVector(vector))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ch := &Chunk{}

		if chReq.RetrieveVector {
			err = rows.Scan(&ch.ID, &ch.DocumentID, &ch.Content, &ch.Attributes, &ch.Distance, &ch.Vector)
			ch.Embedding = ch.Vector.Slice()
		} else {
			err = rows.Scan(&ch.ID, &ch.DocumentID, &ch.Content, &ch.Attributes, &ch.Distance)
		}
		if err != nil {
			return nil, fmt.Errorf("rows.scan error: %s", err)
		}

		// configure attributes returned based on retrieveAttributes parameter
		if chReq.RetrieveAttributes != nil {
			if len(*chReq.RetrieveAttributes) > 0 {
				ch.Attributes = filters.FilterAttributesToRetrieve(ch.Attributes, *chReq.RetrieveAttributes)
			} else {
				ch.Attributes = nil
			}
		}
		chs = append(chs, ch)
	}

	if helpers.IsEncourageSemSearchIndexUseEnabled() {
		_, err = m.execTx(ctx, tx, `
		SET LOCAL enable_seqscan = on;
		SET LOCAL enable_bitmapscan = on;
		`)
		if err != nil {
			return nil, fmt.Errorf("failed enabling back seqscan and bitmapscan: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return chs, nil
}

func (m *embManager) buildFindChunksSqlQuery(chReq *QueryChunksRequest) string {
	limit := chReq.Limit
	distance := chReq.MaxDistance
	embAttrsFilter := m.getAttrsWhereClause(chReq.Filters.SubFilters)

	if limit <= 0 {
		limit = math.MaxInt32
	}

	distanceStr := "1.0"
	if distance != nil {
		distanceStr = fmt.Sprintf("%f", *distance)
	}

	return fmt.Sprintf(embeddings_sql.FindChunksSqlQueryTemplate,
		m.schemaName,
		m.tablesPrefix,
		embAttrsFilter,
		distanceStr,
		limit,
	)
}

func (m *embManager) getAttrsWhereClause(filterAttrs []attributes.AttributeFilter) string {
	if len(filterAttrs) == 0 {
		return ""
	}
	attrClauses := []string{}
	for _, attr := range filterAttrs {
		attrClauseTpl := `
        attr_ids2 && (
            SELECT array_agg(attr_id)
            FROM %[1]s.%[2]s_attr
            WHERE  %[3]s
        )`
		attrClause := fmt.Sprintf(attrClauseTpl, m.schemaName, m.tablesPrefix, attributeToSqlClause(attr))
		attrClauses = append(attrClauses, attrClause)
	}
	return fmt.Sprintf("    WHERE %s", strings.Join(attrClauses, "\n     AND "))
}
