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

// VertexGoogle20240101Extension provides Vertex AI Google Gemini specific processing for 2024-01-01
type VertexGoogle20240101Extension struct {
}

// NewVertexGoogle20240101Extension creates a new Vertex Google 2024-01-01 extension
func NewVertexGoogle20240101Extension() *VertexGoogle20240101Extension {
	return &VertexGoogle20240101Extension{}
}

// GetConfiguration returns the configuration for Vertex Google 2024-01-01 processing
func (e *VertexGoogle20240101Extension) GetConfiguration() ExtensionConfiguration {
	return ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "generationConfig.maxOutputTokens",
			SystemPrompt: "systemInstruction.parts.0.text", // Google system instruction
		},
		Response: ResponseConfig{ // #nosec G101 -- these are JSON field path expressions, not credentials
			UsedTokens:   "usageMetadata.candidatesTokenCount",
			FinishReason: "candidates.0.finishReason",
		},
	}
}

// ParseStreamingResponse parses Vertex Google 2024-01-01 streaming response
func (e *VertexGoogle20240101Extension) ParseStreamingResponse(responseBody []byte) (*ProcessedResponse, error) {
	result := &ProcessedResponse{}
	scanner := bufio.NewScanner(bytes.NewReader(responseBody))

	for scanner.Scan() {
		line := scanner.Text()

		chunk, shouldContinue := e.parseChunkLine(line)
		if shouldContinue {
			continue
		}

		// Parse Google Gemini streaming format - extract finish reason from candidates
		e.extractFinishReason(chunk, result)

		// Extract usage information from final chunk
		e.extractTokenUsage(chunk, result)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning Vertex Google 2024-01-01 streaming response: %w", err)
	}

	return result, nil
}

// parseChunkLine parses a single line and returns the chunk and whether to continue
// Returns (chunk, shouldContinue)
func (e *VertexGoogle20240101Extension) parseChunkLine(line string) (map[string]interface{}, bool) {
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

// extractFinishReason extracts finish reason from the candidates array and updates the result
func (e *VertexGoogle20240101Extension) extractFinishReason(chunk map[string]interface{}, result *ProcessedResponse) {
	candidates, ok := chunk["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return
	}

	candidate, ok := candidates[0].(map[string]interface{})
	if !ok {
		return
	}

	finishReason, ok := candidate["finishReason"].(string)
	if !ok {
		return
	}

	result.FinishReason = finishReason
	// Handle Google Gemini finish reasons: "FINISH_REASON_MAX_TOKENS", "FINISH_REASON_STOP", etc.
	if finishReason == "FINISH_REASON_MAX_TOKENS" {
		result.WasTruncated = true
	}
}

// extractTokenUsage extracts token usage from the usageMetadata and updates the result
func (e *VertexGoogle20240101Extension) extractTokenUsage(chunk map[string]interface{}, result *ProcessedResponse) {
	usageMetadata, ok := chunk["usageMetadata"].(map[string]interface{})
	if !ok {
		return
	}

	tokens, ok := usageMetadata["candidatesTokenCount"].(float64)
	if !ok {
		return
	}

	usedTokens := int(tokens)
	result.UsedTokens = &usedTokens
}

// ValidateProcessingConfig validates the processing configuration for Vertex Google 2024-01-01
func (e *VertexGoogle20240101Extension) ValidateProcessingConfig(config *ProcessingConfig) error {
	if config == nil {
		return fmt.Errorf("processing config cannot be nil")
	}

	// Validate max_tokens configuration
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue <= 0 {
		return fmt.Errorf("OutputTokensBaseValue must be positive, got: %d", *config.OutputTokensBaseValue)
	}

	// Google Gemini on Vertex AI specific validation - based on Gemini 2.5 Flash specs
	if config.OutputTokensBaseValue != nil && *config.OutputTokensBaseValue > 65535 {
		return fmt.Errorf("max_tokens cannot exceed 65535 for Vertex Google Gemini 2024-01-01")
	}

	// Google Gemini models support system instructions natively through copyright protection
	return nil
}
