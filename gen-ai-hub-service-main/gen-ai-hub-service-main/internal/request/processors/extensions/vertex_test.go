/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package extensions

import (
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

func TestVertexGoogle_GetConfiguration(t *testing.T) {
	ext := NewVertexGoogle20240101Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "generationConfig.maxOutputTokens",
			SystemPrompt: "systemInstruction.parts.0.text",
		},
		Response: ResponseConfig{
			UsedTokens:   "usageMetadata.candidatesTokenCount",
			FinishReason: "candidates.0.finishReason",
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

func TestVertexAnthropic_GetConfiguration(t *testing.T) {
	ext := NewVertexAnthropic20240101Extension()
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

func TestVertexGoogle_ParseStreamingResponse(t *testing.T) {
	ext := NewVertexGoogle20240101Extension()
	responseBody := `{"candidates": [{"finishReason": "FINISH_REASON_MAX_TOKENS"}], "usageMetadata": {"candidatesTokenCount": 150}}
{"candidates": [{"content": {"parts": [{"text": "Hello"}]}}]}`

	result, err := ext.ParseStreamingResponse([]byte(responseBody))

	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v", err)
		return
	}

	expected := &ProcessedResponse{
		UsedTokens:   intPtr(150),
		WasTruncated: true,
		FinishReason: "FINISH_REASON_MAX_TOKENS",
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

func TestVertexAnthropic_ParseStreamingResponse(t *testing.T) {
	ext := NewVertexAnthropic20240101Extension()
	responseBody := `{"stop_reason": "max_tokens", "usage": {"output_tokens": 200}}
{"type": "content_block_delta", "delta": {"text": "Hello"}}`

	result, err := ext.ParseStreamingResponse([]byte(responseBody))

	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v", err)
		return
	}

	expected := &ProcessedResponse{
		UsedTokens:   intPtr(200),
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

func TestVertexGoogle_ValidateProcessingConfig(t *testing.T) {
	ext := NewVertexGoogle20240101Extension()

	// Valid config
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(50000),
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

	// Invalid config - exceeds max tokens limit
	exceedsLimitConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(70000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(exceedsLimitConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 65535 for Vertex Google Gemini 2024-01-01" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 65535 for Vertex Google Gemini 2024-01-01")
	}
}

func TestVertexAnthropic_ValidateProcessingConfig(t *testing.T) {
	ext := NewVertexAnthropic20240101Extension()

	// Valid config
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

	// Invalid config - exceeds max tokens limit
	exceedsLimitConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(70000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(exceedsLimitConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 64000 for Vertex Anthropic Claude 2024-01-01" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 64000 for Vertex Anthropic Claude 2024-01-01")
	}
}

// Enhanced edge case tests for Vertex extensions
func TestVertexGoogle_ParseStreamingResponse_EdgeCases(t *testing.T) {
	ext := NewVertexGoogle20240101Extension()

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
			responseBody: `{"candidates": [{"finishReason": "FINISH_REASON_STOP"}], "usageMetadata": {"candidatesTokenCount": 125}}
invalid json line
{"candidates": [{"content": {"parts": [{"text": "test"}]}}]}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(125),
				WasTruncated: false,
				FinishReason: "FINISH_REASON_STOP",
			},
			expectError: false,
		},
		{
			name: "response with missing candidates array",
			responseBody: `{"usageMetadata": {"candidatesTokenCount": 75}}
{"invalidStructure": true}`,
			expected: &ProcessedResponse{
				UsedTokens: intPtr(75),
			},
			expectError: false,
		},
		{
			name:         "response with empty candidates array",
			responseBody: `{"candidates": [], "usageMetadata": {"candidatesTokenCount": 50}}`,
			expected: &ProcessedResponse{
				UsedTokens: intPtr(50),
			},
			expectError: false,
		},
		{
			name:         "response with multiple candidates (uses first)",
			responseBody: `{"candidates": [{"finishReason": "FINISH_REASON_MAX_TOKENS"}, {"finishReason": "FINISH_REASON_STOP"}], "usageMetadata": {"candidatesTokenCount": 200}}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(200),
				WasTruncated: true,
				FinishReason: "FINISH_REASON_MAX_TOKENS",
			},
			expectError: false,
		},
		{
			name: "response with non-numeric token count",
			responseBody: `{"usageMetadata": {"candidatesTokenCount": "invalid"}}
{"candidates": [{"finishReason": "FINISH_REASON_STOP"}], "usageMetadata": {"candidatesTokenCount": 100}}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(100),
				WasTruncated: false,
				FinishReason: "FINISH_REASON_STOP",
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

func TestVertexAnthropic_ParseStreamingResponse_EdgeCases(t *testing.T) {
	ext := NewVertexAnthropic20240101Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name: "response with stop_reason but no usage",
			responseBody: `{"stop_reason": "end_turn"}
{"type": "content_block_delta", "delta": {"text": "Hello"}}`,
			expected: &ProcessedResponse{
				FinishReason: "end_turn",
				WasTruncated: false,
			},
			expectError: false,
		},
		{
			name: "response with usage but no stop_reason",
			responseBody: `{"usage": {"output_tokens": 300}}
{"type": "content_block_delta", "delta": {"text": "Hello"}}`,
			expected: &ProcessedResponse{
				UsedTokens: intPtr(300),
			},
			expectError: false,
		},
		{
			name:         "response with zero token count",
			responseBody: `{"stop_reason": "stop", "usage": {"output_tokens": 0}}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(0),
				WasTruncated: false,
				FinishReason: "stop",
			},
			expectError: false,
		},
		{
			name: "response with non-numeric token count",
			responseBody: `{"usage": {"output_tokens": "not_a_number"}, "stop_reason": "stop"}
{"usage": {"output_tokens": 250}, "stop_reason": "stop"}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(250),
				WasTruncated: false,
				FinishReason: "stop",
			},
			expectError: false,
		},
		{
			name: "response with complex nested structure",
			responseBody: `{"stop_reason": "max_tokens", "usage": {"output_tokens": 400, "input_tokens": 100, "cache_creation_input_tokens": 0}}
{"type": "content_block_start", "content_block": {"type": "text", "text": ""}}
{"type": "content_block_delta", "delta": {"text": "Hello world"}}`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(400),
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

// Test nil config scenarios for all Vertex extensions
func TestVertexExtensions_ValidateNilConfig(t *testing.T) {
	extensions := []struct {
		name string
		ext  interface {
			ValidateProcessingConfig(config *ProcessingConfig) error
		}
	}{
		{"Google", NewVertexGoogle20240101Extension()},
		{"Anthropic", NewVertexAnthropic20240101Extension()},
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

// Test boundary conditions for Vertex Google extension
func TestVertexGoogle_ValidateProcessingConfig_BoundaryConditions(t *testing.T) {
	ext := NewVertexGoogle20240101Extension()

	tests := []struct {
		name          string
		config        *ProcessingConfig
		expectError   bool
		expectedError string
	}{
		{
			name: "at maximum limit",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(65535),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "zero max tokens",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(0),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError:   true,
			expectedError: "OutputTokensBaseValue must be positive, got: 0",
		},
		{
			name: "one token",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(1),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "exceeding limit by one",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(65536),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError:   true,
			expectedError: "max_tokens cannot exceed 65535 for Vertex Google Gemini 2024-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ext.ValidateProcessingConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateProcessingConfig() expected error but got none")
				} else if err.Error() != tt.expectedError {
					t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

// Test boundary conditions for Vertex Anthropic extension
func TestVertexAnthropic_ValidateProcessingConfig_BoundaryConditions(t *testing.T) {
	ext := NewVertexAnthropic20240101Extension()

	tests := []struct {
		name          string
		config        *ProcessingConfig
		expectError   bool
		expectedError string
	}{
		{
			name: "at maximum limit",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(64000),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "negative max tokens",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(-1),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError:   true,
			expectedError: "OutputTokensBaseValue must be positive, got: -1",
		},
		{
			name: "exceeding limit by one",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(64001),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError:   true,
			expectedError: "max_tokens cannot exceed 64000 for Vertex Anthropic Claude 2024-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ext.ValidateProcessingConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateProcessingConfig() expected error but got none")
				} else if err.Error() != tt.expectedError {
					t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
				}
			}
		})
	}
}
