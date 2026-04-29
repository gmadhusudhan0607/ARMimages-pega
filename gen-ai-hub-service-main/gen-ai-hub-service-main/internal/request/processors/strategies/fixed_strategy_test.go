/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import (
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

func TestNewFixedTokenStrategy(t *testing.T) {
	configValue := 1500
	strategy := NewFixedTokenStrategy(configValue)

	if strategy == nil {
		t.Fatal("NewFixedTokenStrategy should not return nil")
	} else if strategy.configValue != configValue {
		t.Errorf("Expected configValue to be %d, got %d", configValue, strategy.configValue)
	}
}

func TestFixedTokenStrategy_ShouldAdjust(t *testing.T) {
	strategy := NewFixedTokenStrategy(1500)

	tests := []struct {
		name            string
		originalTokens  *int
		forceAdjustment bool
		expected        bool
	}{
		{
			name:            "no original tokens, no force - should adjust",
			originalTokens:  nil,
			forceAdjustment: false,
			expected:        true,
		},
		{
			name:            "no original tokens, with force - should adjust",
			originalTokens:  nil,
			forceAdjustment: true,
			expected:        true,
		},
		{
			name:            "with original tokens, no force - should not adjust",
			originalTokens:  intPtr(100),
			forceAdjustment: false,
			expected:        false,
		},
		{
			name:            "with original tokens, with force - should adjust",
			originalTokens:  intPtr(100),
			forceAdjustment: true,
			expected:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.ShouldAdjust(tt.originalTokens, tt.forceAdjustment)
			if result != tt.expected {
				t.Errorf("ShouldAdjust() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFixedTokenStrategy_CalculateAdjustedValue(t *testing.T) {
	strategy := NewFixedTokenStrategy(1500)

	tests := []struct {
		name           string
		originalTokens *int
		modelMaximum   *float64
		configValue    int
		expected       int
	}{
		{
			name:           "use config value when no model maximum",
			originalTokens: nil,
			modelMaximum:   nil,
			configValue:    2000,
			expected:       2000,
		},
		{
			name:           "apply model maximum limit when config exceeds limit",
			originalTokens: nil,
			modelMaximum:   float64Ptr(1800.0),
			configValue:    2000,
			expected:       1800,
		},
		{
			name:           "use config value when under model maximum",
			originalTokens: nil,
			modelMaximum:   float64Ptr(2500.0),
			configValue:    2000,
			expected:       2000,
		},
		{
			name:           "handle fractional model maximum",
			originalTokens: nil,
			modelMaximum:   float64Ptr(1999.7),
			configValue:    2000,
			expected:       1999, // Floor of 1999.7
		},
		{
			name:           "original tokens ignored in calculation",
			originalTokens: intPtr(500),
			modelMaximum:   nil,
			configValue:    2000,
			expected:       2000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CalculateAdjustedValue(tt.originalTokens, tt.modelMaximum, tt.configValue)
			if result == nil {
				t.Fatal("CalculateAdjustedValue should not return nil")
			} else if *result != tt.expected {
				t.Errorf("CalculateAdjustedValue() = %d, want %d", *result, tt.expected)
			}
		})
	}
}

func TestFixedTokenStrategy_GetStrategyName(t *testing.T) {
	strategy := NewFixedTokenStrategy(1500)
	result := strategy.GetStrategyName()
	expected := config.OutputTokensStrategyFixed

	if result != expected {
		t.Errorf("GetStrategyName() = %v, want %v", result, expected)
	}
}
