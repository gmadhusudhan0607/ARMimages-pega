/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit Tests - Helper Functions

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name        string
		modelID     string
		wantVersion string
	}{
		{
			name:        "Azure date format",
			modelID:     "gpt-4o-2024-11-20",
			wantVersion: "2024-11-20",
		},
		{
			name:        "Bedrock date with version suffix",
			modelID:     "anthropic.claude-3-5-sonnet-20241022-v2:0",
			wantVersion: "v2", // Extract only version indicator for registry matching
		},
		{
			name:        "GCP numeric version",
			modelID:     "gemini-1.5-pro-002",
			wantVersion: "002",
		},
		{
			name:        "No version",
			modelID:     "some-model",
			wantVersion: "",
		},
		{
			name:        "Empty model ID",
			modelID:     "",
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := extractVersion(tt.modelID)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}

func TestExtractCreatorFromModelId(t *testing.T) {
	tests := []struct {
		name        string
		modelID     string
		wantCreator string
	}{
		{
			name:        "Anthropic",
			modelID:     "anthropic.claude-3-5-sonnet-20241022-v2:0",
			wantCreator: "anthropic",
		},
		{
			name:        "Anthropic with regional prefix",
			modelID:     "us.anthropic.claude-3-5-sonnet-20241022-v2:0",
			wantCreator: "anthropic",
		},
		{
			name:        "Amazon",
			modelID:     "amazon.titan-text-express-v1",
			wantCreator: "amazon",
		},
		{
			name:        "Amazon with regional prefix",
			modelID:     "us.amazon.nova-lite-v1:0",
			wantCreator: "amazon",
		},
		{
			name:        "Meta",
			modelID:     "meta.llama3-70b-instruct-v1:0",
			wantCreator: "meta",
		},
		{
			name:        "Meta with regional prefix",
			modelID:     "eu.meta.llama3-70b-instruct-v1:0",
			wantCreator: "meta",
		},
		{
			name:        "No creator",
			modelID:     "some-model",
			wantCreator: "some-model",
		},
		{
			name:        "Empty model ID",
			modelID:     "",
			wantCreator: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := extractCreatorFromModelId(tt.modelID)
			assert.Equal(t, tt.wantCreator, creator)
		})
	}
}

func TestEnrichWithInfrastructure_StaticInfoEnrichment(t *testing.T) {
	// Test that enrichWithInfrastructure calls enrichFromStaticInfo
	resolver := &TargetResolver{}

	// Model with incomplete information in config
	modelConfig := &api.Model{
		Name:           "gpt-4o",
		Infrastructure: "azure",
		Provider:       "azure",
		Creator:        "openai",
		ModelId:        "gpt-4o", // No version in ModelId
	}

	req := &ResolutionRequest{
		GinContext: createTestGinContext("POST", "/test"),
		Target:     &ResolvedTarget{},
		Metadata: map[string]interface{}{
			"modelConfig": modelConfig,
			"modelName":   "gpt-4o",
		},
	}

	err := resolver.enrichWithInfrastructure(context.Background(), req)
	require.NoError(t, err)

	// Verify that basic info from config is set
	assert.Equal(t, types.Infrastructure("azure"), req.Target.Infrastructure)
	assert.Equal(t, types.Provider("azure"), req.Target.Provider)
	assert.Equal(t, types.Creator("openai"), req.Target.Creator)
}

func TestEnrichWithModelMetadata_StaticInfoEnrichment(t *testing.T) {
	// Test that enrichWithModelMetadata enriches model version from static map
	resolver := &TargetResolver{}

	// Model config without version in ModelId
	modelConfig := &api.Model{
		Name:    "gpt-35-turbo",
		ModelId: "gpt-35-turbo", // No version suffix
	}

	req := &ResolutionRequest{
		GinContext: createTestGinContext("POST", "/test"),
		Target:     &ResolvedTarget{},
		Metadata: map[string]interface{}{
			"modelConfig": modelConfig,
			"modelName":   "gpt-35-turbo",
		},
	}

	err := resolver.enrichWithModelMetadata(context.Background(), req)
	require.NoError(t, err)

	// Verify that model metadata is enriched
	assert.Equal(t, "gpt-35-turbo", req.Target.ModelName)
	assert.Equal(t, "gpt-35-turbo", req.Target.ModelID)
	// Version should be enriched from static map since extractVersion returns empty
	assert.Equal(t, "1106", req.Target.ModelVersion)
}
