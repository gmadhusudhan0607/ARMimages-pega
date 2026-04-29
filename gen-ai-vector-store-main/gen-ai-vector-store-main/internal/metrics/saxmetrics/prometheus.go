// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package saxmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// PrometheusCollector implements SAXMetricsCollector for Prometheus
type PrometheusCollector struct {
	saxCacheHits     prometheus.Counter
	saxCacheMisses   prometheus.Counter
	saxCacheSize     prometheus.Gauge
	saxCacheHitRatio prometheus.GaugeFunc
}

// NewPrometheusCollector creates a new Prometheus SAX metrics collector
func NewPrometheusCollector() *PrometheusCollector {
	collector := &PrometheusCollector{
		saxCacheHits: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "vector_store_sax_validation_cache_hits_total",
				Help: "Total number of SAX validation cache hits",
			},
		),
		saxCacheMisses: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "vector_store_sax_validation_cache_misses_total",
				Help: "Total number of SAX validation cache misses",
			},
		),
		saxCacheSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "vector_store_sax_validation_cache_size",
				Help: "Current number of cached SAX tokens",
			},
		),
	}

	// GaugeFunc that calculates ratio dynamically from counter values
	collector.saxCacheHitRatio = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "vector_store_sax_validation_cache_hit_ratio",
			Help: "Cache hit ratio for SAX caching validation",
		},
		func() float64 {
			return collector.calculateCacheHitRatio()
		},
	)

	return collector
}

func (p *PrometheusCollector) Register() {
	prometheus.MustRegister(p.saxCacheHits)
	prometheus.MustRegister(p.saxCacheMisses)
	prometheus.MustRegister(p.saxCacheSize)
	prometheus.MustRegister(p.saxCacheHitRatio)
}

// RecordCacheHit implements SAXMetricsCollector
func (p *PrometheusCollector) RecordCacheHit() {
	p.saxCacheHits.Inc()
}

// RecordCacheMiss implements SAXMetricsCollector
func (p *PrometheusCollector) RecordCacheMiss() {
	p.saxCacheMisses.Inc()
}

// RecordCacheSize implements SAXMetricsCollector
func (p *PrometheusCollector) RecordCacheSize(size int) {
	p.saxCacheSize.Set(float64(size))
}

// calculateCacheHitRatio calculates the cache hit ratio from counter values
// This function is called by the GaugeFunc when Prometheus scrapes the metric
func (p *PrometheusCollector) calculateCacheHitRatio() float64 {
	var metric dto.Metric

	// Get hits count from the counter
	if err := p.saxCacheHits.Write(&metric); err != nil {
		return 0
	}
	hits := metric.GetCounter().GetValue()

	// Get misses count from the counter
	metric.Reset()
	if err := p.saxCacheMisses.Write(&metric); err != nil {
		return 0
	}
	misses := metric.GetCounter().GetValue()

	// Calculate ratio
	total := hits + misses
	if total > 0 {
		return hits / total
	}

	return 0
}
