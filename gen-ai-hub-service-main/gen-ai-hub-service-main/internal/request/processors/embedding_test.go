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

func TestNewEmbeddingProcessorImpl(t *testing.T) {
	tests := []struct {
		name        string
		model       *types.Model
		config      *extensions.ProcessingConfig
		expectError bool
	}{
		{
			name: "Valid Azure embedding model",
			model: &types.Model{
				Provider:  types.ProviderAzure,
				Creator:   types.CreatorOpenAI,
				Endpoints: []types.Endpoint{"embeddings"},
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
			name: "Valid Bedrock embedding model",
			model: &types.Model{
				Provider:       types.ProviderBedrock,
				Infrastructure: types.InfrastructureAWS,
				Creator:        types.CreatorAmazon,
				KEY:            "titan-embed-text",
				Name:           "titan-embed-text",
				Version:        "v2",
				Endpoints:      []types.Endpoint{"embeddings"},
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
			name: "Valid Vertex embedding model",
			model: &types.Model{
				Provider:       types.ProviderVertex,
				Infrastructure: types.InfrastructureGCP,
				Creator:        types.CreatorGoogle,
				KEY:            "text-multilingual-embedding",
				Name:           "text-multilingual-embedding",
				Version:        "002",
				Endpoints:      []types.Endpoint{"embeddings"},
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
		},
		{
			name: "Nil config",
			model: &types.Model{
				Provider:  types.ProviderAzure,
				Creator:   types.CreatorOpenAI,
				Endpoints: []types.Endpoint{"embeddings"},
			},
			config:      nil,
			expectError: true,
		},
		{
			name: "Unsupported provider",
			model: &types.Model{
				Provider:  types.Provider("unsupported"),
				Creator:   types.CreatorOpenAI,
				Endpoints: []types.Endpoint{"embeddings"},
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
			processor, err := NewEmbeddingProcessorImpl(context.Background(), tt.model, tt.config)

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

			// Verify it's an EmbeddingProcessor
			if _, ok := processor.(*EmbeddingProcessor); !ok {
				t.Errorf("Expected EmbeddingProcessor but got %T", processor)
			}
		})
	}
}

func TestEmbeddingProcessor_ProcessRequest(t *testing.T) {
	model := &types.Model{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorOpenAI,
		Endpoints: []types.Endpoint{"embeddings"},

		Version: "2023-05-15",
	}

	config := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	processor, err := NewEmbeddingProcessorImpl(context.Background(), model, config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	tests := []struct {
		name         string
		requestBody  string
		expectChange bool
	}{
		{
			name:         "Basic embedding request",
			requestBody:  `{"input": "Hello world", "model": "text-embedding-ada-002"}`,
			expectChange: false,
		},
		{
			name:         "Embedding request with max_tokens (should be ignored)",
			requestBody:  `{"input": "Hello world", "model": "text-embedding-ada-002", "max_tokens": 100}`,
			expectChange: false,
		},
		{
			name:         "Empty request",
			requestBody:  `{}`,
			expectChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// For embeddings, the body should not be modified
			if string(result.ModifiedBody) != tt.requestBody {
				t.Errorf("Expected body to remain unchanged, got: %s", string(result.ModifiedBody))
			}

			// Embeddings don't have token tracking
			if result.OriginalTokens != nil {
				t.Errorf("Expected OriginalTokens to be nil for embeddings, got: %v", *result.OriginalTokens)
			}

			if result.ModifiedTokens != nil {
				t.Errorf("Expected ModifiedTokens to be nil for embeddings, got: %v", *result.ModifiedTokens)
			}

			// System prompts not applicable to embeddings
			if result.HasSystemPrompt {
				t.Errorf("Expected HasSystemPrompt to be false for embeddings")
			}
		})
	}
}

func TestEmbeddingProcessor_ProcessResponse(t *testing.T) {
	model := &types.Model{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorOpenAI,
		Endpoints: []types.Endpoint{"embeddings"},

		Version: "2023-05-15",
	}

	config := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	processor, err := NewEmbeddingProcessorImpl(context.Background(), model, config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	tests := []struct {
		name         string
		responseBody string
		statusCode   int
	}{
		{
			name: "Successful embedding response",
			responseBody: `{
				"data": [{"embedding": [0.1, 0.2, 0.3], "index": 0}],
				"usage": {"prompt_tokens": 5, "total_tokens": 5}
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

			// Embeddings cannot be truncated
			if result.WasTruncated {
				t.Errorf("Expected WasTruncated to be false for embeddings")
			}

			// Finish reason should be "completed"
			if result.FinishReason != "completed" {
				t.Errorf("Expected FinishReason to be 'completed', got: %s", result.FinishReason)
			}
		})
	}
}

func TestEmbeddingProcessor_UpdateMetrics(t *testing.T) {
	model := &types.Model{
		Provider:  types.ProviderAzure,
		Creator:   types.CreatorOpenAI,
		Endpoints: []types.Endpoint{"embeddings"},

		Version: "2023-05-15",
	}

	config := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	processor, err := NewEmbeddingProcessorImpl(context.Background(), model, config)
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
			ModifiedBody: []byte(`{"input": "test"}`),
		}

		usedTokens := 10
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
		} else if *metadata.RequestMetrics.TokenMetrics.Used != 10.0 {
			t.Errorf("Expected Used tokens to be 10.0, got: %f", *metadata.RequestMetrics.TokenMetrics.Used)
		}

		// Embeddings don't have retry metrics
		if metadata.RequestMetrics.RetryMetrics.ResponseTruncated {
			t.Errorf("Expected ResponseTruncated to be false for embeddings")
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
