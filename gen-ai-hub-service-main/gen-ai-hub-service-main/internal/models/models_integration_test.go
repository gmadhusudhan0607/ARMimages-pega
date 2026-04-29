/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package models

import (
	"context"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestGlobalRegistryInitialization(t *testing.T) {
	// Test the global registry initialization using CompositeLoader
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	if registry == nil {
		t.Fatal("Expected registry to be initialized")
	}

	modelCount := registry.GetModelCount()
	if modelCount == 0 {
		t.Error("Expected at least one model to be loaded")
	}
}

func TestGlobalRegistryValidateConfigs(t *testing.T) {
	// Test that the CompositeLoader validates configurations properly
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	// Get all models and verify they have required fields
	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Error("Expected at least one model to be loaded")
	}

	for _, model := range allModels {
		if model.Name == "" {
			t.Error("Model has empty name")
		}
		if model.Version == "" {
			t.Error("Model has empty version")
		}
		if model.Infrastructure == "" {
			t.Error("Model has empty infrastructure")
		}
		if model.Provider == "" {
			t.Error("Model has empty provider")
		}
		if model.Creator == "" {
			t.Error("Model has empty creator")
		}
	}
}

func TestGlobalRegistryOpenAIModels(t *testing.T) {
	// Test OpenAI models using CompositeLoader
	// OpenAI models are configured with provider: openai and creator: OpenAI
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	// Get models by provider openai since OpenAI models have provider=openai, creator=openai
	openAIModels, err := registry.GetModelsByProvider(types.ProviderAzure)
	if err != nil {
		t.Fatalf("Failed to get OpenAI models: %v", err)
	}

	if len(openAIModels) == 0 {
		t.Error("Expected at least one OpenAI model")
	}

	for _, model := range openAIModels {
		if model.Provider != types.ProviderAzure {
			t.Errorf("Expected Azure provider, got %s", model.Provider)
		}
		if model.Creator != types.CreatorOpenAI {
			t.Errorf("Expected OpenAI creator, got %s", model.Creator)
		}
	}
}

func TestGlobalRegistryAmazonModels(t *testing.T) {
	// Test Amazon models using CompositeLoader
	// Amazon models are configured with provider: bedrock and creator: amazon
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	// Get models by creator since Amazon models have provider=bedrock, creator=amazon
	amazonModels, err := registry.GetModelsByCreator(types.CreatorAmazon)
	if err != nil {
		t.Fatalf("Failed to get Amazon models: %v", err)
	}

	if len(amazonModels) == 0 {
		t.Error("Expected at least one Amazon model")
	}

	for _, model := range amazonModels {
		if model.Creator != types.CreatorAmazon {
			t.Errorf("Expected Amazon creator, got %s", model.Creator)
		}
	}
}

func TestGlobalRegistryAnthropicModels(t *testing.T) {
	// Test Anthropic models using CompositeLoader
	// Anthropic models are configured with creator: anthropic (provider varies by infrastructure)
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	anthropicModels, err := registry.GetModelsByCreator(types.CreatorAnthropic)
	if err != nil {
		t.Fatalf("Failed to get Anthropic models: %v", err)
	}

	if len(anthropicModels) == 0 {
		t.Error("Expected at least one Anthropic model")
	}

	for _, model := range anthropicModels {
		if model.Creator != types.CreatorAnthropic {
			t.Errorf("Expected Anthropic creator, got %s", model.Creator)
		}
	}
}

func TestGlobalRegistryGoogleModels(t *testing.T) {
	// Test Google models using CompositeLoader
	// Google models are configured with provider: vertex and creator: google
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	// Get models by creator since Google models have provider=vertex, creator=google
	googleModels, err := registry.GetModelsByCreator(types.CreatorGoogle)
	if err != nil {
		t.Fatalf("Failed to get Google models: %v", err)
	}

	if len(googleModels) == 0 {
		t.Error("Expected at least one Google model")
	}

	for _, model := range googleModels {
		if model.Creator != types.CreatorGoogle {
			t.Errorf("Expected Google creator, got %s", model.Creator)
		}
	}
}

func TestGlobalRegistryMetaModels(t *testing.T) {
	// Test Meta models using CompositeLoader
	// Meta models are configured with provider: bedrock and creator: meta
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	// Get models by creator since Meta models have provider=bedrock, creator=meta
	metaModels, err := registry.GetModelsByCreator(types.CreatorMeta)
	if err != nil {
		t.Fatalf("Failed to get Meta models: %v", err)
	}

	if len(metaModels) == 0 {
		t.Error("Expected at least one Meta model")
	}

	for _, model := range metaModels {
		if model.Creator != types.CreatorMeta {
			t.Errorf("Expected Meta creator, got %s", model.Creator)
		}
	}
}

func TestGlobalRegistryModelTypes(t *testing.T) {
	// Test model types using CompositeLoader
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Error("Expected at least one model")
	}

	// Verify models have valid functional capabilities
	for _, model := range allModels {
		// Models should have functional capabilities
		if len(model.FunctionalCapabilities) == 0 {
			t.Errorf("Model %s has no functional capabilities", model.Name)
		}

		// Check functional capabilities
		validCapabilities := []types.FunctionalCapability{
			types.FunctionalCapabilityChatCompletion,
			types.FunctionalCapabilityEmbedding,
			types.FunctionalCapabilityImage,
			types.FunctionalCapabilityRealtime,
		}
		for _, capability := range model.FunctionalCapabilities {
			isValidCapability := false
			for _, validCapability := range validCapabilities {
				if capability == validCapability {
					isValidCapability = true
					break
				}
			}
			if !isValidCapability {
				t.Errorf("Model %s has invalid functional capability: %s", model.Name, capability)
			}
		}
	}
}

func TestGlobalRegistryModelCapabilities(t *testing.T) {
	// Test model capabilities using CompositeLoader
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Error("Expected at least one model")
	}

	// Verify models have capabilities defined
	for _, model := range allModels {
		// Check that capabilities have some features or modalities defined
		if len(model.Capabilities.Features) == 0 &&
			len(model.Capabilities.InputModalities) == 0 &&
			len(model.Capabilities.OutputModalities) == 0 {
			t.Logf("Model %s has no capabilities defined (may be valid)", model.Name)
		}

		// Verify that if modalities are defined, they are valid
		for _, modality := range model.Capabilities.InputModalities {
			if modality == "" {
				t.Errorf("Model %s has empty input modality", model.Name)
			}
		}
		for _, modality := range model.Capabilities.OutputModalities {
			if modality == "" {
				t.Errorf("Model %s has empty output modality", model.Name)
			}
		}
	}
}

func TestGlobalRegistryModelParameters(t *testing.T) {
	// Test model parameters using CompositeLoader
	registry, err := GetGlobalRegistry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get registry: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Error("Expected at least one model")
	}

	// Verify models have parameters defined
	for _, model := range allModels {
		if len(model.Parameters) == 0 {
			t.Errorf("Model %s has no parameters defined", model.Name)
			continue
		}

		// Check parameter specifications
		for paramName, paramSpec := range model.Parameters {
			if paramName == "" {
				t.Errorf("Model %s has parameter with empty name", model.Name)
			}
			if paramSpec.Type == "" {
				t.Errorf("Model %s parameter %s has empty type", model.Name, paramName)
			}
		}
	}
}
