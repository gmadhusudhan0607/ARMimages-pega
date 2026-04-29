/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

// Package middleware provides HTTP middleware for request modification.
// The main middleware is RequestModificationMiddleware which handles both
// request metadata injection and response metrics collection.
package middleware

import (
	"context"
	"regexp"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
	"github.com/gin-gonic/gin"
)

// Context key types for storing data in request context
type contextKey string

const (
	ProcessorContextKey    contextKey = "processor"
	RetryAttemptContextKey contextKey = "retry_attempt"
	OriginalRequestBodyKey contextKey = "original_request_body"
)

// isBuddyRoute checks if the path matches the buddy pattern /v1/:isolationId/buddies/:buddyName
func isBuddyRoute(path string) bool {
	return regexp.MustCompile(`^/v1/[^/]+/buddies/[^/]+(?:/.*)?$`).MatchString(path)
}

// RequestModificationMiddleware is the main entry point for request processing
func RequestModificationMiddleware(serviceCtx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Merge service context with request context to preserve configuration values
		ctx := cntx.MergeServiceContext(serviceCtx, c.Request.Context())
		c.Request = c.Request.WithContext(ctx)

		logger := cntx.LoggerFromContext(ctx).Sugar()
		logger.Debugf("RequestModificationMiddleware: Starting for %s %s", c.Request.Method, c.Request.URL.Path)

		// Skip processing if OutputTokensStrategyDisabled strategy is set
		if config.IsOutputTokensStrategyDisabled(c.Request.Context()) {
			logger.Debug("RequestModificationMiddleware: Skipping processing due to OutputTokensStrategyDisabled strategy")
			c.Next()
			return
		}

		// Skip processing for buddy routes
		if isBuddyRoute(c.Request.URL.Path) {
			logger.Debugf("RequestModificationMiddleware: Skipping buddy route: %s", c.Request.URL.Path)
			c.Next()
			return
		}

		// Inject request metadata
		if err := injectRequestMetadata(c); err != nil {
			logger.Errorf("RequestModificationMiddleware: Metadata injection failed: %v", err)
			// Continue processing (graceful degradation)
		}

		// Setup response writer
		if err := setupResponseWriter(c); err != nil {
			logger.Errorf("RequestModificationMiddleware: Response writer setup failed: %v", err)
			return // Response already sent by setupResponseWriter
		}

		// Modify request if needed
		if err := modifyRequest(c); err != nil {
			logger.Errorf("RequestModificationMiddleware: Request modification failed: %v", err)
			// Continue processing (graceful degradation)
		}

		// For streaming requests, disable response buffering so that chunks flow through
		// to the client incrementally. Buffering is only needed for non-streaming requests
		// to support truncation detection and retry logic.
		disableBufferingForStreaming(c)

		// Process the request
		c.Next()

		// Handle response and retry logic
		handleResponse(c, logger)

		logger.Debugf("RequestModificationMiddleware: Completed for %s %s", c.Request.Method, c.Request.URL.Path)
	}
}

// disableBufferingForStreaming disables response buffering for streaming requests
// so that SSE chunks flow through to the client incrementally.
func disableBufferingForStreaming(c *gin.Context) {
	if !isStreamingRequest(c) {
		return
	}
	logger := cntx.LoggerFromContext(c.Request.Context()).Sugar()
	logger.Debug("RequestModificationMiddleware: Streaming request detected - disabling response buffering")
	if customWriter := findRequestModificationResponseWriter(c.Writer); customWriter != nil {
		customWriter.DisableBuffering()
	}
}

// SetRequestMetadataInContext updates RequestMetadata in the context
func SetRequestMetadataInContext(c *gin.Context, md interface{}) {
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	c.Request = c.Request.WithContext(ctx)
}
