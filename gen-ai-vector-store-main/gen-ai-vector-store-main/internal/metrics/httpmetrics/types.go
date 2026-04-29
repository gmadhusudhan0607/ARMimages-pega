/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package httpmetrics

import (
	"fmt"
	"time"
)

// HTTPMetricsCollector defines the interface for HTTP metrics collection
type HTTPMetricsCollector interface {
	RecordRequest(path, method, code string, duration time.Duration)
	RecordActiveConnection(path string, delta int) // +1 for start, -1 for end
	RecordDBQueryDuration(path, method, code string, duration time.Duration)
	RecordReturnedItems(path, method, code string, itemCount int)
}

// HTTPMetrics holds the metrics data for a single HTTP request
type HTTPMetrics struct {
	Path        string
	Method      string
	StatusCode  string
	Duration    time.Duration
	DBQueryTime time.Duration
	StartTime   time.Time
}

// Config holds configuration for HTTP metrics
type Config struct {
	Enabled              bool      // Whether HTTP metrics are enabled
	RequestBuckets       []float64 // Histogram buckets for request duration
	DBQueryBuckets       []float64 // Histogram buckets for DB query duration
	IncludeDBMetrics     bool      // Whether to collect DB query metrics
	ReturnedItemsBuckets []float64 // Histogram buckets for number of items returned in responses
}

// DefaultConfig returns sensible defaults for HTTP metrics configuration
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		// we expect requests to be served within a few seconds. whatever goes beyond the 30s will time out from client side anyway.
		RequestBuckets:       []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 20, 30, 60},
		DBQueryBuckets:       []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 20, 30, 60},
		IncludeDBMetrics:     true,
		ReturnedItemsBuckets: []float64{0, 1, 2, 5, 10, 20, 50, 100, 200},
	}
}

// NewHTTPMetrics creates a new HTTPMetrics instance
func NewHTTPMetrics(path, method string) HTTPMetrics {
	return HTTPMetrics{
		Path:      path,
		Method:    method,
		StartTime: time.Now(),
	}
}

// Finish completes the HTTP metrics collection
func (m *HTTPMetrics) Finish(statusCode int) {
	m.StatusCode = fmt.Sprintf("%d", statusCode)
	m.Duration = time.Since(m.StartTime)
}

// SetDBQueryTime sets the database query duration
func (m *HTTPMetrics) SetDBQueryTime(duration time.Duration) {
	m.DBQueryTime = duration
}
