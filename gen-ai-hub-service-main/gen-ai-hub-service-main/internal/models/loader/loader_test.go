/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package loader

import (
	"context"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestNewModelLoader(t *testing.T) {
	loader := NewModelLoader()
	if loader == nil {
		t.Fatal("NewModelLoader() returned nil")
	} else if loader.embedFS == nil {
		t.Fatal("ModelLoader has nil embedFS")
	}
}

func TestModelLoader_LoadModelsIntoRegistry(t *testing.T) {
	loader := NewModelLoader()

	registry, err := loader.LoadModelsIntoRegistry(context.Background())
	if err != nil {
		t.Fatalf("LoadModelsIntoRegistry() failed: %v", err)
	}

	if registry == nil {
		t.Fatal("LoadModelsIntoRegistry() returned nil registry")
	}

	// Verify models were loaded
	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Fatal("No models were loaded")
	}

	// Verify we have models from different infrastructures
	infrastructures := make(map[string]bool)
	for _, model := range allModels {
		infrastructures[string(model.Infrastructure)] = true
	}

	if len(infrastructures) == 0 {
		t.Fatal("No infrastructure models found")
	}

	t.Logf("Loaded %d models across %d infrastructures", len(allModels), len(infrastructures))
}

func TestModelLoader_LoadRegistryConsistency(t *testing.T) {
	loader := NewModelLoader()

	// Load registry multiple times to ensure consistency
	registry1, err1 := loader.LoadModelsIntoRegistry(context.Background())
	if err1 != nil {
		t.Fatalf("First LoadModelsIntoRegistry() failed: %v", err1)
	}

	registry2, err2 := loader.LoadModelsIntoRegistry(context.Background())
	if err2 != nil {
		t.Fatalf("Second LoadModelsIntoRegistry() failed: %v", err2)
	}

	models1 := registry1.GetAllModels()
	models2 := registry2.GetAllModels()

	if len(models1) != len(models2) {
		t.Fatalf("Inconsistent model count: first load=%d, second load=%d", len(models1), len(models2))
	}
}

func TestModelLoader_ValidateModelStructure(t *testing.T) {
	loader := NewModelLoader()

	registry, err := loader.LoadModelsIntoRegistry(context.Background())
	if err != nil {
		t.Fatalf("LoadModelsIntoRegistry() failed: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Skip("No models to validate")
	}

	// Validate that all models have required fields
	for _, model := range allModels {
		if model.Name == "" {
			t.Errorf("Model has empty name: %+v", model)
		}
		if model.Version == "" {
			t.Errorf("Model %s has empty version", model.Name)
		}
		if model.KEY == "" {
			t.Errorf("Model %s has empty ID", model.Name)
		}
		if model.Infrastructure == "" {
			t.Errorf("Model %s has empty infrastructure", model.Name)
		}
		if model.Provider == "" {
			t.Errorf("Model %s has empty provider", model.Name)
		}
		if model.Creator == "" {
			t.Errorf("Model %s has empty creator", model.Name)
		}
		if len(model.Endpoints) == 0 {
			t.Errorf("Model %s has no endpoints", model.Name)
		}
	}
}

func TestModelLoader_InfrastructureSupport(t *testing.T) {
	loader := NewModelLoader()

	registry, err := loader.LoadModelsIntoRegistry(context.Background())
	if err != nil {
		t.Fatalf("LoadModelsIntoRegistry() failed: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Skip("No models to check")
	}

	// Check that we support the expected infrastructures
	expectedInfrastructures := []types.Infrastructure{
		types.InfrastructureAWS,
		types.InfrastructureGCP,
		types.InfrastructureAzure,
	}

	foundInfrastructures := make(map[types.Infrastructure]bool)
	for _, model := range allModels {
		foundInfrastructures[model.Infrastructure] = true
	}

	for _, expected := range expectedInfrastructures {
		if !foundInfrastructures[expected] {
			t.Logf("Warning: No models found for infrastructure %s", expected)
		}
	}

	if len(foundInfrastructures) == 0 {
		t.Fatal("No infrastructure models found")
	}
}

func TestModelLoader_ProviderSupport(t *testing.T) {
	loader := NewModelLoader()

	registry, err := loader.LoadModelsIntoRegistry(context.Background())
	if err != nil {
		t.Fatalf("LoadModelsIntoRegistry() failed: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Skip("No models to check")
	}

	// Count models by provider
	providerCounts := make(map[types.Provider]int)
	for _, model := range allModels {
		providerCounts[model.Provider]++
	}

	if len(providerCounts) == 0 {
		t.Fatal("No provider models found")
	}

	t.Logf("Found models for %d providers:", len(providerCounts))
	for provider, count := range providerCounts {
		t.Logf("  %s: %d models", provider, count)
	}
}

func TestModelLoader_CreatorSupport(t *testing.T) {
	loader := NewModelLoader()

	registry, err := loader.LoadModelsIntoRegistry(context.Background())
	if err != nil {
		t.Fatalf("LoadModelsIntoRegistry() failed: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Skip("No models to check")
	}

	// Count models by creator
	creatorCounts := make(map[types.Creator]int)
	for _, model := range allModels {
		creatorCounts[model.Creator]++
	}

	if len(creatorCounts) == 0 {
		t.Fatal("No creator models found")
	}

	t.Logf("Found models for %d creators:", len(creatorCounts))
	for creator, count := range creatorCounts {
		t.Logf("  %s: %d models", creator, count)
	}
}

func TestModelLoader_EndpointValidation(t *testing.T) {
	loader := NewModelLoader()

	registry, err := loader.LoadModelsIntoRegistry(context.Background())
	if err != nil {
		t.Fatalf("LoadModelsIntoRegistry() failed: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) == 0 {
		t.Skip("No models to validate")
	}

	// Validate that all models have valid endpoints
	for _, model := range allModels {
		if len(model.Endpoints) == 0 {
			t.Errorf("Model %s has no endpoints", model.Name)
			continue
		}

		for i, endpoint := range model.Endpoints {
			if endpoint == "" {
				t.Errorf("Model %s endpoint %d is empty", model.Name, i)
			}
		}
	}
}
