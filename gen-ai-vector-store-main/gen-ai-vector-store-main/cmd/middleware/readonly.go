/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Read-Only Mode Protection
 * ------------------------------------
 * Purpose: Centralized read-only mode enforcement that blocks write operations when READ_ONLY_MODE=true
 * Usage: Add ReadOnlyMiddleware to your Gin middleware chain to automatically protect write endpoints
 * Configuration: Uses environment variable READ_ONLY_MODE=true to enable read-only restrictions
 */

package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ReadOnlyConfig defines the configuration for read-only middleware
type ReadOnlyConfig struct {
	AllowedEndpoints map[string][]string // method -> []paths
}

// DefaultReadOnlyConfig returns the default configuration with all read-only allowed endpoints
// Everything that is not defined in this list must be considered as blocked
var DefaultReadOnlyConfig = ReadOnlyConfig{
	AllowedEndpoints: map[string][]string{
		"GET": {
			// Health and System Endpoints
			"/health/liveness",
			"/health/readiness",
			"/metrics",
			"/",
			"/swagger/*",
			"/debug/*",
			// Service API Read Operations - only these specific GET endpoints are allowed
			"/v1/:isolationID/collections/:collectionName/documents/:documentID",
			"/v1/:isolationID/smart-attributes-group/",
			"/v1/:isolationID/smart-attributes-group/:groupID",
			"/v2/:isolationID/collections",
			"/v2/:isolationID/collections/:collectionID",
			"/v2/:isolationID/collections/:collectionID/documents/:documentID/chunks",
			// Ops API Read Operations
			"/v1/isolations/:isolationID",
			"/v1/isolationsRO/:isolationID",
			"/v1/db/configuration",
			"/v1/db/size",
		},
		"POST": {
			// Service API - POST operations used for complex queries/filtering (read operations only)
			"/v1/:isolationID/collections/:collectionName/documents",
			"/v1/:isolationID/collections/:collectionName/query/chunks",
			"/v1/:isolationID/collections/:collectionName/query/documents",
			"/v1/:isolationID/collections/:collectionName/attributes",
			"/v2/:isolationID/collections/:collectionID/find-documents",
			// Ops API - Read-only isolation endpoints and operations
			"/v1/isolationsRO",
			"/v1/ops/:isolationID/documents",
			"/v1/ops/:isolationID/documentsDetails",
		},
		// NOTE: PUT and DELETE methods are intentionally restricted
		// Only specific read-only operations are allowed, all write operations are blocked
		"PUT": {
			// Ops API - Read-only isolation endpoints only
			"/v1/isolationsRO/:isolationID",
		},
		"DELETE": {
			// Ops API - Read-only isolation endpoints only
			"/v1/isolationsRO/:isolationID",
		},
	},
}

var readonlyLogger = log.GetNamedLogger("readonly-middleware")

// isReadOnlyModeActive determines if read-only mode is active based on environment variables and runtime configuration
func isReadOnlyModeActive(c *gin.Context) bool {
	// 1. First check READ_ONLY_MODE environment variable
	if os.Getenv("READ_ONLY_MODE") == "true" {
		readonlyLogger.Debug("Read-only mode activated from environment variable READ_ONLY_MODE=true")
		return true
	}

	// 2. Check runtime configuration from context (set by RuntimeConfigMiddleware)
	runtimeConfig := config.GetRuntimeConfigFromContext(c.Request.Context())
	if runtimeConfig != nil && runtimeConfig.ServiceMode == config.ServiceModeReadOnly {
		readonlyLogger.Debug("Read-only mode activated from runtime configuration",
			zap.String("service_mode", runtimeConfig.ServiceMode.String()),
			zap.String("client_ip", c.ClientIP()),
		)
		return true
	}

	return false
}

// ReadOnlyMiddleware creates a middleware that enforces read-only mode restrictions
func ReadOnlyMiddleware(config ReadOnlyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If read-only mode is not active, allow all requests
		if !isReadOnlyModeActive(c) {
			c.Next()
			return
		}

		method := c.Request.Method
		path := c.FullPath()

		// Check if this endpoint is allowed in read-only mode
		if isEndpointAllowed(method, path, config.AllowedEndpoints) {
			c.Next()
			return
		}

		// Block the request with 405 Method Not Allowed
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"code":    "405",
			"message": "Method not allowed in Read Only mode",
		})
		c.Abort()
	}
}

// NewReadOnlyMiddleware creates middleware with the default configuration
func NewReadOnlyMiddleware() gin.HandlerFunc {
	return ReadOnlyMiddleware(DefaultReadOnlyConfig)
}

// isEndpointAllowed checks if the given method and path combination is allowed in read-only mode
func isEndpointAllowed(method, path string, allowedEndpoints map[string][]string) bool {
	allowedPaths, exists := allowedEndpoints[method]
	if !exists {
		return false
	}

	for _, allowedPath := range allowedPaths {
		if pathMatches(path, allowedPath) {
			return true
		}
	}

	return false
}

// pathMatches checks if a request path matches an allowed path pattern
// Supports Gin route patterns with parameters (e.g., :isolationID) and wildcards (*)
func pathMatches(requestPath, allowedPattern string) bool {
	// Handle exact matches first
	if requestPath == allowedPattern {
		return true
	}

	// Disallow trailing slash mismatch (except for wildcard patterns)
	if strings.TrimRight(requestPath, "/") == allowedPattern && strings.HasSuffix(requestPath, "/") && !strings.HasSuffix(allowedPattern, "/") {
		return false
	}

	// Handle wildcard patterns (e.g., /swagger/*)
	if strings.HasSuffix(allowedPattern, "/*") {
		prefix := strings.TrimSuffix(allowedPattern, "/*")
		return strings.HasPrefix(requestPath, prefix)
	}

	// Handle Gin parameter patterns (e.g., /v1/:isolationID/collections/:collectionName)
	return ginPathMatches(requestPath, allowedPattern)
}

// ginPathMatches checks if a request path matches a Gin route pattern with parameters
func ginPathMatches(requestPath, pattern string) bool {
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")

	// If different number of parts, they don't match
	if len(requestParts) != len(patternParts) {
		return false
	}

	// Check each part
	for i, patternPart := range patternParts {
		requestPart := requestParts[i]

		// If pattern part starts with :, it's a parameter - matches any value
		if strings.HasPrefix(patternPart, ":") {
			continue
		}

		// If pattern part is *, it matches any value
		if patternPart == "*" {
			continue
		}

		// Otherwise, must be exact match
		if requestPart != patternPart {
			return false
		}
	}

	return true
}
