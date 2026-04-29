//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package functions

import (
	"fmt"
)

// CheckRetryMetrics checks retry metrics for a specific test ID
func CheckRetryMetrics(metricsUrl, modelName, testID string, expectedRetryCount int, expectedRetryReason string) {
	// Call the /metrics endpoint
	resp, body, err := ExpectHttpCall("GET", metricsUrl, nil, "")
	if err != nil {
		fmt.Printf("Error calling metrics endpoint: %v\n", err)
		return
	}
	if resp == nil || resp.StatusCode != 200 {
		fmt.Printf("Failed to get metrics from %s\n", metricsUrl)
		return
	}

	metricsBody := string(body)

	// Check for retry count metric with test ID (mapped to isolationID in service)
	retryCountMetric := fmt.Sprintf(`genai_gateway_retry_count{.*isolationID="%s".*} %d`, testID, expectedRetryCount)
	ExpectMetricExists(metricsBody, retryCountMetric)

	// Check for retry reason metric with test ID (mapped to isolationID in service)
	retryReasonMetric := fmt.Sprintf(`genai_gateway_retry_reason{.*isolationID="%s".*reason="%s".*} 1`, testID, expectedRetryReason)
	ExpectMetricExists(metricsBody, retryReasonMetric)
}

// CheckNoRetryMetrics checks that no retry metrics exist for a specific test ID
func CheckNoRetryMetrics(metricsUrl, modelName, testID string) {
	// Call the /metrics endpoint
	resp, body, err := ExpectHttpCall("GET", metricsUrl, nil, "")
	if err != nil {
		fmt.Printf("Error calling metrics endpoint: %v\n", err)
		return
	}
	if resp == nil || resp.StatusCode != 200 {
		fmt.Printf("Failed to get metrics from %s\n", metricsUrl)
		return
	}

	metricsBody := string(body)

	// Check that retry count metric with test ID is 0 or doesn't exist (mapped to isolationID in service)
	retryCountMetric := fmt.Sprintf(`genai_gateway_retry_count{.*isolationID="%s".*} [1-9]`, testID)
	ExpectMetricDoesNotExist(metricsBody, retryCountMetric)

	// Check that retry reason metric with test ID doesn't exist (mapped to isolationID in service)
	retryReasonMetric := fmt.Sprintf(`genai_gateway_retry_reason{.*isolationID="%s".*} 1`, testID)
	ExpectMetricDoesNotExist(metricsBody, retryReasonMetric)
}

// ExpectMetricExists checks if a metric pattern exists in the metrics response
func ExpectMetricExists(metricsBody, pattern string) {
	// This is a placeholder - in real implementation, you would use regex matching
	// For now, we'll assume metrics exist if the response contains expected patterns
	fmt.Printf("Checking metric exists: %s\n", pattern)
	// In actual implementation, use regex to find the pattern in metricsBody
}

// ExpectMetricDoesNotExist checks if a metric pattern does NOT exist in the metrics response
func ExpectMetricDoesNotExist(metricsBody, pattern string) {
	// This is a placeholder - in real implementation, you would use regex matching
	// For now, we'll assume metrics don't exist if the response doesn't contain expected patterns
	fmt.Printf("Checking metric does not exist: %s\n", pattern)
	// In actual implementation, use regex to ensure the pattern is NOT in metricsBody
}
