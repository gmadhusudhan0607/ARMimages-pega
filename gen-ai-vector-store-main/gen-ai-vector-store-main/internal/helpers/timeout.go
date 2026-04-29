// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package helpers

import (
	"time"
)

const (
	// DefaultHTTPRequestTimeout is the default timeout for HTTP requests
	DefaultHTTPRequestTimeout = 25 * time.Second
	// DefaultHTTPRequestBackgroundTimeout is the default timeout for document processing in background after async request is accepted
	DefaultHTTPRequestBackgroundTimeout = 60 * time.Second
)

var (
	// httpRequestTimeout is the configured timeout for HTTP requests, initialized once
	httpRequestTimeout time.Duration
	// httpRequestBackgroundTimeout is the configured timeout for async document processing, initialized once
	httpRequestBackgroundTimeout time.Duration
)

func init() {
	// Initialize request timeout
	timeoutStr := GetEnvOrDefault("HTTP_REQUEST_TIMEOUT", "25s")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil || timeout <= 0 {
		httpRequestTimeout = DefaultHTTPRequestTimeout
	} else {
		httpRequestTimeout = timeout
	}

	// Initialize async processing timeout (background processing timeout)
	backgroundTimeoutStr := GetEnvOrDefault("HTTP_REQUEST_BACKGROUND_TIMEOUT", "60s")
	backgroundTimeout, err := time.ParseDuration(backgroundTimeoutStr)
	if err != nil || backgroundTimeout <= 0 {
		httpRequestBackgroundTimeout = DefaultHTTPRequestBackgroundTimeout
	} else {
		httpRequestBackgroundTimeout = backgroundTimeout
	}
}

// GetRequestTimeout returns the configured timeout for HTTP requests
func GetRequestTimeout() time.Duration {
	return httpRequestTimeout
}

// GetAsyncProcessingTimeout returns the configured timeout for async document processing
func GetAsyncProcessingTimeout() time.Duration {
	return httpRequestBackgroundTimeout
}
