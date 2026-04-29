/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package dbmetrics

import (
	"sync/atomic"
)

// MetricsRow represents a single row of database metrics data
type MetricsRow struct {
	IsoID         string `db:"iso_id"`
	ColID         string `db:"col_id"`
	ProfileID     string `db:"profile_id"`
	SchemaPrefix  string `db:"schema_prefix"`
	TablesPrefix  string `db:"tables_prefix"`
	DocCount      int64  `db:"doc_count"`
	EmbCount      int64  `db:"emb_count"`
	AttrCount     int64  `db:"attr_count"`
	EmbQueueCount int64  `db:"emb_queue_count"`
}

// IsolationMetrics holds statistics for single isolation
type IsolationMetrics struct {
	IsolationID     string                        `json:"isolation_id"`
	CollectionCount int64                         `json:"collection_count"`
	Collections     map[string]*CollectionMetrics `json:"collections"`
}

// CollectionMetrics holds statistics for a single collection
type CollectionMetrics struct {
	CollectionID   string `json:"collection_id"`
	DocumentCount  int64  `json:"document_count"`
	EmbeddingCount int64  `json:"embedding_count"`
}

// Global variables to store the isolation statistics
var isolationMetricsMap map[string]*IsolationMetrics
var isolationCount int64

// GetCollectionMetrics returns the statistics for a specific collection in an isolation
func GetCollectionMetrics(isolationID, collectionID string) *CollectionMetrics {
	if isolationMetricsMap == nil {
		return nil
	}

	isolation, exists := isolationMetricsMap[isolationID]
	if !exists {
		return nil
	}

	collection, exists := isolation.Collections[collectionID]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	return &CollectionMetrics{
		CollectionID:   collection.CollectionID,
		DocumentCount:  collection.DocumentCount,
		EmbeddingCount: collection.EmbeddingCount,
	}
}

// GetCollectionDocumentCount returns the document count for a specific collection
func GetCollectionDocumentCount(isolationID, collectionID string) int64 {
	metrics := GetCollectionMetrics(isolationID, collectionID)
	if metrics == nil {
		return 0
	}
	return metrics.DocumentCount
}

// GetCollectionEmbeddingCount returns the embedding count for a specific collection
func GetCollectionEmbeddingCount(isolationID, collectionID string) int64 {
	metrics := GetCollectionMetrics(isolationID, collectionID)
	if metrics == nil {
		return 0
	}
	return metrics.EmbeddingCount
}

// UpdateIsolationMetricsMap updates the global isolation statistics map
// This function is used by the background metrics collector
func UpdateIsolationMetricsMap(newMetricsMap map[string]*IsolationMetrics, totalIsolations int64) {
	isolationMetricsMap = newMetricsMap
	atomic.StoreInt64(&isolationCount, totalIsolations)
}

// AttributeCardinalityMetric holds statistics for attribute value counts
type AttributeCardinalityMetric struct {
	IsolationID  string `db:"isolation_id"`
	CollectionID string `db:"collection_id"`
	ProfileID    string `db:"profile_id"`
	AttributeKey string `db:"attribute_key"`
	RecordCount  int64  `db:"record_count"`
}
