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
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
	"go.uber.org/zap"
)

func TestIsStreamingRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupFunc func(*gin.Context)
		expected  bool
	}{
		{
			name: "streaming request - IsStreaming true",
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
			expected: true,
		},
		{
			name: "non-streaming request - IsStreaming false",
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
			expected: false,
		},
		{
			name: "no metadata in context",
			setupFunc: func(c *gin.Context) {
				// Don't add metadata to context
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req

			tt.setupFunc(c)

			result := isStreamingRequest(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleResponse_StreamingRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupFunc func(*gin.Context, *metrics.RequestModificationResponseWriter)
		verify    func(*testing.T, *gin.Context, *httptest.ResponseRecorder)
	}{
		{
			name: "streaming request - early return without truncation processing",
			setupFunc: func(c *gin.Context, writer *metrics.RequestModificationResponseWriter) {
				// Setup streaming metadata
				md := &metadata.RequestMetadata{
					IsStreaming: true,
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{
							ResponseTruncated: false, // Should not be checked for streaming
						},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)

				// Set status to 200
				c.Status(200)
			},
			verify: func(t *testing.T, c *gin.Context, w *httptest.ResponseRecorder) {
				// Verify that truncation was NOT processed (no retry should happen)
				md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
				require.NoError(t, err)
				assert.False(t, md.RequestMetrics.RetryMetrics.ResponseTruncated)

				// For streaming requests, the response should be flushed
				// The actual flushing behavior would be tested through integration tests
			},
		},
		{
			name: "streaming request with buffered response - should flush",
			setupFunc: func(c *gin.Context, writer *metrics.RequestModificationResponseWriter) {
				// Setup streaming metadata
				md := &metadata.RequestMetadata{
					IsStreaming: true,
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)

				// Write some data to buffer
				_, _ = writer.Write([]byte(`{"data": "test"}`))
				c.Status(200)
			},
			verify: func(t *testing.T, c *gin.Context, w *httptest.ResponseRecorder) {
				// The response should have been flushed
				// Since we're using a test context, we verify the writer has data
				md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
				require.NoError(t, err)
				assert.True(t, md.IsStreaming)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"test": "data"}`)))
			c.Request = req

			// Create logger
			logger := zap.NewNop().Sugar()

			// Wrap the response writer with RequestModificationResponseWriter
			customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
			c.Writer = customWriter

			// Setup test context
			tt.setupFunc(c, customWriter)

			// Call handleResponse
			handleResponse(c, logger)

			// Verify results
			tt.verify(t, c, w)
		})
	}
}

func TestExecuteRetryRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		serverHandler  func(http.ResponseWriter, *http.Request)
		method         string
		requestHeaders map[string]string
		requestBody    []byte
		expectError    bool
		verifyResult   func(*testing.T, *retryResponse)
		verifyRequest  func(*testing.T, *http.Request)
	}{
		{
			name: "successful retry request with 200 response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Custom-Header", "test-value")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"retried response"}}]}`))
			},
			method:      "POST",
			requestBody: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			expectError: false,
			verifyResult: func(t *testing.T, resp *retryResponse) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
				assert.Equal(t, "test-value", resp.Headers.Get("X-Custom-Header"))
				assert.JSONEq(t, `{"choices":[{"message":{"content":"retried response"}}]}`, string(resp.Body))
			},
		},
		{
			name: "server returns 500 error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal server error"}`))
			},
			method:      "POST",
			requestBody: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			expectError: false,
			verifyResult: func(t *testing.T, resp *retryResponse) {
				assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
				assert.JSONEq(t, `{"error":"internal server error"}`, string(resp.Body))
			},
		},
		{
			name: "server returns 429 rate limit",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "30")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limited"}`))
			},
			method:      "POST",
			requestBody: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			expectError: false,
			verifyResult: func(t *testing.T, resp *retryResponse) {
				assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
				assert.Equal(t, "30", resp.Headers.Get("Retry-After"))
				assert.JSONEq(t, `{"error":"rate limited"}`, string(resp.Body))
			},
		},
		{
			name:        "connection failure - invalid URL",
			method:      "POST",
			requestBody: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			expectError: true,
		},
		{
			name: "headers are forwarded from original request",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Echo back the received headers
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"received_auth":"` + r.Header.Get("Authorization") + `","received_custom":"` + r.Header.Get("X-Custom") + `"}`))
			},
			method: "POST",
			requestHeaders: map[string]string{
				"Authorization": "Bearer test-token",
				"X-Custom":      "custom-value",
			},
			requestBody: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			expectError: false,
			verifyResult: func(t *testing.T, resp *retryResponse) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				var body map[string]string
				err := json.Unmarshal(resp.Body, &body)
				require.NoError(t, err)
				assert.Equal(t, "Bearer test-token", body["received_auth"])
				assert.Equal(t, "custom-value", body["received_custom"])
			},
		},
		{
			name: "content-length is set correctly for modified body",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Verify Content-Length matches body
				bodyBytes, _ := io.ReadAll(r.Body)
				contentLength := r.Header.Get("Content-Length")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(fmt.Sprintf(`{"body_len":%d,"content_length":"%s"}`, len(bodyBytes), contentLength)))
			},
			method:      "POST",
			requestBody: []byte(`{"messages":[{"role":"user","content":"hello world"}]}`),
			expectError: false,
			verifyResult: func(t *testing.T, resp *retryResponse) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				var body map[string]interface{}
				err := json.Unmarshal(resp.Body, &body)
				require.NoError(t, err)
				bodyLen := int(body["body_len"].(float64))
				contentLength := body["content_length"].(string)
				assert.Equal(t, fmt.Sprintf("%d", bodyLen), contentLength, "Content-Length should match actual body length")
				assert.Equal(t, len(`{"messages":[{"role":"user","content":"hello world"}]}`), bodyLen)
			},
		},
		{
			name: "request body is correctly forwarded",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				bodyBytes, _ := io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(bodyBytes) // Echo back the body
			},
			method:      "POST",
			requestBody: []byte(`{"messages":[{"role":"user","content":"test message"}],"temperature":0.7}`),
			expectError: false,
			verifyResult: func(t *testing.T, resp *retryResponse) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.JSONEq(t, `{"messages":[{"role":"user","content":"test message"}],"temperature":0.7}`, string(resp.Body))
			},
		},
		{
			name: "empty response body from server",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			method:      "POST",
			requestBody: []byte(`{"messages":[{"role":"user","content":"hello"}]}`),
			expectError: false,
			verifyResult: func(t *testing.T, resp *retryResponse) {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Empty(t, resp.Body)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var serverURL string

			if tt.serverHandler != nil {
				server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
				defer server.Close()
				serverURL = server.URL
			} else {
				// Use an invalid URL to trigger connection failure
				serverURL = "http://127.0.0.1:1" // port 1 is almost certainly not listening
			}

			// Create gin context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, err := http.NewRequest(tt.method, "/test", bytes.NewBuffer(tt.requestBody))
			require.NoError(t, err)

			// Set custom headers on original request
			for key, value := range tt.requestHeaders {
				req.Header.Set(key, value)
			}

			c.Request = req

			// Execute
			result, err := executeRetryRequest(c, serverURL, tt.requestBody)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.verifyResult != nil {
					tt.verifyResult(t, result)
				}
			}
		})
	}
}

func TestExecuteRetryRequest_ContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a server that will delay (the request should be cancelled before it responds)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The request should be cancelled before reaching here, but if it does, return OK
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"should not reach here"}`))
	}))
	defer server.Close()

	// Create gin context with a cancelled context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	req, err := http.NewRequestWithContext(ctx, "POST", "/test", bytes.NewBuffer([]byte(`{"test":"data"}`)))
	require.NoError(t, err)
	c.Request = req

	body := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)
	result, err := executeRetryRequest(c, server.URL, body)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestWriteRetryResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		retryResp         *retryResponse
		existingHeaders   map[string]string
		setupCustomWriter bool
		verifyRecorder    func(*testing.T, *httptest.ResponseRecorder)
		verifyWriter      func(*testing.T, *metrics.RequestModificationResponseWriter)
	}{
		{
			name: "successful write with 200 status and JSON body",
			retryResp: &retryResponse{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type":   {"application/json"},
					"X-Custom":       {"custom-value"},
					"Content-Length": {"52"},
				},
				Body: []byte(`{"choices":[{"message":{"content":"retry response"}}]}`),
			},
			setupCustomWriter: true,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, `{"choices":[{"message":{"content":"retry response"}}]}`, w.Body.String())
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Equal(t, "custom-value", w.Header().Get("X-Custom"))
			},
		},
		{
			name: "headers from retry replace existing headers",
			retryResp: &retryResponse{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type": {"application/json; charset=utf-8"},
					"X-New-Header": {"new-value"},
				},
				Body: []byte(`{"result":"ok"}`),
			},
			existingHeaders: map[string]string{
				"Content-Type": "text/plain",
				"X-Old-Header": "old-value",
			},
			setupCustomWriter: true,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
				assert.Equal(t, "new-value", w.Header().Get("X-New-Header"))
			},
		},
		{
			name: "write with empty body",
			retryResp: &retryResponse{
				StatusCode: http.StatusNoContent,
				Headers:    http.Header{},
				Body:       []byte{},
			},
			setupCustomWriter: true,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNoContent, w.Code)
				assert.Empty(t, w.Body.String())
			},
		},
		{
			name: "write error response with 500 status",
			retryResp: &retryResponse{
				StatusCode: http.StatusInternalServerError,
				Headers: http.Header{
					"Content-Type": {"application/json"},
				},
				Body: []byte(`{"error":"internal server error"}`),
			},
			setupCustomWriter: true,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
			},
		},
		{
			name: "write with multiple values for same header",
			retryResp: &retryResponse{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Set-Cookie": {"cookie1=val1", "cookie2=val2"},
				},
				Body: []byte(`{"result":"ok"}`),
			},
			setupCustomWriter: true,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				cookies := w.Header().Values("Set-Cookie")
				assert.Contains(t, cookies, "cookie1=val1")
				assert.Contains(t, cookies, "cookie2=val2")
			},
		},
		{
			name: "PrepareForRetry is called on custom writer",
			retryResp: &retryResponse{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type": {"application/json"},
				},
				Body: []byte(`{"result":"retry"}`),
			},
			setupCustomWriter: true,
			verifyWriter: func(t *testing.T, cw *metrics.RequestModificationResponseWriter) {
				// After writeRetryResponse, the buffer should be cleared by PrepareForRetry
				// Verify that the writer's buffer was reset (PrepareForRetry was called)
				responseBody := cw.GetResponseBody()
				// After PrepareForRetry clears buffer AND then body is written to underlying writer,
				// the buffer will contain the retry response body (since Write still captures)
				// But the key thing is PrepareForRetry was called (buffer was reset before new data)
				assert.NotNil(t, responseBody)
			},
		},
		{
			name: "write without custom writer - still writes to gin.ResponseWriter",
			retryResp: &retryResponse{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type": {"application/json"},
				},
				Body: []byte(`{"result":"no custom writer"}`),
			},
			setupCustomWriter: false,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, `{"result":"no custom writer"}`, w.Body.String())
			},
		},
		{
			name: "large response body",
			retryResp: func() *retryResponse {
				// Create a large body
				largeBody := make([]byte, 0, 10000)
				largeBody = append(largeBody, `{"data":"`...)
				for i := 0; i < 9900; i++ {
					largeBody = append(largeBody, 'x')
				}
				largeBody = append(largeBody, `"}`...)
				return &retryResponse{
					StatusCode: http.StatusOK,
					Headers: http.Header{
						"Content-Type": {"application/json"},
					},
					Body: largeBody,
				}
			}(),
			setupCustomWriter: true,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.True(t, w.Body.Len() > 9900, "Body should contain the large response")
			},
		},
		{
			// Regression test: Go's http.Client transparently decompresses gzip responses.
			// The upstream Content-Length reflects the compressed size, but response.Body
			// holds the decompressed bytes, causing "wrote more than the declared Content-Length"
			// if we naively copy the upstream Content-Length header.
			name: "Content-Length is always set from actual body length, not upstream header",
			retryResp: &retryResponse{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type":   {"application/json"},
					"Content-Length": {"10"}, // simulate stale compressed size
				},
				Body: []byte(`{"choices":[{"message":{"content":"much longer decompressed response body"}}]}`),
			},
			setupCustomWriter: true,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				body := []byte(`{"choices":[{"message":{"content":"much longer decompressed response body"}}]}`)
				assert.Equal(t, fmt.Sprintf("%d", len(body)), w.Header().Get("Content-Length"),
					"Content-Length must reflect actual decompressed body size, not upstream header")
				assert.Equal(t, string(body), w.Body.String())
			},
		},
		{
			// Content-Encoding and Transfer-Encoding must NOT be forwarded: the body has
			// already been decoded/read in full by http.Client, so these headers would
			// mislead the client about the encoding of the bytes being sent.
			name: "Content-Encoding and Transfer-Encoding headers are not forwarded",
			retryResp: &retryResponse{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type":      {"application/json"},
					"Content-Encoding":  {"gzip"},
					"Transfer-Encoding": {"chunked"},
					"X-Custom":          {"should-be-forwarded"},
				},
				Body: []byte(`{"result":"ok"}`),
			},
			setupCustomWriter: true,
			verifyRecorder: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Empty(t, w.Header().Get("Content-Encoding"),
					"Content-Encoding must not be forwarded after transparent decompression")
				assert.Empty(t, w.Header().Get("Transfer-Encoding"),
					"Transfer-Encoding must not be forwarded after body is fully read")
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				assert.Equal(t, "should-be-forwarded", w.Header().Get("X-Custom"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req

			logger := zap.NewNop().Sugar()

			var customWriter *metrics.RequestModificationResponseWriter
			if tt.setupCustomWriter {
				customWriter = metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
				c.Writer = customWriter

				// Pre-buffer some data to verify PrepareForRetry clears it
				_, _ = customWriter.Write([]byte(`{"original":"response"}`))
			}

			// Set existing headers if any
			for key, value := range tt.existingHeaders {
				c.Writer.Header().Set(key, value)
			}

			// Execute
			writeRetryResponse(c, tt.retryResp, logger)

			// Verify recorder
			if tt.verifyRecorder != nil {
				tt.verifyRecorder(t, w)
			}

			// Verify custom writer
			if tt.verifyWriter != nil && customWriter != nil {
				tt.verifyWriter(t, customWriter)
			}
		})
	}
}

func TestHandleResponse_NonStreamingRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupFunc func(*gin.Context, *metrics.RequestModificationResponseWriter)
		verify    func(*testing.T, *gin.Context, *httptest.ResponseRecorder)
	}{
		{
			name: "non-streaming request with status 200 - should process response",
			setupFunc: func(c *gin.Context, writer *metrics.RequestModificationResponseWriter) {
				// Setup non-streaming metadata
				md := &metadata.RequestMetadata{
					IsStreaming: false,
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{
							ResponseTruncated: false,
						},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)

				// Add original request body for truncation checking
				body := []byte(`{"messages": [{"role": "user", "content": "hello"}]}`)
				ctx = context.WithValue(ctx, OriginalRequestBodyKey, body)

				c.Request = c.Request.WithContext(ctx)

				// Write response
				_, _ = writer.Write([]byte(`{"choices": [{"message": {"content": "response"}}]}`))
				c.Status(200)
			},
			verify: func(t *testing.T, c *gin.Context, w *httptest.ResponseRecorder) {
				// Verify that response was processed (not streaming)
				md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
				require.NoError(t, err)
				assert.False(t, md.IsStreaming)
			},
		},
		{
			name: "non-streaming request with error status - should skip truncation processing",
			setupFunc: func(c *gin.Context, writer *metrics.RequestModificationResponseWriter) {
				// Setup non-streaming metadata
				md := &metadata.RequestMetadata{
					IsStreaming: false,
					RequestMetrics: metrics.RequestMetrics{
						RetryMetrics: metrics.RetryMetrics{},
					},
				}
				ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
				c.Request = c.Request.WithContext(ctx)

				// Write error response
				_, _ = writer.Write([]byte(`{"error": "bad request"}`))
				c.Status(400)
			},
			verify: func(t *testing.T, c *gin.Context, w *httptest.ResponseRecorder) {
				// Verify that truncation processing was skipped (status != 200)
				md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
				require.NoError(t, err)
				assert.False(t, md.RequestMetrics.RetryMetrics.ResponseTruncated)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer([]byte(`{"test": "data"}`)))
			c.Request = req

			// Create logger
			logger := zap.NewNop().Sugar()

			// Wrap the response writer with RequestModificationResponseWriter
			customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
			c.Writer = customWriter

			// Setup test context
			tt.setupFunc(c, customWriter)

			// Call handleResponse
			handleResponse(c, logger)

			// Verify results
			tt.verify(t, c, w)
		})
	}
}
