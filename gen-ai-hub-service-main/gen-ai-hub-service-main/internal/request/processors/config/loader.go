/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
)

// ConfigLoader manages processing configurations
type ConfigLoader struct {
	defaultConfig  *extensions.ProcessingConfig
	modelOverrides map[string]*extensions.ProcessingConfig
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		modelOverrides: make(map[string]*extensions.ProcessingConfig),
	}
}

// LoadFromEnvironment loads configuration from environment variables
func LoadFromEnvironment() (*ConfigLoader, error) {
	loader := NewConfigLoader()

	// Load default configuration
	defaultConfig, err := loadDefaultConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}
	loader.defaultConfig = defaultConfig

	// Load model-specific overrides
	if err := loader.loadModelOverrides(); err != nil {
		return nil, fmt.Errorf("failed to load model overrides: %w", err)
	}

	return loader, nil
}

// GetConfig returns the configuration for a specific model
func (cl *ConfigLoader) GetConfig(model *types.Model) (*extensions.ProcessingConfig, error) {
	if model == nil {
		if cl.defaultConfig == nil {
			return nil, fmt.Errorf("no default configuration available")
		}
		return cl.defaultConfig, nil
	}

	// Try model-specific override first
	modelKey := fmt.Sprintf("%s_%s_%s", model.Provider, model.Creator, model.Name)
	if override, exists := cl.modelOverrides[modelKey]; exists {
		return override, nil
	}

	// Try provider-creator override
	providerKey := fmt.Sprintf("%s_%s", model.Provider, model.Creator)
	if override, exists := cl.modelOverrides[providerKey]; exists {
		return override, nil
	}

	// Try provider override
	if override, exists := cl.modelOverrides[string(model.Provider)]; exists {
		return override, nil
	}

	// Return default configuration
	if cl.defaultConfig == nil {
		return nil, fmt.Errorf("no default configuration available")
	}
	return cl.defaultConfig, nil
}

// loadDefaultConfig loads the default processing configuration from environment
func loadDefaultConfig() (*extensions.ProcessingConfig, error) {
	processingConfig := &extensions.ProcessingConfig{
		OutputTokensStrategy:          config.OutputTokensStrategyMonitoringOnly,
		CopyrightProtectionEnabled:    false,
		OutputTokensAdjustmentForced:  false,
		OutputTokensAdjustmentStreams: false,
	}

	// COPYRIGHT PROTECTION
	val, set, err := parseBoolEnv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION")
	if err != nil {
		return nil, err
	}
	if set {
		processingConfig.CopyrightProtectionEnabled = val
	}

	// STRATEGY
	strat, set, err := parseStrategyEnv("GENAI_PROCESSING_MAX_TOKENS_STRATEGY")
	if err != nil {
		return nil, err
	}
	if set {
		processingConfig.OutputTokensStrategy = strat
	}

	// BASE VALUE (positive int)
	v, set, err := parsePositiveIntEnv("GENAI_PROCESSING_MAX_TOKENS_VALUE")
	if err != nil {
		return nil, err
	}
	if set {
		processingConfig.OutputTokensBaseValue = &v
	}

	// FORCE ADJUSTMENT
	val, set, err = parseBoolEnv("GENAI_PROCESSING_FORCE_ADJUSTMENT")
	if err != nil {
		return nil, err
	}
	if set {
		processingConfig.OutputTokensAdjustmentForced = val
	}

	// STREAMS ADJUSTMENT
	val, set, err = parseBoolEnv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_IN_STREAMING")
	if err != nil {
		return nil, err
	}
	if set {
		processingConfig.OutputTokensAdjustmentStreams = val
	}

	return processingConfig, nil
}

// parseBoolEnv parses an environment variable as bool, returning (value, wasSet, error)
func parseBoolEnv(key string) (bool, bool, error) {
	if s := os.Getenv(key); s != "" {
		b, err := strconv.ParseBool(s)
		if err != nil {
			return false, false, fmt.Errorf("invalid %s: %s", key, s)
		}
		return b, true, nil
	}
	return false, false, nil
}

// parsePositiveIntEnv parses an environment variable as a positive int (must be >0)
// returns (value, wasSet, error)
func parsePositiveIntEnv(key string) (int, bool, error) {
	if s := os.Getenv(key); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, false, fmt.Errorf("invalid %s: %s", key, s)
		}
		if v <= 0 {
			return 0, false, fmt.Errorf("%s must be positive: %d", key, v)
		}
		return v, true, nil
	}
	return 0, false, nil
}

// parseStrategyEnv parses the strategy env var and maps it to the enum
// returns (strategy, wasSet, error)
func parseStrategyEnv(key string) (config.OutputTokensStrategy, bool, error) {
	if s := os.Getenv(key); s != "" {
		switch strings.ToUpper(s) {
		case "MONITORING_ONLY":
			return config.OutputTokensStrategyMonitoringOnly, true, nil
		case "FIXED":
			return config.OutputTokensStrategyFixed, true, nil
		default:
			return "", false, fmt.Errorf("invalid %s: %s", key, s)
		}
	}
	return config.OutputTokensStrategyMonitoringOnly, false, nil
}

// loadModelOverrides loads model-specific configuration overrides
func (cl *ConfigLoader) loadModelOverrides() error {
	// For now, simplified approach - no model-specific overrides
	// This can be extended later if needed
	return nil
}

// CreateProcessingConfigFromReqConfig creates a ProcessingConfig from ReqProcessingConfig
func CreateProcessingConfigFromReqConfig(reqConfig *config.ReqProcessingConfig) *extensions.ProcessingConfig {
	if reqConfig == nil {
		return &extensions.ProcessingConfig{
			OutputTokensStrategy:          config.OutputTokensStrategyMonitoringOnly,
			CopyrightProtectionEnabled:    false,
			OutputTokensAdjustmentForced:  false,
			OutputTokensAdjustmentStreams: false,
		}
	}

	processingConfig := &extensions.ProcessingConfig{
		OutputTokensStrategy:          reqConfig.GetOutputTokensStrategy(),
		CopyrightProtectionEnabled:    reqConfig.GetCopyrightProtection(),
		OutputTokensAdjustmentForced:  reqConfig.GetOutputTokensAdjustmentForced(),
		OutputTokensAdjustmentStreams: false, // Default to false, can be extended later
	}

	// Set OutputTokensBaseValue if strategy is FIXED
	if reqConfig.GetOutputTokensStrategy() == config.OutputTokensStrategyFixed {
		value := reqConfig.GetOutputTokensBaseValue()
		if value > 0 {
			processingConfig.OutputTokensBaseValue = &value
		}
	}

	return processingConfig
}

// ValidateConfig validates a processing configuration
func ValidateConfig(processingConfig *extensions.ProcessingConfig) error {
	if processingConfig == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate max tokens configuration
	if processingConfig.OutputTokensStrategy == config.OutputTokensStrategyFixed && processingConfig.OutputTokensBaseValue == nil {
		return fmt.Errorf("OutputTokensBaseValue is required when OutputTokensStrategy is FIXED")
	}

	if processingConfig.OutputTokensBaseValue != nil && *processingConfig.OutputTokensBaseValue <= 0 {
		return fmt.Errorf("OutputTokensBaseValue must be positive, got: %d", *processingConfig.OutputTokensBaseValue)
	}

	return nil
}
