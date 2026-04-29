/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package models

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/loader"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestIsModelSupported(t *testing.T) {
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
			name:      "Unsupported model",
			provider:  types.ProviderGoogle,
			modelName: "unsupported-model",
			supported: false,
		},
		// Test new Amazon Nova models (available through AWS Bedrock)
		{
			name:      "Amazon Nova Lite",
			provider:  types.ProviderBedrock,
			modelName: "nova-lite-v1",
			supported: true,
		},
		{
			name:      "Amazon Nova Micro",
			provider:  types.ProviderBedrock,
			modelName: "nova-micro",
			supported: true,
		},
		{
			name:      "Amazon Nova Premier",
			provider:  types.ProviderBedrock,
			modelName: "nova-premier",
			supported: true,
		},
		{
			name:      "Amazon Nova Pro",
			provider:  types.ProviderBedrock,
			modelName: "nova-pro",
			supported: true,
		},
		// Test Anthropic Claude models (available through AWS Bedrock)
		{
			name:      "Anthropic Claude 3 Haiku",
			provider:  types.ProviderBedrock,
			modelName: "claude-3-haiku",
			supported: true,
		},
		{
			name:      "Anthropic Claude 3.5 Sonnet",
			provider:  types.ProviderBedrock,
			modelName: "claude-3-5-sonnet",
			supported: true,
		},
		{
			name:      "Anthropic Claude 3.5 Haiku",
			provider:  types.ProviderBedrock,
			modelName: "claude-3-5-haiku",
			supported: true,
		},
		// Test Google Gemini models (available through GCP Vertex AI)
		{
			name:      "Google Gemini 1.5 Flash",
			provider:  types.ProviderVertex,
			modelName: "gemini-1.5-flash",
			supported: true,
		},
		{
			name:      "Google Gemini 1.5 Pro",
			provider:  types.ProviderVertex,
			modelName: "gemini-1.5-pro",
			supported: true,
		},
		{
			name:      "Google Gemini 1.0 Pro",
			provider:  types.ProviderVertex,
			modelName: "gemini-1.0-pro",
			supported: true,
		},
		// Test new Google Gemini 2.5 models (available through GCP Vertex AI)
		{
			name:      "Google Gemini 2.5 Pro",
			provider:  types.ProviderVertex,
			modelName: "gemini-2.5-pro",
			supported: true,
		},
		{
			name:      "Google Gemini 2.5 Flash",
			provider:  types.ProviderVertex,
			modelName: "gemini-2.5-flash",
			supported: true,
		},
		{
			name:      "Google Gemini 2.5 Flash-Lite",
			provider:  types.ProviderVertex,
			modelName: "gemini-2.5-flash-lite",
			supported: true,
		},
		// Test Meta Llama models (available through AWS Bedrock)
		{
			name:      "Meta Llama 3.2 90B Instruct",
			provider:  types.ProviderBedrock,
			modelName: "llama-3-2-90b-instruct",
			supported: true,
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

// TestModelKeyUniqueness validates that there are no duplicate model IDs within the same infrastructure
// Note: The same model ID can appear across different infrastructures (this is expected behavior)
func TestModelKeyUniqueness(t *testing.T) {
	// Use the new ModelLoader
	modelLoader := loader.NewModelLoader()

	registry, err := modelLoader.LoadModelsIntoRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to load configurations: %v", err)
	}

	// Group models by infrastructure to check for duplicates within each infrastructure
	infrastructureModels := make(map[string]map[string][]string) // infrastructure -> modelName -> list of sources
	duplicates := make(map[string]map[string][]string)           // infrastructure -> modelName -> list of sources

	// Get all models from the registry
	allModels := registry.GetAllModels()

	for _, model := range allModels {
		modelKey := model.KEY
		if modelKey == "" {
			// Skip models without explicit modelKey
			continue
		}

		infrastructure := string(model.Infrastructure)
		source := fmt.Sprintf("provider: %s, creator: %s, model: %s, version: %s",
			model.Provider, model.Creator, model.Name, model.Version)

		// Initialize infrastructure map if needed
		if infrastructureModels[infrastructure] == nil {
			infrastructureModels[infrastructure] = make(map[string][]string)
		}

		if existing, exists := infrastructureModels[infrastructure][modelKey]; exists {
			// This is a duplicate within the same infrastructure
			if duplicates[infrastructure] == nil {
				duplicates[infrastructure] = make(map[string][]string)
			}
			if _, isDuplicate := duplicates[infrastructure][modelKey]; !isDuplicate {
				// First time we see this as duplicate, add the original source too
				duplicates[infrastructure][modelKey] = append(existing, source)
			} else {
				// Already marked as duplicate, just add this source
				duplicates[infrastructure][modelKey] = append(duplicates[infrastructure][modelKey], source)
			}
		} else {
			// First occurrence within this infrastructure
			infrastructureModels[infrastructure][modelKey] = []string{source}
		}
	}

	// Report any duplicates found within the same infrastructure
	if len(duplicates) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("Found duplicate model IDs within the same infrastructure:\n")

		for infrastructure, infraDuplicates := range duplicates {
			errorMsg.WriteString(fmt.Sprintf("\nInfrastructure '%s':\n", infrastructure))
			for modelKey, sources := range infraDuplicates {
				errorMsg.WriteString(fmt.Sprintf("  Model ID '%s' appears in:\n", modelKey))
				for _, source := range sources {
					errorMsg.WriteString(fmt.Sprintf("    - %s\n", source))
				}
			}
		}

		t.Errorf("%s", errorMsg.String())
	}
}

// TestProviderModelVersionUniqueness validates that there are no duplicate provider/model-name/model-version combinations
func TestProviderModelVersionUniqueness(t *testing.T) {
	// Use the new ModelLoader
	modelLoader := loader.NewModelLoader()

	registry, err := modelLoader.LoadModelsIntoRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to load configurations: %v", err)
	}

	combinations := make(map[string][]string) // combination -> list of sources
	duplicates := make(map[string][]string)

	// Get all models from the registry
	allModels := registry.GetAllModels()

	for _, model := range allModels {
		// Create combination key: infrastructure/provider/creator/model-name/model-version
		combination := fmt.Sprintf("%s/%s/%s/%s/%s", model.Infrastructure, model.Provider, model.Creator, model.Name, model.Version)

		source := fmt.Sprintf("model key: %s", model.KEY)

		if existing, exists := combinations[combination]; exists {
			// This is a duplicate
			if _, isDuplicate := duplicates[combination]; !isDuplicate {
				// First time we see this as duplicate, add the original source too
				duplicates[combination] = append(existing, source)
			} else {
				// Already marked as duplicate, just add this source
				duplicates[combination] = append(duplicates[combination], source)
			}
		} else {
			// First occurrence
			combinations[combination] = []string{source}
		}
	}

	// Report any duplicates found
	if len(duplicates) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("Found duplicate infrastructure/provider/creator/model-name/model-version combinations:\n")

		for combination, sources := range duplicates {
			parts := strings.Split(combination, "/")
			if len(parts) == 5 {
				errorMsg.WriteString(fmt.Sprintf("\nCombination 'infrastructure: %s, provider: %s, creator: %s, model: %s, version: %s' appears in:\n",
					parts[0], parts[1], parts[2], parts[3], parts[4]))
			} else {
				errorMsg.WriteString(fmt.Sprintf("\nCombination '%s' appears in:\n", combination))
			}
			for _, source := range sources {
				errorMsg.WriteString(fmt.Sprintf("  - %s\n", source))
			}
		}

		t.Errorf("%s", errorMsg.String())
	}
}
