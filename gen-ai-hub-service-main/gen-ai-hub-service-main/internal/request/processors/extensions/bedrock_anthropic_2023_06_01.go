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

// BedrockAnthropic20230601Extension provides Bedrock Anthropic Claude specific processing for 2023-06-01
type BedrockAnthropic20230601Extension struct {
}

// NewBedrockAnthropic20230601Extension creates a new Bedrock Anthropic 2023-06-01 extension
func NewBedrockAnthropic20230601Extension() *BedrockAnthropic20230601Extension {
	return &BedrockAnthropic20230601Extension{}
}

// GetConfiguration returns the configuration for Bedrock Anthropic 2023-06-01 processing
func (e *BedrockAnthropic20230601Extension) GetConfiguration() ExtensionConfiguration {
	return ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_tokens",
			SystemPrompt: "system", // System prompt at root level
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.output_tokens", // Nested structure - will be handled in parsing
			FinishReason: "stop_reason",
		},
	}
}

// ParseStreamingResponse parses Bedrock Anthropic 2023-06-01 streaming response
func (e *BedrockAnthropic20230601Extension) ParseStreamingResponse(responseBody []byte) (*ProcessedResponse, error) {
	result := &ProcessedResponse{}
	scanner := bufio.NewScanner(bytes.NewReader(responseBody))

	for scanner.Scan() {
		line := scanner.Text()

		chunk, shouldContinue := e.parseChunkLine(line)
		if shouldContinue {
			continue
		}

		// Extract stop reason
		e.extractStopReason(chunk, result)

		// Extract usage information
		e.extractTokenUsage(chunk, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning Bedrock Anthropic 2023-06-01 streaming response: %w", err)
	}

	return result, nil
}

// parseChunkLine parses a single line and returns the chunk and whether to continue
// Returns (chunk, shouldContinue)
func (e *BedrockAnthropic20230601Extension) parseChunkLine(line string) (map[string]interface{}, bool) {
	var chunk map[string]interface{}
	var err error

	// Try to parse as JSON directly
	if strings.HasPrefix(line, "{") {
		err = json.Unmarshal([]byte(line), &chunk)
	} else if strings.HasPrefix(line, "data: ") {
		// SSE format
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" || data == "" {
			return nil, true
		}
		err = json.Unmarshal([]byte(data), &chunk)
	} else {
		return nil, true // Skip non-JSON lines
	}

	if err != nil {
		return nil, true // Skip invalid JSON chunks
	}

	return chunk, false
}

// extractStopReason extracts stop_reason from the chunk and updates the result
func (e *BedrockAnthropic20230601Extension) extractStopReason(chunk map[string]interface{}, result *ProcessedResponse) {
	stopReason, ok := chunk["stop_reason"].(string)
	if !ok {
		return
	}

	result.FinishReason = stopReason
	// Handle documented stop reasons: "end_turn", "max_tokens", "stop_sequence"
	if stopReason == "max_tokens" {
		result.WasTruncated = true
	}
}

// extractTokenUsage extracts token usage from the chunk and updates the result
func (e *BedrockAnthropic20230601Extension) extractTokenUsage(chunk map[string]interface{}, result *ProcessedResponse) {
	usage, ok := chunk["usage"].(map[string]interface{})
	if !ok {
		return
	}

	tokens, ok := usage["output_tokens"].(float64)
	if !ok {
		return
	}

	usedTokens := int(tokens)
	result.UsedTokens = &usedTokens
}

// ValidateProcessingConfig validates the processing configuration for Bedrock Anthropic 2023-06-01
func (e *BedrockAnthropic20230601Extension) ValidateProcessingConfig(config *ProcessingConfig) error {
	if config == nil {
		return fmt.Errorf("processing config cannot be nil")
	}

	// Validate max_tokens configuration
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue <= 0 {
		return fmt.Errorf("OutputTokensBaseValue must be positive, got: %d", *config.OutputTokensBaseValue)
	}

	// Anthropic Claude specific validation - corrected token limits per API documentation
	// Model-specific limits vary, using documented maximum for Claude models
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue > 100000 {
		return fmt.Errorf("max_tokens cannot exceed 100000 for Bedrock Anthropic Claude 2023-06-01")
	}

	// Anthropic models support system prompts natively through copyright protection
	return nil
}
