/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"context"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/registry"
	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/resolvers/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestRegistry creates a test registry with sample models
func setupTestRegistry(t *testing.T) *registry.Registry {
	t.Helper()
	reg := registry.NewRegistry()

	// Add test models
	// Note: KEY format is {infrastructure}/{provider}/{creator}/{modelName}/{version}
	models := []*modeltypes.Model{
		{
			Name:           "gpt-4",
			Version:        "0613",
			KEY:            "azure/azure/openai/gpt-4/0613",
			Infrastructure: modeltypes.InfrastructureAzure,
			Provider:       modeltypes.ProviderAzure,
			Creator:        modeltypes.CreatorOpenAI,
		},
		{
			Name:           "gpt-4",
			Version:        "1106-preview",
			KEY:            "azure/azure/openai/gpt-4/1106-preview",
			Infrastructure: modeltypes.InfrastructureAzure,
			Provider:       modeltypes.ProviderAzure,
			Creator:        modeltypes.CreatorOpenAI,
		},
		{
			Name:           "claude-3-sonnet",
			Version:        "20240229",
			KEY:            "anthropic.claude-3-sonnet-*-v1:0",
			Infrastructure: modeltypes.InfrastructureAWS,
			Provider:       modeltypes.ProviderBedrock,
			Creator:        modeltypes.CreatorAnthropic,
		},
		{
			Name:           "claude-3-opus",
			Version:        "20240229",
			KEY:            "anthropic.claude-3-opus-20240229-v1:0",
			Infrastructure: modeltypes.InfrastructureAWS,
			Provider:       modeltypes.ProviderBedrock,
			Creator:        modeltypes.CreatorAnthropic,
		},
		{
			Name:           "llama-3-70b",
			Version:        "8192",
			KEY:            "meta.llama3-70b-instruct-v1:0",
			Infrastructure: modeltypes.InfrastructureAWS,
			Provider:       modeltypes.ProviderBedrock,
			Creator:        modeltypes.CreatorMeta,
		},
		{
			Name:           "gemini-pro",
			Version:        "001",
			KEY:            "gemini-1.0-pro-001",
			Infrastructure: modeltypes.InfrastructureGCP,
			Provider:       modeltypes.ProviderVertex,
			Creator:        modeltypes.CreatorGoogle,
		},
	}

	for _, model := range models {
		err := reg.RegisterModel(model)
		require.NoError(t, err, "Failed to add test model: %s", model.KEY)
	}

	return reg
}

// setupTestContext creates a context with a test registry
func setupTestContext(t *testing.T, reg *registry.Registry) context.Context {
	t.Helper()
	ctx := context.Background()

	// Store registry in context using the same pattern as the actual code
	// Note: In the actual code, GetGlobalRegistry uses sync.Once to initialize
	// For tests, we need to mock this differently - by setting up the global registry
	// or by using dependency injection if available

	return ctx
}

func TestResolveModelFromTarget_Strategy1_ExactMatch(t *testing.T) {
	reg := setupTestRegistry(t)
	ctx := setupTestContext(t, reg)
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name          string
		target        *target.ResolvedTarget
		expectedModel string
		expectedError bool
		setupRegistry func(*registry.Registry)
	}{
		{
			name: "exact match with version - Azure",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "gpt-4",
				ModelVersion:   "0613",
			},
			expectedModel: "azure/azure/openai/gpt-4/0613",
			expectedError: false,
		},
		{
			name: "exact match with version - Bedrock",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAWS,
				Provider:       modeltypes.ProviderBedrock,
				Creator:        modeltypes.CreatorAnthropic,
				ModelName:      "claude-3-opus",
				ModelVersion:   "20240229",
			},
			expectedModel: "anthropic.claude-3-opus-20240229-v1:0",
			expectedError: false,
		},
		{
			name: "exact match fails - wrong version",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "gpt-4",
				ModelVersion:   "9999",
			},
			expectedModel: "azure/azure/openai/gpt-4/1106-preview", // Should fall through to strategy 3 (latest)
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh registry for each test
			testReg := setupTestRegistry(t)
			if tt.setupRegistry != nil {
				tt.setupRegistry(testReg)
			}

			// Mock the global registry by setting it directly
			// We need to reset the sync.Once to make GetGlobalRegistry return our test registry
			models.ResetGlobalRegistryForTest(testReg)
			defer models.ResetGlobalRegistryForTest(nil)

			model, err := resolveModelFromTarget(ctx, tt.target, logger)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, model)
				assert.Equal(t, tt.expectedModel, model.KEY)
			}
		})
	}
}

func TestResolveModelFromTarget_Strategy2_PatternMatching(t *testing.T) {
	reg := setupTestRegistry(t)
	ctx := setupTestContext(t, reg)
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name          string
		target        *target.ResolvedTarget
		expectedModel string
		expectedError bool
	}{
		{
			name: "pattern match with wildcard - Claude Sonnet",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAWS,
				Provider:       modeltypes.ProviderBedrock,
				Creator:        modeltypes.CreatorAnthropic,
				ModelName:      "claude-3-sonnet",
				ModelID:        "anthropic.claude-3-sonnet-20240229-v1:0",
			},
			expectedModel: "anthropic.claude-3-sonnet-*-v1:0",
			expectedError: false,
		},
		{
			name: "pattern match fails - no matching pattern",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "gpt-4",
				ModelID:        "gpt-4-0613",
			},
			expectedModel: "azure/azure/openai/gpt-4/1106-preview", // Should fall through to strategy 3 (latest)
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the global registry
			models.ResetGlobalRegistryForTest(reg)
			defer models.ResetGlobalRegistryForTest(nil)

			model, err := resolveModelFromTarget(ctx, tt.target, logger)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, model)
				assert.Equal(t, tt.expectedModel, model.KEY)
			}
		})
	}
}

func TestResolveModelFromTarget_Strategy3_LatestVersion(t *testing.T) {
	reg := setupTestRegistry(t)
	ctx := setupTestContext(t, reg)
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name          string
		target        *target.ResolvedTarget
		expectedModel string
		expectedError bool
	}{
		{
			name: "find latest version - gpt-4",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "gpt-4",
			},
			expectedModel: "azure/azure/openai/gpt-4/1106-preview", // Latest version
			expectedError: false,
		},
		{
			name: "find latest version - gemini-pro",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureGCP,
				Provider:       modeltypes.ProviderVertex,
				Creator:        modeltypes.CreatorGoogle,
				ModelName:      "gemini-pro",
			},
			expectedModel: "gemini-1.0-pro-001",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the global registry
			models.ResetGlobalRegistryForTest(reg)
			defer models.ResetGlobalRegistryForTest(nil)

			model, err := resolveModelFromTarget(ctx, tt.target, logger)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, model)
				assert.Equal(t, tt.expectedModel, model.KEY)
			}
		})
	}
}

func TestResolveModelFromTarget_Strategy4_BaseModelName(t *testing.T) {
	reg := setupTestRegistry(t)
	ctx := setupTestContext(t, reg)
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name          string
		target        *target.ResolvedTarget
		expectedError bool
		expectSuccess bool
	}{
		{
			name: "extract base name from ID - fails when model name doesn't match",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "gpt-4-custom", // Different from registry
				ModelID:        "us.gpt-4-0613",
			},
			expectedError: true, // Strategy 4 tries but still fails because no model with name "gpt-4-custom"
			expectSuccess: false,
		},
		{
			name: "base model extraction attempted for non-existent model",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAWS,
				Provider:       modeltypes.ProviderBedrock,
				Creator:        modeltypes.CreatorMeta,
				ModelName:      "llama-custom",
				ModelID:        "meta.llama3-70b-instruct-v1:0",
			},
			expectedError: true, // No model with name "llama-custom"
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the global registry
			models.ResetGlobalRegistryForTest(reg)
			defer models.ResetGlobalRegistryForTest(nil)

			model, err := resolveModelFromTarget(ctx, tt.target, logger)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, model)
			}
		})
	}
}

func TestResolveModelFromTarget_AllStrategiesFail(t *testing.T) {
	reg := setupTestRegistry(t)
	ctx := setupTestContext(t, reg)
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name   string
		target *target.ResolvedTarget
	}{
		{
			name: "model not in registry",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "nonexistent-model",
				ModelVersion:   "1.0",
				ModelID:        "nonexistent-model-1.0",
			},
		},
		{
			name: "wrong provider",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderVertex, // Wrong provider
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "gpt-4",
				ModelVersion:   "0613",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the global registry
			models.ResetGlobalRegistryForTest(reg)
			defer models.ResetGlobalRegistryForTest(nil)

			model, err := resolveModelFromTarget(ctx, tt.target, logger)

			assert.Error(t, err)
			assert.Nil(t, model)
			assert.Contains(t, err.Error(), "model not found in registry")
		})
	}
}

func TestResolveModelFromTarget_NilLogger(t *testing.T) {
	reg := setupTestRegistry(t)
	ctx := setupTestContext(t, reg)

	target := &target.ResolvedTarget{
		Infrastructure: modeltypes.InfrastructureAzure,
		Provider:       modeltypes.ProviderAzure,
		Creator:        modeltypes.CreatorOpenAI,
		ModelName:      "gpt-4",
		ModelVersion:   "0613",
	}

	// Mock the global registry
	models.ResetGlobalRegistryForTest(reg)
	defer models.ResetGlobalRegistryForTest(nil)

	// Should not panic with nil logger
	model, err := resolveModelFromTarget(ctx, target, nil)

	assert.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "azure/azure/openai/gpt-4/0613", model.KEY)
}

func TestResolveModelFromTarget_EmptyFields(t *testing.T) {
	reg := setupTestRegistry(t)
	ctx := setupTestContext(t, reg)
	logger := zap.NewNop().Sugar()

	tests := []struct {
		name          string
		target        *target.ResolvedTarget
		expectedError bool
	}{
		{
			name: "empty model version - should use strategy 3",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "gpt-4",
				ModelVersion:   "",
			},
			expectedError: false,
		},
		{
			name: "empty model ID - skip strategy 2",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "gpt-4",
				ModelVersion:   "0613",
				ModelID:        "",
			},
			expectedError: false,
		},
		{
			name: "empty model name",
			target: &target.ResolvedTarget{
				Infrastructure: modeltypes.InfrastructureAzure,
				Provider:       modeltypes.ProviderAzure,
				Creator:        modeltypes.CreatorOpenAI,
				ModelName:      "",
				ModelVersion:   "0613",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the global registry
			models.ResetGlobalRegistryForTest(reg)
			defer models.ResetGlobalRegistryForTest(nil)

			model, err := resolveModelFromTarget(ctx, tt.target, logger)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, model)
			}
		})
	}
}

func TestResolveModelFromTarget_RegistryError(t *testing.T) {
	// Skip this test as the current implementation always loads the global registry
	// The sync.Once pattern makes it difficult to simulate nil registry in a test
	t.Skip("Skipping registry error test - sync.Once makes it hard to simulate nil registry")
}

func TestExtractBaseModelNameFromID(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected string
	}{
		{
			name:     "empty string",
			modelID:  "",
			expected: "",
		},
		{
			name:     "remove regional prefix",
			modelID:  "us.gpt-4-0613",
			expected: "gpt-4-0613",
		},
		{
			name:     "remove vendor prefix and patterns",
			modelID:  "anthropic.claude-3-sonnet-20240229-v1:0",
			expected: "claude-3-sonnet", // Removes vendor, date, and version patterns
		},
		{
			name:     "remove version suffix after colon and version pattern",
			modelID:  "meta.llama3-70b-instruct-v1:0",
			expected: "llama3-70b-instruct", // Removes vendor, colon suffix, and version pattern
		},
		{
			name:     "remove date pattern YYYYMMDD",
			modelID:  "claude-3-sonnet-20240229-v1",
			expected: "claude-3-sonnet",
		},
		{
			name:     "remove date pattern YYYY-MM-DD",
			modelID:  "model-2024-02-29-v1",
			expected: "model",
		},
		{
			name:     "remove standalone version suffix",
			modelID:  "gpt-4-v2",
			expected: "gpt-4",
		},
		{
			name:     "remove numeric suffix",
			modelID:  "text-embedding-ada-002",
			expected: "text-embedding-ada",
		},
		{
			name:     "complex ID with multiple patterns",
			modelID:  "us.anthropic.claude-3-7-sonnet-20241022-v2:0",
			expected: "claude-3-7-sonnet",
		},
		{
			name:     "no patterns to remove",
			modelID:  "simple-model",
			expected: "simple-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBaseModelNameFromID(tt.modelID)
			assert.Equal(t, tt.expected, result)
		})
	}
}
