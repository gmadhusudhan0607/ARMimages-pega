/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestModelValidator_ValidateModelGroup(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name        string
		group       *config.ModelGroup
		filePath    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid model group",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
		{
			name: "infrastructure mismatch",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureGCP, // Wrong infrastructure
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "infrastructure mismatch",
		},
		{
			name: "provider mismatch",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderVertex, // Wrong provider
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "provider mismatch",
		},
		{
			name: "creator mismatch",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorGoogle, // Wrong creator
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "creator mismatch",
		},
		{
			name: "missing required model name",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						// Name missing
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.name is required",
		},
		{
			name: "missing required model version",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name: "test-model",
						// Version missing
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.version is required",
		},
		{
			name: "missing required endpoints",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						// Endpoints missing
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.endpoints is required",
		},
		{
			name: "missing required parameters",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						// Parameters missing
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.parameters is required",
		},
		{
			name: "missing maxInputTokens parameter",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							// maxInputTokens missing
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.parameters.maxInputTokens is required",
		},
		{
			name: "missing maxOutputTokens parameter",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							// max_tokens missing
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.parameters.maxOutputTokens is required",
		},
		{
			name: "missing max_tokens.maximum",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type: "integer",
								// Maximum missing - should be required for text models
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.parameters.maxOutputTokens.maximum is required",
		},
		{
			name: "embedding model without max_tokens (valid)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-embedding",
						Version:                "v1",
						FunctionalCapabilities: []string{"embedding"},
						Endpoints: []config.EndpointConfig{
							{Path: "/embedding"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							// max_tokens not required for embedding models
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
		{
			name: "image model without max_tokens (valid)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-image",
						Version:                "v1",
						FunctionalCapabilities: []string{"image"},
						Endpoints: []config.EndpointConfig{
							{Path: "/images/generations"},
						},
						Capabilities: config.ModelCapabilitiesConfig{
							OutputModalities: []string{"image"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							// max_tokens not required for image models
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
		{
			name: "audio model without max_tokens (valid)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-audio",
						Version:                "v1",
						FunctionalCapabilities: []string{"audio"}, // Use valid functional capability
						Endpoints: []config.EndpointConfig{
							{Path: "/audio/generations"},
						},
						Capabilities: config.ModelCapabilitiesConfig{
							OutputModalities: []string{"audio"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							// max_tokens not required for audio models
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
		{
			name: "embedding model with outputModalities without max_tokens (valid)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-embedding",
						Version:                "v1",
						FunctionalCapabilities: []string{"embedding"},
						Endpoints: []config.EndpointConfig{
							{Path: "/embeddings"},
						},
						Capabilities: config.ModelCapabilitiesConfig{
							OutputModalities: []string{"embedding"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							// max_tokens not required for embedding models
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
		{
			name: "text model without max_tokens (invalid)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-text",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Capabilities: config.ModelCapabilitiesConfig{
							OutputModalities: []string{"text"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							// max_tokens missing - should be required for text models
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.parameters.maxOutputTokens is required",
		},
		{
			name: "mixed modalities with text without max_tokens (invalid)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-mixed",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"}, // Use valid functional capability
						Endpoints: []config.EndpointConfig{
							{Path: "/generate"},
						},
						Capabilities: config.ModelCapabilitiesConfig{
							OutputModalities: []string{"text", "audio"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							// max_tokens missing - should be required because of text modality
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.parameters.maxOutputTokens is required",
		},
		{
			name: "mixed modalities without text, without max_tokens (valid)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-mixed-no-text",
						Version:                "v1",
						FunctionalCapabilities: []string{"image"}, // Use valid functional capability
						Endpoints: []config.EndpointConfig{
							{Path: "/generate"},
						},
						Capabilities: config.ModelCapabilitiesConfig{
							OutputModalities: []string{"image", "audio"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							// max_tokens not required - no text modality
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
		{
			name: "text model with max_tokens but missing maximum (invalid)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-text-no-max",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Capabilities: config.ModelCapabilitiesConfig{
							OutputModalities: []string{"text"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type: "integer",
								// Maximum missing - should be required for text models
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "model.parameters.maxOutputTokens.maximum is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateModelGroup(tt.group, tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				// Verify that ID was automatically calculated
				if len(tt.group.Models) > 0 {
					expectedID := fmt.Sprintf("aws/bedrock/amazon/%s/v1", tt.group.Models[0].Name)
					if tt.group.Models[0].KEY != expectedID {
						t.Errorf("expected ID to be '%s', got '%s'", expectedID, tt.group.Models[0].KEY)
					}
				}
			}
		})
	}
}

func TestModelValidator_ExtractPathInfo(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		expected    *PathInfo
	}{
		{
			name:     "valid AWS path",
			filePath: "aws/bedrock/amazon/nova.yaml",
			expected: &PathInfo{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				SpecFile:       "nova",
			},
		},
		{
			name:     "valid GCP path",
			filePath: "gcp/vertex/google/gemini-1.0.yaml",
			expected: &PathInfo{
				Infrastructure: types.InfrastructureGCP,
				Provider:       types.ProviderVertex,
				Creator:        types.CreatorGoogle,
				SpecFile:       "gemini-1.0",
			},
		},
		{
			name:        "invalid path - too few components",
			filePath:    "aws/bedrock.yaml",
			expectError: true,
		},
		{
			name:        "invalid path - no extension",
			filePath:    "aws/bedrock/amazon/nova",
			expectError: false, // Should still work, just no .yaml extension
			expected: &PathInfo{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				SpecFile:       "nova",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathInfo, err := validator.extractPathInfo(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if pathInfo.Infrastructure != tt.expected.Infrastructure {
				t.Errorf("expected infrastructure '%s', got '%s'", tt.expected.Infrastructure, pathInfo.Infrastructure)
			}
			if pathInfo.Provider != tt.expected.Provider {
				t.Errorf("expected provider '%s', got '%s'", tt.expected.Provider, pathInfo.Provider)
			}
			if pathInfo.Creator != tt.expected.Creator {
				t.Errorf("expected creator '%s', got '%s'", tt.expected.Creator, pathInfo.Creator)
			}
			if pathInfo.SpecFile != tt.expected.SpecFile {
				t.Errorf("expected model name '%s', got '%s'", tt.expected.SpecFile, pathInfo.SpecFile)
			}
		})
	}
}

func TestModelValidator_CalculateKey(t *testing.T) {
	validator := NewModelValidator()

	pathInfo := &PathInfo{
		Infrastructure: types.InfrastructureAWS,
		Provider:       types.ProviderBedrock,
		Creator:        types.CreatorAmazon,
		SpecFile:       "nova",
	}

	model := &config.EnhancedModelConfig{
		Name:    "nova-lite",
		Version: "v1",
	}

	expectedKey := "aws/bedrock/amazon/nova-lite/v1"
	actualKey := validator.calculateModelKEY(pathInfo, model)

	if actualKey != expectedKey {
		t.Errorf("expected ModelName '%s', got '%s'", expectedKey, actualKey)
	}
}

func TestModelValidator_ForbiddenProperties(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name        string
		group       *config.ModelGroup
		filePath    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid model without forbidden properties",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
		{
			name: "valid model with ID field (auto-calculated)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						KEY:                    "", // This will be auto-calculated
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateModelGroup(tt.group, tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestModelValidator_CheckForbiddenField(t *testing.T) {
	validator := NewModelValidator()

	model := &config.EnhancedModelConfig{
		Name:                   "test-model",
		Version:                "v1",
		FunctionalCapabilities: []string{"chat_completion"},
		KEY:                    "test-id", // This is the correct field name
		Parameters: map[string]config.ParameterSpec{
			"maxInputTokens": {
				Type:    "integer",
				Maximum: 100000,
			},
			"key": { // This should be detected as forbidden in parameters
				Type: "string",
			},
		},
	}

	// Test checking for 'key' field at top level (should be detected because KEY has a non-zero value)
	err := validator.checkForbiddenField(model, []string{"key"})
	if err == nil {
		t.Errorf("expected error for forbidden 'key' field at top level, but got none")
	} else if !strings.Contains(err.Error(), "forbidden property 'key' found in YAML at path 'key'") {
		t.Errorf("expected specific error message, got: %v", err)
	}

	// Test checking for 'ModelName' exact match (should not exist as exact match)
	err = validator.checkForbiddenFieldExact(model, []string{"ModelName"})
	if err != nil {
		t.Errorf("unexpected error for non-existent 'ModelName' field: %v", err)
	}

	// Test checking for 'key' field in parameters (should be detected)
	err = validator.checkForbiddenField(model, []string{"parameters", "key"})
	if err == nil {
		t.Errorf("expected error for forbidden 'key' field in parameters, but got none")
	} else if !strings.Contains(err.Error(), "forbidden property 'key' found in YAML at path 'parameters.key'") {
		t.Errorf("expected specific error message, got: %v", err)
	}

	// Test checking for non-existent nested path
	err = validator.checkForbiddenField(model, []string{"parameters", "nonexistent", "field"})
	if err != nil {
		t.Errorf("unexpected error for non-existent nested path: %v", err)
	}

	// Test checking existing field 'key' at top level (exact match)
	err = validator.checkForbiddenFieldExact(model, []string{"key"})
	if err == nil {
		t.Errorf("expected error for forbidden 'key' field at top level (exact match), but got none")
	} else if !strings.Contains(err.Error(), "forbidden property 'key' found in YAML at path 'key'") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestModelValidator_PathBasedValidation(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name          string
		model         *config.EnhancedModelConfig
		forbiddenPath []string
		exactMatch    bool
		expectError   bool
		expectedMsg   string
	}{
		{
			name: "top level key field - exact match",
			model: &config.EnhancedModelConfig{
				KEY:  "test-id",
				Name: "test-model",
			},
			forbiddenPath: []string{"key"},
			exactMatch:    true,
			expectError:   true,
			expectedMsg:   "forbidden property 'key' found in YAML at path 'key' (exact match)",
		},
		{
			name: "top level key field - case insensitive",
			model: &config.EnhancedModelConfig{
				KEY:  "test-id",
				Name: "test-model",
			},
			forbiddenPath: []string{"KEY"},
			exactMatch:    false,
			expectError:   true,
			expectedMsg:   "forbidden property 'KEY' found in YAML at path 'key' (case insensitive check)",
		},
		{
			name: "nested field in parameters",
			model: &config.EnhancedModelConfig{
				Name: "test-model",
				Parameters: map[string]config.ParameterSpec{
					"forbiddenParam": {
						Type: "string",
					},
				},
			},
			forbiddenPath: []string{"parameters", "forbiddenParam"},
			exactMatch:    true,
			expectError:   true,
			expectedMsg:   "forbidden property 'forbiddenParam' found in YAML at path 'parameters.forbiddenParam' (exact match)",
		},
		{
			name: "non-existent top level field",
			model: &config.EnhancedModelConfig{
				Name: "test-model",
			},
			forbiddenPath: []string{"nonexistent"},
			exactMatch:    true,
			expectError:   false,
		},
		{
			name: "non-existent nested field",
			model: &config.EnhancedModelConfig{
				Name: "test-model",
				Parameters: map[string]config.ParameterSpec{
					"validParam": {
						Type: "string",
					},
				},
			},
			forbiddenPath: []string{"parameters", "nonexistent"},
			exactMatch:    true,
			expectError:   false,
		},
		{
			name: "deep nested path - non-existent",
			model: &config.EnhancedModelConfig{
				Name: "test-model",
			},
			forbiddenPath: []string{"some-path", "dir", "id"},
			exactMatch:    true,
			expectError:   false,
		},
		{
			name: "empty path",
			model: &config.EnhancedModelConfig{
				Name: "test-model",
			},
			forbiddenPath: []string{},
			exactMatch:    true,
			expectError:   true,
			expectedMsg:   "forbidden path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.exactMatch {
				err = validator.checkForbiddenFieldExact(tt.model, tt.forbiddenPath)
			} else {
				err = validator.checkForbiddenField(tt.model, tt.forbiddenPath)
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.expectedMsg != "" && !strings.Contains(err.Error(), tt.expectedMsg) {
					t.Errorf("expected error message to contain '%s', got: %s", tt.expectedMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestModelValidator_ForbiddenTopLevelProperties tests that 'id' is forbidden at top level
func TestModelValidator_ForbiddenTopLevelProperties(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name        string
		group       *config.ModelGroup
		filePath    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "model with 'key' at top level should be forbidden (case insensitive)",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						KEY:                    "should-be-auto-calculated", // This should trigger validation error
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: true,
			errorMsg:    "forbidden property 'key' found in YAML",
		},
		{
			name: "model without forbidden properties should pass",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models: []config.EnhancedModelConfig{
					{
						Name:                   "test-model",
						Version:                "v1",
						FunctionalCapabilities: []string{"chat_completion"},
						// ID field is not set, will be auto-calculated
						Endpoints: []config.EndpointConfig{
							{Path: "/converse"},
						},
						Parameters: map[string]config.ParameterSpec{
							"maxInputTokens": {
								Type:    "integer",
								Maximum: 100000,
							},
							"maxOutputTokens": {
								Type:    "integer",
								Maximum: 10000,
							},
						},
					},
				},
			},
			filePath:    "aws/bedrock/amazon/test.yaml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateModelGroup(tt.group, tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				// Verify that ID was automatically calculated
				if len(tt.group.Models) > 0 {
					expectedID := fmt.Sprintf("aws/bedrock/amazon/%s/v1", tt.group.Models[0].Name)
					if tt.group.Models[0].KEY != expectedID {
						t.Errorf("expected ID to be '%s', got '%s'", expectedID, tt.group.Models[0].KEY)
					}
				}
			}
		})
	}
}

func TestModelValidator_IsEmbeddingModel(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name     string
		model    *config.EnhancedModelConfig
		expected bool
	}{
		{
			name: "embedding model by type",
			model: &config.EnhancedModelConfig{
				FunctionalCapabilities: []string{"embedding"},
			},
			expected: true,
		},
		{
			name: "embedding model by endpoint path",
			model: &config.EnhancedModelConfig{
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints: []config.EndpointConfig{
					{Path: "/embedding"},
				},
			},
			expected: true,
		},
		{
			name: "non-embedding model",
			model: &config.EnhancedModelConfig{
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints: []config.EndpointConfig{
					{Path: "/converse"},
				},
			},
			expected: false,
		},
		{
			name: "embedding model with mixed case type",
			model: &config.EnhancedModelConfig{
				FunctionalCapabilities: []string{"embedding"},
			},
			expected: true,
		},
		{
			name: "embedding model with mixed case endpoint",
			model: &config.EnhancedModelConfig{
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints: []config.EndpointConfig{
					{Path: "/Embedding"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.isEmbeddingModel(tt.model)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestModelValidator_RequiresMaxOutputTokens(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name     string
		model    *config.EnhancedModelConfig
		expected bool
	}{
		{
			name: "text output modality requires max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"text"},
				},
			},
			expected: true,
		},
		{
			name: "image output modality does not require max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"image"},
				},
			},
			expected: false,
		},
		{
			name: "audio output modality does not require max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"audio"},
				},
			},
			expected: false,
		},
		{
			name: "embedding output modality does not require max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"embedding"},
				},
			},
			expected: false,
		},
		{
			name: "mixed modalities with text requires max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"text", "audio"},
				},
			},
			expected: true,
		},
		{
			name: "mixed modalities without text does not require max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"image", "audio"},
				},
			},
			expected: false,
		},
		{
			name: "case insensitive - TEXT requires max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"TEXT"},
				},
			},
			expected: true,
		},
		{
			name: "case insensitive - IMAGE does not require max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"IMAGE"},
				},
			},
			expected: false,
		},
		{
			name: "case insensitive - AUDIO does not require max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"AUDIO"},
				},
			},
			expected: false,
		},
		{
			name: "case insensitive - EMBEDDING does not require max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"EMBEDDING"},
				},
			},
			expected: false,
		},
		{
			name: "no output modalities requires max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{},
				},
			},
			expected: true,
		},
		{
			name: "custom modality requires max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"custom"},
				},
			},
			expected: true,
		},
		{
			name: "chat modality requires max_tokens",
			model: &config.EnhancedModelConfig{
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"chat"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.requiresMaxOutputTokens(tt.model)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestModelValidator_ValidateModelKeyFormat(t *testing.T) {
	validator := NewModelValidator()

	pathInfo := &PathInfo{
		Infrastructure: types.InfrastructureAWS,
		Provider:       types.ProviderBedrock,
		Creator:        types.CreatorAmazon,
		SpecFile:       "nova",
	}

	model := &config.EnhancedModelConfig{
		Name:    "nova-lite",
		Version: "v1",
	}

	tests := []struct {
		name        string
		id          string
		expectError bool
	}{
		{
			name:        "valid model ID format",
			id:          "aws/bedrock/amazon/nova-lite/v1",
			expectError: false,
		},
		{
			name:        "invalid model ID format - wrong infrastructure",
			id:          "gcp/bedrock/amazon/nova-lite/v1",
			expectError: true,
		},
		{
			name:        "invalid model ID format - wrong provider",
			id:          "aws/vertex/amazon/nova-lite/v1",
			expectError: true,
		},
		{
			name:        "invalid model ID format - wrong creator",
			id:          "aws/bedrock/google/nova-lite/v1",
			expectError: true,
		},
		{
			name:        "invalid model ID format - wrong name",
			id:          "aws/bedrock/amazon/wrong-name/v1",
			expectError: true,
		},
		{
			name:        "invalid model ID format - wrong version",
			id:          "aws/bedrock/amazon/nova-lite/v2",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateModelKEYFormat(tt.id, pathInfo, model)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
