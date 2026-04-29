/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Timing Metrics
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genai_request_duration_ms",
			Help:    "Request processing duration in milliseconds",
			Buckets: []float64{10, 50, 100, 500, 1000, 2000, 5000, 10000, 30000, 60000},
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID", "statusCode", "path", "method"},
	)

	// Token Metrics
	outputTokensRequested = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genai_gateway_output_tokens_requested",
			Help:    "Original max_tokens values in requests before modification",
			Buckets: []float64{1000, 4000, 8000, 16000, 32000, 64000, 128000},
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID", "targetModelEndpoint"},
	)

	outputTokensMaximum = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "genai_gateway_output_tokens_maximum",
			Help: "Maximum output_tokens values for provider/model (static configuration values)",
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID", "modelVersion", "targetModelEndpoint"},
	)

	outputTokensUsed = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genai_gateway_output_tokens_used",
			Help:    "Number of tokens actually used by model",
			Buckets: []float64{1000, 4000, 8000, 16000, 32000, 64000, 128000},
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID"},
	)

	outputTokensAdjusted = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genai_gateway_output_tokens_adjusted",
			Help:    "Adjusted max_tokens values after service modification",
			Buckets: []float64{1000, 4000, 8000, 16000, 32000, 64000, 128000},
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID"},
	)

	outputTokensAdjustedEfficiencyRatio = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genai_gateway_output_tokens_adjusted_efficiency_ratio",
			Help:    "Ratio of adjusted_tokens / used_tokens for completed requests",
			Buckets: []float64{0.5, 0.8, 1.0, 1.5, 3.0, 5.0, 10.0, 20.0, 50.0, 100.0},
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID"},
	)

	outputTokensRequestedEfficiencyRatio = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genai_gateway_output_tokens_requested_efficiency_ratio",
			Help:    "Ratio of requested_tokens / used_tokens for completed requests",
			Buckets: []float64{0.5, 0.8, 1.0, 1.5, 3.0, 5.0, 10.0, 20.0, 50.0, 100.0},
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID"},
	)

	outputTokensAdjustedWastedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genai_gateway_output_tokens_adjusted_wasted_total",
			Help: "Total tokens over-allocated (adjusted - used) when adjustment was inefficient",
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID"},
	)

	outputTokensRequestedWastedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genai_gateway_output_tokens_requested_wasted_total",
			Help: "Total tokens over-allocated ((original or default) - used) when adjustment was inefficient",
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID"},
	)

	outputTokensAdjustedCurrent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "genai_gateway_output_tokens_adjusted_current",
			Help: "Current max_tokens value used for injection/adjustment in requests (for all strategies)",
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID"},
	)

	// Reasoning Token Metrics
	reasoningTokensUsed = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "genai_gateway_reasoning_tokens_used",
			Help:    "Number of reasoning tokens used by reasoning models",
			Buckets: []float64{100, 500, 1000, 4000, 8000, 16000, 32000, 64000, 128000},
		},
		[]string{"isolationID", "infrastructure", "provider", "creator", "originalModelName", "targetModelName", "targetModelVersion", "targetModelID"},
	)

	// Model Recognition Metrics
	modelRecognitionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "genai_gateway_model_recognition_total",
			Help: "Total number of requests with recognized/unrecognized models",
		},
		[]string{"isolationID", "status", "originalModelName"},
	)

	once sync.Once
)

// RegisterMetrics registers all Prometheus metrics
func RegisterMetrics() {
	once.Do(func() {
		prometheus.MustRegister(requestDuration)
		prometheus.MustRegister(outputTokensRequested)
		prometheus.MustRegister(outputTokensMaximum)
		prometheus.MustRegister(outputTokensUsed)
		prometheus.MustRegister(outputTokensAdjusted)
		prometheus.MustRegister(outputTokensAdjustedEfficiencyRatio)
		prometheus.MustRegister(outputTokensRequestedEfficiencyRatio)
		prometheus.MustRegister(outputTokensAdjustedWastedTotal)
		prometheus.MustRegister(outputTokensRequestedWastedTotal)
		prometheus.MustRegister(outputTokensAdjustedCurrent)
		prometheus.MustRegister(reasoningTokensUsed)
		prometheus.MustRegister(modelRecognitionTotal)
	})
}

// init automatically registers metrics when package is imported
func init() {
	RegisterMetrics()
}
