/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package config

import (
	"fmt"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestModelConfigToModel(t *testing.T) {
	tests := []struct {
		name     string
		config   ModelConfig
		provider types.Provider
		expected *types.Model
	}{
		{
			name: "Basic model conversion",
			config: ModelConfig{
				Name:                   "gpt-35-turbo",
				Version:                "1106",
				Label:                  "GPT-3.5 Turbo",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints: []EndpointConfig{
					{Path: "/chat/completions"},
				},
				Capabilities: ModelCapabilitiesConfig{
					Features:         []string{"streaming", "function_calling"},
					InputModalities:  []string{"text"},
					OutputModalities: []string{"text"},
					MimeTypes:        []string{"application/json"},
				},
				Parameters: map[string]ParameterSpec{
					"temperature": {
						Title:       "Temperature",
						Description: "Controls randomness",
						Type:        "float",
						Default:     0.7,
						Maximum:     2.0,
						Minimum:     0.0,
						Required:    false,
					},
				},
			},
			provider: types.ProviderAzure,
			expected: &types.Model{
				Name:                   "gpt-35-turbo",
				Version:                "1106",
				Label:                  "GPT-3.5 Turbo",
				FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
				Provider:               types.ProviderAzure,
				Capabilities: types.ModelCapabilities{
					Features:         []string{"streaming", "function_calling"},
					InputModalities:  []string{"text"},
					OutputModalities: []string{"text"},
					MimeTypes:        []string{"application/json"},
				},
				Parameters: map[string]types.ParameterSpec{
					"temperature": {
						Title:       "Temperature",
						Description: "Controls randomness",
						Type:        "float",
						Default:     0.7,
						Maximum:     2.0,
						Minimum:     0.0,
						Required:    false,
					},
				},
				Endpoints: []types.Endpoint{"chat/completions"},
			},
		},
		{
			name: "Model without explicit ModelKey",
			config: ModelConfig{
				Name:                   "gpt-4",
				Version:                "0125",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints: []EndpointConfig{
					{Path: "/chat/completions"},
				},
				Capabilities: ModelCapabilitiesConfig{
					InputModalities:  []string{"text"},
					OutputModalities: []string{"text"},
				},
			},
			provider: types.ProviderAzure,
			expected: &types.Model{
				Name:                   "gpt-4",
				Version:                "0125",
				FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
				Provider:               types.ProviderAzure,
				Capabilities: types.ModelCapabilities{
					InputModalities:  []string{"text"},
					OutputModalities: []string{"text"},
				},
				Parameters: map[string]types.ParameterSpec{},
				Endpoints:  []types.Endpoint{"chat/completions"},
			},
		},
		{
			name: "Embedding model",
			config: ModelConfig{
				Name:                   "text-embedding-ada-002",
				Version:                "2",
				FunctionalCapabilities: []string{"embedding"},
				Endpoints: []EndpointConfig{
					{Path: "/embeddings"},
				},
				Capabilities: ModelCapabilitiesConfig{
					InputModalities:  []string{"text"},
					OutputModalities: []string{"embedding"},
				},
			},
			provider: types.ProviderAzure,
			expected: &types.Model{
				Name:                   "text-embedding-ada-002",
				Version:                "2",
				FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityEmbedding},
				Provider:               types.ProviderAzure,
				Capabilities: types.ModelCapabilities{
					InputModalities:  []string{"text"},
					OutputModalities: []string{"embedding"},
				},
				Parameters: map[string]types.ParameterSpec{},
				Endpoints:  []types.Endpoint{"embeddings"},
			},
		},
		{
			name: "Image generation model",
			config: ModelConfig{
				Name:                   "dall-e-3",
				Version:                "3.0",
				FunctionalCapabilities: []string{"image"},
				Endpoints: []EndpointConfig{
					{Path: "/images/generations"},
				},
				Capabilities: ModelCapabilitiesConfig{
					InputModalities:  []string{"text"},
					OutputModalities: []string{"image"},
				},
			},
			provider: types.ProviderAzure,
			expected: &types.Model{
				Name:                   "dall-e-3",
				Version:                "3.0",
				FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityImage},
				Provider:               types.ProviderAzure,
				Capabilities: types.ModelCapabilities{
					InputModalities:  []string{"text"},
					OutputModalities: []string{"image"},
				},
				Parameters: map[string]types.ParameterSpec{},
				Endpoints:  []types.Endpoint{"images/generations"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ToModel(tt.provider, types.CreatorOpenAI)

			if result.Name != tt.expected.Name {
				t.Errorf("Expected Name %s, got %s", tt.expected.Name, result.Name)
			}
			if result.Version != tt.expected.Version {
				t.Errorf("Expected Version %s, got %s", tt.expected.Version, result.Version)
			}
			if result.Label != tt.expected.Label {
				t.Errorf("Expected Label %s, got %s", tt.expected.Label, result.Label)
			}
			if result.KEY != tt.expected.KEY {
				t.Errorf("Expected KEY %s, got %s", tt.expected.KEY, result.KEY)
			}
			// Check functional capabilities
			if len(result.FunctionalCapabilities) != len(tt.expected.FunctionalCapabilities) {
				t.Errorf("Expected %d functional capabilities, got %d", len(tt.expected.FunctionalCapabilities), len(result.FunctionalCapabilities))
			}
			for i, expectedCap := range tt.expected.FunctionalCapabilities {
				if i < len(result.FunctionalCapabilities) && result.FunctionalCapabilities[i] != expectedCap {
					t.Errorf("Expected functional capability %s at index %d, got %s", expectedCap, i, result.FunctionalCapabilities[i])
				}
			}
			if result.Provider != tt.expected.Provider {
				t.Errorf("Expected Provider %s, got %s", tt.expected.Provider, result.Provider)
			}

			// Check capabilities
			if len(result.Capabilities.Features) != len(tt.expected.Capabilities.Features) {
				t.Errorf("Expected %d features, got %d", len(tt.expected.Capabilities.Features), len(result.Capabilities.Features))
			}
			if len(result.Capabilities.InputModalities) != len(tt.expected.Capabilities.InputModalities) {
				t.Errorf("Expected %d input modalities, got %d", len(tt.expected.Capabilities.InputModalities), len(result.Capabilities.InputModalities))
			}
			if len(result.Capabilities.OutputModalities) != len(tt.expected.Capabilities.OutputModalities) {
				t.Errorf("Expected %d output modalities, got %d", len(tt.expected.Capabilities.OutputModalities), len(result.Capabilities.OutputModalities))
			}

			// Check parameters
			if len(result.Parameters) != len(tt.expected.Parameters) {
				t.Errorf("Expected %d parameters, got %d", len(tt.expected.Parameters), len(result.Parameters))
			}

			for key, expectedParam := range tt.expected.Parameters {
				if param, exists := result.Parameters[key]; !exists {
					t.Errorf("Expected parameter %s not found", key)
				} else {
					if param.Title != expectedParam.Title {
						t.Errorf("Expected parameter %s title %s, got %s", key, expectedParam.Title, param.Title)
					}
					if param.Type != expectedParam.Type {
						t.Errorf("Expected parameter %s type %s, got %s", key, expectedParam.Type, param.Type)
					}
				}
			}

			// Check endpoints
			if len(result.Endpoints) != len(tt.expected.Endpoints) {
				t.Errorf("Expected %d endpoints, got %d", len(tt.expected.Endpoints), len(result.Endpoints))
			}
			for i, expectedEndpoint := range tt.expected.Endpoints {
				if i < len(result.Endpoints) && result.Endpoints[i] != expectedEndpoint {
					t.Errorf("Expected endpoint %s at index %d, got %s", expectedEndpoint, i, result.Endpoints[i])
				}
			}
		})
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expected    types.Endpoint
		expectError bool
	}{
		{
			name:     "Chat completions",
			path:     "/chat/completions",
			expected: types.EndpointChatCompletions,
		},
		{
			name:     "Embeddings",
			path:     "/embeddings",
			expected: types.EndpointEmbeddings,
		},
		{
			name:     "Images generations",
			path:     "/images/generations",
			expected: types.EndpointImagesGenerations,
		},
		{
			name:     "Converse",
			path:     "/converse",
			expected: types.EndpointConverse,
		},
		{
			name:     "Converse stream",
			path:     "/converse-stream",
			expected: types.EndpointConverseStream,
		},
		{
			name:     "Invoke",
			path:     "/invoke",
			expected: types.EndpointInvoke,
		},
		{
			name:     "Predict",
			path:     "/predict",
			expected: types.EndpointPredict,
		},
		{
			name:     "Invoke stream",
			path:     "/invoke-stream",
			expected: types.EndpointInvokeStream,
		},
		{
			name:        "Unknown endpoint",
			path:        "/unknown/endpoint",
			expectError: true,
		},
		{
			name:        "Empty path",
			path:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := types.NormalizeEndpoint(tt.path)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path %s, but got none", tt.path)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for path %s: %v", tt.path, err)
				}
				if result != tt.expected {
					t.Errorf("Expected endpoint type %s, got %s", tt.expected, result)
				}
			}
		})
	}
}

func TestParameterSpecConversion(t *testing.T) {
	configParam := ParameterSpec{
		Title:       "Max Tokens",
		Description: "Maximum number of tokens to generate",
		Type:        "integer",
		Default:     1000,
		Maximum:     4096,
		Minimum:     1,
		Required:    true,
	}

	config := ModelConfig{
		Name:                   "test-model",
		Version:                "1.0",
		FunctionalCapabilities: []string{"chat_completion"},
		Endpoints: []EndpointConfig{
			{Path: "/chat/completions"},
		},
		Parameters: map[string]ParameterSpec{
			"max_tokens": configParam,
		},
	}

	model := config.ToModel(types.ProviderAzure, types.CreatorOpenAI)

	param, exists := model.Parameters["max_tokens"]
	if !exists {
		t.Fatal("Expected max_tokens parameter to exist")
	}

	if param.Title != configParam.Title {
		t.Errorf("Expected title %s, got %s", configParam.Title, param.Title)
	}
	if param.Description != configParam.Description {
		t.Errorf("Expected description %s, got %s", configParam.Description, param.Description)
	}
	if param.Type != configParam.Type {
		t.Errorf("Expected type %s, got %s", configParam.Type, param.Type)
	}
	if param.Default != configParam.Default {
		t.Errorf("Expected default %v, got %v", configParam.Default, param.Default)
	}
	if param.Maximum != configParam.Maximum {
		t.Errorf("Expected maximum %v, got %v", configParam.Maximum, param.Maximum)
	}
	if param.Minimum != configParam.Minimum {
		t.Errorf("Expected minimum %v, got %v", configParam.Minimum, param.Minimum)
	}
	if param.Required != configParam.Required {
		t.Errorf("Expected required %v, got %v", configParam.Required, param.Required)
	}
}

func TestModelCapabilitiesConversion(t *testing.T) {
	configCapabilities := ModelCapabilitiesConfig{
		Features:         []string{"streaming", "function_calling", "vision"},
		InputModalities:  []string{"text", "image"},
		OutputModalities: []string{"text"},
		MimeTypes:        []string{"application/json", "image/jpeg"},
	}

	config := ModelConfig{
		Name:                   "test-model",
		Version:                "1.0",
		FunctionalCapabilities: []string{"chat_completion"},
		Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
		Capabilities:           configCapabilities,
	}

	model := config.ToModel(types.ProviderAzure, types.CreatorOpenAI)

	if len(model.Capabilities.Features) != len(configCapabilities.Features) {
		t.Errorf("Expected %d features, got %d", len(configCapabilities.Features), len(model.Capabilities.Features))
	}

	for i, feature := range configCapabilities.Features {
		if model.Capabilities.Features[i] != feature {
			t.Errorf("Expected feature %s at index %d, got %s", feature, i, model.Capabilities.Features[i])
		}
	}

	if len(model.Capabilities.InputModalities) != len(configCapabilities.InputModalities) {
		t.Errorf("Expected %d input modalities, got %d", len(configCapabilities.InputModalities), len(model.Capabilities.InputModalities))
	}

	if len(model.Capabilities.OutputModalities) != len(configCapabilities.OutputModalities) {
		t.Errorf("Expected %d output modalities, got %d", len(configCapabilities.OutputModalities), len(model.Capabilities.OutputModalities))
	}

	if len(model.Capabilities.MimeTypes) != len(configCapabilities.MimeTypes) {
		t.Errorf("Expected %d mime types, got %d", len(configCapabilities.MimeTypes), len(model.Capabilities.MimeTypes))
	}
}

func TestEmptyParametersMap(t *testing.T) {
	config := ModelConfig{
		Name:                   "test-model",
		Version:                "1.0",
		FunctionalCapabilities: []string{"chat_completion"},
		Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
		// No parameters specified
	}

	model := config.ToModel(types.ProviderAzure, types.CreatorOpenAI)

	if model.Parameters == nil {
		t.Error("Expected parameters map to be initialized")
	}

	if len(model.Parameters) != 0 {
		t.Errorf("Expected empty parameters map, got %d parameters", len(model.Parameters))
	}
}

func TestDifferentProviders(t *testing.T) {
	config := ModelConfig{
		Name:                   "test-model",
		Version:                "1.0",
		FunctionalCapabilities: []string{"chat_completion"},
		Endpoints: []EndpointConfig{
			{Path: "/chat/completions"},
		},
	}

	providers := []types.Provider{
		types.ProviderAzure,
		types.ProviderGoogle,
		types.ProviderAnthropic,
		types.ProviderAmazon,
		types.ProviderMeta,
	}

	for _, provider := range providers {
		model := config.ToModel(provider, types.CreatorOpenAI)
		if model.Provider != provider {
			t.Errorf("Expected provider %s, got %s", provider, model.Provider)
		}
	}
}

// Comprehensive corner case tests for ModelConfig.ToModel
func TestModelConfig_ToModel_CornerCases(t *testing.T) {
	tests := []struct {
		name     string
		config   *ModelConfig
		provider types.Provider
		creator  types.Creator
		validate func(*testing.T, *types.Model)
	}{
		{
			name:     "nil config",
			config:   nil,
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				// Should handle nil gracefully - this will panic, which is expected behavior
			},
		},
		{
			name:     "empty config",
			config:   &ModelConfig{},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				if model.Name != "" {
					t.Errorf("Expected empty name, got %s", model.Name)
				}
				if model.Version != "" {
					t.Errorf("Expected empty version, got %s", model.Version)
				}
				if len(model.FunctionalCapabilities) != 0 {
					t.Errorf("Expected empty functional capabilities, got %d", len(model.FunctionalCapabilities))
				}
				if len(model.Endpoints) != 0 {
					t.Errorf("Expected empty endpoints, got %d", len(model.Endpoints))
				}
				if model.Parameters == nil {
					t.Errorf("Expected parameters map to be initialized")
				}
				if len(model.Parameters) != 0 {
					t.Errorf("Expected empty parameters, got %d", len(model.Parameters))
				}
			},
		},
		{
			name: "missing required fields",
			config: &ModelConfig{
				// Name missing
				// Version missing
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
			},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				if model.Name != "" {
					t.Errorf("Expected empty name, got %s", model.Name)
				}
				if model.Version != "" {
					t.Errorf("Expected empty version, got %s", model.Version)
				}
			},
		},
		{
			name: "empty arrays",
			config: &ModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{},                 // Empty array
				Endpoints:              []EndpointConfig{},         // Empty array
				Parameters:             map[string]ParameterSpec{}, // Empty map
			},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				if len(model.FunctionalCapabilities) != 0 {
					t.Errorf("Expected empty functional capabilities, got %d", len(model.FunctionalCapabilities))
				}
				if len(model.Endpoints) != 0 {
					t.Errorf("Expected empty endpoints, got %d", len(model.Endpoints))
				}
			},
		},
		{
			name: "nil parameters map",
			config: &ModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
				Parameters:             nil, // Nil map
			},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				if model.Parameters == nil {
					t.Errorf("Expected parameters map to be initialized")
				}
				if len(model.Parameters) != 0 {
					t.Errorf("Expected empty parameters, got %d", len(model.Parameters))
				}
			},
		},
		{
			name: "invalid functional capabilities",
			config: &ModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"invalid", "unknown", "chat_completion", "also_invalid"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
			},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				// Should only include valid capabilities
				if len(model.FunctionalCapabilities) != 1 {
					t.Errorf("Expected 1 valid functional capability, got %d", len(model.FunctionalCapabilities))
				}
				if len(model.FunctionalCapabilities) > 0 && model.FunctionalCapabilities[0] != types.FunctionalCapabilityChatCompletion {
					t.Errorf("Expected chat_completion capability, got %s", model.FunctionalCapabilities[0])
				}
			},
		},
		{
			name: "endpoint normalization failures",
			config: &ModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints: []EndpointConfig{
					{Path: "/invalid/unknown/endpoint"},
					{Path: "/chat/completions"}, // Valid one comes second
				},
			},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				// Should use first endpoint even if invalid for normalization
				if len(model.Endpoints) != 2 {
					t.Errorf("Expected 2 endpoints, got %d", len(model.Endpoints))
				}
			},
		},
		{
			name: "parameter conversion with various data types",
			config: &ModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]ParameterSpec{
					"stringParam": {
						Type:    "string",
						Default: "default_value",
					},
					"intParam": {
						Type:    "integer",
						Default: 42,
						Maximum: 100,
						Minimum: 0,
					},
					"floatParam": {
						Type:    "float",
						Default: 3.14,
						Maximum: 10.0,
						Minimum: 0.0,
					},
					"boolParam": {
						Type:    "boolean",
						Default: true,
					},
					"nilParam": {
						Type:    "string",
						Default: nil,
					},
				},
			},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				if len(model.Parameters) != 5 {
					t.Errorf("Expected 5 parameters, got %d", len(model.Parameters))
				}

				// Check string parameter
				if param, exists := model.Parameters["stringParam"]; !exists {
					t.Errorf("stringParam should exist")
				} else {
					if param.Default != "default_value" {
						t.Errorf("Expected string default 'default_value', got %v", param.Default)
					}
				}

				// Check int parameter
				if param, exists := model.Parameters["intParam"]; !exists {
					t.Errorf("intParam should exist")
				} else {
					if param.Default != 42 {
						t.Errorf("Expected int default 42, got %v", param.Default)
					}
					if param.Maximum != 100 {
						t.Errorf("Expected maximum 100, got %v", param.Maximum)
					}
				}

				// Check nil parameter
				if param, exists := model.Parameters["nilParam"]; !exists {
					t.Errorf("nilParam should exist")
				} else {
					if param.Default != nil {
						t.Errorf("Expected nil default, got %v", param.Default)
					}
				}
			},
		},
		{
			name: "special characters in string fields",
			config: &ModelConfig{
				Name:                   "test-model-with-special-chars!@#$%^&*()",
				Version:                "v1.0.0-beta+build.123",
				Label:                  "Test Model with Unicode: 测试模型 🚀",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
			},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				if model.Name != "test-model-with-special-chars!@#$%^&*()" {
					t.Errorf("Name with special chars not preserved: %s", model.Name)
				}
				if model.Version != "v1.0.0-beta+build.123" {
					t.Errorf("Version with special chars not preserved: %s", model.Version)
				}
				if model.Label != "Test Model with Unicode: 测试模型 🚀" {
					t.Errorf("Label with unicode not preserved: %s", model.Label)
				}
			},
		},
		{
			name: "very large parameter maps",
			config: func() *ModelConfig {
				config := &ModelConfig{
					Name:                   "test-model",
					Version:                "v1",
					FunctionalCapabilities: []string{"chat_completion"},
					Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
					Parameters:             make(map[string]ParameterSpec),
				}
				// Create 1000 parameters
				for i := 0; i < 1000; i++ {
					paramName := fmt.Sprintf("param_%d", i)
					config.Parameters[paramName] = ParameterSpec{
						Type:    "integer",
						Default: i,
						Maximum: i * 10,
						Minimum: 0,
					}
				}
				return config
			}(),
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				if len(model.Parameters) != 1000 {
					t.Errorf("Expected 1000 parameters, got %d", len(model.Parameters))
				}
				// Check a few random parameters
				if param, exists := model.Parameters["param_500"]; !exists {
					t.Errorf("param_500 should exist")
				} else {
					if param.Default != 500 {
						t.Errorf("Expected param_500 default 500, got %v", param.Default)
					}
				}
			},
		},
		{
			name: "maximum and minimum edge cases",
			config: &ModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]ParameterSpec{
					"negativeParam": {
						Type:    "integer",
						Default: -100,
						Maximum: -1,
						Minimum: -1000,
					},
					"zeroParam": {
						Type:    "integer",
						Default: 0,
						Maximum: 0,
						Minimum: 0,
					},
					"veryLargeParam": {
						Type:    "integer",
						Default: 999999999,
						Maximum: 1000000000,
						Minimum: 999999999,
					},
				},
			},
			provider: types.ProviderBedrock,
			creator:  types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				// Check negative parameter
				if param, exists := model.Parameters["negativeParam"]; !exists {
					t.Errorf("negativeParam should exist")
				} else {
					if param.Default != -100 {
						t.Errorf("Expected negative default -100, got %v", param.Default)
					}
					if param.Maximum != -1 {
						t.Errorf("Expected negative maximum -1, got %v", param.Maximum)
					}
				}

				// Check zero parameter
				if param, exists := model.Parameters["zeroParam"]; !exists {
					t.Errorf("zeroParam should exist")
				} else {
					if param.Default != 0 {
						t.Errorf("Expected zero default, got %v", param.Default)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle nil config case specially
			if tt.config == nil {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil config, but got none")
					}
				}()
				tt.config.ToModel(tt.provider, tt.creator)
				return
			}

			result := tt.config.ToModel(tt.provider, tt.creator)
			if result == nil {
				t.Errorf("Expected non-nil result")
				return
			}

			// Verify basic fields are always set correctly
			if result.Provider != tt.provider {
				t.Errorf("Expected provider %s, got %s", tt.provider, result.Provider)
			}
			if result.Creator != tt.creator {
				t.Errorf("Expected creator %s, got %s", tt.creator, result.Creator)
			}

			tt.validate(t, result)
		})
	}
}

// Comprehensive corner case tests for EnhancedModelConfig.ToModel
func TestEnhancedModelConfig_ToModel_CornerCases(t *testing.T) {
	tests := []struct {
		name           string
		config         *EnhancedModelConfig
		infrastructure types.Infrastructure
		provider       types.Provider
		creator        types.Creator
		validate       func(*testing.T, *types.Model)
	}{
		{
			name:           "nil config",
			config:         nil,
			infrastructure: types.InfrastructureAWS,
			provider:       types.ProviderBedrock,
			creator:        types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				// Should handle nil gracefully - this will panic, which is expected behavior
			},
		},
		{
			name:           "empty config",
			config:         &EnhancedModelConfig{},
			infrastructure: types.InfrastructureAWS,
			provider:       types.ProviderBedrock,
			creator:        types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				if model.KEY != "" {
					t.Errorf("Expected empty KEY, got %s", model.KEY)
				}
				if model.Infrastructure != types.InfrastructureAWS {
					t.Errorf("Expected AWS infrastructure, got %s", model.Infrastructure)
				}
				if len(model.FunctionalCapabilities) != 0 {
					t.Errorf("Expected empty functional capabilities, got %d", len(model.FunctionalCapabilities))
				}
			},
		},
		{
			name: "ID field handling",
			config: &EnhancedModelConfig{
				KEY:                    "custom-id-123",
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
				Parameters:             map[string]ParameterSpec{},
			},
			infrastructure: types.InfrastructureGCP,
			provider:       types.ProviderVertex,
			creator:        types.CreatorGoogle,
			validate: func(t *testing.T, model *types.Model) {
				if model.KEY != "custom-id-123" {
					t.Errorf("Expected KEY 'custom-id-123', got %s", model.KEY)
				}
				if model.Infrastructure != types.InfrastructureGCP {
					t.Errorf("Expected GCP infrastructure, got %s", model.Infrastructure)
				}
			},
		},
		{
			name: "deployment info and lifecycle fields",
			config: &EnhancedModelConfig{
				KEY:                    "test-id",
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
				Parameters:             map[string]ParameterSpec{},
				DeploymentInfo: DeploymentConfig{
					Region:       "us-east-1",
					InstanceType: "ml.m5.large",
					Scaling: ScalingConfig{
						MinInstances: 1,
						MaxInstances: 10,
					},
					CustomConfig: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
				Lifecycle: LifecycleConfig{
					Status:          "active",
					DeprecationDate: "2025-12-31",
					EndOfLifeDate:   "2026-12-31",
				},
			},
			infrastructure: types.InfrastructureAWS,
			provider:       types.ProviderBedrock,
			creator:        types.CreatorAmazon,
			validate: func(t *testing.T, model *types.Model) {
				// Note: DeploymentInfo and Lifecycle are not copied to types.Model
				// This is expected behavior as types.Model doesn't have these fields
				if model.Name != "test-model" {
					t.Errorf("Expected name 'test-model', got %s", model.Name)
				}
			},
		},
		{
			name: "enhanced capabilities conversion",
			config: &EnhancedModelConfig{
				KEY:                    "test-id",
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion", "image"},
				Endpoints:              []EndpointConfig{{Path: "/chat/completions"}},
				Parameters:             map[string]ParameterSpec{},
				Capabilities: ModelCapabilitiesConfig{
					Features:         []string{"streaming", "function_calling", "vision"},
					InputModalities:  []string{"text", "image", "audio"},
					OutputModalities: []string{"text", "image"},
					MimeTypes:        []string{"application/json", "image/png", "image/jpeg", "audio/wav"},
				},
			},
			infrastructure: types.InfrastructureAzure,
			provider:       types.ProviderAzure,
			creator:        types.CreatorOpenAI,
			validate: func(t *testing.T, model *types.Model) {
				if len(model.Capabilities.Features) != 3 {
					t.Errorf("Expected 3 features, got %d", len(model.Capabilities.Features))
				}
				if len(model.Capabilities.InputModalities) != 3 {
					t.Errorf("Expected 3 input modalities, got %d", len(model.Capabilities.InputModalities))
				}
				if len(model.Capabilities.OutputModalities) != 2 {
					t.Errorf("Expected 2 output modalities, got %d", len(model.Capabilities.OutputModalities))
				}
				if len(model.Capabilities.MimeTypes) != 4 {
					t.Errorf("Expected 4 mime types, got %d", len(model.Capabilities.MimeTypes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle nil config case specially
			if tt.config == nil {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil config, but got none")
					}
				}()
				tt.config.ToModel(tt.infrastructure, tt.provider, tt.creator)
				return
			}

			result := tt.config.ToModel(tt.infrastructure, tt.provider, tt.creator)
			if result == nil {
				t.Errorf("Expected non-nil result")
				return
			}

			// Verify basic fields are always set correctly
			if result.Infrastructure != tt.infrastructure {
				t.Errorf("Expected infrastructure %s, got %s", tt.infrastructure, result.Infrastructure)
			}
			if result.Provider != tt.provider {
				t.Errorf("Expected provider %s, got %s", tt.provider, result.Provider)
			}
			if result.Creator != tt.creator {
				t.Errorf("Expected creator %s, got %s", tt.creator, result.Creator)
			}

			tt.validate(t, result)
		})
	}
}
