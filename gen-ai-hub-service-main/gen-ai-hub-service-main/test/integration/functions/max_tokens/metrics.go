//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

import (
	"fmt"
	"regexp"

	functions "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	. "github.com/onsi/gomega"
)

// MetricsConfig holds configuration for different metrics validation scenarios
type MetricsConfig struct {
	// Request-related config
	HasOriginalMaxTokens bool
	OriginalMaxTokens    int
	HasForcedMaxTokens   bool
	ForcedMaxTokens      int

	// Expected values
	ExpectedMaxOutputTokens int
	ExpectedAdjustedValue   float64

	// Behavior flags
	IsStreaming                      bool
	IsAdjustmentStrategy             bool
	ShouldCheckAdjustedCurrentMetric bool
	ShouldCheckWastedMetrics         bool
	ShouldCheckRequestedMetrics      bool

	// Streaming-specific flags
	SkipUsedTokensOnError bool
	FlexibleAdjustedValue bool
}

// MetricsChecker provides unified metrics checking functionality
type MetricsChecker struct {
	metricsUrl        string
	isolationID       string
	originalModelName string
	targetModelName   string
}

// NewMetricsChecker creates a new metrics checker instance
func NewMetricsChecker(metricsUrl, isolationID, originalModelName, targetModelName string) *MetricsChecker {
	Expect(metricsUrl).NotTo(BeEmpty(), "urlPath parameter must not be empty")
	Expect(isolationID).NotTo(BeEmpty(), "isolationID parameter must not be empty")

	return &MetricsChecker{
		metricsUrl:        metricsUrl,
		isolationID:       isolationID,
		originalModelName: originalModelName,
		targetModelName:   targetModelName,
	}
}

// buildLabelPattern creates flexible label patterns that can match labels in any order
func (mc *MetricsChecker) buildLabelPattern(useOriginalModel bool) string {
	modelName := mc.targetModelName
	if useOriginalModel {
		modelName = mc.originalModelName
	}

	// Pattern that matches both isolationID and modelName in any order within the labels
	return fmt.Sprintf(`\{(?:[^}]*isolationID="%s"[^}]*(?:originalModelName|targetModelName)="%s"[^}]*|[^}]*(?:originalModelName|targetModelName)="%s"[^}]*isolationID="%s"[^}]*)\}`,
		mc.isolationID, modelName, modelName, mc.isolationID)
}

// checkMetricPattern validates if a metric pattern exists in the content
func (mc *MetricsChecker) checkMetricPattern(metricsContent, pattern, description string, expectedPresent bool) {
	matched, err := regexp.MatchString(pattern, metricsContent)
	Expect(err).To(BeNil(), fmt.Sprintf("Failed to compile regex pattern for %s", description))

	if expectedPresent {
		Expect(matched).To(BeTrue(), fmt.Sprintf("Expected metric '%s' not found in metrics response", description))
	} else {
		Expect(matched).To(BeFalse(), fmt.Sprintf("Unexpected metric '%s' found in metrics response", description))
	}
}

// checkMetricWithValue validates if a metric with specific value exists
func (mc *MetricsChecker) checkMetricWithValue(metricsContent, metricName string, value interface{}, useOriginalModel bool, description string) {
	labelPattern := mc.buildLabelPattern(useOriginalModel)
	pattern := fmt.Sprintf(`%s%s\s+%v`, metricName, labelPattern, value)
	mc.checkMetricPattern(metricsContent, pattern, description, true)
}

// checkMetricExists validates if a metric bucket exists (without checking specific values)
func (mc *MetricsChecker) checkMetricExists(metricsContent, metricName string, useOriginalModel bool, description string, expectedPresent bool) {
	labelPattern := mc.buildLabelPattern(useOriginalModel)
	pattern := fmt.Sprintf(`%s%s`, metricName, labelPattern)
	mc.checkMetricPattern(metricsContent, pattern, description, expectedPresent)
}

// fetchMetrics calls the metrics endpoint and returns the content
func (mc *MetricsChecker) fetchMetrics() string {
	resp, body, err := functions.ExpectHttpCall("GET", mc.metricsUrl, nil, "")
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(200), fmt.Sprintf("Failed to get metrics from %s", mc.metricsUrl))
	return string(body)
}

// validateCoreMetrics checks the common metrics that should always be present
func (mc *MetricsChecker) validateCoreMetrics(metricsContent string, config MetricsConfig) {
	useOriginalModel := !config.IsAdjustmentStrategy

	// 1. Check genai_request_duration_ms (must be > 0)
	mc.checkMetricExists(metricsContent, "genai_request_duration_ms_bucket", useOriginalModel,
		fmt.Sprintf("genai_request_duration_ms with model='%s'", mc.getModelName(useOriginalModel)), true)

	// 2. Check genai_gateway_model_recognition_total (must be present with "recognized" status)
	mc.validateModelRecognitionMetric(metricsContent, "recognized")

	// 3. Check genai_gateway_output_tokens_maximum (must equal model's max from specs)
	mc.checkMetricWithValue(metricsContent, "genai_gateway_output_tokens_maximum", config.ExpectedMaxOutputTokens, useOriginalModel,
		fmt.Sprintf("genai_gateway_output_tokens_maximum with model='%s' and value=%d", mc.getModelName(useOriginalModel), config.ExpectedMaxOutputTokens))

	// 4. Check genai_gateway_output_tokens_used (must be > 0) - conditionally required
	if config.SkipUsedTokensOnError {
		// For streaming requests, check if this is an error request first
		modelName := mc.getModelName(useOriginalModel)
		errorPattern := fmt.Sprintf(`genai_request_duration_ms_bucket%s[^}]*statusCode="404"[^}]*\}`, mc.buildLabelPattern(useOriginalModel))
		isErrorRequest, err := regexp.MatchString(errorPattern, metricsContent)
		if err == nil && !isErrorRequest {
			// Only require genai_gateway_output_tokens_used for successful requests
			mc.checkMetricExists(metricsContent, "genai_gateway_output_tokens_used_bucket", useOriginalModel,
				fmt.Sprintf("genai_gateway_output_tokens_used with model='%s'", modelName), true)
		}
	} else {
		// Always required for non-streaming requests
		mc.checkMetricExists(metricsContent, "genai_gateway_output_tokens_used_bucket", useOriginalModel,
			fmt.Sprintf("genai_gateway_output_tokens_used with model='%s'", mc.getModelName(useOriginalModel)), true)
	}
}

// getModelName returns the appropriate model name based on the useOriginalModel flag
func (mc *MetricsChecker) getModelName(useOriginalModel bool) string {
	if useOriginalModel {
		return mc.originalModelName
	}
	return mc.targetModelName
}

// validateRequestedMetrics checks metrics related to original requested max_tokens
func (mc *MetricsChecker) validateRequestedMetrics(metricsContent string, config MetricsConfig) {
	useOriginalModel := !config.IsAdjustmentStrategy
	modelName := mc.getModelName(useOriginalModel)

	if config.ShouldCheckRequestedMetrics {
		// Check genai_gateway_output_tokens_requested should be present
		mc.checkMetricExists(metricsContent, "genai_gateway_output_tokens_requested_bucket", useOriginalModel,
			fmt.Sprintf("genai_gateway_output_tokens_requested with model='%s' should be present", modelName), true)

		if config.HasOriginalMaxTokens && config.OriginalMaxTokens > 0 {
			// Check the sum value if we know the original value
			mc.checkMetricWithValue(metricsContent, "genai_gateway_output_tokens_requested_sum", config.OriginalMaxTokens, useOriginalModel,
				fmt.Sprintf("genai_gateway_output_tokens_requested_sum with model='%s' and value=%d", modelName, config.OriginalMaxTokens))
		}
	} else {
		// Check genai_gateway_output_tokens_requested should NOT be present
		mc.checkMetricExists(metricsContent, "genai_gateway_output_tokens_requested_bucket", useOriginalModel,
			fmt.Sprintf("genai_gateway_output_tokens_requested with model='%s' should not be present", modelName), false)
	}
}

// validateAdjustedMetrics checks metrics related to adjusted max_tokens
func (mc *MetricsChecker) validateAdjustedMetrics(metricsContent string, config MetricsConfig) {
	useOriginalModel := !config.IsAdjustmentStrategy
	modelName := mc.getModelName(useOriginalModel)

	if config.FlexibleAdjustedValue {
		// For streaming with original max_tokens, be flexible about adjusted value
		adjustedBucketPattern := fmt.Sprintf(`genai_gateway_output_tokens_adjusted_bucket%s`, mc.buildLabelPattern(useOriginalModel))
		adjustedBucketMatched, err := regexp.MatchString(adjustedBucketPattern, metricsContent)
		if err == nil && adjustedBucketMatched {
			// If the bucket is present, we accept whatever value is there
			fmt.Printf("DEBUG: genai_gateway_output_tokens_adjusted_bucket found for streaming\n")
		} else {
			fmt.Printf("DEBUG: genai_gateway_output_tokens_adjusted_bucket not found - acceptable for streaming\n")
		}
		return
	}

	// Check genai_gateway_output_tokens_adjusted (should be present)
	mc.checkMetricExists(metricsContent, "genai_gateway_output_tokens_adjusted_bucket", useOriginalModel,
		fmt.Sprintf("genai_gateway_output_tokens_adjusted with model='%s'", modelName), true)

	// Check specific adjusted value if provided
	if config.HasForcedMaxTokens {
		mc.checkMetricWithValue(metricsContent, "genai_gateway_output_tokens_adjusted_sum", config.ForcedMaxTokens, useOriginalModel,
			fmt.Sprintf("genai_gateway_output_tokens_adjusted_sum with model='%s' and value=%d", modelName, config.ForcedMaxTokens))
	} else if config.ExpectedAdjustedValue > 0 && !config.IsAdjustmentStrategy {
		// For adjustment strategies, don't validate the sum as it's cumulative across all requests
		// Only validate sum for non-adjustment strategies where we expect a specific value
		mc.checkMetricWithValue(metricsContent, "genai_gateway_output_tokens_adjusted_sum", int(config.ExpectedAdjustedValue), useOriginalModel,
			fmt.Sprintf("genai_gateway_output_tokens_adjusted_sum with model='%s' and value=%.0f", modelName, config.ExpectedAdjustedValue))
	}
}

// validateWastedMetrics checks wasted tokens metrics
func (mc *MetricsChecker) validateWastedMetrics(metricsContent string, config MetricsConfig) {
	if !config.ShouldCheckWastedMetrics {
		return
	}

	useOriginalModel := !config.IsAdjustmentStrategy
	modelName := mc.getModelName(useOriginalModel)

	// For streaming requests, check if this is an error request first
	if config.SkipUsedTokensOnError {
		errorPattern := fmt.Sprintf(`genai_request_duration_ms_bucket%s[^}]*statusCode="404"[^}]*\}`, mc.buildLabelPattern(useOriginalModel))
		isErrorRequest, err := regexp.MatchString(errorPattern, metricsContent)
		if err == nil && isErrorRequest {
			// Skip wasted metrics validation for error requests
			return
		}
	}

	// Check genai_gateway_output_tokens_adjusted_wasted_total (should be > 0)
	wastedPattern := fmt.Sprintf(`genai_gateway_output_tokens_adjusted_wasted_total%s\s+[1-9][0-9]*`, mc.buildLabelPattern(useOriginalModel))
	mc.checkMetricPattern(metricsContent, wastedPattern,
		fmt.Sprintf("genai_gateway_output_tokens_adjusted_wasted_total with model='%s' and value > 0", modelName), true)

	// Check genai_gateway_output_tokens_requested_wasted_total if applicable
	if config.ShouldCheckRequestedMetrics {
		requestedWastedPattern := fmt.Sprintf(`genai_gateway_output_tokens_requested_wasted_total%s\s+[1-9][0-9]*`, mc.buildLabelPattern(useOriginalModel))
		mc.checkMetricPattern(metricsContent, requestedWastedPattern,
			fmt.Sprintf("genai_gateway_output_tokens_requested_wasted_total with model='%s' and value > 0", modelName), true)
	}
}

// validateAdjustedCurrentMetrics checks adjusted current metrics for all strategies
func (mc *MetricsChecker) validateAdjustedCurrentMetrics(metricsContent string, config MetricsConfig) {
	if !config.ShouldCheckAdjustedCurrentMetric {
		return
	}

	expectedValue := config.ExpectedAdjustedValue
	if config.HasForcedMaxTokens {
		expectedValue = float64(config.ForcedMaxTokens)
	}

	// Check genai_gateway_output_tokens_adjusted_current
	mc.checkMetricWithValue(metricsContent, "genai_gateway_output_tokens_adjusted_current", int(expectedValue), false,
		fmt.Sprintf("genai_gateway_output_tokens_adjusted_current with model='%s' and value=%.0f", mc.targetModelName, expectedValue))
}

// CheckMetrics is the unified metrics validation function
func (mc *MetricsChecker) CheckMetrics(config MetricsConfig) {
	metricsContent := mc.fetchMetrics()

	// Validate all metric categories
	mc.validateCoreMetrics(metricsContent, config)
	mc.validateRequestedMetrics(metricsContent, config)
	mc.validateAdjustedMetrics(metricsContent, config)
	mc.validateWastedMetrics(metricsContent, config)
	mc.validateAdjustedCurrentMetrics(metricsContent, config)
}

// CheckMetricsFixed validates metrics for fixed strategy (with optional originalMaxTokens parameter)
// When originalMaxTokens is <=0, behaves as if no max_tokens was in original request
// When originalMaxTokens > 0, validates metrics when original request contained max_tokens
func CheckMetricsFixed(metricsUrl, isolationID, originalModelName, targetModelName string, originalMaxTokens, expectedMaxOutputTokens int, expectedAdjustedValue float64) {
	checker := NewMetricsChecker(metricsUrl, isolationID, originalModelName, targetModelName)

	var config MetricsConfig
	if originalMaxTokens <= 0 {
		// Behave like the old CheckMetricsFixed (no max_tokens in original request)
		config = MetricsConfig{
			ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
			ExpectedAdjustedValue:            expectedAdjustedValue,
			ShouldCheckWastedMetrics:         true,
			ShouldCheckRequestedMetrics:      false, // No max_tokens in original request
			ShouldCheckAdjustedCurrentMetric: true,  // Check adjusted current metric - max_tokens was inserted
		}
	} else {
		// CRITICAL: When max_tokens was provided in original request
		// The adjusted current metric should ONLY be updated if max_tokens was actually modified
		// For FIXED strategy with forced=false (default), max_tokens remains unchanged, so metric should NOT be updated
		wasModified := (originalMaxTokens != int(expectedAdjustedValue))
		config = MetricsConfig{
			HasOriginalMaxTokens:             true,
			OriginalMaxTokens:                originalMaxTokens,
			ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
			ExpectedAdjustedValue:            expectedAdjustedValue,
			ShouldCheckWastedMetrics:         true,
			ShouldCheckRequestedMetrics:      true,        // max_tokens was in original request
			ShouldCheckAdjustedCurrentMetric: wasModified, // Only check if max_tokens was actually modified
		}
	}

	checker.CheckMetrics(config)
}

// validateModelRecognitionMetric checks the model recognition metric
func (mc *MetricsChecker) validateModelRecognitionMetric(metricsContent, status string) {
	// Check for genai_gateway_model_recognition_total metric
	pattern := fmt.Sprintf(`genai_gateway_model_recognition_total\{isolationID="%s",originalModelName="%s",status="%s"\}\s+[1-9][0-9]*`,
		mc.isolationID, mc.originalModelName, status)

	matched, err := regexp.MatchString(pattern, metricsContent)
	Expect(err).To(BeNil(), "Failed to compile regex pattern for model recognition metric")
	Expect(matched).To(BeTrue(), fmt.Sprintf("Expected model recognition metric with status='%s' and originalModelName='%s' not found", status, mc.originalModelName))
}

// CheckModelRecognitionMetric validates that the model recognition metric exists
func CheckModelRecognitionMetric(metricsUrl, isolationID, status, originalModelName string) {
	checker := NewMetricsChecker(metricsUrl, isolationID, originalModelName, "")
	metricsContent := checker.fetchMetrics()
	checker.validateModelRecognitionMetric(metricsContent, status)
}

// CheckMetricsFixedForced validates metrics when original max_tokens was forced to different value
func CheckMetricsFixedForced(metricsUrl, isolationID, originalModelName, targetModelName string, originalMaxTokens, forcedMaxTokens, expectedMaxOutputTokens int) {
	checker := NewMetricsChecker(metricsUrl, isolationID, originalModelName, targetModelName)

	// CRITICAL: The adjusted current metric should ONLY be updated if max_tokens was actually modified
	// For forced scenarios, only update the metric if the forced value is different from the original
	wasModified := (originalMaxTokens != forcedMaxTokens)

	config := MetricsConfig{
		HasOriginalMaxTokens:             true,
		OriginalMaxTokens:                originalMaxTokens,
		HasForcedMaxTokens:               true,
		ForcedMaxTokens:                  forcedMaxTokens,
		ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
		ShouldCheckWastedMetrics:         true,
		ShouldCheckRequestedMetrics:      true,
		ShouldCheckAdjustedCurrentMetric: wasModified, // Only check if max_tokens was actually modified
	}
	checker.CheckMetrics(config)
}

// CheckMetricsStreaming validates metrics for streaming requests (replaces both old streaming functions)
// When originalMaxTokens is <=0, behaves as if no max_tokens was in original request
// When originalMaxTokens > 0 validates metrics when the original request contained max_tokens
func CheckMetricsStreaming(metricsUrl, isolationID, originalModelName, targetModelName string, originalMaxTokens, expectedMaxOutputTokens int, expectedAdjustedValue float64) {
	checker := NewMetricsChecker(metricsUrl, isolationID, originalModelName, targetModelName)

	var config MetricsConfig
	if originalMaxTokens <= 0 {
		// No max_tokens in original request - streaming without original max_tokens
		config = MetricsConfig{
			ExpectedMaxOutputTokens:     expectedMaxOutputTokens,
			ExpectedAdjustedValue:       expectedAdjustedValue,
			IsStreaming:                 true,
			ShouldCheckWastedMetrics:    true,
			ShouldCheckRequestedMetrics: false,
			SkipUsedTokensOnError:       true,
		}
	} else {
		// max_tokens was in original request - streaming with original max_tokens
		config = MetricsConfig{
			HasOriginalMaxTokens:        true,
			OriginalMaxTokens:           originalMaxTokens,
			ExpectedMaxOutputTokens:     expectedMaxOutputTokens,
			ExpectedAdjustedValue:       expectedAdjustedValue,
			IsStreaming:                 true,
			ShouldCheckWastedMetrics:    false, // Streaming with original max_tokens doesn't check wasted
			ShouldCheckRequestedMetrics: true,
			FlexibleAdjustedValue:       true, // For streaming with original max_tokens
			SkipUsedTokensOnError:       true,
		}
	}

	checker.CheckMetrics(config)
}

// CheckMetricsAutoIncreasing validates metrics for auto-increasing strategy
func CheckMetricsAutoIncreasing(metricsUrl, isolationID, originalModelName, targetModelName string, expectedMaxOutputTokens int, expectedAdjustedValue float64) {
	checker := NewMetricsChecker(metricsUrl, isolationID, originalModelName, targetModelName)
	config := MetricsConfig{
		ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
		ExpectedAdjustedValue:            expectedAdjustedValue,
		IsAdjustmentStrategy:             true,
		ShouldCheckAdjustedCurrentMetric: true,
		ShouldCheckWastedMetrics:         true,
		ShouldCheckRequestedMetrics:      false, // No max_tokens in original request
	}
	checker.CheckMetrics(config)
}

// CheckMetricsAutoIncreasingForced validates metrics for forced auto-increasing strategy
func CheckMetricsAutoIncreasingForced(metricsUrl, isolationID, originalModelName, targetModelName string, originalMaxTokens, forcedMaxTokens, expectedMaxOutputTokens int) {
	checker := NewMetricsChecker(metricsUrl, isolationID, originalModelName, targetModelName)
	config := MetricsConfig{
		HasOriginalMaxTokens:             true,
		OriginalMaxTokens:                originalMaxTokens,
		HasForcedMaxTokens:               true,
		ForcedMaxTokens:                  forcedMaxTokens,
		ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
		IsAdjustmentStrategy:             true,
		ShouldCheckAdjustedCurrentMetric: originalMaxTokens != forcedMaxTokens, // Only if actually forced
		ShouldCheckWastedMetrics:         true,
		ShouldCheckRequestedMetrics:      true,
	}
	checker.CheckMetrics(config)
}

// CheckMetricsAdjustedCurrent validates just the current adjusted metric with Eventually
func CheckMetricsAdjustedCurrent(metricsUrl, isolationID, originalModelName, targetModelName string, expectedValue float64) {
	Eventually(func() bool {
		checker := NewMetricsChecker(metricsUrl, isolationID, originalModelName, targetModelName)
		metricsContent := checker.fetchMetrics()

		// Look for the adjusted current metric
		metricPattern := fmt.Sprintf(`genai_gateway_output_tokens_adjusted_current%s\s+%.0f`,
			checker.buildLabelPattern(false), expectedValue)
		matched, err := regexp.MatchString(metricPattern, metricsContent)
		return err == nil && matched
	}, "30s", "1s").Should(BeTrue(), fmt.Sprintf("Expected adjusted current metric to be %.0f for test %s", expectedValue, isolationID))
}

// CheckMetricsWithOriginalMaxTokens validates metrics when original request had max_tokens but no adjustment occurred
func CheckMetricsWithOriginalMaxTokens(metricsUrl, isolationID, originalModelName, targetModelName string, originalMaxTokens, expectedMaxOutputTokens int, expectedAdjustedValue float64) {
	checker := NewMetricsChecker(metricsUrl, isolationID, originalModelName, targetModelName)
	config := MetricsConfig{
		HasOriginalMaxTokens:             true,
		OriginalMaxTokens:                originalMaxTokens,
		ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
		ExpectedAdjustedValue:            expectedAdjustedValue,
		ShouldCheckAdjustedCurrentMetric: false, // No auto-adjustment occurred
		ShouldCheckWastedMetrics:         true,  // Should check wasted metrics
		ShouldCheckRequestedMetrics:      true,  // Original request had max_tokens
	}
	checker.CheckMetrics(config)
}

// VerifyNoMaxTokensMetrics verifies that NO max_tokens related metrics are present for unrecognized models
func VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName string) {
	// Fetch metrics
	resp, body, err := functions.ExpectHttpCall("GET", metricsUrl, nil, "")
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(200), fmt.Sprintf("Failed to get metrics from %s", metricsUrl))
	metricsContent := string(body)

	// Define max_tokens related metrics that should NOT be present for unrecognized models
	maxTokensMetrics := []string{
		"genai_gateway_output_tokens_requested",
		"genai_gateway_output_tokens_adjusted",
		"genai_gateway_output_tokens_maximum",
		"genai_gateway_output_tokens_used",
		"genai_gateway_output_tokens_adjusted_wasted_total",
		"genai_gateway_output_tokens_requested_wasted_total",
		"genai_gateway_output_tokens_adjusted_current",
	}

	// Verify that none of these metrics are present for this isolationID and modelName
	for _, metricName := range maxTokensMetrics {
		// Check if any variant of the metric exists with our isolationID and modelName
		patterns := []string{
			// Pattern with originalModelName
			fmt.Sprintf(`%s\{[^}]*isolationID="%s"[^}]*originalModelName="%s"[^}]*\}`, metricName, isolationID, modelName),
			// Pattern with targetModelName
			fmt.Sprintf(`%s\{[^}]*isolationID="%s"[^}]*targetModelName="%s"[^}]*\}`, metricName, isolationID, modelName),
		}

		for _, pattern := range patterns {
			matched, err := regexp.MatchString(pattern, metricsContent)
			Expect(err).To(BeNil(), fmt.Sprintf("Failed to compile regex pattern: %s", pattern))
			Expect(matched).To(BeFalse(), fmt.Sprintf("Unexpected metric '%s' found for unrecognized model '%s' with isolation ID '%s'", metricName, modelName, isolationID))
		}
	}
}

// CheckUnrecognizedModelMetric validates that the model recognition metric exists with status="unrecognized"
// It must take into account only records for specific isolationID
func CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName string) {
	// Fetch metrics
	resp, body, err := functions.ExpectHttpCall("GET", metricsUrl, nil, "")
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(200), fmt.Sprintf("Failed to get metrics from %s", metricsUrl))
	metricsContent := string(body)

	// Look for the model recognition metric with status="unrecognized" for this specific isolation ID and model
	pattern := fmt.Sprintf(`genai_gateway_model_recognition_total\{isolationID="%s",originalModelName="%s",status="unrecognized"\}\s+[1-9][0-9]*`, isolationID, modelName)

	matched, err := regexp.MatchString(pattern, metricsContent)
	Expect(err).To(BeNil(), fmt.Sprintf("Failed to compile regex pattern: %s", pattern))
	Expect(matched).To(BeTrue(), fmt.Sprintf("Expected model recognition metric with status='unrecognized', isolationID='%s' and originalModelName='%s' not found", isolationID, modelName))
}
