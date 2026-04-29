/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metadata

import (
	"context"
	"testing"
	"time"

	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
)

func TestRequestMetadata_GetTimingMetrics(t *testing.T) {
	rm := &RequestMetadata{
		RequestMetrics: metrics.RequestMetrics{
			TimingMetrics: metrics.TimingMetrics{
				Duration: time.Second,
			},
		},
	}

	timingMetrics := rm.GetTimingMetrics()

	if timingMetrics == nil {
		t.Fatal("Expected timing metrics, got nil")
	} else if timingMetrics.Duration != time.Second {
		t.Errorf("Expected duration %v, got %v", time.Second, timingMetrics.Duration)
	}
}

func TestRequestMetadata_SetTimingMetrics(t *testing.T) {
	rm := &RequestMetadata{}
	startTime := time.Now()
	endTime := startTime.Add(time.Second)
	duration := time.Second

	rm.SetTimingMetrics(startTime, endTime, duration)

	timingMetrics := rm.GetTimingMetrics()
	if !timingMetrics.StartTime.Equal(startTime) {
		t.Errorf("Expected start time %v, got %v", startTime, timingMetrics.StartTime)
	}
	if !timingMetrics.EndTime.Equal(endTime) {
		t.Errorf("Expected end time %v, got %v", endTime, timingMetrics.EndTime)
	}
	if timingMetrics.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, timingMetrics.Duration)
	}
}

func TestRequestMetadata_GetTokenMetrics(t *testing.T) {
	expectedTokens := 100.0
	rm := &RequestMetadata{
		RequestMetrics: metrics.RequestMetrics{
			TokenMetrics: metrics.TokenMetrics{
				Requested: &expectedTokens,
			},
		},
	}

	tokenMetrics := rm.GetTokenMetrics()

	if tokenMetrics == nil {
		t.Fatal("Expected token metrics, got nil")
	} else if tokenMetrics.Requested == nil || *tokenMetrics.Requested != expectedTokens {
		t.Errorf("Expected requested tokens %f, got %v", expectedTokens, tokenMetrics.Requested)
	}
}

func TestRequestMetadata_GetIsolationID(t *testing.T) {
	expectedID := "test-isolation-id"
	rm := &RequestMetadata{
		IsolationID: expectedID,
	}

	isolationID := rm.GetIsolationID()

	if isolationID != expectedID {
		t.Errorf("Expected isolation ID %s, got %s", expectedID, isolationID)
	}
}

func TestRequestMetadata_GetRequestMetrics(t *testing.T) {
	expectedDuration := time.Minute
	rm := &RequestMetadata{
		RequestMetrics: metrics.RequestMetrics{
			TimingMetrics: metrics.TimingMetrics{
				Duration: expectedDuration,
			},
		},
	}

	requestMetrics := rm.GetRequestMetrics()

	if requestMetrics == nil {
		t.Fatal("Expected request metrics, got nil")
	} else if requestMetrics.TimingMetrics.Duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, requestMetrics.TimingMetrics.Duration)
	}
}

func TestRequestMetadata_GetOriginalModelName(t *testing.T) {
	expectedName := "gpt-4o"
	rm := &RequestMetadata{
		OriginalModelName: expectedName,
	}

	modelName := rm.GetOriginalModelName()

	if modelName != expectedName {
		t.Errorf("Expected original model name %s, got %s", expectedName, modelName)
	}
}

func TestRequestMetadata_GetTargetModelName(t *testing.T) {
	tests := []struct {
		name        string
		targetModel *modeltypes.Model
		expected    string
	}{
		{
			name:        "nil target model",
			targetModel: nil,
			expected:    "",
		},
		{
			name: "with target model",
			targetModel: &modeltypes.Model{
				Name: "gpt-4",
			},
			expected: "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &RequestMetadata{
				TargetModel: tt.targetModel,
			}

			result := rm.GetTargetModelName()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRequestMetadata_GetTargetModelID(t *testing.T) {
	tests := []struct {
		name        string
		targetModel *modeltypes.Model
		expected    string
	}{
		{
			name:        "nil target model",
			targetModel: nil,
			expected:    "",
		},
		{
			name: "with target model",
			targetModel: &modeltypes.Model{
				KEY: "openai-gpt-4",
			},
			expected: "openai-gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &RequestMetadata{
				TargetModel: tt.targetModel,
			}

			result := rm.GetTargetModelID()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRequestMetadata_GetTargetModelVersion(t *testing.T) {
	tests := []struct {
		name        string
		targetModel *modeltypes.Model
		expected    string
	}{
		{
			name:        "nil target model",
			targetModel: nil,
			expected:    "",
		},
		{
			name: "with target model",
			targetModel: &modeltypes.Model{
				Version: "2024-02-01",
			},
			expected: "2024-02-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &RequestMetadata{
				TargetModel: tt.targetModel,
			}

			result := rm.GetTargetModelVersion()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRequestMetadata_GetTargetModelCreator(t *testing.T) {
	tests := []struct {
		name        string
		targetModel *modeltypes.Model
		expected    string
	}{
		{
			name:        "nil target model",
			targetModel: nil,
			expected:    "",
		},
		{
			name: "with target model",
			targetModel: &modeltypes.Model{
				Creator: "openai",
			},
			expected: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &RequestMetadata{
				TargetModel: tt.targetModel,
			}

			result := rm.GetTargetModelCreator()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRequestMetadata_GetTargetModelInfrastructure(t *testing.T) {
	tests := []struct {
		name        string
		targetModel *modeltypes.Model
		expected    string
	}{
		{
			name:        "nil target model",
			targetModel: nil,
			expected:    "",
		},
		{
			name: "with target model",
			targetModel: &modeltypes.Model{
				Infrastructure: "azure",
			},
			expected: "azure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &RequestMetadata{
				TargetModel: tt.targetModel,
			}

			result := rm.GetTargetModelInfrastructure()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRequestMetadata_GetTargetModel(t *testing.T) {
	expectedModel := &modeltypes.Model{
		Name: "gpt-4",
		KEY:  "openai-gpt-4",
	}
	rm := &RequestMetadata{
		TargetModel: expectedModel,
	}

	result := rm.GetTargetModel()

	if result != expectedModel {
		t.Errorf("Expected model %v, got %v", expectedModel, result)
	}
}

func TestGetRequestMetadataFromContext(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		expectError bool
		expected    *RequestMetadata
	}{
		{
			name:        "context without metadata",
			ctx:         context.Background(),
			expectError: true,
			expected:    nil,
		},
		{
			name: "context with metadata",
			ctx: context.WithValue(context.Background(), metrics.RequestMetadataContextKey{}, &RequestMetadata{
				IsolationID:       "test-id",
				OriginalModelName: "gpt-4",
			}),
			expectError: false,
			expected: &RequestMetadata{
				IsolationID:       "test-id",
				OriginalModelName: "gpt-4",
			},
		},
		{
			name:        "context with wrong type value",
			ctx:         context.WithValue(context.Background(), metrics.RequestMetadataContextKey{}, "not-metadata"),
			expectError: true,
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetRequestMetadataFromContext(tt.ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if result == nil {
					t.Fatal("Expected metadata, got nil")
				} else {
					if result.IsolationID != tt.expected.IsolationID {
						t.Errorf("Expected isolation ID %s, got %s", tt.expected.IsolationID, result.IsolationID)
					}
					if result.OriginalModelName != tt.expected.OriginalModelName {
						t.Errorf("Expected original model name %s, got %s", tt.expected.OriginalModelName, result.OriginalModelName)
					}
				}
			}
		})
	}
}
