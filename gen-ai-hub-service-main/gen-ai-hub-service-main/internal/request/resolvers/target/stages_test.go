/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit Tests - Stage 1: extractBasicInfo

func TestExtractBasicInfo_AzureOpenAI(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantPattern   string
		wantModelName string
		wantOperation string
	}{
		{
			name:          "Azure OpenAI chat completion",
			path:          "/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-15",
			wantPattern:   "openai",
			wantModelName: "gpt-4o",
			wantOperation: "/chat/completions",
		},
		{
			name:          "Azure OpenAI embeddings",
			path:          "/openai/deployments/text-embedding-3-large/embeddings?api-version=2024-02-15",
			wantPattern:   "openai",
			wantModelName: "text-embedding-3-large",
			wantOperation: "/embeddings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TargetResolver{}
			c := createTestGinContext("POST", tt.path)
			req := &ResolutionRequest{
				GinContext: c,
				Target:     &ResolvedTarget{},
				Metadata:   make(map[string]interface{}),
			}

			err := resolver.extractBasicInfo(context.Background(), req)
			require.NoError(t, err)

			assert.Equal(t, tt.wantPattern, req.Metadata["routePattern"])
			assert.Equal(t, tt.wantModelName, req.Metadata["modelName"])
			assert.Equal(t, tt.wantOperation, req.Metadata["operation"])
		})
	}
}

func TestExtractBasicInfo_AWSBedrock(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantPattern   string
		wantModelName string
		wantOperation string
	}{
		{
			name:          "Anthropic converse",
			path:          "/anthropic/deployments/claude-3-5-sonnet/converse",
			wantPattern:   "anthropic",
			wantModelName: "claude-3-5-sonnet",
			wantOperation: "/converse",
		},
		{
			name:          "Meta invoke",
			path:          "/meta/deployments/llama3-70b/invoke",
			wantPattern:   "meta",
			wantModelName: "llama3-70b",
			wantOperation: "/invoke",
		},
		{
			name:          "Amazon invoke",
			path:          "/amazon/deployments/titan-text/invoke",
			wantPattern:   "amazon",
			wantModelName: "titan-text",
			wantOperation: "/invoke",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TargetResolver{}
			c := createTestGinContext("POST", tt.path)
			req := &ResolutionRequest{
				GinContext: c,
				Target:     &ResolvedTarget{},
				Metadata:   make(map[string]interface{}),
			}

			err := resolver.extractBasicInfo(context.Background(), req)
			require.NoError(t, err)

			assert.Equal(t, tt.wantPattern, req.Metadata["routePattern"])
			assert.Equal(t, tt.wantModelName, req.Metadata["modelName"])
			assert.Equal(t, tt.wantOperation, req.Metadata["operation"])
		})
	}
}

func TestExtractBasicInfo_GCPVertex(t *testing.T) {
	c := createTestGinContext("POST", "/google/deployments/gemini-1.5-pro/generateContent")
	req := &ResolutionRequest{
		GinContext: c,
		Target:     &ResolvedTarget{},
		Metadata:   make(map[string]interface{}),
	}

	resolver := &TargetResolver{}
	err := resolver.extractBasicInfo(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "google", req.Metadata["routePattern"])
	assert.Equal(t, "gemini-1.5-pro", req.Metadata["modelName"])
	assert.Equal(t, "/generateContent", req.Metadata["operation"])
}

func TestExtractBasicInfo_Buddies(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		wantBuddyID     string
		wantIsolationID string
		wantOperation   string
	}{
		{
			name:            "Buddy with isolation ID",
			path:            "/v1/tenant123/buddies/selfstudybuddy/question",
			wantBuddyID:     "selfstudybuddy",
			wantIsolationID: "tenant123",
			wantOperation:   "/question",
		},
		{
			name:            "Buddy without isolation ID",
			path:            "/buddies/selfstudybuddy/question",
			wantBuddyID:     "selfstudybuddy",
			wantIsolationID: "",
			wantOperation:   "/question",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TargetResolver{}
			c := createTestGinContext("POST", tt.path)
			req := &ResolutionRequest{
				GinContext: c,
				Target:     &ResolvedTarget{},
				Metadata:   make(map[string]interface{}),
			}

			err := resolver.extractBasicInfo(context.Background(), req)
			require.NoError(t, err)

			assert.Equal(t, "buddies", req.Metadata["routePattern"])
			assert.Equal(t, tt.wantBuddyID, req.Metadata["buddyId"])
			assert.Equal(t, tt.wantIsolationID, req.Metadata["isolationId"])
			assert.Equal(t, tt.wantOperation, req.Metadata["operation"])
		})
	}
}

func TestExtractBasicInfo_LocalEndpoints(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantPattern string
	}{
		{
			name:        "Models endpoint",
			path:        "/models",
			wantPattern: "models",
		},
		{
			name:        "Swagger endpoint",
			path:        "/swagger/index.html",
			wantPattern: "swagger",
		},
		{
			name:        "Health endpoint",
			path:        "/health",
			wantPattern: "health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TargetResolver{}
			c := createTestGinContext("GET", tt.path)
			req := &ResolutionRequest{
				GinContext: c,
				Target:     &ResolvedTarget{},
				Metadata:   make(map[string]interface{}),
			}

			err := resolver.extractBasicInfo(context.Background(), req)
			require.NoError(t, err)

			assert.Equal(t, tt.wantPattern, req.Metadata["routePattern"])
		})
	}
}

// Unit Tests - Stage 2: determineTargetType

func TestDetermineTargetType(t *testing.T) {
	tests := []struct {
		name         string
		routePattern string
		wantType     TargetType
	}{
		{
			name:         "Azure OpenAI",
			routePattern: "openai",
			wantType:     TargetTypeLLM,
		},
		{
			name:         "Anthropic",
			routePattern: "anthropic",
			wantType:     TargetTypeLLM,
		},
		{
			name:         "Meta",
			routePattern: "meta",
			wantType:     TargetTypeLLM,
		},
		{
			name:         "Amazon",
			routePattern: "amazon",
			wantType:     TargetTypeLLM,
		},
		{
			name:         "Google",
			routePattern: "google",
			wantType:     TargetTypeLLM,
		},
		{
			name:         "Buddies",
			routePattern: "buddies",
			wantType:     TargetTypeBuddy,
		},
		{
			name:         "Models endpoint",
			routePattern: "models",
			wantType:     TargetTypeUnknown,
		},
		{
			name:         "Swagger endpoint",
			routePattern: "swagger",
			wantType:     TargetTypeUnknown,
		},
		{
			name:         "Health endpoint",
			routePattern: "health",
			wantType:     TargetTypeUnknown,
		},
		{
			name:         "Unknown route",
			routePattern: "unknown",
			wantType:     TargetTypeUnknown,
		},
		{
			name:         "Empty route",
			routePattern: "",
			wantType:     TargetTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TargetResolver{}
			req := &ResolutionRequest{
				GinContext: createTestGinContext("GET", "/test"),
				Target:     &ResolvedTarget{},
				Metadata: map[string]interface{}{
					"routePattern": tt.routePattern,
				},
			}

			err := resolver.determineTargetType(context.Background(), req)
			require.NoError(t, err)

			assert.Equal(t, tt.wantType, req.Target.TargetType)
		})
	}
}

func TestDetermineTargetType_MissingRoutePattern(t *testing.T) {
	resolver := &TargetResolver{}
	req := &ResolutionRequest{
		GinContext: createTestGinContext("GET", "/test"),
		Target:     &ResolvedTarget{},
		Metadata:   make(map[string]interface{}),
	}

	err := resolver.determineTargetType(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route pattern not found")
}

// Unit Tests - Stage 3: fetchModelConfiguration

func TestFetchFromStaticMapping(t *testing.T) {
	mapping := createTestMapping()
	resolver := &TargetResolver{
		staticMapping: mapping,
	}

	tests := []struct {
		name      string
		modelName string
		infra     string
		wantError bool
	}{
		{
			name:      "Azure model found",
			modelName: "gpt-4o",
			infra:     "azure",
			wantError: false,
		},
		{
			name:      "Bedrock model found",
			modelName: "claude-3-5-sonnet",
			infra:     "bedrock",
			wantError: false,
		},
		{
			name:      "GCP model found",
			modelName: "gemini-1.5-pro",
			infra:     "gcp",
			wantError: false,
		},
		{
			name:      "Model not found",
			modelName: "nonexistent-model",
			infra:     "azure",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ResolutionRequest{
				GinContext: createTestGinContext("POST", "/test"),
				Target:     &ResolvedTarget{},
				Metadata: map[string]interface{}{
					"modelName": tt.modelName,
				},
			}

			err := resolver.fetchFromStaticMapping(context.Background(), req, tt.infra)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, req.Metadata["modelConfig"])
				model := req.Metadata["modelConfig"].(*api.Model)
				assert.Equal(t, tt.modelName, model.Name)
			}
		})
	}
}

func TestFetchBuddyConfiguration(t *testing.T) {
	mapping := createTestMapping()
	resolver := &TargetResolver{
		staticMapping: mapping,
	}

	tests := []struct {
		name      string
		buddyID   string
		wantError bool
	}{
		{
			name:      "Buddy found",
			buddyID:   "selfstudybuddy",
			wantError: false,
		},
		{
			name:      "Buddy not found",
			buddyID:   "nonexistent-buddy",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ResolutionRequest{
				GinContext: createTestGinContext("POST", "/test"),
				Target:     &ResolvedTarget{},
				Metadata: map[string]interface{}{
					"buddyId": tt.buddyID,
				},
			}

			err := resolver.fetchBuddyConfiguration(context.Background(), req)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, req.Metadata["buddyConfig"])
				buddy := req.Metadata["buddyConfig"].(*api.Buddy)
				assert.Equal(t, tt.buddyID, buddy.Name)
			}
		})
	}
}

// Unit Tests - Stage 4: enrichWithInfrastructure

func TestEnrichWithInfrastructure_StaticMapping(t *testing.T) {
	tests := []struct {
		name         string
		model        *api.Model
		wantInfra    types.Infrastructure
		wantProvider types.Provider
		wantCreator  types.Creator
	}{
		{
			name: "Azure model",
			model: &api.Model{
				Name:           "gpt-4o",
				Infrastructure: "azure",
				Provider:       "azure",
				Creator:        "openai",
			},
			wantInfra:    types.InfrastructureAzure,
			wantProvider: types.ProviderAzure,
			wantCreator:  types.CreatorOpenAI,
		},
		{
			name: "Bedrock model",
			model: &api.Model{
				Name:           "claude-3-5-sonnet",
				Infrastructure: "aws",
				Provider:       "bedrock",
				Creator:        "anthropic",
			},
			wantInfra:    types.InfrastructureAWS,
			wantProvider: types.ProviderBedrock,
			wantCreator:  types.CreatorAnthropic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TargetResolver{}
			req := &ResolutionRequest{
				GinContext: createTestGinContext("POST", "/test"),
				Target:     &ResolvedTarget{},
				Metadata: map[string]interface{}{
					"modelConfig": tt.model,
				},
			}

			err := resolver.enrichWithInfrastructure(context.Background(), req)
			require.NoError(t, err)

			assert.Equal(t, tt.wantInfra, req.Target.Infrastructure)
			assert.Equal(t, tt.wantProvider, req.Target.Provider)
			assert.Equal(t, tt.wantCreator, req.Target.Creator)
		})
	}
}

func TestEnrichWithInfrastructure_InfraConfig(t *testing.T) {
	resolver := &TargetResolver{}
	infraConfig := &infra.ModelConfig{
		ModelId:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
		ModelMapping: "claude-3-5-sonnet",
		Endpoint:     "https://bedrock-runtime.us-east-1.amazonaws.com",
	}

	req := &ResolutionRequest{
		GinContext: createTestGinContext("POST", "/test"),
		Target:     &ResolvedTarget{},
		Metadata: map[string]interface{}{
			"infraConfig": infraConfig,
		},
	}

	err := resolver.enrichWithInfrastructure(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, types.InfrastructureAWS, req.Target.Infrastructure)
	assert.Equal(t, types.ProviderBedrock, req.Target.Provider)
	assert.Equal(t, types.CreatorAnthropic, req.Target.Creator)
}

// Unit Tests - Stage 5: enrichWithModelMetadata

func TestEnrichWithModelMetadata_StaticMapping(t *testing.T) {
	tests := []struct {
		name          string
		model         *api.Model
		wantModelName string
		wantModelID   string
		wantVersion   string
	}{
		{
			name: "Azure model with date version",
			model: &api.Model{
				Name:    "gpt-4o",
				ModelId: "gpt-4o-2024-11-20",
			},
			wantModelName: "gpt-4o",
			wantModelID:   "gpt-4o-2024-11-20",
			wantVersion:   "2024-11-20",
		},
		{
			name: "Bedrock model with date and version suffix",
			model: &api.Model{
				Name:    "claude-3-5-sonnet",
				ModelId: "anthropic.claude-3-5-sonnet-20241022-v2:0",
			},
			wantModelName: "claude-3-5-sonnet",
			wantModelID:   "anthropic.claude-3-5-sonnet-20241022-v2:0",
			wantVersion:   "v2", // Extract only version indicator for registry matching
		},
		{
			name: "GCP model with numeric version",
			model: &api.Model{
				Name:    "gemini-1.5-pro",
				ModelId: "gemini-1.5-pro-002",
			},
			wantModelName: "gemini-1.5-pro",
			wantModelID:   "gemini-1.5-pro-002",
			wantVersion:   "002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TargetResolver{}
			req := &ResolutionRequest{
				GinContext: createTestGinContext("POST", "/test"),
				Target:     &ResolvedTarget{},
				Metadata: map[string]interface{}{
					"modelConfig": tt.model,
				},
			}

			err := resolver.enrichWithModelMetadata(context.Background(), req)
			require.NoError(t, err)

			assert.Equal(t, tt.wantModelName, req.Target.ModelName)
			assert.Equal(t, tt.wantModelID, req.Target.ModelID)
			assert.Equal(t, tt.wantVersion, req.Target.ModelVersion)
		})
	}
}

// Unit Tests - Stage 6: constructTargetURL

func TestConstructLLMTargetURL(t *testing.T) {
	tests := []struct {
		name      string
		model     *api.Model
		operation string
		rawQuery  string
		wantURL   string
	}{
		{
			name: "Azure chat completion with query",
			model: &api.Model{
				RedirectURL: "https://azure-openai.openai.azure.com",
			},
			operation: "/chat/completions",
			rawQuery:  "api-version=2024-02-15",
			wantURL:   "https://azure-openai.openai.azure.com/chat/completions?api-version=2024-02-15",
		},
		{
			name: "GCP without query",
			model: &api.Model{
				RedirectURL: "https://us-central1-aiplatform.googleapis.com",
			},
			operation: "/generateContent",
			rawQuery:  "",
			wantURL:   "https://us-central1-aiplatform.googleapis.com/generateContent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &TargetResolver{}
			req := &ResolutionRequest{
				GinContext: createTestGinContext("POST", "/test"),
				Target:     &ResolvedTarget{TargetType: TargetTypeLLM},
				Metadata: map[string]interface{}{
					"modelConfig": tt.model,
					"operation":   tt.operation,
					"rawQuery":    tt.rawQuery,
				},
			}

			err := resolver.constructLLMTargetURL(context.Background(), req)
			require.NoError(t, err)

			assert.Equal(t, tt.wantURL, req.Target.TargetURL)
		})
	}
}

func TestConstructBuddyTargetURL(t *testing.T) {
	resolver := &TargetResolver{}
	buddy := &api.Buddy{
		Name:        "selfstudybuddy",
		RedirectURL: "https://buddy-service.example.com/api/v1",
	}

	req := &ResolutionRequest{
		GinContext: createTestGinContext("POST", "/test"),
		Target:     &ResolvedTarget{TargetType: TargetTypeBuddy},
		Metadata: map[string]interface{}{
			"buddyConfig": buddy,
			"operation":   "/question",
			"rawQuery":    "",
		},
	}

	err := resolver.constructBuddyTargetURL(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "https://buddy-service.example.com/api/v1/question", req.Target.TargetURL)
}

func TestConstructTargetURL_WithInfraConfig(t *testing.T) {
	resolver := &TargetResolver{}
	infraConfig := &infra.ModelConfig{
		Endpoint: "https://bedrock-runtime.us-east-1.amazonaws.com",
		Path:     "/model/anthropic.claude-3-5-sonnet-20241022-v2:0",
	}

	req := &ResolutionRequest{
		GinContext: createTestGinContext("POST", "/test"),
		Target:     &ResolvedTarget{TargetType: TargetTypeLLM},
		Metadata: map[string]interface{}{
			"infraConfig": infraConfig,
			"operation":   "/converse",
			"rawQuery":    "",
		},
	}

	err := resolver.constructLLMTargetURL(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "https://bedrock-runtime.us-east-1.amazonaws.com/model/anthropic.claude-3-5-sonnet-20241022-v2:0/converse", req.Target.TargetURL)
}
