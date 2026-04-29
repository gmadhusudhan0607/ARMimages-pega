// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package test_functions

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
)

// FetchMetrics fetches metrics from the specified URL
func FetchMetrics(url string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// ParsePrometheusGaugeMetric parses a Prometheus gauge metric value from the metrics text
// Format: metric_name{label="value"} numeric_value
func ParsePrometheusGaugeMetric(metricsText, metricName, labelName, labelValue string) (float64, error) {
	// Escape special regex characters in metric name
	escapedMetricName := regexp.QuoteMeta(metricName)
	escapedLabelName := regexp.QuoteMeta(labelName)
	escapedLabelValue := regexp.QuoteMeta(labelValue)

	// Build regex pattern to match the metric line
	// Pattern: metric_name{label="value"} numeric_value
	pattern := fmt.Sprintf(`%s\{%s="%s"\}\s+([0-9.]+)`, escapedMetricName, escapedLabelName, escapedLabelValue)
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(metricsText)
	if len(matches) < 2 {
		return 0, fmt.Errorf("metric not found: %s{%s=\"%s\"}", metricName, labelName, labelValue)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse metric value: %w", err)
	}

	return value, nil
}

// WaitForMetricValue polls the metrics endpoint until the specified metric reaches the target value
// or until the timeout is reached
func WaitForMetricValue(url, metricName, labelName, labelValue string, targetValue float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		metricsText, err := FetchMetrics(url)
		if err != nil {
			// Continue polling if metrics endpoint is not ready yet
			<-ticker.C
			continue
		}

		value, err := ParsePrometheusGaugeMetric(metricsText, metricName, labelName, labelValue)
		if err != nil {
			// Metric might not be published yet, continue polling
			<-ticker.C
			continue
		}

		if value >= targetValue {
			return nil
		}

		<-ticker.C
	}

	return fmt.Errorf("timeout waiting for metric %s{%s=\"%s\"} to reach %.2f", metricName, labelName, labelValue, targetValue)
}

// WaitForMetricValueBetween polls the metrics endpoint until the specified metric value is between min and max values
// or until the timeout is reached
func WaitForMetricValueBetween(url, metricName, labelName, labelValue string, minValue, maxValue float64, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		metricsText, err := FetchMetrics(url)
		if err != nil {
			// Continue polling if metrics endpoint is not ready yet
			<-ticker.C
			continue
		}

		value, err := ParsePrometheusGaugeMetric(metricsText, metricName, labelName, labelValue)
		By(fmt.Sprintf(" -> WaitForMetricValueBetween: current value of %s{%s=\"%s\"} is %.2f", metricName, labelName, labelValue, value))

		if err != nil {
			// Metric might not be published yet, continue polling
			<-ticker.C
			continue
		}

		if value >= minValue && value <= maxValue {
			return nil
		}

		// If value is already above max, stop immediately - it's unlikely to go back down
		if value > maxValue {
			return fmt.Errorf("metric %s{%s=\"%s\"} exceeded maximum value: current=%.2f, max=%.2f", metricName, labelName, labelValue, value, maxValue)
		}

		<-ticker.C
	}

	return fmt.Errorf("timeout waiting for metric %s{%s=\"%s\"} to be between %.2f and %.2f", metricName, labelName, labelValue, minValue, maxValue)
}

// ParsePrometheusLabeledMetric parses a Prometheus metric value with multiple labels
// Format: metric_name{label1="value1",label2="value2",...} numeric_value
func ParsePrometheusLabeledMetric(metricsText, metricName string, labels map[string]string) (float64, error) {
	// Escape special regex characters in metric name
	escapedMetricName := regexp.QuoteMeta(metricName)

	// Build label matcher pattern - order may vary in prometheus output
	// We'll match any order of labels
	labelPatterns := make([]string, 0, len(labels))
	for key, value := range labels {
		escapedKey := regexp.QuoteMeta(key)
		escapedValue := regexp.QuoteMeta(value)
		labelPatterns = append(labelPatterns, fmt.Sprintf(`%s="%s"`, escapedKey, escapedValue))
	}

	// Create a pattern that matches all labels in any order
	// This is a simplified approach - we'll check if the line contains all label pairs
	lines := regexp.MustCompile(`\n`).Split(metricsText, -1)

	for _, line := range lines {
		// Check if line starts with our metric name
		if !regexp.MustCompile(`^` + escapedMetricName + `\{`).MatchString(line) {
			continue
		}

		// Check if all required labels are present
		allLabelsMatch := true
		for _, labelPattern := range labelPatterns {
			if !regexp.MustCompile(labelPattern).MatchString(line) {
				allLabelsMatch = false
				break
			}
		}

		if !allLabelsMatch {
			continue
		}

		// Extract the numeric value at the end of the line
		valuePattern := regexp.MustCompile(`\}\s+([0-9.]+)`)
		matches := valuePattern.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		value, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse metric value: %w", err)
		}

		return value, nil
	}

	return 0, fmt.Errorf("metric not found: %s with labels %v", metricName, labels)
}

// GetLabeledMetricValue fetches and parses a labeled metric from the metrics endpoint
func GetLabeledMetricValue(url, metricName string, labels map[string]string) (float64, error) {
	metricsText, err := FetchMetrics(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch metrics: %w", err)
	}

	return ParsePrometheusLabeledMetric(metricsText, metricName, labels)
}

// WaitForLabeledMetric polls the metrics endpoint until a metric with specific labels appears
func WaitForLabeledMetric(url, metricName string, labels map[string]string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		metricsText, err := FetchMetrics(url)
		if err != nil {
			// Continue polling if metrics endpoint is not ready yet
			<-ticker.C
			continue
		}

		_, err = ParsePrometheusLabeledMetric(metricsText, metricName, labels)
		if err == nil {
			// Metric found
			return nil
		}

		<-ticker.C
	}

	return fmt.Errorf("timeout waiting for metric %s with labels %v", metricName, labels)
}
