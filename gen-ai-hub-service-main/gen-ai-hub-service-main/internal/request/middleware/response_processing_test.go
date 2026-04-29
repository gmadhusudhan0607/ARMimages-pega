/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/ginctx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
)

func TestIsStreamingRequestFromBody(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		expected bool
	}{
		{
			name:     "streaming request with stream true",
			body:     []byte(`{"messages": [{"role": "user", "content": "hello"}], "stream": true}`),
			expected: true,
		},
		{
			name:     "streaming request with stream true and spaces",
			body:     []byte(`{"messages": [{"role": "user", "content": "hello"}], "stream": true, "model": "gpt-4"}`),
			expected: true,
		},
		{
			name:     "streaming request with stream true no spaces",
			body:     []byte(`{"messages":[{"role":"user","content":"hello"}],"stream":true}`),
			expected: true,
		},
		{
			name:     "non-streaming request with stream false",
			body:     []byte(`{"messages": [{"role": "user", "content": "hello"}], "stream": false}`),
			expected: false,
		},
		{
			name:     "non-streaming request without stream parameter",
			body:     []byte(`{"messages": [{"role": "user", "content": "hello"}]}`),
			expected: false,
		},
		{
			name:     "empty body",
			body:     []byte(``),
			expected: false,
		},
		{
			name:     "invalid json",
			body:     []byte(`{"invalid": json`),
			expected: false,
		},
		{
			name:     "nil body",
			body:     nil,
			expected: false,
		},
		{
			name:     "invalid JSON containing stream keyword",
			body:     []byte(`{"stream": not-valid-json`),
			expected: false,
		},
		{
			name:     "stream as string value instead of bool",
			body:     []byte(`{"stream": "true"}`),
			expected: false,
		},
		{
			name:     "nested stream field is ignored at top level",
			body:     []byte(`{"options": {"stream": true}}`),
			expected: false,
		},
		{
			name:     "stream field in a larger payload",
			body:     []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true,"temperature":0.7}`),
			expected: true,
		},
		{
			name:     "stream as numeric value instead of bool",
			body:     []byte(`{"stream": 1}`),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStreamingRequestFromBody(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBedrockStreamingApi(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		targetApi string
		expected  bool
	}{
		{
			name:      "converse-stream API",
			targetApi: "converse-stream",
			expected:  true,
		},
		{
			name:      "invoke-stream API",
			targetApi: "invoke-stream",
			expected:  true,
		},
		{
			name:      "converse-stream with path prefix",
			targetApi: "/model/anthropic.claude-v2/converse-stream",
			expected:  true,
		},
		{
			name:      "non-streaming converse API",
			targetApi: "converse",
			expected:  false,
		},
		{
			name:      "non-streaming invoke API",
			targetApi: "invoke",
			expected:  false,
		},
		{
			name:      "empty targetApi",
			targetApi: "",
			expected:  false,
		},
		{
			name:      "chat completions API",
			targetApi: "chat/completions",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "targetApi", Value: tt.targetApi}}

			result := isBedrockStreamingApi(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveMaxTokensFromRequest(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		originalBody   []byte
		expectedBody   string
		expectError    bool
		expectModified bool
	}{
		{
			name:           "remove max_tokens from request",
			originalBody:   []byte(`{"messages": [{"role": "user", "content": "hello"}], "max_tokens": 100}`),
			expectedBody:   `{"messages":[{"role":"user","content":"hello"}]}`,
			expectError:    false,
			expectModified: true,
		},
		{
			name:           "no max_tokens to remove",
			originalBody:   []byte(`{"messages": [{"role": "user", "content": "hello"}]}`),
			expectedBody:   `{"messages": [{"role": "user", "content": "hello"}]}`,
			expectError:    false,
			expectModified: false,
		},
		{
			name:           "complex request with max_tokens",
			originalBody:   []byte(`{"messages": [{"role": "user", "content": "hello"}], "model": "gpt-4", "max_tokens": 150, "temperature": 0.7}`),
			expectedBody:   `{"messages":[{"role":"user","content":"hello"}],"model":"gpt-4","temperature":0.7}`,
			expectError:    false,
			expectModified: true,
		},
		{
			name:           "invalid json",
			originalBody:   []byte(`{"invalid": json`),
			expectedBody:   `{"invalid": json`,
			expectError:    true,
			expectModified: false,
		},
		{
			name:           "empty json object",
			originalBody:   []byte(`{}`),
			expectedBody:   `{}`,
			expectError:    false,
			expectModified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req

			result, err := removeMaxTokensFromRequest(c, tt.originalBody)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, string(tt.originalBody), string(result)) // Should return original body on error
			} else {
				assert.NoError(t, err)
				if tt.expectModified {
					assert.JSONEq(t, tt.expectedBody, string(result))
				} else {
					assert.Equal(t, string(tt.originalBody), string(result))
				}
			}
		})
	}
}

func TestGetOriginalRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		setupFunc   func(*gin.Context)
		expectError bool
		expected    string
	}{
		{
			name: "valid original request body",
			setupFunc: func(c *gin.Context) {
				body := []byte(`{"messages": [{"role": "user", "content": "hello"}]}`)
				ctx := context.WithValue(c.Request.Context(), OriginalRequestBodyKey, body)
				c.Request = c.Request.WithContext(ctx)
			},
			expectError: false,
			expected:    `{"messages": [{"role": "user", "content": "hello"}]}`,
		},
		{
			name: "no original request body in context",
			setupFunc: func(c *gin.Context) {
				// Don't add anything to context
			},
			expectError: true,
			expected:    "",
		},
		{
			name: "invalid type in context",
			setupFunc: func(c *gin.Context) {
				ctx := context.WithValue(c.Request.Context(), OriginalRequestBodyKey, "not-a-byte-slice")
				c.Request = c.Request.WithContext(ctx)
			},
			expectError: true,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req

			tt.setupFunc(c)

			result, err := getOriginalRequestBody(c)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, string(result))
			}
		})
	}
}

func TestUpdateRetryMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupFunc      func(*gin.Context)
		expectError    bool
		expectedCount  int
		expectedReason string
	}{
		{
			name: "successful retry metrics update",
			setupFunc: func(c *gin.Context) {
				// Create metadata with target model
				md := &metadata.RequestMetadata{
					IsolationID: "test-isolation",
					TargetModel: &types.Model{
						Name: "gpt-4",
					},
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)
			},
			expectError:    false,
			expectedCount:  1,
			expectedReason: "max_tokens_exceeded",
		},
		{
			name: "no metadata in context",
			setupFunc: func(c *gin.Context) {
				// Don't add metadata to context
			},
			expectError: true,
		},
		{
			name: "metadata without target model",
			setupFunc: func(c *gin.Context) {
				md := &metadata.RequestMetadata{
					IsolationID: "test-isolation",
					TargetModel: nil, // No target model
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req

			tt.setupFunc(c)

			result, err := updateRetryMetrics(c)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedCount, result.RequestMetrics.RetryMetrics.Count)
				assert.NotNil(t, result.RequestMetrics.RetryMetrics.Reason)
				assert.Equal(t, tt.expectedReason, *result.RequestMetrics.RetryMetrics.Reason)
			}
		})
	}
}

func TestShouldRetryForTruncation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupFunc func(*gin.Context)
		envVar    string
		envValue  string
		expected  bool
	}{
		{
			name: "should retry - non-streaming truncated request",
			setupFunc: func(c *gin.Context) {
				// Setup metadata with truncation
				md := &metadata.RequestMetadata{
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{
							ResponseTruncated: true,
						},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)

				// Add non-streaming original body
				body := []byte(`{"messages": [{"role": "user", "content": "hello"}], "stream": false}`)
				ctx = context.WithValue(ctx, OriginalRequestBodyKey, body)

				c.Request = c.Request.WithContext(ctx)
			},
			expected: true,
		},
		{
			name: "should not retry - already a retry attempt",
			setupFunc: func(c *gin.Context) {
				// Setup as retry attempt
				ctx := context.WithValue(c.Request.Context(), RetryAttemptContextKey, true)

				// Setup metadata with truncation
				md := &metadata.RequestMetadata{
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{
							ResponseTruncated: true,
						},
					},
				}
				ctx = context.WithValue(ctx, metrics.RequestMetadataContextKey{}, md)

				c.Request = c.Request.WithContext(ctx)
			},
			expected: false,
		},
		{
			name: "should not retry - no truncation",
			setupFunc: func(c *gin.Context) {
				// Setup metadata without truncation
				md := &metadata.RequestMetadata{
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{
							ResponseTruncated: false,
						},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)
			},
			expected: false,
		},
		{
			name: "should not retry - streaming (never retries)",
			setupFunc: func(c *gin.Context) {
				// Setup metadata with truncation
				md := &metadata.RequestMetadata{
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{
							ResponseTruncated: true,
						},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)

				// Add streaming original body
				body := []byte(`{"messages": [{"role": "user", "content": "hello"}], "stream": true}`)
				ctx = context.WithValue(ctx, OriginalRequestBodyKey, body)

				c.Request = c.Request.WithContext(ctx)
			},
			expected: false,
		},
		{
			name: "should not retry - no original body",
			setupFunc: func(c *gin.Context) {
				// Setup metadata with truncation
				md := &metadata.RequestMetadata{
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{
							ResponseTruncated: true,
						},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)
				// Don't add original body
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if specified
			if tt.envVar != "" {
				oldValue := os.Getenv(tt.envVar)
				defer func() {
					if oldValue == "" {
						os.Unsetenv(tt.envVar)
					} else {
						os.Setenv(tt.envVar, oldValue)
					}
				}()
				os.Setenv(tt.envVar, tt.envValue)
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req

			tt.setupFunc(c)

			result := shouldRetryForTruncation(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPerformRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		setupFunc   func(*gin.Context)
		expectError bool
	}{
		{
			name: "perform retry fails - no original body",
			setupFunc: func(c *gin.Context) {
				// Setup metadata without original body
				md := &metadata.RequestMetadata{
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)
				// Don't add original body
			},
			expectError: true,
		},
		{
			name: "perform retry fails - no metadata",
			setupFunc: func(c *gin.Context) {
				// Add original body but no metadata
				body := []byte(`{"messages": [{"role": "user", "content": "hello"}]}`)
				ctx := context.WithValue(c.Request.Context(), OriginalRequestBodyKey, body)
				c.Request = c.Request.WithContext(ctx)
			},
			expectError: true,
		},
		{
			name: "perform retry fails - no target model",
			setupFunc: func(c *gin.Context) {
				// Setup metadata without target model
				md := &metadata.RequestMetadata{
					TargetModel: nil,
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)

				// Add original body
				body := []byte(`{"messages": [{"role": "user", "content": "hello"}]}`)
				ctx = context.WithValue(ctx, OriginalRequestBodyKey, body)

				c.Request = c.Request.WithContext(ctx)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test?api-version=2023-12-01", bytes.NewBuffer([]byte(`{"test": "data"}`)))
			c.Request = req

			tt.setupFunc(c)

			err := performRetry(c)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// setupPerformRetryContext creates a gin context with all required context values for a successful performRetry call.
// It starts a mock HTTP server and returns the server (caller must defer server.Close()) along with a channel
// that receives the request body sent to the mock server.
func setupPerformRetryContext(t *testing.T, originalBody []byte, mockStatus int, mockResponseBody string) (*gin.Context, *httptest.ResponseRecorder, *httptest.Server, chan []byte) {
	t.Helper()

	receivedBody := make(chan []byte, 1)

	// Create mock upstream server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody <- body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(mockStatus)
		_, _ = w.Write([]byte(mockResponseBody))
	}))

	// Create gin context with RequestModificationResponseWriter
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(originalBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	logger := zap.NewNop().Sugar()
	customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = customWriter

	// Set model URL to mock server
	c.Set(ginctx.ModelURLContextKey, server.URL)

	// Setup metadata with target model
	md := &metadata.RequestMetadata{
		IsolationID: "test-isolation",
		TargetModel: &types.Model{
			Name: "gpt-4",
		},
		RequestMetrics: metrics.RequestMetrics{
			RetryMetrics: metrics.RetryMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)

	// Add original body to context
	ctx = context.WithValue(ctx, OriginalRequestBodyKey, originalBody)
	c.Request = c.Request.WithContext(ctx)

	return c, w, server, receivedBody
}

func TestPerformRetry_SuccessfulRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalBody := []byte(`{"messages":[{"role":"user","content":"hello"}],"max_tokens":100}`)
	mockResponse := `{"choices":[{"message":{"content":"full response"}}]}`

	c, w, server, receivedBody := setupPerformRetryContext(t, originalBody, http.StatusOK, mockResponse)
	defer server.Close()

	// Execute
	err := performRetry(c)

	// Verify no error
	assert.NoError(t, err)

	// Verify retry response was written to the recorder
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, mockResponse, w.Body.String())

	// Verify retry metrics were updated
	md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
	assert.NoError(t, err)
	assert.Equal(t, 1, md.RequestMetrics.RetryMetrics.Count)
	assert.NotNil(t, md.RequestMetrics.RetryMetrics.Reason)
	assert.Equal(t, "max_tokens_exceeded", *md.RequestMetrics.RetryMetrics.Reason)

	// Verify the request body sent to mock server has max_tokens removed
	sentBody := <-receivedBody
	var sentData map[string]interface{}
	err = json.Unmarshal(sentBody, &sentData)
	assert.NoError(t, err)
	_, hasMaxTokens := sentData["max_tokens"]
	assert.False(t, hasMaxTokens, "max_tokens should be removed from retry request")
	assert.Contains(t, sentData, "messages", "messages should still be present")
}

func TestPerformRetry_SuccessfulRetryWithoutMaxTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Original body without max_tokens - retry should still work, body forwarded as-is
	originalBody := []byte(`{"messages":[{"role":"user","content":"hello"}],"temperature":0.7}`)
	mockResponse := `{"choices":[{"message":{"content":"response"}}]}`

	c, w, server, receivedBody := setupPerformRetryContext(t, originalBody, http.StatusOK, mockResponse)
	defer server.Close()

	// Execute
	err := performRetry(c)

	// Verify no error
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, mockResponse, w.Body.String())

	// Verify body was forwarded unchanged
	sentBody := <-receivedBody
	assert.JSONEq(t, string(originalBody), string(sentBody))
}

func TestPerformRetry_UpstreamReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalBody := []byte(`{"messages":[{"role":"user","content":"hello"}],"max_tokens":100}`)
	mockErrorResponse := `{"error":{"message":"internal server error","type":"server_error"}}`

	c, w, server, _ := setupPerformRetryContext(t, originalBody, http.StatusInternalServerError, mockErrorResponse)
	defer server.Close()

	// Execute - the retry itself succeeds even though upstream returned 500
	// The error response is forwarded to the client
	err := performRetry(c)

	// performRetry does not return error for non-200 upstream responses; it writes the response through
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, mockErrorResponse, w.Body.String())
}

func TestPerformRetry_UpstreamReturns429(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalBody := []byte(`{"messages":[{"role":"user","content":"hello"}],"max_tokens":100}`)
	mockRateLimitResponse := `{"error":{"message":"rate limited","type":"rate_limit_error"}}`

	// Create mock upstream that returns 429 with Retry-After header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(mockRateLimitResponse))
	}))
	defer server.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(originalBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	logger := zap.NewNop().Sugar()
	customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = customWriter

	c.Set(ginctx.ModelURLContextKey, server.URL)

	md := &metadata.RequestMetadata{
		IsolationID: "test-isolation",
		TargetModel: &types.Model{Name: "gpt-4"},
		RequestMetrics: metrics.RequestMetrics{
			RetryMetrics: metrics.RetryMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	ctx = context.WithValue(ctx, OriginalRequestBodyKey, originalBody)
	c.Request = c.Request.WithContext(ctx)

	// Execute
	err := performRetry(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "30", w.Header().Get("Retry-After"))
	assert.JSONEq(t, mockRateLimitResponse, w.Body.String())
}

func TestPerformRetry_NetworkFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalBody := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(originalBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	logger := zap.NewNop().Sugar()
	customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = customWriter

	// Point to an unreachable address
	c.Set(ginctx.ModelURLContextKey, "http://127.0.0.1:1")

	md := &metadata.RequestMetadata{
		IsolationID: "test-isolation",
		TargetModel: &types.Model{Name: "gpt-4"},
		RequestMetrics: metrics.RequestMetrics{
			RetryMetrics: metrics.RetryMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	ctx = context.WithValue(ctx, OriginalRequestBodyKey, originalBody)
	c.Request = c.Request.WithContext(ctx)

	// Execute
	err := performRetry(c)

	// Should fail due to network error
	assert.Error(t, err)
}

func TestPerformRetry_HeadersForwarded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalBody := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)

	receivedHeaders := make(chan http.Header, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders <- r.Header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(originalBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-123")
	req.Header.Set("X-Custom-Header", "custom-value")
	c.Request = req

	logger := zap.NewNop().Sugar()
	customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = customWriter

	c.Set(ginctx.ModelURLContextKey, server.URL)

	md := &metadata.RequestMetadata{
		IsolationID: "test-isolation",
		TargetModel: &types.Model{Name: "gpt-4"},
		RequestMetrics: metrics.RequestMetrics{
			RetryMetrics: metrics.RetryMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	ctx = context.WithValue(ctx, OriginalRequestBodyKey, originalBody)
	c.Request = c.Request.WithContext(ctx)

	// Execute
	err := performRetry(c)
	assert.NoError(t, err)

	// Verify original request headers were forwarded
	headers := <-receivedHeaders
	assert.Equal(t, "Bearer test-token-123", headers.Get("Authorization"))
	assert.Equal(t, "custom-value", headers.Get("X-Custom-Header"))
}

func TestPerformRetry_ContextCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalBody := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req, _ := http.NewRequestWithContext(cancelCtx, "POST", "/test", bytes.NewBuffer(originalBody))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	logger := zap.NewNop().Sugar()
	customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = customWriter

	c.Set(ginctx.ModelURLContextKey, server.URL)

	md := &metadata.RequestMetadata{
		IsolationID: "test-isolation",
		TargetModel: &types.Model{Name: "gpt-4"},
		RequestMetrics: metrics.RequestMetrics{
			RetryMetrics: metrics.RetryMetrics{},
		},
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	ctx = context.WithValue(ctx, OriginalRequestBodyKey, originalBody)
	c.Request = c.Request.WithContext(ctx)

	// Execute - should fail because context is cancelled
	err := performRetry(c)
	assert.Error(t, err)
}
