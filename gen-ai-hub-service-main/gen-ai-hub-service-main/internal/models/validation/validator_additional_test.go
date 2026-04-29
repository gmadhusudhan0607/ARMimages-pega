/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package validation

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

// Comprehensive corner case tests for validateGroupRequirements
func TestModelValidator_validateGroupRequirements_CornerCases(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name        string
		group       *config.ModelGroup
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil group",
			group:       nil,
			expectError: true,
			errorMsg:    "runtime error", // Will panic on nil pointer dereference
		},
		{
			name: "empty infrastructure string",
			group: &config.ModelGroup{
				Infrastructure: "",
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: true,
			errorMsg:    "infrastructure is required",
		},
		{
			name: "whitespace-only infrastructure",
			group: &config.ModelGroup{
				Infrastructure: "   ",
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: false, // validateGroupRequirements only checks for empty strings, not whitespace
		},
		{
			name: "tab and newline infrastructure",
			group: &config.ModelGroup{
				Infrastructure: "\t\n",
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: false, // validateGroupRequirements only checks for empty strings, not whitespace
		},
		{
			name: "empty provider string",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       "",
				Creator:        types.CreatorAmazon,
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: true,
			errorMsg:    "provider is required",
		},
		{
			name: "whitespace-only provider",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       "   ",
				Creator:        types.CreatorAmazon,
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: false, // validateGroupRequirements only checks for empty strings, not whitespace
		},
		{
			name: "empty creator string",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        "",
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: true,
			errorMsg:    "creator is required",
		},
		{
			name: "whitespace-only creator",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        "   ",
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: false, // validateGroupRequirements only checks for empty strings, not whitespace
		},
		{
			name: "empty models array",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models:         []config.EnhancedModelConfig{},
			},
			expectError: true,
			errorMsg:    "at least one model is required",
		},
		{
			name: "nil models array",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models:         nil,
			},
			expectError: true,
			errorMsg:    "at least one model is required",
		},
		{
			name: "very large models array",
			group: &config.ModelGroup{
				Infrastructure: types.InfrastructureAWS,
				Provider:       types.ProviderBedrock,
				Creator:        types.CreatorAmazon,
				Models:         make([]config.EnhancedModelConfig, 1000),
			},
			expectError: false,
		},
		{
			name: "special characters in enum fields",
			group: &config.ModelGroup{
				Infrastructure: types.Infrastructure("aws@special"),
				Provider:       types.Provider("bedrock#special"),
				Creator:        types.Creator("amazon$special"),
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: false, // validateGroupRequirements doesn't validate enum values
		},
		{
			name: "unicode characters in fields",
			group: &config.ModelGroup{
				Infrastructure: types.Infrastructure("aws测试"),
				Provider:       types.Provider("bedrock🚀"),
				Creator:        types.Creator("amazon中文"),
				Models:         []config.EnhancedModelConfig{{}},
			},
			expectError: false, // validateGroupRequirements doesn't validate enum values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle nil group case specially
			if tt.group == nil {
				defer func() {
					if r := recover(); r != nil {
						// Expected panic for nil group
						return
					}
					t.Errorf("Expected panic for nil group, but got none")
				}()
				_ = validator.validateGroupRequirements(tt.group)
				return
			}

			err := validator.validateGroupRequirements(tt.group)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Comprehensive corner case tests for validateAndEnrichModel
func TestModelValidator_validateAndEnrichModel_CornerCases(t *testing.T) {
	validator := NewModelValidator()
	pathInfo := &PathInfo{
		Infrastructure: types.InfrastructureAWS,
		Provider:       types.ProviderBedrock,
		Creator:        types.CreatorAmazon,
		SpecFile:       "test",
	}

	tests := []struct {
		name        string
		model       *config.EnhancedModelConfig
		pathInfo    *PathInfo
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *config.EnhancedModelConfig)
	}{
		{
			name:        "nil model",
			model:       nil,
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "runtime error", // Will panic on nil pointer dereference
		},
		{
			name: "model with forbidden KEY property",
			model: &config.EnhancedModelConfig{
				KEY:                    "should-not-be-set",
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "forbidden property 'key' found in YAML",
		},
		{
			name: "missing required name",
			model: &config.EnhancedModelConfig{
				// Name missing
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.name is required",
		},
		{
			name: "missing required version",
			model: &config.EnhancedModelConfig{
				Name: "test-model",
				// Version missing
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.version is required",
		},
		{
			name: "empty functional capabilities",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.functionalCapabilities is required",
		},
		{
			name: "nil functional capabilities",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: nil,
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.functionalCapabilities is required",
		},
		{
			name: "invalid functional capability",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"invalid_capability"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "invalid functional capability 'invalid_capability'",
		},
		{
			name: "mixed valid and invalid functional capabilities",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion", "invalid", "image"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "invalid functional capability 'invalid'",
		},
		{
			name: "empty endpoints",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.endpoints is required and must have at least one endpoint",
		},
		{
			name: "nil endpoints",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              nil,
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.endpoints is required and must have at least one endpoint",
		},
		{
			name: "nil parameters",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters:             nil,
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.parameters is required",
		},
		{
			name: "missing maxInputTokens parameter",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters:             map[string]config.ParameterSpec{},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.parameters.maxInputTokens is required",
		},
		{
			name: "maxInputTokens without type",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {
						// Type missing
					},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.parameters.maxInputTokens.type is required",
		},
		{
			name: "text model missing max_tokens",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"text"},
				},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.parameters.maxOutputTokens is required",
		},
		{
			name: "text model max_tokens without maximum",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Capabilities: config.ModelCapabilitiesConfig{
					OutputModalities: []string{"text"},
				},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
					"maxOutputTokens": {
						Type: "integer",
						// Maximum missing
					},
				},
			},
			pathInfo:    pathInfo,
			expectError: true,
			errorMsg:    "model.parameters.maxOutputTokens.maximum is required",
		},
		{
			name: "embedding model without max_tokens (valid)",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "v1",
				FunctionalCapabilities: []string{"embedding"},
				Endpoints:              []config.EndpointConfig{{Path: "/embeddings"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens": {Type: "integer"},
				},
			},
			pathInfo:    pathInfo,
			expectError: false,
			validate: func(t *testing.T, model *config.EnhancedModelConfig) {
				expectedID := "aws/bedrock/amazon/test-model/v1"
				if model.KEY != expectedID {
					t.Errorf("Expected ID '%s', got '%s'", expectedID, model.KEY)
				}
			},
		},
		{
			name: "ID calculation with special characters",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model-with-special!@#",
				Version:                "v1.0.0-beta+build",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens":  {Type: "integer"},
					"maxOutputTokens": {Type: "integer", Maximum: 1000},
				},
			},
			pathInfo: &PathInfo{
				Infrastructure: types.InfrastructureGCP,
				Provider:       types.ProviderVertex,
				Creator:        types.CreatorGoogle,
				SpecFile:       "special-file",
			},
			expectError: false,
			validate: func(t *testing.T, model *config.EnhancedModelConfig) {
				expectedID := "gcp/vertex/google/test-model-with-special!@#/v1.0.0-beta+build"
				if model.KEY != expectedID {
					t.Errorf("Expected ID '%s', got '%s'", expectedID, model.KEY)
				}
			},
		},
		{
			name: "PathInfo with unicode characters",
			model: &config.EnhancedModelConfig{
				Name:                   "测试模型",
				Version:                "版本1",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens":  {Type: "integer"},
					"maxOutputTokens": {Type: "integer", Maximum: 1000},
				},
			},
			pathInfo: &PathInfo{
				Infrastructure: types.Infrastructure("基础设施"),
				Provider:       types.Provider("提供商"),
				Creator:        types.Creator("创建者"),
				SpecFile:       "规格文件",
			},
			expectError: false,
			validate: func(t *testing.T, model *config.EnhancedModelConfig) {
				expectedID := "基础设施/提供商/创建者/测试模型/版本1"
				if model.KEY != expectedID {
					t.Errorf("Expected ID '%s', got '%s'", expectedID, model.KEY)
				}
			},
		},
		{
			name: "version format variations",
			model: &config.EnhancedModelConfig{
				Name:                   "test-model",
				Version:                "latest",
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens":  {Type: "integer"},
					"maxOutputTokens": {Type: "integer", Maximum: 1000},
				},
			},
			pathInfo:    pathInfo,
			expectError: false,
			validate: func(t *testing.T, model *config.EnhancedModelConfig) {
				expectedID := "aws/bedrock/amazon/test-model/latest"
				if model.KEY != expectedID {
					t.Errorf("Expected ID '%s', got '%s'", expectedID, model.KEY)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle nil model case specially
			if tt.model == nil {
				defer func() {
					if r := recover(); r != nil {
						// Expected panic for nil model
						return
					}
					t.Errorf("Expected panic for nil model, but got none")
				}()
				_ = validator.validateAndEnrichModel(tt.model, tt.pathInfo)
				return
			}

			err := validator.validateAndEnrichModel(tt.model, tt.pathInfo)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if tt.validate != nil {
					tt.validate(t, tt.model)
				}
			}
		})
	}
}

// Comprehensive corner case tests for checkForbiddenFieldAtPath
func TestModelValidator_checkForbiddenFieldAtPath_CornerCases(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name        string
		value       reflect.Value
		path        []string
		exactMatch  bool
		currentPath string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty path",
			value:       reflect.ValueOf(struct{}{}),
			path:        []string{},
			exactMatch:  true,
			currentPath: "",
			expectError: false,
		},
		{
			name:        "invalid reflect value",
			value:       reflect.Value{},
			path:        []string{"field"},
			exactMatch:  true,
			currentPath: "",
			expectError: false, // Should handle gracefully
		},
		{
			name: "very deep nesting",
			value: reflect.ValueOf(map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"level4": map[string]interface{}{
								"level5": map[string]interface{}{
									"level6": map[string]interface{}{
										"level7": map[string]interface{}{
											"level8": map[string]interface{}{
												"level9": map[string]interface{}{
													"level10": "found",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}),
			path:        []string{"level1", "level2", "level3", "level4", "level5", "level6", "level7", "level8", "level9", "level10"},
			exactMatch:  true,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property 'level10' found in YAML",
		},
		{
			name: "path with empty strings",
			value: reflect.ValueOf(map[string]interface{}{
				"": "empty_key",
			}),
			path:        []string{""},
			exactMatch:  true,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property '' found in YAML",
		},
		{
			name: "unicode characters in paths",
			value: reflect.ValueOf(map[string]interface{}{
				"测试": "unicode_value",
			}),
			path:        []string{"测试"},
			exactMatch:  true,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property '测试' found in YAML",
		},
		{
			name: "struct with unexported fields",
			value: reflect.ValueOf(struct {
				ExportedField   string `yaml:"exported"`
				unexportedField string `yaml:"unexported"`
			}{
				ExportedField:   "exported_value",
				unexportedField: "unexported_value",
			}),
			path:        []string{"exported"},
			exactMatch:  true,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property 'exported' found in YAML",
		},
		{
			name: "struct with embedded fields",
			value: reflect.ValueOf(struct {
				Embedded struct {
					Field string `yaml:"field"`
				} `yaml:"embedded"`
			}{
				Embedded: struct {
					Field string `yaml:"field"`
				}{
					Field: "embedded_value",
				},
			}),
			path:        []string{"embedded", "field"},
			exactMatch:  true,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property 'field' found in YAML",
		},
		{
			name: "map with non-string keys",
			value: reflect.ValueOf(map[int]string{
				123: "numeric_key",
			}),
			path:        []string{"123"},
			exactMatch:  true,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property '123' found in YAML",
		},
		{
			name: "slice type",
			value: reflect.ValueOf([]string{
				"item1", "item2", "item3",
			}),
			path:        []string{"0"},
			exactMatch:  true,
			currentPath: "",
			expectError: false, // Slices don't have named fields
		},
		{
			name: "array type",
			value: reflect.ValueOf([3]string{
				"item1", "item2", "item3",
			}),
			path:        []string{"0"},
			exactMatch:  true,
			currentPath: "",
			expectError: false, // Arrays don't have named fields
		},
		{
			name: "pointer to struct with non-zero field",
			value: reflect.ValueOf(&struct {
				Field string `yaml:"field"`
			}{
				Field: "pointer_value",
			}),
			path:        []string{"field"},
			exactMatch:  true,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property 'field' found in YAML",
		},
		{
			name:        "nil pointer",
			value:       reflect.ValueOf((*struct{})(nil)),
			path:        []string{"field"},
			exactMatch:  true,
			currentPath: "",
			expectError: false, // Nil pointer should be handled gracefully
		},
		{
			name: "interface with concrete value",
			value: reflect.ValueOf(interface{}(map[string]string{
				"key": "interface_value",
			})),
			path:        []string{"key"},
			exactMatch:  true,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property 'key' found in YAML",
		},
		{
			name:        "nil interface",
			value:       reflect.ValueOf((*interface{})(nil)).Elem(),
			path:        []string{"field"},
			exactMatch:  true,
			currentPath: "",
			expectError: false, // Nil interface should be handled gracefully
		},
		{
			name: "case insensitive matching",
			value: reflect.ValueOf(struct {
				Field string `yaml:"field"`
			}{
				Field: "case_test",
			}),
			path:        []string{"FIELD"},
			exactMatch:  false,
			currentPath: "",
			expectError: true,
			errorMsg:    "forbidden property 'FIELD' found in YAML at path 'field' (case insensitive check)",
		},
		{
			name: "exact matching - case sensitive",
			value: reflect.ValueOf(struct {
				Field string `yaml:"field"`
			}{
				Field: "exact_test",
			}),
			path:        []string{"FIELD"},
			exactMatch:  true,
			currentPath: "",
			expectError: false, // Should not match due to case sensitivity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.checkForbiddenFieldAtPath(tt.value, tt.path, tt.exactMatch, tt.currentPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Comprehensive corner case tests for handleFieldType
func TestModelValidator_handleFieldType_CornerCases(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name          string
		fieldValue    reflect.Value
		remainingPath []string
		exactMatch    bool
		currentPath   string
		nextField     string
		expectError   bool
		errorMsg      string
	}{
		{
			name: "struct type",
			fieldValue: reflect.ValueOf(struct {
				Field string `yaml:"field"`
			}{Field: "test"}),
			remainingPath: []string{"field"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "nested",
			expectError:   true,
			errorMsg:      "forbidden property 'field' found in YAML",
		},
		{
			name: "map type",
			fieldValue: reflect.ValueOf(map[string]string{
				"key": "value",
			}),
			remainingPath: []string{"key"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "mapField",
			expectError:   true,
			errorMsg:      "forbidden property 'key' found in YAML",
		},
		{
			name: "pointer to struct",
			fieldValue: reflect.ValueOf(&struct {
				Field string `yaml:"field"`
			}{Field: "test"}),
			remainingPath: []string{"field"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "ptrField",
			expectError:   true,
			errorMsg:      "forbidden property 'field' found in YAML",
		},
		{
			name:          "nil pointer",
			fieldValue:    reflect.ValueOf((*struct{})(nil)),
			remainingPath: []string{"field"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "nilPtr",
			expectError:   false, // Nil pointer should be handled gracefully
		},
		{
			name: "interface with map",
			fieldValue: reflect.ValueOf(interface{}(map[string]string{
				"key": "value",
			})),
			remainingPath: []string{"key"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "interfaceField",
			expectError:   true,
			errorMsg:      "forbidden property 'key' found in YAML",
		},
		{
			name:          "nil interface",
			fieldValue:    reflect.ValueOf((*interface{})(nil)).Elem(),
			remainingPath: []string{"field"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "nilInterface",
			expectError:   false, // Nil interface should be handled gracefully
		},
		{
			name:          "string type (unsupported)",
			fieldValue:    reflect.ValueOf("string_value"),
			remainingPath: []string{"field"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "stringField",
			expectError:   false, // String type can't be navigated further
		},
		{
			name:          "int type (unsupported)",
			fieldValue:    reflect.ValueOf(42),
			remainingPath: []string{"field"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "intField",
			expectError:   false, // Int type can't be navigated further
		},
		{
			name:          "bool type (unsupported)",
			fieldValue:    reflect.ValueOf(true),
			remainingPath: []string{"field"},
			exactMatch:    true,
			currentPath:   "root",
			nextField:     "boolField",
			expectError:   false, // Bool type can't be navigated further
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.handleFieldType(tt.fieldValue, tt.remainingPath, tt.exactMatch, tt.currentPath, tt.nextField)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Comprehensive corner case tests for getFieldByName
func TestModelValidator_getFieldByName_CornerCases(t *testing.T) {
	validator := NewModelValidator()

	tests := []struct {
		name         string
		value        reflect.Value
		fieldName    string
		exactMatch   bool
		expectFound  bool
		expectedName string
	}{
		{
			name: "struct with yaml tags",
			value: reflect.ValueOf(struct {
				Field string `yaml:"field"`
			}{Field: "test"}),
			fieldName:    "field",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "field",
		},
		{
			name: "struct with yaml tags - case insensitive",
			value: reflect.ValueOf(struct {
				Field string `yaml:"field"`
			}{Field: "test"}),
			fieldName:    "FIELD",
			exactMatch:   false,
			expectFound:  true,
			expectedName: "field",
		},
		{
			name: "struct with yaml tags - case sensitive no match",
			value: reflect.ValueOf(struct {
				Field string `yaml:"field"`
			}{Field: "test"}),
			fieldName:    "FIELD",
			exactMatch:   true,
			expectFound:  false,
			expectedName: "",
		},
		{
			name: "struct with yaml tag options",
			value: reflect.ValueOf(struct {
				Field string `yaml:"field,omitempty"`
			}{Field: "test"}),
			fieldName:    "field",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "field",
		},
		{
			name: "struct with yaml tag inline",
			value: reflect.ValueOf(struct {
				Field string `yaml:"field,inline"`
			}{Field: "test"}),
			fieldName:    "field",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "field",
		},
		{
			name: "struct with yaml tag flow",
			value: reflect.ValueOf(struct {
				Field string `yaml:"field,flow"`
			}{Field: "test"}),
			fieldName:    "field",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "field",
		},
		{
			name: "struct with yaml tag dash (ignored)",
			value: reflect.ValueOf(struct {
				Field string `yaml:"-"`
			}{Field: "test"}),
			fieldName:    "field",
			exactMatch:   true,
			expectFound:  false,
			expectedName: "",
		},
		{
			name: "struct with empty yaml tag",
			value: reflect.ValueOf(struct {
				Field string `yaml:""`
			}{Field: "test"}),
			fieldName:    "field",
			exactMatch:   true,
			expectFound:  false,
			expectedName: "",
		},
		{
			name: "struct with inline yaml tag",
			value: reflect.ValueOf(struct {
				Field string `yaml:",inline"`
			}{Field: "test"}),
			fieldName:    "field",
			exactMatch:   true,
			expectFound:  false,
			expectedName: "",
		},
		{
			name: "map with string keys",
			value: reflect.ValueOf(map[string]string{
				"key": "value",
			}),
			fieldName:    "key",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "key",
		},
		{
			name: "map with string keys - case insensitive",
			value: reflect.ValueOf(map[string]string{
				"key": "value",
			}),
			fieldName:    "KEY",
			exactMatch:   false,
			expectFound:  true,
			expectedName: "key",
		},
		{
			name: "map with int keys",
			value: reflect.ValueOf(map[int]string{
				123: "value",
			}),
			fieldName:    "123",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "123",
		},
		{
			name: "map with interface keys",
			value: reflect.ValueOf(map[interface{}]string{
				"key": "value",
			}),
			fieldName:    "key",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "key",
		},
		{
			name:         "non-struct non-map type",
			value:        reflect.ValueOf("string_value"),
			fieldName:    "field",
			exactMatch:   true,
			expectFound:  false,
			expectedName: "",
		},
		{
			name: "struct with unicode field names",
			value: reflect.ValueOf(map[string]string{
				"测试字段": "unicode_value",
			}),
			fieldName:    "测试字段",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "测试字段",
		},
		{
			name: "struct with special character field names",
			value: reflect.ValueOf(map[string]string{
				"field-with-dashes": "dash_value",
				"field.with.dots":   "dot_value",
				"field with spaces": "space_value",
			}),
			fieldName:    "field-with-dashes",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "field-with-dashes",
		},
		{
			name: "very long field names",
			value: reflect.ValueOf(map[string]string{
				strings.Repeat("long", 250): "long_value",
			}),
			fieldName:    strings.Repeat("long", 250),
			exactMatch:   true,
			expectFound:  true,
			expectedName: strings.Repeat("long", 250),
		},
		{
			name: "empty field name",
			value: reflect.ValueOf(map[string]string{
				"": "empty_key_value",
			}),
			fieldName:    "",
			exactMatch:   true,
			expectFound:  true,
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue, actualName, found := validator.getFieldByName(tt.value, tt.fieldName, tt.exactMatch)

			if found != tt.expectFound {
				t.Errorf("Expected found=%v, got found=%v", tt.expectFound, found)
			}

			if found && actualName != tt.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tt.expectedName, actualName)
			}

			if found && !fieldValue.IsValid() {
				t.Errorf("Expected valid field value when found=true")
			}
		})
	}
}

// Performance and stress tests
func TestModelValidator_Performance(t *testing.T) {
	validator := NewModelValidator()

	t.Run("large model group validation", func(t *testing.T) {
		// Create a large model group with many models
		group := &config.ModelGroup{
			Infrastructure: types.InfrastructureAWS,
			Provider:       types.ProviderBedrock,
			Creator:        types.CreatorAmazon,
			Models:         make([]config.EnhancedModelConfig, 100),
		}

		// Fill with valid models
		for i := 0; i < 100; i++ {
			group.Models[i] = config.EnhancedModelConfig{
				Name:                   fmt.Sprintf("model-%d", i),
				Version:                fmt.Sprintf("v%d", i),
				FunctionalCapabilities: []string{"chat_completion"},
				Endpoints:              []config.EndpointConfig{{Path: "/chat/completions"}},
				Parameters: map[string]config.ParameterSpec{
					"maxInputTokens":  {Type: "integer"},
					"maxOutputTokens": {Type: "integer", Maximum: 1000},
				},
			}
		}

		// This should complete without timeout
		err := validator.ValidateModelGroup(group, "aws/bedrock/amazon/test.yaml")
		if err != nil {
			t.Errorf("Unexpected error in large model validation: %v", err)
		}

		// Verify all models got IDs assigned
		for i, model := range group.Models {
			expectedID := fmt.Sprintf("aws/bedrock/amazon/model-%d/v%d", i, i)
			if model.KEY != expectedID {
				t.Errorf("Model %d: expected ID '%s', got '%s'", i, expectedID, model.KEY)
			}
		}
	})

	t.Run("deep nested forbidden field check", func(t *testing.T) {
		// Create deeply nested structure
		deepMap := make(map[string]interface{})
		current := deepMap
		for i := 0; i < 50; i++ {
			next := make(map[string]interface{})
			current[fmt.Sprintf("level%d", i)] = next
			current = next
		}
		current["forbidden"] = "found"

		// Create path to the forbidden field
		path := make([]string, 51)
		for i := 0; i < 50; i++ {
			path[i] = fmt.Sprintf("level%d", i)
		}
		path[50] = "forbidden"

		err := validator.checkForbiddenFieldAtPath(reflect.ValueOf(deepMap), path, true, "")
		if err == nil {
			t.Errorf("Expected error for deeply nested forbidden field")
		}
	})
}
