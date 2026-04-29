/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration Tests - Full Pipeline

func TestTargetResolver_FullPipeline_AzureOpenAI(t *testing.T) {
	mapping := createTestMapping()
	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(t, err)
	resolver.staticMapping = mapping // Set directly for test

	c := createTestGinContext("POST", "/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-15")
	target, err := resolver.Resolve(context.Background(), c)
	require.NoError(t, err)

	assert.Equal(t, TargetTypeLLM, target.TargetType)
	assert.Equal(t, types.InfrastructureAzure, target.Infrastructure)
	assert.Equal(t, types.ProviderAzure, target.Provider)
	assert.Equal(t, types.CreatorOpenAI, target.Creator)
	assert.Equal(t, "gpt-4o", target.ModelName)
	assert.Equal(t, "gpt-4o-2024-11-20", target.ModelID)
	assert.Equal(t, "2024-11-20", target.ModelVersion)
	assert.Contains(t, target.TargetURL, "https://azure-openai.openai.azure.com/chat/completions")
}

func TestTargetResolver_FullPipeline_Buddy(t *testing.T) {
	mapping := createTestMapping()
	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(t, err)
	resolver.staticMapping = mapping

	c := createTestGinContext("POST", "/v1/tenant123/buddies/selfstudybuddy/question")
	target, err := resolver.Resolve(context.Background(), c)
	require.NoError(t, err)

	assert.Equal(t, TargetTypeBuddy, target.TargetType)
	assert.Contains(t, target.TargetURL, "https://buddy-service.example.com/api/v1/question")
	assert.Empty(t, target.Infrastructure)
	assert.Empty(t, target.Provider)
}

func TestTargetResolver_FullPipeline_LocalEndpoint(t *testing.T) {
	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(t, err)

	c := createTestGinContext("GET", "/models")
	target, err := resolver.Resolve(context.Background(), c)
	require.NoError(t, err)

	assert.Equal(t, TargetTypeUnknown, target.TargetType)
	assert.Empty(t, target.TargetURL)
}

func TestFetchFromMappingEndpoint(t *testing.T) {
	// Create mock server for mapping endpoint
	models := []infra.ModelConfig{
		{
			ModelId:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
			ModelMapping: "claude-3-5-sonnet",
			Endpoint:     "https://bedrock-runtime.us-east-1.amazonaws.com",
			TargetApi:    "converse",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models)
	}))
	defer server.Close()

	resolver, err := NewTargetResolver("", server.URL, "", "")
	require.NoError(t, err)

	c := createTestGinContext("POST", "/anthropic/deployments/claude-3-5-sonnet/converse")
	req := &ResolutionRequest{
		GinContext: c,
		Target:     &ResolvedTarget{},
		Metadata: map[string]interface{}{
			"modelName": "claude-3-5-sonnet",
			"operation": "/converse",
		},
	}

	err = resolver.fetchFromMappingEndpoint(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, req.Metadata["infraConfig"])

	infraConfig := req.Metadata["infraConfig"].(*infra.ModelConfig)
	assert.Equal(t, "claude-3-5-sonnet", infraConfig.ModelMapping)
	assert.Equal(t, "anthropic.claude-3-5-sonnet-20241022-v2:0", infraConfig.ModelId)
}

func TestFullPipeline_WithStaticInfoEnrichment(t *testing.T) {
	// Create a model config with minimal information
	mapping := &api.Mapping{
		Models: []api.Model{
			{
				Name:        "gpt-35-turbo",
				ModelId:     "gpt-35-turbo",
				RedirectURL: "https://azure-openai.openai.azure.com",
				Active:      true,
				// Missing infrastructure, provider, creator - should be enriched from static map
			},
			{
				Name:           "gemini-1.5-flash",
				Infrastructure: "gcp",
				Provider:       "vertex",
				Creator:        "google",
				ModelId:        "gemini-1.5-flash", // No version - should be enriched
				RedirectURL:    "https://us-central1-aiplatform.googleapis.com",
				Active:         true,
			},
		},
	}

	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(t, err)
	resolver.staticMapping = mapping

	tests := []struct {
		name             string
		path             string
		wantInfra        string
		wantProvider     string
		wantCreator      string
		wantModelVersion string
	}{
		{
			name:             "Azure model enriched from static map",
			path:             "/openai/deployments/gpt-35-turbo/chat/completions",
			wantInfra:        "azure",
			wantProvider:     "azure",
			wantCreator:      "openai",
			wantModelVersion: "1106",
		},
		{
			name:             "GCP model version enriched from static map",
			path:             "/google/deployments/gemini-1.5-flash/generateContent",
			wantInfra:        "gcp",
			wantProvider:     "vertex",
			wantCreator:      "google",
			wantModelVersion: "002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createTestGinContext("POST", tt.path)
			target, err := resolver.Resolve(context.Background(), c)
			require.NoError(t, err)

			assert.Equal(t, types.Infrastructure(tt.wantInfra), target.Infrastructure)
			assert.Equal(t, types.Provider(tt.wantProvider), target.Provider)
			assert.Equal(t, types.Creator(tt.wantCreator), target.Creator)
			assert.Equal(t, tt.wantModelVersion, target.ModelVersion)
		})
	}
}
