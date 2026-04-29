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
	"sync"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cache Tests - DefaultsClient

func TestDefaultsClient_Cache_HitAndMiss(t *testing.T) {
	requestCount := 0
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
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(defaults)
	}))
	defer server.Close()

	client := NewDefaultsClient(server.URL)
	ctx := context.Background()

	// First call - cache miss, should hit server
	result1, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result1)
	assert.Equal(t, "gpt-4o-mini", result1.Fast.ModelID)
	assert.Equal(t, 1, requestCount, "First call should hit the server")

	// Second call - cache hit, should NOT hit server
	result2, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result2)
	assert.Equal(t, "gpt-4o-mini", result2.Fast.ModelID)
	assert.Equal(t, 1, requestCount, "Second call should use cache, not hit server")

	// Third call - cache hit, should NOT hit server
	result3, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result3)
	assert.Equal(t, "gpt-4o", result3.Smart.ModelID)
	assert.Equal(t, 1, requestCount, "Third call should use cache, not hit server")

	// Verify all results are consistent
	assert.Equal(t, result1.Fast.ModelID, result2.Fast.ModelID)
	assert.Equal(t, result1.Smart.ModelID, result3.Smart.ModelID)
}

func TestDefaultsClient_Cache_Expiration(t *testing.T) {
	requestCount := 0
	defaults := DefaultModelConfig{
		Fast: &ModelDefault{
			ModelID:  "gpt-4o-mini",
			Provider: "Azure",
			Creator:  "openai",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(defaults)
	}))
	defer server.Close()

	client := NewDefaultsClient(server.URL)
	// Override cache TTL for testing
	client.cacheTTL = 100 * time.Millisecond
	ctx := context.Background()

	// First call - cache miss
	result1, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result1)
	assert.Equal(t, 1, requestCount, "First call should hit the server")

	// Second call immediately - cache hit
	result2, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result2)
	assert.Equal(t, 1, requestCount, "Second call should use cache")

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call after expiration - cache miss
	result3, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result3)
	assert.Equal(t, 2, requestCount, "Third call should hit the server after cache expiration")

	// Verify all results are consistent
	assert.Equal(t, result1.Fast.ModelID, result2.Fast.ModelID)
	assert.Equal(t, result1.Fast.ModelID, result3.Fast.ModelID)
}

func TestDefaultsClient_Cache_ErrorDoesNotCache(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Return error on first call
		if requestCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Return success on subsequent calls
		defaults := DefaultModelConfig{
			Fast: &ModelDefault{ModelID: "gpt-4o-mini"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(defaults)
	}))
	defer server.Close()

	client := NewDefaultsClient(server.URL)
	ctx := context.Background()

	// First call - should fail, should not cache
	_, err := client.GetDefaults(ctx)
	require.Error(t, err)
	assert.Equal(t, 1, requestCount)

	// Second call - should retry and succeed
	result, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, requestCount, "Should retry after error, not use cache")

	// Third call - should use cache from successful call
	result2, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	require.NotNil(t, result2)
	assert.Equal(t, 2, requestCount, "Should use cache after successful call")
}

// Cache Tests - MappingClient

func TestMappingClient_Cache_HitAndMiss(t *testing.T) {
	requestCount := 0
	models := []infra.ModelConfig{
		{
			ModelId:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
			ModelMapping: "claude-3-5-sonnet",
			Endpoint:     "https://bedrock-runtime.us-east-1.amazonaws.com",
			TargetApi:    "converse",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models)
	}))
	defer server.Close()

	client := NewMappingClient(server.URL)
	ctx := context.Background()

	// First call - cache miss
	result1, err := client.GetModels(ctx)
	require.NoError(t, err)
	require.Len(t, result1, 1)
	assert.Equal(t, 1, requestCount, "First call should hit the server")

	// Second call - cache hit
	result2, err := client.GetModels(ctx)
	require.NoError(t, err)
	require.Len(t, result2, 1)
	assert.Equal(t, 1, requestCount, "Second call should use cache")

	// Verify results are consistent
	assert.Equal(t, result1[0].ModelMapping, result2[0].ModelMapping)
}

func TestMappingClient_Cache_Expiration(t *testing.T) {
	requestCount := 0
	models := []infra.ModelConfig{
		{
			ModelId:      "anthropic.claude-3-5-sonnet-20241022-v2:0",
			ModelMapping: "claude-3-5-sonnet",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models)
	}))
	defer server.Close()

	client := NewMappingClient(server.URL)
	// Override cache TTL for testing
	client.cacheTTL = 100 * time.Millisecond
	ctx := context.Background()

	// First call - cache miss
	_, err := client.GetModels(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, requestCount)

	// Second call immediately - cache hit
	_, err = client.GetModels(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, requestCount)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call after expiration - cache miss
	_, err = client.GetModels(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, requestCount, "Should hit server after cache expiration")
}

// Cache Tests - Private Models

func TestPrivateModels_Cache_HitAndMiss(t *testing.T) {
	// Create temporary directory for private models
	tempDir := t.TempDir()

	// Create a private model file
	privateModel := api.Mapping{
		Models: []api.Model{
			{
				Name:           "private-gpt-4o",
				Infrastructure: "azure",
				Provider:       "azure",
				Creator:        "openai",
				ModelId:        "private-gpt-4o-deployment",
				RedirectURL:    "https://private-azure.openai.azure.com",
				Active:         true,
			},
		},
	}

	data, err := json.Marshal(privateModel)
	require.NoError(t, err)

	privateFile := tempDir + "/private-model-test.yaml"
	err = os.WriteFile(privateFile, data, 0644)
	require.NoError(t, err)

	resolver := &TargetResolver{
		privateModelDir: tempDir,
		cacheTTL:        5 * time.Minute,
	}

	// First call - cache miss, should load from disk
	model1, found1, err := resolver.checkPrivateModels(context.Background(), "private-gpt-4o")
	require.NoError(t, err)
	assert.True(t, found1)
	require.NotNil(t, model1)
	assert.Equal(t, "private-gpt-4o", model1.Name)

	// Modify file to verify cache is used (not reloaded from disk)
	err = os.Remove(privateFile)
	require.NoError(t, err)

	// Second call - cache hit, should not load from disk (file is deleted)
	model2, found2, err := resolver.checkPrivateModels(context.Background(), "private-gpt-4o")
	require.NoError(t, err)
	assert.True(t, found2, "Should find model in cache even though file is deleted")
	require.NotNil(t, model2)
	assert.Equal(t, "private-gpt-4o", model2.Name)

	// Verify both models are the same
	assert.Equal(t, model1.Name, model2.Name)
	assert.Equal(t, model1.ModelId, model2.ModelId)
}

func TestPrivateModels_Cache_Expiration(t *testing.T) {
	// Create temporary directory for private models
	tempDir := t.TempDir()

	// Create initial private model file
	privateModel1 := api.Mapping{
		Models: []api.Model{
			{
				Name:    "private-model-v1",
				Active:  true,
				ModelId: "model-v1",
			},
		},
	}

	data1, err := json.Marshal(privateModel1)
	require.NoError(t, err)

	privateFile := tempDir + "/private-model-test.yaml"
	err = os.WriteFile(privateFile, data1, 0644)
	require.NoError(t, err)

	resolver := &TargetResolver{
		privateModelDir: tempDir,
		cacheTTL:        100 * time.Millisecond, // Short TTL for testing
	}

	// First call - cache miss, load v1
	model1, found1, err := resolver.checkPrivateModels(context.Background(), "private-model-v1")
	require.NoError(t, err)
	assert.True(t, found1)
	require.NotNil(t, model1)
	assert.Equal(t, "private-model-v1", model1.Name)
	assert.Equal(t, "model-v1", model1.ModelId)

	// Second call immediately - cache hit, still v1
	model2, found2, err := resolver.checkPrivateModels(context.Background(), "private-model-v1")
	require.NoError(t, err)
	assert.True(t, found2)
	require.NotNil(t, model2)
	assert.Equal(t, "model-v1", model2.ModelId)

	// Update file with new version
	privateModel2 := api.Mapping{
		Models: []api.Model{
			{
				Name:    "private-model-v1",
				Active:  true,
				ModelId: "model-v2-updated", // Changed
			},
		},
	}

	data2, err := json.Marshal(privateModel2)
	require.NoError(t, err)
	err = os.WriteFile(privateFile, data2, 0644)
	require.NoError(t, err)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call after expiration - cache miss, should reload and get v2
	model3, found3, err := resolver.checkPrivateModels(context.Background(), "private-model-v1")
	require.NoError(t, err)
	assert.True(t, found3)
	require.NotNil(t, model3)
	assert.Equal(t, "model-v2-updated", model3.ModelId, "Should reload from disk after cache expiration")
}

func TestPrivateModels_Cache_NotFoundStillCached(t *testing.T) {
	// Create temporary directory with no private models
	tempDir := t.TempDir()

	resolver := &TargetResolver{
		privateModelDir: tempDir,
		cacheTTL:        5 * time.Minute,
	}

	// First call - cache miss, model not found
	model1, found1, err := resolver.checkPrivateModels(context.Background(), "nonexistent-model")
	require.NoError(t, err)
	assert.False(t, found1)
	assert.Nil(t, model1)

	// Add a model file
	privateModel := api.Mapping{
		Models: []api.Model{
			{
				Name:   "nonexistent-model",
				Active: true,
			},
		},
	}
	data, err := json.Marshal(privateModel)
	require.NoError(t, err)
	privateFile := tempDir + "/private-model-new.yaml"
	err = os.WriteFile(privateFile, data, 0644)
	require.NoError(t, err)

	// Second call immediately - cache hit, should still not find (cache contains empty result)
	model2, found2, err := resolver.checkPrivateModels(context.Background(), "nonexistent-model")
	require.NoError(t, err)
	assert.False(t, found2, "Should use cached empty result even though file now exists")
	assert.Nil(t, model2)
}

func TestPrivateModels_Cache_ConcurrentAccess(t *testing.T) {
	// Create temporary directory for private models
	tempDir := t.TempDir()

	// Create a private model file
	privateModel := api.Mapping{
		Models: []api.Model{
			{
				Name:   "concurrent-model",
				Active: true,
			},
		},
	}

	data, err := json.Marshal(privateModel)
	require.NoError(t, err)

	privateFile := tempDir + "/private-model-concurrent.yaml"
	err = os.WriteFile(privateFile, data, 0644)
	require.NoError(t, err)

	resolver := &TargetResolver{
		privateModelDir: tempDir,
		cacheTTL:        5 * time.Minute,
	}

	// Test concurrent access to verify thread safety
	const numGoroutines = 50
	results := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			model, found, err := resolver.checkPrivateModels(context.Background(), "concurrent-model")
			if err != nil {
				errors <- err
				return
			}
			if found && model != nil && model.Name == "concurrent-model" {
				results <- true
			} else {
				results <- false
			}
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Fatalf("Unexpected error in concurrent access: %v", err)
		case success := <-results:
			if success {
				successCount++
			}
		}
	}

	assert.Equal(t, numGoroutines, successCount, "All concurrent accesses should succeed")
}

func TestDefaultsClient_Cache_ConcurrentAccess(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex
	defaults := DefaultModelConfig{
		Fast: &ModelDefault{ModelID: "gpt-4o-mini"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(defaults)
	}))
	defer server.Close()

	client := NewDefaultsClient(server.URL)
	ctx := context.Background()

	// Prime the cache with one call
	_, err := client.GetDefaults(ctx)
	require.NoError(t, err)
	mu.Lock()
	initialCount := requestCount
	mu.Unlock()
	assert.Equal(t, 1, initialCount, "First call should hit server")

	// Now test concurrent access - all should use cache
	const numGoroutines = 50
	results := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			result, err := client.GetDefaults(ctx)
			if err != nil {
				errors <- err
				return
			}
			if result != nil && result.Fast != nil && result.Fast.ModelID == "gpt-4o-mini" {
				results <- true
			} else {
				results <- false
			}
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			t.Fatalf("Unexpected error in concurrent access: %v", err)
		case success := <-results:
			if success {
				successCount++
			}
		}
	}

	assert.Equal(t, numGoroutines, successCount, "All concurrent accesses should succeed")

	// Verify cache was used - server should not be called again after priming
	mu.Lock()
	finalCount := requestCount
	mu.Unlock()
	assert.Equal(t, initialCount, finalCount, "Server should not be called again after cache is primed")
}
