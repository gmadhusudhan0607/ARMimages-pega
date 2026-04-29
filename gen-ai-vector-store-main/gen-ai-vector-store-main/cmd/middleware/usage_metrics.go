// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

/*
 * Middleware: Usage Metrics Collection
 * ------------------------------------
 * Purpose: Collects metrics for semantic search requests and sends them to usage data endpoints.
 * Usage: Add UsageMetricsMiddleware to your Gin middleware chain after ServiceMetricsMiddleware.
 */

package middleware

import (
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/usagemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var usageMetricsLogger = log.GetNamedLogger("middleware.usage_metrics")

const metricTypeSemanticSearchRequest = "SemanticSearchRequest"

// UsageMetricsMiddleware collects metrics for semantic search requests and queues them for usage data upload
func UsageMetricsMiddleware(isoManager isolations.IsoManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record the start time
		startTime := time.Now()

		// Process the request
		c.Next()

		// Only process semantic search endpoints
		if !isSemanticSearchEndpoint(c) {
			return
		}

		endTime := time.Now()

		// Extract isolation and collection info
		isolationID := c.Param("isolationID")
		collectionID := getUsageMetricsCollectionID(c)

		if isolationID == "" || collectionID == "" {
			usageMetricsLogger.Debug("Missing isolation or collection ID, skipping usage metrics",
				zap.String("isolationID", isolationID),
				zap.String("collectionID", collectionID))
			return
		}

		// Extract metrics from service context
		svcMetrics := servicemetrics.FromContext(c.Request.Context())
		if svcMetrics == nil {
			usageMetricsLogger.Debug("Service metrics not found in context, skipping usage metrics",
				zap.String("isolationID", isolationID))
			return
		}

		// Build usage metric
		metric := buildUsageMetric(c, svcMetrics, isolationID, collectionID, startTime, endTime)

		// Add to collector queue
		collector := usagemetrics.GetCollector()
		collector.AddMetric(metric)

		usageMetricsLogger.Debug("Added usage metric to queue",
			zap.String("isolationID", isolationID),
			zap.String("collectionID", collectionID),
			zap.String("endpoint", metric.Endpoint))
	}
}

// isSemanticSearchEndpoint checks if the current request is a semantic search endpoint
func isSemanticSearchEndpoint(c *gin.Context) bool {
	if c.Request.Method != "POST" {
		return false
	}

	// Get normalized path (set by PathNormalizationMiddleware)
	// Fall back to raw URL path if normalization wasn't applied
	path, exists := c.Get("normalizedPath")
	if !exists {
		path = c.Request.URL.Path
	}

	// Ensure path is a string
	pathStr, ok := path.(string)
	if !ok {
		pathStr = c.Request.URL.Path
	}

	// Check for semantic search endpoints:
	// - POST /v1/:isolationID/collections/:collectionName/query/chunks
	// - POST /v1/:isolationID/collections/:collectionName/query/documents
	return strings.Contains(pathStr, "/query/chunks") || strings.Contains(pathStr, "/query/documents")
}

// getUsageMetricsCollectionID extracts collection ID from gin context parameters
func getUsageMetricsCollectionID(c *gin.Context) string {
	// Try collectionName first (v1 API)
	if collectionName := c.Param("collectionName"); collectionName != "" {
		return collectionName
	}
	// Fall back to collectionID (v2 API)
	return c.Param("collectionID")
}

// getEndpointName extracts the endpoint name from the path
func getEndpointName(c *gin.Context) string {
	path, exists := c.Get("normalizedPath")
	if !exists {
		path = c.Request.URL.Path
	}

	pathStr, ok := path.(string)
	if !ok {
		pathStr = c.Request.URL.Path
	}

	if strings.Contains(pathStr, "/query/chunks") {
		return "query_chunks"
	} else if strings.Contains(pathStr, "/query/documents") {
		return "query_documents"
	}

	return "unknown"
}

// buildUsageMetric constructs a SemanticSearchMetric from the request context and service metrics
func buildUsageMetric(c *gin.Context, svcMetrics *servicemetrics.ServiceMetrics, isolationID, collectionID string, startTime, endTime time.Time) usagemetrics.SemanticSearchMetric {
	metric := usagemetrics.SemanticSearchMetric{
		MetricType:   metricTypeSemanticSearchRequest,
		IsolationID:  isolationID,
		CollectionID: collectionID,
		Endpoint:     getEndpointName(c),
		StatusCode:   c.Writer.Status(),

		// Request metrics
		RequestDurationMs: svcMetrics.RequestMetrics.Duration().Milliseconds(),
		StartTime:         startTime.Format(time.RFC3339),
		EndTime:           endTime.Format(time.RFC3339),

		// DB metrics
		DbQueryTimeMs: svcMetrics.DbMetrics.QueryExecutionTime().Milliseconds(),

		// Response metrics
		ItemsReturned: svcMetrics.ResponseMetrics.ItemsReturned(),
	}

	// Add embedding metrics if available
	embeddingMetrics := svcMetrics.EmbeddingMetrics.GetMetrics()
	if len(embeddingMetrics) > 0 {
		// Use metrics from the first embedding model
		embMetric := embeddingMetrics[0]
		metric.ModelID = embMetric.ModelID
		metric.ModelVersion = embMetric.ModelVersion
		metric.EmbeddingTimeMs = embMetric.TotalExecutionTime.Milliseconds()
		metric.EmbeddingCallsCount = embMetric.TotalMeasurementCount
		metric.EmbeddingRetryCount = embMetric.TotalRetryCount
	}

	// Add processing overhead metrics
	processingDurationMs, overheadMs, embNetOverheadMs := svcMetrics.CalculateProcessingOverheads()
	metric.ProcessingDurationMs = processingDurationMs
	metric.OverheadMs = overheadMs
	metric.EmbeddingNetOverheadMs = embNetOverheadMs

	return metric
}
