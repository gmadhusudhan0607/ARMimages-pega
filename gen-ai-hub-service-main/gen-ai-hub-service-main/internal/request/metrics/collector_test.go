/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRequestMetadata implements RequestMetadataInterface for testing
type MockRequestMetadata struct {
	timingMetrics      *TimingMetrics
	targetModelVersion string
}

func (m *MockRequestMetadata) GetTimingMetrics() *TimingMetrics {
	return m.timingMetrics
}

func (m *MockRequestMetadata) SetTimingMetrics(startTime, endTime time.Time, duration time.Duration) {
	if m.timingMetrics == nil {
		m.timingMetrics = &TimingMetrics{}
	}
	m.timingMetrics.StartTime = startTime
	m.timingMetrics.EndTime = endTime
	m.timingMetrics.Duration = duration
}

func (m *MockRequestMetadata) GetTargetModelVersion() string {
	return m.targetModelVersion
}

func TestStartTiming(t *testing.T) {
	t.Run("with metadata in context", func(t *testing.T) {
		mockMetadata := &MockRequestMetadata{}
		ctx := context.WithValue(context.Background(), contextKey, mockMetadata)

		// Start timing
		updatedCtx := StartTiming(ctx)

		// Verify the timing was set
		updatedMeta := getRequestMetadataFromContext(updatedCtx)
		require.NotNil(t, updatedMeta)

		timingMetrics := updatedMeta.GetTimingMetrics()
		require.NotNil(t, timingMetrics)
		assert.False(t, timingMetrics.StartTime.IsZero())
		assert.True(t, timingMetrics.EndTime.IsZero())
		assert.Equal(t, time.Duration(0), timingMetrics.Duration)
	})

	t.Run("without metadata in context", func(t *testing.T) {
		ctx := context.Background()

		// Start timing - should return unchanged context
		updatedCtx := StartTiming(ctx)

		// Should be the same context
		assert.Equal(t, ctx, updatedCtx)

		// Should have no metadata
		metadata := getRequestMetadataFromContext(updatedCtx)
		assert.Nil(t, metadata)
	})
}

func TestEndTiming(t *testing.T) {
	t.Run("with metadata and existing start time", func(t *testing.T) {
		startTime := time.Now().Add(-100 * time.Millisecond)
		mockMetadata := &MockRequestMetadata{
			timingMetrics: &TimingMetrics{
				StartTime: startTime,
			},
		}
		ctx := context.WithValue(context.Background(), contextKey, mockMetadata)

		// End timing
		updatedCtx := EndTiming(ctx)

		// Verify the timing was completed
		updatedMeta := getRequestMetadataFromContext(updatedCtx)
		require.NotNil(t, updatedMeta)

		timingMetrics := updatedMeta.GetTimingMetrics()
		require.NotNil(t, timingMetrics)
		assert.Equal(t, startTime, timingMetrics.StartTime)
		assert.False(t, timingMetrics.EndTime.IsZero())
		assert.True(t, timingMetrics.Duration > 0)

		// Duration should be approximately equal to endTime - startTime
		expectedDuration := timingMetrics.EndTime.Sub(startTime)
		assert.Equal(t, expectedDuration, timingMetrics.Duration)
	})

	t.Run("with metadata but no start time", func(t *testing.T) {
		mockMetadata := &MockRequestMetadata{
			timingMetrics: &TimingMetrics{}, // No start time set
		}
		ctx := context.WithValue(context.Background(), contextKey, mockMetadata)

		// End timing - should not modify timing
		updatedCtx := EndTiming(ctx)

		// Should return same context since no start time was set
		assert.Equal(t, ctx, updatedCtx)

		updatedMeta := getRequestMetadataFromContext(updatedCtx)
		timingMetrics := updatedMeta.GetTimingMetrics()
		assert.True(t, timingMetrics.StartTime.IsZero())
		assert.True(t, timingMetrics.EndTime.IsZero())
		assert.Equal(t, time.Duration(0), timingMetrics.Duration)
	})

	t.Run("with metadata but nil timing metrics", func(t *testing.T) {
		mockMetadata := &MockRequestMetadata{
			timingMetrics: nil,
		}
		ctx := context.WithValue(context.Background(), contextKey, mockMetadata)

		// End timing - should not modify timing
		updatedCtx := EndTiming(ctx)

		// Should return same context
		assert.Equal(t, ctx, updatedCtx)
	})

	t.Run("without metadata in context", func(t *testing.T) {
		ctx := context.Background()

		// End timing - should return unchanged context
		updatedCtx := EndTiming(ctx)

		// Should be the same context
		assert.Equal(t, ctx, updatedCtx)
	})
}

func TestGetTimingMetrics(t *testing.T) {
	t.Run("with metadata and timing metrics", func(t *testing.T) {
		expectedTiming := &TimingMetrics{
			StartTime: time.Now().Add(-100 * time.Millisecond),
			EndTime:   time.Now(),
			Duration:  100 * time.Millisecond,
		}
		mockMetadata := &MockRequestMetadata{
			timingMetrics: expectedTiming,
		}
		ctx := context.WithValue(context.Background(), contextKey, mockMetadata)

		result := GetTimingMetrics(ctx)

		assert.Equal(t, expectedTiming, result)
	})

	t.Run("with metadata but nil timing metrics", func(t *testing.T) {
		mockMetadata := &MockRequestMetadata{
			timingMetrics: nil,
		}
		ctx := context.WithValue(context.Background(), contextKey, mockMetadata)

		result := GetTimingMetrics(ctx)

		assert.Nil(t, result)
	})

	t.Run("without metadata in context", func(t *testing.T) {
		ctx := context.Background()

		result := GetTimingMetrics(ctx)

		assert.Nil(t, result)
	})
}

func TestUpdateTimingInContext(t *testing.T) {
	t.Run("with metadata in context", func(t *testing.T) {
		mockMetadata := &MockRequestMetadata{}
		ctx := context.WithValue(context.Background(), contextKey, mockMetadata)

		startTime := time.Now().Add(-100 * time.Millisecond)
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Update timing
		updatedCtx := UpdateTimingInContext(ctx, startTime, endTime, duration)

		// Verify the timing was updated
		updatedMeta := getRequestMetadataFromContext(updatedCtx)
		require.NotNil(t, updatedMeta)

		timingMetrics := updatedMeta.GetTimingMetrics()
		require.NotNil(t, timingMetrics)
		assert.Equal(t, startTime, timingMetrics.StartTime)
		assert.Equal(t, endTime, timingMetrics.EndTime)
		assert.Equal(t, duration, timingMetrics.Duration)
	})

	t.Run("without metadata in context", func(t *testing.T) {
		ctx := context.Background()

		startTime := time.Now().Add(-100 * time.Millisecond)
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Update timing - should return unchanged context
		updatedCtx := UpdateTimingInContext(ctx, startTime, endTime, duration)

		// Should be the same context
		assert.Equal(t, ctx, updatedCtx)
	})
}

func TestGetRequestMetadataFromContext(t *testing.T) {
	t.Run("with valid metadata", func(t *testing.T) {
		mockMetadata := &MockRequestMetadata{
			targetModelVersion: "v1.0",
		}
		ctx := context.WithValue(context.Background(), contextKey, mockMetadata)

		result := getRequestMetadataFromContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "v1.0", result.GetTargetModelVersion())
	})

	t.Run("with invalid metadata type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), contextKey, "invalid-metadata")

		result := getRequestMetadataFromContext(ctx)

		assert.Nil(t, result)
	})

	t.Run("without metadata in context", func(t *testing.T) {
		ctx := context.Background()

		result := getRequestMetadataFromContext(ctx)

		assert.Nil(t, result)
	})
}

func TestSetRequestMetadataInContext(t *testing.T) {
	t.Run("set metadata in context", func(t *testing.T) {
		ctx := context.Background()
		mockMetadata := &MockRequestMetadata{
			targetModelVersion: "v2.0",
		}

		updatedCtx := setRequestMetadataInContext(ctx, mockMetadata)

		// Verify metadata was set
		result := getRequestMetadataFromContext(updatedCtx)
		require.NotNil(t, result)
		assert.Equal(t, "v2.0", result.GetTargetModelVersion())
	})

	t.Run("override existing metadata", func(t *testing.T) {
		oldMetadata := &MockRequestMetadata{
			targetModelVersion: "v1.0",
		}
		ctx := context.WithValue(context.Background(), contextKey, oldMetadata)

		newMetadata := &MockRequestMetadata{
			targetModelVersion: "v2.0",
		}

		updatedCtx := setRequestMetadataInContext(ctx, newMetadata)

		// Verify new metadata replaced old metadata
		result := getRequestMetadataFromContext(updatedCtx)
		require.NotNil(t, result)
		assert.Equal(t, "v2.0", result.GetTargetModelVersion())
	})
}

func TestRequestMetadataContextKey(t *testing.T) {
	t.Run("context key is consistent", func(t *testing.T) {
		// Verify the context key is a singleton and works correctly
		key1 := RequestMetadataContextKey{}
		key2 := RequestMetadataContextKey{}

		ctx1 := context.WithValue(context.Background(), key1, "value1")
		ctx2 := context.WithValue(context.Background(), key2, "value2")

		// Both keys should access the same value since they're the same type
		val1 := ctx1.Value(key2)
		val2 := ctx2.Value(key1)

		assert.Equal(t, "value1", val1)
		assert.Equal(t, "value2", val2)

		// Verify our contextKey variable works
		ctxWithContextKey := context.WithValue(context.Background(), contextKey, "contextKeyValue")
		result := ctxWithContextKey.Value(RequestMetadataContextKey{})
		assert.Equal(t, "contextKeyValue", result)
	})
}
