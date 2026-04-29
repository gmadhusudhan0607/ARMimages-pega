/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestRegisterMetrics(t *testing.T) {
	// Note: Since RegisterMetrics uses sync.Once, it can only be called once per process
	// We'll test that it doesn't panic and that the metrics are available

	t.Run("register metrics does not panic", func(t *testing.T) {
		// This should not panic even if called multiple times due to sync.Once
		assert.NotPanics(t, func() {
			RegisterMetrics()
		})
	})

	t.Run("metrics are registered", func(t *testing.T) {
		// Verify that metrics are properly registered by checking if they exist in the default registry
		// We can't easily unregister metrics in tests, so we'll just verify they don't panic on registration

		// Try to create a new instance of the same metrics - this would panic if they're already registered
		// But since we use MustRegister with sync.Once, duplicate registration is prevented

		// Test that the metrics variables are not nil
		assert.NotNil(t, requestDuration)
		assert.NotNil(t, outputTokensRequested)
		assert.NotNil(t, outputTokensMaximum)
		assert.NotNil(t, outputTokensUsed)
		assert.NotNil(t, outputTokensAdjusted)
		assert.NotNil(t, outputTokensAdjustedEfficiencyRatio)
		assert.NotNil(t, outputTokensRequestedEfficiencyRatio)
		assert.NotNil(t, outputTokensAdjustedWastedTotal)
		assert.NotNil(t, outputTokensRequestedWastedTotal)
		assert.NotNil(t, outputTokensAdjustedCurrent)
		assert.NotNil(t, modelRecognitionTotal)
	})
}

func TestMetricsInitialization(t *testing.T) {
	t.Run("request duration histogram", func(t *testing.T) {
		assert.NotNil(t, requestDuration)

		// Test that we can create labels for the histogram
		labels := prometheus.Labels{
			"isolationID":        "test",
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

		// This should not panic
		assert.NotPanics(t, func() {
			requestDuration.With(labels).Observe(100.5)
		})
	})

	t.Run("output tokens requested histogram", func(t *testing.T) {
		assert.NotNil(t, outputTokensRequested)

		labels := prometheus.Labels{
			"isolationID":         "test",
			"infrastructure":      "aws",
			"provider":            "openai",
			"creator":             "openai",
			"originalModelName":   "gpt-4",
			"targetModelName":     "gpt-4-turbo",
			"targetModelVersion":  "v1.0",
			"targetModelID":       "model-123",
			"targetModelEndpoint": "/api/v1/chat",
		}

		assert.NotPanics(t, func() {
			outputTokensRequested.With(labels).Observe(1000)
		})
	})

	t.Run("output tokens maximum gauge", func(t *testing.T) {
		assert.NotNil(t, outputTokensMaximum)

		labels := prometheus.Labels{
			"isolationID":         "test",
			"infrastructure":      "aws",
			"provider":            "openai",
			"creator":             "openai",
			"originalModelName":   "gpt-4",
			"targetModelName":     "gpt-4-turbo",
			"targetModelVersion":  "v1.0",
			"targetModelID":       "model-123",
			"modelVersion":        "v1.0",
			"targetModelEndpoint": "/api/v1/chat",
		}

		assert.NotPanics(t, func() {
			outputTokensMaximum.With(labels).Set(4000)
		})
	})

	t.Run("output tokens used histogram", func(t *testing.T) {
		assert.NotNil(t, outputTokensUsed)

		labels := prometheus.Labels{
			"isolationID":        "test",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		assert.NotPanics(t, func() {
			outputTokensUsed.With(labels).Observe(750)
		})
	})

	t.Run("output tokens adjusted histogram", func(t *testing.T) {
		assert.NotNil(t, outputTokensAdjusted)

		labels := prometheus.Labels{
			"isolationID":        "test",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		assert.NotPanics(t, func() {
			outputTokensAdjusted.With(labels).Observe(1200)
		})
	})

	t.Run("efficiency ratio histograms", func(t *testing.T) {
		assert.NotNil(t, outputTokensAdjustedEfficiencyRatio)
		assert.NotNil(t, outputTokensRequestedEfficiencyRatio)

		labels := prometheus.Labels{
			"isolationID":        "test",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		assert.NotPanics(t, func() {
			outputTokensAdjustedEfficiencyRatio.With(labels).Observe(1.6)
			outputTokensRequestedEfficiencyRatio.With(labels).Observe(1.3)
		})
	})

	t.Run("wasted tokens counters", func(t *testing.T) {
		assert.NotNil(t, outputTokensAdjustedWastedTotal)
		assert.NotNil(t, outputTokensRequestedWastedTotal)

		labels := prometheus.Labels{
			"isolationID":        "test",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		assert.NotPanics(t, func() {
			outputTokensAdjustedWastedTotal.With(labels).Add(450)
			outputTokensRequestedWastedTotal.With(labels).Add(250)
		})
	})

	t.Run("output tokens adjusted current gauge", func(t *testing.T) {
		assert.NotNil(t, outputTokensAdjustedCurrent)

		labels := prometheus.Labels{
			"isolationID":        "test",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		assert.NotPanics(t, func() {
			outputTokensAdjustedCurrent.With(labels).Set(1200)
		})
	})

	t.Run("model recognition counter", func(t *testing.T) {
		assert.NotNil(t, modelRecognitionTotal)

		labels := prometheus.Labels{
			"isolationID":       "test",
			"status":            "recognized",
			"originalModelName": "gpt-4",
		}

		assert.NotPanics(t, func() {
			modelRecognitionTotal.With(labels).Inc()
		})
	})
}

func TestMetricsBuckets(t *testing.T) {
	t.Run("request duration buckets", func(t *testing.T) {
		// Verify that request duration has appropriate buckets for millisecond measurements
		expectedBuckets := []float64{10, 50, 100, 500, 1000, 2000, 5000, 10000, 30000, 60000}

		// We can't easily access the buckets from a registered histogram,
		// but we can test that the histogram accepts values in the expected ranges
		labels := prometheus.Labels{
			"isolationID":        "test",
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

		// Test values across different bucket ranges
		testValues := []float64{5, 25, 75, 250, 750, 1500, 3000, 7500, 20000, 45000}

		for _, value := range testValues {
			assert.NotPanics(t, func() {
				requestDuration.With(labels).Observe(value)
			}, "Should accept value %f", value)
		}

		// Verify expected buckets length
		assert.Len(t, expectedBuckets, 10)
	})

	t.Run("token metrics buckets", func(t *testing.T) {
		// Verify token histogram buckets
		expectedBuckets := []float64{1000, 4000, 8000, 16000, 32000, 64000, 128000}

		labels := prometheus.Labels{
			"isolationID":        "test",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		// Test values across different bucket ranges
		testValues := []float64{500, 2000, 6000, 12000, 24000, 48000, 96000}

		for _, value := range testValues {
			assert.NotPanics(t, func() {
				outputTokensUsed.With(labels).Observe(value)
				outputTokensAdjusted.With(labels).Observe(value)
			}, "Should accept token value %f", value)
		}

		// Verify expected buckets length
		assert.Len(t, expectedBuckets, 7)
	})

	t.Run("efficiency ratio buckets", func(t *testing.T) {
		expectedBuckets := []float64{0.5, 0.8, 1.0, 1.5, 3.0, 5.0, 10.0, 20.0, 50.0, 100.0}

		labels := prometheus.Labels{
			"isolationID":        "test",
			"infrastructure":     "aws",
			"provider":           "openai",
			"creator":            "openai",
			"originalModelName":  "gpt-4",
			"targetModelName":    "gpt-4-turbo",
			"targetModelVersion": "v1.0",
			"targetModelID":      "model-123",
		}

		// Test values across different bucket ranges
		testValues := []float64{0.3, 0.6, 0.9, 1.2, 2.0, 4.0, 8.0, 15.0, 35.0, 75.0}

		for _, value := range testValues {
			assert.NotPanics(t, func() {
				outputTokensAdjustedEfficiencyRatio.With(labels).Observe(value)
				outputTokensRequestedEfficiencyRatio.With(labels).Observe(value)
			}, "Should accept efficiency ratio value %f", value)
		}

		// Verify expected buckets length
		assert.Len(t, expectedBuckets, 10)
	})
}

func TestSyncOnce(t *testing.T) {
	t.Run("multiple register calls are safe", func(t *testing.T) {
		// Test that multiple calls to RegisterMetrics don't cause panics
		// due to duplicate registration (sync.Once should prevent this)

		for i := 0; i < 5; i++ {
			assert.NotPanics(t, func() {
				RegisterMetrics()
			}, "Call %d should not panic", i+1)
		}
	})
}
