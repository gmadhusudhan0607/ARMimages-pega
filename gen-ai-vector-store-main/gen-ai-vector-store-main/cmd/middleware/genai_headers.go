/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Response Header Enrichment
 * --------------------------------------
 * Purpose: Enriches HTTP response headers with timing and custom metadata before sending the response.
 * Usage: Add GenaiResponseHeadersMiddleware to your Gin middleware chain to automatically set headers such as request duration.
 * Configuration: Uses a custom response writer to inject headers just before writing the response.
 */

package middleware

import (
	"context"
	"strconv"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/dbmetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/gin-gonic/gin"
)

var genaiHeadersLogger = log.GetNamedLogger("genai-headers-middleware")

// GenaiResponseHeadersMiddleware wraps the Gin response writer to inject custom headers (e.g., request duration)
// before the response is sent. It uses a custom headerResponseWriter to set headers at the appropriate time.
func GenaiResponseHeadersMiddleware(c *gin.Context) {
	// Create a custom response writer that will set headers just before writing
	originalWriter := c.Writer
	c.Writer = &headerResponseWriter{
		ResponseWriter: originalWriter,
		requestContext: c.Request.Context(),
		ginContext:     c,
	}

	// Execute the handler and other middleware
	c.Next()
}

// headerResponseWriter wraps gin.ResponseWriter to set headers just before writing response
// It ensures custom headers are set only once, and supports both WriteHeader and Write methods.
type headerResponseWriter struct {
	gin.ResponseWriter
	requestContext context.Context
	ginContext     *gin.Context
	headersSet     bool
}

func (w *headerResponseWriter) WriteHeader(statusCode int) {
	if !w.headersSet {
		servicemetrics.FromContext(w.requestContext).RequestMetrics.StopProcessing()

		w.setCustomHeaders()
		w.headersSet = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *headerResponseWriter) Write(data []byte) (int, error) {
	if !w.headersSet {
		w.setCustomHeaders()
		w.headersSet = true
	}
	return w.ResponseWriter.Write(data)
}

func (w *headerResponseWriter) setCustomHeaders() {
	// Set other headers based on service metrics
	svcMetrics := servicemetrics.FromContext(w.requestContext)
	if svcMetrics == nil {
		genaiHeadersLogger.Warn("Service metrics not found in context; skipping some headers")
		return
	}
	genaiHeadersLogger.Debug("svcMetrics found in context, setting headers...")

	setDBHeaders(w, svcMetrics)
	setEmbeddingHeaders(w, svcMetrics)
	setResponseCountHeader(w, svcMetrics)
	setRequestDurationHeader(w, svcMetrics)
	setGatewayHeaders(w, svcMetrics)
	setProcessingOverheadHeaders(w, svcMetrics)
	setCollectionMetricsHeaders(w)
}

func setDBHeaders(w *headerResponseWriter, svcMetrics *servicemetrics.ServiceMetrics) {
	queryTime := svcMetrics.DbMetrics.QueryExecutionTime()
	genaiHeadersLogger.Debug("DB QueryExecutionTime value", zap.Int64("queryTimeMs", queryTime.Milliseconds()))
	// Remove always-true condition, always set header
	w.Header().Set(headers.DbQueryTimeMs, strconv.FormatInt(queryTime.Milliseconds(), 10))
	genaiHeadersLogger.Debug("Added header", zap.String(headers.DbQueryTimeMs, strconv.FormatInt(queryTime.Milliseconds(), 10)))
}

func setEmbeddingHeaders(w *headerResponseWriter, svcMetrics *servicemetrics.ServiceMetrics) {
	embeddingMetrics := svcMetrics.EmbeddingMetrics.GetMetrics()
	if len(embeddingMetrics) == 0 {
		genaiHeadersLogger.Debug("No embedding metrics found, not adding embedding headers")
		return
	}

	// Set all headers
	modelID := embeddingMetrics[0].ModelID
	modelVersion := embeddingMetrics[0].ModelVersion
	embeddingTimeMs := strconv.FormatInt(embeddingMetrics[0].TotalExecutionTime.Milliseconds(), 10)
	embeddingCallsCount := strconv.Itoa(embeddingMetrics[0].TotalMeasurementCount)
	embeddingRetryCount := strconv.Itoa(embeddingMetrics[0].TotalRetryCount)

	w.Header().Set(headers.ModelId, modelID)
	w.Header().Set(headers.ModelVersion, modelVersion)
	w.Header().Set(headers.EmbeddingTimeMs, embeddingTimeMs)
	w.Header().Set(headers.EmbeddingCallsCount, embeddingCallsCount)
	w.Header().Set(headers.EmbeddingRetryCount, embeddingRetryCount)

	// Log all headers once
	genaiHeadersLogger.Debug("Added embedding headers",
		zap.String(headers.ModelId, modelID),
		zap.String(headers.ModelVersion, modelVersion),
		zap.String(headers.EmbeddingTimeMs, embeddingTimeMs),
		zap.String(headers.EmbeddingCallsCount, embeddingCallsCount),
		zap.String(headers.EmbeddingRetryCount, embeddingRetryCount))
}

func setResponseCountHeader(w *headerResponseWriter, svcMetrics *servicemetrics.ServiceMetrics) {
	itemsReturned := 0
	if svcMetrics != nil {
		itemsReturned = svcMetrics.ResponseMetrics.ItemsReturned()
	}
	genaiHeadersLogger.Debug("Response metrics", zap.Bool("metricsNotNil", svcMetrics != nil), zap.Int("itemsReturned", itemsReturned))
	if itemsReturned != 0 {
		w.Header().Set(headers.ResponseReturnedItemsCount, strconv.Itoa(itemsReturned))
		genaiHeadersLogger.Debug("Added header", zap.String(headers.ResponseReturnedItemsCount, strconv.Itoa(itemsReturned)))
	} else {
		genaiHeadersLogger.Debug("Response is nil or ItemsReturned is zero, not adding response count header")
	}
}

func setRequestDurationHeader(w *headerResponseWriter, svcMetrics *servicemetrics.ServiceMetrics) {
	duration := svcMetrics.RequestMetrics.Duration()
	w.Header().Set(headers.RequestDurationMs, strconv.FormatInt(duration.Milliseconds(), 10))

	genaiHeadersLogger.Debug("Request metrics", zap.Bool("metricsNotNil", svcMetrics != nil), zap.Duration("requestDuration", duration))

}

func setGatewayHeaders(w *headerResponseWriter, svcMetrics *servicemetrics.ServiceMetrics) {
	gatewayHeaders := svcMetrics.GatewayMetrics.GetHeaders()
	for headerName, headerValue := range gatewayHeaders {
		w.Header().Set(headerName, headerValue)
		genaiHeadersLogger.Debug("Added gateway header", zap.String(headerName, headerValue))
	}
}

func setProcessingOverheadHeaders(w *headerResponseWriter, svcMetrics *servicemetrics.ServiceMetrics) {
	processingDurationMs, overheadMs, embNetOverheadMs := svcMetrics.CalculateProcessingOverheads()

	w.Header().Set(headers.ProcessingDurationMs, strconv.FormatInt(processingDurationMs, 10))
	w.Header().Set(headers.OverheadMs, strconv.FormatInt(overheadMs, 10))
	w.Header().Set(headers.EmbeddingNetOverheadMs, strconv.FormatInt(embNetOverheadMs, 10))

	genaiHeadersLogger.Debug("Added processing overhead headers",
		zap.String(headers.ProcessingDurationMs, strconv.FormatInt(processingDurationMs, 10)),
		zap.String(headers.OverheadMs, strconv.FormatInt(overheadMs, 10)),
		zap.String(headers.EmbeddingNetOverheadMs, strconv.FormatInt(embNetOverheadMs, 10)))
}

func setCollectionMetricsHeaders(w *headerResponseWriter) {
	isolationID, collectionID := extractIDsFromContext(w.ginContext)
	if isolationID == "" || collectionID == "" {
		genaiHeadersLogger.Debug("isolationID or collectionID is empty, not setting collection metrics headers")
		return
	}

	documentCount, embeddingCount := getCollectionMetrics(w, isolationID, collectionID)
	genaiHeadersLogger.Debug("setting collection metrics headers",
		zap.Int64("documentCount", documentCount),
		zap.Int64("embeddingCount", embeddingCount))

	w.Header().Set(headers.DocumentsCount, strconv.FormatInt(documentCount, 10))
	w.Header().Set(headers.VectorsCount, strconv.FormatInt(embeddingCount, 10))
}

// extractIDsFromContext gets isolationID and collectionID from gin.Context
func extractIDsFromContext(c *gin.Context) (string, string) {
	if c == nil {
		genaiHeadersLogger.Debug("ginContext is nil, not extracting IDs")
		return "", ""
	}
	isolationID := c.Param("isolationID")
	collectionName := c.Param("collectionName")
	collectionID := c.Param("collectionID")
	if collectionID == "" {
		collectionID = collectionName
	}
	genaiHeadersLogger.Debug("extractIDsFromContext called",
		zap.String("isolationID", isolationID),
		zap.String("collectionName", collectionName),
		zap.String("collectionID", collectionID))
	return isolationID, collectionID
}

// getCollectionMetrics returns document and embedding counts, using fresh or cached metrics
func getCollectionMetrics(w *headerResponseWriter, isolationID, collectionID string) (int64, int64) {
	var documentCount, embeddingCount int64
	var useFreshDbMetrics bool

	runtimeConfig := getRuntimeConfigFromContext(w.requestContext)
	useFreshDbMetrics = runtimeConfig != nil && runtimeConfig.ForceFreshDbMetrics
	genaiHeadersLogger.Debug("runtime config retrieved",
		zap.Bool("runtimeConfigNotNil", runtimeConfig != nil),
		zap.Bool("forceFreshDbMetrics", useFreshDbMetrics))

	if useFreshDbMetrics {
		genaiHeadersLogger.Debug("force fresh metrics enabled, getting database from requestContext")
		database := getDatabaseFromContext(w)
		if database != nil {
			genaiHeadersLogger.Debug("database retrieved from requestContext, calling GetFreshCollectionMetrics")
			manager := dbmetrics.NewManager(database)
			documentCount, embeddingCount = manager.GetFreshCollectionMetrics(
				w.requestContext, isolationID, collectionID)
			genaiHeadersLogger.Debug("fresh metrics retrieved",
				zap.Int64("documentCount", documentCount),
				zap.Int64("embeddingCount", embeddingCount))
		} else {
			genaiHeadersLogger.Debug("database not found in requestContext, falling back to cached metrics")
			documentCount = dbmetrics.GetCollectionDocumentCount(isolationID, collectionID)
			embeddingCount = dbmetrics.GetCollectionEmbeddingCount(isolationID, collectionID)
		}
	} else {
		genaiHeadersLogger.Debug("using cached metrics (normal behavior)")
		documentCount = dbmetrics.GetCollectionDocumentCount(isolationID, collectionID)
		embeddingCount = dbmetrics.GetCollectionEmbeddingCount(isolationID, collectionID)
	}
	return documentCount, embeddingCount
}

// getRuntimeConfigFromContext retrieves RuntimeConfig from context
func getRuntimeConfigFromContext(ctx context.Context) *config.RuntimeConfig {
	return config.GetRuntimeConfigFromContext(ctx)
}

// getDatabaseFromContext retrieves database connection from Gin context
func getDatabaseFromContext(w *headerResponseWriter) db.Database {
	if w.ginContext != nil {
		if dbInterface, exists := w.ginContext.Get(DBConnectionGeneric); exists {
			if database, ok := dbInterface.(db.Database); ok {
				return database
			}
		}
	}
	return nil
}
