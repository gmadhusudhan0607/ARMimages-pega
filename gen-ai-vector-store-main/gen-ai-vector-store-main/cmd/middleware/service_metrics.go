/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Service Metrics Injection
 * ------------------------------------
 * Purpose: Injects service metrics into the request context for downstream usage (e.g., metrics collection, header enrichment).
 * Usage: Add ServiceMetricsMiddleware to your Gin middleware chain to ensure service metrics are available in the request context.
 * Configuration: Uses servicemetrics.WithMetrics to wrap the context with metrics data.
 */

package middleware

import (
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/gin-gonic/gin"
)

// ServiceMetricsMiddleware injects ServiceMetrics into the request context.
// This enables downstream middleware and handlers to access and record service metrics for the request lifecycle.
func ServiceMetricsMiddleware(c *gin.Context) {
	c.Request = c.Request.WithContext(servicemetrics.WithMetrics(c.Request.Context()))

	serviceMetrics := servicemetrics.FromContext(c.Request.Context())

	serviceMetrics.RequestMetrics.StartProcessing()

	c.Next()

	if helpers.GetEnvOrDefault("LOG_SERVICE_METRICS", "true") == "true" {
		serviceMetrics.
			LogMetrics(
				log.GetLoggerFromContext(c.Request.Context()),
			)
	}
}
