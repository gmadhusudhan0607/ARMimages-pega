/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package dbmetrics

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"go.uber.org/zap"
)

var logger = log.GetNamedLogger("dbmetrics")

// Collector handles database metrics collection
type Collector struct {
	database    db.Database
	cache       []MetricsRow
	cacheExpiry time.Time
	cacheTTL    time.Duration
	cacheMutex  sync.RWMutex
}

// NewCollector creates a new database metrics collector
func NewCollector(database db.Database) *Collector {
	return &Collector{
		database: database,
		cacheTTL: 1 * time.Hour, // Default cache TTL
	}
}

// GetMetricsCollectorHandler returns a handler function that periodically updates the database statistics
func (c *Collector) GetMetricsCollectorHandler(ctx context.Context) func() error {
	return func() error {
		periodSec := 300 // default value in seconds
		if val := os.Getenv("DB_METRICS_UPDATE_PERIOD_SEC"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
				periodSec = parsed
			} else {
				logger.Warn("Invalid DB_METRICS_UPDATE_PERIOD_SEC value, using default", zap.String("value", val))
			}
		}
		ticker := time.NewTicker(time.Duration(periodSec) * time.Second)
		defer ticker.Stop()

		// Initial count on startup
		if err := c.updateDbMetrics(ctx); err != nil {
			logger.Error("failed to get initial database statistics", zap.Error(err))
		}

		for {
			select {
			case <-ctx.Done():
				logger.Info("database metrics collector handler stopped")
				return ctx.Err()
			case <-ticker.C:
				if err := c.updateDbMetrics(ctx); err != nil {
					logger.Error("failed to update database statistics", zap.Error(err))
				}
			}
		}
	}
}

func (c *Collector) updateDbMetrics(ctx context.Context) error {
	// Use UpdateDbMetrics() and convert to metrics format
	results, err := c.UpdateDbMetrics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database metrics: %w", err)
	}

	// Convert MetricsRow[] to IsolationMetrics format
	newDbMetricsMap := make(map[string]*IsolationMetrics)
	isolationCollectionCounts := make(map[string]int64)

	for _, row := range results {
		isolationID := row.IsoID
		collectionID := row.ColID

		// Count collections per isolation
		isolationCollectionCounts[isolationID]++

		// Initialize isolation metrics if not exists
		if _, exists := newDbMetricsMap[isolationID]; !exists {
			newDbMetricsMap[isolationID] = &IsolationMetrics{
				IsolationID: isolationID,
				Collections: make(map[string]*CollectionMetrics),
			}
		}

		// Convert negative values to 0 for error cases
		docCount := row.DocCount
		if docCount < 0 {
			docCount = 0
		}
		embCount := row.EmbCount
		if embCount < 0 {
			embCount = 0
		}

		// Store collection metrics
		newDbMetricsMap[isolationID].Collections[collectionID] = &CollectionMetrics{
			CollectionID:   collectionID,
			DocumentCount:  docCount,
			EmbeddingCount: embCount,
		}
	}

	// Set collection counts for each isolation
	for isolationID, isolationMetrics := range newDbMetricsMap {
		isolationMetrics.CollectionCount = isolationCollectionCounts[isolationID]
	}

	totalIsolations := int64(len(newDbMetricsMap))

	// Update the global variables using the local functions
	UpdateIsolationMetricsMap(newDbMetricsMap, totalIsolations)

	logger.Debug("updating database metrics",
		zap.Int64("isolation_count", totalIsolations),
		zap.Int("isolations_processed", len(newDbMetricsMap)),
		zap.Int("total_collections", len(results)))

	return nil
}

// CountDocuments counts the number of documents in a specific collection using the SQL function
func (c *Collector) CountDocuments(ctx context.Context, isolationID, collectionID string) (int64, error) {
	return c.getMetricForCollection(ctx, isolationID, collectionID, "doc_count")
}

// CountEmbeddings counts the number of embeddings in a specific collection using the SQL function
func (c *Collector) CountEmbeddings(ctx context.Context, isolationID, collectionID string) (int64, error) {
	return c.getMetricForCollection(ctx, isolationID, collectionID, "emb_count")
}

// CountAttributes counts the number of attributes in a specific collection using the SQL function
func (c *Collector) CountAttributes(ctx context.Context, isolationID, collectionID string) (int64, error) {
	return c.getMetricForCollection(ctx, isolationID, collectionID, "attr_count")
}

// CountEmbeddingQueue counts the number of embedding queue entries for a specific collection using the SQL function
func (c *Collector) CountEmbeddingQueue(ctx context.Context, isolationID, collectionID string) (int64, error) {
	return c.getMetricForCollection(ctx, isolationID, collectionID, "emb_queue_count")
}

// getCachedMetrics returns cached metrics or fetches fresh data if cache is expired
func (c *Collector) getCachedMetrics(ctx context.Context) ([]MetricsRow, error) {
	c.cacheMutex.RLock()
	if time.Now().Before(c.cacheExpiry) && len(c.cache) > 0 {
		cached := make([]MetricsRow, len(c.cache))
		copy(cached, c.cache)
		c.cacheMutex.RUnlock()
		return cached, nil
	}
	c.cacheMutex.RUnlock()

	// Cache expired or empty, fetch fresh data
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Double-check after acquiring write lock
	if time.Now().Before(c.cacheExpiry) && len(c.cache) > 0 {
		cached := make([]MetricsRow, len(c.cache))
		copy(cached, c.cache)
		return cached, nil
	}

	results, err := c.UpdateDbMetrics(ctx)
	if err != nil {
		return nil, err
	}

	c.cache = results
	c.cacheExpiry = time.Now().Add(c.cacheTTL)
	return results, nil
}

// UpdateDbMetrics updates database metrics using the SQL function
func (c *Collector) UpdateDbMetrics(ctx context.Context) ([]MetricsRow, error) {
	query := "SELECT * FROM vector_store.get_db_metrics()"

	rows, err := c.database.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MetricsRow
	for rows.Next() {
		var row MetricsRow
		err := rows.Scan(
			&row.IsoID,
			&row.ColID,
			&row.ProfileID,
			&row.SchemaPrefix,
			&row.TablesPrefix,
			&row.DocCount,
			&row.EmbCount,
			&row.AttrCount,
			&row.EmbQueueCount,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// GetAllMetrics returns all database metrics
func (c *Collector) GetAllMetrics(ctx context.Context) ([]MetricsRow, error) {
	return c.UpdateDbMetrics(ctx)
}

// GetMetricsForIsolation returns metrics for a specific isolation
func (c *Collector) GetMetricsForIsolation(ctx context.Context, isolationID string) ([]MetricsRow, error) {
	results, err := c.getCachedMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []MetricsRow
	for _, row := range results {
		if row.IsoID == isolationID {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

// SetCacheTTL sets the cache time-to-live duration
func (c *Collector) SetCacheTTL(ttl time.Duration) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cacheTTL = ttl
}

// ClearCache clears the metrics cache
func (c *Collector) ClearCache() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cache = nil
	c.cacheExpiry = time.Time{}
}

// getMetricForCollection retrieves a specific metric for a collection using cached data
func (c *Collector) getMetricForCollection(ctx context.Context, isolationID, collectionID, metricType string) (int64, error) {
	results, err := c.getCachedMetrics(ctx)
	if err != nil {
		return 0, err
	}

	// Find matching row and return specific metric
	for _, row := range results {
		if row.IsoID == isolationID && row.ColID == collectionID {
			switch metricType {
			case "doc_count":
				if row.DocCount < 0 {
					return 0, nil // Return 0 for error cases
				}
				return row.DocCount, nil
			case "emb_count":
				if row.EmbCount < 0 {
					return 0, nil // Return 0 for error cases
				}
				return row.EmbCount, nil
			case "attr_count":
				if row.AttrCount < 0 {
					return 0, nil // Return 0 for error cases
				}
				return row.AttrCount, nil
			case "emb_queue_count":
				if row.EmbQueueCount < 0 {
					return 0, nil // Return 0 for error cases
				}
				return row.EmbQueueCount, nil
			default:
				return 0, fmt.Errorf("unknown metric type: %s", metricType)
			}
		}
	}
	return 0, fmt.Errorf("collection not found: %s/%s", isolationID, collectionID)
}

type IsolationRequestedSize struct {
	IsolationID       string
	SchemaPrefix      string
	RequestedDiskSize int64
}

func (c *Collector) getIsolationsRequestedSize(ctx context.Context) ([]IsolationRequestedSize, error) {
	parseSizeToBytes := func(sizeStr string) int64 {
		var size int64
		var unit string
		_, err := fmt.Sscanf(sizeStr, "%d%s", &size, &unit)
		if err != nil {
			return 0
		}

		switch unit {
		case "MB":
			return size * 1024 * 1024
		case "GB":
			return size * 1024 * 1024 * 1024
		case "TB":
			return size * 1024 * 1024 * 1024 * 1024
		default:
			logger.Warn("unknown size unit", zap.String("unit", unit))
			return 0
		}
	}

	result := []IsolationRequestedSize{}

	rows, err := c.database.Query(ctx, "SELECT iso_id, iso_prefix, max_storage_size FROM vector_store.isolations")
	if err != nil {
		return nil, fmt.Errorf("error querying isolations requested size: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var isoID, isoPrefix, requestedSize string
		if err := rows.Scan(&isoID, &isoPrefix, &requestedSize); err != nil {
			return nil, fmt.Errorf("error scanning isolation requested size row: %w", err)
		}

		sizeBytes := parseSizeToBytes(requestedSize)

		result = append(result, IsolationRequestedSize{
			IsolationID:       isoID,
			SchemaPrefix:      isoPrefix,
			RequestedDiskSize: sizeBytes,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating isolation requested size rows: %w", err)
	}

	return result, nil
}

type DiskUsageEntry struct {
	IsolationID    string
	SchemaPrefix   string
	ProfileID      string
	TablesPrefix   string
	DiskUsageBytes int64
}

func (pc *Collector) getDiskUsage(ctx context.Context) ([]DiskUsageEntry, error) {
	result := []DiskUsageEntry{}
	rows, err := pc.database.Query(ctx, "SELECT iso_id, iso_prefix, profile_id, tables_prefix, disk_usage_bytes FROM vector_store.metrics_iso_size()")
	if err != nil {
		return nil, fmt.Errorf("error querying disk usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry DiskUsageEntry
		if err := rows.Scan(&entry.IsolationID, &entry.SchemaPrefix, &entry.ProfileID, &entry.TablesPrefix, &entry.DiskUsageBytes); err != nil {
			return nil, fmt.Errorf("error scanning disk usage row: %w", err)
		}
		result = append(result, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating disk usage rows: %w", err)
	}

	return result, nil
}

// CollectAttributeCardinalityMetrics collects metrics about multi-value attributes for a specific profile
// Returns the count of documents that have more than one value for each attribute key
func (c *Collector) CollectAttributeCardinalityMetrics(ctx context.Context, isoID, colID, profileID, schemaPrefix, tablePrefix string) ([]AttributeCardinalityMetric, error) {
	schema := fmt.Sprintf("vector_store_%s", schemaPrefix)
	attrTableName := fmt.Sprintf("%s.t_%s_attr", schema, tablePrefix)

	// Query to find attributes where documents have multiple values
	// First subquery counts values per (doc_id, name) combination
	// Then we count how many documents have more than 1 value for each attribute key
	query := fmt.Sprintf(`
        SELECT 
            name AS attr_key,
            COUNT(DISTINCT value) AS record_count
        FROM %s
        WHERE value IS NOT NULL
        GROUP BY name
        HAVING COUNT(DISTINCT value) > 1;
    `, attrTableName)

	rows, err := c.database.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query attribute cardinality: %w", err)
	}
	defer rows.Close()

	var results []AttributeCardinalityMetric
	for rows.Next() {
		var attrKey string
		var recordCount int64

		if err := rows.Scan(&attrKey, &recordCount); err != nil {
			return nil, fmt.Errorf("failed to scan attribute cardinality row: %w", err)
		}

		results = append(results, AttributeCardinalityMetric{
			IsolationID:  isoID,
			CollectionID: colID,
			ProfileID:    profileID,
			AttributeKey: attrKey,
			RecordCount:  recordCount,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attribute cardinality rows: %w", err)
	}

	return results, nil
}
