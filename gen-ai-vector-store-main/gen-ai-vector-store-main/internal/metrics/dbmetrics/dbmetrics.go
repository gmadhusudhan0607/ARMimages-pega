/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package dbmetrics

import (
	"context"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
)

// Manager provides a unified interface for database metrics collection
type Manager struct {
	collector           *Collector
	prometheusCollector *PrometheusCollector
}

// NewManager creates a new database metrics manager
func NewManager(database db.Database) *Manager {
	return &Manager{
		collector:           NewCollector(database),
		prometheusCollector: NewPrometheusCollector(database),
	}
}

// GetCollector returns the database collector
func (m *Manager) GetCollector() *Collector {
	return m.collector
}

// GetPrometheusCollector returns the Prometheus collector
func (m *Manager) GetPrometheusCollector() *PrometheusCollector {
	return m.prometheusCollector
}

// GetFreshCollectionMetrics retrieves fresh statistics directly from the database
// for a specific collection, bypassing the cached values
func (m *Manager) GetFreshCollectionMetrics(ctx context.Context, isolationID, collectionID string) (int64, int64) {
	logger.Info("GetFreshCollectionMetrics called",
		zap.String("isolationID", isolationID),
		zap.String("collectionID", collectionID))

	// Force cache refresh by calling UpdateDbMetrics directly
	results, err := m.collector.UpdateDbMetrics(ctx)
	if err != nil {
		logger.Error("failed to get fresh metrics", zap.Error(err))
		return 0, 0
	}

	// Find specific collection
	for _, row := range results {
		if row.IsoID == isolationID && row.ColID == collectionID {
			docCount := row.DocCount
			embCount := row.EmbCount
			if docCount < 0 {
				docCount = 0
			}
			if embCount < 0 {
				embCount = 0
			}

			logger.Info("retrieved fresh collection metrics",
				zap.String("isolationID", isolationID),
				zap.String("collectionID", collectionID),
				zap.Int64("documentCount", docCount),
				zap.Int64("embeddingCount", embCount))

			return docCount, embCount
		}
	}

	logger.Warn("collection not found in fresh metrics",
		zap.String("isolationID", isolationID),
		zap.String("collectionID", collectionID))

	return 0, 0
}

// GetAllMetrics returns all database metrics
func (m *Manager) GetAllMetrics(ctx context.Context) ([]MetricsRow, error) {
	return m.collector.GetAllMetrics(ctx)
}

// GetMetricsForIsolation returns metrics for a specific isolation
func (m *Manager) GetMetricsForIsolation(ctx context.Context, isolationID string) ([]MetricsRow, error) {
	return m.collector.GetMetricsForIsolation(ctx, isolationID)
}

// CountAttributes counts the number of attributes in a specific collection
func (m *Manager) CountAttributes(ctx context.Context, isolationID, collectionID string) (int64, error) {
	return m.collector.CountAttributes(ctx, isolationID, collectionID)
}

// CountEmbeddingQueue counts the number of embedding queue entries for a specific collection
func (m *Manager) CountEmbeddingQueue(ctx context.Context, isolationID, collectionID string) (int64, error) {
	return m.collector.CountEmbeddingQueue(ctx, isolationID, collectionID)
}

// SetCacheTTL sets the cache time-to-live duration for the collector
func (m *Manager) SetCacheTTL(ttl time.Duration) {
	m.collector.SetCacheTTL(ttl)
}

// ClearCache clears the metrics cache
func (m *Manager) ClearCache() {
	m.collector.ClearCache()
}
