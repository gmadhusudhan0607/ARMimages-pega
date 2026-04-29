/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"time"
)

// RequestMetrics captures request processing metrics with consistent optional handling
type RequestMetrics struct {
	TokenMetrics     TokenMetrics  `json:"tokens,omitempty"`
	TimingMetrics    TimingMetrics `json:"timing,omitempty"`
	RetryMetrics     RetryMetrics  `json:"retry,omitempty"`
	CollectionErrors []string      `json:"collection_errors,omitempty"`
	IsStreaming      bool          `json:"is_streaming,omitempty"` // True if this is a streaming request
}

type TokenMetrics struct {
	// Core token fields
	Default   *float64 `json:"default,omitempty"`
	Requested *float64 `json:"requested,omitempty"`
	Adjusted  *float64 `json:"adjusted,omitempty"`
	Used      *float64 `json:"used,omitempty"`
	Maximum   *float64 `json:"maximum,omitempty"`

	// Reasoning token fields (for reasoning models like o1, o3, o4-mini, GPT-5, Gemini 2.5)
	ReasoningTokens *float64 `json:"reasoning_tokens,omitempty"`

	// Efficiency fields (previously in separate EfficiencyMetrics struct)
	AdjustmentEfficiencyRatio    *float64 `json:"adjustment_efficiency_ratio,omitempty"`
	AdjustmentEfficiencyCategory *string  `json:"adjustment_efficiency_category,omitempty"` // "optimal", "over_allocated", "under_allocated"
	AdjustmentAccuracy           *float64 `json:"adjustment_accuracy,omitempty"`
	AdjustedWasted               *float64 `json:"adjusted_wasted,omitempty"`
	OriginalWasted               *float64 `json:"original_wasted,omitempty"`
}

type TimingMetrics struct {
	StartTime time.Time     `json:"start_time,omitempty"`
	EndTime   time.Time     `json:"end_time,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
}

type RetryMetrics struct {
	Count             int     `json:"count,omitempty"`
	Reason            *string `json:"reason,omitempty"`
	ResponseTruncated bool    `json:"response_truncated,omitempty"`
}

func NewRequestMetrics() RequestMetrics {
	return RequestMetrics{
		TokenMetrics:     TokenMetrics{},
		TimingMetrics:    TimingMetrics{},
		RetryMetrics:     RetryMetrics{},
		CollectionErrors: make([]string, 0),
	}
}
