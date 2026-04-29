//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package functions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// DefaultRetryConfig returns sensible defaults for retry operations
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   5,
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     2 * time.Second,
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(operation func() error, config RetryConfig) error {
	var lastErr error
	delay := config.InitialDelay

	for i := 0; i < config.MaxRetries; i++ {
		err := operation()
		if err == nil {
			if i > 0 {
				fmt.Printf("  Operation succeeded after %d retries\n", i)
			}
			return nil
		}

		lastErr = err
		if i < config.MaxRetries-1 {
			fmt.Printf("  Attempt %d/%d failed: %v. Retrying in %v...\n", i+1, config.MaxRetries, err, delay)
			time.Sleep(delay)
			// Exponential backoff with max delay cap
			delay *= 2
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, lastErr)
}

// WaitForWireMockReady polls WireMock until it's ready to accept requests
func WaitForWireMockReady(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	fmt.Printf("Waiting for WireMock to be ready at %s...\n", baseURL)
	attempts := 0

	for time.Now().Before(deadline) {
		attempts++
		// Try to access the admin API root endpoint
		resp, err := client.Get(baseURL + "/__admin/")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			fmt.Printf("  WireMock is ready after %d attempts\n", attempts)
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		if attempts%10 == 0 {
			fmt.Printf("  Still waiting for WireMock... (attempt %d)\n", attempts)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("WireMock not ready after %v (%d attempts)", timeout, attempts)
}

// WireMockExpectation represents a WireMock stub mapping
type WireMockExpectation struct {
	Id       string      `json:"id"`
	Request  interface{} `json:"request"`
	Response interface{} `json:"response"`
}

// CreateWireMockExpectation creates a new stub mapping
func CreateWireMockExpectation(wiremockURL string, mapping map[string]interface{}) (*WireMockExpectation, error) {
	response, body, err := makeWireMockRequest("POST", wiremockURL+"/__admin/mappings", mapping)
	if err != nil {
		return nil, fmt.Errorf("failed to create WireMock mapping: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create WireMock mapping, status: %d, body: %s", response.StatusCode, string(body))
	}

	var result WireMockExpectation
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WireMock mapping response: %w", err)
	}

	return &result, nil
}

// DeleteWireMockExpectation removes a stub mapping
// Ignores 404 errors as the mapping may have been cleared by ResetWireMockServer
func DeleteWireMockExpectation(wiremockURL, mappingId string) error {
	response, body, err := makeWireMockRequest("DELETE", wiremockURL+"/__admin/mappings/"+mappingId, nil)
	if err != nil {
		return fmt.Errorf("failed to delete WireMock mapping: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to delete WireMock mapping, status: %d, body: %s", response.StatusCode, string(body))
	}

	return nil
}

// VerifyWireMockRequest verifies request was matched
func VerifyWireMockRequest(wiremockURL string, criteria map[string]interface{}, expectedCount int) error {
	response, body, err := makeWireMockRequest("POST", wiremockURL+"/__admin/requests/count", criteria)
	if err != nil {
		return fmt.Errorf("failed to verify WireMock request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to verify WireMock request, status: %d, body: %s", response.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal WireMock verification response: %w", err)
	}

	count, ok := result["count"].(float64)
	if !ok {
		return fmt.Errorf("invalid count in WireMock verification response: %v", result["count"])
	}

	if int(count) != expectedCount {
		return fmt.Errorf("expected %d requests, but got %d", expectedCount, int(count))
	}

	return nil
}

func VerifyWireMockRequestCountForExpecation(wiremockURL, uuid string, expectedCount int) error {
	response, body, err := makeWireMockRequest("GET", wiremockURL+"/__admin/requests?matchingStub="+uuid, "")
	if err != nil {
		return fmt.Errorf("failed to verify WireMock request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to verify WireMock request, status: %d, body: %s", response.StatusCode, string(body))
	}

	type wiremockGetRequestsResponse struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}

	var result wiremockGetRequestsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal WireMock verification response: %w", err)
	}

	count := result.Meta.Total
	if int(expectedCount) != count {
		return fmt.Errorf("expected %d requests, but got %d", expectedCount, count)
	}

	return nil
}

// ResetWireMockServer clears all mappings and requests
func ResetWireMockServer(wiremockURL string) error {
	response, body, err := makeWireMockRequest("POST", wiremockURL+"/__admin/reset", nil)
	if err != nil {
		return fmt.Errorf("failed to reset WireMock: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to reset WireMock, status: %d, body: %s", response.StatusCode, string(body))
	}

	return nil
}

// makeWireMockRequest makes HTTP request to WireMock Admin API
func makeWireMockRequest(method, url string, body interface{}) (*http.Response, []byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return response, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return response, responseBody, nil
}

// StartWireMock starts the WireMock server for testing
func StartWireMock(mockServerURL string) {
	// First, stop and remove any existing WireMock container to ensure clean state
	fmt.Println("Stopping any existing mock server...")
	stopCmd := exec.Command("make", "-C", "../../../..", "wiremock-down")
	_ = stopCmd.Run() // Ignore errors if container doesn't exist

	// Give time for cleanup
	time.Sleep(1 * time.Second)

	// Start WireMock first, before starting the service
	fmt.Println("Starting mock server at " + mockServerURL + " ...")
	cmd := exec.Command("make", "-C", "../../../..", "wiremock-up")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error starting mock server:", err)
		fmt.Println("Command output:", string(output))
		panic(fmt.Sprintf("Failed to start WireMock: %v", err))
	}

	// Wait for WireMock to be fully ready with health check
	err = WaitForWireMockReady(mockServerURL, 30*time.Second)
	if err != nil {
		fmt.Println("Error: WireMock failed to become ready:", err)
		panic(fmt.Sprintf("Failed to start WireMock: %v", err))
	}

	fmt.Println("Mock server started successfully")

	// Verify mock server accessibility
	fmt.Println("Checking mock server accessibility at " + mockServerURL + " ...")
	ExpectServiceIsAccessible(mockServerURL)
}

// StopWireMock stops the WireMock server
func StopWireMock() {
	// Stop WireMock server
	fmt.Println("Stopping mock server...")
	cmd := exec.Command("make", "-C", "../../../..", "wiremock-down")
	_ = cmd.Run() // Ignore errors during cleanup
	fmt.Println("Mock server stopped")
}
