/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
)

// MappingClient handles communication with the MAPPING_ENDPOINT
// for fetching dynamic AWS Bedrock model configurations
type MappingClient struct {
	endpoint    string
	httpClient  *http.Client
	cache       []infra.ModelConfig
	cacheTTL    time.Duration
	cacheExpiry time.Time
	mu          sync.RWMutex
}

// NewMappingClient creates a new MappingClient
func NewMappingClient(endpoint string) *MappingClient {
	return &MappingClient{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheTTL:   5 * time.Minute,
	}
}

// GetModels fetches model configurations from the mapping endpoint
// Results are cached for cacheTTL duration
func (c *MappingClient) GetModels(ctx context.Context) ([]infra.ModelConfig, error) {
	// Check cache first
	c.mu.RLock()
	if time.Now().Before(c.cacheExpiry) && len(c.cache) > 0 {
		models := c.cache
		c.mu.RUnlock()
		return models, nil
	}
	c.mu.RUnlock()

	// Fetch from endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from mapping endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mapping endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var models []infra.ModelConfig
	if err := json.Unmarshal(body, &models); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models: %w", err)
	}

	// Update cache
	c.mu.Lock()
	c.cache = models
	c.cacheExpiry = time.Now().Add(c.cacheTTL)
	c.mu.Unlock()

	return models, nil
}

// DefaultsClient handles communication with the MODELS_DEFAULTS_ENDPOINT
// for fetching default fast/smart model configurations
type DefaultsClient struct {
	endpoint    string
	httpClient  *http.Client
	cache       *DefaultModelConfig
	cacheTTL    time.Duration
	cacheExpiry time.Time
	mu          sync.RWMutex
}

// NewDefaultsClient creates a new DefaultsClient
func NewDefaultsClient(endpoint string) *DefaultsClient {
	return &DefaultsClient{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheTTL:   5 * time.Minute,
	}
}

// DefaultModelConfig represents the response from the defaults endpoint
type DefaultModelConfig struct {
	Fast  *ModelDefault `json:"fast,omitempty"`
	Smart *ModelDefault `json:"smart,omitempty"`
}

// ModelDefault represents a default model configuration
type ModelDefault struct {
	ModelID  string `json:"model_id"`
	Provider string `json:"provider"`
	Creator  string `json:"creator"`
}

// GetDefaults fetches default model configurations from the defaults endpoint
// Results are cached for cacheTTL duration
// Environment variable overrides (SMART_MODEL_OVERRIDE, FAST_MODEL_OVERRIDE) are handled elsewhere
func (c *DefaultsClient) GetDefaults(ctx context.Context) (*DefaultModelConfig, error) {
	// Check cache first
	c.mu.RLock()
	if time.Now().Before(c.cacheExpiry) && c.cache != nil {
		defaults := c.cache
		c.mu.RUnlock()
		return defaults, nil
	}
	c.mu.RUnlock()

	// Fetch from endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from defaults endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("defaults endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var defaults DefaultModelConfig
	if err := json.Unmarshal(body, &defaults); err != nil {
		return nil, fmt.Errorf("failed to unmarshal defaults: %w", err)
	}

	// Update cache
	c.mu.Lock()
	c.cache = &defaults
	c.cacheExpiry = time.Now().Add(c.cacheTTL)
	c.mu.Unlock()

	return &defaults, nil
}
