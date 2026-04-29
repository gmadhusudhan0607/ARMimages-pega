// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package saxmetrics

// Collector handles SAX JWT validation metrics collection
type Collector struct {
	collector SAXMetricsCollector
	enabled   bool
}

// NewCollector creates a new SAX metrics collector
func NewCollector(collector SAXMetricsCollector, enabled bool) *Collector {
	return &Collector{
		collector: collector,
		enabled:   enabled,
	}
}

// IsEnabled returns whether metrics collection is enabled
func (c *Collector) IsEnabled() bool {
	return c.enabled
}

// RecordCacheHit records a cache hit event
func (c *Collector) RecordCacheHit() {
	if !c.enabled || c.collector == nil {
		return
	}
	c.collector.RecordCacheHit()
}

// RecordCacheMiss records a cache miss event
func (c *Collector) RecordCacheMiss() {
	if !c.enabled || c.collector == nil {
		return
	}
	c.collector.RecordCacheMiss()
}

// RecordCacheSize records the current cache size
func (c *Collector) RecordCacheSize(size int) {
	if !c.enabled || c.collector == nil {
		return
	}
	c.collector.RecordCacheSize(size)
}
