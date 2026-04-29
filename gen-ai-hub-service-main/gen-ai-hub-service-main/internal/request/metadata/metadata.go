/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metadata

import (
	"context"
	"fmt"
	"time"

	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
)

// RequestMetadata contains enriched request information
type RequestMetadata struct {
	IsolationID    string                 // Isolation identifier for the request
	TargetModel    *modeltypes.Model      // Resolved target model information
	RequestMetrics metrics.RequestMetrics // Request processing metrics

	// Model Name tracking fields
	OriginalModelName string // Original model Name from the request (e.g., "gpt-4o-next")

	// Streaming request indicator
	IsStreaming bool // True if this is a streaming request (stream: true in request body)
}

// GetTimingMetrics implements the RequestMetadataInterface for metrics package
func (rm *RequestMetadata) GetTimingMetrics() *metrics.TimingMetrics {
	return &rm.RequestMetrics.TimingMetrics
}

// SetTimingMetrics implements the RequestMetadataInterface for metrics package
func (rm *RequestMetadata) SetTimingMetrics(startTime, endTime time.Time, duration time.Duration) {
	rm.RequestMetrics.TimingMetrics.StartTime = startTime
	rm.RequestMetrics.TimingMetrics.EndTime = endTime
	rm.RequestMetrics.TimingMetrics.Duration = duration
}

// GetTokenMetrics returns the token metrics for accessing requested tokens
func (rm *RequestMetadata) GetTokenMetrics() *metrics.TokenMetrics {
	return &rm.RequestMetrics.TokenMetrics
}

// GetIsolationID returns the isolation ID for metrics labeling
func (rm *RequestMetadata) GetIsolationID() string {
	return rm.IsolationID
}

// GetRequestMetrics returns the full RequestMetrics for metrics collection
func (rm *RequestMetadata) GetRequestMetrics() *metrics.RequestMetrics {
	return &rm.RequestMetrics
}

// GetOriginalModelName returns the original model name for metrics labeling
func (rm *RequestMetadata) GetOriginalModelName() string {
	return rm.OriginalModelName
}

// GetTargetModelName returns the target model name for metrics labeling
func (rm *RequestMetadata) GetTargetModelName() string {
	if rm.TargetModel != nil {
		return rm.TargetModel.Name
	}
	return ""
}

// GetTargetModelID returns the target model ID for metrics labeling
func (rm *RequestMetadata) GetTargetModelID() string {
	if rm.TargetModel != nil {
		return rm.TargetModel.KEY
	}
	return ""
}

// GetTargetModelVersion returns the target model version for metrics labeling
func (rm *RequestMetadata) GetTargetModelVersion() string {
	if rm.TargetModel != nil {
		return rm.TargetModel.Version
	}
	return ""
}

// GetTargetModelCreator returns the target model creator for metrics labeling
func (rm *RequestMetadata) GetTargetModelCreator() string {
	if rm.TargetModel != nil {
		return string(rm.TargetModel.Creator)
	}
	return ""
}

// GetTargetModelInfrastructure returns the target model infrastructure for metrics labeling
func (rm *RequestMetadata) GetTargetModelInfrastructure() string {
	if rm.TargetModel != nil {
		return string(rm.TargetModel.Infrastructure)
	}
	return ""
}

// GetTargetModel returns the target model for provider extraction
func (rm *RequestMetadata) GetTargetModel() *modeltypes.Model {
	return rm.TargetModel
}

// GetRequestMetadataFromContext retrieves RequestMetadata from the context
func GetRequestMetadataFromContext(ctx context.Context) (*RequestMetadata, error) {
	if metadata, ok := ctx.Value(metrics.RequestMetadataContextKey{}).(*RequestMetadata); ok {
		return metadata, nil
	}
	return nil, fmt.Errorf("RequestMetadata not found in context")
}
