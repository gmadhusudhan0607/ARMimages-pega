/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Runtime Configuration via Headers
 * --------------------------------------------
 * Purpose: Parses ALL runtime configuration from request headers and stores it in the request context.
 * Usage: Add RuntimeConfigMiddleware to your Gin middleware chain to enable header-based configuration.
 * Configuration: Controlled by ENABLE_RUNTIME_HEADER_CONFIG environment variable.
 */

package middleware

import (
	"fmt"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var runtimeConfigLogger = log.GetNamedLogger("runtime-config-middleware")

// RuntimeConfigMiddleware parses ALL known runtime configuration from request headers
// and stores it in the request context for use by other middleware and handlers.
func RuntimeConfigMiddleware(c *gin.Context) {
	// Always create a RuntimeConfig object, even if runtime headers are disabled
	// This ensures consistent behavior across all middleware
	runtimeConfig := config.NewRuntimeConfig()

	// Only parse headers if runtime configuration via headers is enabled
	if helpers.IsRuntimeConfigurationViaHeadersEnabled() {
		configChanged := parseAllConfigurationHeaders(c, runtimeConfig)

		if configChanged {
			logConfigChanges(c, runtimeConfig)
		}
	}

	// Always store RuntimeConfig in context for consistent access pattern
	ctx := config.WithRuntimeConfig(c.Request.Context(), runtimeConfig)
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

// parseAllConfigurationHeaders parses all known configuration headers and updates the RuntimeConfig
func parseAllConfigurationHeaders(c *gin.Context, runtimeConfig *config.RuntimeConfig) bool {
	configChanged := false

	// Parse ForceFreshDbMetrics header
	if parseForceFreshDbMetricsHeader(c, runtimeConfig) {
		configChanged = true
	}

	// Parse ServiceMode header
	if parseServiceModeHeader(c, runtimeConfig) {
		configChanged = true
	}

	// Future configuration headers can be added here
	// Example:
	// if parseNewConfigHeader(c, runtimeConfig) {
	//     configChanged = true
	// }

	return configChanged
}

// parseForceFreshDbMetricsHeader parses and applies the ForceFreshDbMetrics header
func parseForceFreshDbMetricsHeader(c *gin.Context, runtimeConfig *config.RuntimeConfig) bool {
	forceFreshDbMetricsHeader := c.GetHeader(headers.ForceFreshDbMetrics)
	if forceFreshDbMetricsHeader == "" {
		return false
	}
	if err := validateConfigurationHeader(headers.ForceFreshDbMetrics, forceFreshDbMetricsHeader); err != nil {
		runtimeConfigLogger.Warn("Invalid force fresh metrics header",
			zap.String("value", forceFreshDbMetricsHeader),
			zap.Error(err))
		return false
	} else if strings.ToLower(strings.TrimSpace(forceFreshDbMetricsHeader)) == "true" {
		runtimeConfig.ForceFreshDbMetrics = true
		runtimeConfigLogger.Info("[RuntimeConfig] Applied custom configuration",
			zap.String("header", headers.ForceFreshDbMetrics),
			zap.String("value", forceFreshDbMetricsHeader),
		)
		return true
	}
	return false
}

// parseServiceModeHeader parses and applies the ServiceMode header
func parseServiceModeHeader(c *gin.Context, runtimeConfig *config.RuntimeConfig) bool {
	serviceModeHeader := c.GetHeader(headers.ServiceMode)
	if serviceModeHeader == "" {
		return false
	}
	if err := validateConfigurationHeader(headers.ServiceMode, serviceModeHeader); err != nil {
		runtimeConfigLogger.Warn("Invalid service mode header",
			zap.String("value", serviceModeHeader),
			zap.Error(err))
		return false
	}
	serviceMode := config.ParseServiceMode(serviceModeHeader)
	if serviceMode != config.ServiceModeNormal {
		runtimeConfig.ServiceMode = serviceMode
		runtimeConfigLogger.Info("[RuntimeConfig] Applied custom configuration",
			zap.String("header", headers.ServiceMode),
			zap.String("value", serviceModeHeader),
		)
		return true
	} else if strings.ToUpper(strings.TrimSpace(serviceModeHeader)) != "NORMAL" {
		// Log warning only if the header value was not "NORMAL"
		runtimeConfigLogger.Warn("Invalid service mode header value, using normal mode",
			zap.String("value", serviceModeHeader))
	}
	return false
}

// logConfigChanges logs configuration changes for audit purposes
func logConfigChanges(c *gin.Context, runtimeConfig *config.RuntimeConfig) {
	runtimeConfigLogger.Info("Runtime configuration applied via headers",
		zap.Bool("force_fresh_db_metrics", runtimeConfig.ForceFreshDbMetrics),
		zap.String("service_mode", runtimeConfig.ServiceMode.String()),
		zap.String("client_ip", c.ClientIP()),
		zap.String("user_agent", c.GetHeader("User-Agent")))
}

// validateConfigurationHeader validates configuration header names and values
func validateConfigurationHeader(name, value string) error {
	if !helpers.IsValidHeaderName(name) {
		return fmt.Errorf("invalid header name: %s", name)
	}

	sanitizedValue := helpers.SanitizeHeaderValue(value)
	if sanitizedValue != value {
		return fmt.Errorf("header value contains invalid characters")
	}

	return nil
}
