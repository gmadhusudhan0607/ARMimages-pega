/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

// doConverseForIntegrationTest performs a simple HTTP call without AWS authentication
// This is only used for integration tests where IS_INTEGRATION_TEST=true
func doConverseForIntegrationTest(modelCall *ConverseModelInference, awsProvider AwsProvider) error {
	ctx := modelCall.Ctx
	l := cntx.LoggerFromContext(ctx).Sugar()
	defer l.Sync() //nolint:errcheck

	endpoint := modelCall.InfraModel.Endpoint
	path := modelCall.InfraModel.Path
	targetURL := fmt.Sprintf("%s%s", endpoint, path)

	l.Infof("Integration test mode: sending request to [%s]", targetURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(modelCall.RawInput))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type header
	req.Header.Set("Content-Type", "application/json")

	// Forward relevant headers from the original request
	if isolationID := modelCall.GinContext.GetHeader("X-Genai-Gateway-Isolation-Id"); isolationID != "" {
		req.Header.Set("X-Genai-Gateway-Isolation-ID", isolationID)
	}
	if auth := modelCall.GinContext.GetHeader("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	// Execute request using the AWS provider's HTTP client
	client := awsProvider.GetAwsClient(targetURL)
	resp, err := client.Do(req)
	if err != nil {
		l.Errorf("Error executing request: %v", err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Copy response status and headers
	modelCall.GinContext.Status(resp.StatusCode)
	for name, values := range resp.Header {
		for _, value := range values {
			modelCall.GinContext.Header(name, value)
		}
	}

	// Stream response body
	if _, err = io.Copy(modelCall.GinContext.Writer, resp.Body); err != nil {
		return fmt.Errorf("error during copy of response body: %w", err)
	}

	l.Infof("Received response from: %s (status: %d)", targetURL, resp.StatusCode)
	return nil
}
