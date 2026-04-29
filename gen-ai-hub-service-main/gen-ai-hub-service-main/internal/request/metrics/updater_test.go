/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMetadataExtractor implements MetadataExtractor for testing
type MockMetadataExtractor struct {
	isolationID       string
	originalModelName string
	targetModel       *modeltypes.Model
}

func (m *MockMetadataExtractor) GetIsolationID() string       { return m.isolationID }
func (m *MockMetadataExtractor) GetOriginalModelName() string { return m.originalModelName }
func (m *MockMetadataExtractor) GetTargetModelName() string {
	if m.targetModel != nil {
		return m.targetModel.Name
	}
	return ""
}
func (m *MockMetadataExtractor) GetTargetModelID() string {
	if m.targetModel != nil {
		return m.targetModel.KEY
	}
	return ""
}
func (m *MockMetadataExtractor) GetTargetModelVersion() string {
	if m.targetModel != nil {
		return m.targetModel.Version
	}
	return ""
}
func (m *MockMetadataExtractor) GetTargetModelCreator() string {
	if m.targetModel != nil {
		return string(m.targetModel.Creator)
	}
	return ""
}
func (m *MockMetadataExtractor) GetTargetModelInfrastructure() string {
	if m.targetModel != nil {
		return string(m.targetModel.Infrastructure)
	}
	return ""
}
func (m *MockMetadataExtractor) GetTargetModel() *modeltypes.Model { return m.targetModel }

// createMockMetadata is a helper function for creating MockMetadataExtractor instances
func createMockMetadata(isolationID, originalModelName string, targetModel *modeltypes.Model) *MockMetadataExtractor {
	return &MockMetadataExtractor{
		isolationID:       isolationID,
		originalModelName: originalModelName,
		targetModel:       targetModel,
	}
}

func TestSafeStringValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "non-empty string returns itself",
			input:    "test-value",
			expected: "test-value",
		},
		{
			name:     "whitespace string returns itself",
			input:    "   ",
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeStringValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreatePrometheusLabels(t *testing.T) {
	labels := CreatePrometheusLabels(PrometheusLabelsInput{
		IsolationID:        "test-isolation",
		Path:               "/api/v1/chat",
		Method:             "POST",
		StatusCode:         "200",
		OriginalModelName:  "gpt-4",
		TargetModelName:    "gpt-4-turbo",
		TargetModelVersion: "v1.0",
		TargetModelID:      "model-123",
		Provider:           "openai",
		Infrastructure:     "cloud",
	})

	expected := map[string]string{
		"isolationID":        "test-isolation",
		"path":               "/api/v1/chat",
		"method":             "POST",
		"statusCode":         "200",
		"originalModelName":  "gpt-4",
		"targetModelName":    "gpt-4-turbo",
		"targetModelVersion": "v1.0",
		"targetModelID":      "model-123",
		"provider":           "openai",
		"infrastructure":     "cloud",
	}

	assert.Equal(t, expected, labels)
}

func TestCreatePrometheusLabelsInfrastructureHandling(t *testing.T) {
	// Test that "unknown" infrastructure is converted to empty string
	labels := CreatePrometheusLabels(PrometheusLabelsInput{
		IsolationID:        "test-isolation",
		Path:               "/api/v1/chat",
		Method:             "POST",
		StatusCode:         "200",
		OriginalModelName:  "gpt-4",
		TargetModelName:    "gpt-4-turbo",
		TargetModelVersion: "v1.0",
		TargetModelID:      "model-123",
		Provider:           "openai",
		Infrastructure:     "unknown",
	})

	assert.Equal(t, "", labels["infrastructure"])
}

func TestCreatePrometheusLabelsFromStatusCode(t *testing.T) {
	labels := CreatePrometheusLabelsFromStatusCode(PrometheusLabelsInputWithIntStatusCode{
		IsolationID:        "test-isolation",
		Path:               "/api/v1/chat",
		Method:             "POST",
		StatusCode:         200,
		OriginalModelName:  "gpt-4",
		TargetModelName:    "gpt-4-turbo",
		TargetModelVersion: "v1.0",
		TargetModelID:      "model-123",
		Provider:           "openai",
		Infrastructure:     "cloud",
	})

	assert.Equal(t, "200", labels["statusCode"])
}

func TestExtractMetadataFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("successful extraction", func(t *testing.T) {
		// Create a gin context
		c, _ := gin.CreateTestContext(nil)
		req := &http.Request{
			URL: &url.URL{Path: "/test"},
		}
		c.Request = req

		// Create mock target model
		targetModel := &modeltypes.Model{
			KEY:            "model-123",
			Name:           "gpt-4-turbo",
			Version:        "v1.0",
			Creator:        modeltypes.CreatorOpenAI,
			Infrastructure: modeltypes.InfrastructureAWS,
		}

		// Create mock metadata
		mockMetadata := createMockMetadata("test-isolation", "gpt-4", targetModel)

		// Add metadata to context
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, RequestMetadataContextKey{}, mockMetadata)
		c.Request = c.Request.WithContext(ctx)

		// Extract metadata
		extractor, ok := ExtractMetadataFromContext(c)
		require.True(t, ok)
		assert.Equal(t, "test-isolation", extractor.GetIsolationID())
		assert.Equal(t, "gpt-4", extractor.GetOriginalModelName())
		assert.Equal(t, "gpt-4-turbo", extractor.GetTargetModelName())
	})

	t.Run("no metadata in context", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		req := &http.Request{
			URL: &url.URL{Path: "/test"},
		}
		c.Request = req

		extractor, ok := ExtractMetadataFromContext(c)
		assert.False(t, ok)
		assert.Nil(t, extractor)
	})
}

func TestCreatePrometheusLabelsFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("with metadata", func(t *testing.T) {
		// Create a gin context
		c, _ := gin.CreateTestContext(nil)
		req := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: "/api/v1/chat"},
		}
		c.Request = req

		// Create mock target model
		targetModel := &modeltypes.Model{
			KEY:            "model-123",
			Name:           "gpt-4-turbo",
			Version:        "v1.0",
			Creator:        modeltypes.CreatorOpenAI,
			Infrastructure: modeltypes.InfrastructureAWS,
			Provider:       modeltypes.ProviderBedrock,
		}

		// Create mock metadata
		mockMetadata := createMockMetadata("test-isolation", "gpt-4", targetModel)

		// Add metadata to context
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, RequestMetadataContextKey{}, mockMetadata)
		c.Request = c.Request.WithContext(ctx)

		// Create labels
		labels := CreatePrometheusLabelsFromContext(c, 200)

		// Verify timing metric labels
		assert.Equal(t, "test-isolation", labels["isolationID"])
		assert.Equal(t, "200", labels["statusCode"])
		assert.Equal(t, "/api/v1/chat", labels["path"])
		assert.Equal(t, "POST", labels["method"])
		assert.Equal(t, "gpt-4", labels["originalModelName"])
		assert.Equal(t, "gpt-4-turbo", labels["targetModelName"])
		assert.Equal(t, "v1.0", labels["targetModelVersion"])
		assert.Equal(t, "model-123", labels["targetModelID"])
		assert.Equal(t, "bedrock", labels["provider"]) // Now populated from target model
		assert.Equal(t, "aws", labels["infrastructure"])

		// Verify token metric labels
		assert.Equal(t, "openai", labels["creator"])
		assert.Equal(t, "/api/v1/chat", labels["targetModelEndpoint"])
		assert.Equal(t, "v1.0", labels["modelVersion"])
	})

	t.Run("without metadata", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		req := &http.Request{
			Method: "GET",
			URL:    &url.URL{Path: "/health"},
		}
		c.Request = req

		labels := CreatePrometheusLabelsFromContext(c, 200)

		// Should have default/empty values
		assert.Equal(t, "", labels["isolationID"])
		assert.Equal(t, "200", labels["statusCode"])
		assert.Equal(t, "/health", labels["path"])
		assert.Equal(t, "GET", labels["method"])
		assert.Equal(t, "", labels["originalModelName"])
		assert.Equal(t, "", labels["targetModelName"])
		assert.Equal(t, "", labels["targetModelVersion"])
		assert.Equal(t, "", labels["targetModelID"])
		assert.Equal(t, "", labels["provider"])
		assert.Equal(t, "", labels["infrastructure"])
		assert.Equal(t, "", labels["creator"])
	})

	t.Run("infrastructure unknown handling", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		req := &http.Request{
			Method: "POST",
			URL:    &url.URL{Path: "/api/v1/chat"},
		}
		c.Request = req

		// Create mock target model with "unknown" infrastructure (using empty string as there's no "unknown" constant)
		targetModel := &modeltypes.Model{
			Infrastructure: "", // Empty string represents unknown infrastructure
		}

		// Create mock metadata with "unknown" infrastructure
		mockMetadata := createMockMetadata("", "", targetModel)

		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, RequestMetadataContextKey{}, mockMetadata)
		c.Request = c.Request.WithContext(ctx)

		labels := CreatePrometheusLabelsFromContext(c, 200)

		// Should convert "unknown" to empty string
		assert.Equal(t, "", labels["infrastructure"])
	})
}

func TestValidateLabelsForTimingMetrics(t *testing.T) {
	t.Run("valid labels", func(t *testing.T) {
		labels := map[string]string{
			"isolationID":        "test",
			"infrastructure":     "cloud",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
			"statusCode":         "200",
			"path":               "/api/v1/chat",
			"method":             "POST",
		}

		err := ValidateLabelsForTimingMetrics(labels)
		assert.NoError(t, err)
	})

	t.Run("missing required label", func(t *testing.T) {
		labels := map[string]string{
			"isolationID": "test",
			// Missing other required labels
		}

		err := ValidateLabelsForTimingMetrics(labels)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required label for timing metrics")
	})
}

func TestValidateLabelsForTokenMetrics(t *testing.T) {
	t.Run("valid labels", func(t *testing.T) {
		labels := map[string]string{
			"isolationID":         "test",
			"infrastructure":      "cloud",
			"provider":            "openai",
			"creator":             "openai",
			"targetModelID":       "model-123",
			"targetModelEndpoint": "/api/v1/chat",
		}

		err := ValidateLabelsForTokenMetrics(labels)
		assert.NoError(t, err)
	})

	t.Run("missing required label", func(t *testing.T) {
		labels := map[string]string{
			"isolationID": "test",
			// Missing other required labels
		}

		err := ValidateLabelsForTokenMetrics(labels)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required label for token metrics")
	})
}

func TestUpdateTokenMetrics(t *testing.T) {
	// Register metrics to avoid panic
	RegisterMetrics()

	t.Run("updates all token metrics", func(t *testing.T) {
		// Create test data
		requested := 1000.0
		maximum := 4000.0
		used := 750.0

		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				Requested: &requested,
				Maximum:   &maximum,
				Used:      &used,
			},
		}

		labels := map[string]string{
			"isolationID":         "test-isolation",
			"infrastructure":      "aws",
			"provider":            "openai",
			"creator":             "openai",
			"originalModelName":   "gpt-4",
			"targetModelName":     "gpt-4-turbo",
			"targetModelVersion":  "v1.0",
			"targetModelID":       "model-123",
			"targetModelEndpoint": "/api/v1/chat",
			"modelVersion":        "v1.0",
		}

		// This should not panic and should update all metrics
		UpdateTokenMetrics(requestMetrics, labels)
	})

	t.Run("handles nil request metrics", func(t *testing.T) {
		labels := map[string]string{
			"isolationID": "test",
		}

		// Should not panic
		UpdateTokenMetrics(nil, labels)
	})

	t.Run("handles nil token values", func(t *testing.T) {
		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				// All fields are nil
			},
		}

		labels := map[string]string{
			"isolationID":         "test-isolation",
			"infrastructure":      "aws",
			"provider":            "openai",
			"creator":             "openai",
			"originalModelName":   "gpt-4",
			"targetModelName":     "gpt-4-turbo",
			"targetModelVersion":  "v1.0",
			"targetModelID":       "model-123",
			"targetModelEndpoint": "/api/v1/chat",
			"modelVersion":        "v1.0",
		}

		// Should not panic
		UpdateTokenMetrics(requestMetrics, labels)
	})

	t.Run("handles zero token values", func(t *testing.T) {
		requested := 0.0
		maximum := 0.0
		used := 0.0

		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				Requested: &requested,
				Maximum:   &maximum,
				Used:      &used,
			},
		}

		labels := map[string]string{
			"isolationID":         "test-isolation",
			"infrastructure":      "aws",
			"provider":            "openai",
			"creator":             "openai",
			"originalModelName":   "gpt-4",
			"targetModelName":     "gpt-4-turbo",
			"targetModelVersion":  "v1.0",
			"targetModelID":       "model-123",
			"targetModelEndpoint": "/api/v1/chat",
			"modelVersion":        "v1.0",
		}

		// Should not panic, but should not update metrics (due to > 0 check)
		UpdateTokenMetrics(requestMetrics, labels)
	})

	t.Run("updates only used tokens when others are nil", func(t *testing.T) {
		used := 500.0

		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				Requested: nil,
				Maximum:   nil,
				Used:      &used,
			},
		}

		labels := map[string]string{
			"isolationID":         "test-isolation",
			"infrastructure":      "aws",
			"provider":            "openai",
			"creator":             "openai",
			"originalModelName":   "gpt-4",
			"targetModelName":     "gpt-4-turbo",
			"targetModelVersion":  "v1.0",
			"targetModelID":       "model-123",
			"targetModelEndpoint": "/api/v1/chat",
			"modelVersion":        "v1.0",
		}

		// Should not panic and should update only used tokens metric
		UpdateTokenMetrics(requestMetrics, labels)
	})
}

func TestUpdateTimingMetrics(t *testing.T) {
	// Register metrics to avoid panic
	RegisterMetrics()

	t.Run("updates timing metrics", func(t *testing.T) {
		requestMetrics := &RequestMetrics{
			TimingMetrics: TimingMetrics{
				Duration: 150 * time.Millisecond,
			},
		}

		labels := map[string]string{
			"isolationID":        "test-isolation",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
			"statusCode":         "200",
			"path":               "/api/v1/chat",
			"method":             "POST",
		}

		// Should not panic and should update timing metrics
		assert.NotPanics(t, func() {
			UpdateTimingMetrics(requestMetrics, labels)
		})
	})

	t.Run("handles nil request metrics", func(t *testing.T) {
		labels := map[string]string{
			"isolationID": "test",
		}

		// Should not panic
		assert.NotPanics(t, func() {
			UpdateTimingMetrics(nil, labels)
		})
	})

	t.Run("handles zero duration", func(t *testing.T) {
		requestMetrics := &RequestMetrics{
			TimingMetrics: TimingMetrics{
				Duration: 0,
			},
		}

		labels := map[string]string{
			"isolationID":        "test-isolation",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
			"statusCode":         "200",
			"path":               "/api/v1/chat",
			"method":             "POST",
		}

		// Should not panic even with zero duration
		assert.NotPanics(t, func() {
			UpdateTimingMetrics(requestMetrics, labels)
		})
	})
}

func TestUpdateAdjustedCurrentMetric(t *testing.T) {
	// Register metrics to avoid panic
	RegisterMetrics()

	t.Run("updates adjusted current metric", func(t *testing.T) {
		adjustedValue := 1200.0
		labels := map[string]string{
			"isolationID":        "test-isolation",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		// Should not panic and should update the metric
		assert.NotPanics(t, func() {
			UpdateAdjustedCurrentMetric(adjustedValue, labels)
		})
	})

	t.Run("handles zero value", func(t *testing.T) {
		labels := map[string]string{
			"isolationID":        "test-isolation",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		// Should not panic even with zero value
		assert.NotPanics(t, func() {
			UpdateAdjustedCurrentMetric(0, labels)
		})
	})

	t.Run("handles negative value", func(t *testing.T) {
		labels := map[string]string{
			"isolationID":        "test-isolation",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		// Should not panic even with negative value
		assert.NotPanics(t, func() {
			UpdateAdjustedCurrentMetric(-100, labels)
		})
	})
}

func TestUpdateTokenMetrics_ReasoningTokens(t *testing.T) {
	// Register metrics to avoid panic
	RegisterMetrics()

	standardLabels := map[string]string{
		"isolationID":         "test-isolation",
		"infrastructure":      "aws",
		"provider":            "openai",
		"creator":             "openai",
		"originalModelName":   "o3",
		"targetModelName":     "o3",
		"targetModelVersion":  "v1.0",
		"targetModelID":       "model-o3",
		"targetModelEndpoint": "/api/v1/chat",
		"modelVersion":        "v1.0",
	}

	t.Run("reasoning tokens > 0 updates histogram", func(t *testing.T) {
		reasoningTokens := 512.0
		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				ReasoningTokens: &reasoningTokens,
			},
		}

		// Should not panic and should update reasoning tokens metric
		assert.NotPanics(t, func() {
			UpdateTokenMetrics(requestMetrics, standardLabels)
		})
	})

	t.Run("reasoning tokens = 0 does not update histogram", func(t *testing.T) {
		reasoningTokens := 0.0
		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				ReasoningTokens: &reasoningTokens,
			},
		}

		// Should not panic; histogram should NOT be updated (due to > 0 check)
		assert.NotPanics(t, func() {
			UpdateTokenMetrics(requestMetrics, standardLabels)
		})
	})

	t.Run("reasoning tokens nil does not update histogram", func(t *testing.T) {
		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				ReasoningTokens: nil,
			},
		}

		// Should not panic; histogram should NOT be updated
		assert.NotPanics(t, func() {
			UpdateTokenMetrics(requestMetrics, standardLabels)
		})
	})

	t.Run("reasoning tokens updated for both streaming and non-streaming", func(t *testing.T) {
		reasoningTokens := 256.0

		// Non-streaming
		nonStreamingMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				ReasoningTokens: &reasoningTokens,
			},
			IsStreaming: false,
		}
		assert.NotPanics(t, func() {
			UpdateTokenMetrics(nonStreamingMetrics, standardLabels)
		})

		// Streaming - reasoning tokens should ALSO be updated for streaming
		streamingMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				ReasoningTokens: &reasoningTokens,
			},
			IsStreaming: true,
		}
		assert.NotPanics(t, func() {
			UpdateTokenMetrics(streamingMetrics, standardLabels)
		})
	})

	t.Run("reasoning tokens with large value (deep thinking)", func(t *testing.T) {
		reasoningTokens := 16384.0
		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				ReasoningTokens: &reasoningTokens,
			},
		}

		assert.NotPanics(t, func() {
			UpdateTokenMetrics(requestMetrics, standardLabels)
		})
	})

	t.Run("reasoning tokens alongside other token metrics", func(t *testing.T) {
		requested := 1000.0
		used := 750.0
		reasoningTokens := 512.0

		requestMetrics := &RequestMetrics{
			TokenMetrics: TokenMetrics{
				Requested:       &requested,
				Used:            &used,
				ReasoningTokens: &reasoningTokens,
			},
		}

		// All metrics including reasoning should be updated without panic
		assert.NotPanics(t, func() {
			UpdateTokenMetrics(requestMetrics, standardLabels)
		})
	})
}

func TestIncrementModelRecognition(t *testing.T) {
	// Register metrics to avoid panic
	RegisterMetrics()

	t.Run("increments model recognition for recognized status", func(t *testing.T) {
		isolationID := "test-isolation"
		status := "recognized"
		originalModelName := "gpt-4"

		// Should not panic and should increment the counter
		assert.NotPanics(t, func() {
			IncrementModelRecognition(isolationID, status, originalModelName)
		})
	})

	t.Run("increments model recognition for unrecognized status", func(t *testing.T) {
		isolationID := "test-isolation"
		status := "unrecognized"
		originalModelName := "unknown-model"

		// Should not panic and should increment the counter
		assert.NotPanics(t, func() {
			IncrementModelRecognition(isolationID, status, originalModelName)
		})
	})

	t.Run("handles empty values", func(t *testing.T) {
		// Should not panic even with empty values
		assert.NotPanics(t, func() {
			IncrementModelRecognition("", "", "")
		})
	})

	t.Run("handles various status values", func(t *testing.T) {
		testStatuses := []string{"recognized", "unrecognized", "error", "timeout"}

		for _, status := range testStatuses {
			assert.NotPanics(t, func() {
				IncrementModelRecognition("test-id", status, "test-model")
			})
		}
	})
}
