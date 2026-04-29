/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import (
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

func TestNewMonitoringOnlyTokenStrategy(t *testing.T) {
	strategy := NewMonitoringOnlyTokenStrategy()
	if strategy == nil {
		t.Fatal("NewMonitoringOnlyTokenStrategy should not return nil")
	}
}

func TestMonitoringOnlyTokenStrategy_ShouldAdjust(t *testing.T) {
	strategy := NewMonitoringOnlyTokenStrategy()

	tests := []struct {
		name            string
		originalTokens  *int
		forceAdjustment bool
		expected        bool
	}{
		{
			name:            "no original tokens, no force",
			originalTokens:  nil,
			forceAdjustment: false,
			expected:        false,
		},
		{
			name:            "no original tokens, with force",
			originalTokens:  nil,
			forceAdjustment: true,
			expected:        false,
		},
		{
			name:            "with original tokens, no force",
			originalTokens:  intPtr(100),
			forceAdjustment: false,
			expected:        false,
		},
		{
			name:            "with original tokens, with force",
			originalTokens:  intPtr(100),
			forceAdjustment: true,
			expected:        false,
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

func TestMonitoringOnlyTokenStrategy_CalculateAdjustedValue(t *testing.T) {
	strategy := NewMonitoringOnlyTokenStrategy()

	tests := []struct {
		name           string
		originalTokens *int
		modelMaximum   *float64
		configValue    int
		expected       *int
	}{
		{
			name:           "nil original tokens",
			originalTokens: nil,
			modelMaximum:   float64Ptr(1000.0),
			configValue:    500,
			expected:       nil,
		},
		{
			name:           "with original tokens",
			originalTokens: intPtr(200),
			modelMaximum:   float64Ptr(1000.0),
			configValue:    500,
			expected:       intPtr(200),
		},
		{
			name:           "original tokens with nil model maximum",
			originalTokens: intPtr(300),
			modelMaximum:   nil,
			configValue:    400,
			expected:       intPtr(300),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CalculateAdjustedValue(tt.originalTokens, tt.modelMaximum, tt.configValue)
			if !equalIntPtr(result, tt.expected) {
				t.Errorf("CalculateAdjustedValue() = %v, want %v", ptrToString(result), ptrToString(tt.expected))
			}
		})
	}
}

func TestMonitoringOnlyTokenStrategy_GetStrategyName(t *testing.T) {
	strategy := NewMonitoringOnlyTokenStrategy()
	result := strategy.GetStrategyName()
	expected := config.OutputTokensStrategyMonitoringOnly

	if result != expected {
		t.Errorf("GetStrategyName() = %v, want %v", result, expected)
	}
}

// Helper functions for testing
func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrToString(ptr *int) string {
	if ptr == nil {
		return "<nil>"
	}
	return string(rune(*ptr))
}
