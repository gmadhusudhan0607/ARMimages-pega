/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package processors

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/cache"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/strategies"
)

func TestBaseProcessor_ProcessRequest(t *testing.T) {
	tests := []struct {
		name           string
		config         *extensions.ProcessingConfig
		requestBody    string
		expectedBody   string
		expectedTokens *int
		expectError    bool
	}{
		{
			name: "No processing - disabled strategy",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyDisabled,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			requestBody:    `{"messages":[{"role":"user","content":"Hello"}],"max_tokens":100}`,
			expectedBody:   `{"messages":[{"role":"user","content":"Hello"}],"max_tokens":100}`,
			expectedTokens: nil, // No strategy means no token processing
			expectError:    false,
		},
		{
			name: "Collect metric only - Monitoring-Only strategy",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			requestBody:    `{"messages":[{"role":"user","content":"Hello"}],"max_tokens":100}`,
			expectedBody:   `{"messages":[{"role":"user","content":"Hello"}],"max_tokens":100}`,
			expectedTokens: nil, // No strategy means no token processing
			expectError:    false,
		},
		{
			name: "Fixed strategy with value",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        intPtr(200),
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			},
			requestBody:    `{"messages":[{"role":"user","content":"Hello"}],"max_tokens":100}`,
			expectedBody:   `{"messages":[{"role":"user","content":"Hello"}],"max_tokens":200}`,
			expectedTokens: nil, // No strategy in BaseProcessor means no token processing
			expectError:    false,
		},
		{
			name: "Copyright protection enabled",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   true,
			},
			requestBody:  `{"messages":[{"role":"user","content":"Hello"}]}`,
			expectedBody: `{"messages":[{"role":"user","content":"Hello"}]}`, // Will be modified by copyright protection
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extension := extensions.NewAzureOpenAI20240201Extension()
			processor := NewBaseProcessor(extension, tt.config)

			result, err := processor.ProcessRequest(context.Background(), []byte(tt.requestBody))

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			// For copyright protection test, just verify that processing occurred
			if tt.config.CopyrightProtectionEnabled {
				if !result.HasSystemPrompt {
					t.Errorf("Expected system prompt to be processed for copyright protection")
				}
			}

			if tt.expectedTokens != nil {
				if result.OriginalTokens == nil {
					t.Errorf("Expected original tokens %d but got nil", *tt.expectedTokens)
				} else if *result.OriginalTokens != *tt.expectedTokens {
					t.Errorf("Expected original tokens %d but got %d", *tt.expectedTokens, *result.OriginalTokens)
				}
			} else {
				// When expectedTokens is nil, we expect OriginalTokens to also be nil
				if result.OriginalTokens != nil {
					t.Errorf("Expected original tokens to be nil but got %d", *result.OriginalTokens)
				}
			}
		})
	}
}

func TestBaseProcessor_ProcessResponse_NonStreaming(t *testing.T) {
	tests := []struct {
		name              string
		responseBody      string
		expectedTokens    *int
		expectedTruncated bool
		expectedFinish    string
	}{
		{
			name: "Normal completion",
			responseBody: `{
				"choices": [{"finish_reason": "stop"}],
				"usage": {"completion_tokens": 50}
			}`,
			expectedTokens:    intPtr(50),
			expectedTruncated: false,
			expectedFinish:    "stop",
		},
		{
			name: "Truncated completion",
			responseBody: `{
				"choices": [{"finish_reason": "length"}],
				"usage": {"completion_tokens": 100}
			}`,
			expectedTokens:    intPtr(100),
			expectedTruncated: true,
			expectedFinish:    "length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extension := extensions.NewAzureOpenAI20240201Extension()
			cfg := &extensions.ProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				OutputTokensBaseValue:        nil,
				OutputTokensAdjustmentForced: false,
				CopyrightProtectionEnabled:   false,
			}
			processor := NewBaseProcessor(extension, cfg)

			// Create mock response
			resp := &http.Response{
				Header: make(http.Header),
				Body:   io.NopCloser(strings.NewReader(tt.responseBody)),
			}
			resp.Header.Set("Content-Type", "application/json")

			result, err := processor.ProcessResponse(context.Background(), resp)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			if tt.expectedTokens != nil {
				if result.UsedTokens == nil {
					t.Errorf("Expected used tokens %d but got nil", *tt.expectedTokens)
				} else if *result.UsedTokens != *tt.expectedTokens {
					t.Errorf("Expected used tokens %d but got %d", *tt.expectedTokens, *result.UsedTokens)
				}
			}

			if result.WasTruncated != tt.expectedTruncated {
				t.Errorf("Expected truncated %v but got %v", tt.expectedTruncated, result.WasTruncated)
			}

			if result.FinishReason != tt.expectedFinish {
				t.Errorf("Expected finish reason %s but got %s", tt.expectedFinish, result.FinishReason)
			}
		})
	}
}

func TestBaseProcessor_UpdateMetrics(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	cfg := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	processor := NewBaseProcessor(extension, cfg)

	// Create test metadata
	testMetadata := &metadata.RequestMetadata{
		RequestMetrics: metrics.RequestMetrics{
			TokenMetrics: metrics.TokenMetrics{},
			RetryMetrics: metrics.RetryMetrics{},
		},
	}

	// Create test processed request and response
	processedReq := &ProcessedRequest{
		OriginalTokens: intPtr(100),
		ModifiedTokens: intPtr(80),
	}

	processedResp := &extensions.ProcessedResponse{
		UsedTokens:   intPtr(50),
		WasTruncated: true,
		FinishReason: "length",
	}

	// Update metrics
	err := processor.UpdateMetrics(testMetadata, processedReq, processedResp)
	if err != nil {
		t.Errorf("Unexpected error updating metrics: %v", err)
	}

	// Verify metrics were updated
	if testMetadata.RequestMetrics.TokenMetrics.Requested == nil || *testMetadata.RequestMetrics.TokenMetrics.Requested != 100 {
		t.Errorf("Expected original tokens 100 but got %v", testMetadata.RequestMetrics.TokenMetrics.Requested)
	}

	if testMetadata.RequestMetrics.TokenMetrics.Adjusted == nil || *testMetadata.RequestMetrics.TokenMetrics.Adjusted != 80 {
		t.Errorf("Expected adjusted tokens 80 but got %v", testMetadata.RequestMetrics.TokenMetrics.Adjusted)
	}

	if testMetadata.RequestMetrics.TokenMetrics.Used == nil || *testMetadata.RequestMetrics.TokenMetrics.Used != 50 {
		t.Errorf("Expected used tokens 50 but got %v", testMetadata.RequestMetrics.TokenMetrics.Used)
	}

	if !testMetadata.RequestMetrics.RetryMetrics.ResponseTruncated {
		t.Errorf("Expected response truncated to be true")
	}

	if testMetadata.RequestMetrics.RetryMetrics.Reason == nil || *testMetadata.RequestMetrics.RetryMetrics.Reason != "length_limit" {
		t.Errorf("Expected retry reason 'length_limit' but got %v", testMetadata.RequestMetrics.RetryMetrics.Reason)
	}
}

func TestBaseProcessor_UpdateMetrics_NilMetadata(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	cfg := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue:        nil,
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}
	processor := NewBaseProcessor(extension, cfg)

	// This should return an error for nil metadata
	err := processor.UpdateMetrics(nil, nil, nil)
	if err == nil {
		t.Errorf("Expected error for nil metadata but got none")
	}
}

// TestNewBaseProcessorWithStrategy tests the constructor with custom strategy
func TestNewBaseProcessorWithStrategy(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	cfg := &extensions.ProcessingConfig{
		OutputTokensStrategy:         config.OutputTokensStrategyFixed,
		OutputTokensBaseValue:        intPtr(150),
		OutputTokensAdjustmentForced: false,
		CopyrightProtectionEnabled:   false,
	}

	// Create a real strategy for testing
	realStrategy := strategies.NewFixedTokenStrategy(100)

	processor := NewBaseProcessorWithStrategy(extension, cfg, realStrategy)

	if processor == nil {
		t.Fatal("Expected processor but got nil")
	} else {
		if processor.extension != extension {
			t.Error("Extension not set correctly")
		}

		if processor.config != cfg {
			t.Error("Config not set correctly")
		}

		if processor.tokenStrategy == nil {
			t.Error("Token strategy not set")
		} else {

			// Strategy name verification is implementation-dependent, just verify it's set
			strategyName := processor.tokenStrategy.GetStrategyName()
			if strategyName == "" {
				t.Error("Expected non-empty strategy name")
			}
		}
	}
}

// TestExtractIntValue tests integer value extraction
func TestExtractIntValue(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	processor := NewBaseProcessor(extension, &extensions.ProcessingConfig{})

	tests := []struct {
		name        string
		value       interface{}
		expected    *int
		expectError bool
	}{
		{
			name:        "Int value",
			value:       123,
			expected:    intPtr(123),
			expectError: false,
		},
		{
			name:        "Float64 value",
			value:       123.0,
			expected:    intPtr(123),
			expectError: false,
		},
		{
			name:        "String number",
			value:       "456",
			expected:    intPtr(456),
			expectError: false,
		},
		{
			name:        "Invalid string",
			value:       "not-a-number",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Boolean value",
			value:       true,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.extractIntValue(tt.value)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expected == nil && result != nil {
				t.Errorf("Expected nil but got %d", *result)
			} else if tt.expected != nil && result == nil {
				t.Error("Expected value but got nil")
			} else if tt.expected != nil && result != nil && *tt.expected != *result {
				t.Errorf("Expected %d but got %d", *tt.expected, *result)
			}
		})
	}
}

// TestExtractFloatValue tests float value extraction
func TestExtractFloatValue(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	processor := NewBaseProcessor(extension, &extensions.ProcessingConfig{})

	tests := []struct {
		name        string
		value       interface{}
		expected    *float64
		expectError bool
	}{
		{
			name:        "Int value",
			value:       123,
			expected:    float64Ptr(123.0),
			expectError: false,
		},
		{
			name:        "Float64 value",
			value:       123.45,
			expected:    float64Ptr(123.45),
			expectError: false,
		},
		{
			name:        "String number",
			value:       "456.78",
			expected:    float64Ptr(456.78),
			expectError: false,
		},
		{
			name:        "Invalid string",
			value:       "not-a-number",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Boolean value",
			value:       false,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.extractFloatValue(tt.value)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expected == nil && result != nil {
				t.Errorf("Expected nil but got %f", *result)
			} else if tt.expected != nil && result == nil {
				t.Error("Expected value but got nil")
			} else if tt.expected != nil && result != nil && *tt.expected != *result {
				t.Errorf("Expected %f but got %f", *tt.expected, *result)
			}
		})
	}
}

// TestIsStreamingRequest tests streaming request detection
func TestIsStreamingRequest(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	processor := NewBaseProcessor(extension, &extensions.ProcessingConfig{})

	tests := []struct {
		name     string
		body     []byte
		expected bool
	}{
		{
			name:     "Streaming enabled",
			body:     []byte(`{"stream": true, "messages": []}`),
			expected: true,
		},
		{
			name:     "Streaming disabled",
			body:     []byte(`{"stream": false, "messages": []}`),
			expected: false,
		},
		{
			name:     "String true",
			body:     []byte(`{"stream": "true", "messages": []}`),
			expected: true,
		},
		{
			name:     "String false",
			body:     []byte(`{"stream": "false", "messages": []}`),
			expected: false,
		},
		{
			name:     "No stream parameter",
			body:     []byte(`{"messages": []}`),
			expected: false,
		},
		{
			name:     "Invalid JSON",
			body:     []byte(`invalid json`),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.isStreamingRequest(tt.body)
			if result != tt.expected {
				t.Errorf("Expected %v but got %v", tt.expected, result)
			}
		})
	}
}

// TestProcessSystemPrompt tests system prompt processing
func TestProcessSystemPrompt(t *testing.T) {
	ctx := context.Background()
	extension := extensions.NewAzureOpenAI20240201Extension()
	cfg := &extensions.ProcessingConfig{
		CopyrightProtectionEnabled: true,
	}
	processor := NewBaseProcessor(extension, cfg)

	tests := []struct {
		name           string
		body           []byte
		expectedPrompt bool
		expectError    bool
	}{
		{
			name:           "Valid messages array",
			body:           []byte(`{"messages": [{"role": "user", "content": "Hello"}]}`),
			expectedPrompt: true,
			expectError:    false,
		},
		{
			name:           "Empty messages array",
			body:           []byte(`{"messages": []}`),
			expectedPrompt: true,
			expectError:    false,
		},
		{
			name:           "No messages field",
			body:           []byte(`{"prompt": "Hello"}`),
			expectedPrompt: false,
			expectError:    true,
		},
		{
			name:           "Invalid JSON",
			body:           []byte(`invalid json`),
			expectedPrompt: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasPrompt, err := processor.processSystemPrompt(ctx, tt.body)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if hasPrompt != tt.expectedPrompt {
				t.Errorf("Expected hasPrompt %v but got %v", tt.expectedPrompt, hasPrompt)
			}

			if result == nil {
				t.Error("Expected modified body but got nil")
			}
		})
	}
}

// TestInjectSystemMessageToArray tests system message injection
func TestInjectSystemMessageToArray(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	processor := NewBaseProcessor(extension, &extensions.ProcessingConfig{})

	testPrompt := "Test system prompt"

	tests := []struct {
		name        string
		body        []byte
		expectError bool
	}{
		{
			name:        "Valid messages array",
			body:        []byte(`{"messages": [{"role": "user", "content": "Hello"}]}`),
			expectError: false,
		},
		{
			name:        "Empty messages array",
			body:        []byte(`{"messages": []}`),
			expectError: false,
		},
		{
			name:        "No messages field",
			body:        []byte(`{"prompt": "Hello"}`),
			expectError: true,
		},
		{
			name:        "Messages not array",
			body:        []byte(`{"messages": "not-an-array"}`),
			expectError: true,
		},
		{
			name:        "Invalid JSON",
			body:        []byte(`invalid json`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, hasPrompt, err := processor.injectSystemMessageToArray(tt.body, testPrompt)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if hasPrompt {
					t.Error("Expected hasPrompt to be false on error")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !hasPrompt {
				t.Error("Expected hasPrompt to be true")
			}

			if result == nil {
				t.Error("Expected modified body but got nil")
			}

			// Verify the system message was added
			var modifiedData map[string]interface{}
			if err := json.Unmarshal(result, &modifiedData); err != nil {
				t.Fatalf("Failed to parse modified JSON: %v", err)
			}

			messages, ok := modifiedData["messages"].([]interface{})
			if !ok {
				t.Fatal("Messages is not an array in modified data")
			}

			// Find system message
			found := false
			for _, msg := range messages {
				if msgMap, ok := msg.(map[string]interface{}); ok {
					if role, ok := msgMap["role"].(string); ok && role == "system" {
						if content, ok := msgMap["content"].(string); ok && content == testPrompt {
							found = true
							break
						}
					}
				}
			}

			if !found {
				t.Error("System message not found in modified messages array")
			}
		})
	}
}

// Helper function to create int pointers
func intPtr(i int) *int {
	return &i
}

// Helper function to create float64 pointers
func float64Ptr(f float64) *float64 {
	return &f
}

// New tests for calculateAdjustedTokens
func TestCalculateAdjustedTokens_NoStrategy_ReturnsNil(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	processor := NewBaseProcessor(extension, &extensions.ProcessingConfig{})
	adjusted := processor.calculateAdjustedTokens(context.Background(), intPtr(100))
	if adjusted != nil {
		t.Errorf("Expected nil adjusted tokens when no strategy is configured, got %v", *adjusted)
	}
}

func TestCalculateAdjustedTokens_FixedStrategy_RespectsModelMaximumAndConfig(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	cfg := &extensions.ProcessingConfig{OutputTokensBaseValue: intPtr(180)}
	processor := NewBaseProcessorWithStrategy(extension, cfg, strategies.NewFixedTokenStrategy(100))

	// Prepare RequestMetadata with target model maxOutputTokens = 150
	md := &metadata.RequestMetadata{
		TargetModel: &modeltypes.Model{
			Parameters: map[string]modeltypes.ParameterSpec{
				"maxOutputTokens": {Maximum: 150.0},
			},
		},
	}
	ctx := context.WithValue(context.Background(), metrics.RequestMetadataContextKey{}, md)

	adjusted := processor.calculateAdjustedTokens(ctx, nil)
	if adjusted == nil {
		t.Fatalf("Expected adjusted tokens but got nil")
	} else if *adjusted != 150 {
		t.Errorf("Expected adjusted tokens to be 150 but got %d", *adjusted)
	}
}

func TestExtractModelMaximum(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	processor := NewBaseProcessor(extension, &extensions.ProcessingConfig{})

	// Case 1: No metadata in context -> nil
	if got := processor.extractModelMaximum(context.Background()); got != nil {
		t.Fatalf("expected nil when no metadata in context, got %v", *got)
	}

	// Helper to run cases with different Maximum types
	runCase := func(value interface{}, expected float64) {
		md := &metadata.RequestMetadata{
			TargetModel: &modeltypes.Model{
				Parameters: map[string]modeltypes.ParameterSpec{
					"maxOutputTokens": {Maximum: value},
				},
			},
		}
		ctx := context.WithValue(context.Background(), metrics.RequestMetadataContextKey{}, md)
		got := processor.extractModelMaximum(ctx)
		if got == nil {
			t.Fatalf("expected %v but got nil for value=%v", expected, value)
		} else if *got != expected {
			t.Fatalf("expected %v but got %v for value=%v", expected, *got, value)
		}
	}

	// Case 2: float64 maximum
	runCase(150.0, 150.0)

	// Case 3: int maximum
	runCase(200, 200.0)

	// Case 4: string maximum
	runCase("123.45", 123.45)
}

func TestUpdateCache_AutoIncreasingStrategy_UpdatesCacheAndMetric(t *testing.T) {
	// Prepare token cache and strategy
	tokenCache := cache.NewTokenCache(10)
	auto := strategies.NewAutoIncreasingStrategy(tokenCache, 100)

	extension := extensions.NewAzureOpenAI20240201Extension()
	cfg := &extensions.ProcessingConfig{}
	processor := NewBaseProcessorWithStrategy(extension, cfg, auto)

	// Prepare metadata and context
	md := &metadata.RequestMetadata{
		IsolationID: "iso-1",
		TargetModel: &modeltypes.Model{
			Creator:  modeltypes.Creator("test-creator"),
			Provider: modeltypes.Provider("azure"),
			Name:     "model-name",
			Version:  "v1",
			KEY:      "model-key",
		},
	}
	ctx := context.WithValue(context.Background(), metrics.RequestMetadataContextKey{}, md)

	cacheKey := processor.createCacheKey(ctx)

	// Ensure cache empty initially
	if _, exists := tokenCache.Get(cacheKey); exists {
		t.Fatal("expected cache to be empty before UpdateCache")
	}

	// Call UpdateCache with usedTokens and configValue
	usedTokens := 250
	configValue := 200
	processor.UpdateCache(ctx, usedTokens, configValue)

	// Expect stored value to be max(usedTokens, configValue) = 250
	if v, exists := tokenCache.Get(cacheKey); !exists {
		t.Fatalf("expected cache value to exist after UpdateCache")
	} else if v != usedTokens {
		t.Fatalf("expected cached value %d but got %d", usedTokens, v)
	}
}

// TestBaseProcessor_ProcessResponse_NonStreaming_ReasoningTokens tests reasoning tokens extraction in non-streaming responses
func TestBaseProcessor_ProcessResponse_NonStreaming_ReasoningTokens(t *testing.T) {
	tests := []struct {
		name                    string
		responseBody            string
		expectedTokens          *int
		expectedReasoningTokens *int
		expectedFinish          string
	}{
		{
			name: "Response with reasoning tokens",
			responseBody: `{
				"choices": [{"finish_reason": "stop"}],
				"usage": {"completion_tokens": 200, "completion_tokens_details": {"reasoning_tokens": 1024}}
			}`,
			expectedTokens:          intPtr(200),
			expectedReasoningTokens: intPtr(1024),
			expectedFinish:          "stop",
		},
		{
			name: "Response without reasoning tokens",
			responseBody: `{
				"choices": [{"finish_reason": "stop"}],
				"usage": {"completion_tokens": 50}
			}`,
			expectedTokens:          intPtr(50),
			expectedReasoningTokens: nil,
			expectedFinish:          "stop",
		},
		{
			name: "Response with zero reasoning tokens",
			responseBody: `{
				"choices": [{"finish_reason": "stop"}],
				"usage": {"completion_tokens": 100, "completion_tokens_details": {"reasoning_tokens": 0}}
			}`,
			expectedTokens:          intPtr(100),
			expectedReasoningTokens: nil, // Zero should not set the pointer
			expectedFinish:          "stop",
		},
		{
			name: "Response with large reasoning tokens (deep thinking)",
			responseBody: `{
				"choices": [{"finish_reason": "stop"}],
				"usage": {"completion_tokens": 500, "completion_tokens_details": {"reasoning_tokens": 16384}}
			}`,
			expectedTokens:          intPtr(500),
			expectedReasoningTokens: intPtr(16384),
			expectedFinish:          "stop",
		},
		{
			name: "Response with completion_tokens_details but no reasoning_tokens key",
			responseBody: `{
				"choices": [{"finish_reason": "stop"}],
				"usage": {"completion_tokens": 60, "completion_tokens_details": {"accepted_prediction_tokens": 10}}
			}`,
			expectedTokens:          intPtr(60),
			expectedReasoningTokens: nil,
			expectedFinish:          "stop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extension := extensions.NewAzureOpenAI20240201Extension()
			cfg := &extensions.ProcessingConfig{
				OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
			}
			processor := NewBaseProcessor(extension, cfg)

			resp := &http.Response{
				Header: make(http.Header),
				Body:   io.NopCloser(strings.NewReader(tt.responseBody)),
			}
			resp.Header.Set("Content-Type", "application/json")

			result, err := processor.ProcessResponse(context.Background(), resp)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if (result.UsedTokens == nil) != (tt.expectedTokens == nil) {
				t.Errorf("UsedTokens nil mismatch")
			} else if result.UsedTokens != nil && *result.UsedTokens != *tt.expectedTokens {
				t.Errorf("UsedTokens: got %d, want %d", *result.UsedTokens, *tt.expectedTokens)
			}

			if (result.ReasoningTokens == nil) != (tt.expectedReasoningTokens == nil) {
				t.Errorf("ReasoningTokens nil mismatch: got nil=%v, want nil=%v", result.ReasoningTokens == nil, tt.expectedReasoningTokens == nil)
			} else if result.ReasoningTokens != nil && *result.ReasoningTokens != *tt.expectedReasoningTokens {
				t.Errorf("ReasoningTokens: got %d, want %d", *result.ReasoningTokens, *tt.expectedReasoningTokens)
			}

			if result.FinishReason != tt.expectedFinish {
				t.Errorf("FinishReason: got %s, want %s", result.FinishReason, tt.expectedFinish)
			}
		})
	}
}

// TestBaseProcessor_UpdateMetrics_ReasoningTokens tests that reasoning tokens flow through UpdateMetrics
func TestBaseProcessor_UpdateMetrics_ReasoningTokens(t *testing.T) {
	extension := extensions.NewAzureOpenAI20240201Extension()
	cfg := &extensions.ProcessingConfig{OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly}
	processor := NewBaseProcessor(extension, cfg)

	t.Run("reasoning tokens propagated to metadata", func(t *testing.T) {
		md := &metadata.RequestMetadata{
			RequestMetrics: metrics.RequestMetrics{
				TokenMetrics: metrics.TokenMetrics{},
			},
		}
		processedReq := &ProcessedRequest{}
		processedResp := &extensions.ProcessedResponse{
			UsedTokens:      intPtr(200),
			ReasoningTokens: intPtr(1024),
			FinishReason:    "stop",
		}

		err := processor.UpdateMetrics(md, processedReq, processedResp)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if md.RequestMetrics.TokenMetrics.ReasoningTokens == nil {
			t.Fatal("Expected ReasoningTokens to be set")
		}
		if *md.RequestMetrics.TokenMetrics.ReasoningTokens != 1024 {
			t.Errorf("Expected ReasoningTokens=1024, got %f", *md.RequestMetrics.TokenMetrics.ReasoningTokens)
		}
	})

	t.Run("nil reasoning tokens not propagated", func(t *testing.T) {
		md := &metadata.RequestMetadata{
			RequestMetrics: metrics.RequestMetrics{
				TokenMetrics: metrics.TokenMetrics{},
			},
		}
		processedReq := &ProcessedRequest{}
		processedResp := &extensions.ProcessedResponse{
			UsedTokens:      intPtr(50),
			ReasoningTokens: nil,
			FinishReason:    "stop",
		}

		err := processor.UpdateMetrics(md, processedReq, processedResp)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if md.RequestMetrics.TokenMetrics.ReasoningTokens != nil {
			t.Errorf("Expected ReasoningTokens to be nil, got %f", *md.RequestMetrics.TokenMetrics.ReasoningTokens)
		}
	})
}

func TestUpdateCache_PercentileStrategy_AddsSampleToCache(t *testing.T) {
	// Prepare percentile cache and strategy
	pctCache := cache.NewPercentileTokenCache(10)
	percentile := 90
	ps := strategies.NewPercentileTokenStrategy(pctCache, percentile, 100, config.OutputTokensStrategy("P90"))

	extension := extensions.NewAzureOpenAI20240201Extension()
	cfg := &extensions.ProcessingConfig{}
	processor := NewBaseProcessorWithStrategy(extension, cfg, ps)

	// Prepare metadata and context
	md := &metadata.RequestMetadata{
		IsolationID: "iso-2",
		TargetModel: &modeltypes.Model{
			Creator:  modeltypes.Creator("creator-2"),
			Provider: modeltypes.Provider("azure"),
			Name:     "model-2",
			Version:  "v2",
			KEY:      "model-key-2",
		},
	}
	ctx := context.WithValue(context.Background(), metrics.RequestMetadataContextKey{}, md)

	cacheKey := processor.createCacheKey(ctx)

	// Ensure no samples initially
	if pctCache.GetSampleCount(cacheKey) != 0 {
		t.Fatal("expected no samples in percentile cache before UpdateCache")
	}

	usedTokens := 120
	configValue := 100
	processor.UpdateCache(ctx, usedTokens, configValue)

	// After update there should be one sample equal to max(usedTokens, configValue) = 120
	samples := pctCache.GetSamples(cacheKey)
	if len(samples) != 1 {
		t.Fatalf("expected 1 sample in percentile cache but got %d", len(samples))
	}
	if samples[0] != usedTokens {
		t.Fatalf("expected sample %d but got %d", usedTokens, samples[0])
	}
}
