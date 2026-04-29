/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelUrlParamsString(t *testing.T) {
	tests := []struct {
		name     string
		params   *ModelUrlParams
		expected string
	}{
		{
			name: "Both model name and isolation ID present",
			params: &ModelUrlParams{
				ModelName:   "gpt-4",
				IsolationId: "tenant-42",
			},
			expected: "modelId=gpt-4",
		},
		{
			name: "Only model name present",
			params: &ModelUrlParams{
				ModelName:   "claude-3",
				IsolationId: "",
			},
			expected: "modelId=claude-3",
		},
		{
			name: "Empty model name",
			params: &ModelUrlParams{
				ModelName:   "",
				IsolationId: "tenant-42",
			},
			expected: "modelId=",
		},
		{
			name: "Both fields empty",
			params: &ModelUrlParams{
				ModelName:   "",
				IsolationId: "",
			},
			expected: "modelId=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.params.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEntityEndpointUrl(t *testing.T) {
	tests := []struct {
		name          string
		modelUrl      string
		operationPath string
		expected      string
	}{
		{
			name:          "Base URL with operation path",
			modelUrl:      "http://example.com/api",
			operationPath: "/chat/completions",
			expected:      "http://example.com/api/chat/completions",
		},
		{
			name:          "Base URL with trailing slash and operation path",
			modelUrl:      "http://example.com/api/",
			operationPath: "/chat/completions",
			expected:      "http://example.com/api//chat/completions", // Note the double slash, this is expected behavior
		},
		{
			name:          "Base URL with operation path with trailing slash",
			modelUrl:      "http://example.com/api",
			operationPath: "/chat/completions/",
			expected:      "http://example.com/api/chat/completions/",
		},
		{
			name:          "Base URL without trailing slash and operation path without leading slash",
			modelUrl:      "http://example.com/api",
			operationPath: "chat/completions",
			expected:      "http://example.com/apichat/completions", // This is the current behavior
		},
		{
			name:          "Empty operation path",
			modelUrl:      "http://example.com/api",
			operationPath: "",
			expected:      "http://example.com/api",
		},
		{
			name:          "Empty model URL",
			modelUrl:      "",
			operationPath: "/chat/completions",
			expected:      "/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEntityEndpointUrl(tt.modelUrl, tt.operationPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
