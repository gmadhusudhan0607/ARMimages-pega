// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

// DBMetric represents database metrics for a single isolation sent to PDC
type DBMetric struct {
	MetricType            string `json:"metricType"` // "DB"
	IsolationID           string `json:"isolationID"`
	DiskUsage             int64  `json:"diskUsage"`
	DocumentsCount        int64  `json:"documentsCount"`
	DocumentsModification string `json:"documentsModification,omitempty"` // ISO8601 format
}

// DBMetricsConfig holds configuration for DB metrics uploader
// All settings except upload interval are shared with usage metrics via Config
type DBMetricsConfig struct {
	UploadIntervalSeconds int
	UsageMetricsConfig    Config // Reuse usage metrics config including Enabled flag
}

// DefaultDBMetricsConfig returns default configuration for DB metrics uploader
func DefaultDBMetricsConfig() DBMetricsConfig {
	return DBMetricsConfig{
		UploadIntervalSeconds: 3600, // 1 hour
		UsageMetricsConfig:    DefaultConfig(),
	}
}
