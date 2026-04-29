/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
)

func TestRequestProcessingMiddleware_WithValidJWTToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequestModificationMiddleware(context.Background()))

	// Create a valid JWT token with isolation ID
	claims := map[string]interface{}{
		"guid": "test-isolation-id-123",
		"sub":  "test-user",
		"iat":  1234567890,
	}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64.URLEncoding.EncodeToString(claimsJSON)

	// Create a simple JWT (header.payload.signature)
	header := base64.URLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	signature := base64.URLEncoding.EncodeToString([]byte("fake-signature"))
	token := header + "." + payload + "." + signature

	router.GET("/test", func(c *gin.Context) {
		requestMetadata, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		assert.NoError(t, err, "RequestMetadata should be injected into context")
		assert.NotNil(t, requestMetadata, "RequestMetadata should be injected into context")

		// Verify isolation ID was extracted
		assert.Equal(t, "test-isolation-id-123", requestMetadata.IsolationID, "IsolationID should be extracted from JWT")

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestProcessingMiddleware_WithInvalidJWTToken(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequestModificationMiddleware(context.Background()))

	router.GET("/test", func(c *gin.Context) {
		requestMetadata, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		assert.NoError(t, err, "RequestMetadata should be injected into context")
		assert.NotNil(t, requestMetadata, "RequestMetadata should be injected into context")

		// Verify isolation ID is empty for invalid token
		assert.Equal(t, "", requestMetadata.IsolationID, "IsolationID should be empty for invalid JWT")

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test with invalid token
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestProcessingMiddleware_ModelDetection(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(RequestModificationMiddleware(context.Background()))

	// Test with a model endpoint path (this will fail model detection but should not crash)
	router.POST("/openai/deployments/:modelId/chat/completions", func(c *gin.Context) {
		requestMetadata, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
		assert.NoError(t, err, "RequestMetadata should be injected into context")
		assert.NotNil(t, requestMetadata, "RequestMetadata should be injected into context")

		// Model detection will likely fail in test environment, but middleware should continue
		// The important thing is that it doesn't crash and metadata is still injected
		assert.NotNil(t, requestMetadata.RequestMetrics, "RequestMetrics should be initialized")

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test
	req, _ := http.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
}
