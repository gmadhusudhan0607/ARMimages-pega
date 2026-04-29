/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package registry

import (
	"fmt"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
)

func TestNewProcessorRegistry(t *testing.T) {
	registry := NewProcessorRegistry()
	if registry == nil {
		t.Fatal("NewProcessorRegistry should not return nil")
	}

	// Verify it's empty initially
	keys := registry.GetRegisteredKeys()
	if len(keys) != 0 {
		t.Errorf("Expected empty registry, got %d keys", len(keys))
	}
}

func TestProcessorRegistry_Register(t *testing.T) {
	registry := NewProcessorRegistry()

	testKey := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "0613",
	}

	tests := []struct {
		name          string
		key           ProcessorKey
		factory       ProcessorFactory
		expectedError bool
		errorContains string
	}{
		{
			name: "successful registration",
			key:  testKey,
			factory: func() interface{} {
				return extensions.NewAzureOpenAI20240201Extension()
			},
			expectedError: false,
		},
		{
			name: "duplicate registration should fail",
			key:  testKey,
			factory: func() interface{} {
				return extensions.NewAzureOpenAI20240201Extension()
			},
			expectedError: true,
			errorContains: "processor already registered",
		},
		{
			name: "invalid key should fail",
			key: ProcessorKey{
				Provider:       "", // Empty provider makes it invalid
				Infrastructure: "azure",
				Creator:        "openai",
				ModelID:        "gpt-4",
				Version:        "0613",
			},
			factory: func() interface{} {
				return extensions.NewAzureOpenAI20240201Extension()
			},
			expectedError: true,
			errorContains: "invalid processor key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.key, tt.factory)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

func TestProcessorRegistry_CreateProcessor(t *testing.T) {
	registry := NewProcessorRegistry()

	validKey := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "0613",
	}

	invalidKey := ProcessorKey{
		Provider:       "nonexistent",
		Infrastructure: "nonexistent",
		Creator:        "nonexistent",
		ModelID:        "nonexistent",
		Version:        "nonexistent",
	}

	// Register a processor
	err := registry.Register(validKey, func() interface{} {
		return extensions.NewAzureOpenAI20240201Extension()
	})
	if err != nil {
		t.Fatalf("Failed to register processor: %s", err)
	}

	tests := []struct {
		name          string
		key           ProcessorKey
		expectedError bool
		errorContains string
	}{
		{
			name:          "create existing processor",
			key:           validKey,
			expectedError: false,
		},
		{
			name:          "create non-existent processor",
			key:           invalidKey,
			expectedError: true,
			errorContains: "no processor registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := registry.CreateProcessor(tt.key)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if processor != nil {
					t.Error("Expected nil processor on error")
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %s", err.Error())
				}
				if processor == nil {
					t.Error("Expected non-nil processor")
				}
			}
		})
	}
}

func TestProcessorRegistry_HasProcessor(t *testing.T) {
	registry := NewProcessorRegistry()

	existingKey := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "0613",
	}

	nonExistentKey := ProcessorKey{
		Provider:       "nonexistent",
		Infrastructure: "nonexistent",
		Creator:        "nonexistent",
		ModelID:        "nonexistent",
		Version:        "nonexistent",
	}

	// Initially, no processor should exist
	if registry.HasProcessor(existingKey) {
		t.Error("Expected HasProcessor to return false for unregistered key")
	}

	// Register a processor
	err := registry.Register(existingKey, func() interface{} {
		return extensions.NewAzureOpenAI20240201Extension()
	})
	if err != nil {
		t.Fatalf("Failed to register processor: %s", err)
	}

	// Now it should exist
	if !registry.HasProcessor(existingKey) {
		t.Error("Expected HasProcessor to return true for registered key")
	}

	// Non-existent key should still return false
	if registry.HasProcessor(nonExistentKey) {
		t.Error("Expected HasProcessor to return false for non-existent key")
	}
}

func TestProcessorRegistry_GetRegisteredKeys(t *testing.T) {
	registry := NewProcessorRegistry()

	// Initially empty
	keys := registry.GetRegisteredKeys()
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}

	// Register some processors
	key1 := ProcessorKey{Provider: "azure", Infrastructure: "azure", Creator: "openai", ModelID: "gpt-4", Version: "0613"}
	key2 := ProcessorKey{Provider: "bedrock", Infrastructure: "aws", Creator: "anthropic", ModelID: "claude-3", Version: "v1"}

	_ = registry.Register(key1, func() interface{} { return "processor1" })
	_ = registry.Register(key2, func() interface{} { return "processor2" })

	keys = registry.GetRegisteredKeys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Check that both keys are present
	found1, found2 := false, false
	for _, key := range keys {
		if key == key1 {
			found1 = true
		}
		if key == key2 {
			found2 = true
		}
	}

	if !found1 {
		t.Error("Expected to find key1 in registered keys")
	}
	if !found2 {
		t.Error("Expected to find key2 in registered keys")
	}
}

func TestProcessorRegistry_GetSupportedCombinations(t *testing.T) {
	registry := NewProcessorRegistry()

	key := ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        "gpt-4",
		Version:        "0613",
	}

	_ = registry.Register(key, func() interface{} {
		return extensions.NewAzureOpenAI20240201Extension()
	})

	combinations := registry.GetSupportedCombinations()
	if len(combinations) != 1 {
		t.Errorf("Expected 1 combination, got %d", len(combinations))
	}

	description, exists := combinations[key]
	if !exists {
		t.Error("Expected to find registered key in combinations")
	}

	expectedDesc := fmt.Sprintf("Provider: %s, Infrastructure: %s, Creator: %s, Endpoint: %s, Version: %s",
		key.Provider, key.Infrastructure, key.Creator, "fixme", key.Version)

	if description != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, description)
	}
}

func TestInitializeProcessorRegistry(t *testing.T) {
	// Reset global registry for this test
	globalRegistry = nil

	InitializeProcessorRegistry()

	if globalRegistry == nil {
		t.Fatal("InitializeProcessorRegistry should set global registry")
	}

	// Should have registered processors
	keys := globalRegistry.GetRegisteredKeys()
	if len(keys) == 0 {
		t.Error("Expected registered processors after initialization")
	}
}

func TestGetGlobalRegistry(t *testing.T) {
	// Reset global registry
	globalRegistry = nil

	registry := GetGlobalRegistry()
	if registry == nil {
		t.Fatal("GetGlobalRegistry should not return nil")
	}

	// Should be the same instance on subsequent calls
	registry2 := GetGlobalRegistry()
	if registry != registry2 {
		t.Error("GetGlobalRegistry should return the same instance")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && s[len(s)-len(substr):] == substr) ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && len(substr) > 0 &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}
