/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import (
	"fmt"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/cache"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"sync"
)

// Global cache instances
var (
	globalTokenCache      *cache.TokenCache
	globalPercentileCache *cache.PercentileTokenCache
	tokenCacheOnce        sync.Once
	percentileCacheOnce   sync.Once
)

// getGlobalTokenCache returns the singleton token cache instance
func getGlobalTokenCache(maxSamples int) *cache.TokenCache {
	tokenCacheOnce.Do(func() {
		globalTokenCache = cache.NewTokenCache(maxSamples)
	})
	return globalTokenCache
}

// getGlobalPercentileCache returns the singleton percentile cache instance
func getGlobalPercentileCache(maxSamples int) *cache.PercentileTokenCache {
	percentileCacheOnce.Do(func() {
		globalPercentileCache = cache.NewPercentileTokenCache(maxSamples)
	})
	return globalPercentileCache
}

// CreateTokenAdjustmentStrategy creates the appropriate strategy based on configuration
func CreateTokenAdjustmentStrategy(cfg *config.ReqProcessingConfig) (TokenAdjustmentStrategy, error) {
	switch cfg.GetOutputTokensStrategy() {
	case config.OutputTokensStrategyFixed:
		if cfg.GetOutputTokensBaseValue() <= 0 {
			return nil, fmt.Errorf("FIXED strategy requires positive OutputTokensBaseValue, got: %d", cfg.GetOutputTokensBaseValue())
		}
		return NewFixedTokenStrategy(cfg.GetOutputTokensBaseValue()), nil

	case config.OutputTokensStrategyMonitoringOnly:
		return NewMonitoringOnlyTokenStrategy(), nil

	case config.OutputTokensStrategyAutoIncreasing:
		if cfg.GetOutputTokensBaseValue() <= 0 {
			return nil, fmt.Errorf("AUTO_INCREASING strategy requires positive OutputTokensBaseValue, got: %d", cfg.GetOutputTokensBaseValue())
		}
		// Always use cache size of 1 for AUTO_INCREASING (ignore cfg.CacheSize)
		tokenCache := getGlobalTokenCache(1)
		return NewAutoIncreasingStrategy(tokenCache, cfg.GetOutputTokensBaseValue()), nil

	case config.OutputTokensStrategyP95:
		if cfg.GetOutputTokensBaseValue() <= 0 {
			return nil, fmt.Errorf("P95 strategy requires positive OutputTokensBaseValue, got: %d", cfg.GetOutputTokensBaseValue())
		}
		percentileCache := getGlobalPercentileCache(cfg.CacheSize)
		return NewPercentileTokenStrategy(percentileCache, 95, cfg.GetOutputTokensBaseValue(), config.OutputTokensStrategyP95), nil

	case config.OutputTokensStrategyP96:
		if cfg.GetOutputTokensBaseValue() <= 0 {
			return nil, fmt.Errorf("P96 strategy requires positive OutputTokensBaseValue, got: %d", cfg.GetOutputTokensBaseValue())
		}
		percentileCache := getGlobalPercentileCache(cfg.CacheSize)
		return NewPercentileTokenStrategy(percentileCache, 96, cfg.GetOutputTokensBaseValue(), config.OutputTokensStrategyP96), nil

	case config.OutputTokensStrategyP97:
		if cfg.GetOutputTokensBaseValue() <= 0 {
			return nil, fmt.Errorf("P97 strategy requires positive OutputTokensBaseValue, got: %d", cfg.GetOutputTokensBaseValue())
		}
		percentileCache := getGlobalPercentileCache(cfg.CacheSize)
		return NewPercentileTokenStrategy(percentileCache, 97, cfg.GetOutputTokensBaseValue(), config.OutputTokensStrategyP97), nil

	case config.OutputTokensStrategyP98:
		if cfg.GetOutputTokensBaseValue() <= 0 {
			return nil, fmt.Errorf("P98 strategy requires positive OutputTokensBaseValue, got: %d", cfg.GetOutputTokensBaseValue())
		}
		percentileCache := getGlobalPercentileCache(cfg.CacheSize)
		return NewPercentileTokenStrategy(percentileCache, 98, cfg.GetOutputTokensBaseValue(), config.OutputTokensStrategyP98), nil

	case config.OutputTokensStrategyP99:
		if cfg.GetOutputTokensBaseValue() <= 0 {
			return nil, fmt.Errorf("P99 strategy requires positive OutputTokensBaseValue, got: %d", cfg.GetOutputTokensBaseValue())
		}
		percentileCache := getGlobalPercentileCache(cfg.CacheSize)
		return NewPercentileTokenStrategy(percentileCache, 99, cfg.GetOutputTokensBaseValue(), config.OutputTokensStrategyP99), nil

	default:
		return nil, fmt.Errorf("unsupported token adjustment strategy: %s", cfg.GetOutputTokensStrategy())
	}
}
