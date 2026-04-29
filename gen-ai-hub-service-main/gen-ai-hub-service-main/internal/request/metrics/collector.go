/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"context"
	"time"
)

// RequestMetadataContextKey is the key used to store RequestMetadata in the context
type RequestMetadataContextKey struct{}

// contextKey is the singleton instance of RequestMetadataContextKey
var contextKey = RequestMetadataContextKey{}

// RequestMetadataInterface defines the interface for RequestMetadata to avoid import cycles
type RequestMetadataInterface interface {
	GetTimingMetrics() *TimingMetrics
	SetTimingMetrics(startTime, endTime time.Time, duration time.Duration)
	GetTargetModelVersion() string
}

// StartTiming marks the start time for request processing
func StartTiming(ctx context.Context) context.Context {
	if metadata := getRequestMetadataFromContext(ctx); metadata != nil {
		metadata.SetTimingMetrics(time.Now(), time.Time{}, 0)
		return setRequestMetadataInContext(ctx, metadata)
	}
	return ctx
}

// EndTiming marks the end time and calculates duration for request processing
func EndTiming(ctx context.Context) context.Context {
	if metadata := getRequestMetadataFromContext(ctx); metadata != nil {
		timingMetrics := metadata.GetTimingMetrics()
		if timingMetrics != nil && !timingMetrics.StartTime.IsZero() {
			endTime := time.Now()
			duration := endTime.Sub(timingMetrics.StartTime)
			metadata.SetTimingMetrics(timingMetrics.StartTime, endTime, duration)
			return setRequestMetadataInContext(ctx, metadata)
		}
	}
	return ctx
}

// GetTimingMetrics retrieves timing metrics from context
func GetTimingMetrics(ctx context.Context) *TimingMetrics {
	if metadata := getRequestMetadataFromContext(ctx); metadata != nil {
		return metadata.GetTimingMetrics()
	}
	return nil
}

// UpdateTimingInContext updates timing metrics in the context
func UpdateTimingInContext(ctx context.Context, startTime, endTime time.Time, duration time.Duration) context.Context {
	if metadata := getRequestMetadataFromContext(ctx); metadata != nil {
		metadata.SetTimingMetrics(startTime, endTime, duration)
		return setRequestMetadataInContext(ctx, metadata)
	}
	return ctx
}

// Helper functions that will be implemented by the middleware package
func getRequestMetadataFromContext(ctx context.Context) RequestMetadataInterface {
	if value := ctx.Value(contextKey); value != nil {
		if metadata, ok := value.(RequestMetadataInterface); ok {
			return metadata
		}
	}
	return nil
}

func setRequestMetadataInContext(ctx context.Context, metadata RequestMetadataInterface) context.Context {
	return context.WithValue(ctx, contextKey, metadata)
}
