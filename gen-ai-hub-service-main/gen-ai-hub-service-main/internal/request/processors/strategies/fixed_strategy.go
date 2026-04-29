/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import (
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"math"
)

// FixedTokenStrategy implements the FIXED strategy for token adjustment
type FixedTokenStrategy struct {
	configValue int
}

// NewFixedTokenStrategy creates a new FixedTokenStrategy
func NewFixedTokenStrategy(configValue int) *FixedTokenStrategy {
	return &FixedTokenStrategy{
		configValue: configValue,
	}
}

func (s *FixedTokenStrategy) ShouldAdjust(originalTokens *int, forceAdjustment bool) bool {
	// Case 2.1: max_tokens exists in request
	if originalTokens != nil {
		// Case 2.1.1: OutputTokensAdjustmentForced=false -> do nothing
		if !forceAdjustment {
			return false
		}
		// Case 2.1.2: OutputTokensAdjustmentForced=true -> update
		return true
	}

	// Case 2.2: max_tokens not in request -> always insert
	return true
}

func (s *FixedTokenStrategy) CalculateAdjustedValue(originalTokens *int, modelMaximum *float64, configValue int) *int {
	// Use the minimum of "suggested fixed" and "model maximum"
	adjustedValue := configValue

	if modelMaximum != nil {
		maxInt := int(math.Floor(*modelMaximum))
		if adjustedValue > maxInt {
			adjustedValue = maxInt
		}
	}

	return &adjustedValue
}

func (s *FixedTokenStrategy) GetStrategyName() config.OutputTokensStrategy {
	return config.OutputTokensStrategyFixed
}
