/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

// OutputTokensStrategy represents the strategy for adjusting max tokens
type OutputTokensStrategy string

const (
	OutputTokensStrategyDisabled       OutputTokensStrategy = "DISABLED"
	OutputTokensStrategyMonitoringOnly OutputTokensStrategy = "MONITORING_ONLY"
	OutputTokensStrategyFixed          OutputTokensStrategy = "FIXED"
	OutputTokensStrategyAutoIncreasing OutputTokensStrategy = "AUTO_INCREASING"
	OutputTokensStrategyP95            OutputTokensStrategy = "P95"
	OutputTokensStrategyP96            OutputTokensStrategy = "P96"
	OutputTokensStrategyP97            OutputTokensStrategy = "P97"
	OutputTokensStrategyP98            OutputTokensStrategy = "P98"
	OutputTokensStrategyP99            OutputTokensStrategy = "P99"
)

// IsValid checks if the strategy is valid
func (s OutputTokensStrategy) IsValid() bool {
	switch s {
	case OutputTokensStrategyDisabled,
		OutputTokensStrategyMonitoringOnly,
		OutputTokensStrategyFixed,
		OutputTokensStrategyAutoIncreasing,
		OutputTokensStrategyP95,
		OutputTokensStrategyP96,
		OutputTokensStrategyP97,
		OutputTokensStrategyP98,
		OutputTokensStrategyP99:
		return true
	default:
		return false
	}
}

// DefaultCopyrightMessage is the constant message used for copyright protection
const DefaultCopyrightMessage = "If the user requests copyrighted content such as books, lyrics, recipes, news articles or other content that may violate copyrights or be considered as copyright infringement, politely refuse and explain that you cannot provide the content. Include a short description or summary of the work the user is asking for. You **must not** violate any copyrights under any circumstances."

// ConfigProvider interface for accessing configuration
type ConfigProvider interface {
	GetCopyrightProtection() bool
	GetOutputTokensBaseValue() int
	GetOutputTokensAdjustmentForced() bool
	GetOutputTokensPercentile() int
	GetOutputTokensStrategy() OutputTokensStrategy
}

// ReqProcessingConfig holds the configuration for request processing
type ReqProcessingConfig struct {
	CopyrightProtection             bool                 // REQUEST_PROCESSING_COPYRIGHT_PROTECTION (default: false)
	OutputTokensStrategy            OutputTokensStrategy // REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY (default: MONITORING_ONLY)
	OutputTokensBaseValue           int                  // REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE (default: -1)
	OutputTokensPercentile          int                  // Derived from OutputTokensStrategy (default: 0) allowed: 0, 95, 96, 97, 98, 99
	OutputTokensAdjustmentForced    bool                 // REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED (default: false)
	OutputTokensAdjustmentInStreams bool                 // REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_IN_STREAMING (default: false)
	CacheSize                       int                  // REQUEST_PROCESSING_CACHE_SIZE (default: 1000, allowed: >0)
}

var (
	globalConfig *ReqProcessingConfig
	configOnce   sync.Once
)

// GetConfig returns the global configuration instance, loading it once on the first call
func GetConfig(ctx context.Context) (*ReqProcessingConfig, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	var configErr error
	configOnce.Do(func() {
		logger.Debug("Loading request processing configuration")
		config, err := LoadConfigFromEnv(ctx)
		if err != nil {
			logger.Errorf("Failed to load configuration: %v", err)
			configErr = fmt.Errorf("failed to load configuration: %w", err)
			return
		}
		globalConfig = config
		logger.Debug("Request processing configuration loaded successfully")
	})

	if configErr != nil {
		return nil, configErr
	}

	if globalConfig == nil {
		return nil, fmt.Errorf("configuration not initialized")
	}

	return globalConfig, nil
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv(ctx context.Context) (*ReqProcessingConfig, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	logger.Debug("Loading configuration from environment variables")

	config := &ReqProcessingConfig{
		// Set defaults
		CopyrightProtection:             false,
		OutputTokensBaseValue:           -1,
		OutputTokensAdjustmentForced:    false,
		OutputTokensAdjustmentInStreams: false,
		OutputTokensPercentile:          0,
		OutputTokensStrategy:            OutputTokensStrategyMonitoringOnly,
		CacheSize:                       1000,
	}

	// Load from environment variables
	if err := loadBoolEnvVar(logger, "REQUEST_PROCESSING_COPYRIGHT_PROTECTION", &config.CopyrightProtection); err != nil {
		return nil, err
	}

	if err := loadIntEnvVar(logger, "REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE", &config.OutputTokensBaseValue); err != nil {
		return nil, err
	}

	if err := loadBoolEnvVar(logger, "REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED", &config.OutputTokensAdjustmentForced); err != nil {
		return nil, err
	}

	if err := loadBoolEnvVar(logger, "REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_IN_STREAMING", &config.OutputTokensAdjustmentInStreams); err != nil {
		return nil, err
	}

	if val := os.Getenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY"); val != "" {
		logger.Debugf("Processing REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY: %s", val)
		strategy := OutputTokensStrategy(val)
		if !strategy.IsValid() {
			return nil, fmt.Errorf("invalid REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY value: %s", val)
		}
		config.OutputTokensStrategy = strategy
	}

	// Set OutputTokensPercentile based on OutputTokensStrategy
	config.OutputTokensPercentile = getPercentileFromStrategy(config.OutputTokensStrategy)

	if err := loadIntEnvVar(logger, "REQUEST_PROCESSING_CACHE_SIZE", &config.CacheSize); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		logger.Errorf("Configuration validation failed: %v", err)
		return nil, err
	}

	logger.Debugf("Request processing configuration loaded - CopyrightProtection: %t, OutputTokensStrategy: %s, OutputTokensBaseValue: %d",
		config.CopyrightProtection, config.OutputTokensStrategy, config.OutputTokensBaseValue)

	return config, nil
}

// loadBoolEnvVar loads a boolean environment variable if it exists
func loadBoolEnvVar(logger interface{ Debugf(string, ...interface{}) }, envKey string, target *bool) error {
	if val := os.Getenv(envKey); val != "" {
		logger.Debugf("Processing %s: %s", envKey, val)
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("invalid %s value: %s", envKey, val)
		}
		*target = parsed
	}
	return nil
}

// loadIntEnvVar loads an integer environment variable if it exists
func loadIntEnvVar(logger interface{ Debugf(string, ...interface{}) }, envKey string, target *int) error {
	if val := os.Getenv(envKey); val != "" {
		logger.Debugf("Processing %s: %s", envKey, val)
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("invalid %s value: %s", envKey, val)
		}
		*target = parsed
	}
	return nil
}

// getPercentileFromStrategy returns the percentile value based on the strategy
func getPercentileFromStrategy(strategy OutputTokensStrategy) int {
	switch strategy {
	case OutputTokensStrategyP95:
		return 95
	case OutputTokensStrategyP96:
		return 96
	case OutputTokensStrategyP97:
		return 97
	case OutputTokensStrategyP98:
		return 98
	case OutputTokensStrategyP99:
		return 99
	default:
		return 0
	}
}

// GetCopyrightProtection implements ConfigProvider interface
func (c *ReqProcessingConfig) GetCopyrightProtection() bool {
	return c.CopyrightProtection
}

// GetOutputTokensBaseValue implements ConfigProvider interface
func (c *ReqProcessingConfig) GetOutputTokensBaseValue() int {
	return c.OutputTokensBaseValue
}

// GetOutputTokensAdjustmentForced GetForceMaxTokensAdjustment implements ConfigProvider interface
func (c *ReqProcessingConfig) GetOutputTokensAdjustmentForced() bool {
	return c.OutputTokensAdjustmentForced
}

// GetOutputTokensPercentile GetMaxTokensPercentile implements ConfigProvider interface
func (c *ReqProcessingConfig) GetOutputTokensPercentile() int {
	return c.OutputTokensPercentile
}

// GetOutputTokensStrategy implements ConfigProvider interface
func (c *ReqProcessingConfig) GetOutputTokensStrategy() OutputTokensStrategy {
	return c.OutputTokensStrategy
}

// GetOutputTokensAdjustmentInStreams returns whether output tokens adjustment is enabled for streams
func (c *ReqProcessingConfig) GetOutputTokensAdjustmentInStreams() bool {
	return c.OutputTokensAdjustmentInStreams
}

// IsOutputTokensStrategyDisabled checks if the output tokens strategy is DISABLED
func IsOutputTokensStrategyDisabled(ctx context.Context) bool {
	config, err := GetConfig(ctx)
	if err != nil {
		logger := cntx.LoggerFromContext(ctx).Sugar()
		logger.Errorf("Failed to get config for strategy check: %v", err)
		return false
	}
	return config.OutputTokensStrategy == OutputTokensStrategyDisabled
}

// Validate validates the configuration values
func (c *ReqProcessingConfig) Validate() error {

	// Validate CacheSize
	if c.CacheSize <= 0 {
		return fmt.Errorf("REQUEST_PROCESSING_CACHE_SIZE must be > 0, got: %d", c.CacheSize)
	}
	return nil
}

// ResetConfigSingleton resets the singleton for testing purposes
func ResetConfigSingleton() {
	globalConfig = nil
	configOnce = sync.Once{}
}
