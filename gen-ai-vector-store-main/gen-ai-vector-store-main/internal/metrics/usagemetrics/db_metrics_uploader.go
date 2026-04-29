// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/opsmetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
	"go.uber.org/zap"
)

const (
	dbMetricsUploaderLoggerName  = "usagemetrics.db_metrics_uploader"
	dbMetricsPdcSenderLoggerName = "usagemetrics.db_metrics_uploader.pdc"
)

// DBMetricsUploader handles periodic uploading of database metrics to PDC
type DBMetricsUploader struct {
	database   db.Database
	isoManager IsolationsGetter
	pdcSender  *PDCSender[DBMetric]
	config     DBMetricsConfig
	logger     *zap.Logger
}

// IsolationsGetter interface for getting all isolations
type IsolationsGetter interface {
	GetIsolations(ctx context.Context) ([]*isolations.Details, error)
}

// NewDBMetricsUploader creates a new database metrics uploader
func NewDBMetricsUploader(database db.Database, isoManager IsolationsGetter) *DBMetricsUploader {
	config := loadDBMetricsConfig()

	pdcSender := NewPDCSender[DBMetric](PDCSenderConfig{
		RequestTimeoutSeconds: config.UsageMetricsConfig.RequestTimeoutSeconds,
		MaxPayloadSizeBytes:   config.UsageMetricsConfig.MaxPayloadSizeBytes,
	}, dbMetricsPdcSenderLoggerName)

	return &DBMetricsUploader{
		database:   database,
		isoManager: isoManager,
		pdcSender:  pdcSender,
		config:     config,
		logger:     log.GetNamedLogger(dbMetricsUploaderLoggerName),
	}
}

// loadDBMetricsConfig loads configuration from environment variables
func loadDBMetricsConfig() DBMetricsConfig {
	config := DefaultDBMetricsConfig()

	// Load usage metrics config (shared settings)
	config.UsageMetricsConfig = Config{
		Enabled:               helpers.IsUsageMetricsEnabled(),
		UploadIntervalSeconds: helpers.GetUsageMetricsUploadIntervalSeconds(),
		MaxPayloadSizeBytes:   helpers.GetUsageMetricsMaxPayloadSizeBytes(),
		RetryCount:            helpers.GetUsageMetricsRetryCount(),
		RequestTimeoutSeconds: helpers.GetUsageMetricsRequestTimeoutSeconds(),
	}

	// DB metrics specific settings - only upload interval
	if interval := helpers.GetEnvOrDefault("DB_METRICS_PDC_UPLOAD_INTERVAL_SECONDS", ""); interval != "" {
		config.UploadIntervalSeconds = helpers.ParseIntOrDefault(interval, config.UploadIntervalSeconds)
	}

	return config
}

// StartBackgroundUploader starts the periodic database metrics upload process
// This should be called in a separate goroutine
func (u *DBMetricsUploader) StartBackgroundUploader(ctx context.Context) error {
	if !u.config.UsageMetricsConfig.Enabled {
		u.logger.Info("DB metrics upload to PDC disabled (USAGE_METRICS_ENABLED=false)")
		return nil
	}

	u.logger.Info("Starting DB metrics background uploader to PDC",
		zap.Int("intervalSeconds", u.config.UploadIntervalSeconds))

	ticker := time.NewTicker(time.Duration(u.config.UploadIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			u.logger.Info("DB metrics uploader stopping due to context cancellation")
			return ctx.Err()
		case <-ticker.C:
			u.collectAndUploadMetrics(ctx)
		}
	}
}

// collectAndUploadMetrics collects DB metrics for all isolations and uploads them to PDC
func (u *DBMetricsUploader) collectAndUploadMetrics(ctx context.Context) {
	u.logger.Debug("Collecting DB metrics for all isolations")

	// Get all isolations
	isolationsList, err := u.isoManager.GetIsolations(ctx)
	if err != nil {
		u.logger.Error("Failed to get isolations list",
			zap.Error(err))
		return
	}

	if len(isolationsList) == 0 {
		u.logger.Debug("No isolations found")
		return
	}

	u.logger.Info("Collecting DB metrics for isolations",
		zap.Int("isolationsCount", len(isolationsList)))

	// Collect metrics for each isolation
	var metricsToUpload []DBMetric
	isolationsByPDCURL := make(map[string][]DBMetric)

	for _, iso := range isolationsList {
		if iso.PDCEndpointURL == "" {
			u.logger.Debug("Skipping isolation without PDC endpoint URL",
				zap.String("isolationID", iso.ID))
			continue
		}

		metric, err := u.collectIsolationMetrics(ctx, iso.ID)
		if err != nil {
			u.logger.Error("Failed to collect metrics for isolation",
				zap.String("isolationID", iso.ID),
				zap.Error(err))
			continue
		}

		// Group metrics by PDC endpoint URL
		isolationsByPDCURL[iso.PDCEndpointURL] = append(isolationsByPDCURL[iso.PDCEndpointURL], *metric)
		metricsToUpload = append(metricsToUpload, *metric)
	}

	if len(metricsToUpload) == 0 {
		u.logger.Debug("No DB metrics to upload")
		return
	}

	u.logger.Info("Uploading DB metrics to PDC",
		zap.Int("metricsCount", len(metricsToUpload)),
		zap.Int("uniquePDCEndpoints", len(isolationsByPDCURL)))

	// Upload metrics grouped by PDC endpoint URL
	for pdcURL, metrics := range isolationsByPDCURL {
		if err := u.uploadMetrics(ctx, pdcURL, metrics); err != nil {
			u.logger.Error("Failed to upload DB metrics to PDC",
				zap.String("pdcURL", pdcURL),
				zap.Int("metricsCount", len(metrics)),
				zap.Error(err))
		} else {
			u.logger.Info("Successfully uploaded DB metrics to PDC",
				zap.String("pdcURL", pdcURL),
				zap.Int("metricsCount", len(metrics)))
		}
	}
}

// collectIsolationMetrics collects database metrics for a single isolation
func (u *DBMetricsUploader) collectIsolationMetrics(ctx context.Context, isolationID string) (*DBMetric, error) {
	// Create opsmetrics instance for this isolation
	opsMetrics := opsmetrics.NewOpsMetrics(u.database, isolationID)

	// Get isolation metrics (diskUsage, documentsCount, documentsModification)
	// Pass empty slice to get all metrics
	metrics, err := opsMetrics.GetIsolationMetrics([]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get isolation metrics: %w", err)
	}

	// Convert to DBMetric
	dbMetric := &DBMetric{
		MetricType:     "DB",
		IsolationID:    isolationID,
		DiskUsage:      metrics.DiskUsage,
		DocumentsCount: metrics.DocumentsCount,
	}

	// Convert modification time to ISO8601 format if available
	if metrics.DocumentsModification != nil {
		dbMetric.DocumentsModification = metrics.DocumentsModification.Format(time.RFC3339)
	}

	u.logger.Debug("Collected DB metrics for isolation",
		zap.String("isolationID", isolationID),
		zap.Int64("diskUsage", dbMetric.DiskUsage),
		zap.Int64("documentsCount", dbMetric.DocumentsCount),
		zap.String("documentsModification", dbMetric.DocumentsModification))

	return dbMetric, nil
}

// uploadMetrics uploads DB metrics to the specified PDC endpoint URL using PDCSender
func (u *DBMetricsUploader) uploadMetrics(ctx context.Context, pdcURL string, metrics []DBMetric) error {
	// PDCSender handles chunking automatically based on payload size
	return u.pdcSender.Send(ctx, pdcURL, metrics)
}
