/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package extensions

import (
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

// Test Azure OpenAI 2022-12-01 Extension
func TestAzureOpenAI20221201_GetConfiguration(t *testing.T) {
	ext := NewAzureOpenAI20221201Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_tokens",
			SystemPrompt: "messages.0.content",
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.completion_tokens",
			FinishReason: "choices.0.finish_reason",
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

func TestAzureOpenAI20221201_ParseStreamingResponse(t *testing.T) {
	ext := NewAzureOpenAI20221201Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name: "valid streaming response with length truncation",
			responseBody: `data: {"choices": [{"finish_reason": "length"}], "usage": {"completion_tokens": 150}}
data: {"choices": [{"delta": {"content": "Hello"}}]}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(150),
				WasTruncated: true,
				FinishReason: "length",
			},
		},
		{
			name: "valid streaming response without truncation",
			responseBody: `data: {"choices": [{"finish_reason": "stop"}], "usage": {"completion_tokens": 100}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(100),
				WasTruncated: false,
				FinishReason: "stop",
			},
		},
		{
			name: "response with invalid JSON chunks (should be skipped)",
			responseBody: `data: {"choices": [{"finish_reason": "stop"}]}
data: invalid json
data: {"usage": {"completion_tokens": 75}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(75),
				WasTruncated: false,
				FinishReason: "stop",
			},
		},
		{
			name:         "empty response body",
			responseBody: ``,
			expected:     &ProcessedResponse{},
		},
		{
			name: "response with non-SSE lines (should be ignored)",
			responseBody: `HTTP/1.1 200 OK
Content-Type: text/event-stream

data: {"choices": [{"finish_reason": "stop"}], "usage": {"completion_tokens": 50}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(50),
				WasTruncated: false,
				FinishReason: "stop",
			},
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

func TestAzureOpenAI20221201_ValidateProcessingConfig(t *testing.T) {
	ext := NewAzureOpenAI20221201Extension()

	tests := []struct {
		name          string
		config        *ProcessingConfig
		expectError   bool
		expectedError string
	}{
		{
			name: "valid config",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(1000),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError: false,
		},
		{
			name: "valid config with disabled strategy",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   true,
			},
			expectError: false,
		},
		{
			name:          "nil config",
			config:        nil,
			expectError:   true,
			expectedError: "processing config cannot be nil",
		},
		{
			name: "negative max tokens",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(-10),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError:   true,
			expectedError: "OutputTokensBaseValue must be positive, got: -10",
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
			name: "exceeding max tokens limit",
			config: &ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(3000),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			expectError:   true,
			expectedError: "max_tokens cannot exceed 2048 for Azure OpenAI 2022-12-01",
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

// Test Azure OpenAI 2023-05-15 Extension
func TestAzureOpenAI20230515_GetConfiguration(t *testing.T) {
	ext := NewAzureOpenAI20230515Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_tokens",
			SystemPrompt: "messages.0.content",
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.completion_tokens",
			FinishReason: "choices.0.finish_reason",
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

func TestAzureOpenAI20230515_ParseStreamingResponse(t *testing.T) {
	ext := NewAzureOpenAI20230515Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name: "valid streaming response with stop",
			responseBody: `data: {"choices": [{"delta": {"role": "assistant"}}]}
data: {"choices": [{"delta": {"content": "Test response"}}]}
data: {"choices": [{"finish_reason": "stop"}], "usage": {"completion_tokens": 75}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(75),
				WasTruncated: false,
				FinishReason: "stop",
			},
		},
		{
			name: "streaming response with length truncation",
			responseBody: `data: {"choices": [{"delta": {"content": "Long response"}}]}
data: {"choices": [{"finish_reason": "length"}], "usage": {"completion_tokens": 200}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(200),
				WasTruncated: true,
				FinishReason: "length",
			},
		},
		{
			name: "malformed SSE data (should be skipped)",
			responseBody: `data: {"choices": [{"delta": {"content": "Valid"}}]}
malformed line without data prefix
data: invalid json chunk
data: {"usage": {"completion_tokens": 30}}
data: {"choices": [{"finish_reason": "stop"}]}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(30),
				WasTruncated: false,
				FinishReason: "stop",
			},
		},
		{
			name:         "empty response",
			responseBody: ``,
			expected:     &ProcessedResponse{},
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

func TestAzureOpenAI20230515_ValidateProcessingConfig(t *testing.T) {
	ext := NewAzureOpenAI20230515Extension()

	// Valid config at limit
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(4096),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err := ext.ValidateProcessingConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
	}

	// Invalid config - exceeding limit
	invalidConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(5000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(invalidConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 4096 for Azure OpenAI 2023-05-15" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 4096 for Azure OpenAI 2023-05-15")
	}
}

// Test Azure OpenAI 2024-02-01 Extension
func TestAzureOpenAI20240201_GetConfiguration(t *testing.T) {
	ext := NewAzureOpenAI20240201Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_tokens",
			SystemPrompt: "messages",
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.completion_tokens",
			FinishReason: "choices.0.finish_reason",
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

func TestAzureOpenAI20240201_ParseStreamingResponse(t *testing.T) {
	ext := NewAzureOpenAI20240201Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name: "streaming response with function call",
			responseBody: `data: {"choices": [{"delta": {"role": "assistant"}}]}
data: {"choices": [{"delta": {"function_call": {"name": "test_function"}}}]}
data: {"choices": [{"finish_reason": "function_call"}], "usage": {"completion_tokens": 125}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(125),
				WasTruncated: false,
				FinishReason: "function_call",
			},
		},
		{
			name: "streaming response with content truncation",
			responseBody: `data: {"choices": [{"delta": {"content": "This is a long response"}}]}
data: {"choices": [{"delta": {"content": " that gets truncated"}}]}
data: {"choices": [{"finish_reason": "length"}], "usage": {"completion_tokens": 300}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(300),
				WasTruncated: true,
				FinishReason: "length",
			},
		},
		{
			name: "streaming response with model refusal",
			responseBody: `data: {"choices": [{"delta": {"content": "I cannot"}}]}
data: {"choices": [{"finish_reason": "content_filter"}], "usage": {"completion_tokens": 15}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(15),
				WasTruncated: false,
				FinishReason: "content_filter",
			},
		},
		{
			name: "mixed valid and invalid chunks",
			responseBody: `data: {"choices": [{"delta": {"content": "Valid start"}}]}
not a data line
data: malformed json here
data: {"usage": {"completion_tokens": 85}}
data: {"choices": [{"finish_reason": "stop"}]}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(85),
				WasTruncated: false,
				FinishReason: "stop",
			},
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

func TestAzureOpenAI20240201_ValidateProcessingConfig(t *testing.T) {
	ext := NewAzureOpenAI20240201Extension()

	// Valid config at limit
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(16384),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err := ext.ValidateProcessingConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
	}

	// Invalid config - exceeding limit
	invalidConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(20000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(invalidConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 16384 for Azure OpenAI 2024-02-01" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 16384 for Azure OpenAI 2024-02-01")
	}
}

// Test Azure OpenAI 2024-06-01 Extension
func TestAzureOpenAI20240601_GetConfiguration(t *testing.T) {
	ext := NewAzureOpenAI20240601Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_tokens",
			SystemPrompt: "messages.0.content",
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.completion_tokens",
			FinishReason: "choices.0.finish_reason",
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

func TestAzureOpenAI20240601_ParseStreamingResponse(t *testing.T) {
	ext := NewAzureOpenAI20240601Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name: "streaming response with tool calls",
			responseBody: `data: {"choices": [{"delta": {"role": "assistant"}}]}
data: {"choices": [{"delta": {"tool_calls": [{"function": {"name": "search"}}]}}]}
data: {"choices": [{"finish_reason": "tool_calls"}], "usage": {"completion_tokens": 45}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(45),
				WasTruncated: false,
				FinishReason: "tool_calls",
			},
		},
		{
			name: "streaming response with normal completion",
			responseBody: `data: {"choices": [{"delta": {"content": "Here is your answer"}}]}
data: {"choices": [{"delta": {"content": " to the question."}}]}
data: {"choices": [{"finish_reason": "stop"}], "usage": {"completion_tokens": 95}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(95),
				WasTruncated: false,
				FinishReason: "stop",
			},
		},
		{
			name: "streaming response with length limit reached",
			responseBody: `data: {"choices": [{"delta": {"content": "This response exceeds"}}]}
data: {"choices": [{"delta": {"content": " the maximum length"}}]}
data: {"choices": [{"finish_reason": "length"}], "usage": {"completion_tokens": 500}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(500),
				WasTruncated: true,
				FinishReason: "length",
			},
		},
		{
			name: "streaming with partial invalid data",
			responseBody: `data: {"choices": [{"delta": {"content": "Start"}}]}
invalid sse line here
data: corrupted json data
data: {"usage": {"completion_tokens": 25}}
data: {"choices": [{"finish_reason": "stop"}]}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(25),
				WasTruncated: false,
				FinishReason: "stop",
			},
		},
		{
			name:         "completely empty response",
			responseBody: ``,
			expected:     &ProcessedResponse{},
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

func TestAzureOpenAI20240601_ValidateProcessingConfig(t *testing.T) {
	ext := NewAzureOpenAI20240601Extension()

	// Valid config at limit
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(32768),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err := ext.ValidateProcessingConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
	}

	// Invalid config - exceeding limit
	invalidConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(40000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(invalidConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 32768 for Azure OpenAI 2024-06-01" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 32768 for Azure OpenAI 2024-06-01")
	}
}

// Test Azure OpenAI 2024-10-21 Extension
func TestAzureOpenAI20241021_GetConfiguration(t *testing.T) {
	ext := NewAzureOpenAI20241021Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_tokens",
			SystemPrompt: "messages.0.content",
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.completion_tokens",
			FinishReason: "choices.0.finish_reason",
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

func TestAzureOpenAI20241021_ParseStreamingResponse(t *testing.T) {
	ext := NewAzureOpenAI20241021Extension()

	// Test multiple streaming chunks in realistic scenario
	responseBody := `data: {"choices": [{"delta": {"role": "assistant"}}]}
data: {"choices": [{"delta": {"content": "Hello"}}]}
data: {"choices": [{"delta": {"content": " world"}}]}
data: {"choices": [{"finish_reason": "length"}], "usage": {"completion_tokens": 250}}
data: [DONE]`

	result, err := ext.ParseStreamingResponse([]byte(responseBody))

	if err != nil {
		t.Errorf("ParseStreamingResponse() error = %v", err)
		return
	}

	expected := &ProcessedResponse{
		UsedTokens:   intPtr(250),
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

func TestAzureOpenAI20241021_ValidateProcessingConfig(t *testing.T) {
	ext := NewAzureOpenAI20241021Extension()

	// Valid config at limit
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(65536),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err := ext.ValidateProcessingConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
	}

	// Invalid config - exceeding limit
	invalidConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(70000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(invalidConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 65536 for Azure OpenAI 2024-10-21" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 65536 for Azure OpenAI 2024-10-21")
	}
}

// Test Azure OpenAI 2025-08-07 Extension
func TestAzureOpenAI20250807_GetConfiguration(t *testing.T) {
	ext := NewAzureOpenAI20250807Extension()
	result := ext.GetConfiguration()

	expected := ExtensionConfiguration{
		Request: RequestConfig{
			MaxTokens:    "max_completion_tokens",
			SystemPrompt: "messages",
		},
		Response: ResponseConfig{
			UsedTokens:   "usage.completion_tokens",
			FinishReason: "choices.0.finish_reason",
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

func TestAzureOpenAI20250807_ParseStreamingResponse(t *testing.T) {
	ext := NewAzureOpenAI20250807Extension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
		expectError  bool
	}{
		{
			name: "streaming response with function call",
			responseBody: `data: {"choices": [{"delta": {"role": "assistant"}}]}
data: {"choices": [{"delta": {"function_call": {"name": "test_function"}}}]}
data: {"choices": [{"finish_reason": "function_call"}], "usage": {"completion_tokens": 125}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(125),
				WasTruncated: false,
				FinishReason: "function_call",
			},
		},
		{
			name: "streaming response with content truncation",
			responseBody: `data: {"choices": [{"delta": {"content": "This is a long response"}}]}
data: {"choices": [{"delta": {"content": " that gets truncated"}}]}
data: {"choices": [{"finish_reason": "length"}], "usage": {"completion_tokens": 300}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(300),
				WasTruncated: true,
				FinishReason: "length",
			},
		},
		{
			name: "streaming response with model refusal",
			responseBody: `data: {"choices": [{"delta": {"content": "I cannot"}}]}
data: {"choices": [{"finish_reason": "content_filter"}], "usage": {"completion_tokens": 15}}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(15),
				WasTruncated: false,
				FinishReason: "content_filter",
			},
		},
		{
			name: "mixed valid and invalid chunks",
			responseBody: `data: {"choices": [{"delta": {"content": "Valid start"}}]}
not a data line
data: malformed json here
data: {"usage": {"completion_tokens": 85}}
data: {"choices": [{"finish_reason": "stop"}]}
data: [DONE]`,
			expected: &ProcessedResponse{
				UsedTokens:   intPtr(85),
				WasTruncated: false,
				FinishReason: "stop",
			},
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

// ============================================================================
// Reasoning Tokens Tests - All Extensions
// ============================================================================

// TestAzureOpenAI_ReasoningTokens_ParseStreamingResponse tests reasoning tokens extraction
// from completion_tokens_details in SSE streaming responses across all Azure OpenAI extensions
func TestAzureOpenAI_ReasoningTokens_ParseStreamingResponse(t *testing.T) {
	type extensionFactory struct {
		name       string
		newExtFunc func() interface {
			ParseStreamingResponse([]byte) (*ProcessedResponse, error)
		}
	}

	factories := []extensionFactory{
		{"AzureOpenAI-2024-06-01", func() interface {
			ParseStreamingResponse([]byte) (*ProcessedResponse, error)
		} {
			return NewAzureOpenAI20240601Extension()
		}},
		{"AzureOpenAI-2024-10-21", func() interface {
			ParseStreamingResponse([]byte) (*ProcessedResponse, error)
		} {
			return NewAzureOpenAI20241021Extension()
		}},
		{"AzureOpenAI-2025-08-07", func() interface {
			ParseStreamingResponse([]byte) (*ProcessedResponse, error)
		} {
			return NewAzureOpenAI20250807Extension()
		}},
		{"VertexGoogleOpenAI", func() interface {
			ParseStreamingResponse([]byte) (*ProcessedResponse, error)
		} {
			return NewVertexGoogleOpenAIExtension()
		}},
	}

	tests := []struct {
		name                    string
		responseBody            string
		expectedUsedTokens      *int
		expectedReasoningTokens *int
		expectedFinishReason    string
		expectedTruncated       bool
	}{
		{
			name: "streaming with reasoning tokens in final chunk",
			responseBody: `data: {"choices":[{"delta":{"role":"assistant"}}]}
data: {"choices":[{"delta":{"content":"The answer is 42"}}]}
data: {"choices":[{"finish_reason":"stop"}],"usage":{"completion_tokens":80,"prompt_tokens":30,"total_tokens":110,"completion_tokens_details":{"reasoning_tokens":512}}}
data: [DONE]`,
			expectedUsedTokens:      intPtr(80),
			expectedReasoningTokens: intPtr(512),
			expectedFinishReason:    "stop",
			expectedTruncated:       false,
		},
		{
			name: "streaming without reasoning tokens (non-reasoning model)",
			responseBody: `data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"finish_reason":"stop"}],"usage":{"completion_tokens":50,"prompt_tokens":25,"total_tokens":75}}
data: [DONE]`,
			expectedUsedTokens:      intPtr(50),
			expectedReasoningTokens: nil,
			expectedFinishReason:    "stop",
			expectedTruncated:       false,
		},
		{
			name: "streaming with zero reasoning tokens (should be nil)",
			responseBody: `data: {"choices":[{"delta":{"content":"Hi"}}]}
data: {"choices":[{"finish_reason":"stop"}],"usage":{"completion_tokens":40,"completion_tokens_details":{"reasoning_tokens":0}}}
data: [DONE]`,
			expectedUsedTokens:      intPtr(40),
			expectedReasoningTokens: nil, // Zero should not set the pointer
			expectedFinishReason:    "stop",
			expectedTruncated:       false,
		},
		{
			name: "streaming with large reasoning tokens (deep thinking model)",
			responseBody: `data: {"choices":[{"delta":{"content":"Proof..."}}]}
data: {"choices":[{"finish_reason":"stop"}],"usage":{"completion_tokens":500,"completion_tokens_details":{"reasoning_tokens":16384}}}
data: [DONE]`,
			expectedUsedTokens:      intPtr(500),
			expectedReasoningTokens: intPtr(16384),
			expectedFinishReason:    "stop",
			expectedTruncated:       false,
		},
		{
			name: "streaming with reasoning tokens and length truncation",
			responseBody: `data: {"choices":[{"delta":{"content":"Long response..."}}]}
data: {"choices":[{"finish_reason":"length"}],"usage":{"completion_tokens":4096,"completion_tokens_details":{"reasoning_tokens":2048}}}
data: [DONE]`,
			expectedUsedTokens:      intPtr(4096),
			expectedReasoningTokens: intPtr(2048),
			expectedFinishReason:    "length",
			expectedTruncated:       true,
		},
		{
			name: "streaming with completion_tokens_details but no reasoning_tokens key",
			responseBody: `data: {"choices":[{"finish_reason":"stop"}],"usage":{"completion_tokens":60,"completion_tokens_details":{"accepted_prediction_tokens":10}}}
data: [DONE]`,
			expectedUsedTokens:      intPtr(60),
			expectedReasoningTokens: nil,
			expectedFinishReason:    "stop",
			expectedTruncated:       false,
		},
	}

	for _, factory := range factories {
		for _, tt := range tests {
			t.Run(factory.name+"/"+tt.name, func(t *testing.T) {
				ext := factory.newExtFunc()
				result, err := ext.ParseStreamingResponse([]byte(tt.responseBody))

				if err != nil {
					t.Fatalf("ParseStreamingResponse() unexpected error = %v", err)
				}

				// Verify used tokens
				if (result.UsedTokens == nil) != (tt.expectedUsedTokens == nil) {
					t.Errorf("UsedTokens nil mismatch: got nil=%v, want nil=%v", result.UsedTokens == nil, tt.expectedUsedTokens == nil)
				} else if result.UsedTokens != nil && tt.expectedUsedTokens != nil && *result.UsedTokens != *tt.expectedUsedTokens {
					t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *tt.expectedUsedTokens)
				}

				// Verify reasoning tokens
				if (result.ReasoningTokens == nil) != (tt.expectedReasoningTokens == nil) {
					t.Errorf("ReasoningTokens nil mismatch: got nil=%v, want nil=%v", result.ReasoningTokens == nil, tt.expectedReasoningTokens == nil)
				} else if result.ReasoningTokens != nil && tt.expectedReasoningTokens != nil && *result.ReasoningTokens != *tt.expectedReasoningTokens {
					t.Errorf("ReasoningTokens: got %d, want %d", *result.ReasoningTokens, *tt.expectedReasoningTokens)
				}

				// Verify finish reason
				if result.FinishReason != tt.expectedFinishReason {
					t.Errorf("FinishReason: got %s, want %s", result.FinishReason, tt.expectedFinishReason)
				}

				// Verify truncation
				if result.WasTruncated != tt.expectedTruncated {
					t.Errorf("WasTruncated: got %v, want %v", result.WasTruncated, tt.expectedTruncated)
				}
			})
		}
	}
}

// TestVertexGoogleOpenAI_GetConfiguration tests Vertex Google OpenAI extension configuration
func TestVertexGoogleOpenAI_GetConfiguration(t *testing.T) {
	ext := NewVertexGoogleOpenAIExtension()
	result := ext.GetConfiguration()

	if result.Request.MaxTokens != "max_tokens" {
		t.Errorf("Request.MaxTokens: got %s, want max_tokens", result.Request.MaxTokens)
	}
	if result.Response.UsedTokens != "usage.completion_tokens" {
		t.Errorf("Response.UsedTokens: got %s, want usage.completion_tokens", result.Response.UsedTokens)
	}
	if result.Response.FinishReason != "choices.0.finish_reason" {
		t.Errorf("Response.FinishReason: got %s, want choices.0.finish_reason", result.Response.FinishReason)
	}
}

// TestVertexGoogleOpenAI_ParseStreamingResponse tests the full SSE parsing flow
func TestVertexGoogleOpenAI_ParseStreamingResponse(t *testing.T) {
	ext := NewVertexGoogleOpenAIExtension()

	tests := []struct {
		name         string
		responseBody string
		expected     *ProcessedResponse
	}{
		{
			name: "normal stop with usage",
			responseBody: `data: {"choices":[{"delta":{"content":"Hello"}}]}
data: {"choices":[{"finish_reason":"stop"}],"usage":{"completion_tokens":100}}
data: [DONE]`,
			expected: &ProcessedResponse{UsedTokens: intPtr(100), FinishReason: "stop"},
		},
		{
			name: "length truncation",
			responseBody: `data: {"choices":[{"finish_reason":"length"}],"usage":{"completion_tokens":500}}
data: [DONE]`,
			expected: &ProcessedResponse{UsedTokens: intPtr(500), FinishReason: "length", WasTruncated: true},
		},
		{
			name:         "empty response",
			responseBody: ``,
			expected:     &ProcessedResponse{},
		},
		{
			name: "invalid JSON skipped gracefully",
			responseBody: `data: invalid
data: {"choices":[{"finish_reason":"stop"}],"usage":{"completion_tokens":25}}
data: [DONE]`,
			expected: &ProcessedResponse{UsedTokens: intPtr(25), FinishReason: "stop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ext.ParseStreamingResponse([]byte(tt.responseBody))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.FinishReason != tt.expected.FinishReason {
				t.Errorf("FinishReason: got %s, want %s", result.FinishReason, tt.expected.FinishReason)
			}
			if result.WasTruncated != tt.expected.WasTruncated {
				t.Errorf("WasTruncated: got %v, want %v", result.WasTruncated, tt.expected.WasTruncated)
			}
			if (result.UsedTokens == nil) != (tt.expected.UsedTokens == nil) {
				t.Errorf("UsedTokens nil mismatch: got nil=%v, want nil=%v", result.UsedTokens == nil, tt.expected.UsedTokens == nil)
			} else if result.UsedTokens != nil && tt.expected.UsedTokens != nil && *result.UsedTokens != *tt.expected.UsedTokens {
				t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *tt.expected.UsedTokens)
			}
		})
	}
}

// TestVertexGoogleOpenAI_ValidateProcessingConfig tests config validation
func TestVertexGoogleOpenAI_ValidateProcessingConfig(t *testing.T) {
	ext := NewVertexGoogleOpenAIExtension()

	if err := ext.ValidateProcessingConfig(nil); err == nil {
		t.Error("expected error for nil config")
	}
	if err := ext.ValidateProcessingConfig(&ProcessingConfig{OutputTokensBaseValue: intPtr(-1)}); err == nil {
		t.Error("expected error for negative OutputTokensBaseValue")
	}
	if err := ext.ValidateProcessingConfig(&ProcessingConfig{OutputTokensBaseValue: intPtr(70000)}); err == nil {
		t.Error("expected error for exceeding 65536 limit")
	}
	if err := ext.ValidateProcessingConfig(&ProcessingConfig{OutputTokensBaseValue: intPtr(1000)}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAzureOpenAI20250807_ValidateProcessingConfig(t *testing.T) {
	ext := NewAzureOpenAI20250807Extension()

	// Valid config at limit
	validConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(16384),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err := ext.ValidateProcessingConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateProcessingConfig() unexpected error = %v", err)
	}

	// Invalid config - exceeding limit
	invalidConfig := &ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(200000),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	err = ext.ValidateProcessingConfig(invalidConfig)
	if err == nil {
		t.Errorf("ValidateProcessingConfig() expected error but got none")
	} else if err.Error() != "max_tokens cannot exceed 128000 for Azure OpenAI 2025-08-07" {
		t.Errorf("ValidateProcessingConfig() error = %v, want %v", err.Error(), "max_tokens cannot exceed 128000 for Azure OpenAI 2025-08-07")
	}
}
