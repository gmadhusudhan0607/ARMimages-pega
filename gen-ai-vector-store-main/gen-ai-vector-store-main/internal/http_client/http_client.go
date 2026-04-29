// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package http_client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	serviceName = "genai-vector-store"
)

var logger = log.GetNamedLogger(serviceName)
var httpLog = log.GetNamedLogger("http-client")

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPDoer interface for clients that can execute HTTP requests
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// httpClient wraps retryablehttp.Client
type httpClient struct {
	retryClient *retryablehttp.Client
	doer        HTTPDoer // Unified interface for both SAX and non-SAX clients
	isSaxClient bool
	config      HTTPClientConfig // Store config for behavior customization
}

func NewHTTPClient() (HTTPClient, error) {
	provider := helpers.GetEnvOrDefault("CLOUD_PROVIDER", "aws")
	cfg := getHTTPClientConfig()
	return newHTTPClient(cfg, !helpers.IsSaxClientDisabled(), provider)
}

// NewHTTPClientWithConfig creates an HTTP client with custom configuration
func NewHTTPClientWithConfig(cfg HTTPClientConfig) (HTTPClient, error) {
	provider := helpers.GetEnvOrDefault("CLOUD_PROVIDER", "aws")
	return newHTTPClient(cfg, !helpers.IsSaxClientDisabled(), provider)
}

// newHTTPClient creates a HTTP client that works for both SAX and non-SAX scenarios
func newHTTPClient(config HTTPClientConfig, isSax bool, provider string) (HTTPClient, error) {
	if isSax {
		return initSaxClientSingleton(config, provider)
	}

	// Create base retryable HTTP client with common configuration
	// Uses Go's default transport which is optimized for general use cases
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = config.MaxRetries
	retryClient.HTTPClient.Timeout = config.Timeout
	retryClient.HTTPClient.Transport = http.DefaultTransport
	retryClient.Logger = nil // Disable retryablehttp's default logging to avoid noise

	// Configure retry policy with logging
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		shouldRetry := false

		// Default retryablehttp retry logic for network errors
		if err != nil {
			shouldRetry = true
		} else if resp != nil {
			// Custom retry logic based on status codes
			shouldRetry = shouldRetryStatusCode(resp.StatusCode)
		}

		// Log retry attempts
		if shouldRetry {
			logFields := []zap.Field{
				zap.Int("maxRetries", config.MaxRetries),
			}
			if err != nil {
				logFields = append(logFields, zap.Error(err))
			}
			if resp != nil {
				logFields = append(logFields,
					zap.Int("statusCode", resp.StatusCode),
					zap.String("status", resp.Status))
			}
			httpLog.Info("HTTP request failed, retrying", logFields...)

			// Update metrics
			if ctx != nil {
				servicemetrics.EmbeddingMeasurementFromContext(ctx).IncreaseRetries()
			}
		}

		return shouldRetry, nil
	}

	// Configure backoff strategy to match existing exponential backoff
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		return time.Duration(attemptNum+1) * 100 * time.Millisecond
	}

	client := &httpClient{
		retryClient: retryClient,
		doer:        nil, // Will use retryClient directly
		isSaxClient: isSax,
		config:      config,
	}

	return client, nil
}

func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	// 1. Check context deadline upfront (fail fast)
	if err := c.checkDeadline(req.Context()); err != nil {
		return nil, err
	}

	// 2. Set authentication headers
	c.setAuthHeaders(req)

	// 3. Execute request through unified client interface
	resp, err := c.executeRequest(req)

	// 4. Process response (extract headers, set retry count)
	if resp != nil {
		c.processResponse(req.Context(), resp)
	}

	// 5. Wrap error if needed
	if err != nil {
		return nil, c.wrapError(err, resp)
	}

	return resp, nil
}

// checkDeadline checks if the request context deadline has already passed
func (c *httpClient) checkDeadline(ctx context.Context) error {
	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) <= 0 {
			return &HTTPClientError{
				MaxRetries:     0,
				Err:            context.DeadlineExceeded,
				LastStatusCode: http.StatusGatewayTimeout,
				LastStatusText: "Request deadline already exceeded",
			}
		}
	}
	return nil
}

// setAuthHeaders sets authentication headers for non-SAX clients
func (c *httpClient) setAuthHeaders(req *http.Request) {
	if c.isSaxClient {
		return // SAX client handles auth internally
	}

	if devToken := helpers.GetEnvOrDefault("SAX_CLIENT_DEV_TOKEN", ""); devToken != "" {
		logger.Info("Using SAX_CLIENT_DEV_TOKEN for authorization")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", devToken))
	}
	if apiKey := helpers.GetEnvOrDefault("GEN_AI_API_KEY", ""); apiKey != "" {
		req.Header.Set("api-key", apiKey)
	}
}

// executeRequest executes the HTTP request through the unified client interface
func (c *httpClient) executeRequest(req *http.Request) (*http.Response, error) {
	// Convert to retryable request if not already using SAX client
	if !c.isSaxClient {
		retryReq, err := retryablehttp.FromRequest(req)
		if err != nil {
			return nil, fmt.Errorf("failed to convert request: %w", err)
		}
		return c.retryClient.Do(retryReq)
	}

	// For SAX client, use the doer interface directly
	return c.doer.Do(req)
}

// processResponse extracts headers and updates metrics from the response
func (c *httpClient) processResponse(ctx context.Context, resp *http.Response) {
	if resp == nil || ctx == nil {
		return
	}

	// Extract gateway headers to service metrics
	metrics := servicemetrics.FromContext(ctx)
	if metrics != nil {
		metrics.GatewayMetrics.SetGenaiHeadersFromResponse(resp)
	}
}

// wrapError wraps an error with additional context
func (c *httpClient) wrapError(err error, resp *http.Response) error {
	statusCode := 0
	statusText := ""
	if resp != nil {
		statusCode = resp.StatusCode
		statusText = resp.Status
	}

	return &HTTPClientError{
		MaxRetries:     c.config.MaxRetries,
		Err:            err,
		LastStatusCode: statusCode,
		LastStatusText: statusText,
	}
}

// shouldRetryStatusCode determines if a status code should trigger a retry
func shouldRetryStatusCode(statusCode int) bool {
	return shouldRetry(statusCode)
}
