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

// BedrockMeta20230601Extension provides Bedrock Meta Llama specific processing for 2023-06-01
type BedrockMeta20230601Extension struct {
}

// NewBedrockMeta20230601Extension creates a new Bedrock Meta 2023-06-01 extension
func NewBedrockMeta20230601Extension() *BedrockMeta20230601Extension {
	return &BedrockMeta20230601Extension{}
}

// GetConfiguration returns the configuration for Bedrock Meta 2023-06-01 processing
func (e *BedrockMeta20230601Extension) GetConfiguration() ExtensionConfiguration {
	return ExtensionConfiguration{
		Request: RequestConfig{ // #nosec G101 -- these are JSON field path expressions, not credentials
			MaxTokens:    "max_gen_len",
			SystemPrompt: "prompt", // Prepend to prompt
		},
		Response: ResponseConfig{ // #nosec G101 -- these are JSON field path expressions, not credentials
			UsedTokens:   "generation_token_count",
			FinishReason: "stop_reason",
		},
	}
}

// ParseStreamingResponse parses Bedrock Meta 2023-06-01 streaming response
func (e *BedrockMeta20230601Extension) ParseStreamingResponse(responseBody []byte) (*ProcessedResponse, error) {
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

		// Extract token usage
		e.extractTokenUsage(chunk, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning Bedrock Meta 2023-06-01 streaming response: %w", err)
	}

	return result, nil
}

// parseChunkLine parses a single line and returns the chunk and whether to continue
// Returns (chunk, shouldContinue)
func (e *BedrockMeta20230601Extension) parseChunkLine(line string) (map[string]interface{}, bool) {
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
func (e *BedrockMeta20230601Extension) extractStopReason(chunk map[string]interface{}, result *ProcessedResponse) {
	stopReason, ok := chunk["stop_reason"].(string)
	if !ok {
		return
	}

	result.FinishReason = stopReason
	// Handle documented stop reasons: "stop", "length"
	if stopReason == "length" {
		result.WasTruncated = true
	}
}

// extractTokenUsage extracts token usage from the chunk and updates the result
func (e *BedrockMeta20230601Extension) extractTokenUsage(chunk map[string]interface{}, result *ProcessedResponse) {
	tokens, ok := chunk["generation_token_count"].(float64)
	if !ok {
		return
	}

	usedTokens := int(tokens)
	result.UsedTokens = &usedTokens
}

// ValidateProcessingConfig validates the processing configuration for Bedrock Meta 2023-06-01
func (e *BedrockMeta20230601Extension) ValidateProcessingConfig(config *ProcessingConfig) error {
	if config == nil {
		return fmt.Errorf("processing config cannot be nil")
	}

	// Validate max_tokens configuration
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue <= 0 {
		return fmt.Errorf("OutputTokensBaseValue must be positive, got: %d", *config.OutputTokensBaseValue)
	}

	// Meta Llama specific validation - correct token limit per documentation
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue > 2048 {
		return fmt.Errorf("max_tokens cannot exceed 2048 for Bedrock Meta Llama 2023-06-01")
	}

	// Meta Llama models support copyright protection through prompt prepending
	return nil
}
