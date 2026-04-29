/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func Test_doesPrivateModelExist(t *testing.T) {

	type args struct {
		privateModelMapping *Mapping
		model               string
		ctx                 context.Context
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test1: Model is present in Private Model Mapping and Active flag is true",
			args: args{
				privateModelMapping: &Mapping{
					Models: []Model{
						{
							Name:    "model1",
							ModelId: "model-123",
							Active:  true,
						},
						{
							Name:    "model2",
							ModelId: "model-678",
							Active:  true,
						},
					},
				},
				model: "model1",
				ctx:   context.Background(),
			},
			want: true,
		},
		{
			name: "Test2: Model is present in Private Model Mapping and Active flag is false",
			args: args{
				privateModelMapping: &Mapping{
					Models: []Model{
						{
							Name:    "model1",
							ModelId: "model-123",
							Active:  false,
						},
						{
							Name:    "model2",
							ModelId: "model-678",
							Active:  true,
						},
					},
				},
				model: "model1",
				ctx:   context.Background(),
			},
			want: false,
		},
		{
			name: "Test3: Model is NOT present in Private Model Mapping",
			args: args{
				privateModelMapping: &Mapping{
					Models: []Model{
						{
							Name:    "model1",
							ModelId: "model-123",
							Active:  true,
						},
						{
							Name:    "model2",
							ModelId: "model-678",
							Active:  true,
						},
					},
				},
				model: "notPresent",
				ctx:   context.Background(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, _ := doesPrivateModelExist(tt.args.privateModelMapping, tt.args.model, tt.args.ctx)
			if got != tt.want {
				t.Errorf("doesPrivateModelExist() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInferAPIFromPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Contains chat", "some/path/chat", []string{"/chat/completions"}},
		{"Contains gemini", "google/gemini/path", []string{"/chat/completions"}},
		{"Contains converse", "api/converse", []string{"/converse"}},
		{"Contains converse-stream", "api/converse-stream", []string{"/converse-stream"}},
		{"Contains embedding", "model/embedding", []string{"/embeddings"}},
		{"Contains image", "vision/image", []string{"/images/generations"}},
		{"Contains vision", "model/vision", []string{"/images/generations"}},
		{"Contains invoke", "model/invoke", []string{"/invoke"}},
		{"Contains realtime", "model/realtime", []string{"/v1/realtime/client_secrets"}},
		{"Unknown path", "model/unknown", []string{"/unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferAPIFromPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferCreatorFromPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Path contains google", "some/path/google", "google"},
		{"Path contains meta", "api/meta/path", "meta"},
		{"Path contains amazon", "amazon/models/path", "amazon"},
		{"Path contains anthropic", "anthropic/ai/path", "anthropic"},
		{"Path does not match any known creator", "unknown/path", "openai"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferCreatorFromPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnrichWithMetadata(t *testing.T) {
	min := 0.0
	max := 1.0

	tests := []struct {
		name     string
		model    *ModelInfo
		metadata map[string]ModelMetadata
		expected ModelInfo
	}{
		{
			name: "Metadata found and applied",
			model: &ModelInfo{
				ModelName: "test-model",
			},
			metadata: map[string]ModelMetadata{
				"test-model": {
					Lifecycle:       "active",
					DeprecationDate: "2026-01-01",
					AlternateModelInfo: &AlternateModelInfo{
						Name:     "fallback-model",
						Provider: "bedrock",
						Creator:  "anthropic",
					},
					ModelCapabilities: ModelCapabilities{
						InputModalities:  []string{"text"},
						OutputModalities: []string{"text"},
						Features:         []string{"streaming"},
						MimeTypes:        []string{"text/plain"},
					},
					Parameters: map[string]ParameterSpec{
						"temperature": {
							Title:       "Temperature",
							Description: "Controls randomness",
							Type:        "float",
							Default:     0.7,
							Minimum:     &min,
							Maximum:     &max,
							Required:    false,
							Examples:    []string{},
						},
					},
					ModelLabel:       "Test Model",
					ModelName:        "test-model",
					ModelDescription: "Test model description",
					Version:          "v1",
					Type:             "chat",
				},
			},
			expected: ModelInfo{
				ModelName:       "test-model",
				Name:            "test-model", // Generated from modelName and version
				Description:     "Test model description",
				Lifecycle:       "active",
				DeprecationDate: "2026-01-01",
				AlternateModelInfo: &AlternateModelInfo{
					Name:     "fallback-model",
					Provider: "bedrock",
					Creator:  "anthropic",
				},
				// New Autopilot-aligned fields
				DeprecationInfo: DeprecationInfo{
					IsDeprecated:             false,
					ScheduledDeprecationDate: "2026-01-01",
				},
				SupportedCapabilities: SupportedCapabilities{
					Streaming:               true,                   // Converted from features
					Multimodal:              []string{"text/plain"}, // From mimeTypes
					Functions:               false,
					ParallelFunctionCalling: false,
					JSONMode:                false,
					IsMultimodal:            true, // true because mimeTypes exist
				},
				Parameters: map[string]ParameterSpec{
					"temperature": {
						Title:       "Temperature",
						Description: "Controls randomness",
						Type:        "float",
						Default:     0.7,
						Minimum:     &min,
						Maximum:     &max,
						Required:    false,
						Examples:    []string{},
					},
				},
				ModelLabel: "Test Model",
				Type:       "chat",
				Version:    "v1",
			},
		},
		{
			name: "Metadata not found",
			model: &ModelInfo{
				ModelName: "unknown-model",
			},
			metadata: map[string]ModelMetadata{},
			expected: ModelInfo{
				ModelName:  "unknown-model",
				ModelLabel: "",
				ModelID:    "",
				Type:       "",
				Version:    "",
				ModelPath:  []string{},
				Creator:    "",
				Parameters: map[string]ParameterSpec{},
				Provider:   "",
				// New Autopilot-aligned fields for unknown models
				SupportedCapabilities: SupportedCapabilities{
					Streaming:               false,
					Multimodal:              []string{}, // Empty slice, not nil
					Functions:               false,
					ParallelFunctionCalling: false,
					JSONMode:                false,
					IsMultimodal:            false,
				},
				DeprecationInfo: DeprecationInfo{
					IsDeprecated:             false,
					ScheduledDeprecationDate: "",
				},
				Lifecycle:       "",
				DeprecationDate: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			enrichWithMetadata(ctx, tt.model, tt.metadata, tt.model.ModelName)
			assert.Equal(t, tt.expected, *tt.model)
		})
	}
}

func TestInferLifecycleFromDate(t *testing.T) {
	tests := []struct {
		name            string
		deprecationDate string
		expected        string
	}{
		{
			name:            "Empty date returns Generally Available",
			deprecationDate: "",
			expected:        "Generally Available",
		},
		{
			name:            "NA date returns Generally Available",
			deprecationDate: "NA",
			expected:        "Generally Available",
		},
		{
			name:            "Invalid format returns Generally Available",
			deprecationDate: "2025/12/31",
			expected:        "Generally Available",
		},
		{
			name:            "Past date returns Deprecated",
			deprecationDate: time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
			expected:        "Deprecated",
		},
		{
			name:            "Date within 3 months returns Nearing Deprecation",
			deprecationDate: time.Now().AddDate(0, 2, 0).Format("2006-01-02"),
			expected:        "Nearing Deprecation",
		},
		{
			name:            "Future date beyond 3 months returns Generally Available",
			deprecationDate: time.Now().AddDate(0, 4, 0).Format("2006-01-02"),
			expected:        "Generally Available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferLifecycleFromDate(tt.deprecationDate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadModelMetadataFromFile(t *testing.T) {
	t.Run("Valid metadata file", func(t *testing.T) {
		content := `
test-model:
  model_name: test-model
  model_label: Test Model
  type: chat
  lifecycle: active
  deprecation_date: "2026-01-01"
  fallback_model: fallback-model
  availableRegions: ["us-east", "eu-west"]
  parameters:
    temperature:
      title: Temperature
      description: Controls randomness
      type: float
      default: 0.7
      minimum: 0.0
      maximum: 1.0
      required: false
`

		tmpFile, err := os.CreateTemp("", "model_metadata_*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte(content))
		require.NoError(t, err)
		tmpFile.Close()

		result, err := LoadModelMetadataFromFile(tmpFile.Name())
		require.NoError(t, err)

		meta, ok := result["test-model"]
		require.True(t, ok)
		assert.Equal(t, "test-model", meta.ModelName)
		assert.Equal(t, "Test Model", meta.ModelLabel)
		assert.Equal(t, "chat", meta.Type)
		assert.Equal(t, "active", meta.Lifecycle)
		assert.Equal(t, "2026-01-01", meta.DeprecationDate)
		assert.Contains(t, meta.Parameters, "temperature")
		assert.Equal(t, "Temperature", meta.Parameters["temperature"].Title)
	})

	t.Run("File not found", func(t *testing.T) {
		_, err := LoadModelMetadataFromFile("nonexistent.yaml")
		assert.Error(t, err)
	})
}

func Test_deduplicateModelsImpl(t *testing.T) {
	tests := []struct {
		name     string
		models   []ModelInfo
		expected []ModelInfo
	}{
		{
			name: "No duplicates",
			models: []ModelInfo{
				{ModelName: "gpt-4", ModelPath: []string{"/openai/deployments/gpt-4"}},
				{ModelName: "claude-3", ModelPath: []string{"/anthropic/deployments/claude-3"}},
			},
			expected: []ModelInfo{
				{ModelName: "gpt-4", ModelPath: []string{"/openai/deployments/gpt-4"}},
				{ModelName: "claude-3", ModelPath: []string{"/anthropic/deployments/claude-3"}},
			},
		},
		{
			name: "Simple duplicate models",
			models: []ModelInfo{
				{ModelName: "gpt-4", ModelPath: []string{"/openai/deployments/gpt-4"}},
				{ModelName: "gpt-4", ModelPath: []string{"/openai/deployments/gpt-4-for-embeddings"}},
				{ModelName: "claude-3", ModelPath: []string{"/anthropic/deployments/claude-3"}},
			},
			expected: []ModelInfo{
				{ModelName: "gpt-4", ModelPath: []string{"/openai/deployments/gpt-4", "/openai/deployments/gpt-4-for-embeddings"}},
				{ModelName: "claude-3", ModelPath: []string{"/anthropic/deployments/claude-3"}},
			},
		},
		{
			name:     "Empty input",
			models:   []ModelInfo{},
			expected: []ModelInfo{},
		},
		{
			name: "Mixed models with duplicates",
			models: []ModelInfo{
				{ModelName: "gpt-4", ModelPath: []string{"/openai/deployments/gpt-4"}, Creator: "openai"},
				{ModelName: "claude-3", ModelPath: []string{"/anthropic/deployments/claude-3"}, Creator: "anthropic"},
				{ModelName: "gpt-4", ModelPath: []string{"/openai/deployments/gpt-4"}, Creator: "openai"},
				{ModelName: "llama-3", ModelPath: []string{"/meta/deployments/llama-3"}, Creator: "meta"},
			},
			expected: []ModelInfo{
				{ModelName: "gpt-4", ModelPath: []string{"/openai/deployments/gpt-4"}, Creator: "openai"},
				{ModelName: "claude-3", ModelPath: []string{"/anthropic/deployments/claude-3"}, Creator: "anthropic"},
				{ModelName: "llama-3", ModelPath: []string{"/meta/deployments/llama-3"}, Creator: "meta"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateModelsImpl(tt.models)

			// Sort results for consistent comparison
			sort.Slice(result, func(i, j int) bool {
				return result[i].ModelName < result[j].ModelName
			})
			sort.Slice(tt.expected, func(i, j int) bool {
				return tt.expected[i].ModelName < tt.expected[j].ModelName
			})

			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_mergeUniqueStrings(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want []string
	}{
		{
			name: "Both empty slices",
			a:    []string{},
			b:    []string{},
			want: []string{},
		},
		{
			name: "First slice empty",
			a:    []string{},
			b:    []string{"one", "two", "three"},
			want: []string{"one", "two", "three"},
		},
		{
			name: "Second slice empty",
			a:    []string{"one", "two", "three"},
			b:    []string{},
			want: []string{"one", "two", "three"},
		},
		{
			name: "No overlapping elements",
			a:    []string{"one", "two"},
			b:    []string{"three", "four"},
			want: []string{"one", "two", "three", "four"},
		},
		{
			name: "Some overlapping elements",
			a:    []string{"one", "two", "three"},
			b:    []string{"two", "three", "four"},
			want: []string{"one", "two", "three", "four"},
		},
		{
			name: "All overlapping elements",
			a:    []string{"one", "two"},
			b:    []string{"one", "two"},
			want: []string{"one", "two"},
		},
		{
			name: "Duplicates within same slice",
			a:    []string{"one", "one", "two"},
			b:    []string{"three", "three"},
			want: []string{"one", "two", "three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeUniqueStrings(tt.a, tt.b)

			// Sort both slices for consistent comparison
			sort.Strings(got)
			sort.Strings(tt.want)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_fetchAzureModels(t *testing.T) {

	// Save original functions and restore after test
	origGetEnvOrPanic := getEnvOrPanic
	origRetrieveMapping := retrieveMapping
	defer func() {
		getEnvOrPanic = origGetEnvOrPanic
		retrieveMapping = origRetrieveMapping
	}()

	// Mock HTTP server
	tests := []struct {
		name           string
		configFile     string
		serverResponse string
		statusCode     int
		wantModels     []ModelInfo
		wantStatus     int
		wantErr        bool
		mapping        *Mapping
		mappingErr     error
	}{
		{
			name:       "successful response with multiple models",
			configFile: "test-config.yaml",
			serverResponse: `{
				"models": [
					{
						"deployment-id": "deploy1",
						"model-name": "gpt4",
						"model-version": "1",
						"type": "chat",
						"endpoint": "/deployments/gpt4/chat/completions"
					},
					{
						"deployment-id": "deploy2",
						"model-name": "dalle",
						"model-version": "2",
						"type": "image",
						"endpoint": "/deployments/dall-e-3/images/generations"
					}
				]
			}`,
			statusCode: http.StatusOK,
			wantModels: []ModelInfo{
				{
					Provider:  "azure",
					ModelPath: []string{"/openai/deployments/gpt4/chat/completions"},
					Creator:   "openai",
					ModelName: "gpt4-1",
				},
				{
					Provider:  "azure",
					ModelPath: []string{"/openai/deployments/dall-e-3/images/generations"},
					Creator:   "openai",
					ModelName: "dalle-2",
				},
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
			mapping:    &Mapping{Models: []Model{}},
			mappingErr: nil,
		},
		{
			name:           "unauthorized response",
			configFile:     "unauthorized-config.yaml",
			serverResponse: `{"error": "Unauthorized"}`,
			statusCode:     http.StatusUnauthorized,
			wantModels:     nil,
			wantStatus:     http.StatusUnauthorized,
			wantErr:        true,
			mapping:        &Mapping{Models: []Model{}},
			mappingErr:     nil,
		},
		{
			name:           "invalid JSON response",
			configFile:     "invalid-config.yaml",
			serverResponse: `invalid json`,
			statusCode:     http.StatusOK,
			wantModels:     nil,
			wantStatus:     http.StatusOK,
			wantErr:        true,
			mapping:        &Mapping{Models: []Model{}},
			mappingErr:     nil,
		},
		{
			name:           "empty response",
			configFile:     "empty-config.yaml",
			serverResponse: `{"models": []}`,
			statusCode:     http.StatusOK,
			wantModels:     []ModelInfo{},
			wantStatus:     http.StatusOK,
			wantErr:        false,
			mapping:        &Mapping{Models: []Model{}},
			mappingErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock getEnvOrPanic
			getEnvOrPanic = func(key string) string {
				assert.Equal(t, "CONFIGURATION_FILE", key)
				return tt.configFile
			}

			// Mock retrieveMapping
			retrieveMapping = func(ctx context.Context, path string) (*Mapping, error) {
				assert.Equal(t, tt.configFile, path)
				if tt.mappingErr != nil {
					return nil, tt.mappingErr
				}
				return tt.mapping, nil
			}
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request path
				assert.Equal(t, "/openai/models", r.URL.Path)
				// Verify the request method
				assert.Equal(t, "GET", r.Method)

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse)) //nolint:errcheck
			}))
			defer server.Close()

			// Create a context for the test
			ctx := context.Background()

			// Call the function
			gotModels, gotStatus, err := fetchAzureModels(ctx, server.URL)

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Check status code
			assert.Equal(t, tt.wantStatus, gotStatus)

			// Check models
			if tt.wantModels != nil {
				assert.Equal(t, tt.wantModels, gotModels)
			}
		})
	}
}

func Test_fetchGCPModelsImpl(t *testing.T) {
	// Save original functions and restore after test
	origGetEnvOrPanic := getEnvOrPanic
	origRetrieveMapping := retrieveMapping
	origLoadModelMetadata := LoadModelMetadataFromFile
	defer func() {
		getEnvOrPanic = origGetEnvOrPanic
		retrieveMapping = origRetrieveMapping
		LoadModelMetadataFromFile = origLoadModelMetadata
	}()

	tests := []struct {
		name           string
		configFile     string
		mapping        *Mapping
		mappingErr     error
		expectedModels []ModelInfo
		wantErr        bool
	}{
		{
			name:       "successful fetch with multiple GCP models",
			configFile: "test-config.yaml",
			mapping: &Mapping{
				Models: []Model{
					{Name: "gemini-1.5-flash", ModelUrl: "google/deployments/gemini-1.5-flash", Infrastructure: "gcp", Creator: "google", TargetAPI: "/chat/completions", Path: "/google/deployments/gemini-1.5-flash/chat/completions"},
					{Name: "imagen-3", ModelUrl: "google/deployments/imagen-3", Infrastructure: "gcp", Creator: "google", TargetAPI: "/images/generations", Path: "/google/deployments/imagen-3/images/generations"},
					{Name: "azure-model", ModelUrl: "/azure/path", Infrastructure: "azure"}, // Should be filtered out
				},
			},
			mappingErr: nil,
			expectedModels: []ModelInfo{
				{Provider: "vertex", ModelPath: []string{"/google/deployments/gemini-1.5-flash/chat/completions"}, Creator: "google", ModelName: "gemini-1.5-flash", ModelID: "gemini-1.5-flash"},
				{Provider: "vertex", ModelPath: []string{"/google/deployments/imagen-3/images/generations"}, Creator: "google", ModelName: "imagen-3", ModelID: "imagen-3"},
			},
			wantErr: false,
		},
		{
			name:       "no GCP models",
			configFile: "test-config.yaml",
			mapping: &Mapping{
				Models: []Model{
					{Name: "azure-model", ModelUrl: "/azure/path", Infrastructure: "azure"},
					{Name: "aws-model", ModelUrl: "/aws/path", Infrastructure: "aws"},
				},
			},
			mappingErr:     nil,
			expectedModels: []ModelInfo{},
			wantErr:        false,
		},
		{
			name:           "error in retrieveMapping",
			configFile:     "invalid-config.yaml",
			mapping:        nil,
			mappingErr:     fmt.Errorf("failed to retrieve mapping"),
			expectedModels: nil,
			wantErr:        true,
		},
		{
			name:       "empty model list",
			configFile: "empty-config.yaml",
			mapping: &Mapping{
				Models: []Model{},
			},
			mappingErr:     nil,
			expectedModels: []ModelInfo{},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the environment variable for testing
			origVal := os.Getenv("CONFIGURATION_FILE")
			os.Setenv("CONFIGURATION_FILE", tt.configFile)
			defer os.Setenv("CONFIGURATION_FILE", origVal)

			// Mock retrieveMapping
			retrieveMapping = func(ctx context.Context, path string) (*Mapping, error) {
				assert.Equal(t, tt.configFile, path)
				if tt.mappingErr != nil {
					return nil, tt.mappingErr
				}
				return tt.mapping, nil
			}

			// Mock LoadModelMetadataFromFile to return test data
			LoadModelMetadataFromFile = func(path string) (map[string]ModelMetadata, error) {
				return map[string]ModelMetadata{
					"gemini-1.5-flash": {ModelID: "gemini-1.5-flash"},
					"imagen-3":         {ModelID: "imagen-3"},
				}, nil
			}

			// Call the function
			got, err := fetchGCPModelsImpl(context.Background())

			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Check models
			assert.Equal(t, tt.expectedModels, got)
		})
	}
}

func Test_fetchAWSModels(t *testing.T) {
	originalGetInfraModels := infra.GetInfraModelsForContext
	defer func() { GetInfraModelsForContext = originalGetInfraModels }()

	tests := []struct {
		name          string
		mockConfigs   []infra.ModelConfig
		mockError     error
		expectedModel []ModelInfo
		wantErr       bool
	}{
		{
			name: "successful fetch with multiple models",
			mockConfigs: []infra.ModelConfig{
				{
					Path:         "/amazon/chat/model1",
					ModelMapping: "claude-v1",
					TargetApi:    "/chat/completions",
				},
				{
					Path:         "/amazon/image/model2",
					ModelMapping: "stable-diffusion-v2",
					TargetApi:    "/images/generations",
				},
			},
			expectedModel: []ModelInfo{
				{
					Provider:  "bedrock",
					ModelPath: []string{"/amazon/deployments/claude-v1/chat/completions"},
					Creator:   "amazon",
					ModelName: "claude-v1",
					ModelID:   "",
				},
				{
					Provider:  "bedrock",
					ModelPath: []string{"/amazon/deployments/stable-diffusion-v2/images/generations"},
					Creator:   "amazon",
					ModelName: "stable-diffusion-v2",
					ModelID:   "",
				},
			},
			wantErr: false,
		},
		{
			name:          "empty config list",
			mockConfigs:   []infra.ModelConfig{},
			expectedModel: []ModelInfo{},
			wantErr:       false,
		},
		{
			name:          "error from GetInfraModelsForContext",
			mockConfigs:   nil,
			mockError:     fmt.Errorf("infrastructure error"),
			expectedModel: nil,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the GetInfraModelsForContext function
			GetInfraModelsForContext = func(ctx context.Context) ([]infra.ModelConfig, error) {
				return tt.mockConfigs, tt.mockError
			}

			// Execute test
			got, err := fetchAWSModels(context.Background())

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchAWSModels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check results
			if !tt.wantErr {
				assert.Equal(t, tt.expectedModel, got)
			}
		})
	}
}

func TestEnrichModels(t *testing.T) {
	tests := []struct {
		name            string
		input           []ModelInfo
		metadataFile    string
		metadataContent string
		want            []ModelInfo
	}{
		{
			name: "Successfully enrich models with metadata",
			input: []ModelInfo{
				{ModelName: "test-model"},
				{ModelName: "another-model"},
			},
			metadataContent: `
test-model:
  model_name: test-model
  model_description: Test model description
  model_label: Test Model
  type: chat
  lifecycle: active
  availableRegions: ["us-east", "eu-west"]
  deprecation_date: "2026-01-01"
  fallback_model: fallback-model
  model_capabilities:
    input_modalities: ["text"]
    maxInputTokens: 4096
`,
			want: []ModelInfo{
				{
					ModelName:       "test-model",
					Description:     "Test model description",
					Name:            "test-model", // Generated from modelName
					ModelLabel:      "Test Model",
					Type:            "chat",
					Lifecycle:       "active",
					DeprecationDate: "2026-01-01",
					// New Autopilot-aligned fields
					DeprecationInfo: DeprecationInfo{
						IsDeprecated:             false,
						ScheduledDeprecationDate: "2026-01-01",
					},
					SupportedCapabilities: SupportedCapabilities{
						Streaming:               false,
						Multimodal:              nil, // nil slice
						Functions:               false,
						ParallelFunctionCalling: false,
						JSONMode:                false,
						IsMultimodal:            false,
					},
					Parameters: map[string]ParameterSpec{},
				},
			},
		},
		{
			name: "Handle missing metadata file",
			input: []ModelInfo{
				{ModelName: "test-model"},
			},
			metadataFile: "/nonexistent/path/metadata.yaml",
			want:         []ModelInfo{},
		},
		{
			name:  "Handle empty input models",
			input: []ModelInfo{},
			metadataContent: `
test-model:
  model_name: test-model
`,
			want: []ModelInfo{},
		},
		{
			name: "Handle invalid metadata content",
			input: []ModelInfo{
				{ModelName: "test-model"},
			},
			metadataContent: "invalid: yaml: content",
			want:            []ModelInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context for testing
			ctx := context.Background()

			// Mock the logger using function variable
			logger := zap.NewNop()
			oldLogger := loggerFromContext
			loggerFromContext = func(context.Context) *zap.Logger {
				return logger
			}
			defer func() { loggerFromContext = oldLogger }()

			// Create temporary metadata file if content is provided
			var tempFile string
			if tt.metadataContent != "" {
				file, err := os.CreateTemp("", "model-metadata-*.yaml")
				require.NoError(t, err)
				defer os.Remove(file.Name())

				_, err = file.WriteString(tt.metadataContent)
				require.NoError(t, err)
				file.Close()
				tempFile = file.Name()
			} else {
				tempFile = tt.metadataFile
			}

			// Mock LoadModelMetadataFromFile by replacing the path
			oldLoadMetadata := LoadModelMetadataFromFile
			defer func() { LoadModelMetadataFromFile = oldLoadMetadata }()
			LoadModelMetadataFromFile = func(path string) (map[string]ModelMetadata, error) {
				return oldLoadMetadata(tempFile)
			}

			// Call enrichModels with context parameter
			got := enrichModels(ctx, tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// MockContextChecker implements ContextChecker for testing
type MockContextChecker struct {
	mock.Mock
}

func (m *MockContextChecker) IsUseGenAiInfraModels(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockContextChecker) IsUseGCPVertex(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockContextChecker) IsUseAzureGenAIURL(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockContextChecker) AzureGenAIURL(ctx context.Context) string {
	args := m.Called(ctx)
	return args.String(0)
}

func (m *MockContextChecker) IsLLMProviderConfigured(ctx context.Context, provider string) bool {
	args := m.Called(ctx, provider)
	return args.Bool(0)
}

func (m *MockContextChecker) LoggerFromContext(ctx context.Context) *zap.Logger {
	args := m.Called(ctx)
	return args.Get(0).(*zap.Logger)
}

func (m *MockContextChecker) ContextWithGinContext(ctx context.Context, gc *gin.Context) context.Context {
	args := m.Called(ctx, gc)
	return args.Get(0).(context.Context)
}

func TestExtractDefaultModelsImpl(t *testing.T) {
	tests := []struct {
		name           string
		models         []ModelInfo
		defaults       infra.DefaultModelConfig
		wantFastModel  *ModelInfo
		wantSmartModel *ModelInfo
		wantProModel   *ModelInfo
	}{
		{
			name: "all three models found",
			models: []ModelInfo{
				{ModelName: "model1", ModelMappingId: "model1", Provider: "bedrock"},
				{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
				{ModelName: "smart-model", ModelMappingId: "smart-model", Provider: "vertex"},
				{ModelName: "pro-model", ModelMappingId: "pro-model", Provider: "bedrock"},
			},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
			wantFastModel:  &ModelInfo{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
			wantSmartModel: &ModelInfo{ModelName: "smart-model", ModelMappingId: "smart-model", Provider: "vertex"},
			wantProModel:   &ModelInfo{ModelName: "pro-model", ModelMappingId: "pro-model", Provider: "bedrock"},
		},
		{
			name: "both models found (legacy - no pro)",
			models: []ModelInfo{
				{ModelName: "model1", ModelMappingId: "model1", Provider: "bedrock"},
				{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
				{ModelName: "smart-model", ModelMappingId: "smart-model", Provider: "vertex"},
			},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
			},
			wantFastModel:  &ModelInfo{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
			wantSmartModel: &ModelInfo{ModelName: "smart-model", ModelMappingId: "smart-model", Provider: "vertex"},
			wantProModel:   nil,
		},
		{
			name: "only fast model found",
			models: []ModelInfo{
				{ModelName: "model1", ModelMappingId: "model1", Provider: "bedrock"},
				{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
			},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
			wantFastModel:  &ModelInfo{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
			wantSmartModel: nil,
			wantProModel:   nil,
		},
		{
			name: "only smart model found",
			models: []ModelInfo{
				{ModelName: "model1", ModelMappingId: "model1", Provider: "bedrock"},
				{ModelName: "smart-model", ModelMappingId: "smart-model", Provider: "vertex"},
			},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
			wantFastModel:  nil,
			wantSmartModel: &ModelInfo{ModelName: "smart-model", ModelMappingId: "smart-model", Provider: "vertex"},
			wantProModel:   nil,
		},
		{
			name: "only pro model found",
			models: []ModelInfo{
				{ModelName: "model1", ModelMappingId: "model1", Provider: "bedrock"},
				{ModelName: "pro-model", ModelMappingId: "pro-model", Provider: "bedrock"},
			},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
			wantFastModel:  nil,
			wantSmartModel: nil,
			wantProModel:   &ModelInfo{ModelName: "pro-model", ModelMappingId: "pro-model", Provider: "bedrock"},
		},
		{
			name: "fast and pro models found",
			models: []ModelInfo{
				{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
				{ModelName: "pro-model", ModelMappingId: "pro-model", Provider: "bedrock"},
			},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
			wantFastModel:  &ModelInfo{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
			wantSmartModel: nil,
			wantProModel:   &ModelInfo{ModelName: "pro-model", ModelMappingId: "pro-model", Provider: "bedrock"},
		},
		{
			name: "no models found",
			models: []ModelInfo{
				{ModelName: "model1", Provider: "bedrock"},
				{ModelName: "model2", Provider: "vertex"},
			},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
			wantFastModel:  nil,
			wantSmartModel: nil,
			wantProModel:   nil,
		},
		{
			name:   "empty models list",
			models: []ModelInfo{},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
			wantFastModel:  nil,
			wantSmartModel: nil,
			wantProModel:   nil,
		},
		{
			name: "pro configured but empty string",
			models: []ModelInfo{
				{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
				{ModelName: "smart-model", ModelMappingId: "smart-model", Provider: "vertex"},
				{ModelName: "pro-model", ModelMappingId: "pro-model", Provider: "bedrock"},
			},
			defaults: infra.DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "", // Empty pro configuration
			},
			wantFastModel:  &ModelInfo{ModelName: "fast-model", ModelMappingId: "fast-model", Provider: "bedrock"},
			wantSmartModel: &ModelInfo{ModelName: "smart-model", ModelMappingId: "smart-model", Provider: "vertex"},
			wantProModel:   nil, // Should not match when Pro is empty
		},
		// Additional test cases for prefix matching
		{
			name: "both exact and prefix matches available, prefer exact",
			models: []ModelInfo{
				{ModelID: "model1", ModelName: "gpt-4o", ModelMappingId: "gpt-4o", Provider: "bedrock"},
				{ModelID: "model2", ModelName: "gpt-4o-turbo", ModelMappingId: "gpt-4o-turbo", Provider: "bedrock"},
				{ModelID: "model3", ModelName: "text-embedding-ada-002", ModelMappingId: "text-embedding-ada-002", Provider: "bedrock"},
				{ModelID: "model4", ModelName: "text-embedding-ada-002-v2", ModelMappingId: "text-embedding-ada-002-v2", Provider: "bedrock"},
			},
			defaults: infra.DefaultModelConfig{
				Smart: "gpt-4o",
				Fast:  "text-embedding-ada-002",
			},
			wantFastModel:  &ModelInfo{ModelID: "model3", ModelName: "text-embedding-ada-002", ModelMappingId: "text-embedding-ada-002", Provider: "bedrock"},
			wantSmartModel: &ModelInfo{ModelID: "model1", ModelName: "gpt-4o", ModelMappingId: "gpt-4o", Provider: "bedrock"},
		},
		{
			name: "prefix matches only",
			models: []ModelInfo{
				{ModelID: "model1", ModelName: "gpt-4o-1234", ModelMappingId: "gpt-4o-1234", Provider: "bedrock"},
				{ModelID: "model2", ModelName: "text-embedding-ada-002-5678", ModelMappingId: "text-embedding-ada-002-5678", Provider: "bedrock"},
			},
			defaults: infra.DefaultModelConfig{
				Smart: "gpt-4o",
				Fast:  "text-embedding-ada-002",
				Pro:   "claude-sonnet-4-5",
			},
			// The implementation only uses strings.EqualFold, not strings.HasPrefix
			// so prefix matches don't work - expect nil for all
			wantFastModel:  nil,
			wantSmartModel: nil,
			wantProModel:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDefaultModels(tt.models, &tt.defaults)
			assert.Equal(t, tt.wantFastModel, got.Fast)
			assert.Equal(t, tt.wantSmartModel, got.Smart)
			assert.Equal(t, tt.wantProModel, got.Pro)
		})
	}
}

func Test_calculateModelPaths(t *testing.T) {
	tests := []struct {
		name     string
		creator  string
		modelID  string
		endpoint string
		apis     []string
		want     []string
	}{
		{
			name:     "Bedrock creator with multiple APIs",
			creator:  "bedrock",
			modelID:  "claude-v2",
			endpoint: "",
			apis:     []string{"/chat/completions", "/embeddings"},
			want:     []string{"/bedrock/deployments/claude-v2/chat/completions", "/bedrock/deployments/claude-v2/embeddings"},
		},
		{
			name:     "Anthropic creator with single API",
			creator:  "anthropic",
			modelID:  "claude-3-opus",
			endpoint: "",
			apis:     []string{"/chat/completions"},
			want:     []string{"/anthropic/deployments/claude-3-opus/chat/completions"},
		},
		{
			name:     "Anthropic creator with API without leading slash",
			creator:  "anthropic",
			modelID:  "claude-3-haiku",
			endpoint: "",
			apis:     []string{"converse"},
			want:     []string{"/anthropic/deployments/claude-3-haiku/converse"},
		},
		{
			name:     "Meta creator with single API",
			creator:  "meta",
			modelID:  "llama-3-70b",
			endpoint: "",
			apis:     []string{"/chat/completions"},
			want:     []string{"/meta/deployments/llama-3-70b/chat/completions"},
		},
		{
			name:     "Google/Vertex creator with multiple APIs",
			creator:  "google",
			modelID:  "gemini-pro",
			endpoint: "",
			apis:     []string{"/chat/completions", "/images/generations"},
			want:     []string{"/google/deployments/gemini-pro/chat/completions", "/google/deployments/gemini-pro/images/generations"},
		},
		{
			name:     "Azure OpenAI with endpoint",
			creator:  "azure",
			modelID:  "gpt-4",
			endpoint: "/v1/completions",
			apis:     []string{"/chat/completions"},
			want:     []string{"/openai/v1/completions"},
		},
		{
			name:     "Azure OpenAI without endpoint",
			creator:  "azure",
			modelID:  "gpt-4",
			endpoint: "",
			apis:     []string{"/chat/completions", "/embeddings"},
			want:     []string{"/openai/deployments/gpt-4/chat/completions", "/openai/deployments/gpt-4/embeddings"},
		},
		{
			name:     "Unknown creator",
			creator:  "somecompany",
			modelID:  "custom-model",
			endpoint: "",
			apis:     []string{"/chat/completions"},
			want:     []string{},
		},
		{
			name:     "Empty creator",
			creator:  "",
			modelID:  "model-x",
			endpoint: "",
			apis:     []string{"/chat/completions"},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got := calculateModelPaths(ctx, tt.creator, tt.modelID, tt.endpoint, tt.apis)
			assert.Equal(t, tt.want, got, "calculateModelPaths returned unexpected result")
		})
	}
}

// getTestMetadataPath resolves the absolute path to model-metadata-test.yaml
func getTestMetadataPath() string {
	_, currentFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(currentFile)
	return filepath.Join(baseDir, "..", "..", "..", "test", "helpers", "model-metadata-test.yaml")
}

func loadModelMetadataSchema() (gojsonschema.JSONLoader, error) {
	_, currentFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(currentFile)
	schemaPath := filepath.Join(baseDir, "..", "..", "..", "test", "helpers", "modelMetadataSchema.json")

	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	return gojsonschema.NewBytesLoader(schemaContent), nil
}

func ValidateModelMetadataFromFile(path string) ([]string, error) {
	yamlContent, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(yamlContent, &yamlData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	jsonData, err := json.Marshal(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	schemaLoader, err := loadModelMetadataSchema()
	if err != nil {
		return nil, err
	}
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %w", err)
	}

	var validationErrors []string
	if !result.Valid() {
		for _, desc := range result.Errors() {
			validationErrors = append(validationErrors, desc.String())
		}
	}

	// Strict date validation
	for modelKey, entry := range yamlData {
		if modelMap, ok := entry.(map[string]interface{}); ok {
			if dateStr, ok := modelMap["deprecationDate"].(string); ok && dateStr != "" && dateStr != "NA" {
				if _, err := time.Parse("2006-01-02", dateStr); err != nil {
					validationErrors = append(validationErrors, fmt.Sprintf("model '%s': invalid deprecationDate '%s' (must be YYYY-MM-DD)", modelKey, dateStr))
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors, nil
	}

	return nil, nil
}

func TestValidateModelMetadataFile(t *testing.T) {
	fPath := getTestMetadataPath()
	validationErrors, err := ValidateModelMetadataFromFile(fPath)
	require.NoError(t, err)

	if len(validationErrors) > 0 {
		t.Errorf("Validation failed with %d error(s):", len(validationErrors))
		for _, e := range validationErrors {
			t.Logf(" - %s", e)
		}
	}
}

// TestSetInputTokensFromParam tests the new setInputTokensFromParam function
func TestSetInputTokensFromParam(t *testing.T) {
	tests := []struct {
		name          string
		param         ParameterSpec
		expectedValue *int
	}{
		{
			name: "Maximum value set",
			param: ParameterSpec{
				Maximum: floatPtr(4096.0),
			},
			expectedValue: intPtr(4096),
		},
		{
			name: "Maximum value with decimal",
			param: ParameterSpec{
				Maximum: floatPtr(8192.5),
			},
			expectedValue: intPtr(8192),
		},
		{
			name: "No maximum value",
			param: ParameterSpec{
				Maximum: nil,
			},
			expectedValue: nil,
		},
		{
			name: "Zero maximum value",
			param: ParameterSpec{
				Maximum: floatPtr(0.0),
			},
			expectedValue: intPtr(0),
		},
		{
			name: "Negative maximum value",
			param: ParameterSpec{
				Maximum: floatPtr(-1.0),
			},
			expectedValue: intPtr(-1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ModelInfo{}
			setInputTokensFromParam(model, tt.param)

			if tt.expectedValue == nil {
				assert.Nil(t, model.InputTokens)
			} else {
				assert.NotNil(t, model.InputTokens)
				assert.Equal(t, *tt.expectedValue, *model.InputTokens)
			}
		})
	}
}

// TestSetOutputTokensFromParam tests the new setOutputTokensFromParam function
func TestSetOutputTokensFromParam(t *testing.T) {
	tests := []struct {
		name          string
		param         ParameterSpec
		expectedValue *int
	}{
		{
			name: "Maximum value set - priority 1",
			param: ParameterSpec{
				Maximum: floatPtr(2048.0),
				Default: 1024.0,
			},
			expectedValue: intPtr(2048),
		},
		{
			name: "No maximum, float64 default - priority 2",
			param: ParameterSpec{
				Maximum: nil,
				Default: 1024.0,
			},
			expectedValue: intPtr(1024),
		},
		{
			name: "No maximum, int default - priority 2",
			param: ParameterSpec{
				Maximum: nil,
				Default: 512,
			},
			expectedValue: intPtr(512),
		},
		{
			name: "No maximum, string default - unsupported type",
			param: ParameterSpec{
				Maximum: nil,
				Default: "1024",
			},
			expectedValue: nil,
		},
		{
			name: "No maximum, nil default",
			param: ParameterSpec{
				Maximum: nil,
				Default: nil,
			},
			expectedValue: nil,
		},
		{
			name: "Maximum value with decimal",
			param: ParameterSpec{
				Maximum: floatPtr(4096.7),
				Default: 2048.0,
			},
			expectedValue: intPtr(4096),
		},
		{
			name: "Zero maximum value",
			param: ParameterSpec{
				Maximum: floatPtr(0.0),
			},
			expectedValue: intPtr(0),
		},
		{
			name: "Negative maximum value",
			param: ParameterSpec{
				Maximum: floatPtr(-1.0),
			},
			expectedValue: intPtr(-1),
		},
		{
			name: "Bool default - unsupported type",
			param: ParameterSpec{
				Maximum: nil,
				Default: true,
			},
			expectedValue: nil,
		},
		{
			name: "Slice default - unsupported type",
			param: ParameterSpec{
				Maximum: nil,
				Default: []string{"test"},
			},
			expectedValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ModelInfo{}
			setOutputTokensFromParam(model, tt.param)

			if tt.expectedValue == nil {
				assert.Nil(t, model.OutputTokens)
			} else {
				assert.NotNil(t, model.OutputTokens)
				assert.Equal(t, *tt.expectedValue, *model.OutputTokens)
			}
		})
	}
}

// TestSetInputTokensFromParameters tests the new setInputTokensFromParameters function
func TestSetInputTokensFromParameters(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]ParameterSpec
		expectedInput *int
	}{
		{
			name: "maxInputTokens parameter",
			params: map[string]ParameterSpec{
				"maxInputTokens": {
					Maximum: floatPtr(4096.0),
				},
			},
			expectedInput: intPtr(4096),
		},
		{
			name: "max_input_tokens parameter",
			params: map[string]ParameterSpec{
				"max_input_tokens": {
					Maximum: floatPtr(8192.0),
				},
			},
			expectedInput: intPtr(8192),
		},
		{
			name: "Case insensitive parameter matching",
			params: map[string]ParameterSpec{
				"MAXINPUTTOKENS": {
					Maximum: floatPtr(16384.0),
				},
			},
			expectedInput: intPtr(16384),
		},
		{
			name: "No matching parameters",
			params: map[string]ParameterSpec{
				"temperature": {
					Maximum: floatPtr(1.0),
				},
			},
			expectedInput: nil,
		},
		{
			name:          "Empty parameters",
			params:        map[string]ParameterSpec{},
			expectedInput: nil,
		},
		{
			name:          "Nil parameters",
			params:        nil,
			expectedInput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ModelInfo{}
			setInputTokensFromParameters(context.Background(), model, tt.params)

			if tt.expectedInput == nil {
				assert.Nil(t, model.InputTokens)
			} else {
				assert.NotNil(t, model.InputTokens)
				assert.Equal(t, *tt.expectedInput, *model.InputTokens)
			}
		})
	}
}

// TestSetOutputTokensFromParameters tests the setOutputTokensFromParameters function with precedence rules
func TestSetOutputTokensFromParameters(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]ParameterSpec
		expectedOutput *int
	}{
		{
			name: "max_tokens parameter",
			params: map[string]ParameterSpec{
				"max_tokens": {
					Default: 2048,
				},
			},
			expectedOutput: intPtr(2048),
		},
		{
			name: "max_completion_tokens parameter",
			params: map[string]ParameterSpec{
				"max_completion_tokens": {
					Maximum: floatPtr(128000.0),
				},
			},
			expectedOutput: intPtr(128000),
		},
		{
			name: "max_completion_tokens with default value",
			params: map[string]ParameterSpec{
				"max_completion_tokens": {
					Default: 16384.0,
				},
			},
			expectedOutput: intPtr(16384),
		},
		{
			name: "Both max_tokens and max_completion_tokens (max_completion_tokens takes precedence)",
			params: map[string]ParameterSpec{
				"max_tokens": {
					Maximum: floatPtr(4096.0),
				},
				"max_completion_tokens": {
					Maximum: floatPtr(8192.0),
				},
			},
			expectedOutput: intPtr(8192), // max_completion_tokens takes precedence
		},
		{
			name: "No matching parameters",
			params: map[string]ParameterSpec{
				"temperature": {
					Maximum: floatPtr(1.0),
				},
			},
			expectedOutput: nil,
		},
		{
			name:           "Empty parameters",
			params:         map[string]ParameterSpec{},
			expectedOutput: nil,
		},
		{
			name:           "Nil parameters",
			params:         nil,
			expectedOutput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &ModelInfo{}
			setOutputTokensFromParameters(context.Background(), model, tt.params)

			if tt.expectedOutput == nil {
				assert.Nil(t, model.OutputTokens)
			} else {
				assert.NotNil(t, model.OutputTokens)
				assert.Equal(t, *tt.expectedOutput, *model.OutputTokens)
			}
		})
	}
}

// TestModelDescriptionFromMetadataInModelsResponse tests the full enrichment pipeline
// from metadata YAML -> enrichment -> API response, ensuring descriptions populate correctly
func TestModelDescriptionFromMetadataInModelsResponse(t *testing.T) {
	tests := []struct {
		name                string
		inputModels         []ModelInfo
		metadataContent     string
		expectedDescription string
		expectInOutput      bool
	}{
		{
			name: "Model with description in metadata appears in enriched output",
			inputModels: []ModelInfo{
				{ModelName: "gpt-4", Provider: "azure", Creator: "openai"},
			},
			metadataContent: `
gpt-4:
  model_name: GPT-4
  model_description: OpenAI Chat Completions model
  model_label: GPT-4
  type: chat
  version: "1"
`,
			expectedDescription: "OpenAI Chat Completions model",
			expectInOutput:      true,
		},
		{
			name: "Model without description in metadata has empty description",
			inputModels: []ModelInfo{
				{ModelName: "claude-3", Provider: "bedrock", Creator: "anthropic"},
			},
			metadataContent: `
claude-3:
  model_name: Claude-3
  model_label: Claude 3
  type: chat
  version: "3"
`,
			expectedDescription: "",
			expectInOutput:      false,
		},
		{
			name: "Multiple models with different descriptions",
			inputModels: []ModelInfo{
				{ModelName: "gemini-pro", Provider: "vertex", Creator: "google"},
			},
			metadataContent: `
gemini-pro:
  model_name: Gemini-Pro
  model_description: Gemini Chat Completions model for advanced reasoning
  model_label: Gemini Pro
  type: chat
  version: "1.5"
`,
			expectedDescription: "Gemini Chat Completions model for advanced reasoning",
			expectInOutput:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary metadata file
			tmpFile, err := os.CreateTemp("", "model-metadata-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.metadataContent)
			require.NoError(t, err)
			tmpFile.Close()

			// Mock LoadModelMetadataFromFile
			originalLoad := LoadModelMetadataFromFile
			defer func() { LoadModelMetadataFromFile = originalLoad }()

			LoadModelMetadataFromFile = func(path string) (map[string]ModelMetadata, error) {
				return originalLoad(tmpFile.Name())
			}

			// Create context and enrich models
			ctx := context.Background()
			enrichedModels := enrichModels(ctx, tt.inputModels)

			// Verify the enriched models
			require.Len(t, enrichedModels, len(tt.inputModels), "Expected same number of models after enrichment")

			if len(enrichedModels) > 0 {
				enrichedModel := enrichedModels[0]

				if tt.expectInOutput {
					assert.Equal(t, tt.expectedDescription, enrichedModel.Description,
						"Description should match metadata model_description")
				} else {
					assert.Empty(t, enrichedModel.Description,
						"Description should be empty when not in metadata")
				}

				// Marshal to JSON and verify description field
				jsonData, err := json.Marshal(enrichedModel)
				require.NoError(t, err, "Failed to marshal enriched model to JSON")

				var jsonMap map[string]interface{}
				err = json.Unmarshal(jsonData, &jsonMap)
				require.NoError(t, err, "Failed to unmarshal JSON")

				// Verify description exists in JSON
				description, exists := jsonMap["description"]
				assert.True(t, exists, "Description field should exist in JSON output")

				if tt.expectInOutput {
					descStr, ok := description.(string)
					assert.True(t, ok, "Description should be a string in JSON")
					assert.Equal(t, tt.expectedDescription, descStr,
						"JSON description should match expected value")
				}
			}
		})
	}
}

// TestShouldDisplayPreviewModels tests the shouldDisplayPreviewModels function
func TestShouldDisplayPreviewModels(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "true returns true",
			envValue: "true",
			expected: true,
		},
		{
			name:     "TRUE returns true (case insensitive)",
			envValue: "TRUE",
			expected: true,
		},
		{
			name:     "True returns true (case insensitive)",
			envValue: "True",
			expected: true,
		},
		{
			name:     "TrUe returns true (case insensitive)",
			envValue: "TrUe",
			expected: true,
		},
		{
			name:     "false returns false",
			envValue: "false",
			expected: false,
		},
		{
			name:     "FALSE returns false",
			envValue: "FALSE",
			expected: false,
		},
		{
			name:     "empty string returns false",
			envValue: "",
			expected: false,
		},
		{
			name:     "random string returns false",
			envValue: "yes",
			expected: false,
		},
		{
			name:     "1 returns false",
			envValue: "1",
			expected: false,
		},
		{
			name:     "0 returns false",
			envValue: "0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalValue := os.Getenv("DISPLAY_PREVIEW_MODELS")
			defer os.Setenv("DISPLAY_PREVIEW_MODELS", originalValue)

			// Set the test value
			os.Setenv("DISPLAY_PREVIEW_MODELS", tt.envValue)

			result := shouldDisplayPreviewModels()
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test when environment variable is not set
	t.Run("unset returns false", func(t *testing.T) {
		// Save original value
		originalValue, wasSet := os.LookupEnv("DISPLAY_PREVIEW_MODELS")
		os.Unsetenv("DISPLAY_PREVIEW_MODELS")
		defer func() {
			if wasSet {
				os.Setenv("DISPLAY_PREVIEW_MODELS", originalValue)
			}
		}()

		result := shouldDisplayPreviewModels()
		assert.False(t, result)
	})
}

// TestIsPreviewModel tests the isPreviewModel function
func TestIsPreviewModel(t *testing.T) {
	tests := []struct {
		name      string
		lifecycle string
		expected  bool
	}{
		{
			name:      "Preview returns true",
			lifecycle: "Preview",
			expected:  true,
		},
		{
			name:      "preview returns true (lowercase)",
			lifecycle: "preview",
			expected:  true,
		},
		{
			name:      "PREVIEW returns true (uppercase)",
			lifecycle: "PREVIEW",
			expected:  true,
		},
		{
			name:      "PrEvIeW returns true (mixed case)",
			lifecycle: "PrEvIeW",
			expected:  true,
		},
		{
			name:      "Generally Available returns false",
			lifecycle: "Generally Available",
			expected:  false,
		},
		{
			name:      "Deprecated returns false",
			lifecycle: "Deprecated",
			expected:  false,
		},
		{
			name:      "Nearing Deprecation returns false",
			lifecycle: "Nearing Deprecation",
			expected:  false,
		},
		{
			name:      "empty string returns false",
			lifecycle: "",
			expected:  false,
		},
		{
			name:      "active returns false",
			lifecycle: "active",
			expected:  false,
		},
		{
			name:      "preview-beta returns false (not exact match)",
			lifecycle: "preview-beta",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPreviewModel(tt.lifecycle)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestEnrichModelsImpl_PreviewFiltering tests that Preview models are filtered correctly
func TestEnrichModelsImpl_PreviewFiltering(t *testing.T) {
	tests := []struct {
		name                     string
		displayPreviewModelsEnv  string
		inputModels              []ModelInfo
		metadataContent          string
		expectedModelNames       []string
		expectedFilteredOutCount int
	}{
		{
			name:                    "Preview models filtered when DISPLAY_PREVIEW_MODELS=false",
			displayPreviewModelsEnv: "false",
			inputModels: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "preview-model", Provider: "bedrock"},
			},
			metadataContent: `
ga-model:
  model_name: GA Model
  lifecycle: Generally Available
preview-model:
  model_name: Preview Model
  lifecycle: Preview
`,
			expectedModelNames:       []string{"GA Model"},
			expectedFilteredOutCount: 1,
		},
		{
			name:                    "Preview models included when DISPLAY_PREVIEW_MODELS=true",
			displayPreviewModelsEnv: "true",
			inputModels: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "preview-model", Provider: "bedrock"},
			},
			metadataContent: `
ga-model:
  model_name: GA Model
  lifecycle: Generally Available
preview-model:
  model_name: Preview Model
  lifecycle: Preview
`,
			expectedModelNames:       []string{"GA Model", "Preview Model"},
			expectedFilteredOutCount: 0,
		},
		{
			name:                    "Preview models filtered when DISPLAY_PREVIEW_MODELS not set",
			displayPreviewModelsEnv: "",
			inputModels: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "preview-model", Provider: "bedrock"},
			},
			metadataContent: `
ga-model:
  model_name: GA Model
  lifecycle: Generally Available
preview-model:
  model_name: Preview Model
  lifecycle: Preview
`,
			expectedModelNames:       []string{"GA Model"},
			expectedFilteredOutCount: 1,
		},
		{
			name:                    "Case insensitive lifecycle check - PREVIEW uppercase filtered",
			displayPreviewModelsEnv: "false",
			inputModels: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "preview-upper", Provider: "bedrock"},
			},
			metadataContent: `
ga-model:
  model_name: GA Model
  lifecycle: Generally Available
preview-upper:
  model_name: Preview Upper
  lifecycle: PREVIEW
`,
			expectedModelNames:       []string{"GA Model"},
			expectedFilteredOutCount: 1,
		},
		{
			name:                    "Case insensitive lifecycle check - preview lowercase filtered",
			displayPreviewModelsEnv: "false",
			inputModels: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "preview-lower", Provider: "bedrock"},
			},
			metadataContent: `
ga-model:
  model_name: GA Model
  lifecycle: Generally Available
preview-lower:
  model_name: Preview Lower
  lifecycle: preview
`,
			expectedModelNames:       []string{"GA Model"},
			expectedFilteredOutCount: 1,
		},
		{
			name:                    "Non-Preview models unaffected regardless of env var",
			displayPreviewModelsEnv: "false",
			inputModels: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "deprecated-model", Provider: "bedrock"},
				{ModelName: "nearing-deprecation-model", Provider: "bedrock"},
			},
			metadataContent: `
ga-model:
  model_name: GA Model
  lifecycle: Generally Available
deprecated-model:
  model_name: Deprecated Model
  lifecycle: Deprecated
nearing-deprecation-model:
  model_name: Nearing Deprecation Model
  lifecycle: Nearing Deprecation
`,
			expectedModelNames:       []string{"GA Model", "Deprecated Model", "Nearing Deprecation Model"},
			expectedFilteredOutCount: 0,
		},
		{
			name:                    "Multiple Preview models filtered",
			displayPreviewModelsEnv: "false",
			inputModels: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "preview-model-1", Provider: "bedrock"},
				{ModelName: "preview-model-2", Provider: "vertex"},
			},
			metadataContent: `
ga-model:
  model_name: GA Model
  lifecycle: Generally Available
preview-model-1:
  model_name: Preview Model 1
  lifecycle: Preview
preview-model-2:
  model_name: Preview Model 2
  lifecycle: Preview
`,
			expectedModelNames:       []string{"GA Model"},
			expectedFilteredOutCount: 2,
		},
		{
			name:                    "DISPLAY_PREVIEW_MODELS=TRUE works (case insensitive)",
			displayPreviewModelsEnv: "TRUE",
			inputModels: []ModelInfo{
				{ModelName: "preview-model", Provider: "bedrock"},
			},
			metadataContent: `
preview-model:
  model_name: Preview Model
  lifecycle: Preview
`,
			expectedModelNames:       []string{"Preview Model"},
			expectedFilteredOutCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env value
			originalValue, wasSet := os.LookupEnv("DISPLAY_PREVIEW_MODELS")
			defer func() {
				if wasSet {
					os.Setenv("DISPLAY_PREVIEW_MODELS", originalValue)
				} else {
					os.Unsetenv("DISPLAY_PREVIEW_MODELS")
				}
			}()

			// Set the test env value
			if tt.displayPreviewModelsEnv == "" {
				os.Unsetenv("DISPLAY_PREVIEW_MODELS")
			} else {
				os.Setenv("DISPLAY_PREVIEW_MODELS", tt.displayPreviewModelsEnv)
			}

			// Create temporary metadata file
			tmpFile, err := os.CreateTemp("", "model-metadata-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.metadataContent)
			require.NoError(t, err)
			tmpFile.Close()

			// Mock LoadModelMetadataFromFile
			originalLoad := LoadModelMetadataFromFile
			defer func() { LoadModelMetadataFromFile = originalLoad }()
			LoadModelMetadataFromFile = func(path string) (map[string]ModelMetadata, error) {
				return originalLoad(tmpFile.Name())
			}

			// Mock logger
			logger := zap.NewNop()
			oldLogger := loggerFromContext
			loggerFromContext = func(context.Context) *zap.Logger {
				return logger
			}
			defer func() { loggerFromContext = oldLogger }()

			// Call enrichModels
			ctx := context.Background()
			result := enrichModels(ctx, tt.inputModels)

			// Extract model names from result
			var resultModelNames []string
			for _, m := range result {
				resultModelNames = append(resultModelNames, m.ModelName)
			}

			// Sort both slices for consistent comparison
			sort.Strings(resultModelNames)
			sort.Strings(tt.expectedModelNames)

			assert.Equal(t, tt.expectedModelNames, resultModelNames,
				"Expected models: %v, got: %v", tt.expectedModelNames, resultModelNames)
		})
	}
}

// TestGetModelIdentifier tests the getModelIdentifier function
func TestGetModelIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		model    *ModelInfo
		expected string
	}{
		{
			name: "Model with ModelName returns ModelName",
			model: &ModelInfo{
				ModelName: "gpt-4",
				Provider:  "azure",
				Creator:   "openai",
				ModelID:   "gpt-4-id",
			},
			expected: "gpt-4",
		},
		{
			name: "Model without ModelName returns formatted string",
			model: &ModelInfo{
				ModelName: "",
				Provider:  "bedrock",
				Creator:   "anthropic",
				ModelID:   "claude-3",
			},
			expected: "Provider:bedrock Creator:anthropic ModelID:claude-3",
		},
		{
			name: "Empty model returns formatted string with empty fields",
			model: &ModelInfo{
				ModelName: "",
				Provider:  "",
				Creator:   "",
				ModelID:   "",
			},
			expected: "Provider: Creator: ModelID:",
		},
		{
			name: "Model with only ModelName set",
			model: &ModelInfo{
				ModelName: "test-model",
			},
			expected: "test-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModelIdentifier(tt.model)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestShouldFilterPreviewModel tests the shouldFilterPreviewModel function
func TestShouldFilterPreviewModel(t *testing.T) {
	tests := []struct {
		name           string
		lifecycle      string
		displayPreview bool
		expected       bool
	}{
		{
			name:           "Preview lifecycle with displayPreview false should filter",
			lifecycle:      "Preview",
			displayPreview: false,
			expected:       true,
		},
		{
			name:           "Preview lifecycle with displayPreview true should not filter",
			lifecycle:      "Preview",
			displayPreview: true,
			expected:       false,
		},
		{
			name:           "preview lowercase with displayPreview false should filter",
			lifecycle:      "preview",
			displayPreview: false,
			expected:       true,
		},
		{
			name:           "PREVIEW uppercase with displayPreview false should filter",
			lifecycle:      "PREVIEW",
			displayPreview: false,
			expected:       true,
		},
		{
			name:           "Generally Available lifecycle with displayPreview false should not filter",
			lifecycle:      "Generally Available",
			displayPreview: false,
			expected:       false,
		},
		{
			name:           "Deprecated lifecycle with displayPreview false should not filter",
			lifecycle:      "Deprecated",
			displayPreview: false,
			expected:       false,
		},
		{
			name:           "Empty lifecycle with displayPreview false should not filter",
			lifecycle:      "",
			displayPreview: false,
			expected:       false,
		},
		{
			name:           "Generally Available lifecycle with displayPreview true should not filter",
			lifecycle:      "Generally Available",
			displayPreview: true,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldFilterPreviewModel(tt.lifecycle, tt.displayPreview)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestProcessModelForEnrichment tests the processModelForEnrichment function
func TestProcessModelForEnrichment(t *testing.T) {
	tests := []struct {
		name           string
		model          ModelInfo
		metadata       map[string]ModelMetadata
		displayPreview bool
		expectedReason filterReason
	}{
		{
			name:  "Model with valid metadata should return filterReasonNone",
			model: ModelInfo{ModelName: "test-model", Provider: "bedrock"},
			metadata: map[string]ModelMetadata{
				"test-model": {
					ModelName: "test-model",
					Lifecycle: "Generally Available",
				},
			},
			displayPreview: false,
			expectedReason: filterReasonNone,
		},
		{
			name:           "Model without metadata should return filterReasonNoMetadata",
			model:          ModelInfo{ModelName: "unknown-model", Provider: "bedrock"},
			metadata:       map[string]ModelMetadata{},
			displayPreview: false,
			expectedReason: filterReasonNoMetadata,
		},
		{
			name:  "Preview model with displayPreview false should return filterReasonPreview",
			model: ModelInfo{ModelName: "preview-model", Provider: "bedrock"},
			metadata: map[string]ModelMetadata{
				"preview-model": {
					ModelName: "preview-model",
					Lifecycle: "Preview",
				},
			},
			displayPreview: false,
			expectedReason: filterReasonPreview,
		},
		{
			name:  "Preview model with displayPreview true should return filterReasonNone",
			model: ModelInfo{ModelName: "preview-model", Provider: "bedrock"},
			metadata: map[string]ModelMetadata{
				"preview-model": {
					ModelName: "preview-model",
					Lifecycle: "Preview",
				},
			},
			displayPreview: true,
			expectedReason: filterReasonNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock logger
			logger := zap.NewNop()
			oldLogger := loggerFromContext
			loggerFromContext = func(context.Context) *zap.Logger {
				return logger
			}
			defer func() { loggerFromContext = oldLogger }()

			ctx := context.Background()
			result := processModelForEnrichment(ctx, &tt.model, tt.metadata, tt.displayPreview)
			assert.Equal(t, tt.expectedReason, result)
		})
	}
}

// TestFilterAndEnrichModels tests the filterAndEnrichModels function
func TestFilterAndEnrichModels(t *testing.T) {
	tests := []struct {
		name                         string
		models                       []ModelInfo
		metadata                     map[string]ModelMetadata
		displayPreview               bool
		expectedValidCount           int
		expectedFilteredCount        int
		expectedPreviewFilteredCount int
		expectedValidModelNames      []string
	}{
		{
			name: "All models valid",
			models: []ModelInfo{
				{ModelName: "model-1", Provider: "bedrock"},
				{ModelName: "model-2", Provider: "vertex"},
			},
			metadata: map[string]ModelMetadata{
				"model-1": {ModelName: "model-1", Lifecycle: "Generally Available"},
				"model-2": {ModelName: "model-2", Lifecycle: "Generally Available"},
			},
			displayPreview:               false,
			expectedValidCount:           2,
			expectedFilteredCount:        0,
			expectedPreviewFilteredCount: 0,
			expectedValidModelNames:      []string{"model-1", "model-2"},
		},
		{
			name: "Some models missing metadata",
			models: []ModelInfo{
				{ModelName: "model-1", Provider: "bedrock"},
				{ModelName: "unknown-model", Provider: "vertex"},
			},
			metadata: map[string]ModelMetadata{
				"model-1": {ModelName: "model-1", Lifecycle: "Generally Available"},
			},
			displayPreview:               false,
			expectedValidCount:           1,
			expectedFilteredCount:        1,
			expectedPreviewFilteredCount: 0,
			expectedValidModelNames:      []string{"model-1"},
		},
		{
			name: "Preview models filtered when displayPreview false",
			models: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "preview-model", Provider: "vertex"},
			},
			metadata: map[string]ModelMetadata{
				"ga-model":      {ModelName: "ga-model", Lifecycle: "Generally Available"},
				"preview-model": {ModelName: "preview-model", Lifecycle: "Preview"},
			},
			displayPreview:               false,
			expectedValidCount:           1,
			expectedFilteredCount:        0,
			expectedPreviewFilteredCount: 1,
			expectedValidModelNames:      []string{"ga-model"},
		},
		{
			name: "Preview models included when displayPreview true",
			models: []ModelInfo{
				{ModelName: "ga-model", Provider: "bedrock"},
				{ModelName: "preview-model", Provider: "vertex"},
			},
			metadata: map[string]ModelMetadata{
				"ga-model":      {ModelName: "ga-model", Lifecycle: "Generally Available"},
				"preview-model": {ModelName: "preview-model", Lifecycle: "Preview"},
			},
			displayPreview:               true,
			expectedValidCount:           2,
			expectedFilteredCount:        0,
			expectedPreviewFilteredCount: 0,
			expectedValidModelNames:      []string{"ga-model", "preview-model"},
		},
		{
			name:                         "Empty models list",
			models:                       []ModelInfo{},
			metadata:                     map[string]ModelMetadata{},
			displayPreview:               false,
			expectedValidCount:           0,
			expectedFilteredCount:        0,
			expectedPreviewFilteredCount: 0,
			expectedValidModelNames:      []string{},
		},
		{
			name: "All models filtered - no metadata",
			models: []ModelInfo{
				{ModelName: "unknown-1", Provider: "bedrock"},
				{ModelName: "unknown-2", Provider: "vertex"},
			},
			metadata:                     map[string]ModelMetadata{},
			displayPreview:               false,
			expectedValidCount:           0,
			expectedFilteredCount:        2,
			expectedPreviewFilteredCount: 0,
			expectedValidModelNames:      []string{},
		},
		{
			name: "Mixed filtering - metadata and preview",
			models: []ModelInfo{
				{ModelName: "valid-model", Provider: "bedrock"},
				{ModelName: "preview-model", Provider: "vertex"},
				{ModelName: "unknown-model", Provider: "azure"},
			},
			metadata: map[string]ModelMetadata{
				"valid-model":   {ModelName: "valid-model", Lifecycle: "Generally Available"},
				"preview-model": {ModelName: "preview-model", Lifecycle: "Preview"},
			},
			displayPreview:               false,
			expectedValidCount:           1,
			expectedFilteredCount:        1,
			expectedPreviewFilteredCount: 1,
			expectedValidModelNames:      []string{"valid-model"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock logger
			logger := zap.NewNop()
			oldLogger := loggerFromContext
			loggerFromContext = func(context.Context) *zap.Logger {
				return logger
			}
			defer func() { loggerFromContext = oldLogger }()

			ctx := context.Background()
			validModels, filteredModels, previewFilteredModels := filterAndEnrichModels(ctx, tt.models, tt.metadata, tt.displayPreview)

			assert.Equal(t, tt.expectedValidCount, len(validModels), "Unexpected valid model count")
			assert.Equal(t, tt.expectedFilteredCount, len(filteredModels), "Unexpected filtered model count")
			assert.Equal(t, tt.expectedPreviewFilteredCount, len(previewFilteredModels), "Unexpected preview filtered count")

			// Verify valid model names
			var resultModelNames []string
			for _, m := range validModels {
				resultModelNames = append(resultModelNames, m.ModelName)
			}
			// Handle nil vs empty slice comparison
			if len(resultModelNames) == 0 && len(tt.expectedValidModelNames) == 0 {
				// Both are effectively empty, test passes
				return
			}
			sort.Strings(resultModelNames)
			sort.Strings(tt.expectedValidModelNames)
			assert.Equal(t, tt.expectedValidModelNames, resultModelNames, "Unexpected valid model names")
		})
	}
}

// TestLogFilteringResults tests the logFilteringResults function
func TestLogFilteringResults(t *testing.T) {
	tests := []struct {
		name                  string
		validCount            int
		filteredModels        []string
		previewFilteredModels []string
	}{
		{
			name:                  "No filtered models",
			validCount:            5,
			filteredModels:        []string{},
			previewFilteredModels: []string{},
		},
		{
			name:                  "Some metadata filtered models",
			validCount:            3,
			filteredModels:        []string{"model-1", "model-2"},
			previewFilteredModels: []string{},
		},
		{
			name:                  "Some preview filtered models",
			validCount:            2,
			filteredModels:        []string{},
			previewFilteredModels: []string{"preview-1"},
		},
		{
			name:                  "Both types of filtered models",
			validCount:            1,
			filteredModels:        []string{"unknown-1"},
			previewFilteredModels: []string{"preview-1", "preview-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just verifies the function doesn't panic
			logger := zap.NewNop().Sugar()
			logFilteringResults(logger, tt.validCount, tt.filteredModels, tt.previewFilteredModels)
			// If we reach here without panic, the test passes
		})
	}
}

// TestLogParsedMetadata tests the logParsedMetadata function
func TestLogParsedMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]ModelMetadata
	}{
		{
			name:     "Empty metadata",
			metadata: map[string]ModelMetadata{},
		},
		{
			name: "Single model metadata",
			metadata: map[string]ModelMetadata{
				"test-model": {
					ModelName: "test-model",
					ModelCapabilities: ModelCapabilities{
						Features: []string{"streaming"},
					},
				},
			},
		},
		{
			name: "Multiple model metadata",
			metadata: map[string]ModelMetadata{
				"model-1": {ModelName: "model-1"},
				"model-2": {ModelName: "model-2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test just verifies the function doesn't panic
			logger := zap.NewNop().Sugar()
			logParsedMetadata(logger, tt.metadata)
			// If we reach here without panic, the test passes
		})
	}
}

// TestFilterReasonConstants verifies filter reason constants are distinct
func TestFilterReasonConstants(t *testing.T) {
	reasons := []filterReason{
		filterReasonNone,
		filterReasonNoMetadata,
		filterReasonPreview,
		filterReasonError,
	}

	// Verify all constants are unique
	seen := make(map[filterReason]bool)
	for _, r := range reasons {
		assert.False(t, seen[r], "Filter reason %d is duplicated", r)
		seen[r] = true
	}

	// Verify filterReasonNone is 0 (the zero value)
	assert.Equal(t, filterReason(0), filterReasonNone)
}

// Helper functions for test data
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
