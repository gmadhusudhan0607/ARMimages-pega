/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestModificationResponseWriter wraps gin.ResponseWriter to capture metrics and update Prometheus
type RequestModificationResponseWriter struct {
	gin.ResponseWriter
	logger          *zap.SugaredLogger
	ginContext      *gin.Context
	startTime       time.Time
	statusCode      int
	updateOnce      sync.Once
	metricsUpdated  bool
	responseBuffer  *bytes.Buffer
	bodyProcessed   bool
	extractedTokens *float64 // Store extracted completion tokens
	headersWritten  bool     // Track if headers have been written to underlying writer
	shouldBuffer    bool     // Whether to buffer the response body instead of writing through
}

// NewRequestModificationResponseWriter creates a new RequestModificationResponseWriter
func NewRequestModificationResponseWriter(w gin.ResponseWriter, c *gin.Context, logger *zap.SugaredLogger) *RequestModificationResponseWriter {
	logger.Debugf("NewRequestModificationResponseWriter: Creating for path %s", c.Request.URL.Path)
	return &RequestModificationResponseWriter{
		ResponseWriter:  w,
		ginContext:      c,
		logger:          logger,
		startTime:       time.Now(),
		statusCode:      200, // Default status code
		responseBuffer:  &bytes.Buffer{},
		bodyProcessed:   false,
		extractedTokens: nil,
		headersWritten:  false,
		shouldBuffer:    true, // Buffer body by default to allow truncation detection
	}
}

// Write implements gin.ResponseWriter interface
func (w *RequestModificationResponseWriter) Write(data []byte) (int, error) {
	w.logger.Debugf("Write: Called with %d bytes (shouldBuffer: %v, headersWritten: %v)", len(data), w.shouldBuffer, w.headersWritten)

	// Capture response body for processing
	if w.responseBuffer != nil {
		w.responseBuffer.Write(data)
	}

	// Process response body to extract token usage if not already processed
	if !w.bodyProcessed {
		w.processResponseBody()
	}

	// If buffering is enabled, don't write to underlying writer yet
	if w.shouldBuffer {
		w.logger.Debugf("Write: Buffering %d bytes (total buffered: %d) - data NOT sent to client yet", len(data), w.responseBuffer.Len())
		// Return success without writing to underlying writer
		return len(data), nil
	}

	// Write headers if not yet written
	if !w.headersWritten {
		w.logger.Debugf("Write: Writing headers with status %d before first data write", w.statusCode)
		w.ResponseWriter.WriteHeader(w.statusCode)
		w.headersWritten = true
	}

	// Write the data to underlying writer (this sends data to the client)
	w.logger.Debugf("Write: Passing %d bytes through to underlying writer (streaming chunk delivery)", len(data))
	n, err := w.ResponseWriter.Write(data)

	// Update metrics after writing (ensures we have processed the response)
	w.updateMetricsOnce()

	return n, err
}

// WriteString implements gin.ResponseWriter interface
func (w *RequestModificationResponseWriter) WriteString(s string) (int, error) {
	// Capture response body for processing
	if w.responseBuffer != nil {
		w.responseBuffer.WriteString(s)
	}

	// Process response body to extract token usage if not already processed
	if !w.bodyProcessed {
		w.processResponseBody()
	}

	// If buffering is enabled, don't write to underlying writer yet
	if w.shouldBuffer {
		w.logger.Debugf("WriteString: Buffering %d bytes (total buffered: %d)", len(s), w.responseBuffer.Len())
		// Return success without writing to underlying writer
		return len(s), nil
	}

	// Write headers if not yet written
	if !w.headersWritten {
		w.ResponseWriter.WriteHeader(w.statusCode)
		w.headersWritten = true
	}

	// Write the string to underlying writer
	n, err := w.ResponseWriter.WriteString(s)

	// Update metrics after writing
	w.updateMetricsOnce()

	return n, err
}

// WriteHeader implements gin.ResponseWriter interface
func (w *RequestModificationResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	// Disable buffering for error responses - they should be written through immediately
	if statusCode >= 400 {
		w.shouldBuffer = false
	}
	w.logger.Debugf("WriteHeader: Status code set to %d (buffering: %v, headersWritten: %v)", statusCode, w.shouldBuffer, w.headersWritten)
	// Only write to underlying writer if not buffering
	if !w.shouldBuffer && !w.headersWritten {
		w.ResponseWriter.WriteHeader(statusCode)
		w.headersWritten = true
	}
	// Don't update metrics here - wait for body to be written
}

// Status returns the HTTP status code
func (w *RequestModificationResponseWriter) Status() int {
	return w.statusCode
}

// Header returns the underlying writer's header map so that all callers (handlers,
// upstream-header copying, LLMMetricsMiddleware) operate on the same map that is
// eventually committed to the wire. Headers must never be split into a separate
// "buffered" map — only the body is buffered.
func (w *RequestModificationResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// updateMetricsOnce ensures metrics are updated exactly once using sync.Once
func (w *RequestModificationResponseWriter) updateMetricsOnce() {
	w.updateOnce.Do(func() {
		w.updateMetrics()
		w.metricsUpdated = true
	})
}

// updateMetrics updates the RequestMetrics and Prometheus metrics
func (w *RequestModificationResponseWriter) updateMetrics() {
	w.logger.Debugf("updateMetrics: Starting for path %s", w.ginContext.Request.URL.Path)

	// Process response body first to extract token usage before updating metrics
	if !w.bodyProcessed {
		w.processResponseBody()
	}

	// Calculate timing metrics
	timingMetrics := w.createTimingMetrics()

	// Extract request metrics from context
	requestMetrics := w.extractRequestMetrics(timingMetrics)

	// Override with extracted tokens if available
	if w.extractedTokens != nil {
		requestMetrics.TokenMetrics.Used = w.extractedTokens
		w.logger.Debugf("updateMetrics: Using extracted tokens from response: %f", *w.extractedTokens)
	}

	// Update Prometheus metrics
	w.updatePrometheusMetrics(requestMetrics)
}

// createTimingMetrics creates timing metrics for the current request
func (w *RequestModificationResponseWriter) createTimingMetrics() *TimingMetrics {
	endTime := time.Now()
	duration := endTime.Sub(w.startTime)

	return &TimingMetrics{
		StartTime: w.startTime,
		EndTime:   endTime,
		Duration:  duration,
	}
}

// extractRequestMetrics extracts or creates RequestMetrics from the context
func (w *RequestModificationResponseWriter) extractRequestMetrics(timingMetrics *TimingMetrics) *RequestMetrics {
	ctx := w.ginContext.Request.Context()
	metadata := ctx.Value(RequestMetadataContextKey{})

	if metadata == nil {
		w.logger.Debugf("updateMetrics: No RequestMetadata found in context for path %s", w.ginContext.Request.URL.Path)
		return w.createMinimalRequestMetrics(timingMetrics)
	}

	w.logger.Debugf("updateMetrics: Found RequestMetadata in context for path %s", w.ginContext.Request.URL.Path)

	// Try to extract full RequestMetrics
	if requestMetrics := w.tryExtractFullRequestMetrics(metadata, timingMetrics); requestMetrics != nil {
		return requestMetrics
	}

	// Try to extract TokenMetrics only
	if requestMetrics := w.tryExtractTokenMetrics(metadata, timingMetrics); requestMetrics != nil {
		return requestMetrics
	}

	// Fallback to minimal metrics
	return w.createMinimalRequestMetrics(timingMetrics)
}

// tryExtractFullRequestMetrics attempts to extract full RequestMetrics from metadata
func (w *RequestModificationResponseWriter) tryExtractFullRequestMetrics(metadata interface{}, timingMetrics *TimingMetrics) *RequestMetrics {
	metadataWithMetrics, ok := metadata.(interface{ GetRequestMetrics() *RequestMetrics })
	if !ok {
		return nil
	}

	w.logger.Debugf("updateMetrics: Metadata implements GetRequestMetrics interface")
	fullMetrics := metadataWithMetrics.GetRequestMetrics()
	if fullMetrics == nil {
		return nil
	}

	requestMetrics := &RequestMetrics{
		TimingMetrics: *timingMetrics,
		TokenMetrics:  fullMetrics.TokenMetrics, // Preserve token metrics including Maximum
	}
	w.logger.Debugf("updateMetrics: Successfully extracted RequestMetrics")
	return requestMetrics
}

// tryExtractTokenMetrics attempts to extract TokenMetrics from metadata
func (w *RequestModificationResponseWriter) tryExtractTokenMetrics(metadata interface{}, timingMetrics *TimingMetrics) *RequestMetrics {
	tokenMetricsGetter, ok := metadata.(interface{ GetTokenMetrics() *TokenMetrics })
	if !ok {
		return nil
	}

	w.logger.Debugf("updateMetrics: Metadata implements GetTokenMetrics interface")
	tokenMetrics := tokenMetricsGetter.GetTokenMetrics()
	if tokenMetrics == nil {
		return nil
	}

	requestMetrics := &RequestMetrics{
		TimingMetrics: *timingMetrics,
		TokenMetrics:  *tokenMetrics,
	}
	w.logger.Debugf("updateMetrics: Successfully extracted TokenMetrics")
	return requestMetrics
}

// createMinimalRequestMetrics creates a minimal RequestMetrics with timing only
func (w *RequestModificationResponseWriter) createMinimalRequestMetrics(timingMetrics *TimingMetrics) *RequestMetrics {
	w.logger.Debugf("updateMetrics: Creating minimal RequestMetrics with timing only")
	return &RequestMetrics{
		TimingMetrics: *timingMetrics,
	}
}

// updatePrometheusMetrics updates Prometheus metrics based on collected RequestMetrics
func (w *RequestModificationResponseWriter) updatePrometheusMetrics(requestMetrics *RequestMetrics) {
	w.logger.Debugf("updatePrometheusMetrics: Starting for path %s", w.ginContext.Request.URL.Path)

	// Check if metadata can be extracted from context
	if _, ok := ExtractMetadataFromContext(w.ginContext); !ok {
		w.logger.Debugf("updatePrometheusMetrics: Failed to extract metadata from context for path %s", w.ginContext.Request.URL.Path)
	} else {
		w.logger.Debugf("updatePrometheusMetrics: Successfully extracted metadata from context for path %s", w.ginContext.Request.URL.Path)
	}

	// Create labels for Prometheus metrics using the consolidated function
	labels := CreatePrometheusLabelsFromContext(w.ginContext, w.statusCode)

	w.logger.Debugf("updatePrometheusMetrics: Created labels - infrastructure=%s, provider=%s, originalModelName=%s, targetModelName=%s, targetModelID=%s, targetModelVersion=%s",
		labels["infrastructure"], labels["provider"], labels["originalModelName"], labels["targetModelName"], labels["targetModelID"], labels["targetModelVersion"])

	// Update timing metrics in Prometheus
	UpdateTimingMetrics(requestMetrics, labels)

	// Update token metrics in Prometheus
	UpdateTokenMetrics(requestMetrics, labels)

	w.logger.Debugf("updatePrometheusMetrics: Updated Prometheus metrics for request: duration=%v, status=%d",
		requestMetrics.TimingMetrics.Duration, w.statusCode)
}

// IsMetricsUpdated returns whether metrics have been updated
func (w *RequestModificationResponseWriter) IsMetricsUpdated() bool {
	return w.metricsUpdated
}

// processResponseBody processes the captured response body to extract token usage
func (w *RequestModificationResponseWriter) processResponseBody() {
	if w.bodyProcessed || w.responseBuffer == nil || w.responseBuffer.Len() == 0 {
		return
	}

	// Check if this is a streaming request - skip response body processing for streams
	ctx := w.ginContext.Request.Context()
	metadataInterface := ctx.Value(RequestMetadataContextKey{})
	if metadataInterface != nil {
		if metadataWithMetrics, ok := metadataInterface.(interface{ GetRequestMetrics() *RequestMetrics }); ok {
			if metrics := metadataWithMetrics.GetRequestMetrics(); metrics != nil && metrics.IsStreaming {
				w.logger.Debug("processResponseBody: Skipping response body processing for streaming request")
				w.bodyProcessed = true
				return
			}
		}
	}

	w.logger.Debugf("processResponseBody: Processing response body for path %s", w.ginContext.Request.URL.Path)

	// Skip JSON parsing for non-JSON response bodies (e.g., SDP from realtime/calls).
	if ct := w.Header().Get("Content-Type"); ct != "" && !strings.Contains(ct, "application/json") {
		w.logger.Debugf("processResponseBody: Skipping non-JSON response (Content-Type: %s)", ct)
		w.bodyProcessed = true
		return
	}

	// Try to parse the response as JSON
	responseData, err := w.parseResponseJSON()
	if err != nil {
		w.logger.Debugf("processResponseBody: Failed to parse response as JSON: %v", err)
		w.bodyProcessed = true
		return
	}

	// Extract and store completion tokens
	w.extractAndStoreCompletionTokens(responseData)
	w.bodyProcessed = true
}

// parseResponseJSON parses the response buffer as JSON
func (w *RequestModificationResponseWriter) parseResponseJSON() (map[string]interface{}, error) {
	var responseData map[string]interface{}
	err := json.Unmarshal(w.responseBuffer.Bytes(), &responseData)
	return responseData, err
}

// extractAndStoreCompletionTokens extracts completion_tokens from response data and stores them
func (w *RequestModificationResponseWriter) extractAndStoreCompletionTokens(responseData map[string]interface{}) {
	usage, ok := responseData["usage"].(map[string]interface{})
	if !ok {
		w.logger.Debugf("processResponseBody: usage object not found in response")
		return
	}

	completionTokens, ok := usage["completion_tokens"].(float64)
	if !ok {
		w.logger.Debugf("processResponseBody: completion_tokens not found or not a number in usage")
		return
	}

	w.logger.Debugf("processResponseBody: Found completion_tokens: %f", completionTokens)

	// Store the extracted tokens directly in the writer
	w.extractedTokens = &completionTokens
	w.logger.Debugf("processResponseBody: Stored completion_tokens in writer: %f", completionTokens)

	// Also try to update the RequestMetadata in context (for compatibility)
	w.updateTokenMetricsInContext(completionTokens)
}

// updateTokenMetricsInContext updates the token metrics in the request context
func (w *RequestModificationResponseWriter) updateTokenMetricsInContext(completionTokens float64) {
	ctx := w.ginContext.Request.Context()
	metadata := ctx.Value(RequestMetadataContextKey{})
	if metadata == nil {
		return
	}

	requestMetadata, ok := metadata.(interface {
		GetTokenMetrics() *TokenMetrics
	})
	if !ok {
		return
	}

	tokenMetrics := requestMetadata.GetTokenMetrics()
	if tokenMetrics == nil {
		return
	}

	usedFloat := completionTokens
	tokenMetrics.Used = &usedFloat
	w.logger.Debugf("processResponseBody: Updated TokenMetrics.Used to %f", usedFloat)
}

// GetDuration returns the current request duration
func (w *RequestModificationResponseWriter) GetDuration() time.Duration {
	return time.Since(w.startTime)
}

// GetResponseBody returns the captured response body
func (w *RequestModificationResponseWriter) GetResponseBody() []byte {
	if w.responseBuffer == nil {
		return nil
	}
	return w.responseBuffer.Bytes()
}

// GetMetricsWriter returns the underlying MetricsResponseWriter if it exists
// This allows LLMMetricsMiddleware to access the MetricsResponseWriter even when wrapped
func (w *RequestModificationResponseWriter) GetMetricsWriter() *middleware.MetricsResponseWriter {
	if mrw, ok := w.ResponseWriter.(*middleware.MetricsResponseWriter); ok {
		return mrw
	}
	return nil
}

// FlushBufferedResponse writes the buffered response to the client
// This is called when no retry is needed and the original response should be sent
func (w *RequestModificationResponseWriter) FlushBufferedResponse() error {
	if !w.shouldBuffer {
		w.logger.Debug("FlushBufferedResponse: Not buffering, nothing to flush")
		return nil
	}

	w.logger.Debugf("FlushBufferedResponse: Flushing %d bytes to client (status: %d)",
		w.responseBuffer.Len(), w.statusCode)

	// Disable buffering
	w.shouldBuffer = false

	// Force the underlying MetricsResponseWriter to write through if it exists
	if metricsWriter := w.GetMetricsWriter(); metricsWriter != nil {
		metricsWriter.ForceWriteThrough()
		w.logger.Debug("FlushBufferedResponse: Forced MetricsResponseWriter write-through")
	}

	// Write headers if not yet written
	if !w.headersWritten {
		w.ResponseWriter.WriteHeader(w.statusCode)
		w.headersWritten = true
	}

	// Write buffered response body
	if w.responseBuffer != nil && w.responseBuffer.Len() > 0 {
		if _, err := w.ResponseWriter.Write(w.responseBuffer.Bytes()); err != nil {
			w.logger.Errorf("FlushBufferedResponse: Failed to write buffered response: %v", err)
			return err
		}
	}

	// Release the buffer to free memory — it is no longer needed after flushing.
	w.releaseBuffer()

	// Update metrics after writing
	w.updateMetricsOnce()

	return nil
}

// ShouldBuffer returns whether response buffering is currently enabled.
func (w *RequestModificationResponseWriter) ShouldBuffer() bool {
	return w.shouldBuffer
}

// DisableBuffering disables response buffering to allow streaming chunks to flow through immediately.
// This should be called for streaming requests where chunks must be delivered incrementally.
func (w *RequestModificationResponseWriter) DisableBuffering() {
	w.logger.Debug("DisableBuffering: Disabling buffering for streaming request - chunks will flow through immediately")
	w.shouldBuffer = false
}

// PrepareForRetry disables buffering to allow direct writing of retry response
func (w *RequestModificationResponseWriter) PrepareForRetry() {
	w.logger.Debug("PrepareForRetry: Disabling buffering for retry response")
	w.shouldBuffer = false
	// Clear the body buffer as we won't use the original response body
	w.responseBuffer.Reset()
	// Clear all currently set headers from the underlying writer so the retry
	// response can set its own clean set of headers
	underlyingHeaders := w.ResponseWriter.Header()
	for key := range underlyingHeaders {
		underlyingHeaders.Del(key)
	}
}

// releaseBuffer nils the response buffer to free memory.
// After this call, any further Write/WriteString calls are safe (they check for nil).
func (w *RequestModificationResponseWriter) releaseBuffer() {
	w.responseBuffer = nil
}
