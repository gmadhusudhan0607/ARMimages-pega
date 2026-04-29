/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import (
	"math"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/cache"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

// AutoIncreasingStrategy implements the AUTO_INCREASING strategy for token adjustment
type AutoIncreasingStrategy struct {
	cache       *cache.TokenCache
	configValue int
}

// NewAutoIncreasingStrategy creates a new AutoIncreasingStrategy
func NewAutoIncreasingStrategy(tokenCache *cache.TokenCache, configValue int) *AutoIncreasingStrategy {
	return &AutoIncreasingStrategy{
		cache:       tokenCache,
		configValue: configValue,
	}
}

// ShouldAdjust determines if tokens should be adjusted based on the current state
func (s *AutoIncreasingStrategy) ShouldAdjust(originalTokens *int, forceAdjustment bool) bool {
	// Case 1: max_tokens exists in request
	if originalTokens != nil {
		// Case 1.1: OutputTokensAdjustmentForced=false -> do nothing
		if !forceAdjustment {
			return false
		}
		// Case 1.2: OutputTokensAdjustmentForced=true -> update with auto-adjusted value
		return true
	}

	// Case 2: max_tokens not in request -> always insert auto-adjusted value
	return true
}

// CalculateAdjustedValue computes the new token value using auto-adjustment logic
func (s *AutoIncreasingStrategy) CalculateAdjustedValue(originalTokens *int, modelMaximum *float64, configValue int) *int {
	// Use the cache to get the current auto-adjusted value for this model configuration
	// Note: The cache key will be constructed by the caller based on model metadata

	// For now, we'll use the configValue as the base value
	// The actual cache lookup will be handled by the BaseProcessor
	adjustedValue := s.configValue
	if configValue > 0 {
		adjustedValue = configValue
	}

	// Apply model maximum limit if available
	if modelMaximum != nil {
		maxInt := int(math.Floor(*modelMaximum))
		if adjustedValue > maxInt {
			adjustedValue = maxInt
		}
	}

	return &adjustedValue
}

// CalculateAdjustedValueWithCache computes the new token value using cache lookup
func (s *AutoIncreasingStrategy) CalculateAdjustedValueWithCache(originalTokens *int, modelMaximum *float64, configValue int, cacheKey cache.CacheKey) *int {
	// Get cached value for this model configuration
	cachedValue, exists := s.cache.Get(cacheKey)

	var adjustedValue int
	if exists {
		// Use max(cached_value, configValue) as the adjusted value
		adjustedValue = cachedValue
		if configValue > cachedValue {
			adjustedValue = configValue
		}
	} else {
		// No cached value, use configValue as starting point
		adjustedValue = configValue
		if adjustedValue <= 0 {
			adjustedValue = s.configValue
		}
	}

	// Apply model maximum limit if available
	if modelMaximum != nil {
		maxInt := int(math.Floor(*modelMaximum))
		if adjustedValue > maxInt {
			adjustedValue = maxInt
		}
	}

	return &adjustedValue
}

// UpdateCache updates the cache with the used tokens after a successful response
func (s *AutoIncreasingStrategy) UpdateCache(cacheKey cache.CacheKey, usedTokens int, configValue int) int {
	// Store max(used_tokens, configValue) in cache
	valueToStore := usedTokens
	if configValue > usedTokens {
		valueToStore = configValue
	}

	return s.cache.Update(cacheKey, valueToStore)
}

// GetStrategyName returns the strategy identifier
func (s *AutoIncreasingStrategy) GetStrategyName() config.OutputTokensStrategy {
	return config.OutputTokensStrategyAutoIncreasing
}

// GetCache returns the underlying cache (for testing purposes)
func (s *AutoIncreasingStrategy) GetCache() *cache.TokenCache {
	return s.cache
}
