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
	"os"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HTTP Client Tests

func TestMappingClient_GetModels(t *testing.T) {
	// Create mock server
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

	client := NewMappingClient(server.URL)
	ctx := context.Background()

	// First call - should fetch from endpoint
	result, err := client.GetModels(ctx)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "claude-3-5-sonnet", result[0].ModelMapping)

	// Second call - should return from cache
	result2, err := client.GetModels(ctx)
	require.NoError(t, err)
	require.Len(t, result2, 1)
	assert.Equal(t, "claude-3-5-sonnet", result2[0].ModelMapping)
}

func TestMappingClient_GetModels_Error(t *testing.T) {
	// Test with unreachable endpoint
	client := NewMappingClient("http://localhost:99999/invalid")
	ctx := context.Background()

	_, err := client.GetModels(ctx)
	require.Error(t, err)
}

func TestDefaultsClient_GetDefaults(t *testing.T) {
	defaults := DefaultModelConfig{
		Fast: &ModelDefault{
			ModelID:  "gpt-4o-mini",
			Provider: "Azure",
			Creator:  "openai",
		},
		Smart: &ModelDefault{
			ModelID:  "gpt-4o",
			Provider: "Azure",
			Creator:  "openai",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(defaults)
	}))
	defer server.Close()

	client := NewDefaultsClient(server.URL)
	ctx := context.Background()

	result, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "gpt-4o-mini", result.Fast.ModelID)
	assert.Equal(t, "gpt-4o", result.Smart.ModelID)
}

func TestTargetResolver_Getters(t *testing.T) {
	mapping := createTestMapping()
	mappingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mappingServer.Close()

	defaultsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer defaultsServer.Close()

	resolver, err := NewTargetResolver("", mappingServer.URL, defaultsServer.URL, "")
	require.NoError(t, err)
	resolver.staticMapping = mapping

	assert.NotNil(t, resolver.GetStaticMapping())
	assert.Equal(t, mapping, resolver.GetStaticMapping())
	assert.NotNil(t, resolver.GetMappingClient())
	assert.NotNil(t, resolver.GetDefaultsClient())
}

func TestNewTargetResolver_EmptyConfig(t *testing.T) {
	resolver, err := NewTargetResolver("", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, resolver)
	assert.Nil(t, resolver.GetStaticMapping())
	assert.Nil(t, resolver.GetMappingClient())
	assert.Nil(t, resolver.GetDefaultsClient())
}

func TestNewTargetResolver_WithAllConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := tempDir + "/test-config.yaml"

	yamlContent := `
models:
  - name: test-model
    infrastructure: azure
    active: true
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resolver, err := NewTargetResolver(configFile, server.URL, server.URL, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.GetStaticMapping())
	assert.NotNil(t, resolver.GetMappingClient())
	assert.NotNil(t, resolver.GetDefaultsClient())
}
