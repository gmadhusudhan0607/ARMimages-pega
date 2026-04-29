/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package resources

const (
	StatusCompleted  = "COMPLETED"
	StatusInProgress = "IN_PROGRESS"
	StatusError      = "ERROR"
)

// AllowedDocumentStatuses returns the list of valid document status values
func AllowedDocumentStatuses() []string {
	return []string{StatusCompleted, StatusInProgress, StatusError}
}

// DefaultPGVectorHnswEfSearch is the default value for the hnsw.ef_search parameter in PostgreSQL vector search.
// This value was chosen empirically to balance search performance and recall.
// Test data is table of size 1 GB and length 126K rows with 1536-dimensional vectors.
// Suggested amount of chunks requested (LIMIT) is below 1000 chunks.
// It can be overridden by the PGVECTOR_HNSW_EF_SEARCH environment variable.
const DefaultPGVectorHnswEfSearch int64 = 280

const PGVectorHnswEfSearchEnvVarName = "PGVECTOR_HNSW_EF_SEARCH"

// DefaultHnswMaxScanTuples is max amount of additional tuples scanned during iterative scan.
// With approximate indexes, queries with filtering can return less results
// since filtering is applied after the index is scanned.
// With iterative index scans enabled, database will automatically scan more of the index until enough results are found
// https://github.com/pgvector/pgvector?tab=readme-ov-file#iterative-scan-options
const DefaultHnswMaxScanTuples = 40000

const PGVectorHnswMaxScanTuplesEnvVarName = "PGVECTOR_HNSW_MAX_SCAN_TUPLES"

// DefaultHnswScanMemMultiplierDefault
// https://github.com/pgvector/pgvector?tab=readme-ov-file#iterative-scan-options
const DefaultHnswScanMemMultiplier = 3

const PGVectorHnswScanMemMultiplierEnvVarName = "PGVECTOR_HNSW_SCAN_MEM_MULTIPLIER"

// DefaultDocumentSemanticSearchMultiplier is the default multiplier applied to the CTE limit
// during document semantic search. The CTE limit controls how many embeddings are scanned
// before deduplication to unique documents. A higher multiplier increases the chance of
// returning the requested number of documents when documents have multiple chunks.
// Suggested LIMIT is below 1000 documents; with a multiplier of 10, CTE scans up to 10,000 embeddings.
const DefaultDocumentSemanticSearchMultiplier int64 = 10

// DocumentSemanticSearchMultiplierEnvVarName is the environment variable name for overriding
// the default document semantic search CTE limit multiplier.
const DocumentSemanticSearchMultiplierEnvVarName = "DOCUMENT_SEMANTIC_SEARCH_MULTIPLIER"
