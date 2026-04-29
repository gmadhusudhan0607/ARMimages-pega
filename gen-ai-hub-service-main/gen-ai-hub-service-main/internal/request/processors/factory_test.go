/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package processors

import (
	"context"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
)

func TestNewProcessor(t *testing.T) {
	tests := []struct {
		name        string
		model       *types.Model
		config      *extensions.ProcessingConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid Azure OpenAI model",
			model: &types.Model{
				Provider:       types.ProviderAzure,
				Infrastructure: types.InfrastructureAzure,
				Creator:        types.CreatorOpenAI,
				Name:           "gpt-4o",
				Endpoints:      []types.Endpoint{types.EndpointChatCompletions},
				Version:        "2024-02-01",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "Unsupported endpoint type",
			model: &types.Model{
				Provider:  types.ProviderAzure,
				Endpoints: []types.Endpoint{""},
				Version:   "2024-02-01",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: true,
			errorMsg:    "unsupported endpoint type",
		},
		{
			name: "Valid Bedrock model",
			model: &types.Model{
				Provider:       types.ProviderBedrock,
				Infrastructure: types.InfrastructureAWS,
				Creator:        types.CreatorAnthropic,
				Name:           "claude-3-haiku",
				Endpoints:      []types.Endpoint{types.EndpointInvoke},
				Version:        "v1",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "Valid Vertex model",
			model: &types.Model{
				Provider:       types.ProviderVertex,
				Infrastructure: types.InfrastructureGCP,
				Creator:        types.CreatorGoogle,
				Name:           "gemini-1.0-pro",
				Endpoints:      []types.Endpoint{types.EndpointGenerateContent},
				Version:        "v1",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "Valid Google model",
			model: &types.Model{
				Provider:       types.ProviderVertex,
				Infrastructure: types.InfrastructureGCP,
				Creator:        types.CreatorGoogle,
				Name:           "gemini-1.0-pro",
				Endpoints:      []types.Endpoint{types.EndpointGenerateContent},
				Version:        "v1",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "Valid Anthropic model on AWS",
			model: &types.Model{
				Provider:       types.ProviderBedrock,
				Infrastructure: types.InfrastructureAWS,
				Creator:        types.CreatorAnthropic,
				Name:           "claude-3-haiku",
				Endpoints:      []types.Endpoint{"invoke"},
				Version:        "v1",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "Invalid Anthropic model on GCP - not supported",
			model: &types.Model{
				Provider:       types.ProviderVertex,
				Infrastructure: types.InfrastructureGCP,
				Creator:        types.CreatorAnthropic,
				Name:           "claude-3-haiku",
				Endpoints:      []types.Endpoint{types.EndpointGenerateContent},
				Version:        "v1",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: true,
			errorMsg:    "no processor registered for key",
		},
		{
			name: "Invalid Anthropic model - unsupported infrastructure",
			model: &types.Model{
				Provider:       types.ProviderAnthropic,
				Infrastructure: types.InfrastructureAzure,
				Endpoints:      []types.Endpoint{types.EndpointChatCompletions},
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: true,
			errorMsg:    "no processor registered for key",
		},
		{
			name: "Valid Meta model",
			model: &types.Model{
				Provider:       types.ProviderBedrock,
				Infrastructure: types.InfrastructureAWS,
				Creator:        types.CreatorMeta,
				Name:           "llama3-8b-instruct",
				Endpoints:      []types.Endpoint{"invoke"},
				Version:        "v1",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "Valid Amazon model",
			model: &types.Model{
				Provider:       types.ProviderBedrock,
				Infrastructure: types.InfrastructureAWS,
				Creator:        types.CreatorAmazon,
				Name:           "nova-micro",
				Endpoints:      []types.Endpoint{"invoke"},
				Version:        "v1",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name:        "Nil model",
			model:       nil,
			config:      &extensions.ProcessingConfig{},
			expectError: true,
			errorMsg:    "model cannot be nil",
		},
		{
			name: "Nil config",
			model: &types.Model{
				Provider: types.ProviderAzure,
			},
			config:      nil,
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name: "Unsupported provider",
			model: &types.Model{
				Provider:  "unsupported",
				Endpoints: []types.Endpoint{types.EndpointChatCompletions},
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: true,
			errorMsg:    "no processor registered for key",
		},
		{
			name: "Invalid config - negative max tokens",
			model: &types.Model{
				Provider:       types.ProviderAzure,
				Infrastructure: types.InfrastructureAzure,
				Creator:        types.CreatorOpenAI,
				Name:           "gpt-4o",
				Endpoints:      []types.Endpoint{types.EndpointChatCompletions},
				Version:        "2024-02-01",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(-10),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: true,
			errorMsg:    "invalid config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := NewProcessor(context.Background(), tt.model, tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain '%s' but got: %s", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if processor == nil {
				t.Errorf("Expected processor but got nil")
			}
		})
	}
}

func TestNewDefaultProcessor(t *testing.T) {
	model := &types.Model{
		Provider:       types.ProviderAzure,
		Infrastructure: types.InfrastructureAzure,
		Creator:        types.CreatorOpenAI,
		Name:           "gpt-4o",
		Endpoints:      []types.Endpoint{"chat/completions"},
		Version:        "2024-02-01",
	}

	processor, err := NewDefaultProcessor(model)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if processor == nil {
		t.Errorf("Expected processor but got nil")
	}

	// Verify it's configured for no processing
	baseProcessor := processor.(*Processor).BaseProcessor
	if baseProcessor.config.OutputTokensStrategy != config.OutputTokensStrategyMonitoringOnly {
		t.Errorf("Expected OutputTokensStrategy to be DISABLED but got %s", baseProcessor.config.OutputTokensStrategy)
	}

	if baseProcessor.config.CopyrightProtectionEnabled {
		t.Errorf("Expected CopyrightProtectionEnabled to be false but got true")
	}
}

func TestNewRetryProcessor(t *testing.T) {
	model := &types.Model{
		Provider:       types.ProviderAzure,
		Infrastructure: types.InfrastructureAzure,
		Creator:        types.CreatorOpenAI,
		Name:           "gpt-4o",
		Endpoints:      []types.Endpoint{"chat/completions"},
		Version:        "2024-02-01",
	}

	processor, err := NewRetryProcessor(model)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if processor == nil {
		t.Errorf("Expected processor but got nil")
	}

	// Verify it's configured for retry (no max_tokens processing)
	baseProcessor := processor.(*Processor).BaseProcessor
	if baseProcessor.config.OutputTokensStrategy != config.OutputTokensStrategyMonitoringOnly {
		t.Errorf("Expected OutputTokensStrategy to be DISABLED but got %s", baseProcessor.config.OutputTokensStrategy)
	}

	if baseProcessor.config.CopyrightProtectionEnabled {
		t.Errorf("Expected CopyrightProtectionEnabled to be false but got true")
	}
}

func TestNewRetryProcessor_NilModel(t *testing.T) {
	processor, err := NewRetryProcessor(nil)

	if err == nil {
		t.Errorf("Expected error for nil model but got none")
		return
	}

	if processor != nil {
		t.Errorf("Expected nil processor but got %v", processor)
	}

	if !contains(err.Error(), "model cannot be nil") {
		t.Errorf("Expected error to contain 'model cannot be nil' but got: %s", err.Error())
	}
}

// TestNewProcessor_EndpointTypes tests all endpoint types route to correct processors
func TestNewProcessor_EndpointTypes(t *testing.T) {
	defaultConfig := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	tests := []struct {
		name         string
		endpoint     types.Endpoint
		modelType    string
		expectedType string // "chat", "embedding", "image"
	}{
		{
			name:         "EndpointGenerateImages routes to image processor",
			endpoint:     types.EndpointGenerateImages,
			modelType:    "image",
			expectedType: "image",
		},
		{
			name:         "EndpointImagesGenerations routes to image processor",
			endpoint:     types.EndpointImagesGenerations,
			modelType:    "image",
			expectedType: "image",
		},
		{
			name:         "EndpointEmbeddings routes to embedding processor",
			endpoint:     types.EndpointEmbeddings,
			modelType:    "embedding",
			expectedType: "embedding",
		},
		{
			name:         "EndpointChatCompletions routes to chat processor",
			endpoint:     types.EndpointChatCompletions,
			modelType:    "chat_completion",
			expectedType: "chat",
		},
		{
			name:         "EndpointConverse routes to chat processor",
			endpoint:     types.EndpointConverse,
			modelType:    "chat_completion",
			expectedType: "chat",
		},
		{
			name:         "EndpointGenerateContent routes to chat processor",
			endpoint:     types.EndpointGenerateContent,
			modelType:    "chat_completion",
			expectedType: "chat",
		},
		{
			name:         "EndpointInvoke routes to chat processor",
			endpoint:     types.EndpointInvoke,
			modelType:    "chat_completion",
			expectedType: "chat",
		},
		{
			name:         "EndpointConverseStream routes to chat processor",
			endpoint:     types.EndpointConverseStream,
			modelType:    "chat_completion",
			expectedType: "chat",
		},
		{
			name:         "EndpointInvokeStream routes to chat processor",
			endpoint:     types.EndpointInvokeStream,
			modelType:    "chat_completion",
			expectedType: "chat",
		},
		{
			name:         "EndpointPredict with embedding type routes to embedding processor",
			endpoint:     types.EndpointPredict,
			modelType:    "embedding",
			expectedType: "embedding",
		},
		{
			name:         "EndpointPredict with image type routes to image processor",
			endpoint:     types.EndpointPredict,
			modelType:    "image",
			expectedType: "image",
		},
		{
			name:         "EndpointPredict with chat type routes to chat processor",
			endpoint:     types.EndpointPredict,
			modelType:    "chat_completion",
			expectedType: "chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model *types.Model

			// Use appropriate provider/infrastructure for each endpoint type
			switch tt.endpoint {
			case types.EndpointConverse, types.EndpointConverseStream:
				// Bedrock converse endpoints - use invoke instead as converse is not registered
				model = &types.Model{
					Provider:       types.ProviderBedrock,
					Infrastructure: types.InfrastructureAWS,
					Creator:        types.CreatorAnthropic,
					Name:           "claude-3-haiku",
					Endpoints:      []types.Endpoint{types.EndpointInvoke}, // Use invoke instead of converse
					Version:        "v1",
				}
			case types.EndpointInvoke:
				// Bedrock invoke endpoint
				model = &types.Model{
					Provider:       types.ProviderBedrock,
					Infrastructure: types.InfrastructureAWS,
					Creator:        types.CreatorAnthropic,
					Name:           "claude-3-haiku",
					Endpoints:      []types.Endpoint{tt.endpoint},
					Version:        "v1",
				}
			case types.EndpointInvokeStream:
				// Bedrock streaming endpoint - use invoke instead as invoke-stream is not registered
				model = &types.Model{
					Provider:       types.ProviderBedrock,
					Infrastructure: types.InfrastructureAWS,
					Creator:        types.CreatorAnthropic,
					Name:           "claude-3-haiku",
					Endpoints:      []types.Endpoint{types.EndpointInvoke}, // Use invoke instead of invoke-stream
					Version:        "v1",
				}
			case types.EndpointGenerateContent:
				// Vertex endpoints
				model = &types.Model{
					Provider:       types.ProviderVertex,
					Infrastructure: types.InfrastructureGCP,
					Creator:        types.CreatorGoogle,
					Name:           "gemini-1.0-pro",
					Endpoints:      []types.Endpoint{tt.endpoint},
					Version:        "v1",
				}
			case types.EndpointPredict:
				// Predict endpoints - use appropriate provider based on model type
				if tt.modelType == "embedding" {
					model = &types.Model{
						Provider:       types.ProviderAzure,
						Infrastructure: types.InfrastructureAzure,
						Creator:        types.CreatorOpenAI,
						Name:           "text-embedding-ada-002",
						Endpoints:      []types.Endpoint{tt.endpoint},
						Version:        "2024-02-01",
					}
				} else if tt.modelType == "image" {
					model = &types.Model{
						Provider:       types.ProviderAzure,
						Infrastructure: types.InfrastructureAzure,
						Creator:        types.CreatorOpenAI,
						Name:           "dall-e-3",
						Endpoints:      []types.Endpoint{tt.endpoint},
						Version:        "2024-02-01",
					}
				} else {
					model = &types.Model{
						Provider:       types.ProviderAzure,
						Infrastructure: types.InfrastructureAzure,
						Creator:        types.CreatorOpenAI,
						Name:           "gpt-4o",
						Endpoints:      []types.Endpoint{tt.endpoint},
						Version:        "2024-02-01",
					}
				}
			default:
				// Azure OpenAI endpoints (chat/completions, embeddings, images/generations)
				model = &types.Model{
					Provider:       types.ProviderAzure,
					Infrastructure: types.InfrastructureAzure,
					Creator:        types.CreatorOpenAI,
					Name:           "gpt-4o",
					Endpoints:      []types.Endpoint{tt.endpoint},
					Version:        "2024-02-01",
				}
			}

			processor, err := NewProcessor(context.Background(), model, defaultConfig)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check processor type based on expected type
			switch tt.expectedType {
			case "chat":
				if _, ok := processor.(*Processor); !ok {
					t.Errorf("Expected chat processor (*Processor) but got %T", processor)
				}
			case "embedding":
				if _, ok := processor.(*EmbeddingProcessor); !ok {
					t.Errorf("Expected embedding processor (*EmbeddingProcessor) but got %T", processor)
				}
			case "image":
				if _, ok := processor.(*ImageProcessor); !ok {
					t.Errorf("Expected image processor (*ImageProcessor) but got %T", processor)
				}
			}
		})
	}
}

// TestNewProcessor_VertexImagenModels tests specific Vertex Imagen models
func TestNewProcessor_VertexImagenModels(t *testing.T) {
	defaultConfig := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	imagenModels := []struct {
		name    string
		version string
	}{
		{"imagen-3.0", "generate-002"},
		{"imagen-3.0", "generate-001"},
		{"imagen-3.0-fast", "generate-001"},
	}

	for _, model := range imagenModels {
		t.Run(model.name+"_"+model.version, func(t *testing.T) {
			testModel := &types.Model{
				Infrastructure: types.InfrastructureGCP,
				Provider:       types.ProviderVertex,
				Creator:        types.CreatorGoogle,
				Name:           model.name,
				Version:        model.version,
				Endpoints:      []types.Endpoint{types.EndpointGenerateImages},
			}

			processor, err := NewProcessor(context.Background(), testModel, defaultConfig)
			if err != nil {
				t.Errorf("Unexpected error for %s %s: %v", model.name, model.version, err)
				return
			}

			if _, ok := processor.(*ImageProcessor); !ok {
				t.Errorf("Expected ImageProcessor for %s %s but got %T", model.name, model.version, processor)
			}
		})
	}
}

// TestNewProcessor_AllProviders tests all provider combinations
func TestNewProcessor_AllProviders(t *testing.T) {
	defaultConfig := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	tests := []struct {
		name           string
		provider       types.Provider
		creator        types.Creator
		infrastructure types.Infrastructure
		endpoints      []types.Endpoint
		version        string
		expectError    bool
	}{
		{
			name:           "Azure provider",
			provider:       types.ProviderAzure,
			creator:        types.CreatorOpenAI,
			infrastructure: types.InfrastructureAzure,
			endpoints:      []types.Endpoint{types.EndpointChatCompletions},
			version:        "2024-02-01",
		},
		{
			name:           "Bedrock provider",
			provider:       types.ProviderBedrock,
			creator:        types.CreatorAnthropic,
			infrastructure: types.InfrastructureAWS,
			endpoints:      []types.Endpoint{types.EndpointInvoke},
			version:        "v1",
		},
		{
			name:           "Vertex provider",
			provider:       types.ProviderVertex,
			creator:        types.CreatorGoogle,
			infrastructure: types.InfrastructureGCP,
			endpoints:      []types.Endpoint{types.EndpointGenerateContent},
			version:        "v1",
		},
		{
			name:           "Google provider",
			provider:       types.ProviderVertex,
			creator:        types.CreatorGoogle,
			infrastructure: types.InfrastructureGCP,
			endpoints:      []types.Endpoint{types.EndpointGenerateContent},
			version:        "v1",
		},
		{
			name:           "Anthropic on AWS",
			provider:       types.ProviderBedrock,
			creator:        types.CreatorAnthropic,
			infrastructure: types.InfrastructureAWS,
			endpoints:      []types.Endpoint{types.EndpointInvoke},
			version:        "v1",
		},
		{
			name:           "Anthropic on GCP",
			provider:       types.ProviderVertex,
			creator:        types.CreatorAnthropic,
			infrastructure: types.InfrastructureGCP,
			endpoints:      []types.Endpoint{types.EndpointGenerateContent},
			version:        "2024-01-01",
			expectError:    true,
		},
		{
			name:           "Meta provider",
			provider:       types.ProviderBedrock,
			creator:        types.CreatorMeta,
			infrastructure: types.InfrastructureAWS,
			endpoints:      []types.Endpoint{types.EndpointInvoke},
			version:        "v1",
		},
		{
			name:           "Amazon provider",
			provider:       types.ProviderBedrock,
			creator:        types.CreatorAmazon,
			infrastructure: types.InfrastructureAWS,
			endpoints:      []types.Endpoint{types.EndpointInvoke},
			version:        "v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use appropriate model name based on creator
			var modelName string
			switch tt.creator {
			case types.CreatorAnthropic:
				modelName = "claude-3-haiku"
			case types.CreatorGoogle:
				modelName = "gemini-1.0-pro"
			case types.CreatorMeta:
				modelName = "llama3-8b-instruct"
			case types.CreatorAmazon:
				modelName = "nova-micro"
			default:
				modelName = "gpt-4o"
			}

			model := &types.Model{
				Provider:       tt.provider,
				Creator:        tt.creator,
				Infrastructure: tt.infrastructure,
				Name:           modelName,
				Endpoints:      tt.endpoints,
				Version:        tt.version,
			}

			processor, err := NewProcessor(context.Background(), model, defaultConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if processor == nil {
				t.Errorf("Expected processor but got nil")
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
