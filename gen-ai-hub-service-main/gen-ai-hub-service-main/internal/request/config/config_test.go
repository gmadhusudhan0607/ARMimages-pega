/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package config

import (
	"context"
	"os"
	"sync"
	"testing"
)

// MockConfigProvider implements ConfigProvider for testing
type MockConfigProvider struct {
	copyrightProtection          bool
	outputTokensValue            int
	outputTokensAdjustmentForced bool
	outputTokensPercentile       int
	outputTokensStrategy         OutputTokensStrategy
}

func (m *MockConfigProvider) GetCopyrightProtection() bool {
	return m.copyrightProtection
}

func (m *MockConfigProvider) GetOutputTokensBaseValue() int {
	return m.outputTokensValue
}

func (m *MockConfigProvider) GetOutputTokensAdjustmentForced() bool {
	return m.outputTokensAdjustmentForced
}

func (m *MockConfigProvider) GetOutputTokensPercentile() int {
	return m.outputTokensPercentile
}

func (m *MockConfigProvider) GetOutputTokensStrategy() OutputTokensStrategy {
	return m.outputTokensStrategy
}

func TestMaxTokensStrategy_IsValid(t *testing.T) {
	tests := []struct {
		strategy OutputTokensStrategy
		expected bool
	}{
		{OutputTokensStrategyDisabled, true},
		{OutputTokensStrategyMonitoringOnly, true},
		{OutputTokensStrategyFixed, true},
		{OutputTokensStrategyAutoIncreasing, true},
		{OutputTokensStrategyP95, true},
		{OutputTokensStrategyP96, true},
		{OutputTokensStrategyP97, true},
		{OutputTokensStrategyP98, true},
		{OutputTokensStrategyP99, true},
		{OutputTokensStrategy("INVALID"), false},
		{OutputTokensStrategy(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			if got := tt.strategy.IsValid(); got != tt.expected {
				t.Errorf("OutputTokensStrategy.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsOutputTokensStrategyDisabled(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
		expected bool
	}{
		{
			name:     "DISABLED strategy returns true",
			strategy: "DISABLED",
			expected: true,
		},
		{
			name:     "MONITORING_ONLY strategy returns false",
			strategy: "MONITORING_ONLY",
			expected: false,
		},
		{
			name:     "FIXED strategy returns false",
			strategy: "FIXED",
			expected: false,
		},
		{
			name:     "AUTO_INCREASING strategy returns false",
			strategy: "AUTO_INCREASING",
			expected: false,
		},
		{
			name:     "P95 strategy returns false",
			strategy: "P95",
			expected: false,
		},
		{
			name:     "P99 strategy returns false",
			strategy: "P99",
			expected: false,
		},
		{
			name:     "default strategy (MONITORING_ONLY) returns false",
			strategy: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset singleton and clear environment
			ResetConfigSingleton()
			clearEnvVars()

			// Set strategy if provided
			if tt.strategy != "" {
				os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY", tt.strategy)
			}
			defer clearEnvVars()

			ctx := context.Background()
			result := IsOutputTokensStrategyDisabled(ctx)

			if result != tt.expected {
				t.Errorf("IsOutputTokensStrategyDisabled() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLoadConfigFromEnv_MaxTokensStrategy(t *testing.T) {
	// Reset singleton before test
	ResetConfigSingleton()

	tests := []struct {
		name        string
		envValue    string
		expected    OutputTokensStrategy
		expectError bool
	}{
		{
			name:        "default value when env not set",
			envValue:    "",
			expected:    OutputTokensStrategyMonitoringOnly,
			expectError: false,
		},
		{
			name:        "valid MONITORING_ONLY strategy",
			envValue:    "MONITORING_ONLY",
			expected:    OutputTokensStrategyMonitoringOnly,
			expectError: false,
		},
		{
			name:        "valid FIXED strategy",
			envValue:    "FIXED",
			expected:    OutputTokensStrategyFixed,
			expectError: false,
		},
		{
			name:        "valid P99 strategy",
			envValue:    "P99",
			expected:    OutputTokensStrategyP99,
			expectError: false,
		},
		{
			name:        "invalid strategy",
			envValue:    "INVALID_STRATEGY",
			expected:    OutputTokensStrategyMonitoringOnly,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY")

			// Set environment variable if provided
			if tt.envValue != "" {
				os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY", tt.envValue)
			}

			config, err := LoadConfigFromEnv(context.Background())

			if tt.expectError {
				if err == nil {
					t.Errorf("LoadConfigFromEnv() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("LoadConfigFromEnv() unexpected error: %v", err)
				return
			}

			if config.OutputTokensStrategy != tt.expected {
				t.Errorf("LoadConfigFromEnv() OutputTokensStrategy = %v, want %v",
					config.OutputTokensStrategy, tt.expected)
			}

			// Clean up
			os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY")
		})
	}
}

func TestLoadConfigFromEnv_MaxTokensPercentileDerivedFromStrategy(t *testing.T) {
	// Reset singleton before test
	ResetConfigSingleton()

	tests := []struct {
		name                           string
		strategyEnvValue               string
		expectedStrategy               OutputTokensStrategy
		expectedOutputTokensPercentile int
	}{
		{
			name:                           "default strategy sets percentile to 0",
			strategyEnvValue:               "",
			expectedStrategy:               OutputTokensStrategyMonitoringOnly,
			expectedOutputTokensPercentile: 0,
		},
		{
			name:                           "MONITORING_ONLY strategy sets percentile to 0",
			strategyEnvValue:               "MONITORING_ONLY",
			expectedStrategy:               OutputTokensStrategyMonitoringOnly,
			expectedOutputTokensPercentile: 0,
		},
		{
			name:                           "FIXED strategy sets percentile to 0",
			strategyEnvValue:               "FIXED",
			expectedStrategy:               OutputTokensStrategyFixed,
			expectedOutputTokensPercentile: 0,
		},
		{
			name:                           "AUTO_INCREASING strategy sets percentile to 0",
			strategyEnvValue:               "AUTO_INCREASING",
			expectedStrategy:               OutputTokensStrategyAutoIncreasing,
			expectedOutputTokensPercentile: 0,
		},
		{
			name:                           "P95 strategy sets percentile to 95",
			strategyEnvValue:               "P95",
			expectedStrategy:               OutputTokensStrategyP95,
			expectedOutputTokensPercentile: 95,
		},
		{
			name:                           "P96 strategy sets percentile to 96",
			strategyEnvValue:               "P96",
			expectedStrategy:               OutputTokensStrategyP96,
			expectedOutputTokensPercentile: 96,
		},
		{
			name:                           "P97 strategy sets percentile to 97",
			strategyEnvValue:               "P97",
			expectedStrategy:               OutputTokensStrategyP97,
			expectedOutputTokensPercentile: 97,
		},
		{
			name:                           "P98 strategy sets percentile to 98",
			strategyEnvValue:               "P98",
			expectedStrategy:               OutputTokensStrategyP98,
			expectedOutputTokensPercentile: 98,
		},
		{
			name:                           "P99 strategy sets percentile to 99",
			strategyEnvValue:               "P99",
			expectedStrategy:               OutputTokensStrategyP99,
			expectedOutputTokensPercentile: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY")

			// Set environment variable if provided
			if tt.strategyEnvValue != "" {
				os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY", tt.strategyEnvValue)
			}

			config, err := LoadConfigFromEnv(context.Background())

			if err != nil {
				t.Errorf("LoadConfigFromEnv() unexpected error: %v", err)
				return
			}

			if config.OutputTokensStrategy != tt.expectedStrategy {
				t.Errorf("LoadConfigFromEnv() OutputTokensStrategy = %v, want %v",
					config.OutputTokensStrategy, tt.expectedStrategy)
			}

			if config.OutputTokensPercentile != tt.expectedOutputTokensPercentile {
				t.Errorf("LoadConfigFromEnv() OutputTokensPercentile = %v, want %v",
					config.OutputTokensPercentile, tt.expectedOutputTokensPercentile)
			}

			// Clean up
			os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY")
		})
	}
}

func TestLoadConfigFromEnv_REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE_NotRead(t *testing.T) {
	// Reset singleton before test
	ResetConfigSingleton()

	// Clean up environment
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY")
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE")

	// Set REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE environment variable - this should be ignored
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE", "95")

	config, err := LoadConfigFromEnv(context.Background())

	if err != nil {
		t.Errorf("LoadConfigFromEnv() unexpected error: %v", err)
		return
	}

	// OutputTokensPercentile should be 0 (default) and not 95 from environment variable
	if config.OutputTokensPercentile != 0 {
		t.Errorf("LoadConfigFromEnv() OutputTokensPercentile = %v, want 0 (REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE should be ignored)",
			config.OutputTokensPercentile)
	}

	// Clean up
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE")
}

func TestLoadConfigFromEnv_REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE_IgnoredWithStrategy(t *testing.T) {
	// Reset singleton before test
	ResetConfigSingleton()

	// Clean up environment
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY")
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE")

	// Set both environment variables - REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE should be ignored
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY", "P97")
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE", "95")

	config, err := LoadConfigFromEnv(context.Background())

	if err != nil {
		t.Errorf("LoadConfigFromEnv() unexpected error: %v", err)
		return
	}

	// OutputTokensPercentile should be 97 (from strategy) and not 95 from environment variable
	if config.OutputTokensPercentile != 97 {
		t.Errorf("LoadConfigFromEnv() OutputTokensPercentile = %v, want 97 (derived from OutputTokensStrategyP97 strategy)",
			config.OutputTokensPercentile)
	}

	if config.OutputTokensStrategy != OutputTokensStrategyP97 {
		t.Errorf("LoadConfigFromEnv() OutputTokensStrategy = %v, want %v",
			config.OutputTokensStrategy, OutputTokensStrategyP97)
	}

	// Clean up
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY")
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE")
}

func TestGetConfig_DefaultValues(t *testing.T) {
	// Reset singleton before test
	ResetConfigSingleton()

	// Clean up environment to ensure defaults
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY")
	os.Unsetenv("REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE")

	config, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}

	if config.OutputTokensStrategy != OutputTokensStrategyMonitoringOnly {
		t.Errorf("GetConfig() OutputTokensStrategy = %v, want %v",
			config.OutputTokensStrategy, OutputTokensStrategyMonitoringOnly)
	}

	if config.OutputTokensPercentile != 0 {
		t.Errorf("GetConfig() OutputTokensPercentile = %v, want 0",
			config.OutputTokensPercentile)
	}
}

func TestLoadConfigFromEnv_Defaults(t *testing.T) {
	// Clear all environment variables
	clearEnvVars()

	cfg, err := LoadConfigFromEnv(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify defaults
	if cfg.CopyrightProtection != false {
		t.Errorf("Expected CopyrightProtection to be false, got: %v", cfg.CopyrightProtection)
	}
	if cfg.OutputTokensBaseValue != -1 {
		t.Errorf("Expected OutputTokensBaseValue to be -1, got: %d", cfg.OutputTokensBaseValue)
	}
	if cfg.OutputTokensAdjustmentForced != false {
		t.Errorf("Expected OutputTokensAdjustmentForced to be false, got: %v", cfg.OutputTokensAdjustmentForced)
	}
	if cfg.OutputTokensPercentile != 0 {
		t.Errorf("Expected OutputTokensPercentile to be 0, got: %d", cfg.OutputTokensPercentile)
	}
	if cfg.CacheSize != 1000 {
		t.Errorf("Expected CacheSize to be 1000, got: %d", cfg.CacheSize)
	}
}

func TestLoadConfigFromEnv_ValidValues(t *testing.T) {
	// Clear all environment variables first
	clearEnvVars()

	// Set valid environment variables
	os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "true")
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE", "500")
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED", "true")
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY", "P95")
	os.Setenv("REQUEST_PROCESSING_CACHE_SIZE", "2000")

	defer clearEnvVars()

	config, err := LoadConfigFromEnv(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify loaded values
	if config.CopyrightProtection != true {
		t.Errorf("Expected CopyrightProtection to be true, got: %v", config.CopyrightProtection)
	}
	if config.OutputTokensBaseValue != 500 {
		t.Errorf("Expected OutputTokensBaseValue to be 500, got: %d", config.OutputTokensBaseValue)
	}
	if config.OutputTokensAdjustmentForced != true {
		t.Errorf("Expected OutputTokensAdjustmentForced to be true, got: %v", config.OutputTokensAdjustmentForced)
	}
	if config.OutputTokensPercentile != 95 {
		t.Errorf("Expected OutputTokensPercentile to be 95, got: %d", config.OutputTokensPercentile)
	}
	if config.CacheSize != 2000 {
		t.Errorf("Expected CacheSize to be 2000, got: %d", config.CacheSize)
	}
}

func TestLoadConfigFromEnv_InvalidBoolValues(t *testing.T) {
	tests := []struct {
		envVar string
		value  string
	}{
		{"REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "invalid"},
		{"REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED", "not_a_bool"},
	}

	for _, test := range tests {
		clearEnvVars()
		os.Setenv(test.envVar, test.value)

		_, err := LoadConfigFromEnv(context.Background())
		if err == nil {
			t.Errorf("Expected error for invalid %s value: %s", test.envVar, test.value)
		}

		clearEnvVars()
	}
}

func TestLoadConfigFromEnv_InvalidIntValues(t *testing.T) {
	tests := []struct {
		envVar string
		value  string
	}{
		{"REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE", "not_a_number"},
		{"REQUEST_PROCESSING_CACHE_SIZE", "xyz"},
	}

	for _, test := range tests {
		clearEnvVars()
		os.Setenv(test.envVar, test.value)

		_, err := LoadConfigFromEnv(context.Background())
		if err == nil {
			t.Errorf("Expected error for invalid %s value: %s", test.envVar, test.value)
		}

		clearEnvVars()
	}
}

func TestValidate_MaxTokensPercentile(t *testing.T) {
	tests := []struct {
		value     int
		shouldErr bool
	}{
		{0, false},  // Valid: default (derived from strategy)
		{95, false}, // Valid: middle range
		{96, false}, // Valid: middle range
		{97, false}, // Valid: middle range
		{98, false}, // Valid: middle range
		{99, false}, // Valid: maximum allowed
	}

	for _, test := range tests {
		cfg := &ReqProcessingConfig{
			OutputTokensPercentile: test.value,
			CacheSize:              1000,
		}

		err := cfg.Validate()
		if test.shouldErr && err == nil {
			t.Errorf("Expected error for OutputTokensPercentile value: %d", test.value)
		}
		if !test.shouldErr && err != nil {
			t.Errorf("Unexpected error for OutputTokensPercentile value %d: %v", test.value, err)
		}
	}
}

func TestValidate_CacheSize(t *testing.T) {
	tests := []struct {
		value     int
		shouldErr bool
	}{
		{1000, false}, // Valid: default
		{2000, false}, // Valid: larger value
		{0, true},     // Invalid: zero
		{-1, true},    // Invalid: negative
	}

	for _, test := range tests {
		cfg := &ReqProcessingConfig{
			OutputTokensPercentile: -1,
			CacheSize:              test.value,
		}

		err := cfg.Validate()
		if test.shouldErr && err == nil {
			t.Errorf("Expected error for CacheMaxSamples value: %d", test.value)
		}
		if !test.shouldErr && err != nil {
			t.Errorf("Unexpected error for CacheMaxSamples value %d: %v", test.value, err)
		}
	}
}

func TestLoadConfigFromEnv_ValidationFailure(t *testing.T) {
	clearEnvVars()

	// Set invalid cache configuration that will cause validation error
	os.Setenv("REQUEST_PROCESSING_CACHE_SIZE", "0")
	defer clearEnvVars()

	_, err := LoadConfigFromEnv(context.Background())
	if err == nil {
		t.Error("Expected validation error for invalid cache configuration")
	}
}

func TestLoadConfigFromEnv_PartialConfiguration(t *testing.T) {
	clearEnvVars()

	// Set only some environment variables
	os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "true")
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE", "200")
	defer clearEnvVars()

	config, err := LoadConfigFromEnv(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify mixed values (some from env, some defaults)
	if config.CopyrightProtection != true {
		t.Errorf("Expected CopyrightProtection to be true, got: %v", config.CopyrightProtection)
	}

	if config.OutputTokensBaseValue != 200 {
		t.Errorf("Expected OutputTokensBaseValue to be 200 (from env), got: %d", config.OutputTokensBaseValue)
	}

	// These should be defaults
	if config.CacheSize != 1000 {
		t.Errorf("Expected CacheSize to be 1000 (default), got: %d", config.CacheSize)
	}
}

func TestGetConfig_Singleton(t *testing.T) {
	// Reset singleton for test
	resetConfigSingleton()
	clearEnvVars()

	// Set some environment variables
	os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "true")
	os.Setenv("REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE", "100")
	defer clearEnvVars()

	// Get config multiple times
	config1, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}
	config2, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}

	// Should be the same instance
	if config1 != config2 {
		t.Error("GetConfig should return the same instance (singleton)")
	}

	// Verify values are loaded correctly
	if !config1.CopyrightProtection {
		t.Error("Expected CopyrightProtection to be true")
	}
	if config1.OutputTokensBaseValue != 100 {
		t.Errorf("Expected OutputTokensBaseValue to be 100, got: %d", config1.OutputTokensBaseValue)
	}
}

func TestGetConfig_LoadOnce(t *testing.T) {
	// Reset singleton for test
	resetConfigSingleton()
	clearEnvVars()

	// Set initial environment
	os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "true")

	// Get config first time
	config1, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}
	if !config1.CopyrightProtection {
		t.Error("Expected CopyrightProtection to be true")
	}

	// Change environment variable
	os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "false")

	// Get config again - should still have old value (loaded once)
	config2, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}
	if !config2.CopyrightProtection {
		t.Error("Expected CopyrightProtection to still be true (loaded once)")
	}

	clearEnvVars()
}

func TestGetConfig_DefaultsOnError(t *testing.T) {
	// Reset singleton for test
	resetConfigSingleton()
	clearEnvVars()

	// Set invalid environment variable that will cause validation error
	os.Setenv("REQUEST_PROCESSING_CACHE_SIZE", "0") // Invalid value
	defer clearEnvVars()

	// Should return error when loading fails
	_, err := GetConfig(context.Background())
	if err == nil {
		t.Error("Expected error when loading config with invalid environment")
	}
}

func TestGetConfig_ConcurrentAccess(t *testing.T) {
	// Reset singleton for test
	resetConfigSingleton()
	clearEnvVars()

	os.Setenv("REQUEST_PROCESSING_COPYRIGHT_PROTECTION", "true")
	defer clearEnvVars()

	const numGoroutines = 10
	configs := make([]*ReqProcessingConfig, numGoroutines)
	var wg sync.WaitGroup

	// Start multiple goroutines to access config concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			config, err := GetConfig(context.Background())
			if err != nil {
				t.Errorf("GetConfig() unexpected error: %v", err)
				return
			}
			configs[index] = config
		}(i)
	}

	wg.Wait()

	// All should be the same instance
	firstConfig := configs[0]
	for i := 1; i < numGoroutines; i++ {
		if configs[i] != firstConfig {
			t.Errorf("Config instance %d is different from first instance", i)
		}
	}

	// Verify the config is correct
	if !firstConfig.CopyrightProtection {
		t.Error("Expected CopyrightProtection to be true")
	}
}

// Helper function to reset the singleton for testing
func resetConfigSingleton() {
	ResetConfigSingleton()
}

func TestCopyrightProtectionConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "default copyright protection disabled",
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name: "copyright protection enabled",
			envVars: map[string]string{
				"REQUEST_PROCESSING_COPYRIGHT_PROTECTION": "true",
			},
			expected: true,
		},
		{
			name: "copyright protection disabled explicitly",
			envVars: map[string]string{
				"REQUEST_PROCESSING_COPYRIGHT_PROTECTION": "false",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables first
			clearEnvVars()

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer clearEnvVars()

			config, err := LoadConfigFromEnv(context.Background())
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if config.GetCopyrightProtection() != tt.expected {
				t.Errorf("Expected copyright protection %t, got %t", tt.expected, config.GetCopyrightProtection())
			}
		})
	}
}

// Helper function to clear all relevant environment variables
func clearEnvVars() {
	envVars := []string{
		"REQUEST_PROCESSING_COPYRIGHT_PROTECTION",
		"REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE",
		"REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED",
		"REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY",
		"REQUEST_PROCESSING_OUTPUT_TOKENS_PERCENTILE",
		"REQUEST_PROCESSING_CACHE_SIZE",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

// Tests moved from internal/request/modifiers/base/config_test.go

func TestDefaultConfigProvider(t *testing.T) {
	// Reset config singleton to ensure clean test
	ResetConfigSingleton()
	clearEnvVars()

	provider, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}

	// Test all methods return expected default values
	if provider.GetCopyrightProtection() != false {
		t.Error("Expected default copyright protection to be false")
	}

	if provider.GetOutputTokensBaseValue() != -1 {
		t.Error("Expected default max tokens value to be -1")
	}

	if provider.GetOutputTokensAdjustmentForced() != false {
		t.Error("Expected default max tokens override original to be false")
	}

	if provider.GetOutputTokensPercentile() != 0 {
		t.Error("Expected default max tokens percentile to be 0")
	}
}

func TestProvider_ConfigProviderNeverNull(t *testing.T) {
	// Test that configProvider is never null in any scenario
	// Test that we can call all config methods without panic
	configProvider, err := GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig() unexpected error: %v", err)
	}
	_ = configProvider.GetCopyrightProtection()
	_ = configProvider.GetOutputTokensBaseValue()
	_ = configProvider.GetOutputTokensAdjustmentForced()
	_ = configProvider.GetOutputTokensPercentile()
}

func TestProvider_ConfigProviderInterface(t *testing.T) {
	// Test that we can replace it with a custom implementation
	customConfig := &MockConfigProvider{
		copyrightProtection:          true,
		outputTokensValue:            500,
		outputTokensAdjustmentForced: true,
		outputTokensPercentile:       95,
	}

	// Verify the custom config is working
	if customConfig.GetCopyrightProtection() != true {
		t.Error("Expected custom config copyright protection to be true")
	}
	if customConfig.GetOutputTokensBaseValue() != 500 {
		t.Error("Expected custom config max tokens value to be 500")
	}
	if customConfig.GetOutputTokensAdjustmentForced() != true {
		t.Error("Expected custom config force max tokens adjustment to be true")
	}
	if customConfig.GetOutputTokensPercentile() != 95 {
		t.Error("Expected custom config max tokens percentile to be 95")
	}
}
