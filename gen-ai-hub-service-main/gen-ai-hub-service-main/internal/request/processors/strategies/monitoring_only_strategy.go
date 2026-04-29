/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

import "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"

// MonitoringOnlyTokenStrategy implements monitoring-only token behavior - collects metrics without modifying requests
type MonitoringOnlyTokenStrategy struct{}

func NewMonitoringOnlyTokenStrategy() *MonitoringOnlyTokenStrategy {
	return &MonitoringOnlyTokenStrategy{}
}

func (s *MonitoringOnlyTokenStrategy) ShouldAdjust(originalTokens *int, forceAdjustment bool) bool {
	return false
}

func (s *MonitoringOnlyTokenStrategy) CalculateAdjustedValue(originalTokens *int, modelMaximum *float64, configValue int) *int {
	return originalTokens
}

func (s *MonitoringOnlyTokenStrategy) GetStrategyName() config.OutputTokensStrategy {
	return config.OutputTokensStrategyMonitoringOnly
}
