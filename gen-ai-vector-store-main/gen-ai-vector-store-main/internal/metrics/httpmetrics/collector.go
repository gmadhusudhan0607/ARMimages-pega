/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package httpmetrics

import (
	"time"
)

// Collector handles HTTP metrics collection
type Collector struct {
	config  Config
	backend HTTPMetricsCollector
}

// NewCollector creates a new HTTP metrics collector
func NewCollector(config Config, backend HTTPMetricsCollector) *Collector {
	return &Collector{
		config:  config,
		backend: backend,
	}
}

// RecordRequest records metrics for an HTTP request
func (c *Collector) RecordRequest(metrics HTTPMetrics) {
	if !c.config.Enabled {
		return
	}

	c.backend.RecordRequest(metrics.Path, metrics.Method, metrics.StatusCode, metrics.Duration)
}

// RecordActiveConnection records active connection changes
func (c *Collector) RecordActiveConnection(path string, delta int) {
	if !c.config.Enabled {
		return
	}

	c.backend.RecordActiveConnection(path, delta)
}

// RecordDBQueryDuration records database query duration
func (c *Collector) RecordDBQueryDuration(path, method, code string, duration time.Duration) {
	if !c.config.Enabled || !c.config.IncludeDBMetrics {
		return
	}

	c.backend.RecordDBQueryDuration(path, method, code, duration)
}

// RecordReturnedItems records the number of items returned in the response
func (c *Collector) RecordReturnedItems(path, method, code string, itemCount int) {
	if !c.config.Enabled {
		return
	}

	c.backend.RecordReturnedItems(path, method, code, itemCount)
}

// IsEnabled returns whether metrics collection is enabled
func (c *Collector) IsEnabled() bool {
	return c.config.Enabled
}

// Config returns the current configuration
func (c *Collector) Config() Config {
	return c.config
}
