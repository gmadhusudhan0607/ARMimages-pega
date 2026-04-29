// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package saxmetrics

// SAXMetricsCollector defines the interface for collecting SAX JWT validation metrics
type SAXMetricsCollector interface {
	// RecordCacheHit records a cache hit event
	RecordCacheHit()

	// RecordCacheMiss records a cache miss event
	RecordCacheMiss()

	// RecordCacheSize records the current cache size
	RecordCacheSize(size int)
}
