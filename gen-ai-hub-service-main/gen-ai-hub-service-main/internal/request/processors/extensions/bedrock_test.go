/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package extensions

import (
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

func TestBedrockAnthropic_GetConfiguration(t *testing.T) {
	ext := NewBedrockAnthropic20230601Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_tokens",
			SystemPrompt: "system",
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.output_tokens",
			FinishReason: "stop_reason",
		},
	}

	if result.Request.MaxTokens != expected.Request.MaxTokens {
		t.Errorf("Request.MaxTokens: got %s, want %s", result.Request.MaxTokens, expected.Request.MaxTokens)
	}

	if result.Request.SystemPrompt != expected.Request.SystemPrompt {
		t.Errorf("Request.SystemPrompt: got %s, want %s", result.Request.SystemPrompt, expected.Request.SystemPrompt)
	}

	if result.Response.UsedTokens != expected.Response.UsedTokens {
		t.Errorf("Response.UsedTokens: got %s, want %s", result.Response.UsedTokens, expected.Response.UsedTokens)
	}

	if result.Response.FinishReason != expected.Response.FinishReason {
		t.Errorf("Response.FinishReason: got %s, want %s", result.Response.FinishReason, expected.Response.FinishReason)
	}
}

func TestBedrockAmazon_GetConfiguration(t *testing.T) {
	ext := NewBedrockAmazon20230601Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "textGenerationConfig.maxTokenCount",
			SystemPrompt: "inputText",
		},
		Response: ResponseConfig{
			UsedTokens:   "results.0.tokenCount",
			FinishReason: "results.0.completionReason",
		},
	}

	if result.Request.MaxTokens != expected.Request.MaxTokens {
		t.Errorf("Request.MaxTokens: got %s, want %s", result.Request.MaxTokens, expected.Request.MaxTokens)
	}

	if result.Request.SystemPrompt != expected.Request.SystemPrompt {
		t.Errorf("Request.SystemPrompt: got %s, want %s", result.Request.SystemPrompt, expected.Request.SystemPrompt)
	}

	if result.Response.UsedTokens != expected.Response.UsedTokens {
		t.Errorf("Response.UsedTokens: got %s, want %s", result.Response.UsedTokens, expected.Response.UsedTokens)
	}

	if result.Response.FinishReason != expected.Response.FinishReason {
		t.Errorf("Response.FinishReason: got %s, want %s", result.Response.FinishReason, expected.Response.FinishReason)
	}
}

func TestBedrockMeta_GetConfiguration(t *testing.T) {
	ext := NewBedrockMeta20230601Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_gen_len",
			SystemPrompt: "prompt",
		},
		Response: ResponseConfig{
			UsedTokens:   "generation_token_count",
			FinishReason: "stop_reason",
		},
	}

	if result.Request.MaxTokens != expected.Request.MaxTokens {
		t.Errorf("Request.MaxTokens: got %s, want %s", result.Request.MaxTokens, expected.Request.MaxTokens)
	}

	if result.Request.SystemPrompt != expected.Request.SystemPrompt {
		t.Errorf("Request.SystemPrompt: got %s, want %s", result.Request.SystemPrompt, expected.Request.SystemPrompt)
	}

	if result.Response.UsedTokens != expected.Response.UsedTokens {
		t.Errorf("Response.UsedTokens: got %s, want %s", result.Response.UsedTokens, expected.Response.UsedTokens)
	}

	if result.Response.FinishReason != expected.Response.FinishReason {
		t.Errorf("Response.FinishReason: got %s, want %s", result.Response.FinishReason, expected.Response.FinishReason)
	}
}

func TestBedrockAnthropic_ParseStreamingResponse(t *testing.T) {
	ext := NewBedrockAnthropic20230601Extension()
	responseBody := `{"stop_reason": "max_tokens", "usage": {"output_tokens": 150}}
{"type": "content_block_delta", "delta": {"text": "Hello"}}`

	result, err := ext.ParseStreamingResponse([]byte(responseBody))

	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v", err)
		return
	}

	expected := &ProcessedResponse{
		UsedTokens:   intPtr(150),
		WasTruncated: true,
		FinishReason: "max_tokens",
	}

	if result.FinishReason != expected.FinishReason {
		t.Errorf("FinishReason: got %s, want %s", result.FinishReason, expected.FinishReason)
	}

	if result.WasTruncated != expected.WasTruncated {
		t.Errorf("WasTruncated: got %v, want %v", result.WasTruncated, expected.WasTruncated)
	}

	if (result.UsedTokens == nil) != (expected.UsedTokens == nil) {
		t.Errorf("UsedTokens nil mismatch: got %v, want %v", result.UsedTokens == nil, expected.UsedTokens == nil)
	} else if result.UsedTokens != nil && *result.UsedTokens != *expected.UsedTokens {
		t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *expected.UsedTokens)
	}
}

func TestBedrockAmazon_ParseStreamingResponse(t *testing.T) {
	ext := NewBedrockAmazon20230601Extension()
	responseBody := `{"results": [{"tokenCount": 200, "completionReason": "LENGTH"}]}
{"completionReason": "LENGTH"}`

	result, err := ext.ParseStreamingResponse([]byte(responseBody))

	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v", err)
		return
	}

	expected := &ProcessedResponse{
		UsedTokens:   intPtr(200),
		WasTruncated: true,
		FinishReason: "LENGTH",
	}

	if result.FinishReason != expected.FinishReason {
		t.Errorf("FinishReason: got %s, want %s", result.FinishReason, expected.FinishReason)
	}

	if result.WasTruncated != expected.WasTruncated {
		t.Errorf("WasTruncated: got %v, want %v", result.WasTruncated, expected.WasTruncated)
	}

	if (result.UsedTokens == nil) != (expected.UsedTokens == nil) {
		t.Errorf("UsedTokens nil mismatch: got %v, want %v", result.UsedTokens == nil, expected.UsedTokens == nil)
	} else if result.UsedTokens != nil && *result.UsedTokens != *expected.UsedTokens {
		t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *expected.UsedTokens)
	}
}

func TestBedrockMeta_ParseStreamingResponse(t *testing.T) {
	ext := NewBedrockMeta20230601Extension()
	responseBody := `{"stop_reason": "length", "generation_token_count": 100}
{"generation": "Hello"}`

	result, err := ext.ParseStreamingResponse([]byte(responseBody))

	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v", err)
		return
	}

	expected := &ProcessedResponse{
		UsedTokens:   intPtr(100),
		WasTruncated: true,
		FinishReason: "length",
	}

	if result.FinishReason != expected.FinishReason {
		t.Errorf("FinishReason: got %s, want %s", result.FinishReason, expected.FinishReason)
	}

	if result.WasTruncated != expected.WasTruncated {
		t.Errorf("WasTruncated: got %v, want %v", result.WasTruncated, expected.WasTruncated)
	}

	if (result.UsedTokens == nil) != (expected.UsedTokens == nil) {
		t.Errorf("UsedTokens nil mismatch: got %v, want %v", result.UsedTokens == nil, expected.UsedTokens == nil)
	} else if result.UsedTokens != nil && *result.UsedTokens != *expected.UsedTokens {
		t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *expected.UsedTokens)
	}
}

func TestBedrockAnthropic_ValidateProcessingConfig(t *testing.T) {
	ext := NewBedrockAnthropic20230601Extension()

	// Valid config
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(100),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err := ext.ValidateProcessingConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
	}

	// Invalid config - negative max tokens
	invalidConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(-10),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(invalidConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "OutputTokensBaseValue must be positive, got: -10" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "OutputTokensBaseValue must be positive, got: -10")
	}
}

func TestBedrockAmazon_ValidateProcessingConfig(t *testing.T) {
	ext := NewBedrockAmazon20230601Extension()

	// Valid config with copyright protection
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   true,
	}
	err := ext.ValidateProcessingConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
	}

	// Invalid config - too high max tokens
	invalidConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(10000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(invalidConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 3072 for Bedrock Amazon Titan 2023-06-01 (Premier model limit)" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 3072 for Bedrock Amazon Titan 2023-06-01 (Premier model limit)")
	}
}

func TestBedrockMeta_ValidateProcessingConfig(t *testing.T) {
	ext := NewBedrockMeta20230601Extension()

	// Valid config
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(2000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err := ext.ValidateProcessingConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
	}

	// Invalid config - too high max tokens
	invalidConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(5000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(invalidConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 2048 for Bedrock Meta Llama 2023-06-01" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 2048 for Bedrock Meta Llama 2023-06-01")
	}
}

// Enhanced edge case tests for Bedrock extensions
func TestBedrockAnthropic_ParseStreamingResponse_EdgeCases(t *testing.T) {
	ext := NewBedrockAnthropic20230601Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name:         "empty response body",
			responseBody: ``,
			expected:     &ProcessedResponse{},
			expectError:  false,
		},
		{
			name: "response with only invalid JSON lines",
			responseBody: `invalid json line 1
invalid json line 2
not json at all`,
			expected:    &ProcessedResponse{},
			expectError: false,
		},
		{
			name: "response with mixed valid and invalid JSON",
			responseBody: `{"stop_reason": "end_turn", "usage": {"output_tokens": 75}}
invalid json line
{"type": "content_block_delta", "delta": {"text": "test"}}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(75),
				WasTruncated: false,
				FinishReason: "end_turn",
			},
			expectError: false,
		},
		{
			name: "response with missing fields",
			responseBody: `{"type": "message_start"}
{"incomplete": "data"}
{"usage": {"output_tokens": 100}}`,
			expected: &ProcessedResponse{
				UsedTokens: intPtr(100),
			},
			expectError: false,
		},
		{
			name: "response with nested JSON structures",
			responseBody: `{"stop_reason": "max_tokens", "usage": {"output_tokens": 200, "input_tokens": 50}}
{"type": "content_block_delta", "delta": {"text": "Hello", "nested": {"field": "value"}}}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(200),
				WasTruncated: true,
				FinishReason: "max_tokens",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ext.ParseStreamingResponse([]byte(tt.responseBody))

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseStreamingResponse() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseStreamingResponse() unexpected error = %v", err)
				return
			}

			if result.FinishReason != tt.expected.FinishReason {
				t.Errorf("FinishReason: got %s, want %s", result.FinishReason, tt.expected.FinishReason)
			}

			if result.WasTruncated != tt.expected.WasTruncated {
				t.Errorf("WasTruncated: got %v, want %v", result.WasTruncated, tt.expected.WasTruncated)
			}

			if (result.UsedTokens == nil) != (tt.expected.UsedTokens == nil) {
				t.Errorf("UsedTokens nil mismatch: got %v, want %v", result.UsedTokens == nil, tt.expected.UsedTokens == nil)
			} else if result.UsedTokens != nil && tt.expected.UsedTokens != nil && *result.UsedTokens != *tt.expected.UsedTokens {
				t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *tt.expected.UsedTokens)
			}
		})
	}
}

func TestBedrockAmazon_ParseStreamingResponse_EdgeCases(t *testing.T) {
	ext := NewBedrockAmazon20230601Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name:         "response with zero token count",
			responseBody: `{"results": [{"tokenCount": 0, "completionReason": "FINISH"}]}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(0),
				WasTruncated: false,
				FinishReason: "FINISH",
			},
			expectError: false,
		},
		{
			name:         "response with multiple results (uses first)",
			responseBody: `{"results": [{"tokenCount": 100, "completionReason": "LENGTH"}, {"tokenCount": 200, "completionReason": "FINISH"}]}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(100),
				WasTruncated: true,
				FinishReason: "LENGTH",
			},
			expectError: false,
		},
		{
			name:         "response with empty results array",
			responseBody: `{"results": []}`,
			expected:     &ProcessedResponse{},
			expectError:  false,
		},
		{
			name: "response with non-numeric token count",
			responseBody: `{"results": [{"tokenCount": "invalid", "completionReason": "FINISH"}]}
{"results": [{"tokenCount": 150, "completionReason": "LENGTH"}]}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(150),
				WasTruncated: true,
				FinishReason: "LENGTH",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ext.ParseStreamingResponse([]byte(tt.responseBody))

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseStreamingResponse() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseStreamingResponse() unexpected error = %v", err)
				return
			}

			if result.FinishReason != tt.expected.FinishReason {
				t.Errorf("FinishReason: got %s, want %s", result.FinishReason, tt.expected.FinishReason)
			}

			if result.WasTruncated != tt.expected.WasTruncated {
				t.Errorf("WasTruncated: got %v, want %v", result.WasTruncated, tt.expected.WasTruncated)
			}

			if (result.UsedTokens == nil) != (tt.expected.UsedTokens == nil) {
				t.Errorf("UsedTokens nil mismatch: got %v, want %v", result.UsedTokens == nil, tt.expected.UsedTokens == nil)
			} else if result.UsedTokens != nil && tt.expected.UsedTokens != nil && *result.UsedTokens != *tt.expected.UsedTokens {
				t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *tt.expected.UsedTokens)
			}
		})
	}
}

func TestBedrockMeta_ParseStreamingResponse_EdgeCases(t *testing.T) {
	ext := NewBedrockMeta20230601Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name: "response with stop_reason but no token count",
			responseBody: `{"stop_reason": "end_of_turn"}
{"generation": "Hello world"}`,
			expected: &ProcessedResponse{
				FinishReason: "end_of_turn",
				WasTruncated: false,
			},
			expectError: false,
		},
		{
			name: "response with token count but no stop_reason",
			responseBody: `{"generation_token_count": 250}
{"generation": "Hello world"}`,
			expected: &ProcessedResponse{
				UsedTokens: intPtr(250),
			},
			expectError: false,
		},
		{
			name: "response with non-numeric token count",
			responseBody: `{"generation_token_count": "not_a_number", "stop_reason": "end_of_turn"}
{"generation_token_count": 150, "stop_reason": "end_of_turn"}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(150),
				WasTruncated: false,
				FinishReason: "end_of_turn",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ext.ParseStreamingResponse([]byte(tt.responseBody))

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseStreamingResponse() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseStreamingResponse() unexpected error = %v", err)
				return
			}

			if result.FinishReason != tt.expected.FinishReason {
				t.Errorf("FinishReason: got %s, want %s", result.FinishReason, tt.expected.FinishReason)
			}

			if result.WasTruncated != tt.expected.WasTruncated {
				t.Errorf("WasTruncated: got %v, want %v", result.WasTruncated, tt.expected.WasTruncated)
			}

			if (result.UsedTokens == nil) != (tt.expected.UsedTokens == nil) {
				t.Errorf("UsedTokens nil mismatch: got %v, want %v", result.UsedTokens == nil, tt.expected.UsedTokens == nil)
			} else if result.UsedTokens != nil && tt.expected.UsedTokens != nil && *result.UsedTokens != *tt.expected.UsedTokens {
				t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *tt.expected.UsedTokens)
			}
		})
	}
}

// Test nil config scenarios for all Bedrock extensions
func TestBedrockExtensions_ValidateNilConfig(t *testing.T) {
	extensions := []struct {
		name string
		ext  interface {
			ValidateProcessingConfig(config *ProcessingConfig) error
		}
	}{
		{"Anthropic", NewBedrockAnthropic20230601Extension()},
		{"Amazon", NewBedrockAmazon20230601Extension()},
		{"Meta", NewBedrockMeta20230601Extension()},
	}

	for _, extension := range extensions {
		t.Run(extension.name, func(t *testing.T) {
			err := extension.ext.ValidateProcessingConfig(nil)
			if err == nil {
				t.Errorf("ValidateProcessingConfig() expected error for nil config but got none")
			} else if err.Error() != "processing config cannot be nil" {
				t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "processing config cannot be nil")
			}
		})
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}
