/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

func TestHandleRealtimeProxyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns 500 when GENAI_URL is not configured", func(t *testing.T) {
		ctx := cntx.NewTestContext("realtime-test")
		// No AzureGenAIURL set — should return 500

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/openai/deployments/gpt-realtime/v1/realtime/client_secrets", nil)

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "service configuration error")
	})

	t.Run("proxies request to correct target URL", func(t *testing.T) {
		var receivedPath string
		var receivedMethod string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedPath = r.URL.RequestURI()
			receivedMethod = r.Method
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"status":"ok"}`)
		}))
		defer backend.Close()

		ctx := cntx.NewTestContext("realtime-test")
		ctx = cntx.WithAzureGenAIURL(ctx, backend.URL)

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(
			http.MethodPost,
			"/openai/deployments/gpt-realtime/v1/realtime/client_secrets",
			strings.NewReader(`{"session":{"type":"realtime"}}`),
		)

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		assert.Equal(t, http.MethodPost, receivedMethod)
		assert.Equal(t, "/openai/deployments/gpt-realtime/v1/realtime/client_secrets", receivedPath)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("preserves safe query parameters in target URL", func(t *testing.T) {
		var receivedURI string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedURI = r.URL.RequestURI()
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		}))
		defer backend.Close()

		ctx := cntx.NewTestContext("realtime-test")
		ctx = cntx.WithAzureGenAIURL(ctx, backend.URL)

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(
			http.MethodPost,
			"/openai/deployments/gpt-realtime/v1/realtime/calls?model=gpt-realtime",
			strings.NewReader("v=0\r\no=- 0 0 IN IP4 127.0.0.1"),
		)

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		assert.Equal(t, "/openai/deployments/gpt-realtime/v1/realtime/calls?model=gpt-realtime", receivedURI)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("trims trailing slash from base URL", func(t *testing.T) {
		var receivedURI string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedURI = r.URL.RequestURI()
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		}))
		defer backend.Close()

		ctx := cntx.NewTestContext("realtime-test")
		// URL with trailing slash
		ctx = cntx.WithAzureGenAIURL(ctx, backend.URL+"/")

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(
			http.MethodPost,
			"/openai/deployments/gpt-realtime/v1/realtime/client_secrets",
			nil,
		)

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		// Should not have double slash
		assert.Equal(t, "/openai/deployments/gpt-realtime/v1/realtime/client_secrets", receivedURI)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("returns upstream error status", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, `{"error":"unauthorized"}`)
		}))
		defer backend.Close()

		ctx := cntx.NewTestContext("realtime-test")
		ctx = cntx.WithAzureGenAIURL(ctx, backend.URL)

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(
			http.MethodPost,
			"/openai/deployments/gpt-realtime/v1/realtime/client_secrets",
			strings.NewReader(`{"session":{}}`),
		)

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("forwards request headers to upstream", func(t *testing.T) {
		var receivedContentType string
		var receivedAuth string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedContentType = r.Header.Get("Content-Type")
			receivedAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		}))
		defer backend.Close()

		ctx := cntx.NewTestContext("realtime-test")
		ctx = cntx.WithAzureGenAIURL(ctx, backend.URL)

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		req := httptest.NewRequest(
			http.MethodPost,
			"/openai/deployments/gpt-realtime/v1/realtime/client_secrets",
			strings.NewReader(`{"session":{}}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		c.Request = req

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		assert.Equal(t, "application/json", receivedContentType)
		assert.Equal(t, "Bearer test-token", receivedAuth)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("handles unreachable upstream", func(t *testing.T) {
		ctx := cntx.NewTestContext("realtime-test")
		// Port 1 is reserved/unreachable
		ctx = cntx.WithAzureGenAIURL(ctx, "http://127.0.0.1:1")

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(
			http.MethodPost,
			"/openai/deployments/gpt-realtime/v1/realtime/client_secrets",
			strings.NewReader(`{"session":{}}`),
		)

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	})

	t.Run("rejects path traversal attempts", func(t *testing.T) {
		ctx := cntx.NewTestContext("realtime-test")
		ctx = cntx.WithAzureGenAIURL(ctx, "http://127.0.0.1:9999")

		paths := []string{
			"/openai/deployments/../../admin/secrets",
			"/other-service/endpoint",
			"/v1/realtime/client_secrets",
		}
		for _, p := range paths {
			t.Run(p, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, p, nil)

				handler := HandleRealtimeProxyRequest(ctx)
				handler(c)

				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assert.Contains(t, recorder.Body.String(), "invalid realtime request path")
			})
		}
	})

	t.Run("strips sensitive query parameters", func(t *testing.T) {
		var receivedURI string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedURI = r.URL.RequestURI()
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		}))
		defer backend.Close()

		ctx := cntx.NewTestContext("realtime-test")
		ctx = cntx.WithAzureGenAIURL(ctx, backend.URL)

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(
			http.MethodPost,
			"/openai/deployments/gpt-realtime/v1/realtime/calls?model=gpt-realtime&api-key=secret123&subscription-key=sub456",
			strings.NewReader(`{}`),
		)

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.NotContains(t, receivedURI, "api-key")
		assert.NotContains(t, receivedURI, "subscription-key")
		assert.Contains(t, receivedURI, "model=gpt-realtime")
	})

	t.Run("strips all query params when only sensitive ones present", func(t *testing.T) {
		var receivedURI string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedURI = r.URL.RequestURI()
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		}))
		defer backend.Close()

		ctx := cntx.NewTestContext("realtime-test")
		ctx = cntx.WithAzureGenAIURL(ctx, backend.URL)

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(
			http.MethodPost,
			"/openai/deployments/gpt-realtime/v1/realtime/calls?api-key=secret",
			strings.NewReader(`{}`),
		)

		handler := HandleRealtimeProxyRequest(ctx)
		handler(c)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "/openai/deployments/gpt-realtime/v1/realtime/calls", receivedURI)
	})
}

func TestSanitizeLogValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean string passes through",
			input:    "/openai/deployments/gpt-realtime/v1/realtime/calls",
			expected: "/openai/deployments/gpt-realtime/v1/realtime/calls",
		},
		{
			name:     "newlines replaced",
			input:    "/path\nfake-log-line",
			expected: "/path_fake-log-line",
		},
		{
			name:     "carriage returns replaced",
			input:    "/path\r\nfake-log-line",
			expected: "/path__fake-log-line",
		},
		{
			name:     "tabs replaced",
			input:    "/path\ttab-injected",
			expected: "/path_tab-injected",
		},
		{
			name:     "null bytes replaced",
			input:    "/path\x00null",
			expected: "/path_null",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeLogValue(tt.input))
		})
	}
}

func TestStripSensitiveQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:     "empty query",
			input:    "",
			contains: nil,
			excludes: nil,
		},
		{
			name:     "no sensitive params",
			input:    "model=gpt-4&version=2024",
			contains: []string{"model=gpt-4", "version=2024"},
			excludes: nil,
		},
		{
			name:     "strips api-key",
			input:    "model=gpt-4&api-key=secret",
			contains: []string{"model=gpt-4"},
			excludes: []string{"api-key", "secret"},
		},
		{
			name:     "strips subscription-key",
			input:    "subscription-key=sub123&model=gpt-4",
			contains: []string{"model=gpt-4"},
			excludes: []string{"subscription-key", "sub123"},
		},
		{
			name:     "strips all sensitive leaves empty",
			input:    "api-key=a&subscription-key=b",
			contains: nil,
			excludes: []string{"api-key", "subscription-key"},
		},
		{
			name:     "strips case-insensitive Api-Key",
			input:    "model=gpt-4&Api-Key=secret",
			contains: []string{"model=gpt-4"},
			excludes: []string{"Api-Key", "secret"},
		},
		{
			name:     "strips case-insensitive API-KEY",
			input:    "API-KEY=secret&model=gpt-4",
			contains: []string{"model=gpt-4"},
			excludes: []string{"API-KEY", "secret"},
		},
		{
			name:     "malformed query returns empty",
			input:    "invalid%zzquery&bad=",
			contains: nil,
			excludes: []string{"invalid", "bad"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripSensitiveQuery(tt.input)
			// For malformed query test, result should be empty
			if tt.name == "malformed query returns empty" {
				assert.Empty(t, result)
				return
			}
			for _, c := range tt.contains {
				assert.Contains(t, result, c)
			}
			for _, e := range tt.excludes {
				assert.NotContains(t, result, e)
			}
		})
	}
}
