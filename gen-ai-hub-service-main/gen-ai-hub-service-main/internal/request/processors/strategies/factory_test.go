/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/cache"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

func TestCreateTokenAdjustmentStrategy_AutoIncreasingCacheSize(t *testing.T) {
	// Reset global cache instances to ensure clean test state
	globalTokenCache = nil
	globalPercentileCache = nil
	tokenCacheOnce = sync.Once{}
	percentileCacheOnce = sync.Once{}

	// Create config with AUTO_INCREASING strategy and large cache size
	cfg := &config.ReqProcessingConfig{
		OutputTokensStrategy:  config.OutputTokensStrategyAutoIncreasing,
		OutputTokensBaseValue: 1000,
		CacheSize:             2000, // This should be ignored for AUTO_INCREASING
	}

	strategy, err := CreateTokenAdjustmentStrategy(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	autoStrategy, ok := strategy.(*AutoIncreasingStrategy)
	if !ok {
		t.Fatal("Expected AutoIncreasingStrategy")
	}

	// Verify cache size is forced to 1 for AUTO_INCREASING
	tokenCache := autoStrategy.GetCache()
	if tokenCache == nil {
		t.Fatal("Expected cache to be non-nil")
	}

	// The cache size is internal to TokenCache, but we can verify the behavior
	// by checking that only 1 entry can be stored per isolation
	testKey1 := cache.CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "provider1",
		Creator:        "creator1",
		ModelName:      "model1",
		ModelVersion:   "v1",
	}
	testKey2 := cache.CacheKey{
		IsolationID:    "iso1", // Same isolation ID to test per-isolation limit
		Infrastructure: "infra1",
		Provider:       "provider1",
		Creator:        "creator1",
		ModelName:      "model2", // Different model to create different cache key
		ModelVersion:   "v1",
	}

	// Set first entry
	tokenCache.Set(testKey1, 1000)
	if tokenCache.Size() != 1 {
		t.Errorf("Expected cache size to be 1 after first entry, got %d", tokenCache.Size())
	}

	// Set second entry in same isolation - should evict first one due to per-isolation cache size limit of 1
	tokenCache.Set(testKey2, 2000)
	if tokenCache.Size() != 1 {
		t.Errorf("Expected cache size to be 1 after second entry, got %d", tokenCache.Size())
	}

	// Verify first entry was evicted
	_, exists := tokenCache.Get(testKey1)
	if exists {
		t.Error("Expected first entry to be evicted when cache size is 1 per isolation")
	}

	// Verify second entry exists
	value, exists := tokenCache.Get(testKey2)
	if !exists {
		t.Error("Expected second entry to exist")
	}
	if value != 2000 {
		t.Errorf("Expected second entry value to be 2000, got %d", value)
	}

	// Test that different isolations can have their own entries
	testKey3 := cache.CacheKey{
		IsolationID:    "iso2", // Different isolation
		Infrastructure: "infra2",
		Provider:       "provider2",
		Creator:        "creator2",
		ModelName:      "model3",
		ModelVersion:   "v1",
	}

	// Set entry in different isolation - should coexist with iso1 entry
	tokenCache.Set(testKey3, 3000)
	if tokenCache.Size() != 2 {
		t.Errorf("Expected cache size to be 2 with entries from different isolations, got %d", tokenCache.Size())
	}

	// Verify both entries exist (from different isolations)
	value2, exists2 := tokenCache.Get(testKey2)
	if !exists2 {
		t.Error("Expected iso1 entry to still exist")
	}
	if value2 != 2000 {
		t.Errorf("Expected iso1 entry value to be 2000, got %d", value2)
	}

	value3, exists3 := tokenCache.Get(testKey3)
	if !exists3 {
		t.Error("Expected iso2 entry to exist")
	}
	if value3 != 3000 {
		t.Errorf("Expected iso2 entry value to be 3000, got %d", value3)
	}
}

func TestCreateTokenAdjustmentStrategy_P95CacheSize(t *testing.T) {
	// Reset global cache instances to ensure clean test state
	globalTokenCache = nil
	globalPercentileCache = nil
	tokenCacheOnce = sync.Once{}
	percentileCacheOnce = sync.Once{}

	// Create config with P95 strategy and specific cache size
	cfg := &config.ReqProcessingConfig{
		OutputTokensStrategy:  config.OutputTokensStrategyP95,
		OutputTokensBaseValue: 1000,
		CacheSize:             5, // P95 should use this value for samples per key
	}

	strategy, err := CreateTokenAdjustmentStrategy(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	p95Strategy, ok := strategy.(*PercentileTokenStrategy)
	if !ok {
		t.Fatal("Expected PercentileTokenStrategy")
	}

	// For P95 strategy, verify that the cache respects the sample limit per key
	percentileCache := p95Strategy.cache

	testKey := cache.CacheKey{
		IsolationID:    "test-iso",
		Infrastructure: "test-infra",
		Provider:       "test-provider",
		Creator:        "test-creator",
		ModelName:      "test-model",
		ModelVersion:   "v1",
	}

	// Add more samples than the cache size limit (5)
	for i := 0; i < 7; i++ {
		percentileCache.AddSample(testKey, 1000+i*100)
	}

	// Verify that only the configured number of samples (5) are kept
	sampleCount := percentileCache.GetSampleCount(testKey)
	if sampleCount != 5 {
		t.Errorf("Expected P95 strategy cache to limit samples to 5 per key, got %d", sampleCount)
	}

	// Verify that we can store multiple different keys (no limit on unique keys)
	for i := 0; i < 3; i++ {
		key := cache.CacheKey{
			IsolationID:    fmt.Sprintf("iso%d", i),
			Infrastructure: fmt.Sprintf("infra%d", i),
			Provider:       fmt.Sprintf("provider%d", i),
			Creator:        fmt.Sprintf("creator%d", i),
			ModelName:      fmt.Sprintf("model%d", i),
			ModelVersion:   fmt.Sprintf("v%d", i),
		}
		percentileCache.AddSample(key, 1000+i*100)
	}

	// Verify all keys exist (no limit on unique keys for percentile cache)
	totalUniqueKeys := percentileCache.Size()
	expectedKeys := 4 // 1 testKey + 3 additional keys
	if totalUniqueKeys != expectedKeys {
		t.Errorf("Expected P95 strategy to store %d unique keys, got %d", expectedKeys, totalUniqueKeys)
	}
}

func TestCreateTokenAdjustmentStrategy_P99CacheSize(t *testing.T) {
	// Reset global cache instances to ensure clean test state
	globalTokenCache = nil
	globalPercentileCache = nil
	tokenCacheOnce = sync.Once{}
	percentileCacheOnce = sync.Once{}

	// Create config with P99 strategy and specific cache size
	cfg := &config.ReqProcessingConfig{
		OutputTokensStrategy:  config.OutputTokensStrategyP99,
		OutputTokensBaseValue: 1000,
		CacheSize:             3, // P99 should use this value
	}

	strategy, err := CreateTokenAdjustmentStrategy(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	p99Strategy, ok := strategy.(*PercentileTokenStrategy)
	if !ok {
		t.Fatal("Expected PercentileTokenStrategy")
	}

	// Verify strategy was created correctly
	if p99Strategy.percentile != 99 {
		t.Errorf("Expected P99 strategy to have percentile 99, got %d", p99Strategy.percentile)
	}

	if p99Strategy.GetStrategyName() != config.OutputTokensStrategyP99 {
		t.Errorf("Expected strategy name to be %s, got %s", config.OutputTokensStrategyP99, p99Strategy.GetStrategyName())
	}
}

func TestCreateTokenAdjustmentStrategy_FixedStrategyNoCache(t *testing.T) {
	// Reset global cache instances to ensure clean test state
	globalTokenCache = nil
	globalPercentileCache = nil
	tokenCacheOnce = sync.Once{}
	percentileCacheOnce = sync.Once{}

	// Create config with FIXED strategy
	cfg := &config.ReqProcessingConfig{
		OutputTokensStrategy:  config.OutputTokensStrategyFixed,
		OutputTokensBaseValue: 1000,
		CacheSize:             2000, // Should be irrelevant for FIXED strategy
	}

	strategy, err := CreateTokenAdjustmentStrategy(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	fixedStrategy, ok := strategy.(*FixedTokenStrategy)
	if !ok {
		t.Fatal("Expected FixedTokenStrategy")
	}

	if fixedStrategy.GetStrategyName() != config.OutputTokensStrategyFixed {
		t.Errorf("Expected strategy name to be %s, got %s", config.OutputTokensStrategyFixed, fixedStrategy.GetStrategyName())
	}
}

func TestCreateTokenAdjustmentStrategy_MonitoringOnlyStrategyNoCache(t *testing.T) {
	// Reset global cache instances to ensure clean test state
	globalTokenCache = nil
	globalPercentileCache = nil
	tokenCacheOnce = sync.Once{}
	percentileCacheOnce = sync.Once{}

	// Create config with MONITORING_ONLY strategy
	cfg := &config.ReqProcessingConfig{
		OutputTokensStrategy:  config.OutputTokensStrategyMonitoringOnly,
		OutputTokensBaseValue: 0,    // Not required for MONITORING_ONLY
		CacheSize:             2000, // Should be irrelevant for MONITORING_ONLY strategy
	}

	strategy, err := CreateTokenAdjustmentStrategy(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	monitoringOnlyStrategy, ok := strategy.(*MonitoringOnlyTokenStrategy)
	if !ok {
		t.Fatal("Expected MonitoringOnlyTokenStrategy")
	}

	if monitoringOnlyStrategy.GetStrategyName() != config.OutputTokensStrategyMonitoringOnly {
		t.Errorf("Expected strategy name to be %s, got %s", config.OutputTokensStrategyMonitoringOnly, monitoringOnlyStrategy.GetStrategyName())
	}
}
