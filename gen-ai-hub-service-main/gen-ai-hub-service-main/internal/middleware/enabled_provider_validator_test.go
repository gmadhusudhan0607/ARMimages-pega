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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestProviderEnabled(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		provider         string
		enabledProviders string
		expectedStatus   int
	}{
		{
			name:             "Provider is enabled",
			provider:         "Azure",
			enabledProviders: "Azure,Vertex,Bedrock",
			expectedStatus:   http.StatusOK,
		},
		{
			name:             "Provider is not enabled",
			provider:         "OpenAI",
			enabledProviders: "Azure,Vertex,Bedrock",
			expectedStatus:   http.StatusForbidden,
		},
		{
			name:             "Empty provider list",
			provider:         "Azure",
			enabledProviders: "",
			expectedStatus:   http.StatusOK, // Default is "Azure,Vertex,Bedrock" when empty
		},
		{
			name:             "Provider with whitespace in list",
			provider:         "Vertex",
			enabledProviders: "Azure, Vertex , Bedrock",
			expectedStatus:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable for the test
			originalValue := os.Getenv("ENABLED_PROVIDERS")
			os.Setenv("ENABLED_PROVIDERS", tt.enabledProviders)
			defer os.Setenv("ENABLED_PROVIDERS", originalValue) // Restore original value after test

			// Create a new gin router with the middleware
			router := gin.New()
			router.Use(ProviderEnabled(tt.provider))

			// Add a handler that will be called if middleware passes
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(w, req)

			// Assert the expected status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// For forbidden cases, verify the error message
			if tt.expectedStatus == http.StatusForbidden {
				expectedErrorMsg := `{"error":"Provider ` + tt.provider + ` is not enabled"}`
				assert.Equal(t, expectedErrorMsg, w.Body.String())
			}
		})
	}
}
