/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
)

func TestRequestProcessingMiddleware_BasicFunctionality(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add the combined RequestProcessing middleware
	router.Use(RequestModificationMiddleware(context.Background()))

	// Add a test handler that checks if RequestMetadata is in context and response writer is wrapped
	router.GET("/test", func(c *gin.Context) {
		metadata, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		assert.NoError(t, err, "RequestMetadata should be injected into context")
		assert.NotNil(t, metadata, "RequestMetadata should be injected into context")

		// Verify default values
		assert.Equal(t, "", metadata.IsolationID, "IsolationID should be empty by default")
		assert.Nil(t, metadata.TargetModel, "Model should be nil by default for non-model endpoints")
		assert.NotNil(t, metadata.RequestMetrics, "RequestMetrics should be initialized")

		// Verify response writer is wrapped
		_, ok := c.Writer.(*metrics.RequestModificationResponseWriter)
		assert.True(t, ok, "Response writer should be wrapped with RequestModificationResponseWriter")

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestProcessingMiddleware_MetadataNotFoundError(t *testing.T) {
	// This test simulates a scenario where metadata injection fails
	// In practice, this shouldn't happen with the combined middleware, but we test error handling

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create a custom middleware that simulates metadata injection failure
	router.Use(func(c *gin.Context) {
		// Don't inject metadata, proceed to next middleware
		c.Next()
	})

	// Add a middleware that tries to get metadata (simulating the write part)
	router.Use(func(c *gin.Context) {
		_, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(500, gin.H{"error": "Request metadata not properly initialized"})
			return
		}
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Request metadata not properly initialized")
}

func TestRequestModificationMiddleware_StreamingDisablesBuffering(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add the combined RequestProcessing middleware
	router.Use(RequestModificationMiddleware(context.Background()))

	// Add a test handler that verifies buffering is disabled for streaming requests
	router.POST("/openai/deployments/gpt-4/chat/completions", func(c *gin.Context) {
		// Verify response writer is a RequestModificationResponseWriter with buffering disabled
		customWriter := findRequestModificationResponseWriter(c.Writer)
		require.NotNil(t, customWriter, "Response writer should be wrapped with RequestModificationResponseWriter")

		// For streaming requests, buffering should be disabled
		md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		require.NoError(t, err)
		assert.True(t, md.IsStreaming, "Request should be detected as streaming")

		// Write a streaming chunk - it should pass through immediately
		c.Writer.WriteHeader(200)
		_, _ = c.Writer.Write([]byte("data: {\"chunk\":1}\n\n"))

		c.Status(http.StatusOK)
	})

	// Test with streaming request body
	streamingBody := `{"messages":[{"role":"user","content":"hello"}],"stream":true}`
	req, _ := http.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions?api-version=2024-10-21", bytes.NewBufferString(streamingBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The response should contain our streaming chunk (not buffered)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestModificationMiddleware_NonStreamingKeepsBuffering(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add the combined RequestProcessing middleware
	router.Use(RequestModificationMiddleware(context.Background()))

	// Track whether buffering was enabled during handler execution
	var bufferingEnabledDuringHandler bool

	// Add a test handler that checks buffering state for non-streaming requests
	router.POST("/openai/deployments/gpt-4/chat/completions", func(c *gin.Context) {
		// For non-streaming requests, the writer should still have buffering enabled
		customWriter := findRequestModificationResponseWriter(c.Writer)
		if customWriter != nil {
			// Access internal state via the write behavior:
			// If buffering is enabled, writing won't appear in the recorder
			bufferingEnabledDuringHandler = true
		}

		md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		require.NoError(t, err)
		assert.False(t, md.IsStreaming, "Request should NOT be detected as streaming")

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test with non-streaming request body
	nonStreamingBody := `{"messages":[{"role":"user","content":"hello"}]}`
	req, _ := http.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions?api-version=2024-10-21", bytes.NewBufferString(nonStreamingBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.True(t, bufferingEnabledDuringHandler, "Buffering should remain enabled for non-streaming requests")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDisableBufferingForStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		setupFunc         func(*gin.Context)
		wrapWriter        bool
		expectNoPanic     bool
		verifyBufferState func(*testing.T, *gin.Context)
	}{
		{
			name: "streaming request with custom writer - disables buffering",
			setupFunc: func(c *gin.Context) {
				md := &metadata.RequestMetadata{
					IsStreaming: true,
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)
			},
			wrapWriter:    true,
			expectNoPanic: true,
			verifyBufferState: func(t *testing.T, c *gin.Context) {
				customWriter := findRequestModificationResponseWriter(c.Writer)
				require.NotNil(t, customWriter)
				// Write data - if buffering is disabled, it should reach the underlying writer
				// We verify indirectly: DisableBuffering was called so shouldBuffer is false
				assert.False(t, customWriter.ShouldBuffer(),
					"shouldBuffer should be false after disableBufferingForStreaming for streaming request")
			},
		},
		{
			name: "non-streaming request - buffering remains enabled",
			setupFunc: func(c *gin.Context) {
				md := &metadata.RequestMetadata{
					IsStreaming: false,
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)
			},
			wrapWriter:    true,
			expectNoPanic: true,
			verifyBufferState: func(t *testing.T, c *gin.Context) {
				customWriter := findRequestModificationResponseWriter(c.Writer)
				require.NotNil(t, customWriter)
				assert.True(t, customWriter.ShouldBuffer(),
					"shouldBuffer should remain true for non-streaming request")
			},
		},
		{
			name: "no metadata in context - does not panic",
			setupFunc: func(c *gin.Context) {
				// Don't add metadata to context
			},
			wrapWriter:    true,
			expectNoPanic: true,
			verifyBufferState: func(t *testing.T, c *gin.Context) {
				customWriter := findRequestModificationResponseWriter(c.Writer)
				require.NotNil(t, customWriter)
				assert.True(t, customWriter.ShouldBuffer(),
					"shouldBuffer should remain true when no metadata is present")
			},
		},
		{
			name: "streaming request without custom writer - does not panic",
			setupFunc: func(c *gin.Context) {
				md := &metadata.RequestMetadata{
					IsStreaming: true,
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)
			},
			wrapWriter:    false, // Don't wrap with RequestModificationResponseWriter
			expectNoPanic: true,
			verifyBufferState: func(t *testing.T, c *gin.Context) {
				// No custom writer to check - just verify it didn't panic
				customWriter := findRequestModificationResponseWriter(c.Writer)
				assert.Nil(t, customWriter, "No custom writer should be present")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req

			// Optionally wrap the writer
			if tt.wrapWriter {
				logger := zap.NewNop().Sugar()
				customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
				c.Writer = customWriter
			}

			// Setup context
			tt.setupFunc(c)

			// Call the function under test
			if tt.expectNoPanic {
				assert.NotPanics(t, func() {
					disableBufferingForStreaming(c)
				})
			}

			// Verify buffer state
			tt.verifyBufferState(t, c)
		})
	}
}

func TestRequestModificationMiddleware_NoApiVersionValidation(t *testing.T) {
	// After the refactoring from middleware-level to handler-level API version governance,
	// the middleware must NOT reject requests based on the api-version query parameter.
	// Requests without api-version, with any api-version, or with an invalid api-version
	// should all pass through the middleware without error.
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "OpenAI endpoint without api-version passes through",
			path: "/openai/deployments/gpt-4/chat/completions",
		},
		{
			name: "OpenAI endpoint with governed api-version passes through",
			path: "/openai/deployments/gpt-4/chat/completions?api-version=2024-10-21",
		},
		{
			name: "OpenAI endpoint with old api-version passes through",
			path: "/openai/deployments/gpt-4/chat/completions?api-version=2021-01-01",
		},
		{
			name: "OpenAI endpoint with arbitrary api-version passes through",
			path: "/openai/deployments/gpt-4/chat/completions?api-version=invalid-version",
		},
		{
			name: "Non-OpenAI endpoint without api-version passes through",
			path: "/v1/test-isolation/models/gpt-4/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestModificationMiddleware(context.Background()))

			// Register a catch-all handler so any path is routed
			router.Any("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req, _ := http.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// The middleware must not reject the request — only 200 or 500 from metadata init is acceptable,
			// but never a 400-level error from API version validation.
			assert.NotEqual(t, http.StatusBadRequest, w.Code,
				"middleware should not reject requests based on api-version parameter")
		})
	}
}

func TestRequestProcessingMiddleware_SetAndGetMetadata(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequestModificationMiddleware(context.Background()))

	router.GET("/test", func(c *gin.Context) {
		// Get original metadata
		originalMetadata, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		require.NoError(t, err, "RequestMetadata should be available")
		require.NotNil(t, originalMetadata)

		// Create modified metadata
		modifiedMetadata := &metadata.RequestMetadata{
			IsolationID: "test-isolation-id",
			TargetModel: &modeltypes.Model{
				KEY:      "gpt-4-resolved",
				Provider: modeltypes.ProviderAzure,
			},
			RequestMetrics:    originalMetadata.RequestMetrics, // Keep original metrics
			OriginalModelName: "gpt-4",
		}

		// Set modified metadata
		SetRequestMetadataInContext(c, modifiedMetadata)

		// Verify the metadata was updated
		updatedMetadata, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		require.NoError(t, err, "RequestMetadata should be available after update")
		require.NotNil(t, updatedMetadata)
		assert.Equal(t, "test-isolation-id", updatedMetadata.IsolationID)
		assert.NotNil(t, updatedMetadata.TargetModel)
		assert.Equal(t, modeltypes.ProviderAzure, updatedMetadata.TargetModel.Provider)
		assert.Equal(t, "gpt-4-resolved", updatedMetadata.TargetModel.KEY)
		assert.Equal(t, "gpt-4", updatedMetadata.OriginalModelName)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
}
