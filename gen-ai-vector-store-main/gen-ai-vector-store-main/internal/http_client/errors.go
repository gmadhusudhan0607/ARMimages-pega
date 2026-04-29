// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package http_client

import (
	"fmt"
)

// HTTPClientError is returned when the HTTP client encounters an error, storing original response details
type HTTPClientError struct {
	MaxRetries     int
	Err            error
	LastStatusCode int
	LastStatusText string
}

func (e *HTTPClientError) Error() string {
	if e.LastStatusCode > 0 {
		return fmt.Sprintf("HTTP client error after %d retries: %s (last status: Code:%d, Error:%s)",
			e.MaxRetries, e.Err.Error(), e.LastStatusCode, e.LastStatusText)
	}
	return fmt.Sprintf("HTTP client error after %d retries: %v", e.MaxRetries, e.Err)
}

func (e *HTTPClientError) Unwrap() error {
	return e.Err
}

func (e *HTTPClientError) GetLastStatusCode() int {
	return e.LastStatusCode
}

func (e *HTTPClientError) GetLastStatusText() string {
	return e.LastStatusText
}

// shouldRetry determines if a request should be retried based on the HTTP status code
func shouldRetry(statusCode int) bool {
	// Retry on 5xx server errors and some 4xx client errors
	switch statusCode {
	case 408, 409, 429: // Request Timeout, Conflict, Too Many Requests
		return true
	case 500, 502, 503, 504: // Internal Server Error, Bad Gateway, Service Unavailable, Gateway Timeout
		return true
	default:
		return false
	}
}
