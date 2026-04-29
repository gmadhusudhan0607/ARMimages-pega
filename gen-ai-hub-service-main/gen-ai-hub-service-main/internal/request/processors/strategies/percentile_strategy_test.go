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

func TestPercentileTokenStrategy_ShouldAdjust(t *testing.T) {
	tests := []struct {
		name            string
		originalTokens  *int
		forceAdjustment bool
		expected        bool
	}{
		{
			name:            "no max_tokens in request - should adjust",
			originalTokens:  nil,
			forceAdjustment: false,
			expected:        true,
		},
		{
			name:            "max_tokens exists, force=false - should not adjust",
			originalTokens:  intPtr(500),
			forceAdjustment: false,
			expected:        false,
		},
		{
			name:            "max_tokens exists, force=true - should adjust",
			originalTokens:  intPtr(500),
			forceAdjustment: true,
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(10)
			strategy := NewPercentileTokenStrategy(percentileCache, 95, 1000, config.OutputTokensStrategyP95)

			result := strategy.ShouldAdjust(tt.originalTokens, tt.forceAdjustment)
			if result != tt.expected {
				t.Errorf("ShouldAdjust() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPercentileTokenStrategy_CalculateAdjustedValue(t *testing.T) {
	tests := []struct {
		name           string
		configValue    int
		modelMaximum   *float64
		expectedResult int
	}{
		{
			name:           "basic config value",
			configValue:    1000,
			modelMaximum:   nil,
			expectedResult: 1000,
		},
		{
			name:           "config value with model maximum - under limit",
			configValue:    1000,
			modelMaximum:   floatPtr(2000.0),
			expectedResult: 1000,
		},
		{
			name:           "config value with model maximum - over limit",
			configValue:    3000,
			modelMaximum:   floatPtr(2000.0),
			expectedResult: 2000,
		},
		{
			name:           "config value with fractional model maximum",
			configValue:    1500,
			modelMaximum:   floatPtr(1200.7),
			expectedResult: 1200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(10)
			strategy := NewPercentileTokenStrategy(percentileCache, 95, tt.configValue, config.OutputTokensStrategyP95)

			result := strategy.CalculateAdjustedValue(nil, tt.modelMaximum, tt.configValue)
			if *result != tt.expectedResult {
				t.Errorf("CalculateAdjustedValue() = %v, want %v", *result, tt.expectedResult)
			}
		})
	}
}

func TestPercentileTokenStrategy_CalculateAdjustedValueWithCache(t *testing.T) {
	tests := []struct {
		name           string
		percentile     int
		configValue    int
		samples        []int
		modelMaximum   *float64
		expectedResult int
	}{
		{
			name:           "P95 - no cache samples, use config value",
			percentile:     95,
			configValue:    1000,
			samples:        []int{},
			modelMaximum:   nil,
			expectedResult: 1000,
		},
		{
			name:           "P95 - single sample",
			percentile:     95,
			configValue:    1000,
			samples:        []int{1200},
			modelMaximum:   nil,
			expectedResult: 1200,
		},
		{
			name:           "P95 - multiple samples",
			percentile:     95,
			configValue:    1000,
			samples:        []int{800, 900, 1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700},
			modelMaximum:   nil,
			expectedResult: 1700, // P95 of 10 samples: ceil(95/100 * 10) = ceil(9.5) = 10, index 9, value 1700
		},
		{
			name:           "P99 - multiple samples",
			percentile:     99,
			configValue:    1000,
			samples:        []int{800, 900, 1000, 1100, 1200, 1300, 1400, 1500, 1600, 1700},
			modelMaximum:   nil,
			expectedResult: 1700, // P99 of 10 samples: ceil(99/100 * 10) = ceil(9.9) = 10, index 9, value 1700
		},
		{
			name:           "P95 - percentile value lower than config, use config",
			percentile:     95,
			configValue:    1500,
			samples:        []int{800, 900, 1000},
			modelMaximum:   nil,
			expectedResult: 1500, // max(percentile=1000, config=1500) = 1500
		},
		{
			name:           "P95 - with model maximum limit",
			percentile:     95,
			configValue:    1000,
			samples:        []int{1800, 1900, 2000},
			modelMaximum:   floatPtr(1500.0),
			expectedResult: 1500, // percentile=2000 but limited by model max=1500
		},
		{
			name:           "P97 - multiple samples",
			percentile:     97,
			configValue:    1000,
			samples:        []int{1000, 1100, 1200, 1300, 1400},
			modelMaximum:   nil,
			expectedResult: 1400, // P97 of 5 samples: ceil(97/100 * 5) = ceil(4.85) = 5, index 4, value 1400
		},
		{
			name:           "P98 - multiple samples",
			percentile:     98,
			configValue:    1000,
			samples:        []int{1000, 1100, 1200, 1300, 1400},
			modelMaximum:   nil,
			expectedResult: 1400, // P98 of 5 samples: ceil(98/100 * 5) = ceil(4.9) = 5, index 4, value 1400
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(20)
			strategy := NewPercentileTokenStrategy(percentileCache, tt.percentile, tt.configValue, config.OutputTokensStrategyP95)

			// Add samples to cache
			cacheKey := cache.CacheKey{
				Infrastructure: "test-infra",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "test-version",
				IsolationID:    "test-isolation",
			}

			for _, sample := range tt.samples {
				percentileCache.AddSample(cacheKey, sample)
			}

			result := strategy.CalculateAdjustedValueWithCache(nil, tt.modelMaximum, tt.configValue, cacheKey)
			if *result != tt.expectedResult {
				t.Errorf("CalculateAdjustedValueWithCache() = %v, want %v", *result, tt.expectedResult)
			}
		})
	}
}

func TestPercentileTokenStrategy_UpdateCache(t *testing.T) {
	tests := []struct {
		name        string
		usedTokens  int
		configValue int
		expected    int // expected value stored in cache
	}{
		{
			name:        "used tokens higher than config",
			usedTokens:  1200,
			configValue: 1000,
			expected:    1200,
		},
		{
			name:        "used tokens lower than config",
			usedTokens:  800,
			configValue: 1000,
			expected:    1000,
		},
		{
			name:        "used tokens equal to config",
			usedTokens:  1000,
			configValue: 1000,
			expected:    1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(10)
			strategy := NewPercentileTokenStrategy(percentileCache, 95, tt.configValue, config.OutputTokensStrategyP95)

			cacheKey := cache.CacheKey{
				Infrastructure: "test-infra",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "test-version",
				IsolationID:    "test-isolation",
			}

			strategy.UpdateCache(cacheKey, tt.usedTokens, tt.configValue)

			// Verify the value was stored correctly
			percentile, exists := percentileCache.GetPercentile(cacheKey, 95)
			if !exists {
				t.Errorf("Expected cache to contain sample after UpdateCache")
			}
			if percentile != tt.expected {
				t.Errorf("UpdateCache() stored %v, want %v", percentile, tt.expected)
			}
		})
	}
}

func TestPercentileTokenStrategy_GetStrategyName(t *testing.T) {
	tests := []struct {
		name     string
		strategy config.OutputTokensStrategy
	}{
		{
			name:     "P95 strategy",
			strategy: config.OutputTokensStrategyP95,
		},
		{
			name:     "P96 strategy",
			strategy: config.OutputTokensStrategyP96,
		},
		{
			name:     "P97 strategy",
			strategy: config.OutputTokensStrategyP97,
		},
		{
			name:     "P98 strategy",
			strategy: config.OutputTokensStrategyP98,
		},
		{
			name:     "P99 strategy",
			strategy: config.OutputTokensStrategyP99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(10)
			strategy := NewPercentileTokenStrategy(percentileCache, 95, 1000, tt.strategy)

			result := strategy.GetStrategyName()
			if result != tt.strategy {
				t.Errorf("GetStrategyName() = %v, want %v", result, tt.strategy)
			}
		})
	}
}

func TestPercentileTokenStrategy_GetPercentile(t *testing.T) {
	tests := []struct {
		name       string
		percentile int
	}{
		{name: "P95", percentile: 95},
		{name: "P96", percentile: 96},
		{name: "P97", percentile: 97},
		{name: "P98", percentile: 98},
		{name: "P99", percentile: 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(10)
			strategy := NewPercentileTokenStrategy(percentileCache, tt.percentile, 1000, config.OutputTokensStrategyP95)

			result := strategy.GetPercentile()
			if result != tt.percentile {
				t.Errorf("GetPercentile() = %v, want %v", result, tt.percentile)
			}
		})
	}
}

func TestPercentileTokenStrategy_GetCache(t *testing.T) {
	percentileCache := cache.NewPercentileTokenCache(10)
	strategy := NewPercentileTokenStrategy(percentileCache, 95, 1000, config.OutputTokensStrategyP95)

	result := strategy.GetCache()
	if result != percentileCache {
		t.Errorf("GetCache() returned different cache instance")
	}
}

func TestPercentileTokenStrategy_EmptyCacheScenarios(t *testing.T) {
	tests := []struct {
		name           string
		percentile     int
		strategyConfig int
		configValue    int
		modelMaximum   *float64
		expectedResult int
		description    string
	}{
		{
			name:           "empty cache - use valid configValue",
			percentile:     95,
			strategyConfig: 1500,
			configValue:    1000,
			modelMaximum:   nil,
			expectedResult: 1000,
			description:    "When cache is empty and configValue > 0, should use configValue",
		},
		{
			name:           "empty cache - configValue zero, use strategy default",
			percentile:     95,
			strategyConfig: 1500,
			configValue:    0,
			modelMaximum:   nil,
			expectedResult: 1500,
			description:    "When cache is empty and configValue <= 0, should use strategy's configValue",
		},
		{
			name:           "empty cache - configValue negative, use strategy default",
			percentile:     99,
			strategyConfig: 2000,
			configValue:    -1,
			modelMaximum:   nil,
			expectedResult: 2000,
			description:    "When cache is empty and configValue < 0, should use strategy's configValue",
		},
		{
			name:           "empty cache - apply model maximum limit",
			percentile:     95,
			strategyConfig: 1500,
			configValue:    2000,
			modelMaximum:   floatPtr(1800.0),
			expectedResult: 1800,
			description:    "When cache is empty, should apply model maximum limits",
		},
		{
			name:           "empty cache - model maximum with fractional value",
			percentile:     97,
			strategyConfig: 1500,
			configValue:    2000,
			modelMaximum:   floatPtr(1750.7),
			expectedResult: 1750,
			description:    "When cache is empty, should floor fractional model maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(10)
			strategy := NewPercentileTokenStrategy(percentileCache, tt.percentile, tt.strategyConfig, config.OutputTokensStrategyP95)

			cacheKey := cache.CacheKey{
				Infrastructure: "test-infra",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "test-version",
				IsolationID:    "test-isolation-empty",
			}

			// Ensure cache is empty for this key
			_, exists := percentileCache.GetPercentile(cacheKey, tt.percentile)
			if exists {
				t.Errorf("Expected cache to be empty for key, but found existing data")
			}

			result := strategy.CalculateAdjustedValueWithCache(nil, tt.modelMaximum, tt.configValue, cacheKey)
			if *result != tt.expectedResult {
				t.Errorf("%s: CalculateAdjustedValueWithCache() = %v, want %v", tt.description, *result, tt.expectedResult)
			}
		})
	}
}

func TestPercentileTokenStrategy_CacheKeyIsolation(t *testing.T) {
	tests := []struct {
		name        string
		key1        cache.CacheKey
		key2        cache.CacheKey
		description string
	}{
		{
			name: "different isolation IDs",
			key1: cache.CacheKey{
				IsolationID:    "isolation-1",
				Infrastructure: "test-infra",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "v1",
			},
			key2: cache.CacheKey{
				IsolationID:    "isolation-2",
				Infrastructure: "test-infra",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "v1",
			},
			description: "Different isolation IDs should maintain separate caches",
		},
		{
			name: "different model names",
			key1: cache.CacheKey{
				IsolationID:    "test-isolation",
				Infrastructure: "test-infra",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "gpt-3.5-turbo",
				ModelVersion:   "v1",
			},
			key2: cache.CacheKey{
				IsolationID:    "test-isolation",
				Infrastructure: "test-infra",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "gpt-4",
				ModelVersion:   "v1",
			},
			description: "Different model names should maintain separate caches",
		},
		{
			name: "different providers",
			key1: cache.CacheKey{
				IsolationID:    "test-isolation",
				Infrastructure: "test-infra",
				Provider:       "openai",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "v1",
			},
			key2: cache.CacheKey{
				IsolationID:    "test-isolation",
				Infrastructure: "test-infra",
				Provider:       "anthropic",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "v1",
			},
			description: "Different providers should maintain separate caches",
		},
		{
			name: "different infrastructures",
			key1: cache.CacheKey{
				IsolationID:    "test-isolation",
				Infrastructure: "infra-1",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "v1",
			},
			key2: cache.CacheKey{
				IsolationID:    "test-isolation",
				Infrastructure: "infra-2",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "v1",
			},
			description: "Different infrastructures should maintain separate caches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(20)
			strategy := NewPercentileTokenStrategy(percentileCache, 95, 1500, config.OutputTokensStrategyP95)

			// Add samples to first key
			percentileCache.AddSample(tt.key1, 1000)
			percentileCache.AddSample(tt.key1, 2000)

			// Verify first key has samples
			samples1 := percentileCache.GetSamples(tt.key1)
			if len(samples1) != 2 {
				t.Errorf("Expected key1 to have 2 samples, got %d", len(samples1))
			}

			// Verify second key is empty (isolated)
			samples2 := percentileCache.GetSamples(tt.key2)
			if len(samples2) != 0 {
				t.Errorf("%s: Expected key2 to have 0 samples (isolated), got %d", tt.description, len(samples2))
			}

			// Test empty cache behavior for second key
			result := strategy.CalculateAdjustedValueWithCache(nil, nil, 1200, tt.key2)
			if *result != 1200 {
				t.Errorf("%s: Expected empty cache to use configValue 1200, got %d", tt.description, *result)
			}

			// Verify first key still has its percentile calculation
			result1 := strategy.CalculateAdjustedValueWithCache(nil, nil, 1200, tt.key1)
			if *result1 != 2000 { // P95 of [1000, 2000] should be 2000
				t.Errorf("%s: Expected key1 to use percentile value 2000, got %d", tt.description, *result1)
			}
		})
	}
}

func TestPercentileTokenStrategy_FallbackValues(t *testing.T) {
	tests := []struct {
		name           string
		percentile     int
		strategyConfig int
		configValue    int
		samples        []int
		expectedResult int
		description    string
	}{
		{
			name:           "percentile lower than config - use config",
			percentile:     95,
			strategyConfig: 1500,
			configValue:    2000,
			samples:        []int{800, 900, 1000}, // P95 will be 1000
			expectedResult: 2000,                  // max(1000, 2000) = 2000
			description:    "Should use max(percentile_value, configValue)",
		},
		{
			name:           "percentile higher than config - use percentile",
			percentile:     95,
			strategyConfig: 1500,
			configValue:    1000,
			samples:        []int{1800, 1900, 2000}, // P95 will be 2000
			expectedResult: 2000,                    // max(2000, 1000) = 2000
			description:    "Should use max(percentile_value, configValue)",
		},
		{
			name:           "percentile equal to config - use either",
			percentile:     95,
			strategyConfig: 1500,
			configValue:    1500,
			samples:        []int{1400, 1500, 1600}, // P95 will be 1600
			expectedResult: 1600,                    // max(1600, 1500) = 1600
			description:    "Should use max(percentile_value, configValue)",
		},
		{
			name:           "config zero with samples - use percentile",
			percentile:     99,
			strategyConfig: 1500,
			configValue:    0,
			samples:        []int{1000, 1100, 1200},
			expectedResult: 1200, // P99 of 3 samples
			description:    "With configValue=0, should use percentile when available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			percentileCache := cache.NewPercentileTokenCache(20)
			strategy := NewPercentileTokenStrategy(percentileCache, tt.percentile, tt.strategyConfig, config.OutputTokensStrategyP95)

			cacheKey := cache.CacheKey{
				Infrastructure: "test-infra",
				Provider:       "test-provider",
				Creator:        "test-creator",
				ModelName:      "test-model",
				ModelVersion:   "test-version",
				IsolationID:    "test-isolation-fallback",
			}

			// Add samples to cache
			for _, sample := range tt.samples {
				percentileCache.AddSample(cacheKey, sample)
			}

			result := strategy.CalculateAdjustedValueWithCache(nil, nil, tt.configValue, cacheKey)
			if *result != tt.expectedResult {
				t.Errorf("%s: CalculateAdjustedValueWithCache() = %v, want %v", tt.description, *result, tt.expectedResult)
			}
		})
	}
}

func TestPercentileTokenStrategy_EdgeCases(t *testing.T) {
	t.Run("percentile calculation with edge case sample counts", func(t *testing.T) {
		percentileCache := cache.NewPercentileTokenCache(20)
		strategy := NewPercentileTokenStrategy(percentileCache, 95, 1000, config.OutputTokensStrategyP95)

		cacheKey := cache.CacheKey{
			Infrastructure: "test-infra",
			Provider:       "test-provider",
			Creator:        "test-creator",
			ModelName:      "test-model",
			ModelVersion:   "test-version",
			IsolationID:    "test-isolation",
		}

		// Test with exactly 2 samples
		percentileCache.AddSample(cacheKey, 1000)
		percentileCache.AddSample(cacheKey, 2000)

		result := strategy.CalculateAdjustedValueWithCache(nil, nil, 1000, cacheKey)
		if *result != 2000 { // P95 of [1000, 2000]: ceil(95/100 * 2) = ceil(1.9) = 2, index 1, value 2000
			t.Errorf("CalculateAdjustedValueWithCache() with 2 samples = %v, want %v", *result, 2000)
		}
	})

	t.Run("percentile calculation with many samples", func(t *testing.T) {
		percentileCache := cache.NewPercentileTokenCache(100)
		strategy := NewPercentileTokenStrategy(percentileCache, 95, 1000, config.OutputTokensStrategyP95)

		cacheKey := cache.CacheKey{
			Infrastructure: "test-infra",
			Provider:       "test-provider",
			Creator:        "test-creator",
			ModelName:      "test-model",
			ModelVersion:   "test-version",
			IsolationID:    "test-isolation-many",
		}

		// Add 20 samples: 1000, 1100, 1200, ..., 2900
		for i := 0; i < 20; i++ {
			percentileCache.AddSample(cacheKey, 1000+i*100)
		}

		result := strategy.CalculateAdjustedValueWithCache(nil, nil, 1000, cacheKey)
		// P95 of 20 samples: ceil(95/100 * 20) = ceil(19.0) = 19, index 18, value 2800
		if *result != 2800 {
			t.Errorf("CalculateAdjustedValueWithCache() with 20 samples = %v, want %v", *result, 2800)
		}
	})
}
