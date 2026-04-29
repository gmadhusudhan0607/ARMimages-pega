/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetModelRequestParams(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		path           string
		expectedParams ModelUrlParams
	}{
		{
			name: "Valid model and isolation IDs",
			path: "/openai/deployments/gpt-4/isolations/tenant123/chat/completions",
			expectedParams: ModelUrlParams{
				ModelName:   "gpt-4",
				IsolationId: "tenant123",
			},
		},
		{
			name: "Only model ID",
			path: "/openai/deployments/gpt-4/chat/completions",
			expectedParams: ModelUrlParams{
				ModelName:   "gpt-4",
				IsolationId: "",
			},
		},
		{
			name: "Empty path",
			path: "/",
			expectedParams: ModelUrlParams{
				ModelName:   "",
				IsolationId: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a router with the test path
			router := gin.New()
			router.GET("/openai/deployments/:modelId/isolations/:isolationId/chat/completions", func(c *gin.Context) {
				params := GetModelRequestParams(c)
				assert.Equal(t, tt.expectedParams.ModelName, params.ModelName)
				assert.Equal(t, tt.expectedParams.IsolationId, params.IsolationId)
				c.Status(http.StatusOK)
			})
			router.GET("/openai/deployments/:modelId/chat/completions", func(c *gin.Context) {
				params := GetModelRequestParams(c)
				assert.Equal(t, tt.expectedParams.ModelName, params.ModelName)
				assert.Equal(t, tt.expectedParams.IsolationId, params.IsolationId)
				c.Status(http.StatusOK)
			})
			router.GET("/", func(c *gin.Context) {
				params := GetModelRequestParams(c)
				assert.Equal(t, tt.expectedParams.ModelName, params.ModelName)
				assert.Equal(t, tt.expectedParams.IsolationId, params.IsolationId)
				c.Status(http.StatusOK)
			})

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			resp := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(resp, req)

			// Check that the handler was called
			assert.Equal(t, http.StatusOK, resp.Code)
		})
	}
}
