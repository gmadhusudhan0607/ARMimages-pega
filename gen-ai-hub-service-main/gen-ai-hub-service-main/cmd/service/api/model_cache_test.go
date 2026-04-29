/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// setupMockProviderVars saves all provider-level vars, installs test-friendly
// mocks and returns a restore function that MUST be deferred by the caller.
func setupMockProviderVars(t *testing.T, awsModels []ModelInfo) func() {
	t.Helper()

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		return awsModels, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) {
		return []ModelInfo{}, nil
	}
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo {
		return models // pass-through
	}
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo {
		return models // pass-through
	}

	return func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}
}

// newTestCheckerForCache creates a MockContextChecker that enables the Bedrock
// provider only and returns the provided logger for any context.
func newTestCheckerForCache(t *testing.T) *MockContextChecker {
	t.Helper()

	logger := zaptest.NewLogger(t)
	checker := new(MockContextChecker)
	checker.On("IsUseGenAiInfraModels", mock.Anything).Return(true)
	checker.On("IsUseGCPVertex", mock.Anything).Return(false)
	checker.On("IsUseAzureGenAIURL", mock.Anything).Return(false)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Bedrock").Return(true)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Vertex").Return(false)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Azure").Return(false)
	checker.On("LoggerFromContext", mock.Anything).Return(logger)
	checker.On("ContextWithGinContext", mock.Anything, mock.Anything).
		Return(context.Background())

	return checker
}

func TestNewModelListCache(t *testing.T) {
	ctx := cntx.NewTestContext("test-new-cache")
	checker := newTestCheckerForCache(t)

	cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

	require.NotNil(t, cache, "NewModelListCache should return a non-nil cache")
	assert.NotNil(t, cache.logger, "logger should be initialised")
	assert.Nil(t, cache.snap.Load(), "cache should not be populated initially")
	assert.Equal(t, DefaultCacheTTL, cache.ttl, "TTL should match the provided value")
}

func TestModelListCache_GetModels_PopulatesOnFirstCall(t *testing.T) {
	ctx := cntx.NewTestContext("test-lazy-populate")
	checker := newTestCheckerForCache(t)

	testModels := []ModelInfo{
		{ModelName: "model-a", Provider: "bedrock", ModelPath: []string{"/a"}},
		{ModelName: "model-b", Provider: "bedrock", ModelPath: []string{"/b"}},
	}

	restore := setupMockProviderVars(t, testModels)
	defer restore()

	cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

	// Cache should not be populated yet.
	assert.Nil(t, cache.snap.Load())

	// First call triggers population.
	models, warnings := cache.GetModels(ctx)

	require.Len(t, models, 2, "first call should populate and return models")
	assert.Equal(t, "model-a", models[0].ModelName)
	assert.Equal(t, "model-b", models[1].ModelName)
	assert.Empty(t, warnings)
	assert.NotNil(t, cache.snap.Load(), "cache should be marked as populated")
}

func TestModelListCache_GetModels_ReturnsCachedOnSubsequentCalls(t *testing.T) {
	ctx := cntx.NewTestContext("test-cached-subsequent")
	checker := newTestCheckerForCache(t)

	callCount := 0
	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		callCount++
		return []ModelInfo{
			{ModelName: "counted-model", Provider: "bedrock", ModelPath: []string{"/c"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

	// First call populates.
	models, _ := cache.GetModels(ctx)
	require.Len(t, models, 1)
	assert.Equal(t, 1, callCount, "provider should be called once on first GetModels")

	// Second and third calls return cached data without calling providers again.
	models, _ = cache.GetModels(ctx)
	require.Len(t, models, 1)
	models, _ = cache.GetModels(ctx)
	require.Len(t, models, 1)
	assert.Equal(t, 1, callCount, "provider should not be called again for cached data")
}

func TestModelListCache_GetModels_DefensiveCopy(t *testing.T) {
	ctx := cntx.NewTestContext("test-defensive-copy")
	checker := newTestCheckerForCache(t)

	testModels := []ModelInfo{
		{ModelName: "original-model", Provider: "bedrock", ModelPath: []string{"/orig"}},
	}

	restore := setupMockProviderVars(t, testModels)
	defer restore()

	cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

	// Retrieve and mutate the returned slice.
	models, _ := cache.GetModels(ctx)
	require.Len(t, models, 1)
	models[0].ModelName = "mutated-model"

	// A second retrieval must still show the original data.
	modelsAgain, _ := cache.GetModels(ctx)
	require.Len(t, modelsAgain, 1)
	assert.Equal(t, "original-model", modelsAgain[0].ModelName,
		"mutating the returned slice must not affect the cached data")
}

func TestModelListCache_GetModels_ConcurrentAccess(t *testing.T) {
	ctx := cntx.NewTestContext("test-concurrent")
	checker := newTestCheckerForCache(t)

	var mu sync.Mutex
	callCount := 0
	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return []ModelInfo{
			{ModelName: "concurrent-model", Provider: "bedrock", ModelPath: []string{"/c"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

	// Launch multiple goroutines that all call GetModels concurrently.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			models, _ := cache.GetModels(ctx)
			assert.Len(t, models, 1)
			assert.Equal(t, "concurrent-model", models[0].ModelName)
		}()
	}
	wg.Wait()

	// The provider should have been called exactly once despite concurrent access.
	mu.Lock()
	count := callCount
	mu.Unlock()
	assert.Equal(t, 1, count, "populate should run exactly once under concurrent access")
}

func TestModelListCache_GetModels_EmptyProviders(t *testing.T) {
	ctx := cntx.NewTestContext("test-empty-providers")
	checker := newTestCheckerForCache(t)

	restore := setupMockProviderVars(t, []ModelInfo{})
	defer restore()

	cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

	models, warnings := cache.GetModels(ctx)

	require.NotNil(t, models, "models slice should be non-nil even when empty")
	assert.Empty(t, models)
	require.NotNil(t, warnings, "warnings slice should be non-nil even when empty")
	assert.Empty(t, warnings)
}

func TestModelListCache_GetModels_WithWarnings(t *testing.T) {
	ctx := cntx.NewTestContext("test-populate-warnings")

	logger := zaptest.NewLogger(t)
	checker := new(MockContextChecker)
	checker.On("IsUseGenAiInfraModels", mock.Anything).Return(true)
	checker.On("IsUseGCPVertex", mock.Anything).Return(false)
	checker.On("IsUseAzureGenAIURL", mock.Anything).Return(false)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Bedrock").Return(true)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Vertex").Return(false)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Azure").Return(false)
	checker.On("LoggerFromContext", mock.Anything).Return(logger)

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		return nil, errors.New("simulated bedrock failure")
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

	models, warnings := cache.GetModels(ctx)

	assert.Empty(t, models, "models should be empty when the only enabled provider fails")
	require.NotEmpty(t, warnings, "warnings should contain the provider error message")
	assert.Contains(t, warnings[0], "Failed to fetch models")
}

func TestBuildModelsResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("without warnings returns plain array", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/models", nil)

		models := []ModelInfo{
			{ModelName: "model-a", Provider: "bedrock", ModelID: "a-id"},
			{ModelName: "model-b", Provider: "vertex", ModelID: "b-id"},
		}

		buildModelsResponse(c, models, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var arr []ModelInfo
		err := json.Unmarshal(w.Body.Bytes(), &arr)
		require.NoError(t, err, "response body should be a JSON array")
		require.Len(t, arr, 2)
		assert.Equal(t, "model-a", arr[0].ModelName)
		assert.Equal(t, "model-b", arr[1].ModelName)
	})

	t.Run("with warnings returns envelope", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/models", nil)

		models := []ModelInfo{
			{ModelName: "model-a", Provider: "bedrock", ModelID: "a-id"},
		}
		warnings := []string{"provider-x had an error"}

		buildModelsResponse(c, models, warnings)

		assert.Equal(t, http.StatusOK, w.Code)

		var envelope struct {
			Models   []ModelInfo `json:"models"`
			Warnings []string    `json:"warnings"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &envelope)
		require.NoError(t, err, "response body should be a JSON object with models and warnings")
		require.Len(t, envelope.Models, 1)
		assert.Equal(t, "model-a", envelope.Models[0].ModelName)
		require.Len(t, envelope.Warnings, 1)
		assert.Equal(t, "provider-x had an error", envelope.Warnings[0])
	})

	t.Run("empty models without warnings returns empty array", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/models", nil)

		buildModelsResponse(c, []ModelInfo{}, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		var arr []ModelInfo
		err := json.Unmarshal(w.Body.Bytes(), &arr)
		require.NoError(t, err)
		assert.Empty(t, arr)
	})
}

func TestHandleCachedGetModelsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("populates cache on first request and returns models", func(t *testing.T) {
		ctx := cntx.NewTestContext("test-cached-models-no-warnings")
		checker := newTestCheckerForCache(t)

		testModels := []ModelInfo{
			{ModelName: "cached-a", Provider: "bedrock", ModelID: "id-a", ModelPath: []string{"/a"}},
			{ModelName: "cached-b", Provider: "bedrock", ModelID: "id-b", ModelPath: []string{"/b"}},
		}

		restore := setupMockProviderVars(t, testModels)
		defer restore()

		cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

		router := gin.New()
		router.GET("/models", HandleCachedGetModelsRequest(ctx, checker, cache))

		req := httptest.NewRequest(http.MethodGet, "/models", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var arr []ModelInfo
		err := json.Unmarshal(resp.Body.Bytes(), &arr)
		require.NoError(t, err, "response should be a JSON array when no warnings")
		require.Len(t, arr, 2)
		assert.Equal(t, "cached-a", arr[0].ModelName)
		assert.Equal(t, "cached-b", arr[1].ModelName)
	})

	t.Run("with warnings returns envelope", func(t *testing.T) {
		ctx := cntx.NewTestContext("test-cached-models-with-warnings")

		logger := zaptest.NewLogger(t)
		checker := new(MockContextChecker)
		checker.On("IsUseGenAiInfraModels", mock.Anything).Return(true)
		checker.On("IsUseGCPVertex", mock.Anything).Return(true)
		checker.On("IsUseAzureGenAIURL", mock.Anything).Return(false)
		checker.On("IsLLMProviderConfigured", mock.Anything, "Bedrock").Return(true)
		checker.On("IsLLMProviderConfigured", mock.Anything, "Vertex").Return(true)
		checker.On("IsLLMProviderConfigured", mock.Anything, "Azure").Return(false)
		checker.On("LoggerFromContext", mock.Anything).Return(logger)
		checker.On("ContextWithGinContext", mock.Anything, mock.Anything).
			Return(context.Background())

		origFetchAWS := fetchAWSModels
		origFetchGCP := fetchGCPModels
		origFetchAzure := fetchAzureModels
		origDeduplicate := deduplicateModels
		origEnrich := enrichModels
		defer func() {
			fetchAWSModels = origFetchAWS
			fetchGCPModels = origFetchGCP
			fetchAzureModels = origFetchAzure
			deduplicateModels = origDeduplicate
			enrichModels = origEnrich
		}()

		fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
			return []ModelInfo{
				{ModelName: "aws-model", Provider: "bedrock", ModelID: "aws-id"},
			}, nil
		}
		fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) {
			return nil, errors.New("vertex unavailable")
		}
		fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
			return []ModelInfo{}, 200, nil
		}
		deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
		enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

		cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

		router := gin.New()
		router.GET("/models", HandleCachedGetModelsRequest(ctx, checker, cache))

		req := httptest.NewRequest(http.MethodGet, "/models", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var envelope struct {
			Models   []ModelInfo `json:"models"`
			Warnings []string    `json:"warnings"`
		}
		err := json.Unmarshal(resp.Body.Bytes(), &envelope)
		require.NoError(t, err, "response should be a JSON envelope when warnings exist")
		require.Len(t, envelope.Models, 1)
		assert.Equal(t, "aws-model", envelope.Models[0].ModelName)
		require.NotEmpty(t, envelope.Warnings)
		assert.Contains(t, envelope.Warnings[0], "Failed to fetch models")
	})

	t.Run("empty providers returns empty array", func(t *testing.T) {
		ctx := cntx.NewTestContext("test-cached-models-empty")
		checker := newTestCheckerForCache(t)

		restore := setupMockProviderVars(t, []ModelInfo{})
		defer restore()

		cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

		router := gin.New()
		router.GET("/models", HandleCachedGetModelsRequest(ctx, checker, cache))

		req := httptest.NewRequest(http.MethodGet, "/models", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var arr []ModelInfo
		err := json.Unmarshal(resp.Body.Bytes(), &arr)
		require.NoError(t, err)
		assert.Empty(t, arr)
	})
}

func TestHandleCachedGetDefaultModelsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success returns fast and smart defaults", func(t *testing.T) {
		ctx := cntx.NewTestContext("test-cached-defaults-success")
		checker := newTestCheckerForCache(t)

		testModels := []ModelInfo{
			{ModelName: "fast-model", ModelMappingId: "fast-mapping", Provider: "bedrock", ModelID: "fast-id", Creator: "openai", ModelPath: []string{"/f"}},
			{ModelName: "smart-model", ModelMappingId: "smart-mapping", Provider: "bedrock", ModelID: "smart-id", Creator: "anthropic", ModelPath: []string{"/s"}},
			{ModelName: "other-model", ModelMappingId: "other-mapping", Provider: "bedrock", ModelID: "other-id", Creator: "meta", ModelPath: []string{"/o"}},
		}

		restore := setupMockProviderVars(t, testModels)
		defer restore()

		origGetDefaults := getDefaults
		origExtract := extractDefaultModels
		defer func() {
			getDefaults = origGetDefaults
			extractDefaultModels = origExtract
		}()

		t.Setenv("SMART_MODEL_OVERRIDE", "smart-mapping")
		t.Setenv("FAST_MODEL_OVERRIDE", "fast-mapping")
		t.Setenv("PRO_MODEL_OVERRIDE", "pro-mapping")
		t.Setenv("ENABLE_PRO_MODEL_DEFAULT", "false")

		extractDefaultModels = extractDefaultModelsImpl

		cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

		router := gin.New()
		router.GET("/models/defaults", HandleCachedGetDefaultModelsRequest(ctx, checker, cache))

		req := httptest.NewRequest(http.MethodGet, "/models/defaults", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var result DefaultModels
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		require.NotNil(t, result.Fast, "fast default should be present")
		assert.Equal(t, "fast-model", result.Fast.ModelName)
		assert.Equal(t, "fast-id", result.Fast.ModelID)

		require.NotNil(t, result.Smart, "smart default should be present")
		assert.Equal(t, "smart-model", result.Smart.ModelName)
		assert.Equal(t, "smart-id", result.Smart.ModelID)

		assert.Nil(t, result.Pro, "pro should be nil when ENABLE_PRO_MODEL_DEFAULT is false")
	})

	t.Run("success with pro model enabled", func(t *testing.T) {
		ctx := cntx.NewTestContext("test-cached-defaults-with-pro")
		checker := newTestCheckerForCache(t)

		testModels := []ModelInfo{
			{ModelName: "fast-model", ModelMappingId: "fast-mapping", Provider: "bedrock", ModelID: "fast-id", Creator: "openai", ModelPath: []string{"/f"}},
			{ModelName: "smart-model", ModelMappingId: "smart-mapping", Provider: "bedrock", ModelID: "smart-id", Creator: "anthropic", ModelPath: []string{"/s"}},
			{ModelName: "pro-model", ModelMappingId: "pro-mapping", Provider: "bedrock", ModelID: "pro-id", Creator: "anthropic", ModelPath: []string{"/p"}},
		}

		restore := setupMockProviderVars(t, testModels)
		defer restore()

		origGetDefaults := getDefaults
		origExtract := extractDefaultModels
		defer func() {
			getDefaults = origGetDefaults
			extractDefaultModels = origExtract
		}()

		t.Setenv("SMART_MODEL_OVERRIDE", "smart-mapping")
		t.Setenv("FAST_MODEL_OVERRIDE", "fast-mapping")
		t.Setenv("PRO_MODEL_OVERRIDE", "pro-mapping")
		t.Setenv("ENABLE_PRO_MODEL_DEFAULT", "true")

		extractDefaultModels = extractDefaultModelsImpl

		cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

		router := gin.New()
		router.GET("/models/defaults", HandleCachedGetDefaultModelsRequest(ctx, checker, cache))

		req := httptest.NewRequest(http.MethodGet, "/models/defaults", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var result DefaultModels
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		require.NotNil(t, result.Fast, "fast should be present")
		assert.Equal(t, "fast-model", result.Fast.ModelName)

		require.NotNil(t, result.Smart, "smart should be present")
		assert.Equal(t, "smart-model", result.Smart.ModelName)

		require.NotNil(t, result.Pro, "pro should be present when ENABLE_PRO_MODEL_DEFAULT is true")
		assert.Equal(t, "pro-model", result.Pro.ModelName)
		assert.Equal(t, "pro-id", result.Pro.ModelID)
	})

	t.Run("defaults fetch error returns 500", func(t *testing.T) {
		ctx := cntx.NewTestContext("test-cached-defaults-error")
		checker := newTestCheckerForCache(t)

		restore := setupMockProviderVars(t, []ModelInfo{})
		defer restore()

		origGetDefaults := getDefaults
		defer func() { getDefaults = origGetDefaults }()

		t.Setenv("SMART_MODEL_OVERRIDE", "")
		t.Setenv("FAST_MODEL_OVERRIDE", "")
		t.Setenv("PRO_MODEL_OVERRIDE", "")

		getDefaults = func(_ context.Context) (infra.DefaultModelConfig, error) {
			return infra.DefaultModelConfig{}, errors.New("ops endpoint unreachable")
		}

		cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

		router := gin.New()
		router.GET("/models/defaults", HandleCachedGetDefaultModelsRequest(ctx, checker, cache))

		req := httptest.NewRequest(http.MethodGet, "/models/defaults", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusInternalServerError, resp.Code)

		var errResp map[string]string
		err := json.Unmarshal(resp.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp["error"], "ops endpoint unreachable")
	})

	t.Run("no matching defaults returns empty defaults", func(t *testing.T) {
		ctx := cntx.NewTestContext("test-cached-defaults-no-match")
		checker := newTestCheckerForCache(t)

		testModels := []ModelInfo{
			{ModelName: "unrelated-model", ModelMappingId: "unrelated", Provider: "bedrock", ModelPath: []string{"/u"}},
		}

		restore := setupMockProviderVars(t, testModels)
		defer restore()

		origGetDefaults := getDefaults
		origExtract := extractDefaultModels
		defer func() {
			getDefaults = origGetDefaults
			extractDefaultModels = origExtract
		}()

		t.Setenv("SMART_MODEL_OVERRIDE", "missing-smart")
		t.Setenv("FAST_MODEL_OVERRIDE", "missing-fast")
		t.Setenv("PRO_MODEL_OVERRIDE", "missing-pro")
		t.Setenv("ENABLE_PRO_MODEL_DEFAULT", "false")

		extractDefaultModels = extractDefaultModelsImpl

		cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

		router := gin.New()
		router.GET("/models/defaults", HandleCachedGetDefaultModelsRequest(ctx, checker, cache))

		req := httptest.NewRequest(http.MethodGet, "/models/defaults", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)

		var result DefaultModels
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)

		assert.Nil(t, result.Fast, "fast should be nil when no match found")
		assert.Nil(t, result.Smart, "smart should be nil when no match found")
		assert.Nil(t, result.Pro, "pro should be nil when no match found")
	})
}

func TestCollectModelsFromProviders_Parallel(t *testing.T) {
	ctx := cntx.NewTestContext("test-parallel-providers")
	checker := newAllProvidersChecker(t)

	// Track execution order using a channel to prove concurrency.
	started := make(chan string, 3)

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		started <- "bedrock"
		return []ModelInfo{{ModelName: "bedrock-model", Provider: "bedrock"}}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) {
		started <- "vertex"
		return []ModelInfo{{ModelName: "vertex-model", Provider: "vertex"}}, nil
	}
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		started <- "azure"
		return []ModelInfo{{ModelName: "azure-model", Provider: "azure"}}, 200, nil
	}

	models, warnings := collectModelsFromProviders(ctx, checker)

	assert.Empty(t, warnings)
	require.Len(t, models, 3, "all three providers should contribute models")

	// Verify deterministic ordering: Bedrock, Vertex, Azure.
	assert.Equal(t, "bedrock-model", models[0].ModelName)
	assert.Equal(t, "vertex-model", models[1].ModelName)
	assert.Equal(t, "azure-model", models[2].ModelName)

	// Drain the channel — all three should have started.
	close(started)
	var providers []string
	for p := range started {
		providers = append(providers, p)
	}
	assert.Len(t, providers, 3, "all three providers should have been invoked")
}

func TestCollectModelsFromProviders_PartialFailure(t *testing.T) {
	ctx := cntx.NewTestContext("test-parallel-partial-failure")
	checker := newAllProvidersChecker(t)

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		return []ModelInfo{{ModelName: "aws-ok", Provider: "bedrock"}}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) {
		return nil, errors.New("vertex timeout")
	}
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{{ModelName: "azure-ok", Provider: "azure"}}, 200, nil
	}

	models, warnings := collectModelsFromProviders(ctx, checker)

	assert.Len(t, models, 2, "successful providers should still return models")
	require.Len(t, warnings, 1, "failed provider should produce a warning")
	assert.Contains(t, warnings[0], "Failed to fetch models")
}

func TestModelListCache_GetModels_PopulateTimeout(t *testing.T) {
	ctx := cntx.NewTestContext("test-populate-timeout")
	checker := newTestCheckerForCache(t)

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	// The AWS fetch blocks until the context is cancelled. This verifies
	// that the PopulateTimeout wraps the populate call with a deadline.
	fetchAWSModels = func(ctx context.Context) ([]ModelInfo, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	cache := NewModelListCache(ctx, checker, DefaultCacheTTL)

	// Use a very short deadline on the parent context to make the test fast.
	shortCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	models, warnings := cache.GetModels(shortCtx)

	// The cache should still be marked populated (even with zero models)
	// so that subsequent requests don't re-trigger population.
	assert.NotNil(t, cache.snap.Load(), "cache should be marked populated after timeout")
	assert.NotNil(t, models, "models slice should be non-nil")
	require.NotEmpty(t, warnings, "timeout should surface as a warning")
}

func TestModelListCache_TTL_ExpiryTriggersRepopulation(t *testing.T) {
	ctx := cntx.NewTestContext("test-ttl-expiry")
	checker := newTestCheckerForCache(t)

	var mu sync.Mutex
	callCount := 0
	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		mu.Lock()
		callCount++
		count := callCount
		mu.Unlock()
		return []ModelInfo{
			{ModelName: fmt.Sprintf("model-v%d", count), Provider: "bedrock", ModelPath: []string{"/a"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	// Use a very short TTL so it expires quickly.
	cache := NewModelListCache(ctx, checker, 50*time.Millisecond)

	// First call populates.
	models, _ := cache.GetModels(ctx)
	require.Len(t, models, 1)
	assert.Equal(t, "model-v1", models[0].ModelName)

	mu.Lock()
	assert.Equal(t, 1, callCount, "should have populated once")
	mu.Unlock()

	// Before TTL expires: should serve cached data.
	models, _ = cache.GetModels(ctx)
	assert.Equal(t, "model-v1", models[0].ModelName)
	mu.Lock()
	assert.Equal(t, 1, callCount, "should still be 1 — cache not expired")
	mu.Unlock()

	// Wait for TTL to expire.
	time.Sleep(60 * time.Millisecond)

	// Next call should trigger re-population with fresh data.
	models, _ = cache.GetModels(ctx)
	require.Len(t, models, 1)
	assert.Equal(t, "model-v2", models[0].ModelName, "should have re-populated with new data")

	mu.Lock()
	assert.Equal(t, 2, callCount, "provider should have been called twice after TTL expiry")
	mu.Unlock()
}

func TestModelListCache_TTL_ConcurrentReadersBlockUntilFreshData(t *testing.T) {
	ctx := cntx.NewTestContext("test-ttl-block-until-fresh")
	checker := newTestCheckerForCache(t)

	populateStarted := make(chan struct{})
	populateFinish := make(chan struct{})

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	callCount := 0
	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		callCount++
		if callCount > 1 {
			// Second population: signal that we started and wait to be released.
			close(populateStarted)
			<-populateFinish
		}
		return []ModelInfo{
			{ModelName: fmt.Sprintf("model-v%d", callCount), Provider: "bedrock", ModelPath: []string{"/a"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	cache := NewModelListCache(ctx, checker, 50*time.Millisecond)

	// Populate the cache initially.
	models, _ := cache.GetModels(ctx)
	assert.Equal(t, "model-v1", models[0].ModelName)

	// Wait for TTL to expire.
	time.Sleep(60 * time.Millisecond)

	// Start re-population in a goroutine (it will block on populateFinish).
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		cache.GetModels(ctx)
	}()

	// Wait for re-population to start.
	<-populateStarted

	// A concurrent reader should block until re-population completes,
	// then receive the fresh data — never the stale v1 snapshot.
	readerDone := make(chan struct{})
	var readerModels []ModelInfo
	go func() {
		readerModels, _ = cache.GetModels(ctx)
		close(readerDone)
	}()

	// Give the reader goroutine time to reach mu.Lock() and block.
	time.Sleep(20 * time.Millisecond)
	select {
	case <-readerDone:
		t.Fatal("concurrent reader returned before re-population finished — should have blocked")
	default:
		// Expected: reader is still blocked.
	}

	// Release the re-population.
	close(populateFinish)
	wg.Wait()
	<-readerDone

	assert.Equal(t, "model-v2", readerModels[0].ModelName,
		"concurrent reader should receive fresh data after blocking, not stale data")
}

func TestModelListCache_TTL_FailedProviderRecoversAfterExpiry(t *testing.T) {
	ctx := cntx.NewTestContext("test-ttl-recovery")

	var mu sync.Mutex
	callCount := 0

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	// AWS always succeeds.
	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		return []ModelInfo{
			{ModelName: "bedrock-model", Provider: "bedrock", ModelPath: []string{"/a"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	// Azure fails on first call, succeeds on second.
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		mu.Lock()
		callCount++
		count := callCount
		mu.Unlock()
		if count == 1 {
			return nil, http.StatusUnauthorized, fmt.Errorf("unauthorized")
		}
		return []ModelInfo{
			{ModelName: "azure-model", Provider: "azure", ModelPath: []string{"/b"}},
		}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	// Enable Azure provider.
	logger := zaptest.NewLogger(t)
	azureChecker := new(MockContextChecker)
	azureChecker.On("IsUseGenAiInfraModels", mock.Anything).Return(true)
	azureChecker.On("IsUseGCPVertex", mock.Anything).Return(false)
	azureChecker.On("IsUseAzureGenAIURL", mock.Anything).Return(true)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Bedrock").Return(true)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Vertex").Return(false)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Azure").Return(true)
	azureChecker.On("LoggerFromContext", mock.Anything).Return(logger)
	azureChecker.On("ContextWithGinContext", mock.Anything, mock.Anything).
		Return(context.Background())
	azureChecker.On("AzureGenAIURL", mock.Anything).Return("https://mock-azure.example.com")

	cache := NewModelListCache(ctx, azureChecker, 50*time.Millisecond)

	// First call: Azure fails.
	models, warnings := cache.GetModels(ctx)
	assert.Len(t, models, 1, "only bedrock model should be present")
	assert.Equal(t, "bedrock-model", models[0].ModelName)
	require.NotEmpty(t, warnings, "should have Azure warning")
	assert.Contains(t, warnings[0], "Azure")

	// Wait for TTL to expire.
	time.Sleep(60 * time.Millisecond)

	// Second call: Azure recovers.
	models, warnings = cache.GetModels(ctx)
	assert.Len(t, models, 2, "both bedrock and azure models should be present after recovery")
	assert.Empty(t, warnings, "no warnings after successful re-population")

	// Verify model names.
	names := []string{models[0].ModelName, models[1].ModelName}
	assert.Contains(t, names, "bedrock-model")
	assert.Contains(t, names, "azure-model")
}

func TestModelListCache_TTL_CustomTTLRespected(t *testing.T) {
	ctx := cntx.NewTestContext("test-ttl-custom")
	checker := newTestCheckerForCache(t)

	var mu sync.Mutex
	callCount := 0
	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return []ModelInfo{
			{ModelName: "model", Provider: "bedrock", ModelPath: []string{"/a"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	// Use a 200ms TTL.
	cache := NewModelListCache(ctx, checker, 200*time.Millisecond)

	// Populate.
	cache.GetModels(ctx)
	mu.Lock()
	assert.Equal(t, 1, callCount)
	mu.Unlock()

	// At 50ms: still cached.
	time.Sleep(50 * time.Millisecond)
	cache.GetModels(ctx)
	mu.Lock()
	assert.Equal(t, 1, callCount, "should not re-populate before TTL")
	mu.Unlock()

	// At 150ms: still cached (TTL is 200ms).
	time.Sleep(100 * time.Millisecond)
	cache.GetModels(ctx)
	mu.Lock()
	assert.Equal(t, 1, callCount, "should not re-populate at 150ms with 200ms TTL")
	mu.Unlock()

	// At 210ms: TTL expired.
	time.Sleep(60 * time.Millisecond)
	cache.GetModels(ctx)
	mu.Lock()
	assert.Equal(t, 2, callCount, "should re-populate after 200ms TTL expires")
	mu.Unlock()
}

func TestModelListCache_TTL_ConcurrentExpiredRequestsOnlyPopulateOnce(t *testing.T) {
	ctx := cntx.NewTestContext("test-ttl-concurrent-expiry")
	checker := newTestCheckerForCache(t)

	var mu sync.Mutex
	callCount := 0
	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		// Simulate a slow provider to ensure concurrent goroutines overlap.
		time.Sleep(50 * time.Millisecond)
		return []ModelInfo{
			{ModelName: "model", Provider: "bedrock", ModelPath: []string{"/a"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	cache := NewModelListCache(ctx, checker, 50*time.Millisecond)

	// Initial population.
	cache.GetModels(ctx)
	mu.Lock()
	assert.Equal(t, 1, callCount)
	mu.Unlock()

	// Wait for TTL to expire.
	time.Sleep(60 * time.Millisecond)

	// Launch 10 concurrent requests after TTL expiry.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			models, _ := cache.GetModels(ctx)
			assert.Len(t, models, 1, "every goroutine should receive models")
		}()
	}
	wg.Wait()

	// The provider should have been called at most twice (initial + one re-population).
	// Concurrent requests during re-population block and then read the fresh snapshot.
	mu.Lock()
	count := callCount
	mu.Unlock()
	assert.Equal(t, 2, count, "re-population should happen exactly once despite concurrent expired requests")
}

// ctxWithAuth returns a context that embeds a gin context whose request
// carries an Authorization header. This simulates an authenticated caller
// whose credentials might fix a provider failure (e.g. Azure 401).
func ctxWithAuth(parent context.Context) context.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(w)
	gc.Request = httptest.NewRequest(http.MethodGet, "/models", nil)
	gc.Request.Header.Set("Authorization", "Bearer test-token")
	return cntx.ContextWithGinContext(parent, gc)
}

func TestModelListCache_WarningsWithTokenTriggersRepopulation(t *testing.T) {
	ctx := cntx.NewTestContext("test-warnings-with-token")

	var mu sync.Mutex
	callCount := 0

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		return []ModelInfo{
			{ModelName: "bedrock-model", Provider: "bedrock", ModelPath: []string{"/a"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	// Azure fails on first call, succeeds on second.
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		mu.Lock()
		callCount++
		count := callCount
		mu.Unlock()
		if count == 1 {
			return nil, http.StatusUnauthorized, fmt.Errorf("unauthorized")
		}
		return []ModelInfo{
			{ModelName: "azure-model", Provider: "azure", ModelPath: []string{"/b"}},
		}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	logger := zaptest.NewLogger(t)
	azureChecker := new(MockContextChecker)
	azureChecker.On("IsUseGenAiInfraModels", mock.Anything).Return(true)
	azureChecker.On("IsUseGCPVertex", mock.Anything).Return(false)
	azureChecker.On("IsUseAzureGenAIURL", mock.Anything).Return(true)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Bedrock").Return(true)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Vertex").Return(false)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Azure").Return(true)
	azureChecker.On("LoggerFromContext", mock.Anything).Return(logger)
	azureChecker.On("ContextWithGinContext", mock.Anything, mock.Anything).
		Return(context.Background())
	azureChecker.On("AzureGenAIURL", mock.Anything).Return("https://mock-azure.example.com")

	// Use a long TTL so it won't expire during the test.
	cache := NewModelListCache(ctx, azureChecker, 10*time.Minute)

	// First call (no auth): Azure fails, cache has warnings.
	models, warnings := cache.GetModels(ctx)
	assert.Len(t, models, 1, "only bedrock model should be present")
	require.NotEmpty(t, warnings, "should have Azure warning")

	mu.Lock()
	assert.Equal(t, 1, callCount)
	mu.Unlock()

	// Second call WITH auth token: should trigger re-population despite TTL
	// not having expired, because cache has warnings.
	authCtx := ctxWithAuth(ctx)
	models, warnings = cache.GetModels(authCtx)
	assert.Len(t, models, 2, "both bedrock and azure models should be present after re-population")
	assert.Empty(t, warnings, "no warnings after successful re-population")

	mu.Lock()
	assert.Equal(t, 2, callCount, "provider should have been called twice")
	mu.Unlock()
}

func TestModelListCache_WarningsWithoutTokenStaysCached(t *testing.T) {
	ctx := cntx.NewTestContext("test-warnings-without-token")

	var mu sync.Mutex
	callCount := 0

	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		return []ModelInfo{
			{ModelName: "bedrock-model", Provider: "bedrock", ModelPath: []string{"/a"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return nil, http.StatusUnauthorized, fmt.Errorf("unauthorized")
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	logger := zaptest.NewLogger(t)
	azureChecker := new(MockContextChecker)
	azureChecker.On("IsUseGenAiInfraModels", mock.Anything).Return(true)
	azureChecker.On("IsUseGCPVertex", mock.Anything).Return(false)
	azureChecker.On("IsUseAzureGenAIURL", mock.Anything).Return(true)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Bedrock").Return(true)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Vertex").Return(false)
	azureChecker.On("IsLLMProviderConfigured", mock.Anything, "Azure").Return(true)
	azureChecker.On("LoggerFromContext", mock.Anything).Return(logger)
	azureChecker.On("ContextWithGinContext", mock.Anything, mock.Anything).
		Return(context.Background())
	azureChecker.On("AzureGenAIURL", mock.Anything).Return("https://mock-azure.example.com")

	// Use a long TTL so it won't expire during the test.
	cache := NewModelListCache(ctx, azureChecker, 10*time.Minute)

	// First call: Azure fails.
	models, warnings := cache.GetModels(ctx)
	assert.Len(t, models, 1)
	require.NotEmpty(t, warnings)

	mu.Lock()
	assert.Equal(t, 1, callCount)
	mu.Unlock()

	// Second call WITHOUT auth token: should serve cached data (no re-population).
	models, warnings = cache.GetModels(ctx)
	assert.Len(t, models, 1, "should still have only bedrock model")
	require.NotEmpty(t, warnings, "warnings should still be present")

	mu.Lock()
	assert.Equal(t, 1, callCount, "provider should NOT have been called again without auth token")
	mu.Unlock()
}

func TestModelListCache_NoWarningsWithTokenStaysCached(t *testing.T) {
	ctx := cntx.NewTestContext("test-no-warnings-with-token")
	checker := newTestCheckerForCache(t)

	var mu sync.Mutex
	callCount := 0
	origFetchAWS := fetchAWSModels
	origFetchGCP := fetchGCPModels
	origFetchAzure := fetchAzureModels
	origDeduplicate := deduplicateModels
	origEnrich := enrichModels
	defer func() {
		fetchAWSModels = origFetchAWS
		fetchGCPModels = origFetchGCP
		fetchAzureModels = origFetchAzure
		deduplicateModels = origDeduplicate
		enrichModels = origEnrich
	}()

	fetchAWSModels = func(_ context.Context) ([]ModelInfo, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return []ModelInfo{
			{ModelName: "bedrock-model", Provider: "bedrock", ModelPath: []string{"/a"}},
		}, nil
	}
	fetchGCPModels = func(_ context.Context) ([]ModelInfo, error) { return []ModelInfo{}, nil }
	fetchAzureModels = func(_ context.Context, _ string) ([]ModelInfo, int, error) {
		return []ModelInfo{}, 200, nil
	}
	deduplicateModels = func(models []ModelInfo) []ModelInfo { return models }
	enrichModels = func(_ context.Context, models []ModelInfo) []ModelInfo { return models }

	cache := NewModelListCache(ctx, checker, 10*time.Minute)

	// First call: all providers succeed, no warnings.
	models, warnings := cache.GetModels(ctx)
	assert.Len(t, models, 1)
	assert.Empty(t, warnings)

	mu.Lock()
	assert.Equal(t, 1, callCount)
	mu.Unlock()

	// Second call WITH auth token: should still serve cached data because
	// there are no warnings — the cache is healthy.
	authCtx := ctxWithAuth(ctx)
	models, warnings = cache.GetModels(authCtx)
	assert.Len(t, models, 1, "should still serve cached data")
	assert.Empty(t, warnings)

	mu.Lock()
	assert.Equal(t, 1, callCount, "provider should NOT be called again when cache is healthy")
	mu.Unlock()
}

func TestCacheTTLFromEnv(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	t.Run("returns default when env var is not set", func(t *testing.T) {
		t.Setenv("MODEL_CACHE_TTL", "")
		assert.Equal(t, DefaultCacheTTL, CacheTTLFromEnv(logger))
	})

	t.Run("parses valid duration", func(t *testing.T) {
		t.Setenv("MODEL_CACHE_TTL", "30s")
		assert.Equal(t, 30*time.Second, CacheTTLFromEnv(logger))
	})

	t.Run("parses minutes", func(t *testing.T) {
		t.Setenv("MODEL_CACHE_TTL", "5m")
		assert.Equal(t, 5*time.Minute, CacheTTLFromEnv(logger))
	})

	t.Run("returns default for invalid value", func(t *testing.T) {
		t.Setenv("MODEL_CACHE_TTL", "not-a-duration")
		assert.Equal(t, DefaultCacheTTL, CacheTTLFromEnv(logger))
	})

	t.Run("returns default for zero", func(t *testing.T) {
		t.Setenv("MODEL_CACHE_TTL", "0s")
		assert.Equal(t, DefaultCacheTTL, CacheTTLFromEnv(logger))
	})

	t.Run("returns default for negative", func(t *testing.T) {
		t.Setenv("MODEL_CACHE_TTL", "-5m")
		assert.Equal(t, DefaultCacheTTL, CacheTTLFromEnv(logger))
	})
}

func TestResolveModelIDFromMetadata(t *testing.T) {
	metadata := map[string]ModelMetadata{
		"gpt-4o":   {ModelID: "gpt-4o-id"},
		"claude-3": {ModelID: "claude-3-id"},
	}

	t.Run("found", func(t *testing.T) {
		id, err := resolveModelIDFromMetadata("gpt-4o", metadata)
		require.NoError(t, err)
		assert.Equal(t, "gpt-4o-id", id)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := resolveModelIDFromMetadata("unknown-model", metadata)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "modelID not found in metadata")
	})

	t.Run("nil metadata", func(t *testing.T) {
		_, err := resolveModelIDFromMetadata("gpt-4o", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "modelID not found in metadata")
	})
}

// newAllProvidersChecker creates a MockContextChecker that enables all three
// providers (Bedrock, Vertex, Azure).
func newAllProvidersChecker(t *testing.T) *MockContextChecker {
	t.Helper()

	logger := zaptest.NewLogger(t)
	checker := new(MockContextChecker)
	checker.On("IsUseGenAiInfraModels", mock.Anything).Return(true)
	checker.On("IsUseGCPVertex", mock.Anything).Return(true)
	checker.On("IsUseAzureGenAIURL", mock.Anything).Return(true)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Bedrock").Return(true)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Vertex").Return(true)
	checker.On("IsLLMProviderConfigured", mock.Anything, "Azure").Return(true)
	checker.On("LoggerFromContext", mock.Anything).Return(logger)
	checker.On("ContextWithGinContext", mock.Anything, mock.Anything).
		Return(context.Background())
	checker.On("AzureGenAIURL", mock.Anything).Return("https://mock-azure.example.com")

	return checker
}
