/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	api "github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/proxy"
)

// gpt4oModelVersion holds the cached version fetched from Azure API
var gpt4oModelVersion string
var gpt4oVersionMutex sync.RWMutex
var gpt4oInitialized bool = false

// AzureModelsResponse represents the full response from Azure's /openai/models API
type AzureModelsResponse struct {
	Models []api.AzureModelResponse `json:"models"`
}

// InitGPT4oVersion fetches the GPT-4o model version from Azure API
// and caches it for later use. If the fetch fails, it logs a warning and continues
// without failing. This function now accepts request headers to include JWT tokens.
func InitGPT4oVersion(ctx context.Context, genaiURL string, headers http.Header) error {
	l := cntx.LoggerFromContext(ctx).Sugar()
	l.Debug("Initializing GPT-4o version from Azure API")

	if genaiURL == "" {
		l.Warn("GENAI_URL is empty, skipping GPT-4o version initialization")
		return nil
	}

	// Construct the models endpoint URL
	modelsURL := fmt.Sprintf("%s/openai/models", genaiURL)
	l.Debugf("Fetching GPT-4o version from: %s", modelsURL)

	// Make the API call using the proxy client
	client := proxy.NewClient(modelsURL)

	// Create a new request
	req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for GPT-4o version: %w", err)
	}

	// Copy headers from the original request (includes JWT token)
	if headers != nil {
		for key, values := range headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		l.Debugf("Copied %d headers to GPT-4o version request", len(headers))
	}

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch GPT-4o version from Azure API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error %d Azure API when fetching GPT-4o version: %w", resp.StatusCode, err)
	}

	// Parse the JSON response
	var modelsResp AzureModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return fmt.Errorf("failed to parse Azure models response:: %w", err)
	}

	// Find the gpt-4o model with both deployment-id and model-name matching "gpt-4o"
	version := findGPT4oVersion(modelsResp.Models)
	if version == "" {
		return fmt.Errorf("gpt-4o model not found in Azure API response: %w", err)
	}

	// Cache the version
	gpt4oVersionMutex.Lock()
	gpt4oModelVersion = version
	gpt4oVersionMutex.Unlock()

	l.Debug("Successfully fetched GPT-4o version: %s", version)
	return nil
}

// findGPT4oVersion searches for a model with both deployment-id and model-name equal to "gpt-4o"
// and returns its model-version. Returns empty string if not found.
func findGPT4oVersion(models []api.AzureModelResponse) string {
	for _, model := range models {
		if model.DeploymentID == "gpt-4o" && model.ModelName == "gpt-4o" {
			return model.ModelVersion
		}
	}
	return ""
}

// GetGPT4oVersion returns the cached GPT-4o version.
// Returns empty string if the version was not successfully fetched.
func GetGPT4oVersion() string {
	gpt4oVersionMutex.RLock()
	defer gpt4oVersionMutex.RUnlock()
	return gpt4oModelVersion
}

// LazyInitGPT4oVersion performs lazy initialization of GPT-4o version on first use.
// It uses the provided context which should contain request headers with JWT token.
// This function is thread-safe and will retry on subsequent requests if initialization fails.
func LazyInitGPT4oVersion(ctx context.Context, genaiURL string) {
	// Check if already successfully initialized
	if gpt4oInitialized {
		return
	}

	l := cntx.LoggerFromContext(ctx).Sugar()
	l.Debug("Performing lazy initialization of GPT-4o version")

	// Extract headers from gin.Context if available
	var headers http.Header
	if ginCtx := cntx.GetGinContext(ctx); ginCtx != nil {
		headers = ginCtx.Request.Header.Clone()
		l.Debugf("Using request headers from gin context for GPT-4o version fetch")
	} else {
		l.Warn("No gin context available for GPT-4o version fetch, proceeding without request headers")
		headers = http.Header{}
	}

	if err := InitGPT4oVersion(ctx, genaiURL, headers); err == nil {
		gpt4oVersionMutex.Lock()
		gpt4oInitialized = true
		gpt4oVersionMutex.Unlock()
	}
}
