/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit Tests - StaticTargetsByModelName Enrichment

func TestEnrichFromStaticInfo(t *testing.T) {
	tests := []struct {
		name           string
		modelName      string
		existingTarget *ResolvedTarget
		wantTarget     *ResolvedTarget
	}{
		{
			name:      "Azure model - enrich missing version",
			modelName: "gpt-4o",
			existingTarget: &ResolvedTarget{
				Infrastructure: "azure",
				Provider:       "Azure",
				Creator:        "openai",
				ModelName:      "gpt-4o",
				ModelID:        "gpt-4o",
				// ModelVersion is missing
			},
			wantTarget: &ResolvedTarget{
				Infrastructure: "azure",
				Provider:       "azure", // Provider is always overridden from static map for correct model lookup
				Creator:        "openai",
				ModelName:      "gpt-4o",
				ModelID:        "gpt-4o",
				ModelVersion:   "2024-11-20", // Should be enriched
			},
		},
		{
			name:           "Azure model - enrich all missing fields",
			modelName:      "gpt-35-turbo",
			existingTarget: &ResolvedTarget{
				// All fields missing
			},
			wantTarget: &ResolvedTarget{
				Infrastructure: "azure",
				Provider:       "azure",
				Creator:        "openai",
				ModelName:      "gpt-35-turbo",
				ModelVersion:   "1106",
			},
		},
		{
			name:      "AWS Bedrock model - NOT enriched from static map (should be from MAPPING_ENDPOINT)",
			modelName: "claude-3-5-sonnet",
			existingTarget: &ResolvedTarget{
				Infrastructure: "bedrock",
				Provider:       "Bedrock",
				Creator:        "anthropic",
				ModelName:      "claude-3-5-sonnet",
				// ModelVersion is missing
			},
			wantTarget: &ResolvedTarget{
				Infrastructure: "bedrock",
				Provider:       "Bedrock",
				Creator:        "anthropic",
				ModelName:      "claude-3-5-sonnet",
				// ModelVersion should remain empty - AWS models not enriched from static map
			},
		},
		{
			name:      "GCP model - enrich missing fields",
			modelName: "gemini-1.5-pro",
			existingTarget: &ResolvedTarget{
				ModelName: "gemini-1.5-pro",
				// Other fields missing
			},
			wantTarget: &ResolvedTarget{
				Infrastructure: "gcp",
				Provider:       "vertex",
				Creator:        "google",
				ModelName:      "gemini-1.5-pro",
				ModelVersion:   "002",
			},
		},
		{
			name:      "Do not override existing infrastructure/creator/version but always override provider and modelName",
			modelName: "gpt-4o",
			existingTarget: &ResolvedTarget{
				Infrastructure: "custom-infra",
				Provider:       "CustomProvider",
				Creator:        "custom-creator",
				ModelName:      "custom-name",
				ModelVersion:   "custom-version",
			},
			wantTarget: &ResolvedTarget{
				Infrastructure: "custom-infra", // Should not be overridden
				Provider:       "azure",        // Provider is always overridden from static map
				Creator:        "custom-creator",
				ModelName:      "gpt-4o", // ModelName is always overridden to resolve aliases
				ModelVersion:   "custom-version",
			},
		},
		{
			name:      "Model not in static map",
			modelName: "unknown-model",
			existingTarget: &ResolvedTarget{
				ModelName: "unknown-model",
			},
			wantTarget: &ResolvedTarget{
				ModelName: "unknown-model",
				// No enrichment should happen
			},
		},
		{
			name:      "Empty model name",
			modelName: "",
			existingTarget: &ResolvedTarget{
				ModelName: "some-model",
			},
			wantTarget: &ResolvedTarget{
				ModelName: "some-model",
				// No enrichment should happen
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying test data
			target := *tt.existingTarget
			enrichFromStaticInfo(&target, tt.modelName)

			assert.Equal(t, tt.wantTarget.Infrastructure, target.Infrastructure)
			assert.Equal(t, tt.wantTarget.Provider, target.Provider)
			assert.Equal(t, tt.wantTarget.Creator, target.Creator)
			assert.Equal(t, tt.wantTarget.ModelName, target.ModelName)
			assert.Equal(t, tt.wantTarget.ModelVersion, target.ModelVersion)
		})
	}
}

func TestStaticTargetsInfoMap_Coverage(t *testing.T) {
	// Test that Azure and GCP models are in the map (AWS models should NOT be here)
	expectedModels := []string{
		// Azure OpenAI
		"gpt-35-turbo",
		"gpt-35-turbo-1106",
		"gpt-4o",
		"gpt-4o-2024-11-20",
		"gpt-4o-2024-05-13",
		"gpt-4o-mini",
		"gpt-4-1106-preview",
		"gpt-4-vision-preview",
		"dall-e-3",
		"text-embedding-ada-002",
		"text-embedding-3-large",
		"text-embedding-3-small",
		// GCP Vertex
		"gemini-1.5-flash",
		"gemini-1.5-pro",
		"gemini-2.0-flash",
		"imagen-3",
		"imagen-3-fast",
		"text-multilingual-embedding-002",
	}

	for _, modelName := range expectedModels {
		t.Run(modelName, func(t *testing.T) {
			info, found := StaticTargetsByModelName[modelName]
			assert.True(t, found, "Model %s should be in StaticTargetsByModelName", modelName)
			assert.NotEmpty(t, info.Infrastructure, "Model %s should have Infrastructure", modelName)
			assert.NotEmpty(t, info.Provider, "Model %s should have Provider", modelName)
			assert.NotEmpty(t, info.Creator, "Model %s should have Creator", modelName)
			assert.NotEmpty(t, info.ModelName, "Model %s should have ModelName", modelName)
			assert.NotEmpty(t, info.ModelVersion, "Model %s should have ModelVersion", modelName)
		})
	}
}
