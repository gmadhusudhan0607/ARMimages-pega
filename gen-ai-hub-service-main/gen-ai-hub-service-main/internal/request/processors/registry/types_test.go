/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package registry

import (
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestProcessorKey_String(t *testing.T) {
	key := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "0613",
	}

	expected := "azure/azure/openai/gpt-4/0613"
	result := key.String()

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestProcessorKey_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		key      ProcessorKey
		expected bool
	}{
		{
			name: "valid key with all fields",
			key: ProcessorKey{
				Provider:       "azure",
				Infrastructure: "azure",
				Creator:        "openai",
				ModelID:        "gpt-4",
				Version:        "0613",
			},
			expected: true,
		},
		{
			name: "valid key with minimal fields",
			key: ProcessorKey{
				Provider: "azure",
			},
			expected: true,
		},
		{
			name: "invalid key with empty provider",
			key: ProcessorKey{
				Provider:       "",
				Infrastructure: "azure",
				Creator:        "openai",
				ModelID:        "gpt-4",
				Version:        "0613",
			},
			expected: false,
		},
		{
			name:     "invalid empty key",
			key:      ProcessorKey{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.key.IsValid()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCreateProcessorKey(t *testing.T) {
	tests := []struct {
		name     string
		model    *types.Model
		expected ProcessorKey
	}{
		{
			name: "azure openai model",
			model: &types.Model{
				Provider:       "azure",
				Infrastructure: "azure",
				Creator:        "openai",
				Name:           "gpt-4",
				Version:        "0613",
			},
			expected: ProcessorKey{
				Provider:       "azure",
				Infrastructure: "azure",
				Creator:        "openai",
				ModelID:        "gpt-4",
				Version:        "0613",
			},
		},
		{
			name: "bedrock anthropic model",
			model: &types.Model{
				Provider:       "bedrock",
				Infrastructure: "aws",
				Creator:        "anthropic",
				Name:           "claude-3-haiku",
				Version:        "v1",
			},
			expected: ProcessorKey{
				Provider:       "bedrock",
				Infrastructure: "aws",
				Creator:        "anthropic",
				ModelID:        "claude-3-haiku",
				Version:        "v1",
			},
		},
		{
			name: "vertex google model",
			model: &types.Model{
				Provider:       "vertex",
				Infrastructure: "gcp",
				Creator:        "google",
				Name:           "gemini-1.5-pro",
				Version:        "002",
			},
			expected: ProcessorKey{
				Provider:       "vertex",
				Infrastructure: "gcp",
				Creator:        "google",
				ModelID:        "gemini-1.5-pro",
				Version:        "002",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateProcessorKey(tt.model)

			if result.Provider != tt.expected.Provider {
				t.Errorf("Expected Provider %s, got %s", tt.expected.Provider, result.Provider)
			}
			if result.Infrastructure != tt.expected.Infrastructure {
				t.Errorf("Expected Infrastructure %s, got %s", tt.expected.Infrastructure, result.Infrastructure)
			}
			if result.Creator != tt.expected.Creator {
				t.Errorf("Expected Creator %s, got %s", tt.expected.Creator, result.Creator)
			}
			if result.ModelID != tt.expected.ModelID {
				t.Errorf("Expected ModelID %s, got %s", tt.expected.ModelID, result.ModelID)
			}
			if result.Version != tt.expected.Version {
				t.Errorf("Expected Version %s, got %s", tt.expected.Version, result.Version)
			}
		})
	}
}

func TestProcessorKey_Equality(t *testing.T) {
	key1 := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "0613",
	}

	key2 := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "0613",
	}

	key3 := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "1106", // Different version
	}

	if key1 != key2 {
		t.Error("Expected identical keys to be equal")
	}

	if key1 == key3 {
		t.Error("Expected different keys to be unequal")
	}
}

func TestProcessorKey_AsMapKey(t *testing.T) {
	// Test that ProcessorKey can be used as a map key
	registry := make(map[ProcessorKey]string)

	key1 := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "0613",
	}

	key2 := ProcessorKey{
		Provider:       "bedrock",
		Infrastructure: "aws",
		Creator:        "anthropic",
		ModelID:        "claude-3",
		Version:        "v1",
	}

	registry[key1] = "processor1"
	registry[key2] = "processor2"

	if len(registry) != 2 {
		t.Errorf("Expected map to have 2 entries, got %d", len(registry))
	}

	if registry[key1] != "processor1" {
		t.Error("Expected to retrieve processor1 for key1")
	}

	if registry[key2] != "processor2" {
		t.Error("Expected to retrieve processor2 for key2")
	}

	// Test duplicate key overwrites
	registry[key1] = "updated_processor1"
	if registry[key1] != "updated_processor1" {
		t.Error("Expected map value to be updated")
	}

	if len(registry) != 2 {
		t.Error("Expected map size to remain 2 after update")
	}
}
