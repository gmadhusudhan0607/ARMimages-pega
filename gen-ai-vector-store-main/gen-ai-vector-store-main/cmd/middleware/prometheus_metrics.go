/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Prometheus Metrics Collection
 * ----------------------------------------
 * Purpose: Collects and exposes Prometheus metrics for HTTP requests, durations, active connections, DB query durations, and SAX JWT cache metrics.
 */

package middleware

import (
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/httpmetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/saxmetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var prometheusLogger = log.GetNamedLogger("middleware.prometheus")

func PrometheusGinMiddleware(config ...httpmetrics.Config) gin.HandlerFunc {
	cfg := httpmetrics.DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	prometheusCollector := httpmetrics.NewPrometheusCollector(cfg)

	// Initialize SAX metrics collector if caching is enabled
	var saxCollector *saxmetrics.Collector

	// Only register metrics if enabled to avoid duplicate registration in tests
	if cfg.Enabled {
		if err := prometheusCollector.Register(); err != nil {
			// Log error but continue - metrics collection will be disabled
			// This prevents the middleware from failing if metrics registration fails
			prometheusLogger.Error("Failed to register Prometheus metrics collector", zap.Error(err))
		}

		if helpers.IsSaxTokenCacheEnabled() {
			saxCollectorEnabled := true
			saxPrometheusCollector := saxmetrics.NewPrometheusCollector()
			saxPrometheusCollector.Register()
			saxCollector = saxmetrics.NewCollector(saxPrometheusCollector, saxCollectorEnabled)
			prometheusLogger.Info("SAX metrics collection enabled")
		}
	}

	collector := httpmetrics.NewCollector(cfg, prometheusCollector)

	return NewGinMiddleware(collector, saxCollector)
}

func NewGinMiddleware(collector *httpmetrics.Collector, saxCollector *saxmetrics.Collector) gin.HandlerFunc {
	if collector == nil {
		// Return a no-op middleware if collector is nil to prevent panics
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// Skip metrics collection if disabled
		if !collector.IsEnabled() {
			c.Next()
			return
		}

		// Get normalized path (set by PathNormalizationMiddleware)
		// Fall back to raw URL path if normalization wasn't applied
		path, exists := c.Get("normalizedPath")
		if !exists {
			path = c.Request.URL.Path
		}

		// Ensure path is a string (defensive programming)
		pathStr, ok := path.(string)
		if !ok {
			pathStr = c.Request.URL.Path
		}

		// Create metrics instance for this request
		metrics := httpmetrics.NewHTTPMetrics(pathStr, c.Request.Method)

		// Record active connection increment
		collector.RecordActiveConnection(metrics.Path, 1)

		// Ensure active connection is decremented even if panic occurs
		defer collector.RecordActiveConnection(metrics.Path, -1)

		// Process the request
		c.Next()

		// Complete metrics collection with final status
		metrics.Finish(c.Writer.Status())

		// Record the completed request metrics
		collector.RecordRequest(metrics)

		if svcMetrics := servicemetrics.FromContext(c.Request.Context()); svcMetrics != nil {
			// Record DB query duration if available from service context
			if dbTime := svcMetrics.DbMetrics.QueryExecutionTime(); dbTime != 0 {
				collector.RecordDBQueryDuration(metrics.Path, metrics.Method, metrics.StatusCode, dbTime)
			}

			// Record returned items
			collector.RecordReturnedItems(metrics.Path, metrics.Method, metrics.StatusCode, svcMetrics.ResponseMetrics.ItemsReturned())
		}

		// SAX metrics: Collect SAX JWT validation metrics if available
		if saxCollector != nil && saxCollector.IsEnabled() {
			if cacheHitVal, exists := c.Get("sax_cache_hit"); exists {
				cacheHit, ok := cacheHitVal.(bool)
				if !ok {
					return
				}

				prometheusLogger.Info("SAX cache metric recorded", zap.Bool("cache_hit", cacheHit))

				// Record cache hit or miss
				if cacheHit {
					saxCollector.RecordCacheHit()
				} else {
					saxCollector.RecordCacheMiss()
				}

				// Record cache size if available
				if cacheSizeVal, exists := c.Get("sax_cache_size"); exists {
					if cacheSize, ok := cacheSizeVal.(int); ok {
						prometheusLogger.Info("SAX cache size recorded", zap.Int("cache_size", cacheSize))
						saxCollector.RecordCacheSize(cacheSize)
					}
				}
			}
		}
	}
}
