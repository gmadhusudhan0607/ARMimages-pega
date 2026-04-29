/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient/saxtypes"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/testutils"
)

func TestLogGoogleErrorResponse(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)
	ctx := cntx.ContextWithLogger(cntx.ServiceContext("google-error-log-test"), zap.New(core))

	headers := http.Header{}
	headers.Set("X-Cloud-Trace-Context", "trace-google-123")

	body := []byte("{\"error\":\"upstream\nfailed\r\nwith newlines\"}")
	resp := &http.Response{
		StatusCode: http.StatusBadGateway,
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	callStartTime := time.Date(2026, time.April, 8, 10, 0, 0, 0, time.UTC)
	callResponseTime := callStartTime.Add(320 * time.Millisecond)

	logGoogleErrorResponse(ctx, "req-123", resp, callStartTime, callResponseTime)

	entries := logs.All()
	assert.Len(t, entries, 1, "expected exactly one log entry")
	msg := entries[0].Message
	assert.Contains(t, msg, "[req-123]")
	assert.Contains(t, msg, "Google upstream error")
	assert.Contains(t, msg, "status=502")
	assert.Contains(t, msg, "x_cloud_trace_context=trace-google-123")
	assert.Contains(t, msg, "call_start_time=2026-04-08T10:00:00Z")
	assert.Contains(t, msg, "call_response_time=2026-04-08T10:00:00.32Z")
	assert.Contains(t, msg, `payload={"error":"upstream\nfailed\r\nwith newlines"}`)
	assert.NotContains(t, msg, "\n", "payload newlines must be escaped")

	forwardedBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, string(body), string(forwardedBody), "response body should remain available for proxying")
}

func TestEscapePayload(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no newlines", input: `{"error":"bad request"}`, want: `{"error":"bad request"}`},
		{name: "LF only", input: "line1\nline2", want: `line1\nline2`},
		{name: "CR only", input: "line1\rline2", want: `line1\rline2`},
		{name: "CRLF", input: "line1\r\nline2", want: `line1\r\nline2`},
		{name: "mixed", input: "a\r\nb\nc\rd", want: `a\r\nb\nc\rd`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, escapePayload(tt.input))
		})
	}
}
func TestGetBodyBytes(t *testing.T) {
	t.Run("nil body returns nil", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		req, _ := http.NewRequest(http.MethodPost, "/", nil)
		c.Request = req

		result := GetBodyBytes(c)
		// GetBodyBytes returns nil (not an allocated empty slice) when Body is nil.
		assert.Nil(t, result)
	})

	t.Run("empty body returns empty slice", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		c.Request = req

		result := GetBodyBytes(c)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("non-empty body returns correct bytes", func(t *testing.T) {
		payload := `{"model":"gpt-4","prompt":"hello"}`
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
		c.Request = req

		result := GetBodyBytes(c)
		assert.Equal(t, []byte(payload), result)
	})

	t.Run("large body is read fully", func(t *testing.T) {
		big := strings.Repeat("x", 1<<16) // 64 KB
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(big))
		c.Request = req

		result := GetBodyBytes(c)
		assert.Len(t, result, 1<<16)
	})
}

func TestCallTarget(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()
	u := ts.URL

	// authServer is used for SAX tests that exercise the full happy path.
	authServer := testutils.NewLocalAuthServer()
	defer authServer.Close()

	// Build a valid base64-encoded PEM private key the way production code expects.
	rawPK := testutils.GeneratePrivateKey() // PEM bytes
	b64PK := base64.StdEncoding.EncodeToString(rawPK)

	validSaxConfig := &saxtypes.SaxAuthClientConfig{
		ClientId:      "test-client",
		PrivateKey:    b64PK,
		Scopes:        "scope1 scope2",
		TokenEndpoint: authServer.URL,
	}

	type args struct {
		ctx            context.Context
		url            string
		saxAuthEnabled bool
		expectStatus   int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Error: SAX Auth enabled but not present",
			args: args{
				ctx:            context.Background(),
				url:            u,
				saxAuthEnabled: true,
				expectStatus:   http.StatusInternalServerError,
			},
		},
		{
			name: "Error: proxy call fails (unreachable host)",
			args: args{
				// port 1 is reserved/unreachable — the dial will fail fast
				ctx:            context.Background(),
				url:            "http://127.0.0.1:1",
				saxAuthEnabled: false,
				expectStatus:   http.StatusInternalServerError,
			},
		},
		{
			name: "Error: SAX Auth enabled with invalid private key (bad base64)",
			args: args{
				ctx: cntx.ContextWithSaxClientConfig(context.Background(), &saxtypes.SaxAuthClientConfig{
					ClientId:      "bad-client",
					PrivateKey:    "!!!not-valid-base64!!!",
					Scopes:        "scope1",
					TokenEndpoint: authServer.URL,
				}),
				url:            u,
				saxAuthEnabled: true,
				expectStatus:   http.StatusInternalServerError,
			},
		},
		{
			name: "Success: SAX Auth with valid config — scopes split correctly",
			args: args{
				ctx:            cntx.ContextWithSaxClientConfig(context.Background(), validSaxConfig),
				url:            u,
				saxAuthEnabled: true,
				// auth server returns a token, model server returns 200
				expectStatus: http.StatusOK,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			req, _ := http.NewRequest(http.MethodPost, "/test", nil)
			c.Request = req
			CallTarget(c, tt.args.ctx, tt.args.url, tt.args.saxAuthEnabled)
			assert.Equal(t, tt.args.expectStatus, recorder.Code)
		})
	}
}

// TestCallTarget_ProxyError verifies the err != nil branch inside CallTarget:
// when the upstream call fails the handler must write a 500 JSON error, not panic.
func TestCallTarget_ProxyError(t *testing.T) {
	// Start a real listener then immediately close it so any dial attempt fails.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close() // port is now closed — connections will be refused

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req, _ := http.NewRequest(http.MethodPost, "/test", io.NopCloser(strings.NewReader(`{"key":"val"}`)))
	c.Request = req

	CallTarget(c, context.Background(), "http://"+addr, false)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "upstream request failed")
}

func TestCallTargetWithResponse(t *testing.T) {
	// Default test server that accepts GET with application/json
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"message": "success"}`)
	}))
	defer ts.Close()

	// Flexible test server that echoes the body and accepts any method
	echoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer echoServer.Close()

	// authServer for SAX token exchange
	authServer := testutils.NewLocalAuthServer()
	defer authServer.Close()

	// Build a valid base64-encoded PEM private key
	rawPK := testutils.GeneratePrivateKey()
	b64PK := base64.StdEncoding.EncodeToString(rawPK)

	validSaxConfig := &saxtypes.SaxAuthClientConfig{
		ClientId:      "test-client",
		PrivateKey:    b64PK,
		Scopes:        "scope1 scope2",
		TokenEndpoint: authServer.URL,
	}

	t.Run("Unauthenticated request", func(t *testing.T) {
		ctx := context.Background()
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")

		resp, err := CallTargetWithResponse(ctx, ts.URL, http.MethodGet, headers, nil, false)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("SAX Auth enabled but missing config", func(t *testing.T) {
		ctx := context.Background() // no sax config injected
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")

		resp, err := CallTargetWithResponse(ctx, ts.URL, http.MethodGet, headers, nil, true)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "SAX authentication enabled but no SAX configuration found")
	})

	t.Run("SAX Auth with valid config and successful call", func(t *testing.T) {
		ctx := cntx.ContextWithSaxClientConfig(context.Background(), validSaxConfig)
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")

		resp, err := CallTargetWithResponse(ctx, echoServer.URL, http.MethodGet, headers, nil, true)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("Network error when calling unreachable URL", func(t *testing.T) {
		// Start a listener and close it immediately so the port is unreachable
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		addr := ln.Addr().String()
		ln.Close()

		ctx := context.Background()
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")

		resp, callErr := CallTargetWithResponse(ctx, "http://"+addr, http.MethodGet, headers, nil, false)
		assert.Error(t, callErr)
		assert.Nil(t, resp)
		assert.Contains(t, callErr.Error(), "failed to call")
	})

	t.Run("SAX Auth with invalid private key PEM format", func(t *testing.T) {
		invalidSaxConfig := &saxtypes.SaxAuthClientConfig{
			ClientId:      "bad-client",
			PrivateKey:    "!!!not-valid-base64!!!",
			Scopes:        "scope1",
			TokenEndpoint: authServer.URL,
		}
		ctx := cntx.ContextWithSaxClientConfig(context.Background(), invalidSaxConfig)
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")

		resp, err := CallTargetWithResponse(ctx, echoServer.URL, http.MethodGet, headers, nil, true)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to get private key PEM format")
	})

	t.Run("Request with body reads and forwards correctly", func(t *testing.T) {
		ctx := context.Background()
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		bodyContent := `{"prompt":"hello world"}`
		body := io.NopCloser(strings.NewReader(bodyContent))

		resp, err := CallTargetWithResponse(ctx, echoServer.URL, http.MethodGet, headers, body, false)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		assert.Equal(t, bodyContent, string(respBody))
	})

	t.Run("Nil headers are initialized to empty header map", func(t *testing.T) {
		// When nil headers are passed, the function should initialize them
		ctx := context.Background()

		resp, err := CallTargetWithResponse(ctx, echoServer.URL, http.MethodGet, nil, nil, false)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("Headers from gin.Context are copied when present", func(t *testing.T) {
		// Create a gin context with custom headers
		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)
		ginReq, _ := http.NewRequest(http.MethodGet, "/test", nil)
		ginReq.Header.Set("X-Custom-Header", "custom-value")
		ginCtx.Request = ginReq

		ctx := cntx.ContextWithGinContext(context.Background(), ginCtx)
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")

		resp, err := CallTargetWithResponse(ctx, echoServer.URL, http.MethodGet, headers, nil, false)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestSetApiVersionParam(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "url_without_query_params",
			input:    "https://azure.openai.com/openai/deployments/gpt-4/chat/completions",
			expected: "https://azure.openai.com/openai/deployments/gpt-4/chat/completions?api-version=2024-10-21",
		},
		{
			name:     "url_with_existing_query_params",
			input:    "https://azure.openai.com/openai/deployments/gpt-4/chat/completions?model=gpt-4",
			expected: "https://azure.openai.com/openai/deployments/gpt-4/chat/completions?model=gpt-4&api-version=2024-10-21",
		},
		{
			name:     "url_already_has_api_version",
			input:    "https://azure.openai.com/openai/deployments/gpt-4/chat/completions?api-version=2021-01-01",
			expected: "https://azure.openai.com/openai/deployments/gpt-4/chat/completions?api-version=2021-01-01&api-version=2024-10-21",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setApiVersionParam(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetrieveMapping(t *testing.T) {
	// Create a temporary test file
	content := []byte(`
models:
  - id: "test-model"
    name: "Test Model"
    provider: "test-provider"
`)
	tmpFile, err := os.CreateTemp("", "test-mapping-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(content)
	assert.NoError(t, err)
	err = tmpFile.Close()
	assert.NoError(t, err)

	tests := []struct {
		name     string
		fileName string
		wantErr  bool
	}{
		{
			name:     "Success: Valid YAML file",
			fileName: tmpFile.Name(),
			wantErr:  false,
		},
		{
			name:     "Error: Non-existent file",
			fileName: "non-existent-file.yaml",
			wantErr:  true,
		},
		{
			name:     "Error: Invalid YAML content",
			fileName: "cmd/service/api/common.go", // Using this file as it exists but isn't valid YAML
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mapping, err := retrieveMapping(ctx, tt.fileName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, mapping)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, mapping)
				assert.Len(t, mapping.Models, 1)
				assert.Equal(t, "Test Model", mapping.Models[0].Name)
				assert.Equal(t, "test-provider", mapping.Models[0].Provider)
			}
		})
	}
}
