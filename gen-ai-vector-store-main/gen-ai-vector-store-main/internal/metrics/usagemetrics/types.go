// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

// SemanticSearchMetric represents metrics for semantic search requests sent to usage data endpoints
type SemanticSearchMetric struct {
	MetricType   string `json:"metricType"` // "SemanticSearchRequest"
	IsolationID  string `json:"isolationID"`
	CollectionID string `json:"collectionID"`
	Endpoint     string `json:"endpoint"` // "query_documents" | "query_chunks"
	StatusCode   int    `json:"statusCode"`

	// Request metrics
	RequestDurationMs int64  `json:"requestDurationMs"`
	StartTime         string `json:"startTime"` // ISO8601
	EndTime           string `json:"endTime"`   // ISO8601

	// DB metrics
	DbQueryTimeMs int64 `json:"dbQueryTimeMs"`

	// Embedding metrics
	ModelID             string `json:"modelId,omitempty"`
	ModelVersion        string `json:"modelVersion,omitempty"`
	EmbeddingTimeMs     int64  `json:"embeddingTimeMs,omitempty"`
	EmbeddingCallsCount int    `json:"embeddingCallsCount,omitempty"`
	EmbeddingRetryCount int    `json:"embeddingRetryCount,omitempty"`

	// Response metrics
	ItemsReturned int `json:"itemsReturned"`

	// Processing overhead metrics
	ProcessingDurationMs   int64 `json:"processingDurationMs,omitempty"`
	OverheadMs             int64 `json:"overheadMs,omitempty"`
	EmbeddingNetOverheadMs int64 `json:"embeddingNetOverheadMs,omitempty"`

	// To prevent for infinite requeue
	retryCount int `json:"-"`
}

// UsageDataPayload represents the payload structure sent to usage data endpoints
type UsageDataPayload[T any] struct {
	Data     []T               `json:"data"`
	Metadata UsageDataMetadata `json:"metadata"`
}

// UsageDataMetadata represents metadata included in usage data payload
type UsageDataMetadata struct {
	SegmentNumber int    `json:"segmentNumber"`
	SegmentsTotal int    `json:"segmentsTotal"`
	Source        string `json:"source"` // "GenAIVectorStore"
}

// Config holds configuration for usage metrics collector
type Config struct {
	Enabled               bool
	UploadIntervalSeconds int
	MaxPayloadSizeBytes   int
	RetryCount            int
	RequestTimeoutSeconds int
}

// DefaultConfig returns default configuration for usage metrics
func DefaultConfig() Config {
	return Config{
		Enabled:               false,
		UploadIntervalSeconds: 3600,
		MaxPayloadSizeBytes:   819200, // 800KB
		RetryCount:            3,
		RequestTimeoutSeconds: 30,
	}
}
