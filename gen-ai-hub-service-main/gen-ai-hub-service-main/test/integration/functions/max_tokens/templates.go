//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

import (
	"fmt"
	"strings"
)

// ResponseTemplate holds templates for generating mock responses
type ResponseTemplate struct {
	JSONResponse      string
	StreamingResponse string
}

// ResponseOptions configures response generation
type ResponseOptions struct {
	CompletionTokens int
	PromptTokens     int
	Content          string
}

// DefaultResponseOptions returns sensible defaults
func DefaultResponseOptions() *ResponseOptions {
	return &ResponseOptions{
		CompletionTokens: 50,
		PromptTokens:     10,
		Content:          "Test response",
	}
}

// GetJSONResponseTemplate returns JSON response template for a model
func GetJSONResponseTemplate(config *ModelConfig, opts *ResponseOptions) map[string]interface{} {
	if opts == nil {
		opts = DefaultResponseOptions()
	}

	return map[string]interface{}{
		"id":      config.ResponseID,
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   config.ResponseModelName,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": formatContentForModel(config, opts.Content),
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     opts.PromptTokens,
			"completion_tokens": opts.CompletionTokens,
			"total_tokens":      opts.PromptTokens + opts.CompletionTokens,
		},
	}
}

// GetStreamingResponseTemplate returns SSE streaming response template
func GetStreamingResponseTemplate(config *ModelConfig, opts *ResponseOptions) string {
	if opts == nil {
		opts = DefaultResponseOptions()
	}

	content := formatContentForModel(config, opts.Content)
	words := strings.Fields(content)

	var chunks []string

	// Initial role chunk
	chunks = append(chunks, fmt.Sprintf(
		`data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"%s","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		config.ResponseModelName))

	// Content chunks (split by words)
	if len(words) > 0 {
		// First word
		chunks = append(chunks, fmt.Sprintf(
			`data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"%s","choices":[{"index":0,"delta":{"content":"%s"},"finish_reason":null}]}`,
			config.ResponseModelName, words[0]))

		// Remaining words
		for _, word := range words[1:] {
			chunks = append(chunks, fmt.Sprintf(
				`data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"%s","choices":[{"index":0,"delta":{"content":" %s"},"finish_reason":null}]}`,
				config.ResponseModelName, word))
		}
	}

	// Final chunk
	chunks = append(chunks, fmt.Sprintf(
		`data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"%s","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		config.ResponseModelName))

	// End marker
	chunks = append(chunks, "data: [DONE]")

	// Join with double newlines and add final newline
	return strings.Join(chunks, "\n\n") + "\n\n"
}

// formatContentForModel adds model-specific content formatting
func formatContentForModel(config *ModelConfig, baseContent string) string {
	if baseContent == "" {
		baseContent = "Test response"
	}

	// For models other than gpt-35-turbo, append model name to match existing patterns
	if config.Name != "gpt-35-turbo" {
		return fmt.Sprintf("%s from %s", baseContent, config.ResponseModelName)
	}

	return baseContent
}

// RequestBodyPatterns provides common request body patterns for WireMock
type RequestBodyPatterns struct {
	MaxTokensOnly       []map[string]interface{}
	MaxTokensWithStream []map[string]interface{}
	StreamOnly          []map[string]interface{}
	WithoutMaxTokens    []map[string]interface{}
}

// GetRequestBodyPatterns returns request patterns for different scenarios
func GetRequestBodyPatterns(expectedMaxTokens int) *RequestBodyPatterns {
	return &RequestBodyPatterns{
		MaxTokensOnly: []map[string]interface{}{
			{
				"matchesJsonPath": "$.max_tokens",
			},
			{
				"equalToJson":         fmt.Sprintf(`{"max_tokens":%d}`, expectedMaxTokens),
				"ignoreExtraElements": true,
			},
		},
		MaxTokensWithStream: []map[string]interface{}{
			{
				"matchesJsonPath": fmt.Sprintf("$.max_tokens[?(@ == %d)]", expectedMaxTokens),
			},
			{
				"matchesJsonPath": "$.stream[?(@ == true)]",
			},
		},
		StreamOnly: []map[string]interface{}{
			{
				"matchesJsonPath": "$.stream[?(@ == true)]",
			},
			{
				"absent": "$.max_tokens",
			},
		},
		WithoutMaxTokens: []map[string]interface{}{
			{
				"absent": "$.max_tokens",
			},
		},
	}
}

// GetCommonHeaders returns standard headers used in requests
func GetCommonHeaders(isolationID string) map[string]interface{} {
	return map[string]interface{}{
		"X-Genai-Gateway-Isolation-ID": map[string]string{
			"equalTo": isolationID,
		},
	}
}

// GetResponseHeaders returns standard response headers
func GetResponseHeaders(contentType string) map[string]string {
	headers := map[string]string{
		"Content-Type": contentType,
	}

	if contentType == "text/event-stream" {
		headers["Cache-Control"] = "no-cache"
		headers["Connection"] = "keep-alive"
	}

	return headers
}
