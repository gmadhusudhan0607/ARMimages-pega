// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package http_client

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	gcpsecretmanager "cloud.google.com/go/secretmanager/apiv1beta2"
	awssecret "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/aws"
	gcpsecret "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/gcp"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/go-sax/retryablehttpsax"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/hashicorp/go-retryablehttp"
)

// For SAX client we use singleton to reduce the number of calls to Okta and AWS (to SAX_CLIENT_SECRET)
var saxClientOnce sync.Once
var singleHTTPClientSax HTTPClient
var saxClientInitErr error

// Allow overriding secret retrieval for tests
var getAWSSaxClientPrivateKeyFunc = getAWSSaxClientPrivateKey
var getGCPSaxClientPrivateKeyFunc = getGCPSaxClientPrivateKey

func scopesStrToList(scopes string) (scopesList []string) {
	return append(scopesList, strings.Split(scopes, " ")...)
}

// initSaxClientSingleton initializes the SAX client singleton instance
func initSaxClientSingleton(config HTTPClientConfig, provider string) (HTTPClient, error) {
	saxClientOnce.Do(func() {
		var pKey []byte
		var err error
		switch strings.ToLower(provider) {
		case "aws":
			pKey, err = getAWSSaxClientPrivateKeyFunc(context.Background(), helpers.GetEnvOrPanic("SAX_CLIENT_SECRET"))
		case "gcp":
			pKey, err = getGCPSaxClientPrivateKeyFunc(context.Background(), helpers.GetEnvOrPanic("SAX_CLIENT_SECRET"))
		default:
			saxClientInitErr = fmt.Errorf("initSaxClientSingleton(): unknown provider: %s", provider)
			return
		}
		if err != nil {
			saxClientInitErr = fmt.Errorf("failed to get SAX private key: %w", err)
			return
		}

		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = config.MaxRetries
		retryClient.HTTPClient.Timeout = config.Timeout
		retryClient.HTTPClient.Transport = http.DefaultTransport
		retryClient.Logger = nil

		// Configure retry policy with logging (same as non-SAX client)
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

		saxClient := retryablehttpsax.NewClient(retryClient)
		saxClient.ClientID = helpers.GetEnvOrPanic("SAX_CLIENT_ID")
		saxClient.PrivateKey = pKey
		saxClient.Scopes = scopesStrToList(helpers.GetEnvOrPanic("SAX_CLIENT_SCOPES"))
		saxClient.TokenEndpoint = helpers.GetEnvOrPanic("SAX_CLIENT_TOKEN_ENDPOINT")

		singleHTTPClientSax = &httpClient{
			retryClient: retryClient,
			doer:        saxClient,
			isSaxClient: true,
			config:      config,
		}
	})
	if saxClientInitErr != nil {
		return nil, saxClientInitErr
	}
	return singleHTTPClientSax, nil
}

func getAWSSaxClientPrivateKey(ctx context.Context, secretArn string) (value []byte, err error) {
	region := helpers.GetEnvOrPanic("REGION")
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return value, fmt.Errorf("initialize AWS config: %w", err)
	}

	b64Str, err := awssecret.GetSecretAsString(ctx, httpLog, secretsmanager.NewFromConfig(cfg), secretArn)
	if err != nil {
		return value, fmt.Errorf("failed to read secret '%s' : %s", secretArn, err)
	}

	pk, err := b64.URLEncoding.DecodeString(b64Str)
	if err != nil {
		return value, fmt.Errorf("failed to decode SAX private key: %w", err)
	}

	return pk, nil
}

func getGCPSaxClientPrivateKey(ctx context.Context, secretName string) (value []byte, err error) {
	client, err := gcpsecretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			httpLog.Warn("Failed to close secret manager client", zap.Error(closeErr))
		}
	}()

	b64Str, err := gcpsecret.GetSaxCredentials(httpLog)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret '%s': %s", secretName, err)
	}

	pk, err := b64.URLEncoding.DecodeString(b64Str)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SAX private key: %w", err)
	}

	return pk, nil
}
