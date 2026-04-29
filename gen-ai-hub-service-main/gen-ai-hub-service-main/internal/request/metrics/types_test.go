/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRequestMetrics(t *testing.T) {
	metrics := NewRequestMetrics()

	// Verify all fields are initialized to their zero values
	assert.Equal(t, TokenMetrics{}, metrics.TokenMetrics)
	assert.Equal(t, TimingMetrics{}, metrics.TimingMetrics)
	assert.Equal(t, RetryMetrics{}, metrics.RetryMetrics)

	// Verify CollectionErrors is initialized as empty slice, not nil
	assert.NotNil(t, metrics.CollectionErrors)
	assert.Len(t, metrics.CollectionErrors, 0)
}

func TestRequestMetrics_Structure(t *testing.T) {
	// Test that RequestMetrics has all expected fields with correct types
	metrics := &RequestMetrics{}

	// Verify field types can be set correctly
	// TokenMetrics
	requestedTokens := 1000.0
	metrics.TokenMetrics.Requested = &requestedTokens
	assert.Equal(t, 1000.0, *metrics.TokenMetrics.Requested)

	// TimingMetrics
	startTime := time.Now()
	metrics.TimingMetrics.StartTime = startTime
	assert.Equal(t, startTime, metrics.TimingMetrics.StartTime)

	// RetryMetrics
	metrics.RetryMetrics.Count = 3
	assert.Equal(t, 3, metrics.RetryMetrics.Count)

	// CollectionErrors
	metrics.CollectionErrors = []string{"error1", "error2"}
	assert.Len(t, metrics.CollectionErrors, 2)
	assert.Equal(t, "error1", metrics.CollectionErrors[0])
}

func TestTokenMetrics_Structure(t *testing.T) {
	tokenMetrics := &TokenMetrics{}

	// Test all pointer fields can be set
	def := 2000.0
	tokenMetrics.Default = &def
	assert.Equal(t, 2000.0, *tokenMetrics.Default)

	req := 1500.0
	tokenMetrics.Requested = &req
	assert.Equal(t, 1500.0, *tokenMetrics.Requested)

	adj := 1800.0
	tokenMetrics.Adjusted = &adj
	assert.Equal(t, 1800.0, *tokenMetrics.Adjusted)

	used := 1200.0
	tokenMetrics.Used = &used
	assert.Equal(t, 1200.0, *tokenMetrics.Used)

	max := 4000.0
	tokenMetrics.Maximum = &max
	assert.Equal(t, 4000.0, *tokenMetrics.Maximum)

	// Test efficiency fields
	efficiencyRatio := 1.5
	tokenMetrics.AdjustmentEfficiencyRatio = &efficiencyRatio
	assert.Equal(t, 1.5, *tokenMetrics.AdjustmentEfficiencyRatio)

	category := "optimal"
	tokenMetrics.AdjustmentEfficiencyCategory = &category
	assert.Equal(t, "optimal", *tokenMetrics.AdjustmentEfficiencyCategory)

	accuracy := 0.95
	tokenMetrics.AdjustmentAccuracy = &accuracy
	assert.Equal(t, 0.95, *tokenMetrics.AdjustmentAccuracy)

	adjustedWasted := 300.0
	tokenMetrics.AdjustedWasted = &adjustedWasted
	assert.Equal(t, 300.0, *tokenMetrics.AdjustedWasted)

	originalWasted := 500.0
	tokenMetrics.OriginalWasted = &originalWasted
	assert.Equal(t, 500.0, *tokenMetrics.OriginalWasted)
}

func TestTimingMetrics_Structure(t *testing.T) {
	timingMetrics := &TimingMetrics{}

	// Test timing fields
	startTime := time.Now()
	endTime := startTime.Add(100 * time.Millisecond)
	duration := endTime.Sub(startTime)

	timingMetrics.StartTime = startTime
	timingMetrics.EndTime = endTime
	timingMetrics.Duration = duration

	assert.Equal(t, startTime, timingMetrics.StartTime)
	assert.Equal(t, endTime, timingMetrics.EndTime)
	assert.Equal(t, duration, timingMetrics.Duration)

	// Test zero values
	emptyTiming := TimingMetrics{}
	assert.True(t, emptyTiming.StartTime.IsZero())
	assert.True(t, emptyTiming.EndTime.IsZero())
	assert.Equal(t, time.Duration(0), emptyTiming.Duration)
}

func TestRetryMetrics_Structure(t *testing.T) {
	retryMetrics := &RetryMetrics{}

	// Test retry fields
	retryMetrics.Count = 2
	assert.Equal(t, 2, retryMetrics.Count)

	reason := "rate_limit"
	retryMetrics.Reason = &reason
	assert.Equal(t, "rate_limit", *retryMetrics.Reason)

	retryMetrics.ResponseTruncated = true
	assert.True(t, retryMetrics.ResponseTruncated)

	// Test zero values
	emptyRetry := RetryMetrics{}
	assert.Equal(t, 0, emptyRetry.Count)
	assert.Nil(t, emptyRetry.Reason)
	assert.False(t, emptyRetry.ResponseTruncated)
}

func TestTokenMetrics_OptionalFields(t *testing.T) {
	// Test that all TokenMetrics fields are optional (pointers)
	tokenMetrics := TokenMetrics{}

	// All pointer fields should be nil by default
	assert.Nil(t, tokenMetrics.Default)
	assert.Nil(t, tokenMetrics.Requested)
	assert.Nil(t, tokenMetrics.Adjusted)
	assert.Nil(t, tokenMetrics.Used)
	assert.Nil(t, tokenMetrics.Maximum)
	assert.Nil(t, tokenMetrics.AdjustmentEfficiencyRatio)
	assert.Nil(t, tokenMetrics.AdjustmentEfficiencyCategory)
	assert.Nil(t, tokenMetrics.AdjustmentAccuracy)
	assert.Nil(t, tokenMetrics.AdjustedWasted)
	assert.Nil(t, tokenMetrics.OriginalWasted)
}

func TestRetryMetrics_OptionalFields(t *testing.T) {
	// Test that some RetryMetrics fields are optional
	retryMetrics := RetryMetrics{}

	// Reason should be nil by default (optional)
	assert.Nil(t, retryMetrics.Reason)

	// Count and ResponseTruncated should have zero values
	assert.Equal(t, 0, retryMetrics.Count)
	assert.False(t, retryMetrics.ResponseTruncated)
}

func TestRequestMetrics_Integration(t *testing.T) {
	// Test creating a fully populated RequestMetrics
	requested := 1000.0
	used := 800.0
	adjusted := 1200.0
	maximum := 4000.0

	metrics := RequestMetrics{
		TokenMetrics: TokenMetrics{
			Requested: &requested,
			Used:      &used,
			Adjusted:  &adjusted,
			Maximum:   &maximum,
		},
		TimingMetrics: TimingMetrics{
			StartTime: time.Now().Add(-100 * time.Millisecond),
			EndTime:   time.Now(),
			Duration:  100 * time.Millisecond,
		},
		RetryMetrics: RetryMetrics{
			Count:             1,
			ResponseTruncated: false,
		},
		CollectionErrors: []string{"warning: high latency"},
	}

	// Verify all values are set correctly
	assert.Equal(t, 1000.0, *metrics.TokenMetrics.Requested)
	assert.Equal(t, 800.0, *metrics.TokenMetrics.Used)
	assert.Equal(t, 1200.0, *metrics.TokenMetrics.Adjusted)
	assert.Equal(t, 4000.0, *metrics.TokenMetrics.Maximum)

	assert.False(t, metrics.TimingMetrics.StartTime.IsZero())
	assert.False(t, metrics.TimingMetrics.EndTime.IsZero())
	assert.Equal(t, 100*time.Millisecond, metrics.TimingMetrics.Duration)

	assert.Equal(t, 1, metrics.RetryMetrics.Count)
	assert.False(t, metrics.RetryMetrics.ResponseTruncated)

	assert.Len(t, metrics.CollectionErrors, 1)
	assert.Equal(t, "warning: high latency", metrics.CollectionErrors[0])
}

func TestTokenMetrics_EfficiencyCategories(t *testing.T) {
	// Test different efficiency categories
	testCases := []string{
		"optimal",
		"over_allocated",
		"under_allocated",
	}

	for _, category := range testCases {
		tokenMetrics := TokenMetrics{
			AdjustmentEfficiencyCategory: &category,
		}

		assert.Equal(t, category, *tokenMetrics.AdjustmentEfficiencyCategory)
	}
}
