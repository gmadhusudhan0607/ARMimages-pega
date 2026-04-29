/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/httpmetrics"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewPrometheusGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with default configuration
	middleware := PrometheusGinMiddleware()
	assert.NotNil(t, middleware)

	engine := gin.New()
	engine.Use(PathNormalizationMiddleware)
	engine.Use(middleware)

	engine.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "ok")
}

func TestNewPrometheusGinMiddleware_WithCustomConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with custom configuration - use disabled to avoid metric registration conflicts
	config := httpmetrics.Config{
		Enabled:          false, // Disabled to avoid duplicate registration in tests
		IncludeDBMetrics: false,
		RequestBuckets:   []float64{0.1, 0.5, 1.0},
		DBQueryBuckets:   []float64{0.1, 0.5, 1.0},
	}

	middleware := PrometheusGinMiddleware(config)
	assert.NotNil(t, middleware)

	engine := gin.New()
	engine.Use(PathNormalizationMiddleware)
	engine.Use(middleware)

	engine.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "ok")
}

func TestNewPrometheusGinMiddleware_WithNormalizedPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Use disabled config to avoid duplicate registration
	config := httpmetrics.DefaultConfig()
	config.Enabled = false

	engine := gin.New()
	engine.Use(PathNormalizationMiddleware)
	engine.Use(PrometheusGinMiddleware(config))

	engine.GET("/v1/:isolationID/collections/:collectionName/documents", func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "not found"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/iso-test123/collections/col-test123/documents", nil)
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, 404, recorder.Code)
	// The middleware should process the request without errors
}

func TestNewPrometheusGinMiddleware_DisabledConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with disabled configuration
	config := httpmetrics.Config{
		Enabled:          false,
		IncludeDBMetrics: false,
	}

	middleware := PrometheusGinMiddleware(config)

	engine := gin.New()
	engine.Use(middleware)

	engine.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "ok")
}

func TestNewGinMiddleware_NilCollector(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with nil collector
	middleware := NewGinMiddleware(nil, nil)
	assert.NotNil(t, middleware)

	engine := gin.New()
	engine.Use(middleware)

	engine.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "ok")
}

func TestNewGinMiddleware_WithValidCollector(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with valid collector - use disabled config to avoid registration conflicts
	config := httpmetrics.DefaultConfig()
	config.Enabled = false // Disabled to avoid duplicate registration in tests
	prometheusCollector := httpmetrics.NewPrometheusCollector(config)
	collector := httpmetrics.NewCollector(config, prometheusCollector)

	middleware := NewGinMiddleware(collector, nil)
	assert.NotNil(t, middleware)

	engine := gin.New()
	engine.Use(PathNormalizationMiddleware)
	engine.Use(middleware)

	engine.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "ok")
}
