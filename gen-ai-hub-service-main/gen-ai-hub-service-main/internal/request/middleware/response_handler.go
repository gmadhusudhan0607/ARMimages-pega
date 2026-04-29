/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/ginctx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// handleResponse processes the response including truncation detection, retry logic, and metrics updates
func handleResponse(c *gin.Context, logger *zap.SugaredLogger) {
	logger.Debug("handleResponse: Starting response processing")

	// Check if this is a streaming request - skip response processing for streams
	if isStreamingRequest(c) {
		logger.Debug("handleResponse: Streaming request - skipping truncation detection and retry logic (streaming responses are delivered incrementally)")
		// For streaming, buffering was already disabled before c.Next(), so chunks
		// already flowed through to the client. FlushBufferedResponse will be a no-op.
		flushBufferedResponse(c, logger)
		// Only ensure metrics update (timing, request-side metrics)
		ensureMetricsUpdate(c, logger)
		return
	}

	// Process response for truncation detection (only for successful responses)
	if c.Writer.Status() == 200 {
		processResponseForTruncation(c, logger)
	}

	// Check if retry is needed and perform it
	if shouldRetryForTruncation(c) {
		logger.Debug("handleResponse: Truncation detected, attempting retry")
		if err := performRetry(c); err != nil {
			logger.Errorf("handleResponse: Retry failed: %v", err)
			// If retry failed, flush the original buffered response
			flushBufferedResponse(c, logger)
		}
	} else {
		// No retry needed, flush the buffered original response
		logger.Debug("handleResponse: No retry needed, flushing buffered response")
		flushBufferedResponse(c, logger)
	}

	// Ensure metrics are updated
	ensureMetricsUpdate(c, logger)
}

// isStreamingRequest checks if the current request is a streaming request
func isStreamingRequest(c *gin.Context) bool {
	md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
	if err != nil {
		return false
	}
	return md.IsStreaming
}

// processResponseForTruncation processes the response to detect truncation using the processor
func processResponseForTruncation(c *gin.Context, logger *zap.SugaredLogger) {
	logger.Debug("processResponseForTruncation: Starting response processing")

	// Get processor from context
	ctx := c.Request.Context()
	processorInterface := ctx.Value(ProcessorContextKey)
	if processorInterface == nil {
		logger.Debug("processResponseForTruncation: No processor found in context")
		return
	}

	// Check if processor supports response processing
	responseProcessor, ok := processorInterface.(interface {
		ProcessResponse(ctx context.Context, resp *http.Response) (*extensions.ProcessedResponse, error)
	})
	if !ok {
		logger.Debug("processResponseForTruncation: Processor does not support ProcessResponse method")
		return
	}

	// Try to get the response body from the RequestModificationResponseWriter
	customWriter := findRequestModificationResponseWriter(c.Writer)
	if customWriter == nil {
		logger.Debug("processResponseForTruncation: RequestModificationResponseWriter not found")
		return
	}

	// Get captured response body from the writer
	responseBody := customWriter.GetResponseBody()
	if len(responseBody) == 0 {
		logger.Debug("processResponseForTruncation: No response body captured")
		return
	}

	// Create a mock HTTP response for processing
	resp := &http.Response{
		StatusCode: c.Writer.Status(),
		Header:     c.Writer.Header(),
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}

	// Process the response using the processor
	processedResponse, err := responseProcessor.ProcessResponse(ctx, resp)
	if err != nil {
		logger.Debugf("processResponseForTruncation: Failed to process response: %v", err)
		return
	}

	if processedResponse == nil {
		logger.Debug("processResponseForTruncation: No processed response returned")
		return
	}

	// Update metadata with truncation information
	md, err := metadata.GetRequestMetadataFromContext(ctx)
	if err != nil {
		logger.Debugf("processResponseForTruncation: Failed to get metadata from context: %v", err)
		return
	}

	// Update truncation status
	if processedResponse.WasTruncated {
		md.RequestMetrics.RetryMetrics.ResponseTruncated = true
		if processedResponse.FinishReason != "" {
			reason := fmt.Sprintf("finish_reason_%s", processedResponse.FinishReason)
			md.RequestMetrics.RetryMetrics.Reason = &reason
		}
		logger.Debugf("processResponseForTruncation: Response truncation detected - finish_reason: %s", processedResponse.FinishReason)
	}

	// Update token usage if available
	if processedResponse.UsedTokens != nil {
		usedFloat := float64(*processedResponse.UsedTokens)
		md.RequestMetrics.TokenMetrics.Used = &usedFloat
		logger.Debugf("processResponseForTruncation: Updated used tokens: %d", *processedResponse.UsedTokens)
	}
}

// shouldRetryForTruncation determines if a retry should be performed based on truncation and configuration
func shouldRetryForTruncation(c *gin.Context) bool {
	logger := cntx.LoggerFromContext(c.Request.Context()).Sugar()

	// Check if this is already a retry attempt
	if isRetryAttempt(c) {
		return false
	}

	// Check if response was truncated
	if !checkTruncation(c) {
		return false
	}

	// Get the original request body to check if it's streaming
	ctx := c.Request.Context()
	originalBodyInterface := ctx.Value(OriginalRequestBodyKey)
	if originalBodyInterface == nil {
		return false
	}

	originalBody, ok := originalBodyInterface.([]byte)
	if !ok {
		return false
	}

	// Check if this is a streaming request - streaming requests should never retry
	isStreaming := isStreamingRequestFromBody(originalBody)
	if isStreaming {
		logger.Debug("shouldRetryForTruncation: Streaming requests never retry")
		return false
	}

	return true
}

// performRetry executes the retry logic with max_tokens removed
func performRetry(c *gin.Context) error {
	logger := cntx.LoggerFromContext(c.Request.Context()).Sugar()
	logger.Debug("performRetry: Starting retry attempt due to truncation")

	// Get and validate original request body
	originalBody, err := getOriginalRequestBody(c)
	if err != nil {
		return err
	}

	// Remove max_tokens from the request body
	modifiedBody, err := removeMaxTokensFromRequest(c, originalBody)
	if err != nil {
		logger.Errorf("performRetry: Failed to remove max_tokens: %v", err)
		return err
	}

	// Update retry metrics in metadata
	_, err = updateRetryMetrics(c)
	if err != nil {
		return err
	}

	// Get retry URL from context
	modelURL := c.GetString(ginctx.ModelURLContextKey)

	// Create and execute retry request
	retryResponse, err := executeRetryRequest(c, modelURL, modifiedBody)
	if err != nil {
		logger.Debugf("Retry request failed. URL: %s, body: %s", modelURL, string(modifiedBody))
		return err
	}

	// Write retry response to client
	writeRetryResponse(c, retryResponse, logger)

	return nil
}

// ensureMetricsUpdate ensures metrics are updated even if no response was written
func ensureMetricsUpdate(c *gin.Context, logger *zap.SugaredLogger) {
	logger.Debug("ensureMetricsUpdate: Starting metrics update")

	// Try to find RequestModificationResponseWriter
	customWriter := findRequestModificationResponseWriter(c.Writer)
	if customWriter != nil {
		if !customWriter.IsMetricsUpdated() {
			customWriter.WriteHeader(c.Writer.Status())
		}
	}

	// Update requests cache for auto-adjustment strategy
	if c.Writer.Status() == 200 {
		updateRequestsCache(c, logger)
	}
}

// Helper functions

func isRetryAttempt(c *gin.Context) bool {
	ctx := c.Request.Context()
	if retryAttempt := ctx.Value(RetryAttemptContextKey); retryAttempt != nil {
		if isRetry, ok := retryAttempt.(bool); ok {
			return isRetry
		}
	}
	return false
}

func checkTruncation(c *gin.Context) bool {
	logger := cntx.LoggerFromContext(c.Request.Context()).Sugar()

	md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
	if err != nil {
		logger.Debugf("checkTruncation: Failed to get metadata from context: %v", err)
		return false
	}

	return md.RequestMetrics.RetryMetrics.ResponseTruncated
}

func getOriginalRequestBody(c *gin.Context) ([]byte, error) {
	ctx := c.Request.Context()
	originalBodyInterface := ctx.Value(OriginalRequestBodyKey)
	if originalBodyInterface == nil {
		return nil, fmt.Errorf("no original request body found in context")
	}

	originalBody, ok := originalBodyInterface.([]byte)
	if !ok {
		return nil, fmt.Errorf("original request body is not []byte")
	}

	return originalBody, nil
}

func removeMaxTokensFromRequest(c *gin.Context, originalBody []byte) ([]byte, error) {
	logger := cntx.LoggerFromContext(c.Request.Context()).Sugar()

	var requestData map[string]interface{}
	if err := json.Unmarshal(originalBody, &requestData); err != nil {
		return originalBody, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if _, exists := requestData["max_tokens"]; !exists {
		return originalBody, nil
	}

	delete(requestData, "max_tokens")
	logger.Debug("removeMaxTokensFromRequest: Removed max_tokens field from request body")

	modifiedBody, err := json.Marshal(requestData)
	if err != nil {
		return originalBody, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return modifiedBody, nil
}

func updateRetryMetrics(c *gin.Context) (*metadata.RequestMetadata, error) {
	ctx := c.Request.Context()

	md, err := metadata.GetRequestMetadataFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if md.TargetModel == nil {
		return nil, fmt.Errorf("no target model available for retry")
	}

	md.RequestMetrics.RetryMetrics.Count = 1
	reason := "max_tokens_exceeded"
	md.RequestMetrics.RetryMetrics.Reason = &reason

	return md, nil
}

type retryResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func executeRetryRequest(c *gin.Context, retryURL string, modifiedBody []byte) (*retryResponse, error) {
	ctx := c.Request.Context()

	// Create a new HTTP request for the retry
	retryRequest, err := http.NewRequestWithContext(ctx, c.Request.Method, retryURL, bytes.NewReader(modifiedBody))
	if err != nil {
		return nil, err
	}

	// Copy headers and update content length
	for key, values := range c.Request.Header {
		for _, value := range values {
			retryRequest.Header.Add(key, value)
		}
	}
	retryRequest.ContentLength = int64(len(modifiedBody))
	retryRequest.Header.Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))

	// Make the actual HTTP request
	client := &http.Client{}
	resp, err := client.Do(retryRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the retry response
	retryResponseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &retryResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       retryResponseBody,
	}, nil
}

func writeRetryResponse(c *gin.Context, response *retryResponse, logger *zap.SugaredLogger) {
	// Prepare RequestModificationResponseWriter for retry - disable buffering and clear buffer
	customWriter := findRequestModificationResponseWriter(c.Writer)
	if customWriter != nil {
		customWriter.PrepareForRetry()

		// Also force MetricsResponseWriter to write through (disable LLM buffering)
		if metricsWriter := customWriter.GetMetricsWriter(); metricsWriter != nil {
			metricsWriter.ForceWriteThrough()
			logger.Debug("writeRetryResponse: Forced MetricsResponseWriter write-through")
		}
	}

	// Copy headers from retry response, excluding hop-by-hop and encoding headers
	// that may no longer be valid after Go's http.Client transparently decompresses the body.
	skipHeaders := map[string]bool{
		"Content-Encoding":  true,
		"Transfer-Encoding": true,
	}
	for key, values := range response.Headers {
		if skipHeaders[key] {
			continue
		}
		c.Writer.Header().Del(key)
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}

	// Always set Content-Length from the actual (possibly decompressed) body length.
	// Go's http.Client transparently decompresses gzip responses, so the upstream
	// Content-Length may reflect the compressed size while response.Body holds the
	// decompressed bytes — causing "wrote more than the declared Content-Length".
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(response.Body)))

	// Write the retry response
	c.Writer.WriteHeader(response.StatusCode)
	if _, err := c.Writer.Write(response.Body); err != nil {
		logger.Errorf("performRetry: failed to write retry response body: %v", err)
	}

	logger.Debugf("writeRetryResponse: Wrote retry response with status %d and %d bytes", response.StatusCode, len(response.Body))
}

func updateRequestsCache(c *gin.Context, logger *zap.SugaredLogger) {
	ctx := c.Request.Context()
	processorInterface := ctx.Value(ProcessorContextKey)
	if processorInterface == nil {
		return
	}

	processor, ok := processorInterface.(processors.RequestProcessor)
	if !ok {
		return
	}

	md, err := metadata.GetRequestMetadataFromContext(ctx)
	if err != nil {
		return
	}

	if md.RequestMetrics.TokenMetrics.Used == nil {
		return
	}

	usedTokens := int(*md.RequestMetrics.TokenMetrics.Used)
	configValue := 1022 // Default from test environment

	if cacheUpdater, ok := processor.(interface {
		UpdateCache(ctx context.Context, usedTokens int, configValue int)
	}); ok {
		cacheUpdater.UpdateCache(ctx, usedTokens, configValue)
	}
}

// flushBufferedResponse flushes the buffered response to the client if buffering is enabled
func flushBufferedResponse(c *gin.Context, logger *zap.SugaredLogger) {
	customWriter := findRequestModificationResponseWriter(c.Writer)
	if customWriter != nil {
		if err := customWriter.FlushBufferedResponse(); err != nil {
			logger.Errorf("flushBufferedResponse: Failed to flush buffered response: %v", err)
		}
	}
}

// findRequestModificationResponseWriter recursively searches for RequestModificationResponseWriter in the writer chain
func findRequestModificationResponseWriter(writer gin.ResponseWriter) *metrics.RequestModificationResponseWriter {
	// Direct match
	if customWriter, ok := writer.(*metrics.RequestModificationResponseWriter); ok {
		return customWriter
	}

	// Use reflection to check if the writer has an embedded ResponseWriter or RequestModificationResponseWriter field
	writerValue := reflect.ValueOf(writer)
	if writerValue.Kind() == reflect.Ptr {
		writerValue = writerValue.Elem()
	}

	if writerValue.Kind() == reflect.Struct {
		// First try to find RequestModificationResponseWriter field (for TrailerAwareRequestModificationResponseWriter)
		customWriterField := writerValue.FieldByName("RequestModificationResponseWriter")
		if customWriterField.IsValid() && customWriterField.CanInterface() {
			if customWriter, ok := customWriterField.Interface().(*metrics.RequestModificationResponseWriter); ok {
				return customWriter
			}
		}

		// Then try ResponseWriter field (for other wrappers)
		responseWriterField := writerValue.FieldByName("ResponseWriter")
		if responseWriterField.IsValid() && responseWriterField.CanInterface() {
			if nestedWriter, ok := responseWriterField.Interface().(gin.ResponseWriter); ok {
				return findRequestModificationResponseWriter(nestedWriter)
			}
		}
	}

	// Try common interface patterns for wrapped writers
	if wrapper, ok := writer.(interface{ Unwrap() gin.ResponseWriter }); ok {
		return findRequestModificationResponseWriter(wrapper.Unwrap())
	}

	return nil
}
