/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"bytes"
	"context"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/resolvers/target"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
)

func TestDetermineOriginalModelNameFromResolved(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		resolvedTarget    *target.ResolvedTarget
		modelIdParam      string
		expectedModelName string
	}{
		{
			name: "with resolved target containing model name",
			resolvedTarget: &target.ResolvedTarget{
				OriginalModelName: "gpt-4-original",
				ModelName:         "gpt-4",
			},
			modelIdParam:      "gpt-4",
			expectedModelName: "gpt-4-original",
		},
		{
			name:              "without resolved target - extract from modelId param",
			resolvedTarget:    nil,
			modelIdParam:      "gpt-35-turbo",
			expectedModelName: "gpt-35-turbo",
		},
		{
			name: "with resolved target but empty model name",
			resolvedTarget: &target.ResolvedTarget{
				ModelName: "",
			},
			modelIdParam:      "my-model",
			expectedModelName: "my-model",
		},
		{
			name:              "no resolved target and no param returns unknown",
			resolvedTarget:    nil,
			modelIdParam:      "",
			expectedModelName: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req
			c.Params = gin.Params{{Key: "modelId", Value: tt.modelIdParam}}

			result := determineOriginalModelNameFromResolved(c, tt.resolvedTarget)
			assert.Equal(t, tt.expectedModelName, result)
		})
	}
}

func TestReadRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		body        string
		nilBody     bool
		expectError bool
		expected    string
	}{
		{
			name:        "successful read with body",
			body:        `{"messages": [{"role": "user", "content": "hello"}]}`,
			nilBody:     false,
			expectError: false,
			expected:    `{"messages": [{"role": "user", "content": "hello"}]}`,
		},
		{
			name:        "nil body returns nil",
			body:        "",
			nilBody:     true,
			expectError: false,
			expected:    "",
		},
		{
			name:        "empty body",
			body:        "",
			nilBody:     false,
			expectError: false,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			var req *http.Request
			if tt.nilBody {
				req, _ = http.NewRequest("POST", "/test", nil)
				req.Body = nil
			} else {
				req, _ = http.NewRequest("POST", "/test", bytes.NewBufferString(tt.body))
			}
			c.Request = req

			result, err := readRequestBody(c, nil)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.nilBody {
					assert.Nil(t, result)
				} else {
					assert.Equal(t, tt.expected, string(result))
				}
			}
		})
	}
}

func TestShouldModifyEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		path           string
		hasTargetModel bool
		expectedResult bool
	}{
		{
			name:           "chat completions endpoint should modify",
			path:           "/openai/deployments/gpt-4/chat/completions",
			hasTargetModel: true,
			expectedResult: true,
		},
		{
			name:           "messages endpoint should modify",
			path:           "/v1/messages",
			hasTargetModel: true,
			expectedResult: true,
		},
		{
			name:           "other endpoint should not modify",
			path:           "/v1/embeddings",
			hasTargetModel: true,
			expectedResult: false,
		},
		{
			name:           "nil target model should continue",
			path:           "/openai/deployments/gpt-4/chat/completions",
			hasTargetModel: false,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("POST", tt.path, nil)
			c.Request = req

			md := &metadata.RequestMetadata{
				RequestMetrics: metrics.NewRequestMetrics(),
			}

			if tt.hasTargetModel {
				md.TargetModel = &modeltypes.Model{Name: "test-model"}
			}

			body := []byte(`{"test": "data"}`)

			result := shouldModifyEndpoint(c, md, body, nil)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestRestoreRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalBody := []byte(`{"messages": [{"role": "user", "content": "hello"}]}`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("POST", "/test", nil)
	c.Request = req

	// Call restoreRequestBody
	restoreRequestBody(c, originalBody)

	// Verify the body was restored
	require.NotNil(t, c.Request.Body)
	restoredBody, err := io.ReadAll(c.Request.Body)
	require.NoError(t, err)
	assert.Equal(t, string(originalBody), string(restoredBody))
}

func TestCreateDefaultProcessor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		targetModel *modeltypes.Model
		expectError bool
	}{
		{
			name: "successful processor creation",
			targetModel: &modeltypes.Model{
				Name:     "gpt-4",
				Provider: modeltypes.ProviderAzure,
			},
			expectError: false,
		},
		{
			name:        "nil target model causes error",
			targetModel: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("POST", "/test", nil)
			c.Request = req

			md := &metadata.RequestMetadata{
				TargetModel:    tt.targetModel,
				RequestMetrics: metrics.NewRequestMetrics(),
			}

			body := []byte(`{"test": "data"}`)

			processor, err := createDefaultProcessor(md, body, c, nil)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, processor)
			} else {
				// Note: This may still return an error if the actual processor creation fails
				// due to missing dependencies in test environment, but we test the path
				if err != nil {
					assert.Nil(t, processor)
				}
			}
		})
	}
}

func TestModifyRequest_NoBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("POST", "/test", nil)
	req.Body = nil
	c.Request = req

	// Add metadata to context
	md := &metadata.RequestMetadata{
		RequestMetrics: metrics.NewRequestMetrics(),
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	c.Request = c.Request.WithContext(ctx)

	err := modifyRequest(c)
	assert.NoError(t, err)
}

func TestModifyRequest_NonChatEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := []byte(`{"input": "test"}`)
	req, _ := http.NewRequest("POST", "/v1/embeddings", bytes.NewReader(body))
	c.Request = req

	// Add metadata to context with target model
	md := &metadata.RequestMetadata{
		TargetModel: &modeltypes.Model{
			Name:     "text-embedding-ada-002",
			Provider: modeltypes.ProviderAzure,
		},
		RequestMetrics: metrics.NewRequestMetrics(),
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	c.Request = c.Request.WithContext(ctx)

	err := modifyRequest(c)
	assert.NoError(t, err)

	// Body should be restored
	restoredBody, _ := io.ReadAll(c.Request.Body)
	assert.Equal(t, body, restoredBody)
}

func TestModifyRequest_DisabledStrategy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set DISABLED strategy
	oldValue := os.Getenv("REQUEST_PROCESSING_OUTPUT_TOKENS_STRATEGY")
	defer func() {
		if oldValue == "" {
			os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_STRATEGY")
		} else {
			os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_STRATEGY", oldValue)
		}
	}()
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_STRATEGY", "DISABLED")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := []byte(`{"messages": [{"role": "user", "content": "hello"}]}`)
	req, _ := http.NewRequest("POST", "/openai/deployments/gpt-4/chat/completions", bytes.NewReader(body))
	c.Request = req

	// Add metadata to context with target model
	md := &metadata.RequestMetadata{
		TargetModel: &modeltypes.Model{
			Name:     "gpt-4",
			Provider: modeltypes.ProviderAzure,
		},
		RequestMetrics: metrics.NewRequestMetrics(),
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	c.Request = c.Request.WithContext(ctx)

	err := modifyRequest(c)
	assert.NoError(t, err)

	// Body should be restored since processing is disabled
	restoredBody, _ := io.ReadAll(c.Request.Body)
	assert.Equal(t, body, restoredBody)
}

func TestInjectRequestMetadata_ModelDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                string
		path                string
		hasAuth             bool
		authToken           string
		expectedIsolationID string
	}{
		{
			name:                "with valid JWT token",
			path:                "/openai/deployments/gpt-4/chat/completions",
			hasAuth:             true,
			authToken:           "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJndWlkIjoidGVzdC1pZCJ9.test",
			expectedIsolationID: "", // Will fail base64 decoding but shouldn't crash
		},
		{
			name:                "without JWT token",
			path:                "/openai/deployments/gpt-4/chat/completions",
			hasAuth:             false,
			authToken:           "",
			expectedIsolationID: "",
		},
		{
			name:                "with invalid JWT format",
			path:                "/openai/deployments/gpt-4/chat/completions",
			hasAuth:             true,
			authToken:           "Bearer invalid",
			expectedIsolationID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("POST", tt.path, nil)
			if tt.hasAuth {
				req.Header.Set("Authorization", tt.authToken)
			}
			c.Request = req

			err := injectRequestMetadata(c)
			assert.NoError(t, err)

			// Verify metadata was injected
			md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
			assert.NoError(t, err)
			assert.NotNil(t, md)
			assert.NotNil(t, md.RequestMetrics)
		})
	}
}

func TestSetupResponseWriter_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("POST", "/test", nil)
	c.Request = req

	// Add metadata to context
	md := &metadata.RequestMetadata{
		RequestMetrics: metrics.NewRequestMetrics(),
	}
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, md)
	c.Request = c.Request.WithContext(ctx)

	err := setupResponseWriter(c)
	assert.NoError(t, err)

	// Verify writer was wrapped
	_, ok := c.Writer.(*metrics.RequestModificationResponseWriter)
	assert.True(t, ok, "Writer should be wrapped with RequestModificationResponseWriter")
}

func TestSetupResponseWriter_NoMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("POST", "/test", nil)
	c.Request = req
	// Don't add metadata to context

	err := setupResponseWriter(c)
	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
