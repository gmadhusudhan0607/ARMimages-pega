/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package processors

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
)

func TestNewImageProcessorImpl(t *testing.T) {
	tests := []struct {
		name        string
		model       *types.Model
		config      *extensions.ProcessingConfig
		expectError bool
	}{
		{
			name: "Valid Azure image model",
			model: &types.Model{
				Provider:  types.ProviderAzure,
				Creator:   types.CreatorOpenAI,
				Endpoints: []types.Endpoint{types.EndpointImagesGenerations},
				Version:   "2023-05-15",
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
			name: "Valid Vertex image model",
			model: &types.Model{
				Provider:       types.ProviderVertex,
				Infrastructure: types.InfrastructureGCP,
				Creator:        types.CreatorGoogle,
				Name:           "imagen-3.0",
				Endpoints:      []types.Endpoint{types.EndpointImagesGenerations},
				Version:        "generate-001",
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   true,
			},
			expectError: false,
		},
		{
			name:        "Nil model",
			model:       nil,
			config:      &extensions.ProcessingConfig{},
			expectError: true,
		},
		{
			name: "Nil config",
			model: &types.Model{
				Provider:  types.ProviderAzure,
				Creator:   types.CreatorOpenAI,
				Endpoints: []types.Endpoint{types.EndpointImagesGenerations},
			},
			config:      nil,
			expectError: true,
		},
		{
			name: "Unsupported provider",
			model: &types.Model{
				Provider:  types.ProviderBedrock, // Bedrock doesn't support image generation in this implementation
				Creator:   types.CreatorAmazon,
				Endpoints: []types.Endpoint{types.EndpointImagesGenerations},
			},
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := NewImageProcessorImpl(context.Background(), tt.model, tt.config)

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
				return
			}

			// Verify it's an ImageProcessor
			if _, ok := processor.(*ImageProcessor); !ok {
				t.Errorf("Expected ImageProcessor but got %T", processor)
			}
		})
	}
}

func TestImageProcessor_ProcessRequest(t *testing.T) {
	tests := []struct {
		name         string
		config       *extensions.ProcessingConfig
		requestBody  string
		expectChange bool
	}{
		{
			name: "Basic image request - no processing",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			requestBody:  `{"prompt": "A beautiful sunset", "size": "1024x1024"}`,
			expectChange: false,
		},
		{
			name: "Image request with copyright protection",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   true,
			},
			requestBody:  `{"prompt": "A beautiful sunset", "size": "1024x1024"}`,
			expectChange: true,
		},
		{
			name: "Image request with max_tokens (should be ignored)",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(100),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			requestBody:  `{"prompt": "A beautiful sunset", "size": "1024x1024", "max_tokens": 50}`,
			expectChange: false, // max_tokens should be ignored for images
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &types.Model{
				Provider:  types.ProviderAzure,
				Creator:   types.CreatorOpenAI,
				Endpoints: []types.Endpoint{types.EndpointImagesGenerations},
				Version:   "2023-05-15",
			}

			processor, err := NewImageProcessorImpl(context.Background(), model, tt.config)
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}

			ctx := context.Background()
			result, err := processor.ProcessRequest(ctx, []byte(tt.requestBody))

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			// Check if body was modified as expected
			bodyChanged := string(result.ModifiedBody) != tt.requestBody
			if bodyChanged != tt.expectChange {
				t.Errorf("Expected body change: %v, got change: %v", tt.expectChange, bodyChanged)
				t.Errorf("Original: %s", tt.requestBody)
				t.Errorf("Modified: %s", string(result.ModifiedBody))
			}

			// Images don't have token tracking
			if result.OriginalTokens != nil {
				t.Errorf("Expected OriginalTokens to be nil for images, got: %v", *result.OriginalTokens)
			}

			if result.ModifiedTokens != nil {
				t.Errorf("Expected ModifiedTokens to be nil for images, got: %v", *result.ModifiedTokens)
			}

			// Check copyright protection processing
			expectedHasPrompt := tt.config.CopyrightProtectionEnabled
			if result.HasSystemPrompt != expectedHasPrompt {
				t.Errorf("Expected HasSystemPrompt: %v, got: %v", expectedHasPrompt, result.HasSystemPrompt)
			}
		})
	}
}

func TestImageProcessor_ProcessResponse(t *testing.T) {
	model := &types.Model{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorOpenAI,
		Endpoints: []types.Endpoint{types.EndpointImagesGenerations},
		Version:   "2023-05-15",
	}

	config := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	processor, err := NewImageProcessorImpl(context.Background(), model, config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	tests := []struct {
		name         string
		responseBody string
		statusCode   int
	}{
		{
			name: "Successful image generation response",
			responseBody: `{
				"data": [{"url": "https://example.com/image.png"}],
				"created": 1677649420
			}`,
			statusCode: 200,
		},
		{
			name: "Image generation with usage info",
			responseBody: `{
				"data": [{"url": "https://example.com/image.png"}],
				"usage": {"prompt_tokens": 10},
				"created": 1677649420
			}`,
			statusCode: 200,
		},
		{
			name:         "Empty response",
			responseBody: `{}`,
			statusCode:   200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				Header:     make(http.Header),
			}

			result, err := processor.ProcessResponse(ctx, resp)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			// Images cannot be truncated
			if result.WasTruncated {
				t.Errorf("Expected WasTruncated to be false for images")
			}

			// Finish reason should be "completed"
			if result.FinishReason != "completed" {
				t.Errorf("Expected FinishReason to be 'completed', got: %s", result.FinishReason)
			}
		})
	}
}

func TestImageProcessor_UpdateMetrics(t *testing.T) {
	model := &types.Model{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorOpenAI,
		Endpoints: []types.Endpoint{types.EndpointImagesGenerations},
		Version:   "2023-05-15",
	}

	config := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	processor, err := NewImageProcessorImpl(context.Background(), model, config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	t.Run("Update metrics with usage", func(t *testing.T) {
		metadata := &metadata.RequestMetadata{
			RequestMetrics: metrics.RequestMetrics{
				TokenMetrics: metrics.TokenMetrics{},
				RetryMetrics: metrics.RetryMetrics{},
			},
		}

		req := &ProcessedRequest{
			ModifiedBody: []byte(`{"prompt": "test image"}`),
		}

		usedTokens := 15
		resp := &extensions.ProcessedResponse{
			UsedTokens:   &usedTokens,
			WasTruncated: false,
			FinishReason: "completed",
		}

		err := processor.UpdateMetrics(metadata, req, resp)
		if err != nil {
			t.Errorf("Unexpected error updating metrics: %v", err)
		}

		if metadata.RequestMetrics.TokenMetrics.Used == nil {
			t.Errorf("Expected Used tokens to be set")
		} else if *metadata.RequestMetrics.TokenMetrics.Used != 15.0 {
			t.Errorf("Expected Used tokens to be 15.0, got: %f", *metadata.RequestMetrics.TokenMetrics.Used)
		}

		// Images don't have retry metrics
		if metadata.RequestMetrics.RetryMetrics.ResponseTruncated {
			t.Errorf("Expected ResponseTruncated to be false for images")
		}
	})

	t.Run("Update metrics with nil metadata", func(t *testing.T) {
		req := &ProcessedRequest{}
		resp := &extensions.ProcessedResponse{}

		// Should return error for nil metadata
		err := processor.UpdateMetrics(nil, req, resp)
		if err == nil {
			t.Errorf("Expected error for nil metadata but got none")
		}
	})
}

func TestImageProcessor_PromptProcessing(t *testing.T) {
	model := &types.Model{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorOpenAI,
		Endpoints: []types.Endpoint{types.EndpointImagesGenerations},
		Version:   "2023-05-15",
	}

	tests := []struct {
		name                string
		copyrightProtection bool
		requestBody         string
		expectModification  bool
	}{
		{
			name:                "No copyright protection",
			copyrightProtection: false,
			requestBody:         `{"prompt": "sunset"}`,
			expectModification:  false,
		},
		{
			name:                "With copyright protection",
			copyrightProtection: true,
			requestBody:         `{"prompt": "sunset"}`,
			expectModification:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   tt.copyrightProtection,
			}

			processor, err := NewImageProcessorImpl(context.Background(), model, config)
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}

			ctx := context.Background()
			result, err := processor.ProcessRequest(ctx, []byte(tt.requestBody))

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			// Check if the request was modified as expected
			wasModified := string(result.ModifiedBody) != tt.requestBody
			if wasModified != tt.expectModification {
				t.Errorf("Expected modification: %v, got modification: %v", tt.expectModification, wasModified)
				t.Errorf("Original: %s", tt.requestBody)
				t.Errorf("Modified: %s", string(result.ModifiedBody))
			}

			if result.HasSystemPrompt != tt.copyrightProtection {
				t.Errorf("Expected HasSystemPrompt: %v, got: %v", tt.copyrightProtection, result.HasSystemPrompt)
			}
		})
	}
}
