/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

type DefaultModelConfig struct {
	Fast  string `json:"fast"`
	Smart string `json:"smart"`
	Pro   string `json:"pro"`
}

// DefaultModelResponse is used for API responses with optional Pro field for now
type DefaultModelResponse struct {
	Fast  string `json:"fast"`
	Smart string `json:"smart"`
	Pro   string `json:"pro,omitempty"`
}

// ToResponse converts config to response based on feature flag
func (c DefaultModelConfig) ToResponse(includeProModel bool) DefaultModelResponse {
	r := DefaultModelResponse{
		Fast:  c.Fast,
		Smart: c.Smart,
	}
	if includeProModel {
		r.Pro = c.Pro // Will be omitted if empty due to omitempty tag
	}
	return r
}

func GetDefaultModelsForContext(ctx context.Context) (DefaultModelConfig, error) {
	l := cntx.LoggerFromContext(ctx).Sugar()
	defer l.Sync() //nolint:errcheck

	var defaults DefaultModelConfig

	url := os.Getenv("MODELS_DEFAULTS_ENDPOINT")
	if url == "" {
		return defaults, fmt.Errorf("MODELS_DEFAULTS_ENDPOINT environment variable is not set")
	}
	l.Infof("The default fast, smart and pro models mapping endpoint url is: %s", url)

	// Make the GET request
	resp, err := http.Get(url)
	if err != nil {
		return defaults, fmt.Errorf("error occurred when getting the default models secret content from the /models/default endpoint exposed by the genai-ops-gateway svc %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return defaults, fmt.Errorf("received non-200 response from %s: %d", url, resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return defaults, fmt.Errorf("error occurred when getting the default models secret content from the /models/default endpoint exposed by the genai-ops-gateway svc %s: %w", url, err)
	}
	l.Debugf("MODELS_DEFAULTS_ENDPOINT Response body:\n %s", string(body))

	// First, try to parse as the new format with full model objects

	if err := json.Unmarshal(body, &defaults); err != nil {
		return defaults, fmt.Errorf("error while unmarshalling the json response body as map: %w", err)
	}

	l.Debugf("Final extracted default models: Fast=%s, Smart=%s, Pro=%s", defaults.Fast, defaults.Smart, defaults.Pro)

	return defaults, nil
}
