/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import (
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/cache"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

func TestNewAutoIncreasingStrategy(t *testing.T) {
	tokenCache := cache.NewTokenCache(100)
	configValue := 4000

	strategy := NewAutoIncreasingStrategy(tokenCache, configValue)

	if strategy == nil {
		t.Fatal("Expected strategy to be created, got nil")
	} else {
		if strategy.cache != tokenCache {
			t.Error("Expected cache to be set correctly")
		}
		if strategy.configValue != configValue {
			t.Errorf("Expected configValue to be %d, got %d", configValue, strategy.configValue)
		}
		if strategy.GetStrategyName() != config.OutputTokensStrategyAutoIncreasing {
			t.Errorf("Expected strategy name to be %s, got %s", config.OutputTokensStrategyAutoIncreasing, strategy.GetStrategyName())
		}
	}
}

func TestAutoIncreasingStrategy_ShouldAdjust(t *testing.T) {
	tokenCache := cache.NewTokenCache(100)
	strategy := NewAutoIncreasingStrategy(tokenCache, 4000)

	tests := []struct {
		name            string
		originalTokens  *int
		forceAdjustment bool
		expected        bool
	}{
		{
			name:            "No original tokens - should adjust",
			originalTokens:  nil,
			forceAdjustment: false,
			expected:        true,
		},
		{
			name:            "Original tokens present, no force - should not adjust",
			originalTokens:  intPtr(2000),
			forceAdjustment: false,
			expected:        false,
		},
		{
			name:            "Original tokens present, force enabled - should adjust",
			originalTokens:  intPtr(2000),
			forceAdjustment: true,
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.ShouldAdjust(tt.originalTokens, tt.forceAdjustment)
			if result != tt.expected {
				t.Errorf("Expected %t, got %t", tt.expected, result)
			}
		})
	}
}

func TestAutoIncreasingStrategy_CalculateAdjustedValue(t *testing.T) {
	tokenCache := cache.NewTokenCache(100)
	strategy := NewAutoIncreasingStrategy(tokenCache, 4000)

	tests := []struct {
		name           string
		originalTokens *int
		modelMaximum   *float64
		configValue    int
		expected       int
	}{
		{
			name:           "Use config value when no model maximum",
			originalTokens: nil,
			modelMaximum:   nil,
			configValue:    3000,
			expected:       3000,
		},
		{
			name:           "Apply model maximum limit",
			originalTokens: nil,
			modelMaximum:   floatPtr(2500.0),
			configValue:    3000,
			expected:       2500,
		},
		{
			name:           "Use strategy config value when configValue is 0",
			originalTokens: nil,
			modelMaximum:   nil,
			configValue:    0,
			expected:       4000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CalculateAdjustedValue(tt.originalTokens, tt.modelMaximum, tt.configValue)
			if result == nil {
				t.Fatal("Expected result to be non-nil")
			} else if *result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, *result)
			}
		})
	}
}

func TestAutoIncreasingStrategy_CalculateAdjustedValueWithCache(t *testing.T) {
	tokenCache := cache.NewTokenCache(100)
	strategy := NewAutoIncreasingStrategy(tokenCache, 4000)

	cacheKey := cache.CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	tests := []struct {
		name           string
		setupCache     func()
		originalTokens *int
		modelMaximum   *float64
		configValue    int
		expected       int
	}{
		{
			name:           "No cached value - use config value",
			setupCache:     func() {},
			originalTokens: nil,
			modelMaximum:   nil,
			configValue:    3000,
			expected:       3000,
		},
		{
			name: "Use cached value when higher than config",
			setupCache: func() {
				tokenCache.Set(cacheKey, 5000)
			},
			originalTokens: nil,
			modelMaximum:   nil,
			configValue:    3000,
			expected:       5000,
		},
		{
			name: "Use config value when higher than cached",
			setupCache: func() {
				tokenCache.Set(cacheKey, 2000)
			},
			originalTokens: nil,
			modelMaximum:   nil,
			configValue:    3000,
			expected:       3000,
		},
		{
			name: "Apply model maximum limit with cached value",
			setupCache: func() {
				tokenCache.Set(cacheKey, 5000)
			},
			originalTokens: nil,
			modelMaximum:   floatPtr(4000.0),
			configValue:    3000,
			expected:       4000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenCache.Clear()
			tt.setupCache()

			result := strategy.CalculateAdjustedValueWithCache(tt.originalTokens, tt.modelMaximum, tt.configValue, cacheKey)
			if result == nil {
				t.Fatal("Expected result to be non-nil")
			} else if *result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, *result)
			}
		})
	}
}

func TestAutoIncreasingStrategy_UpdateCache(t *testing.T) {
	tokenCache := cache.NewTokenCache(100)
	strategy := NewAutoIncreasingStrategy(tokenCache, 4000)

	cacheKey := cache.CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	tests := []struct {
		name        string
		setupCache  func()
		usedTokens  int
		configValue int
		expected    int
	}{
		{
			name:        "Store used tokens when higher than config",
			setupCache:  func() {},
			usedTokens:  5000,
			configValue: 3000,
			expected:    5000,
		},
		{
			name:        "Store config value when higher than used tokens",
			setupCache:  func() {},
			usedTokens:  2000,
			configValue: 3000,
			expected:    3000,
		},
		{
			name: "Update only if new value is higher than cached",
			setupCache: func() {
				tokenCache.Set(cacheKey, 4000)
			},
			usedTokens:  3500,
			configValue: 3000,
			expected:    4000, // Should remain unchanged
		},
		{
			name: "Update when new value is higher than cached",
			setupCache: func() {
				tokenCache.Set(cacheKey, 3000)
			},
			usedTokens:  4500,
			configValue: 3000,
			expected:    4500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenCache.Clear()
			tt.setupCache()

			result := strategy.UpdateCache(cacheKey, tt.usedTokens, tt.configValue)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}

			// Verify cache was updated correctly
			cachedValue, exists := tokenCache.Get(cacheKey)
			if !exists {
				t.Error("Expected value to exist in cache")
			}
			if cachedValue != tt.expected {
				t.Errorf("Expected cached value to be %d, got %d", tt.expected, cachedValue)
			}
		})
	}
}

func TestAutoIncreasingStrategy_GetCache(t *testing.T) {
	tokenCache := cache.NewTokenCache(100)
	strategy := NewAutoIncreasingStrategy(tokenCache, 4000)

	retrievedCache := strategy.GetCache()
	if retrievedCache != tokenCache {
		t.Error("Expected GetCache to return the same cache instance")
	}
}
