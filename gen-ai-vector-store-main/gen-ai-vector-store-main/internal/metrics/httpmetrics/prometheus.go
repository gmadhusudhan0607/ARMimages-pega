/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package httpmetrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusCollector implements HTTPMetricsCollector for Prometheus
type PrometheusCollector struct {
	httpRequestsTotal *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	activeConnections *prometheus.GaugeVec
	dbQueryDuration   *prometheus.HistogramVec
	returnedItems     *prometheus.HistogramVec
}

// NewPrometheusCollector creates a new Prometheus HTTP metrics collector
func NewPrometheusCollector(config Config) *PrometheusCollector {
	collector := &PrometheusCollector{
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"path", "method", "code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "A histogram of the HTTP request durations in seconds",
				Buckets: config.RequestBuckets,
			},
			[]string{"path", "method", "code"},
		),
		activeConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_active_connections",
				Help: "Number of active HTTP requests",
			},
			[]string{"path"},
		),
		returnedItems: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_returned_items",
				Help:    "A histogram of the number of items returned in HTTP responses",
				Buckets: config.ReturnedItemsBuckets,
			},
			[]string{"path", "method", "code"},
		),
	}

	if config.IncludeDBMetrics {
		collector.dbQueryDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "vector_store_db_query_duration_seconds",
				Help:    "A histogram of the DB query durations in seconds",
				Buckets: config.DBQueryBuckets,
			},
			[]string{"path", "method", "code"},
		)
	}

	return collector
}

// Register registers all metrics with Prometheus
func (p *PrometheusCollector) Register() error {
	prometheus.MustRegister(p.httpRequestsTotal)
	prometheus.MustRegister(p.requestDuration)
	prometheus.MustRegister(p.activeConnections)
	prometheus.MustRegister(p.returnedItems)

	if p.dbQueryDuration != nil {
		prometheus.MustRegister(p.dbQueryDuration)
	}

	return nil
}

// RecordRequest implements HTTPMetricsCollector
func (p *PrometheusCollector) RecordRequest(path, method, code string, duration time.Duration) {
	p.httpRequestsTotal.WithLabelValues(path, method, code).Inc()
	p.requestDuration.WithLabelValues(path, method, code).Observe(duration.Seconds())
}

// RecordActiveConnection implements HTTPMetricsCollector
func (p *PrometheusCollector) RecordActiveConnection(path string, delta int) {
	if delta > 0 {
		p.activeConnections.WithLabelValues(path).Inc()
	} else {
		p.activeConnections.WithLabelValues(path).Dec()
	}
}

// RecordDBQueryDuration implements HTTPMetricsCollector
func (p *PrometheusCollector) RecordDBQueryDuration(path, method, code string, duration time.Duration) {
	if p.dbQueryDuration != nil {
		p.dbQueryDuration.WithLabelValues(path, method, code).Observe(duration.Seconds())
	}
}

// RecordReturnedItems implements HTTPMetricsCollector
func (p *PrometheusCollector) RecordReturnedItems(path, method, code string, itemCount int) {
	p.returnedItems.WithLabelValues(path, method, code).Observe(float64(itemCount))
}
