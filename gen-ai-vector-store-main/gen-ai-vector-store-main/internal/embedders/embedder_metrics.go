/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedders

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	modelHttpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vector_store_model_http_requests_total",
			Help: "Total number of HTTP requests made to embedding model",
		},
		[]string{"host", "path", "method", "code", "model", "version"},
	)
)

var (
	modelRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vector_store_model_http_request_duration_seconds",
			Help:    "A histogram of the embedding model HTTP request durations in seconds ",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 20, 30, 60}, // Define bucket boundaries
		},
		[]string{"host", "path", "method", "model", "version"},
	)
)

var (
	modelActiveConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vector_store_model_http_active_connections",
			Help: "Number of active embedding model HTTP requests",
		},
		[]string{"model", "version"},
	)
)

var (
	modelHttpRetriesCount = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vector_store_model_http_retries_count",
			Help:    "Total number of HTTP retries made to embedding model",
			Buckets: []float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		[]string{"host", "path", "method", "code", "model", "version"},
	)
)

func init() {
	prometheus.MustRegister(modelHttpRequestsTotal)
	prometheus.MustRegister(modelRequestDuration)
	prometheus.MustRegister(modelActiveConnections)
	prometheus.MustRegister(modelHttpRetriesCount)
}

func AddModelHttpMetrics(hostName, path, method, code, model, modelVersion string, duration float64, callComplete bool, retries int) {

	if callComplete {
		modelHttpRequestsTotal.WithLabelValues(hostName, path, method, code, model, modelVersion).Inc()
		modelRequestDuration.WithLabelValues(hostName, path, method, model, modelVersion).Observe(duration)
		modelActiveConnections.WithLabelValues(model, modelVersion).Dec()
		modelHttpRetriesCount.WithLabelValues(hostName, path, method, code, model, modelVersion).Observe(float64(retries))
	} else {
		modelActiveConnections.WithLabelValues(model, modelVersion).Inc()

	}

}
