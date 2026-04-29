// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package workers

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	// maintenanceWorkerProgress tracks the progress percentage (0-100) of maintenance workers
	maintenanceWorkerProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_maintenance_worker_progress",
			Help: "Progress percentage (0-100) of maintenance workers",
		},
		[]string{"worker_name"},
	)

	// dbConfigurationInfo exposes database configuration values as an info-style metric
	// Since Prometheus metrics are numeric, we use labels to expose string values
	dbConfigurationInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_db_configuration_info",
			Help: "Database configuration values exposed as labels (value is always 1)",
		},
		[]string{"key", "value"},
	)
)

func init() {
	prometheus.MustRegister(maintenanceWorkerProgress)
	prometheus.MustRegister(dbConfigurationInfo)
}

// SetWorkerProgress sets the progress percentage for a maintenance worker
func SetWorkerProgress(logger *zap.Logger, workerName string, progressPercent float64) {
	logger.Debug("Setting maintenance worker progress", zap.String("worker_name", workerName), zap.Float64("progress_percent", progressPercent))
	maintenanceWorkerProgress.WithLabelValues(workerName).Set(progressPercent)
}

// SetConfigurationValue sets a configuration key-value pair as a Prometheus metric
// The gauge value is always 1; the actual data is in the labels
func SetConfigurationValue(logger *zap.Logger, key string, value string) {
	logger.Debug("Setting DB configuration metric",
		zap.String("key", key),
		zap.String("value", value))

	// Reset all metrics for this key to avoid stale labels
	dbConfigurationInfo.DeletePartialMatch(prometheus.Labels{"key": key})

	// Set the new value (gauge is always 1, info is in labels)
	dbConfigurationInfo.WithLabelValues(key, value).Set(1)
}
