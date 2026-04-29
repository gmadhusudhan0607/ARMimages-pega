/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package registry

import (
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

func TestModelKeyString(t *testing.T) {
	key := types.ModelKey{
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-35-turbo",
		Version:        "1106",
	}

	expected := "azure/azure/openai/gpt-35-turbo/1106"
	result := key.String()

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created")
	} else if registry.models == nil {
		t.Error("Expected models map to be initialized")
	} else {
		if registry.indexes.byInfrastructure == nil {
			t.Error("Expected byInfrastructure index to be initialized")
		}
		if registry.indexes.byProvider == nil {
			t.Error("Expected byProvider index to be initialized")
		}
	}
}

func TestLoadFromConfig(t *testing.T) {
	// Test loading from configuration using Registry
	registry := NewRegistry()

	// Create a test model
	testModel := &types.Model{
		Name:                   "test-model",
		Version:                "1.0",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(testModel)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	// Verify model was registered
	if registry.GetModelCount() != 1 {
		t.Errorf("Expected 1 model, got %d", registry.GetModelCount())
	}
}

func TestFindModelExactMatch(t *testing.T) {
	// Test exact model matching using Registry
	registry := NewRegistry()

	// Create a test model
	testModel := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(testModel)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	// Test exact match
	key := types.ModelKey{
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4",
		Version:        "0125",
	}

	foundModel, err := registry.FindModel(
		key.Infrastructure,
		key.Provider,
		key.Creator,
		key.ModelName,
		key.Version,
	)
	if err != nil {
		t.Fatalf("Failed to find model: %v", err)
	}

	if foundModel.Name != testModel.Name {
		t.Errorf("Expected model name %s, got %s", testModel.Name, foundModel.Name)
	}
}

func TestFindModelLatestVersion(t *testing.T) {
	// Test latest version resolution using Registry
	registry := NewRegistry()

	// Create multiple versions of the same model
	model1 := &types.Model{
		Name:                   "gpt-4",
		Version:                "1106",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	model2 := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(model1)
	if err != nil {
		t.Fatalf("Failed to register model1: %v", err)
	}
	err = registry.RegisterModel(model2)
	if err != nil {
		t.Fatalf("Failed to register model2: %v", err)
	}

	// Test latest version lookup using explicit FindLatestModel function
	foundModel, err := registry.FindLatestModel(
		types.InfrastructureAzure,
		types.ProviderAzure,
		types.CreatorOpenAI,
		"gpt-4",
	)
	if err != nil {
		t.Fatalf("Failed to find latest model: %v", err)
	}

	// Should return the newer version (0125 is newer than 1106 for GPT models)
	if foundModel.Version != "0125" {
		t.Errorf("Expected latest version 0125, got %s", foundModel.Version)
	}
}

func TestGetAllModels(t *testing.T) {
	// Test getting all models using Registry
	registry := NewRegistry()

	// Create test models
	model1 := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	model2 := &types.Model{
		Name:                   "claude-3",
		Version:                "1.0",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAnthropic,
		Creator:                types.CreatorAnthropic,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{"https://api.anthropic.com/v1/messages"},
	}

	err := registry.RegisterModel(model1)
	if err != nil {
		t.Fatalf("Failed to register model1: %v", err)
	}
	err = registry.RegisterModel(model2)
	if err != nil {
		t.Fatalf("Failed to register model2: %v", err)
	}

	allModels := registry.GetAllModels()
	if len(allModels) != 2 {
		t.Errorf("Expected 2 models, got %d", len(allModels))
	}
}

func TestGetSupportedModelsForProvider(t *testing.T) {
	// Test getting models by provider using Registry
	registry := NewRegistry()

	// Create test models for different providers
	azureModel := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	anthropicModel := &types.Model{
		Name:                   "claude-3",
		Version:                "1.0",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAnthropic,
		Creator:                types.CreatorAnthropic,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{"https://api.anthropic.com/v1/messages"},
	}

	bedrockModel := &types.Model{
		Name:                   "titan-text",
		Version:                "1.0",
		Infrastructure:         types.InfrastructureAWS,
		Provider:               types.ProviderBedrock,
		Creator:                types.CreatorAmazon,
		FunctionalCapabilities: []types.FunctionalCapability{"text_generation"},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{"https://bedrock.amazonaws.com/invoke"},
	}

	// Register all models
	err := registry.RegisterModel(azureModel)
	if err != nil {
		t.Fatalf("Failed to register Azure model: %v", err)
	}
	err = registry.RegisterModel(anthropicModel)
	if err != nil {
		t.Fatalf("Failed to register Anthropic model: %v", err)
	}
	err = registry.RegisterModel(bedrockModel)
	if err != nil {
		t.Fatalf("Failed to register Bedrock model: %v", err)
	}

	// Build indexes for enhanced registry
	err = registry.RebuildIndexes()
	if err != nil {
		t.Fatalf("Failed to build indexes: %v", err)
	}

	// Test getting Azure provider models
	azureModels, err := registry.GetModelsByProvider(types.ProviderAzure)
	if err != nil {
		t.Fatalf("Failed to get Azure models: %v", err)
	}

	if len(azureModels) != 1 {
		t.Errorf("Expected 1 Azure model, got %d", len(azureModels))
	}

	if azureModels[0].Name != "gpt-4" {
		t.Errorf("Expected Azure model name 'gpt-4', got '%s'", azureModels[0].Name)
	}

	if azureModels[0].Provider != types.ProviderAzure {
		t.Errorf("Expected Azure provider, got %s", azureModels[0].Provider)
	}

	// Test getting Anthropic provider models
	anthropicModels, err := registry.GetModelsByProvider(types.ProviderAnthropic)
	if err != nil {
		t.Fatalf("Failed to get Anthropic models: %v", err)
	}

	if len(anthropicModels) != 1 {
		t.Errorf("Expected 1 Anthropic model, got %d", len(anthropicModels))
	}

	if anthropicModels[0].Name != "claude-3" {
		t.Errorf("Expected Anthropic model name 'claude-3', got '%s'", anthropicModels[0].Name)
	}

	if anthropicModels[0].Provider != types.ProviderAnthropic {
		t.Errorf("Expected Anthropic provider, got %s", anthropicModels[0].Provider)
	}

	// Test getting Bedrock provider models
	bedrockModels, err := registry.GetModelsByProvider(types.ProviderBedrock)
	if err != nil {
		t.Fatalf("Failed to get Bedrock models: %v", err)
	}

	if len(bedrockModels) != 1 {
		t.Errorf("Expected 1 Bedrock model, got %d", len(bedrockModels))
	}

	if bedrockModels[0].Name != "titan-text" {
		t.Errorf("Expected Bedrock model name 'titan-text', got '%s'", bedrockModels[0].Name)
	}

	if bedrockModels[0].Provider != types.ProviderBedrock {
		t.Errorf("Expected Bedrock provider, got %s", bedrockModels[0].Provider)
	}

	// Test getting models for non-existent provider
	_, err = registry.GetModelsByProvider(types.ProviderGoogle)
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}

	// Test with multiple models for same provider
	azureModel2 := &types.Model{
		Name:                   "gpt-3.5-turbo",
		Version:                "1106",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err = registry.RegisterModel(azureModel2)
	if err != nil {
		t.Fatalf("Failed to register second Azure model: %v", err)
	}

	// Rebuild indexes
	err = registry.RebuildIndexes()
	if err != nil {
		t.Fatalf("Failed to rebuild indexes: %v", err)
	}

	// Test getting Azure models again - should now have 2
	azureModelsUpdated, err := registry.GetModelsByProvider(types.ProviderAzure)
	if err != nil {
		t.Fatalf("Failed to get updated Azure models: %v", err)
	}

	if len(azureModelsUpdated) != 2 {
		t.Errorf("Expected 2 Azure models, got %d", len(azureModelsUpdated))
	}

	// Verify both models are present
	modelNames := make(map[string]bool)
	for _, model := range azureModelsUpdated {
		modelNames[model.Name] = true
		if model.Provider != types.ProviderAzure {
			t.Errorf("Expected Azure provider for all models, got %s", model.Provider)
		}
	}

	if !modelNames["gpt-4"] {
		t.Error("Expected to find gpt-4 model in Azure models")
	}
	if !modelNames["gpt-3.5-turbo"] {
		t.Error("Expected to find gpt-3.5-turbo model in Azure models")
	}
}

func TestIsModelSupported(t *testing.T) {
	// Test model support checking using Registry
	registry := NewRegistry()

	// Create a test model
	testModel := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(testModel)
	if err != nil {
		t.Fatalf("Failed to register test model: %v", err)
	}

	// Test supported model
	key := types.ModelKey{
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4",
		Version:        "0125",
	}

	_, err = registry.FindModel(
		key.Infrastructure,
		key.Provider,
		key.Creator,
		key.ModelName,
		key.Version,
	)
	if err != nil {
		t.Errorf("Expected model to be supported, but got error: %v", err)
	}

	// Test unsupported model
	unsupportedKey := types.ModelKey{
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "nonexistent-model",
		Version:        "1.0",
	}

	_, err = registry.FindModel(
		unsupportedKey.Infrastructure,
		unsupportedKey.Provider,
		unsupportedKey.Creator,
		unsupportedKey.ModelName,
		unsupportedKey.Version,
	)
	if err == nil {
		t.Error("Expected error for unsupported model")
	}
}

func TestGetAvailableVersions(t *testing.T) {
	// Test getting available versions using Registry
	registry := NewRegistry()

	// Create multiple versions of the same model
	model1 := &types.Model{
		Name:                   "gpt-4",
		Version:                "1106",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	model2 := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(model1)
	if err != nil {
		t.Fatalf("Failed to register model1: %v", err)
	}
	err = registry.RegisterModel(model2)
	if err != nil {
		t.Fatalf("Failed to register model2: %v", err)
	}

	// Get all models and check versions
	allModels := registry.GetAllModels()
	versions := make(map[string]bool)
	for _, model := range allModels {
		if model.Name == "gpt-4" {
			versions[model.Version] = true
		}
	}

	if !versions["1106"] || !versions["0125"] {
		t.Error("Expected both versions 1106 and 0125 to be available")
	}
}

func TestGetLatestVersion(t *testing.T) {
	// Test getting latest version using Registry
	registry := NewRegistry()

	// Create multiple versions of the same model
	model1 := &types.Model{
		Name:                   "gpt-4",
		Version:                "1106",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	model2 := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(model1)
	if err != nil {
		t.Fatalf("Failed to register model1: %v", err)
	}
	err = registry.RegisterModel(model2)
	if err != nil {
		t.Fatalf("Failed to register model2: %v", err)
	}

	// Test latest version lookup using explicit FindLatestModel function
	latestModel, err := registry.FindLatestModel(
		types.InfrastructureAzure,
		types.ProviderAzure,
		types.CreatorOpenAI,
		"gpt-4",
	)
	if err != nil {
		t.Fatalf("Failed to find latest model: %v", err)
	}

	// Should return the newer version
	if latestModel.Version != "0125" {
		t.Errorf("Expected latest version 0125, got %s", latestModel.Version)
	}
}

func TestVersionSorting(t *testing.T) {
	// Test version sorting logic using Registry
	registry := NewRegistry()

	// Create models with different version formats (using valid GPT versions)
	versions := []string{"1106", "0125", "0613", "0301"}
	for _, version := range versions {
		model := &types.Model{
			Name:                   "gpt-4",
			Version:                version,
			Infrastructure:         types.InfrastructureAzure,
			Provider:               types.ProviderAzure,
			Creator:                types.CreatorOpenAI,
			FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
			Parameters:             make(map[string]types.ParameterSpec),
			Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
		}
		err := registry.RegisterModel(model)
		if err != nil {
			t.Fatalf("Failed to register model with version %s: %v", version, err)
		}
	}

	// Get latest version using explicit FindLatestModel function
	latestModel, err := registry.FindLatestModel(
		types.InfrastructureAzure,
		types.ProviderAzure,
		types.CreatorOpenAI,
		"gpt-4",
	)
	if err != nil {
		t.Fatalf("Failed to find latest model: %v", err)
	}

	// Should return the newest version (0125 is January 25, 2024 - the latest)
	if latestModel.Version != "0125" {
		t.Errorf("Expected latest version 0125, got %s", latestModel.Version)
	}
}

func TestFindModelRequiresVersion(t *testing.T) {
	// Test that FindModel now requires explicit version and returns error when empty
	registry := NewRegistry()

	// Create a test model
	testModel := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(testModel)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	// Test that empty version returns error
	_, err = registry.FindModel(
		types.InfrastructureAzure,
		types.ProviderAzure,
		types.CreatorOpenAI,
		"gpt-4",
		"", // Empty version should now return error
	)
	if err == nil {
		t.Error("Expected error when version is empty, but got none")
	}

	// Test that whitespace-only version returns error
	_, err = registry.FindModel(
		types.InfrastructureAzure,
		types.ProviderAzure,
		types.CreatorOpenAI,
		"gpt-4",
		"   ", // Whitespace-only version should return error
	)
	if err == nil {
		t.Error("Expected error when version is whitespace-only, but got none")
	}

	// Test that explicit version works
	foundModel, err := registry.FindModel(
		types.InfrastructureAzure,
		types.ProviderAzure,
		types.CreatorOpenAI,
		"gpt-4",
		"0125", // Explicit version should work
	)
	if err != nil {
		t.Fatalf("Expected explicit version to work, but got error: %v", err)
	}
	if foundModel.Version != "0125" {
		t.Errorf("Expected version 0125, got %s", foundModel.Version)
	}
}

func TestModelCopyProtection(t *testing.T) {
	// Test that models are properly copied to prevent external modifications
	registry := NewRegistry()

	// Create a test model
	originalModel := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(originalModel)
	if err != nil {
		t.Fatalf("Failed to register original model: %v", err)
	}

	// Get the model from registry
	key := types.ModelKey{
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4",
		Version:        "0125",
	}

	retrievedModel, err := registry.FindModel(
		key.Infrastructure,
		key.Provider,
		key.Creator,
		key.ModelName,
		key.Version,
	)
	if err != nil {
		t.Fatalf("Failed to find model: %v", err)
	}

	// Modify the retrieved model
	retrievedModel.Name = "modified-name"

	// Get the model again and verify it wasn't modified
	retrievedAgain, err := registry.FindModel(
		key.Infrastructure,
		key.Provider,
		key.Creator,
		key.ModelName,
		key.Version,
	)
	if err != nil {
		t.Fatalf("Failed to find model again: %v", err)
	}

	if retrievedAgain.Name != "gpt-4" {
		t.Errorf("Model was modified externally, expected 'gpt-4', got '%s'", retrievedAgain.Name)
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Test concurrent access to Registry
	registry := NewRegistry()

	// Create a test model
	testModel := &types.Model{
		Name:                   "gpt-4",
		Version:                "0125",
		Infrastructure:         types.InfrastructureAzure,
		Provider:               types.ProviderAzure,
		Creator:                types.CreatorOpenAI,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{types.EndpointChatCompletions},
	}

	err := registry.RegisterModel(testModel)
	if err != nil {
		t.Fatalf("Failed to register test model: %v", err)
	}

	// Test concurrent reads
	const numGoroutines = 10
	errors := make(chan error, numGoroutines)
	results := make(chan string, numGoroutines)

	key := types.ModelKey{
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4",
		Version:        "0125",
	}

	for i := 0; i < numGoroutines; i++ {
		go func() {
			model, err := registry.FindModel(
				key.Infrastructure,
				key.Provider,
				key.Creator,
				key.ModelName,
				key.Version,
			)
			if err != nil {
				errors <- err
				return
			}
			results <- model.Name
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Errorf("Concurrent access error: %v", err)
		case result := <-results:
			if result != "gpt-4" {
				t.Errorf("Expected 'gpt-4', got '%s'", result)
			}
		}
	}
}

func TestMatchesModelIDPattern(t *testing.T) {
	tests := []struct {
		name        string
		modelID     string
		pattern     string
		shouldMatch bool
	}{
		{
			name:        "Exact match no wildcard",
			modelID:     "anthropic.claude-3-7-sonnet-20250219-v1:0",
			pattern:     "anthropic.claude-3-7-sonnet-20250219-v1:0",
			shouldMatch: true,
		},
		{
			name:        "Wildcard for date",
			modelID:     "anthropic.claude-3-7-sonnet-20250219-v1:0",
			pattern:     "anthropic.claude-3-7-sonnet-*-v1:0",
			shouldMatch: true,
		},
		{
			name:        "Different date matches wildcard",
			modelID:     "anthropic.claude-3-7-sonnet-20240229-v1:0",
			pattern:     "anthropic.claude-3-7-sonnet-*-v1:0",
			shouldMatch: true,
		},
		{
			name:        "Regional prefix with wildcard",
			modelID:     "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
			pattern:     "*.anthropic.claude-3-7-sonnet-*-v1:0",
			shouldMatch: true,
		},
		{
			name:        "Regional prefix exact then wildcard",
			modelID:     "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
			pattern:     "us.anthropic.claude-3-7-sonnet-*-v1:0",
			shouldMatch: true,
		},
		{
			name:        "No match different model",
			modelID:     "anthropic.claude-3-5-sonnet-20241022-v2:0",
			pattern:     "anthropic.claude-3-7-sonnet-*-v1:0",
			shouldMatch: false,
		},
		{
			name:        "Empty model ID",
			modelID:     "",
			pattern:     "anthropic.claude-*",
			shouldMatch: false,
		},
		{
			name:        "Empty pattern",
			modelID:     "anthropic.claude-3-7-sonnet-20250219-v1:0",
			pattern:     "",
			shouldMatch: false,
		},
		{
			name:        "Multiple wildcards",
			modelID:     "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
			pattern:     "*.*.claude-*-*-v1:0",
			shouldMatch: true,
		},
		{
			name:        "Amazon nova with wildcard",
			modelID:     "amazon.nova-lite-v1:0",
			pattern:     "amazon.nova-*-v1:0",
			shouldMatch: true,
		},
		{
			name:        "Amazon nova with regional prefix",
			modelID:     "us.amazon.nova-lite-v1:0",
			pattern:     "*.amazon.nova-*-v1:0",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesModelIDPattern(tt.modelID, tt.pattern)
			if result != tt.shouldMatch {
				t.Errorf("Expected match=%v, got match=%v for modelID=%s, pattern=%s",
					tt.shouldMatch, result, tt.modelID, tt.pattern)
			}
		})
	}
}

func TestFindModelByIDPattern(t *testing.T) {
	registry := NewRegistry()

	// Create test models with wildcard patterns in their IDs
	claude37Model := &types.Model{
		Name:                   "claude-3-7-sonnet",
		Version:                "v1",
		KEY:                    "anthropic.claude-3-7-sonnet-*-v1:0", // Wildcard pattern
		Infrastructure:         types.InfrastructureAWS,
		Provider:               types.ProviderBedrock,
		Creator:                types.CreatorAnthropic,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{"https://bedrock.amazonaws.com/invoke"},
	}

	novaModel := &types.Model{
		Name:                   "nova-lite",
		Version:                "v1",
		KEY:                    "amazon.nova-lite-v1:0", // No wildcard
		Infrastructure:         types.InfrastructureAWS,
		Provider:               types.ProviderBedrock,
		Creator:                types.CreatorAmazon,
		FunctionalCapabilities: []types.FunctionalCapability{types.FunctionalCapabilityChatCompletion},
		Parameters:             make(map[string]types.ParameterSpec),
		Endpoints:              []types.Endpoint{"https://bedrock.amazonaws.com/invoke"},
	}

	err := registry.RegisterModel(claude37Model)
	if err != nil {
		t.Fatalf("Failed to register claude-3-7 model: %v", err)
	}

	err = registry.RegisterModel(novaModel)
	if err != nil {
		t.Fatalf("Failed to register nova model: %v", err)
	}

	tests := []struct {
		name          string
		modelID       string
		expectFound   bool
		expectedModel string
	}{
		{
			name:          "Match claude with date 20250219",
			modelID:       "anthropic.claude-3-7-sonnet-20250219-v1:0",
			expectFound:   true,
			expectedModel: "claude-3-7-sonnet",
		},
		{
			name:          "Match claude with different date",
			modelID:       "anthropic.claude-3-7-sonnet-20240229-v1:0",
			expectFound:   true,
			expectedModel: "claude-3-7-sonnet",
		},
		{
			name:          "Match claude with regional prefix",
			modelID:       "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
			expectFound:   false, // Won't match because pattern doesn't include regional prefix
			expectedModel: "",
		},
		{
			name:          "Match nova exact",
			modelID:       "amazon.nova-lite-v1:0",
			expectFound:   true,
			expectedModel: "nova-lite",
		},
		{
			name:          "No match for claude-3-5",
			modelID:       "anthropic.claude-3-5-sonnet-20241022-v2:0",
			expectFound:   false,
			expectedModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var modelName string
			var creator types.Creator
			if strings.Contains(tt.modelID, "claude") {
				modelName = "claude-3-7-sonnet"
				creator = types.CreatorAnthropic
			} else {
				modelName = "nova-lite"
				creator = types.CreatorAmazon
			}

			model, err := registry.FindModelByIDPattern(
				types.InfrastructureAWS,
				types.ProviderBedrock,
				creator,
				modelName,
				tt.modelID,
			)

			if tt.expectFound {
				if err != nil {
					t.Errorf("Expected to find model, but got error: %v", err)
				}
				if model == nil {
					t.Error("Expected to find model, but got nil")
				} else if model.Name != tt.expectedModel {
					t.Errorf("Expected model name %s, got %s", tt.expectedModel, model.Name)
				}
			} else {
				if err == nil {
					t.Error("Expected error for non-matching model ID, but got none")
				}
			}
		})
	}
}
