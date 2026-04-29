/*
 * Copyright (c) 2024 Pegasystems Inc.
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
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func debugContext() (context.Context, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	return cntx.ContextWithLogger(context.Background(), logger), logs
}

func infoContext() (context.Context, *observer.ObservedLogs) {
	core, logs := observer.New(zap.InfoLevel)
	logger := zap.New(core)
	return cntx.ContextWithLogger(context.Background(), logger), logs
}

// --- parseIntEnv ---

func TestParseIntEnv_NotSet(t *testing.T) {
	assert.Equal(t, 42, parseIntEnv("TEST_PARSE_INT_ENV_NOT_SET_EVER", 42))
}

func TestParseIntEnv_ValidInt(t *testing.T) {
	t.Setenv("TEST_PARSE_INT_VALID", "100")
	assert.Equal(t, 100, parseIntEnv("TEST_PARSE_INT_VALID", 0))
}

func TestParseIntEnv_InvalidString(t *testing.T) {
	t.Setenv("TEST_PARSE_INT_INVALID", "not-a-number")
	assert.Equal(t, 7, parseIntEnv("TEST_PARSE_INT_INVALID", 7))
}

// --- isJSON ---

func TestIsJSON_ValidObject(t *testing.T) {
	assert.True(t, isJSON(`{"key":"value"}`))
}

func TestIsJSON_ValidArray(t *testing.T) {
	assert.True(t, isJSON(`[1,2,3]`))
}

func TestIsJSON_Invalid(t *testing.T) {
	assert.False(t, isJSON(`not json`))
}

func TestIsJSON_Empty(t *testing.T) {
	assert.False(t, isJSON(""))
}

// --- GetCopyOfBodyBytes ---

func TestGetCopyOfBodyBytes(t *testing.T) {
	original := "hello world"
	body := io.NopCloser(strings.NewReader(original))
	result := GetCopyOfBodyBytes(&body)

	assert.Equal(t, original, string(result))

	// The body should still be readable after the copy.
	remaining, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, original, string(remaining))
}

// --- bodyLogWriter ---

func TestBodyLogWriter_Write(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()

	blw := &bodyLogWriter{
		body:           bytes.NewBufferString(""),
		ResponseWriter: createGinResponseWriter(recorder),
	}

	data := []byte("response data")
	n, err := blw.Write(data)

	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, "response data", blw.body.String())
	assert.Equal(t, "response data", recorder.Body.String())
}

func createGinResponseWriter(w http.ResponseWriter) gin.ResponseWriter {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	var gw gin.ResponseWriter
	engine.GET("/capture", func(c *gin.Context) {
		gw = c.Writer
	})
	req := httptest.NewRequest(http.MethodGet, "/capture", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	// Override the underlying writer so writes go to our recorder.
	return &writerOverride{ResponseWriter: gw, w: w}
}

// writerOverride delegates Write/WriteHeader to the supplied http.ResponseWriter
// while keeping the gin.ResponseWriter interface satisfied.
type writerOverride struct {
	gin.ResponseWriter
	w http.ResponseWriter
}

func (wo *writerOverride) Write(b []byte) (int, error) { return wo.w.Write(b) }
func (wo *writerOverride) WriteHeader(code int)        { wo.w.WriteHeader(code) }
func (wo *writerOverride) Header() http.Header         { return wo.w.Header() }

// --- truncateValue ---

func TestTruncateValue_ShortString(t *testing.T) {
	var v any = "short"
	truncateValue(&v, 100, 0)
	assert.Equal(t, "short", v)
}

func TestTruncateValue_LongString(t *testing.T) {
	long := strings.Repeat("a", 50)
	var v any = long
	truncateValue(&v, 10, 0)
	assert.Equal(t, "aaaaaaaaaa... [TRUNCATED: 50 bytes]", v)
}

func TestTruncateValue_MultiByteBoundary(t *testing.T) {
	// Each '€' is 3 bytes in UTF-8.
	multibyte := strings.Repeat("€", 10) // 30 bytes
	var v any = multibyte
	// Truncate at 4 bytes — sits in the middle of a 3-byte rune.
	truncateValue(&v, 4, 0)
	s := v.(string)
	// Should have backed off to 3 bytes (one complete '€').
	assert.True(t, strings.HasPrefix(s, "€"))
	assert.Contains(t, s, "TRUNCATED: 30 bytes")
}

func TestTruncateValue_Map(t *testing.T) {
	m := map[string]any{
		"key": strings.Repeat("x", 20),
	}
	var v any = m
	truncateValue(&v, 5, 0)
	result := v.(map[string]any)
	assert.Contains(t, result["key"].(string), "TRUNCATED")
}

func TestTruncateValue_Array(t *testing.T) {
	arr := []any{strings.Repeat("y", 20)}
	var v any = arr
	truncateValue(&v, 5, 0)
	result := v.([]any)
	assert.Contains(t, result[0].(string), "TRUNCATED")
}

func TestTruncateValue_MaxDepth(t *testing.T) {
	long := strings.Repeat("z", 50)
	var v any = long
	truncateValue(&v, 5, maxTruncateDepth+1)
	// Should return unchanged because depth exceeded.
	assert.Equal(t, long, v)
}

func TestTruncateValue_NonStringPrimitive(t *testing.T) {
	var v any = 42.0
	truncateValue(&v, 5, 0)
	assert.Equal(t, 42.0, v)
}

func TestTruncateValue_Bool(t *testing.T) {
	var v any = true
	truncateValue(&v, 5, 0)
	assert.Equal(t, true, v)
}

func TestTruncateValue_Nil(t *testing.T) {
	var v any
	truncateValue(&v, 5, 0)
	assert.Nil(t, v)
}

// --- truncateBody ---

func TestTruncateBody_ValidJSON(t *testing.T) {
	body := fmt.Sprintf(`{"msg":"%s"}`, strings.Repeat("a", 50))
	result := truncateBody(body, 10)
	assert.Contains(t, result, "TRUNCATED")
}

// --- maybeTruncate ---

func TestMaybeTruncate_TruncationDisabled(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 0
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	body := fmt.Sprintf(`{"data":"%s"}`, strings.Repeat("x", 100))
	assert.Equal(t, body, maybeTruncate(body))
}

func TestMaybeTruncate_TruncationEnabledJSON(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 5
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	body := fmt.Sprintf(`{"data":"%s"}`, strings.Repeat("x", 100))
	assert.Contains(t, maybeTruncate(body), "TRUNCATED")
}

func TestMaybeTruncate_TruncationEnabledNonJSON(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 5
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	body := "plain text"
	assert.Equal(t, body, maybeTruncate(body))
}

func TestTruncateBody_InvalidJSON(t *testing.T) {
	body := "not json at all"
	assert.Equal(t, body, truncateBody(body, 10))
}

func TestTruncateBody_ShortValues(t *testing.T) {
	body := `{"msg":"hi"}`
	result := truncateBody(body, 100)
	// No truncation, but re-serialized with indentation.
	var parsed any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	m := parsed.(map[string]any)
	assert.Equal(t, "hi", m["msg"])
}

// --- DoLogRequest ---

func TestDoLogRequest_JSONBody(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 0
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()
	logger := cntx.LoggerFromContext(ctx).Sugar()

	body := `{"key":"value"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	DoLogRequest(logger, req, "Test ")

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "Test Request:")
	assert.Contains(t, entries[0].Message, body)
}

func TestDoLogRequest_NonJSONBody(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 0
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()
	logger := cntx.LoggerFromContext(ctx).Sugar()

	req := httptest.NewRequest(http.MethodGet, "/test", strings.NewReader("plain text"))
	DoLogRequest(logger, req, "Prefix ")

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "plain text")
}

func TestDoLogRequest_WithTruncation(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 5
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()
	logger := cntx.LoggerFromContext(ctx).Sugar()

	body := fmt.Sprintf(`{"data":"%s"}`, strings.Repeat("x", 100))
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	DoLogRequest(logger, req, "")

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "TRUNCATED")
}

func TestDoLogRequest_NonJSONWithTruncationEnabled(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 5
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()
	logger := cntx.LoggerFromContext(ctx).Sugar()

	req := httptest.NewRequest(http.MethodGet, "/test", strings.NewReader("plain text"))
	DoLogRequest(logger, req, "")

	entries := logs.All()
	require.NotEmpty(t, entries)
	// Non-JSON body should not be truncated even when truncation is enabled.
	assert.Contains(t, entries[0].Message, "plain text")
	assert.NotContains(t, entries[0].Message, "TRUNCATED")
}

// --- DoLogResponse ---

func TestDoLogResponse_JSONBody(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 0
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()
	logger := cntx.LoggerFromContext(ctx).Sugar()

	body := `{"result":"ok"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	DoLogResponse(logger, resp, "Test ")

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "Test Response:")
	assert.Contains(t, entries[0].Message, body)
}

func TestDoLogResponse_NonJSONBody(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 0
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()
	logger := cntx.LoggerFromContext(ctx).Sugar()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("plain response")),
	}
	DoLogResponse(logger, resp, "")

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "plain response")
}

func TestDoLogResponse_WithTruncation(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 5
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()
	logger := cntx.LoggerFromContext(ctx).Sugar()

	body := fmt.Sprintf(`{"data":"%s"}`, strings.Repeat("z", 100))
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	DoLogResponse(logger, resp, "")

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "TRUNCATED")
}

func TestDoLogResponse_NonJSONWithTruncationEnabled(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 5
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()
	logger := cntx.LoggerFromContext(ctx).Sugar()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("text body")),
	}
	DoLogResponse(logger, resp, "")

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "text body")
	assert.NotContains(t, entries[0].Message, "TRUNCATED")
}

// --- RequestLoggerMiddleware ---

func TestRequestLoggerMiddleware(t *testing.T) {
	ctx, logs := debugContext()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RequestLoggerMiddleware(ctx))
	engine.POST("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	body := `{"hello":"world"}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "Received Request:")
}

// --- ResponseLoggerMiddleware ---

func TestResponseLoggerMiddleware_DebugEnabled(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 0
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ResponseLoggerMiddleware(ctx))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "Returning Response:")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestResponseLoggerMiddleware_DebugDisabled(t *testing.T) {
	ctx, logs := infoContext()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ResponseLoggerMiddleware(ctx))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// No debug logs should be emitted.
	assert.Empty(t, logs.All())
	// But the handler should still execute.
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestResponseLoggerMiddleware_WithTruncation(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 5
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ResponseLoggerMiddleware(ctx))
	engine.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": strings.Repeat("a", 100)})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "TRUNCATED")
}

func TestResponseLoggerMiddleware_NonJSONBodyTruncationEnabled(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 5
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ResponseLoggerMiddleware(ctx))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "not json but truncation is on")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "not json but truncation is on")
	assert.NotContains(t, entries[0].Message, "TRUNCATED")
}

func TestResponseLoggerMiddleware_NonJSONBody(t *testing.T) {
	origTruncate := logTruncateMaxLen
	logTruncateMaxLen = 0
	t.Cleanup(func() { logTruncateMaxLen = origTruncate })

	ctx, logs := debugContext()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ResponseLoggerMiddleware(ctx))
	engine.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "plain text response")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	entries := logs.All()
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Message, "plain text response")
}

// --- truncateValue nested structures ---

func TestTruncateValue_NestedMapAndArray(t *testing.T) {
	var v any = map[string]any{
		"arr": []any{
			map[string]any{
				"deep": strings.Repeat("d", 30),
			},
		},
	}
	truncateValue(&v, 5, 0)
	m := v.(map[string]any)
	arr := m["arr"].([]any)
	inner := arr[0].(map[string]any)
	assert.Contains(t, inner["deep"].(string), "TRUNCATED")
}

// --- truncateBody edge cases ---

func TestTruncateBody_Array(t *testing.T) {
	body := fmt.Sprintf(`["%s"]`, strings.Repeat("b", 50))
	result := truncateBody(body, 5)
	assert.Contains(t, result, "TRUNCATED")
}

// --- verify debug log level gating via zapcore ---

func TestResponseLoggerMiddleware_LogLevelGating(t *testing.T) {
	// Create a logger that only accepts Warn and above.
	core, logs := observer.New(zapcore.WarnLevel)
	logger := zap.New(core)
	ctx := cntx.ContextWithLogger(context.Background(), logger)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ResponseLoggerMiddleware(ctx))

	called := false
	engine.GET("/test", func(c *gin.Context) {
		called = true
		c.String(http.StatusOK, "warn-level-test")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Empty(t, logs.All())
	assert.Equal(t, "warn-level-test", rec.Body.String())
}
