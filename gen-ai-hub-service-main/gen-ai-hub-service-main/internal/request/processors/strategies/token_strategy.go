/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import (
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
)

// TokenAdjustmentStrategy defines the interface for token adjustment strategies
type TokenAdjustmentStrategy interface {
	// ShouldAdjust determines if tokens should be adjusted based on current state
	ShouldAdjust(originalTokens *int, forceAdjustment bool) bool

	// CalculateAdjustedValue computes the new token value
	CalculateAdjustedValue(originalTokens *int, modelMaximum *float64, configValue int) *int

	// GetStrategyName returns the strategy identifier
	GetStrategyName() config.OutputTokensStrategy
}
