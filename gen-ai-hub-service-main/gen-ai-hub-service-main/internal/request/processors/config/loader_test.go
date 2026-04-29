/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package config

import (
	"os"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
)

func TestNewConfigLoader(t *testing.T) {
	loader := NewConfigLoader()

	if loader == nil {
		t.Fatal("Expected config loader, got nil")
	} else if loader.modelOverrides == nil {
		t.Error("Expected modelOverrides to be initialized")
	} else if len(loader.modelOverrides) != 0 {
		t.Errorf("Expected empty modelOverrides, got %d items", len(loader.modelOverrides))
	}
}

func TestConfigLoader_GetConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupLoader    func() *ConfigLoader
		model          *types.Model
		expectError    bool
		expectedConfig *extensions.ProcessingConfig
	}{
		{
			name: "nil model with default config",
			setupLoader: func() *ConfigLoader {
				loader := NewConfigLoader()
				loader.defaultConfig = &extensions.ProcessingConfig{
					OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
				}
				return loader
			},
			model:       nil,
			expectError: false,
			expectedConfig: &extensions.ProcessingConfig{
				OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
			},
		},
		{
			name: "nil model without default config",
			setupLoader: func() *ConfigLoader {
				return NewConfigLoader()
			},
			model:       nil,
			expectError: true,
		},
		{
			name: "model with specific override",
			setupLoader: func() *ConfigLoader {
				loader := NewConfigLoader()
				loader.defaultConfig = &extensions.ProcessingConfig{
					OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
				}
				loader.modelOverrides["azure_openai_gpt-4"] = &extensions.ProcessingConfig{
					OutputTokensStrategy: config.OutputTokensStrategyFixed,
				}
				return loader
			},
			model: &types.Model{
				Provider: "azure",
				Creator:  "openai",
				Name:     "gpt-4",
			},
			expectError: false,
			expectedConfig: &extensions.ProcessingConfig{
				OutputTokensStrategy: config.OutputTokensStrategyFixed,
			},
		},
		{
			name: "model with provider-creator override",
			setupLoader: func() *ConfigLoader {
				loader := NewConfigLoader()
				loader.defaultConfig = &extensions.ProcessingConfig{
					OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
				}
				loader.modelOverrides["azure_openai"] = &extensions.ProcessingConfig{
					OutputTokensStrategy: config.OutputTokensStrategyFixed,
				}
				return loader
			},
			model: &types.Model{
				Provider: "azure",
				Creator:  "openai",
				Name:     "gpt-3.5",
			},
			expectError: false,
			expectedConfig: &extensions.ProcessingConfig{
				OutputTokensStrategy: config.OutputTokensStrategyFixed,
			},
		},
		{
			name: "model with provider override",
			setupLoader: func() *ConfigLoader {
				loader := NewConfigLoader()
				loader.defaultConfig = &extensions.ProcessingConfig{
					OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
				}
				loader.modelOverrides["azure"] = &extensions.ProcessingConfig{
					OutputTokensStrategy: config.OutputTokensStrategyFixed,
				}
				return loader
			},
			model: &types.Model{
				Provider: "azure",
				Creator:  "microsoft",
				Name:     "phi-3",
			},
			expectError: false,
			expectedConfig: &extensions.ProcessingConfig{
				OutputTokensStrategy: config.OutputTokensStrategyFixed,
			},
		},
		{
			name: "model without overrides uses default",
			setupLoader: func() *ConfigLoader {
				loader := NewConfigLoader()
				loader.defaultConfig = &extensions.ProcessingConfig{
					OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
				}
				return loader
			},
			model: &types.Model{
				Provider: "aws",
				Creator:  "anthropic",
				Name:     "claude",
			},
			expectError: false,
			expectedConfig: &extensions.ProcessingConfig{
				OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
			},
		},
		{
			name: "model without overrides and no default config",
			setupLoader: func() *ConfigLoader {
				return NewConfigLoader()
			},
			model: &types.Model{
				Provider: "aws",
				Creator:  "anthropic",
				Name:     "claude",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := tt.setupLoader()
			config, err := loader.GetConfig(tt.model)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if config == nil {
					t.Fatal("Expected config, got nil")
				} else if config.OutputTokensStrategy != tt.expectedConfig.OutputTokensStrategy {
					t.Errorf("Expected strategy %s, got %s", tt.expectedConfig.OutputTokensStrategy, config.OutputTokensStrategy)
				}
			}
		})
	}
}

func TestParseBoolEnv(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		setEnv      bool
		expected    bool
		expectedSet bool
		expectError bool
	}{
		{
			name:        "env not set",
			setEnv:      false,
			expected:    false,
			expectedSet: false,
			expectError: false,
		},
		{
			name:        "env set to true",
			envValue:    "true",
			setEnv:      true,
			expected:    true,
			expectedSet: true,
			expectError: false,
		},
		{
			name:        "env set to false",
			envValue:    "false",
			setEnv:      true,
			expected:    false,
			expectedSet: true,
			expectError: false,
		},
		{
			name:        "env set to invalid value",
			envValue:    "invalid",
			setEnv:      true,
			expected:    false,
			expectedSet: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_BOOL_ENV"

			// Clean up environment
			defer os.Unsetenv(key)

			if tt.setEnv {
				os.Setenv(key, tt.envValue)
			}

			value, wasSet, err := parseBoolEnv(key)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if value != tt.expected {
					t.Errorf("Expected value %v, got %v", tt.expected, value)
				}
				if wasSet != tt.expectedSet {
					t.Errorf("Expected wasSet %v, got %v", tt.expectedSet, wasSet)
				}
			}
		})
	}
}

func TestParsePositiveIntEnv(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		setEnv      bool
		expected    int
		expectedSet bool
		expectError bool
	}{
		{
			name:        "env not set",
			setEnv:      false,
			expected:    0,
			expectedSet: false,
			expectError: false,
		},
		{
			name:        "env set to positive int",
			envValue:    "100",
			setEnv:      true,
			expected:    100,
			expectedSet: true,
			expectError: false,
		},
		{
			name:        "env set to zero",
			envValue:    "0",
			setEnv:      true,
			expected:    0,
			expectedSet: false,
			expectError: true,
		},
		{
			name:        "env set to negative int",
			envValue:    "-10",
			setEnv:      true,
			expected:    0,
			expectedSet: false,
			expectError: true,
		},
		{
			name:        "env set to invalid value",
			envValue:    "invalid",
			setEnv:      true,
			expected:    0,
			expectedSet: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT_ENV"

			// Clean up environment
			defer os.Unsetenv(key)

			if tt.setEnv {
				os.Setenv(key, tt.envValue)
			}

			value, wasSet, err := parsePositiveIntEnv(key)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if value != tt.expected {
					t.Errorf("Expected value %d, got %d", tt.expected, value)
				}
				if wasSet != tt.expectedSet {
					t.Errorf("Expected wasSet %v, got %v", tt.expectedSet, wasSet)
				}
			}
		})
	}
}

func TestParseStrategyEnv(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		setEnv      bool
		expected    config.OutputTokensStrategy
		expectedSet bool
		expectError bool
	}{
		{
			name:        "env not set",
			setEnv:      false,
			expected:    config.OutputTokensStrategyMonitoringOnly,
			expectedSet: false,
			expectError: false,
		},
		{
			name:        "env set to MONITORING_ONLY",
			envValue:    "MONITORING_ONLY",
			setEnv:      true,
			expected:    config.OutputTokensStrategyMonitoringOnly,
			expectedSet: true,
			expectError: false,
		},
		{
			name:        "env set to monitoring_only (lowercase)",
			envValue:    "monitoring_only",
			setEnv:      true,
			expected:    config.OutputTokensStrategyMonitoringOnly,
			expectedSet: true,
			expectError: false,
		},
		{
			name:        "env set to FIXED",
			envValue:    "FIXED",
			setEnv:      true,
			expected:    config.OutputTokensStrategyFixed,
			expectedSet: true,
			expectError: false,
		},
		{
			name:        "env set to fixed (lowercase)",
			envValue:    "fixed",
			setEnv:      true,
			expected:    config.OutputTokensStrategyFixed,
			expectedSet: true,
			expectError: false,
		},
		{
			name:        "env set to invalid value",
			envValue:    "invalid",
			setEnv:      true,
			expected:    "",
			expectedSet: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_STRATEGY_ENV"

			// Clean up environment
			defer os.Unsetenv(key)

			if tt.setEnv {
				os.Setenv(key, tt.envValue)
			}

			value, wasSet, err := parseStrategyEnv(key)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if value != tt.expected {
					t.Errorf("Expected value %s, got %s", tt.expected, value)
				}
				if wasSet != tt.expectedSet {
					t.Errorf("Expected wasSet %v, got %v", tt.expectedSet, wasSet)
				}
			}
		})
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	// Save original environment
	originalEnvs := map[string]string{
		"REQUEST_PROCESSING_COPYRIGHT_PROTECTION":                  os.Getenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION"),
		"GENAI_PROCESSING_MAX_TOKENS_STRATEGY":                     os.Getenv("GENAI_PROCESSING_MAX_TOKENS_STRATEGY"),
		"GENAI_PROCESSING_MAX_TOKENS_VALUE":                        os.Getenv("GENAI_PROCESSING_MAX_TOKENS_VALUE"),
		"GENAI_PROCESSING_FORCE_ADJUSTMENT":                        os.Getenv("GENAI_PROCESSING_FORCE_ADJUSTMENT"),
		"REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_IN_STREAMING": os.Getenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_IN_STREAMING"),
	}

	// Clean up environment after test
	defer func() {
		for key, value := range originalEnvs {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear all environment variables first
	for key := range originalEnvs {
		os.Unsetenv(key)
	}

	t.Run("default configuration", func(t *testing.T) {
		config, err := loadDefaultConfig()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if config == nil {
			t.Fatal("Expected config, got nil")
		} else {
			if config.OutputTokensStrategy != "MONITORING_ONLY" {
				t.Errorf("Expected default strategy MONITORING_ONLY, got %s", config.OutputTokensStrategy)
			}
			if config.CopyrightProtectionEnabled != false {
				t.Error("Expected default copyright protection to be false")
			}
			if config.OutputTokensAdjustmentForced != false {
				t.Error("Expected default forced adjustment to be false")
			}
			if config.OutputTokensAdjustmentStreams != false {
				t.Error("Expected default streams adjustment to be false")
			}
		}
	})

	t.Run("with environment variables set", func(t *testing.T) {
		os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "true")
		os.Setenv("GENAI_PROCESSING_MAX_TOKENS_STRATEGY", "FIXED")
		os.Setenv("GENAI_PROCESSING_MAX_TOKENS_VALUE", "1000")
		os.Setenv("GENAI_PROCESSING_FORCE_ADJUSTMENT", "true")
		os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_IN_STREAMING", "true")

		config, err := loadDefaultConfig()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if config.CopyrightProtectionEnabled != true {
			t.Error("Expected copyright protection to be true")
		}
		if config.OutputTokensStrategy != "FIXED" {
			t.Errorf("Expected strategy FIXED, got %s", config.OutputTokensStrategy)
		}
		if config.OutputTokensBaseValue == nil || *config.OutputTokensBaseValue != 1000 {
			t.Errorf("Expected base value 1000, got %v", config.OutputTokensBaseValue)
		}
		if config.OutputTokensAdjustmentForced != true {
			t.Error("Expected forced adjustment to be true")
		}
		if config.OutputTokensAdjustmentStreams != true {
			t.Error("Expected streams adjustment to be true")
		}
	})

	t.Run("with invalid environment variable", func(t *testing.T) {
		os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "invalid")

		_, err := loadDefaultConfig()

		if err == nil {
			t.Error("Expected error for invalid environment variable")
		}
	})
}

func TestLoadModelOverrides(t *testing.T) {
	loader := NewConfigLoader()
	err := loader.loadModelOverrides()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	// Currently this function is empty, so just ensure it doesn't error
}

func TestCreateProcessingConfigFromReqConfig(t *testing.T) {
	tests := []struct {
		name      string
		reqConfig *config.ReqProcessingConfig
		expected  *extensions.ProcessingConfig
	}{
		{
			name:      "nil reqConfig",
			reqConfig: nil,
			expected: &extensions.ProcessingConfig{
				OutputTokensStrategy:          config.OutputTokensStrategyMonitoringOnly,
				CopyrightProtectionEnabled:    false,
				OutputTokensAdjustmentForced:  false,
				OutputTokensAdjustmentStreams: false,
			},
		},
		{
			name: "reqConfig with disabled strategy",
			reqConfig: &config.ReqProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyMonitoringOnly,
				CopyrightProtection:          true,
				OutputTokensAdjustmentForced: true,
			},
			expected: &extensions.ProcessingConfig{
				OutputTokensStrategy:          config.OutputTokensStrategyMonitoringOnly,
				CopyrightProtectionEnabled:    true,
				OutputTokensAdjustmentForced:  true,
				OutputTokensAdjustmentStreams: false,
			},
		},
		{
			name: "reqConfig with fixed strategy and base value",
			reqConfig: &config.ReqProcessingConfig{
				OutputTokensStrategy:         config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:        1000,
				CopyrightProtection:          false,
				OutputTokensAdjustmentForced: false,
			},
			expected: &extensions.ProcessingConfig{
				OutputTokensStrategy:          config.OutputTokensStrategyFixed,
				OutputTokensBaseValue:         intPtr(1000),
				CopyrightProtectionEnabled:    false,
				OutputTokensAdjustmentForced:  false,
				OutputTokensAdjustmentStreams: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateProcessingConfigFromReqConfig(tt.reqConfig)

			if result == nil {
				t.Fatal("Expected result, got nil")
			} else {
				if result.OutputTokensStrategy != tt.expected.OutputTokensStrategy {
					t.Errorf("Expected strategy %s, got %s", tt.expected.OutputTokensStrategy, result.OutputTokensStrategy)
				}
				if result.CopyrightProtectionEnabled != tt.expected.CopyrightProtectionEnabled {
					t.Errorf("Expected copyright protection %v, got %v", tt.expected.CopyrightProtectionEnabled, result.CopyrightProtectionEnabled)
				}
				if result.OutputTokensAdjustmentForced != tt.expected.OutputTokensAdjustmentForced {
					t.Errorf("Expected forced adjustment %v, got %v", tt.expected.OutputTokensAdjustmentForced, result.OutputTokensAdjustmentForced)
				}
				if result.OutputTokensAdjustmentStreams != tt.expected.OutputTokensAdjustmentStreams {
					t.Errorf("Expected streams adjustment %v, got %v", tt.expected.OutputTokensAdjustmentStreams, result.OutputTokensAdjustmentStreams)
				}
			}

			// Check base value
			if tt.expected.OutputTokensBaseValue == nil {
				if result.OutputTokensBaseValue != nil {
					t.Errorf("Expected nil base value, got %v", *result.OutputTokensBaseValue)
				}
			} else {
				if result.OutputTokensBaseValue == nil {
					t.Error("Expected base value, got nil")
				} else if *result.OutputTokensBaseValue != *tt.expected.OutputTokensBaseValue {
					t.Errorf("Expected base value %d, got %d", *tt.expected.OutputTokensBaseValue, *result.OutputTokensBaseValue)
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *extensions.ProcessingConfig
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "valid config with disabled strategy",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy: config.OutputTokensStrategyMonitoringOnly,
			},
			expectError: false,
		},
		{
			name: "valid config with fixed strategy and base value",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:  config.OutputTokensStrategyFixed,
				OutputTokensBaseValue: intPtr(1000),
			},
			expectError: false,
		},
		{
			name: "invalid config with fixed strategy but no base value",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy: config.OutputTokensStrategyFixed,
			},
			expectError: true,
		},
		{
			name: "invalid config with negative base value",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:  config.OutputTokensStrategyFixed,
				OutputTokensBaseValue: intPtr(-100),
			},
			expectError: true,
		},
		{
			name: "invalid config with zero base value",
			config: &extensions.ProcessingConfig{
				OutputTokensStrategy:  config.OutputTokensStrategyFixed,
				OutputTokensBaseValue: intPtr(0),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	// Save original environment
	originalEnvs := map[string]string{
		"REQUEST_PROCESSING_COPYRIGHT_PROTECTION":                  os.Getenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION"),
		"GENAI_PROCESSING_MAX_TOKENS_STRATEGY":                     os.Getenv("GENAI_PROCESSING_MAX_TOKENS_STRATEGY"),
		"GENAI_PROCESSING_MAX_TOKENS_VALUE":                        os.Getenv("GENAI_PROCESSING_MAX_TOKENS_VALUE"),
		"GENAI_PROCESSING_FORCE_ADJUSTMENT":                        os.Getenv("GENAI_PROCESSING_FORCE_ADJUSTMENT"),
		"REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_IN_STREAMING": os.Getenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_IN_STREAMING"),
	}

	// Clean up environment after test
	defer func() {
		for key, value := range originalEnvs {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Clear all environment variables first
	for key := range originalEnvs {
		os.Unsetenv(key)
	}

	t.Run("successful load", func(t *testing.T) {
		loader, err := LoadFromEnvironment()

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if loader == nil {
			t.Fatal("Expected loader, got nil")
		} else {
			if loader.defaultConfig == nil {
				t.Error("Expected default config to be set")
			}
			if loader.modelOverrides == nil {
				t.Error("Expected modelOverrides to be initialized")
			}
		}
	})

	t.Run("invalid environment variable", func(t *testing.T) {
		os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "invalid")

		_, err := LoadFromEnvironment()

		if err == nil {
			t.Error("Expected error for invalid environment variable")
		}
	})
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
