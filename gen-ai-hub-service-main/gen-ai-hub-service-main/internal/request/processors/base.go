/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package processors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/cache"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	requestjson "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/json"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/strategies"
	"go.uber.org/zap"
)

// BaseProcessor implements common request processing logic
type BaseProcessor struct {
	extension     ProviderExtension
	config        *extensions.ProcessingConfig
	extConfig     extensions.ExtensionConfiguration // Cache configuration
	tokenStrategy strategies.TokenAdjustmentStrategy
}

// NewBaseProcessor creates a new base processor with the given extension and config
func NewBaseProcessor(extension ProviderExtension, config *extensions.ProcessingConfig) *BaseProcessor {
	return &BaseProcessor{
		extension: extension,
		config:    config,
		extConfig: extension.GetConfiguration(), // Cache configuration
	}
}

// NewBaseProcessorWithStrategy creates a new base processor with the given extension, config, and strategy
func NewBaseProcessorWithStrategy(extension ProviderExtension, config *extensions.ProcessingConfig, strategy strategies.TokenAdjustmentStrategy) *BaseProcessor {
	return &BaseProcessor{
		extension:     extension,
		config:        config,
		extConfig:     extension.GetConfiguration(), // Cache configuration
		tokenStrategy: strategy,
	}
}

// ProcessRequest processes the request body according to configuration
func (p *BaseProcessor) ProcessRequest(ctx context.Context, body []byte) (*ProcessedRequest, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	logger.Debugf("Processing request - OutputTokensStrategy: %s, CopyrightProtectionEnabled: %t",
		p.config.OutputTokensStrategy, p.config.CopyrightProtectionEnabled)

	result := &ProcessedRequest{ModifiedBody: body}

	// Always check for max_tokens to track original value, even if not modifying
	modifiedBody, modifiedTokens, err := p.processMaxTokens(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("failed to process max_tokens: %w", err)
	}
	result.ModifiedBody = modifiedBody
	result.ModifiedTokens = modifiedTokens

	// Set OriginalTokens - only when we have a strategy
	if p.tokenStrategy != nil {
		path := p.extConfig.Request.MaxTokens
		if path != "" {
			if value, err := requestjson.GetValueByPath(body, path); err == nil {
				if tokens, err := p.extractIntValue(value); err == nil {
					result.OriginalTokens = tokens
				}
			}
		}
	}
	// When no strategy is configured, OriginalTokens remains nil

	if result.OriginalTokens != nil {
		logger.Debugf("Found original max_tokens in request: %d", *result.OriginalTokens)
	}
	if modifiedTokens != nil {
		logger.Debugf("Modified max_tokens: %d", *modifiedTokens)
	}

	// Handle copyright protection (system prompt injection)
	if p.config.CopyrightProtectionEnabled {
		logger.Debug("Processing copyright protection system prompt")
		modifiedBody, hasPrompt, err := p.processSystemPrompt(ctx, result.ModifiedBody)
		if err != nil {
			return nil, fmt.Errorf("failed to process copyright protection: %w", err)
		}
		result.ModifiedBody = modifiedBody
		result.HasSystemPrompt = hasPrompt

		if hasPrompt {
			logger.Debug("Copyright protection system prompt processed successfully")
		}
	}

	return result, nil
}

// ProcessResponse processes the response to extract metrics and detect truncation
func (p *BaseProcessor) ProcessResponse(ctx context.Context, resp *http.Response) (*extensions.ProcessedResponse, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	_ = resp.Body.Close()

	// Check if this is a streaming response
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		logger.Debug("Processing streaming response")
		// Use provider-specific streaming parser
		result, err := p.extension.ParseStreamingResponse(responseBody)
		if err != nil {
			return nil, fmt.Errorf("failed to parse streaming response: %w", err)
		}
		// Reconstruct response body
		resp.Body = io.NopCloser(bytes.NewReader(responseBody))
		return result, nil
	}

	logger.Debug("Processing non-streaming response")
	// Process non-streaming response
	result, err := p.parseNonStreamingResponse(responseBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse non-streaming response: %w", err)
	}

	if result.WasTruncated {
		logger.Debugf("Response truncation detected - finish reason: %s", result.FinishReason)
	}

	// Reconstruct response body
	resp.Body = io.NopCloser(bytes.NewReader(responseBody))
	return result, nil
}

// UpdateMetrics updates the request metadata with processing results
func (p *BaseProcessor) UpdateMetrics(metadata *metadata.RequestMetadata, req *ProcessedRequest, resp *extensions.ProcessedResponse) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	// Update token metrics
	if req.OriginalTokens != nil {
		originalFloat := float64(*req.OriginalTokens)
		metadata.RequestMetrics.TokenMetrics.Requested = &originalFloat
	}

	if req.ModifiedTokens != nil {
		adjustedFloat := float64(*req.ModifiedTokens)
		metadata.RequestMetrics.TokenMetrics.Adjusted = &adjustedFloat
	}

	if resp.UsedTokens != nil {
		usedFloat := float64(*resp.UsedTokens)
		metadata.RequestMetrics.TokenMetrics.Used = &usedFloat
	}

	if resp.ReasoningTokens != nil {
		reasoningFloat := float64(*resp.ReasoningTokens)
		metadata.RequestMetrics.TokenMetrics.ReasoningTokens = &reasoningFloat
	}

	// NEW: Extract maximum tokens from target model
	if metadata.TargetModel != nil {
		if maxOutputTokensParam, exists := metadata.TargetModel.Parameters["maxOutputTokens"]; exists {
			if maxValue := maxOutputTokensParam.Maximum; maxValue != nil {
				if maxFloat, err := p.extractFloatValue(maxValue); err == nil && maxFloat != nil {
					metadata.RequestMetrics.TokenMetrics.Maximum = maxFloat
				}
			}
		}
	}

	// Update retry metrics if truncated
	if resp.WasTruncated {
		metadata.RequestMetrics.RetryMetrics.ResponseTruncated = true
		reason := "length_limit"
		metadata.RequestMetrics.RetryMetrics.Reason = &reason
	}

	return nil
}

// processMaxTokens handles max_tokens parameter modification
func (p *BaseProcessor) processMaxTokens(ctx context.Context, body []byte) ([]byte, *int, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	tokenPath := p.extConfig.Request.MaxTokens

	// Extract original tokens and check if streaming
	isStreaming := p.isStreamingRequest(body)
	originalTokens := p.extractOriginalTokens(body, tokenPath)

	// Calculate adjusted tokens using strategy
	adjustedTokens := p.calculateAdjustedTokens(ctx, originalTokens)

	// Determine if adjustment should be applied to the request
	shouldApplyAdjustment := p.shouldApplyAdjustment(originalTokens, adjustedTokens, isStreaming, logger)

	// Check if we should store adjusted metrics without modifying request (streaming case)
	if !shouldApplyAdjustment && p.shouldStoreAdjustedMetrics(originalTokens, adjustedTokens, isStreaming) {
		p.storeAdjustedMetricsOnly(ctx, adjustedTokens, logger)
		// Return original body but with adjusted tokens for metrics
		return body, adjustedTokens, nil
	}

	// If we shouldn't apply adjustment and don't need to store metrics, return original
	if !shouldApplyAdjustment {
		return body, originalTokens, nil
	}

	// Apply the adjustment and update metrics
	return p.applyTokenAdjustment(ctx, body, tokenPath, originalTokens, adjustedTokens, logger)
}

// extractOriginalTokens extracts the original max_tokens value from the request body
func (p *BaseProcessor) extractOriginalTokens(body []byte, tokenPath string) *int {
	if tokenPath == "" {
		return nil
	}

	value, err := requestjson.GetValueByPath(body, tokenPath)
	if err != nil {
		return nil
	}

	tokens, err := p.extractIntValue(value)
	if err != nil {
		return nil
	}

	return tokens
}

// calculateAdjustedTokens calculates the adjusted token value using the configured strategy
func (p *BaseProcessor) calculateAdjustedTokens(ctx context.Context, originalTokens *int) *int {
	// If no strategy is configured (disabled), return nil
	if p.tokenStrategy == nil {
		return nil
	}

	modelMaximum := p.extractModelMaximum(ctx)
	configValue := 0
	if p.config.OutputTokensBaseValue != nil {
		configValue = *p.config.OutputTokensBaseValue
	}

	// Special handling for cache-based strategies
	if autoStrategy, ok := p.tokenStrategy.(*strategies.AutoIncreasingStrategy); ok {
		cacheKey := p.createCacheKey(ctx)
		return autoStrategy.CalculateAdjustedValueWithCache(originalTokens, modelMaximum, configValue, cacheKey)
	}

	if percentileStrategy, ok := p.tokenStrategy.(*strategies.PercentileTokenStrategy); ok {
		cacheKey := p.createCacheKey(ctx)
		return percentileStrategy.CalculateAdjustedValueWithCache(originalTokens, modelMaximum, configValue, cacheKey)
	}

	return p.tokenStrategy.CalculateAdjustedValue(originalTokens, modelMaximum, configValue)
}

// shouldApplyAdjustment determines if token adjustment should be applied based on configuration
func (p *BaseProcessor) shouldApplyAdjustment(originalTokens, adjustedTokens *int, isStreaming bool, logger *zap.SugaredLogger) bool {
	// For streaming requests: NEVER modify max_tokens, regardless of strategy
	// All strategies (except DISABLED) behave as MONITORING_ONLY for streams
	// This means we still collect metrics (original, adjusted) but don't modify the request
	if isStreaming {
		if originalTokens == nil {
			logger.Debug("Streaming request: Not adding max_tokens (treating all strategies as MONITORING_ONLY for streams)")
		} else {
			logger.Debug("Streaming request: Not modifying max_tokens (treating all strategies as MONITORING_ONLY for streams)")
		}
		return false
	}

	// Non-streaming request logic below
	// If original tokens don't exist, always insert adjusted value
	if originalTokens == nil {
		return true
	}

	// max_tokens exists in request - check forcing rules
	if p.config.OutputTokensAdjustmentForced {
		return p.shouldForceAdjustment(originalTokens, adjustedTokens, logger)
	}

	// When forced=false, don't adjust
	return false
}

// shouldStoreAdjustedMetrics determines if we should store adjusted metrics even without modifying request
func (p *BaseProcessor) shouldStoreAdjustedMetrics(originalTokens, adjustedTokens *int, isStreaming bool) bool {
	// For streaming requests, always store adjusted metrics for monitoring (if we have adjusted value)
	if isStreaming && adjustedTokens != nil {
		return true
	}
	return false
}

// shouldForceAdjustment checks if forced adjustment should be applied
func (p *BaseProcessor) shouldForceAdjustment(originalTokens, adjustedTokens *int, logger *zap.SugaredLogger) bool {
	if adjustedTokens == nil {
		return false
	}

	// When forced=true, only apply if suggested < original
	if *adjustedTokens < *originalTokens {
		logger.Debugf("Forcing adjustment: original=%d, suggested=%d (suggested < original)", *originalTokens, *adjustedTokens)
		return true
	}

	logger.Debugf("Skipping forced adjustment: original=%d, suggested=%d (suggested >= original)", *originalTokens, *adjustedTokens)
	return false
}

// applyTokenAdjustment applies the token adjustment to the request body and updates metrics
func (p *BaseProcessor) applyTokenAdjustment(ctx context.Context, body []byte, tokenPath string, originalTokens, adjustedTokens *int, logger *zap.SugaredLogger) ([]byte, *int, error) {
	if tokenPath == "" || adjustedTokens == nil {
		return body, originalTokens, nil
	}

	modifiedBody, err := requestjson.SetValueByPath(body, tokenPath, *adjustedTokens)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set adjusted max_tokens: %w", err)
	} else {
		originalValue := "nil"
		if originalTokens != nil {
			originalValue = fmt.Sprintf("%d", *originalTokens)
		}
		logger.Debugf("Applied %s strategy - original: %s, adjusted: %d",
			p.tokenStrategy.GetStrategyName(), originalValue, *adjustedTokens)
	}

	p.updateAdjustedMetricIfModified(ctx, originalTokens, adjustedTokens)

	return modifiedBody, adjustedTokens, nil
}

// storeAdjustedMetricsOnly stores adjusted metrics without modifying the request (for streaming)
func (p *BaseProcessor) storeAdjustedMetricsOnly(ctx context.Context, adjustedTokens *int, logger *zap.SugaredLogger) {
	if adjustedTokens == nil {
		return
	}

	logger.Debugf("Storing adjusted metrics for streaming request (no modification): adjusted=%d", *adjustedTokens)

	// For streaming, we don't update the adjusted current metric since we're not modifying the request
	// We only store the adjusted value in the result for metrics collection
}

// updateAdjustedMetricIfModified updates the adjusted current metric only if max_tokens was actually modified or inserted
func (p *BaseProcessor) updateAdjustedMetricIfModified(ctx context.Context, originalTokens, adjustedTokens *int) {
	// Check if max_tokens was modified or inserted
	wasModified := (originalTokens == nil && adjustedTokens != nil) || // Insertion
		(originalTokens != nil && *originalTokens != *adjustedTokens) // Modification

	if !wasModified {
		return
	}

	requestMetadata, err := metadata.GetRequestMetadataFromContext(ctx)
	if err != nil {
		return
	}

	labels := p.createMetricsLabels(requestMetadata)
	metrics.UpdateAdjustedCurrentMetric(float64(*adjustedTokens), labels)
}

// extractModelMaximum extracts the model maximum from RequestMetadata in context
func (p *BaseProcessor) extractModelMaximum(ctx context.Context) *float64 {
	// Extract from RequestMetadata in context
	if requestMetadata, err := metadata.GetRequestMetadataFromContext(ctx); err == nil {
		if requestMetadata.TargetModel != nil {
			if maxParam, exists := requestMetadata.TargetModel.Parameters["maxOutputTokens"]; exists {
				if maximum := maxParam.Maximum; maximum != nil {
					if maxFloat, err := p.extractFloatValue(maximum); err == nil {
						return maxFloat
					}
				}
			}
		}
	}
	return nil
}

// processSystemPrompt handles copyright protection system prompt injection
func (p *BaseProcessor) processSystemPrompt(ctx context.Context, body []byte) ([]byte, bool, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	path := p.extConfig.Request.SystemPrompt

	if path == "" {
		return body, false, nil
	}

	// Always inject DefaultCopyrightMessage when copyright protection is enabled
	logger.Debugf("Injecting copyright protection system prompt at path: %s", path)
	return p.injectSystemPrompt(body, path, config.DefaultCopyrightMessage)
}

// injectSystemPrompt injects a system prompt
func (p *BaseProcessor) injectSystemPrompt(body []byte, path string, prompt string) ([]byte, bool, error) {
	// Special handling for chat completions messages array
	if path == "messages" {
		return p.injectSystemMessageToArray(body, prompt)
	}

	// Basic implementation for other paths
	modifiedBody, err := requestjson.SetValueByPath(body, path, prompt)
	if err != nil {
		return body, false, err
	}
	return modifiedBody, true, nil
}

// injectSystemMessageToArray adds a system message to the messages array
func (p *BaseProcessor) injectSystemMessageToArray(body []byte, prompt string) ([]byte, bool, error) {
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		return body, false, fmt.Errorf("failed to parse request JSON: %w", err)
	}

	// Get messages array
	messagesInterface, exists := requestData["messages"]
	if !exists {
		return body, false, fmt.Errorf("messages array not found in request")
	}

	messages, ok := messagesInterface.([]interface{})
	if !ok {
		return body, false, fmt.Errorf("messages is not an array")
	}

	// Create system message
	systemMessage := map[string]interface{}{
		"role":    "system",
		"content": prompt,
	}

	// Add system message to the end of the messages array
	messages = append(messages, systemMessage)
	requestData["messages"] = messages

	// Marshal back to JSON
	modifiedBody, err := json.Marshal(requestData)
	if err != nil {
		return body, false, fmt.Errorf("failed to marshal modified request: %w", err)
	}

	return modifiedBody, true, nil
}

// parseNonStreamingResponse parses a regular JSON response
func (p *BaseProcessor) parseNonStreamingResponse(responseBody []byte) (*extensions.ProcessedResponse, error) {
	result := &extensions.ProcessedResponse{}

	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return result, nil // Not JSON, return empty result
	}

	// Extract used tokens
	usedTokensPath := p.extConfig.Response.UsedTokens
	if usedTokensPath != "" {
		if value, err := requestjson.GetValueByPath(responseBody, usedTokensPath); err == nil {
			if tokens, err := p.extractIntValue(value); err == nil && tokens != nil {
				result.UsedTokens = tokens
			}
		}
	}

	// Extract reasoning tokens from completion_tokens_details.reasoning_tokens
	if value, err := requestjson.GetValueByPath(responseBody, "usage.completion_tokens_details.reasoning_tokens"); err == nil {
		if tokens, err := p.extractIntValue(value); err == nil && tokens != nil && *tokens > 0 {
			result.ReasoningTokens = tokens
		}
	}

	// Extract finish reason and check for truncation
	finishReasonPath := p.extConfig.Response.FinishReason
	if finishReasonPath != "" {
		if value, err := requestjson.GetValueByPath(responseBody, finishReasonPath); err == nil {
			if finishReason, ok := value.(string); ok {
				result.FinishReason = finishReason
				if finishReason == "length" {
					result.WasTruncated = true
				}
			}
		}
	}

	return result, nil
}

// extractIntValue safely extracts an integer value from an interface{}
func (p *BaseProcessor) extractIntValue(value interface{}) (*int, error) {
	switch v := value.(type) {
	case int:
		return &v, nil
	case float64:
		intVal := int(v)
		return &intVal, nil
	case json.Number:
		if intVal, err := strconv.Atoi(string(v)); err == nil {
			return &intVal, nil
		} else {
			return nil, fmt.Errorf("failed to convert json.Number to int: %w", err)
		}
	case string:
		if intVal, err := strconv.Atoi(v); err == nil {
			return &intVal, nil
		} else {
			return nil, fmt.Errorf("failed to convert string '%s' to int: %w", v, err)
		}
	default:
		return nil, fmt.Errorf("unsupported type for integer extraction: %T", value)
	}
}

// extractFloatValue safely extracts a float64 value from an interface{}
func (p *BaseProcessor) extractFloatValue(value interface{}) (*float64, error) {
	switch v := value.(type) {
	case int:
		floatVal := float64(v)
		return &floatVal, nil
	case float64:
		return &v, nil
	case json.Number:
		if floatVal, err := v.Float64(); err == nil {
			return &floatVal, nil
		} else {
			return nil, fmt.Errorf("failed to convert json.Number to float64: %w", err)
		}
	case string:
		if floatVal, err := strconv.ParseFloat(v, 64); err == nil {
			return &floatVal, nil
		} else {
			return nil, fmt.Errorf("failed to convert string '%s' to float64: %w", v, err)
		}
	default:
		return nil, fmt.Errorf("unsupported type for float extraction: %T", value)
	}
}

// createCacheKey creates a cache key from the request metadata
func (p *BaseProcessor) createCacheKey(ctx context.Context) cache.CacheKey {
	key := cache.CacheKey{}

	if requestMetadata, err := metadata.GetRequestMetadataFromContext(ctx); err == nil {
		// Include IsolationID to ensure cache is unique per isolation
		key.IsolationID = requestMetadata.GetIsolationID()
		key.Infrastructure = requestMetadata.GetTargetModelInfrastructure()
		key.Creator = requestMetadata.GetTargetModelCreator()
		key.ModelName = requestMetadata.GetTargetModelName()
		key.ModelVersion = requestMetadata.GetTargetModelVersion()

		// Extract provider from target model if available
		if targetModel := requestMetadata.GetTargetModel(); targetModel != nil {
			key.Provider = string(targetModel.Provider)
		}
	}

	return key
}

// createMetricsLabels creates metrics labels from request metadata
func (p *BaseProcessor) createMetricsLabels(requestMetadata *metadata.RequestMetadata) map[string]string {
	labels := make(map[string]string)

	if requestMetadata != nil {
		labels["isolationID"] = requestMetadata.GetIsolationID()
		labels["infrastructure"] = requestMetadata.GetTargetModelInfrastructure()
		labels["provider"] = ""
		labels["creator"] = requestMetadata.GetTargetModelCreator()
		labels["originalModelName"] = requestMetadata.GetOriginalModelName()
		labels["targetModelName"] = requestMetadata.GetTargetModelName()
		labels["targetModelVersion"] = requestMetadata.GetTargetModelVersion()
		labels["targetModelID"] = requestMetadata.GetTargetModelID()

		// Extract provider from target model if available
		if targetModel := requestMetadata.GetTargetModel(); targetModel != nil {
			labels["provider"] = string(targetModel.Provider)
		}
	}

	return labels
}

// UpdateCache updates the cache after a successful response for auto-adjustment and percentile strategies
func (p *BaseProcessor) UpdateCache(ctx context.Context, usedTokens int, configValue int) {
	cacheKey := p.createCacheKey(ctx)

	// Update cache for AutoIncreasingStrategy
	if autoStrategy, ok := p.tokenStrategy.(*strategies.AutoIncreasingStrategy); ok {
		// Update the cache first
		autoStrategy.UpdateCache(cacheKey, usedTokens, configValue)

		// CRITICAL: For AUTO_INCREASING strategy, we MUST update the adjusted current metric after cache update
		// because the cached value may change after processing usage data (e.g., max of used tokens and config value)
		newAdjustedValue := autoStrategy.CalculateAdjustedValueWithCache(nil, nil, configValue, cacheKey)
		if newAdjustedValue != nil {
			if requestMetadata, err := metadata.GetRequestMetadataFromContext(ctx); err == nil {
				labels := p.createMetricsLabels(requestMetadata)
				metrics.UpdateAdjustedCurrentMetric(float64(*newAdjustedValue), labels)
			}
		}
	} else if percentileStrategy, ok := p.tokenStrategy.(*strategies.PercentileTokenStrategy); ok {
		// Update cache for PercentileTokenStrategy
		percentileStrategy.UpdateCache(cacheKey, usedTokens, configValue)

		// CRITICAL: For percentile strategies, we MUST update the adjusted current metric after cache update
		// because the percentile value may change after adding new usage samples
		newPercentileValue := percentileStrategy.CalculateAdjustedValueWithCache(nil, nil, configValue, cacheKey)
		if newPercentileValue != nil {
			if requestMetadata, err := metadata.GetRequestMetadataFromContext(ctx); err == nil {
				labels := p.createMetricsLabels(requestMetadata)
				metrics.UpdateAdjustedCurrentMetric(float64(*newPercentileValue), labels)
			}
		}
	}
}

// isStreamingRequest checks if the request is for streaming by looking for "stream": true in the request body
func (p *BaseProcessor) isStreamingRequest(body []byte) bool {
	// Try to extract stream parameter from the request body
	if value, err := requestjson.GetValueByPath(body, "stream"); err == nil {
		if streamBool, ok := value.(bool); ok {
			return streamBool
		}
		// Handle string representation of boolean
		if streamStr, ok := value.(string); ok {
			return strings.ToLower(streamStr) == "true"
		}
	}
	return false
}
