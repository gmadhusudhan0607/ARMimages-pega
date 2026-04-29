/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package models

import (
	"context"
	"sync"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestModelKeyString(t *testing.T) {
	key := types.ModelKey{
		Infrastructure: types.InfrastructureAWS,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-35-turbo",
		Version:        "1.0",
	}

	expected := "aws/azure/openai/gpt-35-turbo/1.0"
	result := key.String()

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestIsModelSupportedRegistry(t *testing.T) {
	tests := []struct {
		name      string
		provider  types.Provider
		modelName string
		supported bool
	}{
		{
			name:      "Supported OpenAI model",
			provider:  types.ProviderAzure,
			modelName: "gpt-35-turbo",
			supported: true,
		},
		{
			name:      "Supported Amazon model",
			provider:  types.ProviderBedrock,
			modelName: "nova-lite-v1",
			supported: true,
		},
		{
			name:      "Supported Anthropic model",
			provider:  types.ProviderBedrock,
			modelName: "claude-3-haiku",
			supported: true,
		},
		{
			name:      "Supported Google model",
			provider:  types.ProviderVertex,
			modelName: "gemini-1.5-flash",
			supported: true,
		},
		{
			name:      "Supported Meta model",
			provider:  types.ProviderBedrock,
			modelName: "llama-3-2-90b-instruct",
			supported: true,
		},
		{
			name:      "Unsupported model",
			provider:  types.ProviderAzure,
			modelName: "nonexistent-model",
			supported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := GetGlobalRegistry(context.Background())
			if err != nil {
				t.Fatalf("Failed to get registry: %v", err)
			}

			// Check if any model matches provider and modelName
			allModels := reg.GetAllModels()
			result := false
			for _, model := range allModels {
				if model.Provider == tt.provider && model.Name == tt.modelName {
					result = true
					break
				}
			}

			if result != tt.supported {
				t.Errorf("Expected %v, got %v", tt.supported, result)
			}
		})
	}
}

func TestGetSupportedModelsForProviderRegistry(t *testing.T) {
	tests := []struct {
		name     string
		provider types.Provider
		minCount int // Minimum expected number of models
	}{
		{
			name:     "Azure OpenAI models",
			provider: types.ProviderAzure,
			minCount: 1,
		},
		{
			name:     "Google models",
			provider: types.ProviderVertex,
			minCount: 1,
		},
		{
			name:     "Bedrock models (includes Anthropic)",
			provider: types.ProviderBedrock,
			minCount: 1,
		},
		{
			name:     "Amazon models",
			provider: types.ProviderBedrock,
			minCount: 1,
		},
		{
			name:     "Meta models",
			provider: types.ProviderBedrock,
			minCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg, err := GetGlobalRegistry(context.Background())
			if err != nil {
				t.Fatalf("Failed to get registry: %v", err)
			}

			models, err := reg.GetModelsByProvider(tt.provider)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(models) < tt.minCount {
				t.Errorf("Expected at least %d models for provider %s, got %d",
					tt.minCount, tt.provider, len(models))
			}

			// Verify all returned models belong to the requested provider
			for _, model := range models {
				if model.Provider != tt.provider {
					t.Errorf("Expected model provider %s, got %s", tt.provider, model.Provider)
				}
			}
		})
	}
}

func TestRegistryInitialization(t *testing.T) {
	// Reset the global registry to test initialization
	globalRegistry = nil
	registryOnce = sync.Once{}
	registryError = nil

	// Call a function that should trigger initialization
	reg, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check if gpt-35-turbo is supported
	allModels := reg.GetAllModels()
	supported := false
	for _, model := range allModels {
		if model.Provider == types.ProviderAzure && model.Name == "gpt-35-turbo" {
			supported = true
			break
		}
	}

	if !supported {
		t.Error("Expected gpt-35-turbo to be supported")
	}

	// Verify registry is initialized
	if reg == nil {
		t.Error("Expected registry to be initialized")
	}
}

func TestRegistryConcurrency(t *testing.T) {
	// Test concurrent access to registry functions
	const numGoroutines = 10
	const numIterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				// Test concurrent registry access
				reg, err := GetGlobalRegistry(context.Background())
				if err != nil {
					errors <- err
					return
				}

				// Test concurrent reads using registry directly
				allModels := reg.GetAllModels()
				_ = len(allModels) // Use the result to avoid compiler warnings

				_, _ = reg.GetModelsByProvider(types.ProviderAzure)
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestRegistryModelTypes(t *testing.T) {
	reg, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	models := reg.GetAllModels()

	// Count models by functional capabilities
	capabilityCount := make(map[string]int)

	for _, model := range models {
		// Count by functional capabilities
		for _, capability := range model.FunctionalCapabilities {
			capabilityCount[string(capability)]++
		}
	}

	// Verify we have models of different functional capabilities
	expectedCapabilities := []string{"chat_completion", "embedding", "image"}
	for _, expectedCapability := range expectedCapabilities {
		if count, exists := capabilityCount[expectedCapability]; !exists || count == 0 {
			t.Errorf("Expected to find models with functional capability %s, but found %d", expectedCapability, count)
		}
	}
}

func TestRegistryModelCapabilities(t *testing.T) {
	reg, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	models := reg.GetAllModels()

	for _, model := range models {
		// Test that models have appropriate capabilities
		if len(model.Capabilities.InputModalities) == 0 {
			t.Errorf("Model %s has no input modalities defined", model.KEY)
		}

		if len(model.Capabilities.OutputModalities) == 0 {
			t.Errorf("Model %s has no output modalities defined", model.KEY)
		}

		// Chat completion models should support text input
		if model.HasCapability(types.FunctionalCapabilityChatCompletion) {
			hasTextInput := false
			for _, modality := range model.Capabilities.InputModalities {
				if modality == "text" {
					hasTextInput = true
					break
				}
			}
			if !hasTextInput {
				t.Errorf("Chat completion model %s should support text input", model.KEY)
			}
		}

		// Embedding models should have embedding output
		if model.HasCapability(types.FunctionalCapabilityEmbedding) {
			hasEmbeddingOutput := false
			for _, modality := range model.Capabilities.OutputModalities {
				if modality == "embedding" {
					hasEmbeddingOutput = true
					break
				}
			}
			if !hasEmbeddingOutput {
				t.Errorf("Embedding model %s should support embedding output", model.KEY)
			}
		}

		// Image models should have image output
		if model.HasCapability(types.FunctionalCapabilityImage) {
			hasImageOutput := false
			for _, modality := range model.Capabilities.OutputModalities {
				if modality == "image" {
					hasImageOutput = true
					break
				}
			}
			if !hasImageOutput {
				t.Errorf("Image model %s should support image output", model.KEY)
			}
		}
	}
}
