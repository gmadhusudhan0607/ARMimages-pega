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
	requestjson "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/json"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/registry"
)

// ImageProcessor implements specialized processing for image generation endpoints
type ImageProcessor struct {
	*BaseProcessor
}

// NewImageProcessorImpl creates a new image processor with provider-specific extension
func NewImageProcessorImpl(ctx context.Context, model *types.Model, config *extensions.ProcessingConfig) (RequestProcessor, error) {
	if model == nil {
		return nil, fmt.Errorf("model cannot be nil")
	}

	logger := cntx.LoggerFromContext(ctx).Sugar()
	logger.Debugf("Creating image processor for model %s (provider: %s)", model.Name, model.Provider)

	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Try to get extension from registry first
	processorKey := registry.CreateProcessorKey(model)
	reg := registry.GetGlobalRegistry()

	var extension ProviderExtension

	if reg.HasProcessor(processorKey) {
		logger.Debugf("Using registry to create image processor for key: %s", processorKey.String())
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
			logger.Debugf("Creating Azure OpenAI 2024-02-01 extension for Azure images")
			extension = extensions.NewAzureOpenAI20240201Extension()
		default:
			logger.Errorf("No processor registered for key: %s", processorKey.String())
			return nil, fmt.Errorf("no processor registered for key: %s", processorKey.String())
		}
	}

	// Create image-specific configuration
	imageConfig := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategy,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,                             // Image generation doesn't use max_tokens
		CopyrightProtectionEnabled:   config.CopyrightProtectionEnabled, // System prompts can be used for prompt modification
	}

	logger.Debugf("Image processor created successfully for model %s", model.Name)
	return &ImageProcessor{
		BaseProcessor: NewBaseProcessor(extension, imageConfig),
	}, nil
}

// ProcessRequest processes image generation requests
func (p *ImageProcessor) ProcessRequest(ctx context.Context, body []byte) (*ProcessedRequest, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	logger.Debug("Processing image generation request")

	result := &ProcessedRequest{ModifiedBody: body}

	// Handle copyright protection for prompt modification
	if p.config.CopyrightProtectionEnabled {
		logger.Debug("Processing image prompt with copyright protection")
		modifiedBody, hasPrompt, err := p.processImagePrompt(ctx, body)
		if err != nil {
			return nil, fmt.Errorf("failed to process image prompt: %w", err)
		}
		result.ModifiedBody = modifiedBody
		result.HasSystemPrompt = hasPrompt

		if hasPrompt {
			logger.Debug("Image prompt modified successfully")
		}
	}

	return result, nil
}

// ProcessResponse processes image generation responses
func (p *ImageProcessor) ProcessResponse(ctx context.Context, resp *http.Response) (*extensions.ProcessedResponse, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	logger.Debug("Processing image generation response")

	// Use base processor for response processing
	processedResp, err := p.BaseProcessor.ProcessResponse(ctx, resp)
	if err != nil {
		return nil, err
	}

	// Image generation cannot be truncated in the traditional sense
	processedResp.WasTruncated = false
	processedResp.FinishReason = "completed" // Images either generate successfully or fail

	logger.Debug("Image generation response processed successfully")
	return processedResp, nil
}

// UpdateMetrics updates metrics for image generation requests
func (p *ImageProcessor) UpdateMetrics(metadata *metadata.RequestMetadata, req *ProcessedRequest, resp *extensions.ProcessedResponse) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	// Image generation typically doesn't report token usage in the same way
	// But some providers might report usage metrics
	if resp.UsedTokens != nil {
		usedFloat := float64(*resp.UsedTokens)
		metadata.RequestMetrics.TokenMetrics.Used = &usedFloat
		// Note: Using debug level since this is called frequently
		// logger.Debugf("Updated image generation metrics - tokens used: %d", *resp.UsedTokens)
	}

	// Image generation doesn't have retry scenarios based on truncation
	return nil
}

// processImagePrompt handles prompt modification for image generation
func (p *ImageProcessor) processImagePrompt(ctx context.Context, body []byte) ([]byte, bool, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	// For image generation, we work with the "prompt" field directly
	// regardless of what the extension provides (which is for chat completions)
	path := "prompt"

	// Always inject copyright protection message when enabled
	logger.Debug("Injecting copyright protection into image generation request")
	return p.injectImagePrompt(body, path, "Please ensure generated content respects copyright and intellectual property rights.")
}

// injectImagePrompt injects system prompt into image generation request
func (p *ImageProcessor) injectImagePrompt(body []byte, path string, prompt string) ([]byte, bool, error) {
	// For image generation, injection means prepending to the existing prompt
	// Get existing value first
	existing, err := requestjson.GetValueByPath(body, path)
	if err != nil {
		// If path doesn't exist, just set it
		modifiedBody, err := requestjson.SetValueByPath(body, path, prompt)
		if err != nil {
			return body, false, err
		}
		return modifiedBody, true, nil
	}

	// Prepend to existing value
	existingStr, ok := existing.(string)
	if !ok {
		existingStr = ""
	}

	newPrompt := prompt + " " + existingStr
	modifiedBody, err := requestjson.SetValueByPath(body, path, newPrompt)
	if err != nil {
		return body, false, err
	}
	return modifiedBody, true, nil
}
