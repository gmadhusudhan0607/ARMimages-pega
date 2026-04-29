//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

import (
	"fmt"
)

// RequestOptions configures test request parameters
type RequestOptions struct {
	// Request configuration
	MaxTokens   *int    // nil means don't include max_tokens
	Streaming   bool    // whether to include "stream": true
	Temperature float64 // temperature setting (default: 0.7)

	// Content configuration
	Messages []map[string]string // custom messages array
	Content  string              // shorthand for simple user message
}

// DefaultRequestOptions returns sensible defaults for test requests
func DefaultRequestOptions() *RequestOptions {
	return &RequestOptions{
		Temperature: 0.7,
		Content:     "Hello, how are you?",
	}
}

// LLMRequestBodyBuilder provides a fluent API for building test requests
type LLMRequestBodyBuilder struct {
	modelName    string
	modelVersion string
	apiVersion   string
	options      *RequestOptions
}

// NewLLMRequestBodyBuilder creates a new request builder with specific model version and API version
func NewLLMRequestBodyBuilder(targetModelName, modelVersion, apiVersion string) *LLMRequestBodyBuilder {

	if targetModelName == "" {
		panic("targetModelName cannot be empty when using LLMRequestBodyBuilder")
	}

	return &LLMRequestBodyBuilder{
		modelName:    targetModelName,
		modelVersion: modelVersion,
		apiVersion:   apiVersion,
		options:      DefaultRequestOptions(),
	}
}

// WithMaxTokens sets the max_tokens parameter
func (b *LLMRequestBodyBuilder) WithMaxTokens(maxTokens int) *LLMRequestBodyBuilder {
	b.options.MaxTokens = &maxTokens
	return b
}

// WithoutMaxTokens ensures max_tokens is not included
func (b *LLMRequestBodyBuilder) WithoutMaxTokens() *LLMRequestBodyBuilder {
	b.options.MaxTokens = nil
	return b
}

// WithStreaming enables streaming mode
func (b *LLMRequestBodyBuilder) WithStreaming(streaming bool) *LLMRequestBodyBuilder {
	b.options.Streaming = streaming
	return b
}

// WithTemperature sets the temperature parameter
func (b *LLMRequestBodyBuilder) WithTemperature(temp float64) *LLMRequestBodyBuilder {
	b.options.Temperature = temp
	return b
}

// WithContent sets simple user message content
func (b *LLMRequestBodyBuilder) WithContent(content string) *LLMRequestBodyBuilder {
	b.options.Content = content
	b.options.Messages = nil // Clear custom messages
	return b
}

// WithMessages sets custom messages array
func (b *LLMRequestBodyBuilder) WithMessages(messages []map[string]string) *LLMRequestBodyBuilder {
	b.options.Messages = messages
	b.options.Content = "" // Clear simple content
	return b
}

// Build generates the JSON request body with model/version/API awareness
func (b *LLMRequestBodyBuilder) Build() string {
	return b.BuildWithValidation(false)
}

// BuildWithValidation generates the JSON request body and optionally validates against model constraints
func (b *LLMRequestBodyBuilder) BuildWithValidation(validate bool) string {
	// Get model configuration
	modelConfig := b.getModelConfig()

	// Perform validation if requested and config is available
	if validate && modelConfig != nil {
		err := b.validateRequest(modelConfig)
		if err != nil {
			panic(fmt.Sprintf("Request validation must pass, got error: %v", err))
		}
	}

	// Determine messages
	var messages []map[string]string
	if len(b.options.Messages) > 0 {
		messages = b.options.Messages
	} else {
		content := b.options.Content
		if content == "" {
			panic("Content must be provided when not using custom messages - no fallback to default content")
		}
		messages = []map[string]string{
			{"role": "user", "content": content},
		}
	}

	// Build request body parts
	bodyParts := []string{}

	// Add messages (always required)
	bodyParts = append(bodyParts, fmt.Sprintf(`"messages": %s`, formatMessages(messages)))

	// Add temperature
	bodyParts = append(bodyParts, fmt.Sprintf(`"temperature": %.1f`, b.options.Temperature))

	// Add token limit field (model/API version aware)
	if b.options.MaxTokens != nil {
		tokenField := "max_tokens" // default field name
		if modelConfig != nil {
			tokenField = b.getTokenLimitFieldName(modelConfig)
		}
		bodyParts = append(bodyParts, fmt.Sprintf(`"%s": %d`, tokenField, *b.options.MaxTokens))
	}

	// Add streaming if enabled
	if b.options.Streaming {
		bodyParts = append(bodyParts, `"stream": true`)
	}

	return fmt.Sprintf("{\n\t\t%s\n\t}", joinBodyParts(bodyParts))
}

// getModelConfig retrieves the model configuration for the current builder settings
func (b *LLMRequestBodyBuilder) getModelConfig() *RequestModelConfig {
	// Get configuration from registry using modelName and modelVersion directly
	return GetRequestModelConfig(b.modelName, b.modelVersion)
}

// getTokenLimitFieldName returns the appropriate field name for token limits
func (b *LLMRequestBodyBuilder) getTokenLimitFieldName(modelConfig *RequestModelConfig) string {
	if modelConfig == nil {
		panic("modelConfig must not be nil - no fallback to default token field")
	}

	apiInfo := modelConfig.GetAPIVersionInfo(b.apiVersion)
	if apiInfo == nil {
		panic(fmt.Sprintf("API version info must be available for version '%s' - no fallback to default token field", b.apiVersion))
	}

	return string(apiInfo.TokenLimitField)
}

// validateRequest validates the current request configuration against model capabilities
func (b *LLMRequestBodyBuilder) validateRequest(modelConfig *RequestModelConfig) error {
	hasMaxTokens := b.options.MaxTokens != nil
	hasStreaming := b.options.Streaming
	hasTools := false // TODO: Add tools support later

	return modelConfig.ValidateRequest(b.apiVersion, hasMaxTokens, hasStreaming, hasTools)
}

// formatMessages converts messages array to JSON string
func formatMessages(messages []map[string]string) string {
	var messageParts []string
	for _, msg := range messages {
		messageParts = append(messageParts, fmt.Sprintf(`{"role": "%s", "content": "%s"}`, msg["role"], msg["content"]))
	}
	return fmt.Sprintf("[%s]", joinBodyParts(messageParts))
}

// joinBodyParts joins body parts with proper formatting (with newlines for readability)
func joinBodyParts(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ",\n\t\t"
		}
		result += part
	}
	return result
}
