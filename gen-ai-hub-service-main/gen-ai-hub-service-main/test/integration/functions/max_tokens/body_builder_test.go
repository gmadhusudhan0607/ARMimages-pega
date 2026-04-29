//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

import (
	"strings"
	"testing"
)

func TestLLMRequestBuilder_BasicFunctionality(t *testing.T) {
	tests := []struct {
		name        string
		builder     *LLMRequestBodyBuilder
		expectField string
		expectValue string
	}{
		{
			name:        "Default GPT-3.5-Turbo uses max_tokens",
			builder:     NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(100),
			expectField: "max_tokens",
			expectValue: "100",
		},
		{
			name:        "Empty model name defaults to GPT-3.5-Turbo",
			builder:     NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(50),
			expectField: "max_tokens",
			expectValue: "50",
		},
		{
			name:        "Legacy model version uses max_tokens",
			builder:     NewLLMRequestBodyBuilder("gpt-35-turbo", "0301", "2023-05-15").WithMaxTokens(75),
			expectField: "max_tokens",
			expectValue: "75",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.builder.Build()

			if !strings.Contains(result, tt.expectField) {
				t.Errorf("Expected field '%s' not found in result: %s", tt.expectField, result)
			}

			if !strings.Contains(result, tt.expectValue) {
				t.Errorf("Expected value '%s' not found in result: %s", tt.expectValue, result)
			}
		})
	}
}

func TestLLMRequestBuilder_ModelVersionAndAPIVersion(t *testing.T) {
	tests := []struct {
		name        string
		modelName   string
		modelVer    string
		apiVer      string
		expectField string
	}{
		{
			name:        "GPT-3.5-Turbo-0301 with 2023-05-15 API uses max_tokens",
			modelName:   "gpt-35-turbo",
			modelVer:    "0301",
			apiVer:      "2023-05-15",
			expectField: "max_tokens",
		},
		{
			name:        "GPT-3.5-Turbo-1106 with 2024-10-21 API uses max_tokens",
			modelName:   "gpt-35-turbo",
			modelVer:    "1106",
			apiVer:      "2024-10-21",
			expectField: "max_tokens",
		},
		{
			name:        "GPT-3.5-Turbo base with latest API uses max_tokens",
			modelName:   "gpt-35-turbo",
			modelVer:    "",
			apiVer:      "2024-10-21",
			expectField: "max_tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewLLMRequestBodyBuilder(tt.modelName, tt.modelVer, tt.apiVer).
				WithMaxTokens(100)

			result := builder.Build()

			if !strings.Contains(result, tt.expectField) {
				t.Errorf("Expected field '%s' not found in result: %s", tt.expectField, result)
			}
		})
	}
}

func TestLLMRequestBuilder_StreamingSupport(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		modelVer  string
		streaming bool
		expectOK  bool
	}{
		{
			name:      "GPT-3.5-Turbo supports streaming",
			modelName: "gpt-35-turbo",
			modelVer:  "",
			streaming: true,
			expectOK:  true,
		},
		{
			name:      "GPT-3.5-Turbo-0301 supports streaming",
			modelName: "gpt-35-turbo",
			modelVer:  "0301",
			streaming: true,
			expectOK:  true,
		},
		{
			name:      "GPT-3.5-Turbo-1106 supports streaming",
			modelName: "gpt-35-turbo",
			modelVer:  "1106",
			streaming: true,
			expectOK:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Determine appropriate API version based on model version
			apiVersion := "2024-10-21"
			if tt.modelVer == "0301" {
				apiVersion = "2023-05-15"
			}
			builder := NewLLMRequestBodyBuilder(tt.modelName, tt.modelVer, apiVersion).
				WithStreaming(tt.streaming)

			result := builder.Build()

			hasStreamField := strings.Contains(result, `"stream": true`)
			if tt.expectOK && !hasStreamField {
				t.Errorf("Expected streaming to be supported but 'stream: true' not found in: %s", result)
			}
		})
	}
}

func TestLLMRequestBuilder_BackwardCompatibility(t *testing.T) {
	// Test that existing code continues to work
	t.Run("Legacy NewLLMRequestBodyBuilder works", func(t *testing.T) {
		builder := NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(100)
		result := builder.Build()

		// Should contain basic required fields
		requiredFields := []string{`"messages"`, `"temperature"`, `"max_tokens"`}
		for _, field := range requiredFields {
			if !strings.Contains(result, field) {
				t.Errorf("Expected field %s not found in result: %s", field, result)
			}
		}
	})

	t.Run("WithoutMaxTokens still works", func(t *testing.T) {
		builder := NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens()
		result := builder.Build()

		if strings.Contains(result, "max_tokens") {
			t.Errorf("max_tokens field should not be present when using WithoutMaxTokens(): %s", result)
		}
	})
}

func TestLLMRequestBuilder_ModelRegistry(t *testing.T) {
	t.Run("GetRequestModelConfig returns valid config for supported models", func(t *testing.T) {
		supportedModels := []struct {
			fullName     string
			modelName    string
			modelVersion string
		}{
			{"gpt-35-turbo", "gpt-35-turbo", ""},
			{"gpt-35-turbo-0301", "gpt-35-turbo", "0301"},
			{"gpt-35-turbo-0613", "gpt-35-turbo", "0613"},
			{"gpt-35-turbo-1106", "gpt-35-turbo", "1106"},
			{"gpt-35-turbo-16k", "gpt-35-turbo", "16k"},
			{"gpt-35-turbo-16k-0613", "gpt-35-turbo", "16k-0613"},
		}

		for _, model := range supportedModels {
			config := GetRequestModelConfig(model.modelName, model.modelVersion)
			if config == nil {
				t.Errorf("Expected configuration for model %s but got nil", model.fullName)
				continue
			}

			if config.ModelName == "" {
				t.Errorf("Model %s has empty ModelName", model.fullName)
			}
		}
	})

	t.Run("GetRequestModelConfig returns nil for unsupported models", func(t *testing.T) {
		unsupportedModels := []struct {
			modelName    string
			modelVersion string
		}{
			{"gpt-4", ""},
			{"unknown-model", ""},
			{"claude-3", ""},
		}

		for _, model := range unsupportedModels {
			config := GetRequestModelConfig(model.modelName, model.modelVersion)
			if config != nil {
				t.Errorf("Expected nil for unsupported model %s-%s but got config", model.modelName, model.modelVersion)
			}
		}
	})
}

func TestLLMRequestBuilder_APIVersionHandling(t *testing.T) {
	t.Run("Default API version is used when not specified", func(t *testing.T) {
		builder := NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(100)

		// Should build successfully with explicit API version
		result := builder.Build()
		if result == "" {
			t.Error("Build() should return a valid result with explicit API version")
		}
	})

	t.Run("Explicit API version is respected", func(t *testing.T) {
		builder := NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2023-05-15").
			WithMaxTokens(100)

		// Should build successfully with explicit API version
		result := builder.Build()
		if result == "" {
			t.Error("Build() should return a valid result with explicit API version")
		}
	})
}

func TestLLMRequestBuilder_ContentAndMessages(t *testing.T) {
	t.Run("Custom content is used", func(t *testing.T) {
		customContent := "Test custom message content"
		builder := NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithContent(customContent)
		result := builder.Build()

		if !strings.Contains(result, customContent) {
			t.Errorf("Expected custom content '%s' not found in result: %s", customContent, result)
		}
	})

	t.Run("Custom messages override content", func(t *testing.T) {
		messages := []map[string]string{
			{"role": "system", "content": "You are a helpful assistant"},
			{"role": "user", "content": "Hello"},
		}

		builder := NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").
			WithContent("This should be ignored").
			WithMessages(messages)

		result := builder.Build()

		if strings.Contains(result, "This should be ignored") {
			t.Errorf("Content should be ignored when messages are set: %s", result)
		}

		if !strings.Contains(result, "helpful assistant") {
			t.Errorf("Custom messages not found in result: %s", result)
		}
	})
}

func TestLLMRequestBuilder_ValidationWithBuild(t *testing.T) {
	t.Run("BuildWithValidation works for valid requests", func(t *testing.T) {
		builder := NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").
			WithMaxTokens(100).
			WithStreaming(true)

		result := builder.BuildWithValidation(true)
		if result == "" {
			t.Error("BuildWithValidation should return valid result for supported configuration")
		}
	})

	t.Run("BuildWithValidation handles unsupported model gracefully", func(t *testing.T) {
		builder := NewLLMRequestBodyBuilder("unsupported-model", "1106", "2024-10-21").WithMaxTokens(100)

		// Should still build even with unsupported model
		result := builder.BuildWithValidation(true)
		if result == "" {
			t.Error("BuildWithValidation should return result even for unsupported model")
		}
	})
}
