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

// BedrockAmazon20230601Extension provides Bedrock Amazon Titan specific processing for 2023-06-01
type BedrockAmazon20230601Extension struct {
}

// NewBedrockAmazon20230601Extension creates a new Bedrock Amazon 2023-06-01 extension
func NewBedrockAmazon20230601Extension() *BedrockAmazon20230601Extension {
	return &BedrockAmazon20230601Extension{}
}

// GetConfiguration returns the configuration for Bedrock Amazon 2023-06-01 processing
func (e *BedrockAmazon20230601Extension) GetConfiguration() ExtensionConfiguration {
	return ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "textGenerationConfig.maxTokenCount", // Nested structure handled in processing
			SystemPrompt: "inputText",                          // Prepend to inputText
		},
		Response: ResponseConfig{
			UsedTokens:   "results.0.tokenCount", // Path in results array
			FinishReason: "results.0.completionReason",
		},
	}
}

// ParseStreamingResponse parses Bedrock Amazon 2023-06-01 streaming response
func (e *BedrockAmazon20230601Extension) ParseStreamingResponse(responseBody []byte) (*ProcessedResponse, error) {
	result := &ProcessedResponse{}
	scanner := bufio.NewScanner(bytes.NewReader(responseBody))

	for scanner.Scan() {
		line := scanner.Text()

		chunk, shouldContinue := e.parseChunkLine(line)
		if shouldContinue {
			continue
		}

		// Extract result data from chunk
		e.extractResultData(chunk, result)

		// Handle direct completion reason for streaming chunks
		e.extractDirectCompletionReason(chunk, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning Bedrock Amazon 2023-06-01 streaming response: %w", err)
	}

	return result, nil
}

// parseChunkLine parses a single line and returns the chunk and whether to continue
// Returns (chunk, shouldContinue)
func (e *BedrockAmazon20230601Extension) parseChunkLine(line string) (map[string]interface{}, bool) {
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

// extractResultData extracts token count and completion reason from the results array
func (e *BedrockAmazon20230601Extension) extractResultData(chunk map[string]interface{}, result *ProcessedResponse) {
	results, ok := chunk["results"].([]interface{})
	if !ok || len(results) == 0 {
		return
	}

	result0, ok := results[0].(map[string]interface{})
	if !ok {
		return
	}

	// Extract token count
	if tokens, ok := result0["tokenCount"].(float64); ok {
		usedTokens := int(tokens)
		result.UsedTokens = &usedTokens
	}

	// Extract completion reason
	if reason, ok := result0["completionReason"].(string); ok {
		result.FinishReason = reason
		// Handle documented completion reasons: "FINISHED", "LENGTH", "STOP_CRITERIA_MET"
		if reason == "LENGTH" {
			result.WasTruncated = true
		}
	}
}

// extractDirectCompletionReason handles direct completion reason for streaming chunks
func (e *BedrockAmazon20230601Extension) extractDirectCompletionReason(chunk map[string]interface{}, result *ProcessedResponse) {
	reason, ok := chunk["completionReason"].(string)
	if !ok {
		return
	}

	result.FinishReason = reason
	// Handle documented completion reasons: "FINISHED", "LENGTH", "STOP_CRITERIA_MET"
	if reason == "LENGTH" {
		result.WasTruncated = true
	}
}

// ValidateProcessingConfig validates the processing configuration for Bedrock Amazon 2023-06-01
func (e *BedrockAmazon20230601Extension) ValidateProcessingConfig(config *ProcessingConfig) error {
	if config == nil {
		return fmt.Errorf("processing config cannot be nil")
	}

	// Validate max_tokens configuration
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue <= 0 {
		return fmt.Errorf("OutputTokensBaseValue must be positive, got: %d", *config.OutputTokensBaseValue)
	}

	// Amazon Titan specific validation - model-specific token limits per API documentation
	// Titan Text Lite: 4096, Titan Text Express: 8192, Titan Text Premier: 3072
	// Using most restrictive limit for compatibility
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue > 3072 {
		return fmt.Errorf("max_tokens cannot exceed 3072 for Bedrock Amazon Titan 2023-06-01 (Premier model limit)")
	}

	// Amazon Titan models support copyright protection through input text prepending
	return nil
}
