// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package http_client

import (
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
)

const (
	// Default configuration values
	defaultTimeout    = 5 * time.Minute
	defaultMaxRetries = 3
)

// HTTPClientConfig holds configuration for an HTTP client
type HTTPClientConfig struct {
	Timeout    time.Duration
	MaxRetries int
}

// getHTTPClientConfig reads configuration from environment variables or uses defaults
func getHTTPClientConfig() HTTPClientConfig {
	cfg := HTTPClientConfig{
		Timeout:    defaultTimeout,
		MaxRetries: defaultMaxRetries,
	}

	// Read timeout from environment
	if timeoutStr := helpers.GetEnvOrDefault("HTTP_CLIENT_TIMEOUT", ""); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Timeout = timeout
		} else {
			httpLog.Warn("Invalid HTTP_CLIENT_TIMEOUT value, using default",
				zap.String("value", timeoutStr),
				zap.Duration("default", defaultTimeout))
		}
	}

	// Read max retries from environment
	if retriesStr := helpers.GetEnvOrDefault("HTTP_CLIENT_MAX_RETRIES", ""); retriesStr != "" {
		if retries, err := strconv.Atoi(retriesStr); err == nil && retries >= 0 {
			cfg.MaxRetries = retries
		} else {
			httpLog.Warn("Invalid HTTP_CLIENT_MAX_RETRIES value, using default",
				zap.String("value", retriesStr),
				zap.Int("default", defaultMaxRetries))
		}
	}

	return cfg
}

// GetDefaultHTTPClientConfig returns the default HTTP client config (timeout 5m, retries 3)
func GetDefaultHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:    defaultTimeout,
		MaxRetries: defaultMaxRetries,
	}
}
