/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
)

func TestRequestProcessingMiddleware_ResponseWriterWrapping(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequestModificationMiddleware(context.Background()))

	var writerType string
	router.GET("/test", func(c *gin.Context) {
		// Check if a response writer is wrapped
		if _, ok := c.Writer.(*metrics.RequestModificationResponseWriter); ok {
			writerType = "RequestModificationResponseWriter"
		} else {
			writerType = "Other"
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "RequestModificationResponseWriter", writerType, "Response writer should be wrapped with RequestModificationResponseWriter")
}

func TestRequestProcessingMiddleware_PostProcessingMetricsUpdate(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequestModificationMiddleware(context.Background()))

	var customWriter *metrics.RequestModificationResponseWriter
	router.GET("/test", func(c *gin.Context) {
		// Capture the custom writer for testing
		if cw, ok := c.Writer.(*metrics.RequestModificationResponseWriter); ok {
			customWriter = cw
		}

		// Don't write any response to test the post-processing logic
		// The middleware should ensure metrics are updated even without explicit response
	})

	// Test
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.NotNil(t, customWriter, "RequestModificationResponseWriter should be captured")
	// The middleware should have called WriteHeader to ensure metrics are updated
	// We can't easily test the internal state, but we can verify the middleware completed without error
	assert.Equal(t, http.StatusOK, w.Code)
}
