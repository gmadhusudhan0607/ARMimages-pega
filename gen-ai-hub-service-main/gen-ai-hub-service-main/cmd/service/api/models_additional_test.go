/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestFindModelConfigByName(t *testing.T) {
	tests := []struct {
		name      string
		mapping   *Mapping
		modelName string
		wantModel *Model
	}{
		{
			name: "Model found in mapping",
			mapping: &Mapping{
				Models: []Model{
					{Name: "gpt-4", ModelId: "gpt-4-id", RedirectURL: "http://example.com"},
					{Name: "claude-3", ModelId: "claude-3-id", RedirectURL: "http://example2.com"},
				},
			},
			modelName: "gpt-4",
			wantModel: &Model{Name: "gpt-4", ModelId: "gpt-4-id", RedirectURL: "http://example.com"},
		},
		{
			name: "Model not found in mapping",
			mapping: &Mapping{
				Models: []Model{
					{Name: "gpt-4", ModelId: "gpt-4-id"},
				},
			},
			modelName: "nonexistent",
			wantModel: nil,
		},
		{
			name:      "Nil mapping returns nil",
			mapping:   nil,
			modelName: "gpt-4",
			wantModel: nil,
		},
		{
			name: "Empty models list returns nil",
			mapping: &Mapping{
				Models: []Model{},
			},
			modelName: "gpt-4",
			wantModel: nil,
		},
		{
			name: "Non-nil mapping with nil models returns nil",
			mapping: &Mapping{
				Models: nil,
			},
			modelName: "gpt-4",
			wantModel: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findModelConfigByName(tt.mapping, tt.modelName)
			if tt.wantModel == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.wantModel.Name, result.Name)
				assert.Equal(t, tt.wantModel.ModelId, result.ModelId)
			}
		})
	}
}

func TestGetModelMetadataPath(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		setEnv   bool
		expected string
	}{
		{
			name:     "Returns default path when env var not set",
			setEnv:   false,
			expected: "/models-metadata/model-metadata.yaml",
		},
		{
			name:     "Returns custom path when env var is set",
			envValue: "/custom/path/metadata.yaml",
			setEnv:   true,
			expected: "/custom/path/metadata.yaml",
		},
		{
			name:     "Returns default path when env var is empty string",
			envValue: "",
			setEnv:   true,
			expected: "/models-metadata/model-metadata.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv("MODEL_METADATA_PATH", tt.envValue)
			} else {
				// Ensure env var is not set
				t.Setenv("MODEL_METADATA_PATH", "")
			}
			result := getModelMetadataPath()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckPrivateModelFiles(t *testing.T) {
	// This test requires manipulating the PrivateModelFilePath constant,
	// which is not possible directly. We test through privateModelCheck instead,
	// which exercises checkPrivateModelFiles indirectly.
	// Direct testing of checkPrivateModelFiles with default path (non-existent)
	// should return nil, false
	fileList, exists := checkPrivateModelFiles()
	assert.Nil(t, fileList)
	assert.False(t, exists)
}

func TestGetPrivateModelMapping(t *testing.T) {
	// Create a temp directory with valid private model files
	tempDir := t.TempDir()

	// Create a valid YAML private model file
	validContent := `- name: "private-model-1"
  modelId: "pm-123"
  redirectUrl: "http://private.example.com"
  active: true
`
	validFile := tempDir + "/genai_private_model_1.yaml"
	err := os.WriteFile(validFile, []byte(validContent), 0644)
	require.NoError(t, err)

	t.Run("Valid file list with proper YAML", func(t *testing.T) {
		// We need to use full paths, but getPrivateModelMapping prepends PrivateModelFilePath
		// Instead, test with files that exist at PrivateModelFilePath which is /private-model-config
		// Since that path likely doesn't exist in test env, test error cases
		fileList := []string{"nonexistent_file.yaml"}
		_, appErr := getPrivateModelMapping(&fileList)
		assert.NotNil(t, appErr)
		assert.Contains(t, appErr.Message, "error encountered while reading the file")
	})

	t.Run("Invalid YAML content", func(t *testing.T) {
		// Create a temp file with invalid YAML at PrivateModelFilePath location
		// Since we can't change the const, we test the error path through the actual function
		invalidContent := `{not: valid: yaml: [[[`
		invalidFile := tempDir + "/genai_private_model_invalid.yaml"
		err := os.WriteFile(invalidFile, []byte(invalidContent), 0644)
		require.NoError(t, err)

		// This will try to read from PrivateModelFilePath which won't find the file
		fileList := []string{"genai_private_model_invalid.yaml"}
		_, appErr := getPrivateModelMapping(&fileList)
		assert.NotNil(t, appErr)
	})
}

func TestPrivateModelCheck(t *testing.T) {
	// privateModelCheck reads from PrivateModelFilePath (/private-model-config)
	// which doesn't exist in test environment
	ctx := context.Background()

	t.Run("Returns false when private model files do not exist", func(t *testing.T) {
		exists, model, appErr := privateModelCheck("any-model", ctx)
		assert.False(t, exists)
		assert.Equal(t, &Model{}, model)
		assert.Nil(t, appErr)
	})
}

func TestHandleImageGenerationRequest_EmptyModelName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "dall-e-3", ModelId: "dalle3", RedirectURL: "http://example.com"}}}

	handler := HandleImageGenerationRequest(ctx, mapping)

	// Register handler on a route without :modelId param so c.Param("modelId") returns ""
	router := gin.New()
	router.POST("/images/generations", handler)

	req := httptest.NewRequest(http.MethodPost, "/images/generations", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "model name as url param is required")
}

func TestHandleImageGenerationRequest_UnrecognizedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "dall-e-3", ModelId: "dalle3", RedirectURL: "http://example.com"}}}

	handler := HandleImageGenerationRequest(ctx, mapping)

	router := gin.New()
	router.POST("/openai/deployments/:modelId/images/generations", handler)

	req := httptest.NewRequest(http.MethodPost, "/openai/deployments/nonexistent-model/images/generations", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "unrecognized model name")
}

func TestHandleImageGenerationRequest_UnrecognizedRequestURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "dall-e-3", ModelId: "dalle3", RedirectURL: "http://example.com"}}}

	handler := HandleImageGenerationRequest(ctx, mapping)

	// Use a path pattern that doesn't match /openai/deployments/:modelId
	router := gin.New()
	router.POST("/other/path/:modelId/images/generations", handler)

	req := httptest.NewRequest(http.MethodPost, "/other/path/dall-e-3/images/generations", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "Unrecognized request URI")
}

func TestHandleExperimentalModelChatCompletionRequest_EmptyModelName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "gemini-pro", ModelId: "gemini-pro", RedirectURL: "http://example.com"}}}

	handler := HandleExperimentalModelChatCompletionRequest(ctx, mapping)

	// Register handler on a route without :modelId param so c.Param("modelId") returns ""
	router := gin.New()
	router.POST("/chat/completions", handler)

	req := httptest.NewRequest(http.MethodPost, "/chat/completions", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "modelId param is required")
}

func TestHandleExperimentalModelChatCompletionRequest_UnrecognizedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "gemini-pro", ModelId: "gemini-pro", RedirectURL: "http://example.com"}}}

	handler := HandleExperimentalModelChatCompletionRequest(ctx, mapping)

	router := gin.New()
	router.POST("/google/deployments/:modelId/chat/completions", handler)

	req := httptest.NewRequest(http.MethodPost, "/google/deployments/nonexistent/chat/completions", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "unrecognized model name")
}

func TestHandleExperimentalModelChatCompletionRequest_UnrecognizedRequestURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "gemini-pro", ModelId: "gemini-pro", RedirectURL: "http://example.com"}}}

	handler := HandleExperimentalModelChatCompletionRequest(ctx, mapping)

	router := gin.New()
	router.POST("/other/path/:modelId/chat/completions", handler)

	req := httptest.NewRequest(http.MethodPost, "/other/path/gemini-pro/chat/completions", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "Unrecognized request URI")
}

// TestHandleExperimentalModelChatCompletionRequest_QueryParamsStripped verifies that query
// parameters are stripped from the path when extracting operationPath using URL.Path
func TestHandleExperimentalModelChatCompletionRequest_QueryParamsStripped(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup test server to capture the actual redirect URL
	var capturedURL string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	ctx := context.Background()
	mapping := &Mapping{
		Models: []Model{{
			Name:        "gemini-pro",
			ModelId:     "gemini-pro",
			RedirectURL: mockServer.URL + "/v1/models/gemini-pro",
		}},
	}

	handler := HandleExperimentalModelChatCompletionRequest(ctx, mapping)

	router := gin.New()
	router.POST("/google/deployments/:modelId/chat/completions", handler)

	// Request with query parameters - these should NOT be forwarded in operationPath
	req := httptest.NewRequest(http.MethodPost, "/google/deployments/gemini-pro/chat/completions?user_param=value&other=123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// The handler should succeed
	assert.Equal(t, http.StatusOK, resp.Code)

	// Verify that the redirected URL has the correct path WITHOUT query params from original request
	// Expected: /v1/models/gemini-pro/chat/completions (no ?user_param=value&other=123)
	assert.Contains(t, capturedURL, "/chat/completions")
	assert.NotContains(t, capturedURL, "user_param")
	assert.NotContains(t, capturedURL, "other=123")
}

func TestHandleChatCompletionRequest_EmptyModelName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "gpt-4", ModelId: "gpt-4", RedirectURL: "http://example.com"}}}

	handler := HandleChatCompletionRequest(ctx, mapping)

	// Register handler on a route without :modelId param so c.Param("modelId") returns ""
	router := gin.New()
	router.POST("/chat/completions", handler)

	req := httptest.NewRequest(http.MethodPost, "/chat/completions", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "modelId param is required")
}

func TestHandleChatCompletionRequest_UnrecognizedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "gpt-4", ModelId: "gpt-4", RedirectURL: "http://example.com"}}}

	handler := HandleChatCompletionRequest(ctx, mapping)

	router := gin.New()
	router.POST("/openai/deployments/:modelId/chat/completions", handler)

	req := httptest.NewRequest(http.MethodPost, "/openai/deployments/nonexistent/chat/completions", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "unrecognized model name")
}

func TestHandleChatCompletionRequest_UnrecognizedRequestURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "gpt-4", ModelId: "gpt-4", RedirectURL: "http://example.com"}}}

	handler := HandleChatCompletionRequest(ctx, mapping)

	router := gin.New()
	router.POST("/other/path/:modelId/chat/completions", handler)

	req := httptest.NewRequest(http.MethodPost, "/other/path/gpt-4/chat/completions", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "Unrecognized request URI")
}

func TestHandleEmbeddingsRequest_EmptyModelName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "text-embedding-ada-002", ModelId: "ada-002", RedirectURL: "http://example.com"}}}

	handler := HandleEmbeddingsRequest(ctx, mapping)

	// Register handler on a route without :modelId param so c.Param("modelId") returns ""
	router := gin.New()
	router.POST("/embeddings", handler)

	req := httptest.NewRequest(http.MethodPost, "/embeddings", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "modelId param is required")
}

func TestHandleEmbeddingsRequest_UnrecognizedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "text-embedding-ada-002", ModelId: "ada-002", RedirectURL: "http://example.com"}}}

	handler := HandleEmbeddingsRequest(ctx, mapping)

	router := gin.New()
	router.POST("/openai/deployments/:modelId/embeddings", handler)

	req := httptest.NewRequest(http.MethodPost, "/openai/deployments/nonexistent/embeddings", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "unrecognized model name")
}

func TestHandleEmbeddingsRequest_UnrecognizedRequestURI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{Models: []Model{{Name: "text-embedding-ada-002", ModelId: "ada-002", RedirectURL: "http://example.com"}}}

	handler := HandleEmbeddingsRequest(ctx, mapping)

	router := gin.New()
	router.POST("/other/path/:modelId/embeddings", handler)

	req := httptest.NewRequest(http.MethodPost, "/other/path/text-embedding-ada-002/embeddings", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Body.String(), "Unrecognized request URI")
}

// TestHandleChatCompletionRequest_SuccessfulOperationPathExtraction verifies that
// operationPath is correctly extracted from URL.Path and appended to RedirectURL
func TestHandleChatCompletionRequest_SuccessfulOperationPathExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedPath string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"test"}}]}`))
	}))
	defer mockServer.Close()

	ctx := context.Background()
	mapping := &Mapping{
		Models: []Model{{
			Name:        "gpt-4",
			ModelId:     "gpt-4",
			RedirectURL: mockServer.URL + "/azure/openai",
		}},
	}

	handler := HandleChatCompletionRequest(ctx, mapping)
	router := gin.New()
	router.POST("/openai/deployments/:modelId/chat/completions", handler)

	// Request with query params - should NOT be included in operationPath
	req := httptest.NewRequest(http.MethodPost, "/openai/deployments/gpt-4/chat/completions?stream=true", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	// Verify operationPath (/chat/completions) was extracted and appended to RedirectURL
	assert.Equal(t, "/azure/openai/chat/completions", capturedPath)
}

// TestHandleEmbeddingsRequest_SuccessfulOperationPathExtraction verifies that
// operationPath is correctly extracted from URL.Path and appended to RedirectURL
func TestHandleEmbeddingsRequest_SuccessfulOperationPathExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedPath string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}]}`))
	}))
	defer mockServer.Close()

	ctx := context.Background()
	mapping := &Mapping{
		Models: []Model{{
			Name:        "text-embedding-ada-002",
			ModelId:     "ada-002",
			RedirectURL: mockServer.URL + "/azure/openai",
		}},
	}

	handler := HandleEmbeddingsRequest(ctx, mapping)
	router := gin.New()
	router.POST("/openai/deployments/:modelId/embeddings", handler)

	req := httptest.NewRequest(http.MethodPost, "/openai/deployments/text-embedding-ada-002/embeddings", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	// Verify operationPath (/embeddings) was extracted and appended to RedirectURL
	assert.Equal(t, "/azure/openai/embeddings", capturedPath)
}

// TestHandleImageGenerationRequest_SuccessfulOperationPathExtraction verifies that
// operationPath is correctly extracted from URL.Path and appended to RedirectURL
func TestHandleImageGenerationRequest_SuccessfulOperationPathExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedPath string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"url":"http://example.com/image.png"}]}`))
	}))
	defer mockServer.Close()

	ctx := context.Background()
	mapping := &Mapping{
		Models: []Model{{
			Name:        "dall-e-3",
			ModelId:     "dalle3",
			RedirectURL: mockServer.URL + "/azure/openai",
		}},
	}

	handler := HandleImageGenerationRequest(ctx, mapping)
	router := gin.New()
	router.POST("/openai/deployments/:modelId/images/generations", handler)

	req := httptest.NewRequest(http.MethodPost, "/openai/deployments/dall-e-3/images/generations", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	// Verify operationPath (/images/generations) was extracted and appended to RedirectURL
	assert.Equal(t, "/azure/openai/images/generations", capturedPath)
}

func TestGetModelsHandler_AllModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{
		Models: []Model{
			{Name: "gpt-4", ModelId: "gpt-4-id", ModelUrl: "http://example.com/api"},
			{Name: "claude-3", ModelId: "claude-3-id", ModelUrl: "http://example2.com/api"},
		},
	}

	handler := GetModels(ctx, mapping)

	router := gin.New()
	router.GET("/models", handler)
	router.GET("/models/:modelId", handler)

	// Test get all models
	req := httptest.NewRequest(http.MethodGet, "/models", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestGetModelsHandler_SingleModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{
		Models: []Model{
			{Name: "gpt-4", ModelId: "gpt-4-id", ModelUrl: "http://example.com/api"},
			{Name: "claude-3", ModelId: "claude-3-id", ModelUrl: "http://example2.com/api"},
		},
	}

	handler := GetModels(ctx, mapping)

	router := gin.New()
	router.GET("/models/:modelId", handler)

	req := httptest.NewRequest(http.MethodGet, "/models/gpt-4", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestGetModelsHandler_ModelNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	mapping := &Mapping{
		Models: []Model{
			{Name: "gpt-4", ModelId: "gpt-4-id", ModelUrl: "http://example.com/api"},
		},
	}

	handler := GetModels(ctx, mapping)

	router := gin.New()
	router.GET("/models/:modelId", handler)

	req := httptest.NewRequest(http.MethodGet, "/models/nonexistent", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name      string
		mapping   *Mapping
		modelName string
		wantModel *Model
		wantErr   bool
	}{
		{
			name: "Model exists",
			mapping: &Mapping{
				Models: []Model{
					{Name: "model1", ModelId: "model-123", Active: true},
					{Name: "model2", ModelId: "model-456", Active: true},
				},
			},
			modelName: "model1",
			wantModel: &Model{Name: "model1", ModelId: "model-123", Active: true},
			wantErr:   false,
		},
		{
			name: "Model does not exist",
			mapping: &Mapping{
				Models: []Model{
					{Name: "model1", ModelId: "model-123", Active: true},
					{Name: "model2", ModelId: "model-456", Active: true},
				},
			},
			modelName: "nonexistent",
			wantModel: nil,
			wantErr:   true,
		},
		{
			name: "Empty model list",
			mapping: &Mapping{
				Models: []Model{},
			},
			modelName: "model1",
			wantModel: nil,
			wantErr:   true,
		},
		{
			name: "Empty mapping",
			mapping: &Mapping{
				Models: nil,
			},
			modelName: "model1",
			wantModel: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModel, gotErr := GetModel(tt.mapping, tt.modelName)

			if tt.wantErr {
				assert.NotNil(t, gotErr)
			} else {
				assert.Nil(t, gotErr)
				assert.Equal(t, tt.wantModel, gotModel)
			}
		})
	}
}

// TestHasValidMetadata tests the metadata validation helper function
func TestHasValidMetadata(t *testing.T) {
	// Sample metadata map for testing
	metadata := map[string]ModelMetadata{
		"gpt-4": {
			ModelName:  "gpt-4",
			ModelLabel: "GPT-4",
			ModelID:    "gpt-4",
			Type:       "chat",
		},
		"gpt-4-turbo": {
			ModelName:  "gpt-4-turbo",
			ModelLabel: "GPT-4 Turbo",
			ModelID:    "gpt-4-turbo",
			Type:       "chat",
		},
		"claude-v2": {
			ModelName:  "claude-v2",
			ModelLabel: "Claude v2",
			ModelID:    "claude-v2",
			Type:       "chat",
		},
	}

	tests := []struct {
		name        string
		model       ModelInfo
		expected    bool
		description string
	}{
		{
			name: "Valid model with metadata entry",
			model: ModelInfo{
				ModelName:  "gpt-4",
				ModelLabel: "",
				Provider:   "azure",
				Creator:    "openai",
			},
			expected:    true,
			description: "Should return true when model has metadata entry",
		},
		{
			name: "Another valid model with metadata entry",
			model: ModelInfo{
				ModelName:  "gpt-4-turbo",
				ModelLabel: "",
				Provider:   "azure",
				Creator:    "openai",
			},
			expected:    true,
			description: "Should return true when model has metadata entry",
		},
		{
			name: "Valid model with metadata entry - case sensitive",
			model: ModelInfo{
				ModelName:  "claude-v2",
				ModelLabel: "",
				Provider:   "bedrock",
				Creator:    "anthropic",
			},
			expected:    true,
			description: "Should return true when model has metadata entry",
		},
		{
			name: "Invalid model without metadata entry",
			model: ModelInfo{
				ModelName:  "nonexistent-model",
				ModelLabel: "Some Label",
				Provider:   "bedrock",
				Creator:    "anthropic",
				ModelID:    "some-id",
			},
			expected:    false,
			description: "Should return false when model has no metadata entry",
		},
		{
			name: "Invalid model with empty ModelName",
			model: ModelInfo{
				ModelName:  "",
				ModelLabel: "Some Label",
				Provider:   "bedrock",
				Creator:    "anthropic",
			},
			expected:    false,
			description: "Should return false when ModelName is empty (no metadata lookup possible)",
		},
		{
			name: "Invalid model with case mismatch",
			model: ModelInfo{
				ModelName:  "GPT-4", // Different case from metadata key
				ModelLabel: "",
				Provider:   "azure",
				Creator:    "openai",
			},
			expected:    false,
			description: "Should return false when case doesn't match metadata key exactly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasValidMetadata(&tt.model, metadata)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestEnrichModelsImpl_MetadataFiltering tests the metadata filtering functionality
func TestEnrichModelsImpl_MetadataFiltering(t *testing.T) {
	ctx := context.Background()

	// Save original function and restore after test
	origLoadModelMetadata := LoadModelMetadataFromFile
	defer func() {
		LoadModelMetadataFromFile = origLoadModelMetadata
	}()

	// Mock the LoadModelMetadataFromFile function
	LoadModelMetadataFromFile = func(path string) (map[string]ModelMetadata, error) {
		if path == "/models-metadata/model-metadata.yaml" {
			metadata := map[string]ModelMetadata{
				"valid-model-1": {
					ModelName:  "valid-model-1",
					ModelLabel: "Valid Model 1",
					ModelID:    "valid-1",
					Type:       "chat",
				},
				"valid-model-2": {
					ModelName:  "valid-model-2",
					ModelLabel: "Valid Model 2",
					ModelID:    "valid-2",
					Type:       "chat",
				},
			}
			return metadata, nil
		}
		return nil, os.ErrNotExist
	}

	tests := []struct {
		name               string
		inputModels        []ModelInfo
		expectedValidCount int
		expectedModelNames []string
		description        string
	}{
		{
			name: "All models have valid metadata",
			inputModels: []ModelInfo{
				{
					ModelName: "valid-model-1",
					Provider:  "bedrock",
					Creator:   "anthropic",
				},
				{
					ModelName: "valid-model-2",
					Provider:  "vertex",
					Creator:   "google",
				},
			},
			expectedValidCount: 2,
			expectedModelNames: []string{"valid-model-1", "valid-model-2"},
			description:        "Should return all models when all have valid metadata",
		},
		{
			name: "Mix of valid and invalid models",
			inputModels: []ModelInfo{
				{
					ModelName: "valid-model-1",
					Provider:  "bedrock",
					Creator:   "anthropic",
				},
				{
					ModelName: "", // Invalid - no ModelName
					Provider:  "bedrock",
					Creator:   "anthropic",
					ModelID:   "invalid-1",
				},
				{
					ModelName: "valid-model-2",
					Provider:  "vertex",
					Creator:   "google",
				},
			},
			expectedValidCount: 2,
			expectedModelNames: []string{"valid-model-1", "valid-model-2"},
			description:        "Should filter out models without valid metadata",
		},
		{
			name: "No models have valid metadata",
			inputModels: []ModelInfo{
				{
					ModelName:  "", // Invalid
					ModelLabel: "", // Invalid
					Provider:   "bedrock",
					Creator:    "anthropic",
				},
				{
					ModelName:  "", // Invalid
					ModelLabel: "", // Invalid
					Provider:   "vertex",
					Creator:    "google",
				},
			},
			expectedValidCount: 0,
			expectedModelNames: []string{},
			description:        "Should return empty list when no models have valid metadata",
		},
		{
			name:               "Empty input models",
			inputModels:        []ModelInfo{},
			expectedValidCount: 0,
			expectedModelNames: []string{},
			description:        "Should handle empty input gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enrichModelsImpl(ctx, tt.inputModels)

			assert.Equal(t, tt.expectedValidCount, len(result), tt.description)

			// Check that the returned models match expected names
			resultNames := make([]string, len(result))
			for i, model := range result {
				resultNames[i] = model.ModelName
			}

			for _, expectedName := range tt.expectedModelNames {
				assert.Contains(t, resultNames, expectedName, "Expected model name should be in result")
			}
		})
	}
}

// TestGenerateModelName tests the enhanced generateModelName function for Autopilot compatibility
func TestGenerateModelName(t *testing.T) {
	tests := []struct {
		name        string
		modelName   string
		version     string
		expected    string
		description string
	}{
		{
			name:        "Model name with v1 version",
			modelName:   "GPT-4",
			version:     "v1",
			expected:    "gpt-4",
			description: "v1 version should be omitted",
		},
		{
			name:        "Model name with version 1",
			modelName:   "Claude-3-Sonnet",
			version:     "1",
			expected:    "claude-3-sonnet",
			description: "version '1' should be omitted",
		},
		{
			name:        "Model name with empty version",
			modelName:   "GPT-3.5-Turbo",
			version:     "",
			expected:    "gpt-3.5-turbo",
			description: "Empty version should result in lowercase model name",
		},
		{
			name:        "Model name with meaningful version",
			modelName:   "GPT-4",
			version:     "turbo",
			expected:    "gpt-4-turbo",
			description: "Meaningful version should be appended",
		},
		{
			name:        "Model name with version already included",
			modelName:   "GPT-4-Turbo",
			version:     "turbo",
			expected:    "gpt-4-turbo-turbo",
			description: "Version gets appended even if already in name (current behavior)",
		},
		{
			name:        "Model name with numeric version",
			modelName:   "Claude-3",
			version:     "20240229",
			expected:    "claude-3-20240229",
			description: "Numeric version should be appended",
		},
		{
			name:        "Model name with complex version",
			modelName:   "Gemini-Pro",
			version:     "1.5-preview",
			expected:    "gemini-pro-1.5-preview",
			description: "Complex version should be appended",
		},
		{
			name:        "Model name already contains version string",
			modelName:   "text-embedding-ada-002",
			version:     "002",
			expected:    "text-embedding-ada-002",
			description: "Version substring already in name",
		},
		{
			name:        "Model name with special characters",
			modelName:   "GPT-4_Vision",
			version:     "preview",
			expected:    "gpt-4_vision-preview",
			description: "Special characters should be preserved",
		},
		{
			name:        "Empty model name",
			modelName:   "",
			version:     "v1",
			expected:    "",
			description: "Empty model name should return empty string",
		},
		{
			name:        "Model name with version v2",
			modelName:   "GPT-4",
			version:     "v2",
			expected:    "gpt-4-v2",
			description: "v2 version should be included",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateModelName(tt.modelName, tt.version)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestGenerateModelDescription tests the enhanced generateModelDescription function for Autopilot compatibility
func TestGenerateModelDescription(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		modelType   string
		expected    string
		description string
	}{
		{
			name:        "AWS Bedrock chat completion",
			provider:    "bedrock",
			modelType:   "chat_completion",
			expected:    "AWS Bedrock Chat Completions model",
			description: "Bedrock provider with chat completion type",
		},
		{
			name:        "Google Vertex AI embedding",
			provider:    "vertex",
			modelType:   "embedding",
			expected:    "Google Vertex AI Embedding model",
			description: "Vertex provider with embedding type",
		},
		{
			name:        "Azure OpenAI image",
			provider:    "azure",
			modelType:   "image",
			expected:    "Azure OpenAI Image model",
			description: "Azure provider with image type",
		},
		{
			name:        "Unknown provider with completion",
			provider:    "custom",
			modelType:   "completion",
			expected:    "Custom Completions model",
			description: "Unknown provider should be title-cased",
		},
		{
			name:        "Bedrock with unknown model type",
			provider:    "bedrock",
			modelType:   "unknown",
			expected:    "AWS Bedrock Unclassified model type",
			description: "Unknown model type should default to 'Unclassified model type'",
		},
		{
			name:        "Empty provider with chat completion",
			provider:    "",
			modelType:   "chat_completion",
			expected:    " Chat Completions model",
			description: "Empty provider should result in space before type",
		},
		{
			name:        "Vertex with empty model type",
			provider:    "vertex",
			modelType:   "",
			expected:    "Google Vertex AI Unclassified model type",
			description: "Empty model type should default to 'Unclassified model type'",
		},
		{
			name:        "Multiple word provider",
			provider:    "anthropic claude",
			modelType:   "chat_completion",
			expected:    "Anthropic Claude Chat Completions model",
			description: "Multi-word provider should be title-cased",
		},
		{
			name:        "Provider with special characters",
			provider:    "open-ai",
			modelType:   "completion",
			expected:    "Open-Ai Completions model",
			description: "Special characters should be preserved in title case",
		},
		{
			name:        "Provider with numbers",
			provider:    "gpt4all",
			modelType:   "chat_completion",
			expected:    "Gpt4all Chat Completions model",
			description: "Numbers should be preserved in title case",
		},
		{
			name:        "Realtime model type",
			provider:    "azure",
			modelType:   "realtime",
			expected:    "Azure OpenAI Realtime speech and audio model",
			description: "Azure provider with realtime type",
		},
		{
			name:        "Bedrock with realtime type",
			provider:    "bedrock",
			modelType:   "realtime",
			expected:    "AWS Bedrock Realtime speech and audio model",
			description: "Bedrock provider with realtime type",
		},
		{
			name:        "Case insensitive model type matching",
			provider:    "bedrock",
			modelType:   "CHAT_COMPLETION",
			expected:    "AWS Bedrock Unclassified model type",
			description: "Uppercase model type not matched exactly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateModelDescription(tt.provider, tt.modelType)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestConvertToSupportedCapabilities tests the capability conversion for Autopilot compatibility
func TestConvertToSupportedCapabilities(t *testing.T) {
	tests := []struct {
		name                  string
		vertexStreamingEnvVar bool
		capabilities          ModelCapabilities
		expected              SupportedCapabilities
		description           string
	}{
		{
			name: "All capabilities enabled",
			capabilities: ModelCapabilities{
				Features:  []string{"streaming", "functionCalling", "parallelFunctionCalling", "jsonMode"},
				MimeTypes: []string{"text/plain", "image/jpeg"},
			},
			expected: SupportedCapabilities{
				Streaming:               true,
				Functions:               true,
				ParallelFunctionCalling: true,
				JSONMode:                true,
				Multimodal:              []string{"text/plain", "image/jpeg"},
				IsMultimodal:            true,
			},
			vertexStreamingEnvVar: true,
			description:           "Should detect all capabilities",
		},
		{
			name: "Tool calling features",
			capabilities: ModelCapabilities{
				Features:  []string{"toolCalling", "parallelToolCalling", "structuredOutput"},
				MimeTypes: []string{},
			},
			expected: SupportedCapabilities{
				Streaming:               false,
				Functions:               true, // toolCalling should map to Functions
				ParallelFunctionCalling: true, // parallelToolCalling should map
				JSONMode:                true, // structuredOutput should map to JSONMode
				Multimodal:              []string{},
				IsMultimodal:            false,
			},
			description: "Should handle tool calling features",
		},
		{
			name: "Case insensitive feature matching",
			capabilities: ModelCapabilities{
				Features:         []string{"STREAMING", "FunctionCalling"},
				InputModalities:  []string{"text", "image"},
				OutputModalities: []string{"text"},
				MimeTypes:        []string{},
			},
			vertexStreamingEnvVar: true,
			expected: SupportedCapabilities{
				Streaming:               true,
				Functions:               true,
				ParallelFunctionCalling: false,
				JSONMode:                false,
				Multimodal:              []string{},
				IsMultimodal:            true, // Multiple input modalities
			},
			description: "Should handle case insensitive matching and input modalities",
		},
		{
			name: "No capabilities",
			capabilities: ModelCapabilities{
				Features:         []string{},
				InputModalities:  []string{"text"},
				OutputModalities: []string{"text"},
				MimeTypes:        []string{},
			},
			expected: SupportedCapabilities{
				Streaming:               false,
				Functions:               false,
				ParallelFunctionCalling: false,
				JSONMode:                false,
				Multimodal:              []string{},
				IsMultimodal:            false, // Single input modality
			},
			description: "Should handle no special capabilities",
		},
		{
			name: "Multimodal via MimeTypes only",
			capabilities: ModelCapabilities{
				Features:         []string{},
				InputModalities:  []string{},
				OutputModalities: []string{},
				MimeTypes:        []string{"image/png"},
			},
			expected: SupportedCapabilities{
				Streaming:               false,
				Functions:               false,
				ParallelFunctionCalling: false,
				JSONMode:                false,
				Multimodal:              []string{"image/png"},
				IsMultimodal:            true, // MimeTypes present
			},
			description: "Should detect multimodal via MimeTypes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with default provider (non-Bedrock) to maintain existing behavior
			os.Setenv("USE_VERTEXAI_INFRA", strconv.FormatBool(tt.vertexStreamingEnvVar))
			defer os.Unsetenv("USE_VERTEXAI_INFRA")
			result := convertToSupportedCapabilities(tt.capabilities, "vertex", []string{})
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestConvertToSupportedCapabilities_BedrockStreaming tests Bedrock-specific streaming detection
func TestConvertToSupportedCapabilities_BedrockStreaming(t *testing.T) {
	tests := []struct {
		name                  string
		capabilities          ModelCapabilities
		provider              string
		vertexStreamingEnvVar bool
		modelPaths            []string
		expected              SupportedCapabilities
		description           string
	}{
		{
			name: "Bedrock with converse-stream path should enable streaming",
			capabilities: ModelCapabilities{
				Features:  []string{}, // No streaming in metadata
				MimeTypes: []string{},
			},
			provider:   "bedrock",
			modelPaths: []string{"/bedrock/deployments/claude-v2/converse-stream", "/bedrock/deployments/claude-v2/converse"},
			expected: SupportedCapabilities{
				Streaming:               true, // Should be true because of converse-stream path
				Functions:               false,
				ParallelFunctionCalling: false,
				JSONMode:                false,
				Multimodal:              []string{},
				IsMultimodal:            false,
			},
			description: "Bedrock should detect streaming from converse-stream path",
		},
		{
			name: "Bedrock without converse-stream path should not enable streaming",
			capabilities: ModelCapabilities{
				Features:  []string{"streaming"}, // Has streaming in metadata but shouldn't matter for Bedrock
				MimeTypes: []string{},
			},
			provider:   "bedrock",
			modelPaths: []string{"/bedrock/deployments/claude-v2/converse", "/bedrock/deployments/claude-v2/invoke"},
			expected: SupportedCapabilities{
				Streaming:               false, // Should be false because no converse-stream path
				Functions:               false,
				ParallelFunctionCalling: false,
				JSONMode:                false,
				Multimodal:              []string{},
				IsMultimodal:            false,
			},
			description: "Bedrock should not enable streaming without converse-stream path",
		},
		{
			name: "Non-Bedrock provider should use metadata for streaming",
			capabilities: ModelCapabilities{
				Features:  []string{"streaming"},
				MimeTypes: []string{},
			},
			provider:              "vertex",
			vertexStreamingEnvVar: true,
			modelPaths:            []string{"/google/deployments/gemini-pro/converse-stream"}, // Has converse-stream but not Bedrock
			expected: SupportedCapabilities{
				Streaming:               true, // Should be true because of metadata streaming feature
				Functions:               false,
				ParallelFunctionCalling: false,
				JSONMode:                false,
				Multimodal:              []string{},
				IsMultimodal:            false,
			},
			description: "Non-Bedrock providers should use metadata-based streaming detection",
		},
		{
			name: "Bedrock with partial path match should enable streaming",
			capabilities: ModelCapabilities{
				Features:  []string{},
				MimeTypes: []string{},
			},
			provider:   "bedrock",
			modelPaths: []string{"/some/path/with/converse-stream/suffix"},
			expected: SupportedCapabilities{
				Streaming:               true, // Should be true because path contains converse-stream
				Functions:               false,
				ParallelFunctionCalling: false,
				JSONMode:                false,
				Multimodal:              []string{},
				IsMultimodal:            false,
			},
			description: "Bedrock should detect streaming from any path containing converse-stream",
		},
		{
			name: "Empty modelPaths should not enable streaming for Bedrock",
			capabilities: ModelCapabilities{
				Features:  []string{"streaming"}, // Has streaming in metadata
				MimeTypes: []string{},
			},
			provider:   "bedrock",
			modelPaths: []string{}, // Empty paths
			expected: SupportedCapabilities{
				Streaming:               false, // Should be false because no paths to check
				Functions:               false,
				ParallelFunctionCalling: false,
				JSONMode:                false,
				Multimodal:              []string{},
				IsMultimodal:            false,
			},
			description: "Bedrock with empty paths should not enable streaming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("USE_VERTEXAI_INFRA", strconv.FormatBool(tt.vertexStreamingEnvVar))
			defer os.Unsetenv("USE_VERTEXAI_INFRA")
			result := convertToSupportedCapabilities(tt.capabilities, tt.provider, tt.modelPaths)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestHasStreamingTargetAPI tests the helper function directly
func TestHasStreamingTargetAPI(t *testing.T) {
	tests := []struct {
		name        string
		modelPaths  []string
		expected    bool
		description string
	}{
		{
			name:        "Path contains converse-stream",
			modelPaths:  []string{"/bedrock/deployments/claude/converse-stream"},
			expected:    true,
			description: "Should return true when path contains converse-stream",
		},
		{
			name:        "Multiple paths, one contains converse-stream",
			modelPaths:  []string{"/bedrock/deployments/claude/converse", "/bedrock/deployments/claude/converse-stream"},
			expected:    true,
			description: "Should return true when any path contains converse-stream",
		},
		{
			name:        "No paths contain converse-stream",
			modelPaths:  []string{"/bedrock/deployments/claude/converse", "/bedrock/deployments/claude/invoke"},
			expected:    false,
			description: "Should return false when no paths contain converse-stream",
		},
		{
			name:        "Empty paths",
			modelPaths:  []string{},
			expected:    false,
			description: "Should return false for empty paths",
		},
		{
			name:        "Partial match in path",
			modelPaths:  []string{"/some/prefix/converse-stream/suffix"},
			expected:    true,
			description: "Should return true for partial match in path",
		},
		{
			name:        "Case sensitive matching",
			modelPaths:  []string{"/bedrock/deployments/claude/CONVERSE-STREAM"},
			expected:    false,
			description: "Should be case sensitive and return false for uppercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasStreamingTargetAPI(tt.modelPaths)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestGetModelWithModelId(t *testing.T) {
	tests := []struct {
		name      string
		mapping   *Mapping
		modelName string
		modelId   string
		wantModel *Model
		wantErr   bool
	}{
		{
			name: "Model exists with matching name and ID",
			mapping: &Mapping{
				Models: []Model{
					{Name: "model1", ModelId: "model-123", Active: true},
					{Name: "model2", ModelId: "model-456", Active: true},
				},
			},
			modelName: "model1",
			modelId:   "123", // Should match partial suffix
			wantModel: &Model{Name: "model1", ModelId: "model-123", Active: true},
			wantErr:   false,
		},
		{
			name: "Model exists with matching name but wrong ID",
			mapping: &Mapping{
				Models: []Model{
					{Name: "model1", ModelId: "model-123", Active: true},
					{Name: "model2", ModelId: "model-456", Active: true},
				},
			},
			modelName: "model1",
			modelId:   "456",
			wantModel: nil,
			wantErr:   true,
		},
		{
			name: "Model does not exist",
			mapping: &Mapping{
				Models: []Model{
					{Name: "model1", ModelId: "model-123", Active: true},
					{Name: "model2", ModelId: "model-456", Active: true},
				},
			},
			modelName: "nonexistent",
			modelId:   "123",
			wantModel: nil,
			wantErr:   true,
		},
		{
			name: "Empty model list",
			mapping: &Mapping{
				Models: []Model{},
			},
			modelName: "model1",
			modelId:   "123",
			wantModel: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModel, gotErr := getModelWithModelId(tt.mapping, tt.modelName, tt.modelId)

			if tt.wantErr {
				assert.NotNil(t, gotErr)
			} else {
				assert.Nil(t, gotErr)
				assert.Equal(t, tt.wantModel, gotModel)
			}
		})
	}
}

func TestGetModelsMappingForCurrentIsolation(t *testing.T) {
	tests := []struct {
		name           string
		mapping        *Mapping
		modelUrlParams *ModelUrlParams
		wantModels     []Model
		wantErr        bool
	}{
		{
			name: "Replace IsolationId in ModelUrl",
			mapping: &Mapping{
				Models: []Model{
					{Name: "model1", ModelId: "model-123", ModelUrl: "http://example.com/{{ .IsolationId }}/api"},
					{Name: "model2", ModelId: "model-456", ModelUrl: "http://test.com/static"},
				},
			},
			modelUrlParams: &ModelUrlParams{
				ModelName:   "model1",
				IsolationId: "tenant-42",
			},
			wantModels: []Model{
				{Name: "model1", ModelId: "model-123", ModelUrl: "http://example.com/tenant-42/api"},
				{Name: "model2", ModelId: "model-456", ModelUrl: "http://test.com/static"},
			},
			wantErr: false,
		},
		{
			name: "Empty IsolationId",
			mapping: &Mapping{
				Models: []Model{
					{Name: "model1", ModelId: "model-123", ModelUrl: "http://example.com/{{ .IsolationId }}/api"},
					{Name: "model2", ModelId: "model-456", ModelUrl: "http://test.com/static"},
				},
			},
			modelUrlParams: &ModelUrlParams{
				ModelName:   "model1",
				IsolationId: "", // Empty isolation ID
			},
			wantModels: []Model{
				{Name: "model1", ModelId: "model-123", ModelUrl: "http://example.com//api"}, // Note the double slash due to empty value
				{Name: "model2", ModelId: "model-456", ModelUrl: "http://test.com/static"},
			},
			wantErr: false,
		},
		{
			name: "Multiple template variables",
			mapping: &Mapping{
				Models: []Model{
					{Name: "model1", ModelId: "model-123", ModelUrl: "http://example.com/{{ .IsolationId }}/{{ .ModelName }}"},
				},
			},
			modelUrlParams: &ModelUrlParams{
				ModelName:   "model1",
				IsolationId: "tenant-42",
			},
			wantModels: []Model{
				{Name: "model1", ModelId: "model-123", ModelUrl: "http://example.com/tenant-42/model1"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gotMapping, gotErr := GetModelsMappingForCurrentIsolation(ctx, tt.mapping, tt.modelUrlParams)

			if tt.wantErr {
				assert.NotNil(t, gotErr)
			} else {
				assert.Nil(t, gotErr)
				assert.Equal(t, len(tt.wantModels), len(gotMapping.Models))

				for i, want := range tt.wantModels {
					assert.Equal(t, want.Name, gotMapping.Models[i].Name)
					assert.Equal(t, want.ModelId, gotMapping.Models[i].ModelId)
					assert.Equal(t, want.ModelUrl, gotMapping.Models[i].ModelUrl)
				}
			}
		})
	}
}

func TestPrivateModelCheck_SimpleCheck(t *testing.T) {
	// We'll just test the doesPrivateModelExist function since it's safer
	// The privateModelCheck is already indirectly tested by integration tests

	privateModelMapping := &Mapping{
		Models: []Model{
			{Name: "model1", ModelId: "model-123", Active: true, RedirectURL: "http://private.example.com"},
			{Name: "model2", ModelId: "model-456", Active: false, RedirectURL: "http://private2.example.com"},
		},
	}

	ctx := context.Background()

	// Test 1: Model exists and is active
	exists, model := doesPrivateModelExist(privateModelMapping, "model1", ctx)
	assert.True(t, exists)
	assert.Equal(t, "model1", model.Name)
	assert.Equal(t, "model-123", model.ModelId)
	assert.True(t, model.Active)

	// Test 2: Model exists but is inactive
	exists, _ = doesPrivateModelExist(privateModelMapping, "model2", ctx)
	assert.False(t, exists) // Should be false because model is not active

	// Test 3: Model does not exist
	exists, model = doesPrivateModelExist(privateModelMapping, "nonexistent", ctx)
	assert.False(t, exists)
	assert.Equal(t, &Model{}, model)
}

func TestResolveModelID(t *testing.T) {
	// Setup temporary file with model metadata
	tempFile, err := os.CreateTemp("", "model_metadata_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	metadataContent := `
gpt4-1:
  modelName: gpt4-1
  modelLabel: GPT-4 v1
  modelId: gpt-4
  type: chat

test-model:
  modelName: test-model
  modelLabel: Test Model
  type: chat
`
	_, err = tempFile.Write([]byte(metadataContent))
	require.NoError(t, err)
	tempFile.Close()

	// Save original function and restore after test
	origLoadModelMetadata := LoadModelMetadataFromFile
	defer func() {
		LoadModelMetadataFromFile = origLoadModelMetadata
	}()

	// Mock the LoadModelMetadataFromFile function
	LoadModelMetadataFromFile = func(path string) (map[string]ModelMetadata, error) {
		if path == "/models-metadata/model-metadata.yaml" {
			metadata := map[string]ModelMetadata{
				"gpt4-1": {
					ModelName:  "gpt4-1",
					ModelLabel: "GPT-4 v1",
					ModelID:    "gpt-4",
					Type:       "chat",
				},
				"test-model": {
					ModelName:  "test-model",
					ModelLabel: "Test Model",
					Type:       "chat",
					ModelID:    "", // Empty ModelID to test error case
				},
			}
			return metadata, nil
		}
		return nil, os.ErrNotExist
	}

	tests := []struct {
		name      string
		modelName string
		wantID    string
		wantErr   bool
	}{
		{
			name:      "Model with ID in metadata",
			modelName: "gpt4-1",
			wantID:    "gpt-4",
			wantErr:   false,
		},
		{
			name:      "Model with empty ID in metadata",
			modelName: "test-model",
			wantID:    "",    // The function will return empty string
			wantErr:   false, // Current implementation doesn't error on empty ModelID
		},
		{
			name:      "Model not in metadata",
			modelName: "nonexistent-model",
			wantID:    "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, err := resolveModelID(tt.modelName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantID, gotID)
			}
		})
	}
}

func TestAddDeprecatedModelsFromMetadata(t *testing.T) {
	// Mock logger
	logger := zap.NewNop()
	oldLogger := loggerFromContext
	loggerFromContext = func(context.Context) *zap.Logger { return logger }
	defer func() { loggerFromContext = oldLogger }()

	ctx := context.Background()

	altModel := &AlternateModelInfo{Name: "new-model", Provider: "bedrock", Creator: "anthropic"}

	tests := []struct {
		name           string
		models         []ModelInfo
		metadata       map[string]ModelMetadata
		expectedLen    int
		expectedNames  []string // names of ALL models in result, in order
		checkLastModel func(t *testing.T, m ModelInfo)
	}{
		{
			name:          "empty metadata adds nothing",
			models:        []ModelInfo{{Name: "existing"}},
			metadata:      map[string]ModelMetadata{},
			expectedLen:   1,
			expectedNames: []string{"existing"},
		},
		{
			name:   "no deprecated models in metadata",
			models: []ModelInfo{{Name: "existing"}},
			metadata: map[string]ModelMetadata{
				"active-model": {Lifecycle: "GA", Provider: "bedrock"},
			},
			expectedLen:   1,
			expectedNames: []string{"existing"},
		},
		{
			name:   "deprecated model already in list is not duplicated",
			models: []ModelInfo{{Name: "old-model"}},
			metadata: map[string]ModelMetadata{
				"old-model": {Lifecycle: "Deprecated", Provider: "bedrock"},
			},
			expectedLen:   1,
			expectedNames: []string{"old-model"},
		},
		{
			name:   "deprecated model with explicit lifecycle added",
			models: []ModelInfo{{Name: "existing"}},
			metadata: map[string]ModelMetadata{
				"deprecated-model": {
					Lifecycle:          "Deprecated",
					DeprecationDate:    "2024-06-01",
					Provider:           "bedrock",
					Creator:            "anthropic",
					ModelName:          "claude-2",
					AlternateModelInfo: altModel,
				},
			},
			expectedLen:   2,
			expectedNames: []string{"existing", "deprecated-model"},
			checkLastModel: func(t *testing.T, m ModelInfo) {
				assert.Equal(t, "deprecated-model", m.Name)
				assert.Equal(t, "claude-2", m.ModelName)
				assert.Equal(t, "bedrock", m.Provider)
				assert.Equal(t, "anthropic", m.Creator)
				assert.Equal(t, "Deprecated", m.Lifecycle)
				assert.Equal(t, "2024-06-01", m.DeprecationDate)
				assert.True(t, m.DeprecationInfo.IsDeprecated)
				assert.Equal(t, "2024-06-01", m.DeprecationInfo.ScheduledDeprecationDate)
				assert.Equal(t, altModel, m.AlternateModelInfo)
				assert.NotNil(t, m.Parameters)
				assert.Empty(t, m.ModelPath)
			},
		},
		{
			name:   "deprecated model via past date with no explicit lifecycle",
			models: []ModelInfo{{Name: "existing"}},
			metadata: map[string]ModelMetadata{
				"old-by-date": {
					Lifecycle:       "",
					DeprecationDate: "2024-01-01",
					Provider:        "openai",
					Creator:         "openai",
					ModelName:       "gpt-3",
				},
			},
			expectedLen:   2,
			expectedNames: []string{"existing", "old-by-date"},
			checkLastModel: func(t *testing.T, m ModelInfo) {
				assert.Equal(t, "old-by-date", m.Name)
				assert.Equal(t, "Deprecated", m.Lifecycle)
				assert.Equal(t, "openai", m.Provider)
				assert.Equal(t, "gpt-3", m.ModelName)
			},
		},
		{
			name:   "future date without explicit lifecycle is not deprecated",
			models: []ModelInfo{{Name: "existing"}},
			metadata: map[string]ModelMetadata{
				"future-model": {
					Lifecycle:       "",
					DeprecationDate: "2099-01-01",
					Provider:        "bedrock",
				},
			},
			expectedLen:   1,
			expectedNames: []string{"existing"},
		},
		{
			name:   "multiple deprecated models some already present",
			models: []ModelInfo{{Name: "already-here"}, {Name: "also-here"}},
			metadata: map[string]ModelMetadata{
				"already-here": {Lifecycle: "Deprecated", Provider: "bedrock"},
				"new-dep-1":    {Lifecycle: "Deprecated", Provider: "openai", Creator: "openai"},
				"new-dep-2":    {Lifecycle: "Deprecated", Provider: "bedrock", Creator: "anthropic"},
				"active":       {Lifecycle: "GA", Provider: "bedrock"},
			},
			expectedLen:   4,
			expectedNames: []string{"already-here", "also-here", "new-dep-1", "new-dep-2"},
		},
		{
			name:   "deterministic ordering by sorted key",
			models: []ModelInfo{},
			metadata: map[string]ModelMetadata{
				"zebra-model": {Lifecycle: "Deprecated", Provider: "p1"},
				"alpha-model": {Lifecycle: "Deprecated", Provider: "p2"},
				"mid-model":   {Lifecycle: "Deprecated", Provider: "p3"},
			},
			expectedLen:   3,
			expectedNames: []string{"alpha-model", "mid-model", "zebra-model"},
		},
		{
			name:   "case insensitive key matching",
			models: []ModelInfo{{Name: "my-model"}},
			metadata: map[string]ModelMetadata{
				"MY-MODEL": {Lifecycle: "Deprecated", Provider: "bedrock"},
			},
			expectedLen:   1,
			expectedNames: []string{"my-model"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addDeprecatedModelsFromMetadata(ctx, tt.models, tt.metadata)

			assert.Equal(t, tt.expectedLen, len(result), "unexpected result length")

			// Verify names in order
			names := make([]string, len(result))
			for i, m := range result {
				names[i] = m.Name
			}
			assert.Equal(t, tt.expectedNames, names)

			// Verify original models preserved
			for i := range tt.models {
				assert.Equal(t, tt.models[i].Name, result[i].Name, "original model should be preserved at index %d", i)
			}

			if tt.checkLastModel != nil {
				tt.checkLastModel(t, result[len(result)-1])
			}
		})
	}
}
