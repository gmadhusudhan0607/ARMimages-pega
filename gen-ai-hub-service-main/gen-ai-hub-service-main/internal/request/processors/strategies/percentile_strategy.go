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

// PercentileTokenStrategy implements the PXX strategy for token adjustment
type PercentileTokenStrategy struct {
	cache       *cache.PercentileTokenCache
	percentile  int
	configValue int
	strategy    config.OutputTokensStrategy
}

// NewPercentileTokenStrategy creates a new PercentileTokenStrategy
func NewPercentileTokenStrategy(percentileCache *cache.PercentileTokenCache, percentile int, configValue int, strategy config.OutputTokensStrategy) *PercentileTokenStrategy {
	return &PercentileTokenStrategy{
		cache:       percentileCache,
		percentile:  percentile,
		configValue: configValue,
		strategy:    strategy,
	}
}

// ShouldAdjust determines if tokens should be adjusted based on current state
func (s *PercentileTokenStrategy) ShouldAdjust(originalTokens *int, forceAdjustment bool) bool {
	// Case 1: max_tokens exists in request
	if originalTokens != nil {
		// Case 1.1: OutputTokensAdjustmentForced=false -> do nothing
		if !forceAdjustment {
			return false
		}
		// Case 1.2: OutputTokensAdjustmentForced=true -> update with percentile value
		return true
	}

	// Case 2: max_tokens not in request -> always insert percentile value
	return true
}

// CalculateAdjustedValue computes the new token value using percentile logic
func (s *PercentileTokenStrategy) CalculateAdjustedValue(originalTokens *int, modelMaximum *float64, configValue int) *int {
	// For compatibility with the interface, we'll use the configValue as fallback
	// The actual percentile calculation will be handled by CalculateAdjustedValueWithCache
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

// CalculateAdjustedValueWithCache computes the new token value using percentile calculation from cache
func (s *PercentileTokenStrategy) CalculateAdjustedValueWithCache(originalTokens *int, modelMaximum *float64, configValue int, cacheKey cache.CacheKey) *int {
	// Get percentile value from cache for this model configuration
	percentileValue, exists := s.cache.GetPercentile(cacheKey, s.percentile)

	var adjustedValue int
	if exists {
		// Use max(percentile_value, configValue) as the adjusted value
		adjustedValue = percentileValue
		if configValue > percentileValue {
			adjustedValue = configValue
		}
	} else {
		// No cached samples, use configValue as starting point
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

// UpdateCache adds the used tokens to the cache samples after a successful response
func (s *PercentileTokenStrategy) UpdateCache(cacheKey cache.CacheKey, usedTokens int, configValue int) {
	// Store max(used_tokens, configValue) as a sample in cache
	valueToStore := usedTokens
	if configValue > usedTokens {
		valueToStore = configValue
	}

	s.cache.AddSample(cacheKey, valueToStore)
}

// GetStrategyName returns the strategy identifier
func (s *PercentileTokenStrategy) GetStrategyName() config.OutputTokensStrategy {
	return s.strategy
}

// GetCache returns the underlying cache (for testing purposes)
func (s *PercentileTokenStrategy) GetCache() *cache.PercentileTokenCache {
	return s.cache
}

// GetPercentile returns the percentile value used by this strategy
func (s *PercentileTokenStrategy) GetPercentile() int {
	return s.percentile
}
