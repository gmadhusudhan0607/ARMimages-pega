/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package metrics

import (
	"fmt"
	"strconv"

	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/gin-gonic/gin"
)

// UpdateTimingMetrics updates Prometheus timing metrics
func UpdateTimingMetrics(requestMetrics *RequestMetrics, labels map[string]string) {
	if requestMetrics == nil {
		return
	}

	timingMetrics := &requestMetrics.TimingMetrics

	// Update request duration histogram
	if timingMetrics.Duration > 0 {
		durationMs := float64(timingMetrics.Duration.Nanoseconds()) / 1e6 // Convert to milliseconds
		requestDuration.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
			labels["statusCode"],
			labels["path"],
			labels["method"],
		).Observe(durationMs)
	}

}

// PrometheusLabelsInput contains all the fields needed to create Prometheus labels
type PrometheusLabelsInput struct {
	IsolationID        string
	Path               string
	Method             string
	StatusCode         string
	OriginalModelName  string
	TargetModelName    string
	TargetModelVersion string
	TargetModelID      string
	Provider           string
	Infrastructure     string
}

// PrometheusLabelsInputWithIntStatusCode contains fields for creating Prometheus labels with int status code
type PrometheusLabelsInputWithIntStatusCode struct {
	IsolationID        string
	Path               string
	Method             string
	StatusCode         int
	OriginalModelName  string
	TargetModelName    string
	TargetModelVersion string
	TargetModelID      string
	Provider           string
	Infrastructure     string
}

// CreatePrometheusLabels creates a standardized label map for Prometheus metrics
func CreatePrometheusLabels(input PrometheusLabelsInput) map[string]string {
	// Handle infrastructure: use empty string instead of "unknown"
	infrastructure := input.Infrastructure
	if infrastructure == "unknown" {
		infrastructure = ""
	}

	return map[string]string{
		"isolationID":        input.IsolationID,
		"path":               input.Path,
		"method":             input.Method,
		"statusCode":         input.StatusCode,
		"originalModelName":  input.OriginalModelName,
		"targetModelName":    input.TargetModelName,
		"targetModelVersion": input.TargetModelVersion,
		"targetModelID":      input.TargetModelID,
		"provider":           input.Provider,
		"infrastructure":     infrastructure,
	}
}

// CreatePrometheusLabelsFromStatusCode creates labels with status code as int
func CreatePrometheusLabelsFromStatusCode(input PrometheusLabelsInputWithIntStatusCode) map[string]string {
	return CreatePrometheusLabels(PrometheusLabelsInput{
		IsolationID:        input.IsolationID,
		Path:               input.Path,
		Method:             input.Method,
		StatusCode:         strconv.Itoa(input.StatusCode),
		OriginalModelName:  input.OriginalModelName,
		TargetModelName:    input.TargetModelName,
		TargetModelVersion: input.TargetModelVersion,
		TargetModelID:      input.TargetModelID,
		Provider:           input.Provider,
		Infrastructure:     input.Infrastructure,
	})
}

// UpdateTokenMetrics updates Prometheus token metrics
func UpdateTokenMetrics(requestMetrics *RequestMetrics, labels map[string]string) {
	if requestMetrics == nil {
		return
	}

	tokenMetrics := &requestMetrics.TokenMetrics
	isStreaming := requestMetrics.IsStreaming

	// Always update request-side metrics
	updateRequestedTokensMetric(tokenMetrics, labels)
	updateMaximumTokensMetric(tokenMetrics, labels)
	updateAdjustedTokensMetric(tokenMetrics, labels)

	// Always update reasoning tokens metric (for both streaming and non-streaming)
	updateReasoningTokensMetric(tokenMetrics, labels)

	// Skip response-side metrics for streaming requests
	if isStreaming {
		return
	}

	// Only update these for non-streaming requests
	updateUsedTokensMetric(tokenMetrics, labels)

	// Update efficiency ratio metrics and wasted tokens - only when Used tokens > 0 to avoid division by zero
	if tokenMetrics.Used != nil && *tokenMetrics.Used > 0 {
		updateEfficiencyMetrics(tokenMetrics, labels)
		updateWastedTokenMetrics(tokenMetrics, labels)
	}
}

// updateRequestedTokensMetric updates the requested tokens histogram
func updateRequestedTokensMetric(tokenMetrics *TokenMetrics, labels map[string]string) {
	if tokenMetrics.Requested != nil && *tokenMetrics.Requested > 0 {
		outputTokensRequested.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
			labels["targetModelEndpoint"],
		).Observe(*tokenMetrics.Requested)
	}
}

// updateMaximumTokensMetric updates the maximum tokens gauge
func updateMaximumTokensMetric(tokenMetrics *TokenMetrics, labels map[string]string) {
	if tokenMetrics.Maximum != nil && *tokenMetrics.Maximum > 0 {
		outputTokensMaximum.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
			labels["modelVersion"],
			labels["targetModelEndpoint"],
		).Set(*tokenMetrics.Maximum)
	}
}

// updateUsedTokensMetric updates the used tokens histogram
func updateUsedTokensMetric(tokenMetrics *TokenMetrics, labels map[string]string) {
	if tokenMetrics.Used != nil && *tokenMetrics.Used > 0 {
		outputTokensUsed.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
		).Observe(*tokenMetrics.Used)
	}
}

// updateReasoningTokensMetric updates the reasoning tokens histogram for reasoning models
func updateReasoningTokensMetric(tokenMetrics *TokenMetrics, labels map[string]string) {
	if tokenMetrics.ReasoningTokens != nil && *tokenMetrics.ReasoningTokens > 0 {
		reasoningTokensUsed.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
		).Observe(*tokenMetrics.ReasoningTokens)
	}
}

// updateAdjustedTokensMetric updates the adjusted tokens histogram
func updateAdjustedTokensMetric(tokenMetrics *TokenMetrics, labels map[string]string) {
	if tokenMetrics.Adjusted != nil && *tokenMetrics.Adjusted > 0 {
		outputTokensAdjusted.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
		).Observe(*tokenMetrics.Adjusted)
	}
}

// updateEfficiencyMetrics updates efficiency ratio metrics
func updateEfficiencyMetrics(tokenMetrics *TokenMetrics, labels map[string]string) {
	// Calculate adjustment efficiency ratio: adjusted_tokens / used_tokens
	if tokenMetrics.Adjusted != nil && *tokenMetrics.Adjusted > 0 {
		adjustmentRatio := *tokenMetrics.Adjusted / *tokenMetrics.Used
		outputTokensAdjustedEfficiencyRatio.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
		).Observe(adjustmentRatio)
	}

	// Calculate requestment efficiency ratio: requested_tokens / used_tokens
	if tokenMetrics.Requested != nil && *tokenMetrics.Requested > 0 {
		requestmentRatio := *tokenMetrics.Requested / *tokenMetrics.Used
		outputTokensRequestedEfficiencyRatio.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
		).Observe(requestmentRatio)
	}
}

// updateWastedTokenMetrics updates wasted token metrics
func updateWastedTokenMetrics(tokenMetrics *TokenMetrics, labels map[string]string) {
	// Calculate and record wasted tokens when adjustment was inefficient
	if tokenMetrics.Adjusted != nil && *tokenMetrics.Adjusted > *tokenMetrics.Used {
		wastedAdjusted := *tokenMetrics.Adjusted - *tokenMetrics.Used
		outputTokensAdjustedWastedTotal.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
		).Add(wastedAdjusted)
	}

	// Calculate and record wasted tokens when original/default request was inefficient
	// This covers both cases:
	// 1. When max_tokens was explicitly provided in original request (tokenMetrics.Requested != nil)
	// 2. When no max_tokens was provided and service used default value (tokenMetrics.Requested == nil, use tokenMetrics.Adjusted as the "requested" amount)
	requestedTokens := determineRequestedTokens(tokenMetrics)
	if requestedTokens > 0 && requestedTokens > *tokenMetrics.Used {
		wastedRequested := requestedTokens - *tokenMetrics.Used
		outputTokensRequestedWastedTotal.WithLabelValues(
			labels["isolationID"],
			labels["infrastructure"],
			labels["provider"],
			labels["creator"],
			labels["originalModelName"],
			labels["targetModelName"],
			labels["targetModelVersion"],
			labels["targetModelID"],
		).Add(wastedRequested)
	}
}

// determineRequestedTokens determines the requested tokens value for waste calculation
func determineRequestedTokens(tokenMetrics *TokenMetrics) float64 {
	if tokenMetrics.Requested != nil {
		return *tokenMetrics.Requested
	}
	if tokenMetrics.Adjusted != nil {
		// When no max_tokens was provided in original request, use the adjusted value as the "requested" amount
		// since that represents what the service decided to use as the effective max_tokens
		return *tokenMetrics.Adjusted
	}
	return 0
}

// SafeStringValue returns the string value or empty string if empty
func SafeStringValue(value string) string {
	if value == "" {
		return ""
	}
	return value
}

// MetadataExtractor defines the interface for extracting metadata fields
type MetadataExtractor interface {
	GetIsolationID() string
	GetOriginalModelName() string
	GetTargetModelName() string
	GetTargetModelID() string
	GetTargetModelVersion() string
	GetTargetModelCreator() string
	GetTargetModelInfrastructure() string
	GetTargetModel() *modeltypes.Model
}

// ExtractMetadataFromContext safely extracts metadata from gin context
func ExtractMetadataFromContext(c *gin.Context) (MetadataExtractor, bool) {
	ctx := c.Request.Context()
	// Use the proper context key type to avoid collisions
	if metadata := ctx.Value(RequestMetadataContextKey{}); metadata != nil {
		// Since metadata is now stored as a pointer, try direct interface cast
		if extractor, ok := metadata.(MetadataExtractor); ok {
			return extractor, true
		}
	}
	return nil, false
}

// CreatePrometheusLabelsFromContext creates comprehensive labels from gin context
func CreatePrometheusLabelsFromContext(c *gin.Context, statusCode int) map[string]string {
	// Initialize default values
	isolationID := ""
	originalModelName := ""
	targetModelName := ""
	targetModelVersion := ""
	targetModelID := ""
	creator := ""
	infrastructure := ""
	provider := ""

	// Extract metadata from context
	if metadata, ok := ExtractMetadataFromContext(c); ok {
		isolationID = metadata.GetIsolationID()
		originalModelName = metadata.GetOriginalModelName()
		targetModelName = metadata.GetTargetModelName()
		targetModelID = metadata.GetTargetModelID()
		targetModelVersion = metadata.GetTargetModelVersion()
		creator = metadata.GetTargetModelCreator()
		infrastructure = metadata.GetTargetModelInfrastructure()

		// Extract provider from target model if available
		if targetModel := metadata.GetTargetModel(); targetModel != nil {
			provider = string(targetModel.Provider)
		}
	}

	// Handle infrastructure: use empty string instead of "unknown"
	if infrastructure == "unknown" {
		infrastructure = ""
	}

	// Create comprehensive label map for both timing and token metrics
	labels := map[string]string{
		// Labels for timing metrics
		"isolationID":        isolationID,
		"statusCode":         strconv.Itoa(statusCode),
		"path":               SafeStringValue(c.Request.URL.Path),
		"method":             SafeStringValue(c.Request.Method),
		"originalModelName":  SafeStringValue(originalModelName),
		"targetModelName":    SafeStringValue(targetModelName),
		"targetModelVersion": SafeStringValue(targetModelVersion),
		"targetModelID":      SafeStringValue(targetModelID),
		"provider":           SafeStringValue(provider),
		"infrastructure":     infrastructure,

		// Labels for token metrics (additional/different naming)
		"creator":             SafeStringValue(creator),
		"targetModelEndpoint": SafeStringValue(c.Request.URL.Path),
		"modelVersion":        SafeStringValue(targetModelVersion),
	}

	return labels
}

// ValidateLabelsForTimingMetrics ensures all required labels for timing metrics are present
func ValidateLabelsForTimingMetrics(labels map[string]string) error {
	requiredLabels := []string{
		"isolationID", "infrastructure", "provider", "creator", "originalModelName",
		"targetModelName", "targetModelVersion", "targetModelID",
		"statusCode", "path", "method",
	}

	for _, label := range requiredLabels {
		if _, exists := labels[label]; !exists {
			return fmt.Errorf("missing required label for timing metrics: %s", label)
		}
	}
	return nil
}

// UpdateAdjustedCurrentMetric updates the adjusted current metric for all strategies
func UpdateAdjustedCurrentMetric(currentValue float64, labels map[string]string) {
	outputTokensAdjustedCurrent.WithLabelValues(
		labels["isolationID"],
		labels["infrastructure"],
		labels["provider"],
		labels["creator"],
		labels["originalModelName"],
		labels["targetModelName"],
		labels["targetModelVersion"],
		labels["targetModelID"],
	).Set(currentValue)
}

// ValidateLabelsForTokenMetrics ensures all required labels for token metrics are present
func ValidateLabelsForTokenMetrics(labels map[string]string) error {
	requiredLabels := []string{
		"isolationID", "infrastructure", "provider", "creator",
		"targetModelID", "targetModelEndpoint",
	}

	for _, label := range requiredLabels {
		if _, exists := labels[label]; !exists {
			return fmt.Errorf("missing required label for token metrics: %s", label)
		}
	}
	return nil
}

// IncrementModelRecognition increments the model recognition counter
func IncrementModelRecognition(isolationID, status, originalModelName string) {
	modelRecognitionTotal.WithLabelValues(
		isolationID,
		status,
		originalModelName,
	).Inc()
}
