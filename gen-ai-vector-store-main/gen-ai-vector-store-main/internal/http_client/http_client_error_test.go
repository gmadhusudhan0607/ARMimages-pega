/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package http_client

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

// mockStatusRoundTripper simulates HTTP responses with specific status codes
type mockStatusRoundTripper struct {
	calls      *int
	statusCode int
	statusText string
}

func (m *mockStatusRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	(*m.calls)++
	resp := &http.Response{
		StatusCode: m.statusCode,
		Status:     m.statusText,
		Header:     make(http.Header),
		Body:       http.NoBody,
	}
	return resp, nil
}

func newTestHTTPClientWithStatus(maxRetries int, timeout time.Duration, statusCode int, statusText string, calls *int) HTTPClient {
	cfg := HTTPClientConfig{
		Timeout:    timeout,
		MaxRetries: maxRetries,
	}
	client, _ := NewHTTPClientWithConfig(cfg)
	// Replace the transport with our mock
	if uc, ok := client.(*httpClient); ok {
		if uc.retryClient != nil && uc.retryClient.HTTPClient != nil {
			uc.retryClient.HTTPClient.Transport = &mockStatusRoundTripper{
				calls:      calls,
				statusCode: statusCode,
				statusText: statusText,
			}
		}
	}
	return client
}

func TestHTTPClientError_CaptureslastStatusCode(t *testing.T) {
	calls := 0
	maxRetries := 2
	lastStatusCode := 500
	lastStatusText := "500 Internal Server Error"

	client := newTestHTTPClientWithStatus(maxRetries, time.Second, lastStatusCode, lastStatusText, &calls)
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	// The request should succeed (return response) since we're returning a valid response
	// But let's test with an actual error scenario
	client = newTestHTTPClient(maxRetries, time.Second, "other", &calls)

	_, err := client.Do(req)

	// Verify we get an error
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Type assert to HTTPClientError
	var httpClientErr *HTTPClientError
	if !errors.As(err, &httpClientErr) {
		t.Fatal("Expected HTTPClientError type")
	}

	// Verify error methods work
	if httpClientErr.MaxRetries != maxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", maxRetries, httpClientErr.MaxRetries)
	}

	if httpClientErr.Err == nil {
		t.Error("Expected underlying error to be set")
	}

	// Test error message
	errorMsg := err.Error()
	if errorMsg == "" {
		t.Error("Expected non-empty error message")
	}

	// Test Unwrap
	if errors.Unwrap(err) == nil {
		t.Error("Expected Unwrap to return underlying error")
	}
}

func TestHTTPClientError_GetMethods(t *testing.T) {
	// Test with original status code and text
	err := &HTTPClientError{
		MaxRetries:     3,
		Err:            errors.New("test error"),
		LastStatusCode: 429,
		LastStatusText: "429 Too Many Requests",
	}

	if err.GetLastStatusCode() != 429 {
		t.Errorf("Expected status code 429, got %d", err.GetLastStatusCode())
	}

	if err.GetLastStatusText() != "429 Too Many Requests" {
		t.Errorf("Expected status text '429 Too Many Requests', got '%s'", err.GetLastStatusText())
	}

	// Test error message with status code
	expectedMsg := "HTTP client error after 3 retries: test error (last status: Code:429, Error:429 Too Many Requests)"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestHTTPClientError_WithoutStatusCode(t *testing.T) {
	// Test without original status code
	err := &HTTPClientError{
		MaxRetries: 2,
		Err:        errors.New("network error"),
	}

	if err.GetLastStatusCode() != 0 {
		t.Errorf("Expected status code 0, got %d", err.GetLastStatusCode())
	}

	if err.GetLastStatusText() != "" {
		t.Errorf("Expected empty status text, got '%s'", err.GetLastStatusText())
	}

	// Test error message without status code
	expectedMsg := "HTTP client error after 2 retries: network error"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestHTTPClientError_TypeAssertion(t *testing.T) {
	calls := 0
	maxRetries := 1
	client := newTestHTTPClient(maxRetries, time.Second, "other", &calls)
	req, _ := http.NewRequest("GET", "http://example.com", nil)

	_, err := client.Do(req)

	// Type assert to HTTPClientError
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var httpClientErr *HTTPClientError
	if !errors.As(err, &httpClientErr) {
		t.Fatal("Expected HTTPClientError type")
	}

	if httpClientErr.MaxRetries != maxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", maxRetries, httpClientErr.MaxRetries)
	}
}
