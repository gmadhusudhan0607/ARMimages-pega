/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package processors

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/registry"
)

// EmbeddingProcessor implements specialized processing for embedding endpoints
type EmbeddingProcessor struct {
	*BaseProcessor
}

// NewEmbeddingProcessorImpl creates a new embedding processor with provider-specific extension
func NewEmbeddingProcessorImpl(ctx context.Context, model *types.Model, config *extensions.ProcessingConfig) (RequestProcessor, error) {
	if model == nil {
		return nil, fmt.Errorf("model cannot be nil")
	}

	logger := cntx.LoggerFromContext(ctx).Sugar()
	logger.Debugf("Creating embedding processor for model %s (provider: %s)", model.Name, model.Provider)

	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Try to get extension from registry first
	processorKey := registry.CreateProcessorKey(model)
	reg := registry.GetGlobalRegistry()

	var extension ProviderExtension

	if reg.HasProcessor(processorKey) {
		logger.Debugf("Using registry to create embedding processor for key: %s", processorKey.String())
		extensionInterface, err := reg.CreateProcessor(processorKey)
		if err != nil {
			logger.Warnf("Failed to create processor from registry: %v", err)
			return nil, fmt.Errorf("failed to create processor from registry: %w", err)
		}

		// Type assert to ProviderExtension
		var ok bool
		extension, ok = extensionInterface.(ProviderExtension)
		if !ok {
			logger.Warnf("Registry returned invalid processor type for key: %s", processorKey.String())
			return nil, fmt.Errorf("registry returned invalid processor type for key: %s", processorKey.String())
		}
	} else {
		// Fallback to switch statements for unknown combinations
		logger.Debugf("No registry entry found for key: %s, using fallback logic", processorKey.String())

		switch model.Provider {
		case types.ProviderAzure:
			logger.Debugf("Creating Azure OpenAI 2024-02-01 extension for Azure embeddings")
			extension = extensions.NewAzureOpenAI20240201Extension()
		default:
			logger.Errorf("No processor registered for key: %s", processorKey.String())
			return nil, fmt.Errorf("no processor registered for key: %s", processorKey.String())
		}
	}

	// Create embedding-specific configuration
	embeddingConfig := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategy,
		OutputTokensBaseValue:        config.OutputTokensBaseValue,
		OutputTokensAdjustmentForced: false, // Embeddings don't use max_tokens adjustment
		CopyrightProtectionEnabled:   false, // System prompts not applicable to embeddings
	}

	logger.Debugf("Embedding processor created successfully for model %s", model.Name)
	return &EmbeddingProcessor{
		BaseProcessor: NewBaseProcessor(extension, embeddingConfig),
	}, nil
}

// ProcessRequest processes embedding requests (minimal processing needed)
func (p *EmbeddingProcessor) ProcessRequest(ctx context.Context, body []byte) (*ProcessedRequest, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	logger.Debug("Processing embedding request - minimal processing required")

	// For embeddings, we typically don't modify the request
	// Just return the original body without processing max_tokens or system prompts
	return &ProcessedRequest{
		ModifiedBody:    body,
		OriginalTokens:  nil, // Not applicable for embeddings
		ModifiedTokens:  nil,
		HasSystemPrompt: false,
	}, nil
}

// ProcessResponse processes embedding responses (extract usage metrics)
func (p *EmbeddingProcessor) ProcessResponse(ctx context.Context, resp *http.Response) (*extensions.ProcessedResponse, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	logger.Debug("Processing embedding response - extracting usage metrics")

	// Use base processor for response processing, but embeddings don't have truncation
	processedResp, err := p.BaseProcessor.ProcessResponse(ctx, resp)
	if err != nil {
		return nil, err
	}

	// Embeddings cannot be truncated, so always set to false
	processedResp.WasTruncated = false
	processedResp.FinishReason = "completed" // Embeddings always complete successfully

	logger.Debug("Embedding response processed successfully")
	return processedResp, nil
}

// UpdateMetrics updates metrics for embedding requests
func (p *EmbeddingProcessor) UpdateMetrics(metadata *metadata.RequestMetadata, req *ProcessedRequest, resp *extensions.ProcessedResponse) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	// Update token usage metrics (embeddings typically report input tokens used)
	if resp.UsedTokens != nil {
		usedFloat := float64(*resp.UsedTokens)
		metadata.RequestMetrics.TokenMetrics.Used = &usedFloat
		// Note: Using debug level since this is called frequently
		// logger.Debugf("Updated embedding metrics - tokens used: %d", *resp.UsedTokens)
	}

	// Embeddings don't have retry scenarios, so no retry metrics to update
	return nil
}
