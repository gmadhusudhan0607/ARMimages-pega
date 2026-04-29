/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package dbmetrics

import (
	"context"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	dbDocumentCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_document_count",
			Help: "Number of documents stored in database tables",
		},
		[]string{"iso_id", "profile_id", "schema_prefix", "tables_prefix"},
	)

	dbEmbeddingCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_embedding_count",
			Help: "Number of embeddings stored in database tables",
		},
		[]string{"iso_id", "profile_id", "schema_prefix", "tables_prefix"},
	)

	dbAttributeCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_attribute_count",
			Help: "Number of attributes stored in database tables",
		},
		[]string{"iso_id", "profile_id", "schema_prefix", "tables_prefix"},
	)

	dbEmbeddingQueueCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_embedding_queue_count",
			Help: "Number of documents in embedding queue",
		},
		[]string{"iso_id", "profile_id", "schema_prefix", "tables_prefix"},
	)

	dbIsolationsCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_isolations_count",
			Help: "Number of isolations in the database",
		},
		[]string{},
	)

	dbCollectionsCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_collections_count",
			Help: "Number of collections in the database",
		},
		[]string{"iso_id", "schema_prefix"},
	)

	dbIsolationsRequestedDiskSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_isolations_requested_disk_size_bytes",
			Help: "Requested disk size for isolations in bytes",
		},
		[]string{"iso_id", "schema_prefix"},
	)

	dbDiskUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_disk_usage_bytes",
			Help: "Disk usage of the database in bytes",
		},
		[]string{"iso_id", "schema_prefix", "profile_id", "tables_prefix"},
	)
	attributeValueCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_attribute_value_count",
			Help: "Number of documents with multiple values assigned to each attribute key (only attributes with multi-value documents are exposed)",
		},
		[]string{"iso_id", "col_id", "profile_id", "attribute_key"},
	)
)

func init() {
	prometheus.MustRegister(dbDocumentCount)
	prometheus.MustRegister(dbEmbeddingCount)
	prometheus.MustRegister(dbAttributeCount)
	prometheus.MustRegister(dbEmbeddingQueueCount)
	prometheus.MustRegister(dbIsolationsCount)
	prometheus.MustRegister(dbCollectionsCount)
	prometheus.MustRegister(dbIsolationsRequestedDiskSize)
	prometheus.MustRegister(dbDiskUsage)
	prometheus.MustRegister(attributeValueCount)
}

// PrometheusCollector handles a Prometheus metrics collection for database metrics
type PrometheusCollector struct {
	collector    *Collector
	updatePeriod time.Duration
}

// NewPrometheusCollector creates a new Prometheus metrics collector
func NewPrometheusCollector(database db.Database) *PrometheusCollector {
	return &PrometheusCollector{
		collector:    NewCollector(database),
		updatePeriod: time.Duration(60) * time.Minute,
	}
}

// SetUpdatePeriod sets the update period for metrics collection
func (pc *PrometheusCollector) SetUpdatePeriod(period time.Duration) {
	pc.updatePeriod = period
}

// GetDbMetricsHandler returns a function that runs database metrics collection for Prometheus
func (pc *PrometheusCollector) GetDbMetricsHandler(ctx context.Context) func() error {
	return func() error {
		for {
			if err := pc.updatePrometheusMetrics(ctx); err != nil {
				logger.Error("failed to update database metrics", zap.Error(err))
			}
			time.Sleep(pc.updatePeriod)
		}
	}
}

func (pc *PrometheusCollector) updatePrometheusMetrics(ctx context.Context) error {
	logger.Debug("updating database metrics")
	results, err := pc.collector.UpdateDbMetrics(ctx)
	if err != nil {
		return err
	}

	// Clear existing metrics to avoid stale data
	dbDocumentCount.Reset()
	dbEmbeddingCount.Reset()
	dbAttributeCount.Reset()
	dbEmbeddingQueueCount.Reset()
	attributeValueCount.Reset()

	// Update Prometheus metrics
	for _, row := range results {
		labels := []string{row.IsoID, row.ProfileID, row.SchemaPrefix, row.TablesPrefix}

		// Only set positive values, negative values indicate errors
		if row.DocCount >= 0 {
			dbDocumentCount.WithLabelValues(labels...).Set(float64(row.DocCount))
		}
		if row.EmbCount >= 0 {
			dbEmbeddingCount.WithLabelValues(labels...).Set(float64(row.EmbCount))
		}
		if row.AttrCount >= 0 {
			dbAttributeCount.WithLabelValues(labels...).Set(float64(row.AttrCount))
		}
		if row.EmbQueueCount >= 0 {
			dbEmbeddingQueueCount.WithLabelValues(labels...).Set(float64(row.EmbQueueCount))
		}
		// Collect attribute cardinality metrics for this profile
		cardinalityMetrics, err := pc.collector.CollectAttributeCardinalityMetrics(
			ctx, row.IsoID, row.ColID, row.ProfileID, row.SchemaPrefix, row.TablesPrefix)
		if err != nil {
			logger.Warn("failed to collect attribute cardinality metrics",
				zap.String("iso_id", row.IsoID),
				zap.String("col_id", row.ColID),
				zap.String("profile_id", row.ProfileID),
				zap.Error(err))
			continue
		}

		// Update attribute value count metrics (only for multi-value attributes)
		for _, metric := range cardinalityMetrics {
			attributeValueCount.WithLabelValues(
				metric.IsolationID,
				metric.CollectionID,
				metric.ProfileID,
				metric.AttributeKey,
			).Set(float64(metric.RecordCount))
		}

	}

	updatePrometheusMetricsForIsolationsCount(results)

	if err := pc.updatePrometheusMetricsForDbIsolationsRequestedDiskSize(ctx); err != nil {
		return err
	}

	if err := pc.updatePrometheusMetricsForDbDiskUsage(ctx); err != nil {
		return err
	}

	return nil
}

// UpdatePrometheusMetrics manually updates Prometheus metrics
func (pc *PrometheusCollector) UpdatePrometheusMetrics(ctx context.Context) error {
	return pc.updatePrometheusMetrics(ctx)
}

func updatePrometheusMetricsForIsolationsCount(results []MetricsRow) {
	// Clear existing metrics to avoid stale data
	dbIsolationsCount.Reset()
	dbCollectionsCount.Reset()

	type isolationKey struct {
		IsoID        string
		SchemaPrefix string
	}
	isolationsToCollections := make(map[isolationKey]map[string]struct{})

	for _, row := range results {
		isolationKey := isolationKey{row.IsoID, row.SchemaPrefix}
		if _, exists := isolationsToCollections[isolationKey]; !exists {
			isolationsToCollections[isolationKey] = make(map[string]struct{})
		}

		if _, exists := isolationsToCollections[isolationKey][row.ColID]; !exists {
			isolationsToCollections[isolationKey][row.ColID] = struct{}{}
		}
	}

	dbIsolationsCount.WithLabelValues().Set(float64(len(isolationsToCollections)))

	for isolationKey, collections := range isolationsToCollections {
		dbCollectionsCount.WithLabelValues(isolationKey.IsoID, isolationKey.SchemaPrefix).Set(float64(len(collections)))
	}
}

func (pc *PrometheusCollector) updatePrometheusMetricsForDbIsolationsRequestedDiskSize(ctx context.Context) error {
	isoDiskSizes, err := pc.collector.getIsolationsRequestedSize(ctx)
	if err != nil {
		return err
	}

	// Clear existing metrics to avoid stale data
	dbIsolationsRequestedDiskSize.Reset()

	for _, isoDiskSize := range isoDiskSizes {
		labels := []string{isoDiskSize.IsolationID, isoDiskSize.SchemaPrefix}
		dbIsolationsRequestedDiskSize.WithLabelValues(labels...).Set(float64(isoDiskSize.RequestedDiskSize))
	}

	return nil
}

func (pc *PrometheusCollector) updatePrometheusMetricsForDbDiskUsage(ctx context.Context) error {
	isoDiskUsages, err := pc.collector.getDiskUsage(ctx)
	if err != nil {
		return err
	}

	// Clear existing metrics to avoid stale data
	dbDiskUsage.Reset()

	for _, isoDiskUsage := range isoDiskUsages {
		labels := []string{isoDiskUsage.IsolationID, isoDiskUsage.SchemaPrefix, isoDiskUsage.ProfileID, isoDiskUsage.TablesPrefix}
		dbDiskUsage.WithLabelValues(labels...).Set(float64(isoDiskUsage.DiskUsageBytes))
	}

	return nil
}
