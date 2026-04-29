/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package extensions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// AzureOpenAI20240601Extension provides Azure OpenAI API 2024-06-01 specific processing
type AzureOpenAI20240601Extension struct {
}

// NewAzureOpenAI20240601Extension creates a new Azure OpenAI 2024-06-01 extension
func NewAzureOpenAI20240601Extension() *AzureOpenAI20240601Extension {
	return &AzureOpenAI20240601Extension{}
}

// GetConfiguration returns the configuration for Azure OpenAI 2024-06-01 processing
func (e *AzureOpenAI20240601Extension) GetConfiguration() ExtensionConfiguration {
	return ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_tokens",
			SystemPrompt: "messages.0.content", // Prepend to first message content
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.completion_tokens",
			FinishReason: "choices.0.finish_reason",
		},
	}
}

// ParseStreamingResponse parses Azure OpenAI 2024-06-01 streaming response (SSE format)
func (e *AzureOpenAI20240601Extension) ParseStreamingResponse(responseBody []byte) (*ProcessedResponse, error) {
	result := &ProcessedResponse{}
	scanner := bufio.NewScanner(bytes.NewReader(responseBody))

	for scanner.Scan() {
		line := scanner.Text()

		chunk, shouldContinue, err := e.parseSSELine(line)
		if err != nil {
			return nil, err
		}
		if shouldContinue {
			continue
		}
		if chunk == nil {
			break // [DONE] marker encountered
		}

		// Extract finish_reason from choices
		e.extractFinishReason(chunk, result)

		// Extract token usage from final chunk
		e.extractTokenUsage(chunk, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning Azure OpenAI 2024-06-01 streaming response: %w", err)
	}

	return result, nil
}

// parseSSELine parses a single SSE line and returns the chunk, whether to continue, and any error
// Returns (chunk, shouldContinue, error)
// - If shouldContinue is true, the line should be skipped
// - If chunk is nil and shouldContinue is false, it means [DONE] was encountered
func (e *AzureOpenAI20240601Extension) parseSSELine(line string) (map[string]interface{}, bool, error) {
	if !strings.HasPrefix(line, "data: ") {
		return nil, true, nil
	}

	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return nil, false, nil
	}

	var chunk map[string]interface{}
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, true, nil // Skip invalid JSON chunks
	}

	return chunk, false, nil
}

// extractFinishReason extracts finish_reason from the chunk and updates the result
func (e *AzureOpenAI20240601Extension) extractFinishReason(chunk map[string]interface{}, result *ProcessedResponse) {
	choices, ok := chunk["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return
	}

	finishReason, ok := choice["finish_reason"].(string)
	if !ok {
		return
	}

	result.FinishReason = finishReason
	if finishReason == "length" {
		result.WasTruncated = true
	}
}

// extractTokenUsage extracts token usage from the chunk and updates the result
func (e *AzureOpenAI20240601Extension) extractTokenUsage(chunk map[string]interface{}, result *ProcessedResponse) {
	usage, ok := chunk["usage"].(map[string]interface{})
	if !ok {
		return
	}

	tokens, ok := usage["completion_tokens"].(float64)
	if !ok {
		return
	}

	usedTokens := int(tokens)
	result.UsedTokens = &usedTokens

	// Extract reasoning tokens from completion_tokens_details if present
	if details, ok := usage["completion_tokens_details"].(map[string]interface{}); ok {
		if reasoningTokensFloat, ok := details["reasoning_tokens"].(float64); ok && reasoningTokensFloat > 0 {
			reasoningTokens := int(reasoningTokensFloat)
			result.ReasoningTokens = &reasoningTokens
		}
	}
}

// ValidateProcessingConfig validates the processing configuration for Azure OpenAI 2024-06-01
func (e *AzureOpenAI20240601Extension) ValidateProcessingConfig(config *ProcessingConfig) error {
	if config == nil {
		return fmt.Errorf("processing config cannot be nil")
	}

	// Validate max_tokens configuration
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue <= 0 {
		return fmt.Errorf("OutputTokensBaseValue must be positive, got: %d", *config.OutputTokensBaseValue)
	}

	// Azure OpenAI 2024-06-01 specific validation - enhanced extensions, audio speech generation
	// Higher token limits for newer models
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue > 32768 {
		return fmt.Errorf("max_tokens cannot exceed 32768 for Azure OpenAI 2024-06-01")
	}

	return nil
}
