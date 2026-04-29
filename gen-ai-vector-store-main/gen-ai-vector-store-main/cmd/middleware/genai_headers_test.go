/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		// Simulate service metrics in context
		ctx := servicemetrics.WithMetrics(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	engine.Use(GenaiResponseHeadersMiddleware)
	engine.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Header(), headers.RequestDurationMs)
}

func TestProcessingOverheadHeaders_WithEmbedding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		ctx := servicemetrics.WithMetrics(c.Request.Context())
		svcMetrics := servicemetrics.FromContext(ctx)
		svcMetrics.RequestMetrics.StartProcessing()

		m := svcMetrics.EmbeddingMetrics.NewMeasurement("test-model", "v1")
		m.Start()
		m.Stop()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	engine.Use(GenaiResponseHeadersMiddleware)
	engine.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 200, recorder.Code)
	for _, h := range []string{headers.ProcessingDurationMs, headers.OverheadMs, headers.EmbeddingNetOverheadMs} {
		assert.NotEmpty(t, recorder.Header().Get(h), "header %s should be set", h)
		val, err := strconv.Atoi(recorder.Header().Get(h))
		assert.NoError(t, err, "header %s should be a valid integer", h)
		assert.GreaterOrEqual(t, val, 0, "header %s should be >= 0", h)
	}
}

func TestProcessingOverheadHeaders_WithoutEmbedding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		ctx := servicemetrics.WithMetrics(c.Request.Context())
		svcMetrics := servicemetrics.FromContext(ctx)
		svcMetrics.RequestMetrics.StartProcessing()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	engine.Use(GenaiResponseHeadersMiddleware)
	engine.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "0", recorder.Header().Get(headers.EmbeddingNetOverheadMs),
		"EmbeddingNetOverheadMs should be 0 when no embedding metrics present")

	for _, h := range []string{headers.ProcessingDurationMs, headers.OverheadMs} {
		assert.NotEmpty(t, recorder.Header().Get(h), "header %s should be set", h)
		val, err := strconv.Atoi(recorder.Header().Get(h))
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, val, 0)
	}
}

func TestProcessingOverheadHeaders_ClampToZero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		ctx := servicemetrics.WithMetrics(c.Request.Context())
		svcMetrics := servicemetrics.FromContext(ctx)
		// Do NOT start/stop request (Duration=0), but record DB metric.
		// This can cause requestMs < dbMs, testing the clamp-to-zero logic.
		dbM := svcMetrics.DbMetrics.NewMeasurement()
		dbM.Start()
		dbM.Stop()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	engine.Use(GenaiResponseHeadersMiddleware)
	engine.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest("GET", "/test", nil))

	assert.Equal(t, 200, recorder.Code)
	for _, h := range []string{headers.ProcessingDurationMs, headers.OverheadMs, headers.EmbeddingNetOverheadMs} {
		val, err := strconv.Atoi(recorder.Header().Get(h))
		assert.NoError(t, err, "header %s should be a valid integer", h)
		assert.GreaterOrEqual(t, val, 0, "header %s should never be negative (clamped to 0)", h)
	}
}
