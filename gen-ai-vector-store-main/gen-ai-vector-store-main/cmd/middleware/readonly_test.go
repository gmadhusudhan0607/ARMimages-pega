/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestReadOnlyMiddleware(t *testing.T) {
	// Save original environment variables
	originalReadOnlyMode := os.Getenv("READ_ONLY_MODE")
	originalRuntimeConfig := os.Getenv("ENABLE_RUNTIME_HEADER_CONFIG")
	defer func() {
		if originalReadOnlyMode == "" {
			os.Unsetenv("READ_ONLY_MODE")
		} else {
			os.Setenv("READ_ONLY_MODE", originalReadOnlyMode)
		}
		if originalRuntimeConfig == "" {
			os.Unsetenv("ENABLE_RUNTIME_HEADER_CONFIG")
		} else {
			os.Setenv("ENABLE_RUNTIME_HEADER_CONFIG", originalRuntimeConfig)
		}
	}()

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                string
		readOnlyMode        string
		runtimeConfigHeader string
		serviceModeHeader   string
		method              string
		path                string
		routePath           string
		expectedStatus      int
		expectedBody        string
	}{
		// Test when READ_ONLY_MODE is disabled
		{
			name:           "ReadOnlyMode disabled - write operation allowed",
			readOnlyMode:   "false",
			method:         "PUT",
			path:           "/v1/test-isolation/collections/test-collection/documents",
			routePath:      "/v1/:isolationID/collections/:collectionName/documents",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode disabled - read operation allowed",
			readOnlyMode:   "false",
			method:         "GET",
			path:           "/v1/test-isolation/collections/test-collection/documents/doc1",
			routePath:      "/v1/:isolationID/collections/:collectionName/documents/:documentID",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},

		// Test when READ_ONLY_MODE is enabled - allowed endpoints
		{
			name:           "ReadOnlyMode enabled - health endpoint allowed",
			readOnlyMode:   "true",
			method:         "GET",
			path:           "/health/liveness",
			routePath:      "/health/liveness",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - metrics endpoint allowed",
			readOnlyMode:   "true",
			method:         "GET",
			path:           "/metrics",
			routePath:      "/metrics",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - swagger wildcard allowed",
			readOnlyMode:   "true",
			method:         "GET",
			path:           "/swagger/service.yaml",
			routePath:      "/swagger/*filepath",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - debug wildcard allowed",
			readOnlyMode:   "true",
			method:         "GET",
			path:           "/debug/memory",
			routePath:      "/debug/*filepath",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - GET document allowed",
			readOnlyMode:   "true",
			method:         "GET",
			path:           "/v1/test-isolation/collections/test-collection/documents/doc1",
			routePath:      "/v1/:isolationID/collections/:collectionName/documents/:documentID",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - POST documents (query) allowed",
			readOnlyMode:   "true",
			method:         "POST",
			path:           "/v1/test-isolation/collections/test-collection/documents",
			routePath:      "/v1/:isolationID/collections/:collectionName/documents",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - POST query chunks allowed",
			readOnlyMode:   "true",
			method:         "POST",
			path:           "/v1/test-isolation/collections/test-collection/query/chunks",
			routePath:      "/v1/:isolationID/collections/:collectionName/query/chunks",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - GET collections V2 allowed",
			readOnlyMode:   "true",
			method:         "GET",
			path:           "/v2/test-isolation/collections",
			routePath:      "/v2/:isolationID/collections",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - GET isolation allowed",
			readOnlyMode:   "true",
			method:         "GET",
			path:           "/v1/isolations/test-isolation",
			routePath:      "/v1/isolations/:isolationID",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - POST isolationsRO allowed",
			readOnlyMode:   "true",
			method:         "POST",
			path:           "/v1/isolationsRO",
			routePath:      "/v1/isolationsRO",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - PUT isolationsRO allowed",
			readOnlyMode:   "true",
			method:         "PUT",
			path:           "/v1/isolationsRO/test-isolation",
			routePath:      "/v1/isolationsRO/:isolationID",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},
		{
			name:           "ReadOnlyMode enabled - DELETE isolationsRO allowed",
			readOnlyMode:   "true",
			method:         "DELETE",
			path:           "/v1/isolationsRO/test-isolation",
			routePath:      "/v1/isolationsRO/:isolationID",
			expectedStatus: http.StatusOK,
			expectedBody:   "success",
		},

		// Test runtime configuration with ServiceMode header
		{
			name:                "Runtime config enabled - ServiceMode READONLY blocks write operations",
			readOnlyMode:        "false",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "READONLY",
			method:              "PUT",
			path:                "/v1/test-isolation/collections/test-collection/documents",
			routePath:           "/v1/:isolationID/collections/:collectionName/documents",
			expectedStatus:      http.StatusMethodNotAllowed,
			expectedBody:        `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:                "Runtime config enabled - ServiceMode READONLY allows read operations",
			readOnlyMode:        "false",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "READONLY",
			method:              "GET",
			path:                "/v1/test-isolation/collections/test-collection/documents/doc1",
			routePath:           "/v1/:isolationID/collections/:collectionName/documents/:documentID",
			expectedStatus:      http.StatusOK,
			expectedBody:        "success",
		},
		{
			name:                "Runtime config enabled - ServiceMode NORMAL allows write operations",
			readOnlyMode:        "false",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "NORMAL",
			method:              "PUT",
			path:                "/v1/test-isolation/collections/test-collection/documents",
			routePath:           "/v1/:isolationID/collections/:collectionName/documents",
			expectedStatus:      http.StatusOK,
			expectedBody:        "success",
		},
		{
			name:                "Runtime config disabled - ServiceMode header ignored",
			readOnlyMode:        "false",
			runtimeConfigHeader: "false",
			serviceModeHeader:   "READONLY",
			method:              "PUT",
			path:                "/v1/test-isolation/collections/test-collection/documents",
			routePath:           "/v1/:isolationID/collections/:collectionName/documents",
			expectedStatus:      http.StatusOK,
			expectedBody:        "success",
		},
		{
			name:                "READ_ONLY_MODE takes precedence over ServiceMode header",
			readOnlyMode:        "true",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "NORMAL",
			method:              "PUT",
			path:                "/v1/test-isolation/collections/test-collection/documents",
			routePath:           "/v1/:isolationID/collections/:collectionName/documents",
			expectedStatus:      http.StatusMethodNotAllowed,
			expectedBody:        `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},

		// Test when READ_ONLY_MODE is enabled - blocked endpoints
		{
			name:           "ReadOnlyMode enabled - PUT document blocked",
			readOnlyMode:   "true",
			method:         "PUT",
			path:           "/v1/test-isolation/collections/test-collection/documents",
			routePath:      "/v1/:isolationID/collections/:collectionName/documents",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - DELETE document blocked",
			readOnlyMode:   "true",
			method:         "DELETE",
			path:           "/v1/test-isolation/collections/test-collection/documents/doc1",
			routePath:      "/v1/:isolationID/collections/:collectionName/documents/:documentID",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - PATCH document blocked",
			readOnlyMode:   "true",
			method:         "PATCH",
			path:           "/v1/test-isolation/collections/test-collection/documents/doc1",
			routePath:      "/v1/:isolationID/collections/:collectionName/documents/:documentID",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - POST collection V2 blocked",
			readOnlyMode:   "true",
			method:         "POST",
			path:           "/v2/test-isolation/collections",
			routePath:      "/v2/:isolationID/collections",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - DELETE collection V2 blocked",
			readOnlyMode:   "true",
			method:         "DELETE",
			path:           "/v2/test-isolation/collections/test-collection",
			routePath:      "/v2/:isolationID/collections/:collectionID",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - POST isolation blocked",
			readOnlyMode:   "true",
			method:         "POST",
			path:           "/v1/isolations",
			routePath:      "/v1/isolations",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - PUT isolation blocked",
			readOnlyMode:   "true",
			method:         "PUT",
			path:           "/v1/isolations/test-isolation",
			routePath:      "/v1/isolations/:isolationID",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - DELETE isolation blocked",
			readOnlyMode:   "true",
			method:         "DELETE",
			path:           "/v1/isolations/test-isolation",
			routePath:      "/v1/isolations/:isolationID",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - POST smart attributes group blocked",
			readOnlyMode:   "true",
			method:         "POST",
			path:           "/v1/test-isolation/smart-attributes-group/",
			routePath:      "/v1/:isolationID/smart-attributes-group/",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "ReadOnlyMode enabled - DELETE smart attributes group blocked",
			readOnlyMode:   "true",
			method:         "DELETE",
			path:           "/v1/test-isolation/smart-attributes-group/group1",
			routePath:      "/v1/:isolationID/smart-attributes-group/:groupID",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("READ_ONLY_MODE", tt.readOnlyMode)
			if tt.runtimeConfigHeader != "" {
				os.Setenv("ENABLE_RUNTIME_HEADER_CONFIG", tt.runtimeConfigHeader)
			}

			// Create router with middleware
			router := gin.New()
			// Add RuntimeConfigMiddleware first, then ReadOnlyMiddleware
			if tt.runtimeConfigHeader != "" {
				router.Use(RuntimeConfigMiddleware)
			}
			router.Use(NewReadOnlyMiddleware())

			// Add test route
			router.Handle(tt.method, tt.routePath, func(c *gin.Context) {
				c.String(http.StatusOK, "success")
			})

			// Create request
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			if tt.serviceModeHeader != "" {
				req.Header.Set(headers.ServiceMode, tt.serviceModeHeader)
			}
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Assert results
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")
			assert.Equal(t, tt.expectedBody, w.Body.String(), "Response body mismatch")
		})
	}
}

func TestPathMatches(t *testing.T) {
	tests := []struct {
		name           string
		requestPath    string
		allowedPattern string
		expectedMatch  bool
	}{
		// Exact matches
		{
			name:           "Exact match",
			requestPath:    "/health/liveness",
			allowedPattern: "/health/liveness",
			expectedMatch:  true,
		},
		{
			name:           "Exact match - no match",
			requestPath:    "/health/readiness",
			allowedPattern: "/health/liveness",
			expectedMatch:  false,
		},

		// Wildcard matches
		{
			name:           "Wildcard match - swagger",
			requestPath:    "/swagger/service.yaml",
			allowedPattern: "/swagger/*",
			expectedMatch:  true,
		},
		{
			name:           "Wildcard match - debug",
			requestPath:    "/debug/memory",
			allowedPattern: "/debug/*",
			expectedMatch:  true,
		},
		{
			name:           "Wildcard match - nested path",
			requestPath:    "/swagger/ui/index.html",
			allowedPattern: "/swagger/*",
			expectedMatch:  true,
		},
		{
			name:           "Wildcard no match",
			requestPath:    "/api/swagger/service.yaml",
			allowedPattern: "/swagger/*",
			expectedMatch:  false,
		},

		// Gin parameter matches
		{
			name:           "Gin parameter match - single param",
			requestPath:    "/v1/test-isolation/collections",
			allowedPattern: "/v1/:isolationID/collections",
			expectedMatch:  true,
		},
		{
			name:           "Gin parameter match - multiple params",
			requestPath:    "/v1/test-isolation/collections/test-collection/documents/doc1",
			allowedPattern: "/v1/:isolationID/collections/:collectionName/documents/:documentID",
			expectedMatch:  true,
		},
		{
			name:           "Gin parameter no match - different structure",
			requestPath:    "/v1/test-isolation/collections/test-collection",
			allowedPattern: "/v1/:isolationID/collections/:collectionName/documents/:documentID",
			expectedMatch:  false,
		},
		{
			name:           "Gin parameter no match - extra segment",
			requestPath:    "/v1/test-isolation/collections/test-collection/documents/doc1/extra",
			allowedPattern: "/v1/:isolationID/collections/:collectionName/documents/:documentID",
			expectedMatch:  false,
		},
		{
			name:           "Gin parameter no match - missing segment",
			requestPath:    "/v1/test-isolation/collections",
			allowedPattern: "/v1/:isolationID/collections/:collectionName",
			expectedMatch:  false,
		},

		// Edge cases
		{
			name:           "Root path",
			requestPath:    "/",
			allowedPattern: "/",
			expectedMatch:  true,
		},
		{
			name:           "Empty paths",
			requestPath:    "",
			allowedPattern: "",
			expectedMatch:  true,
		},
		{
			name:           "Trailing slash handling",
			requestPath:    "/health/liveness/",
			allowedPattern: "/health/liveness",
			expectedMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pathMatches(tt.requestPath, tt.allowedPattern)
			assert.Equal(t, tt.expectedMatch, result, "Path match result mismatch")
		})
	}
}

func TestIsEndpointAllowed(t *testing.T) {
	config := map[string][]string{
		"GET": {
			"/health/liveness",
			"/v1/:isolationID/collections/:collectionName/documents/:documentID",
		},
		"POST": {
			"/v1/:isolationID/collections/:collectionName/documents",
		},
	}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedResult bool
	}{
		{
			name:           "Allowed GET endpoint",
			method:         "GET",
			path:           "/health/liveness",
			expectedResult: true,
		},
		{
			name:           "Allowed GET endpoint with parameters",
			method:         "GET",
			path:           "/v1/test-isolation/collections/test-collection/documents/doc1",
			expectedResult: true,
		},
		{
			name:           "Allowed POST endpoint",
			method:         "POST",
			path:           "/v1/test-isolation/collections/test-collection/documents",
			expectedResult: true,
		},
		{
			name:           "Not allowed method",
			method:         "PUT",
			path:           "/health/liveness",
			expectedResult: false,
		},
		{
			name:           "Not allowed path",
			method:         "GET",
			path:           "/v1/test-isolation/collections/test-collection/documents",
			expectedResult: false,
		},
		{
			name:           "Method not in config",
			method:         "DELETE",
			path:           "/some/path",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEndpointAllowed(tt.method, tt.path, config)
			assert.Equal(t, tt.expectedResult, result, "Endpoint allowed result mismatch")
		})
	}
}

func TestIsReadOnlyModeActive(t *testing.T) {
	// Save original environment variables
	originalReadOnlyMode := os.Getenv("READ_ONLY_MODE")
	originalRuntimeConfig := os.Getenv("ENABLE_RUNTIME_HEADER_CONFIG")
	defer func() {
		if originalReadOnlyMode == "" {
			os.Unsetenv("READ_ONLY_MODE")
		} else {
			os.Setenv("READ_ONLY_MODE", originalReadOnlyMode)
		}
		if originalRuntimeConfig == "" {
			os.Unsetenv("ENABLE_RUNTIME_HEADER_CONFIG")
		} else {
			os.Setenv("ENABLE_RUNTIME_HEADER_CONFIG", originalRuntimeConfig)
		}
	}()

	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                string
		readOnlyMode        string
		runtimeConfigHeader string
		serviceModeHeader   string
		expectedResult      bool
	}{
		{
			name:           "READ_ONLY_MODE=true should return true",
			readOnlyMode:   "true",
			expectedResult: true,
		},
		{
			name:           "READ_ONLY_MODE=false should return false",
			readOnlyMode:   "false",
			expectedResult: false,
		},
		{
			name:           "READ_ONLY_MODE not set should return false",
			readOnlyMode:   "",
			expectedResult: false,
		},
		{
			name:                "Runtime config disabled, ServiceMode header ignored",
			readOnlyMode:        "false",
			runtimeConfigHeader: "false",
			serviceModeHeader:   "READONLY",
			expectedResult:      false,
		},
		{
			name:                "Runtime config enabled, ServiceMode READONLY should return true",
			readOnlyMode:        "false",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "READONLY",
			expectedResult:      true,
		},
		{
			name:                "Runtime config enabled, ServiceMode NORMAL should return false",
			readOnlyMode:        "false",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "NORMAL",
			expectedResult:      false,
		},
		{
			name:                "Runtime config enabled, ServiceMode EMULATION should return false",
			readOnlyMode:        "false",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "EMULATION",
			expectedResult:      false,
		},
		{
			name:                "Runtime config enabled, invalid ServiceMode should return false",
			readOnlyMode:        "false",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "INVALID",
			expectedResult:      false,
		},
		{
			name:                "Runtime config enabled, no ServiceMode header should return false",
			readOnlyMode:        "false",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "",
			expectedResult:      false,
		},
		{
			name:                "READ_ONLY_MODE takes precedence over ServiceMode header",
			readOnlyMode:        "true",
			runtimeConfigHeader: "true",
			serviceModeHeader:   "NORMAL",
			expectedResult:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.readOnlyMode != "" {
				os.Setenv("READ_ONLY_MODE", tt.readOnlyMode)
			} else {
				os.Unsetenv("READ_ONLY_MODE")
			}
			if tt.runtimeConfigHeader != "" {
				os.Setenv("ENABLE_RUNTIME_HEADER_CONFIG", tt.runtimeConfigHeader)
			} else {
				os.Unsetenv("ENABLE_RUNTIME_HEADER_CONFIG")
			}

			// Create a mock Gin context
			router := gin.New()
			// Add RuntimeConfigMiddleware if runtime config is enabled
			if tt.runtimeConfigHeader != "" {
				router.Use(RuntimeConfigMiddleware)
			}
			router.GET("/test", func(c *gin.Context) {
				result := isReadOnlyModeActive(c)
				assert.Equal(t, tt.expectedResult, result, "isReadOnlyModeActive result mismatch")
			})

			// Create request with ServiceMode header if specified
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.serviceModeHeader != "" {
				req.Header.Set(headers.ServiceMode, tt.serviceModeHeader)
			}
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)
		})
	}
}

func TestReadOnlyMiddlewareWithCustomConfig(t *testing.T) {
	// Save original environment variable
	originalReadOnlyMode := os.Getenv("READ_ONLY_MODE")
	defer func() {
		if originalReadOnlyMode == "" {
			os.Unsetenv("READ_ONLY_MODE")
		} else {
			os.Setenv("READ_ONLY_MODE", originalReadOnlyMode)
		}
	}()

	gin.SetMode(gin.TestMode)
	os.Setenv("READ_ONLY_MODE", "true")

	// Custom config that only allows GET /test
	customConfig := ReadOnlyConfig{
		AllowedEndpoints: map[string][]string{
			"GET": {"/test"},
		},
	}

	router := gin.New()
	router.Use(ReadOnlyMiddleware(customConfig))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "allowed")
	})
	router.GET("/other", func(c *gin.Context) {
		c.String(http.StatusOK, "should be blocked")
	})
	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should be blocked")
	})

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Custom config - allowed endpoint",
			method:         "GET",
			path:           "/test",
			expectedStatus: http.StatusOK,
			expectedBody:   "allowed",
		},
		{
			name:           "Custom config - blocked GET endpoint",
			method:         "GET",
			path:           "/other",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
		{
			name:           "Custom config - blocked POST endpoint",
			method:         "POST",
			path:           "/test",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   `{"code":"405","message":"Method not allowed in Read Only mode"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")
			assert.Equal(t, tt.expectedBody, w.Body.String(), "Response body mismatch")
		})
	}
}
