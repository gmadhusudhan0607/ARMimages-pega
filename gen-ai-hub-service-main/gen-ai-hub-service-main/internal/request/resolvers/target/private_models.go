/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"gopkg.in/yaml.v3"
)

// loadPrivateModels loads private Azure OpenAI model configurations from the specified directory
// Private models are stored in separate YAML files with prefix "private-model-"
// Only active models are returned
func loadPrivateModels(ctx context.Context, privateModelDir string) (*api.Mapping, error) {
	if privateModelDir == "" {
		return &api.Mapping{Models: []api.Model{}}, nil
	}

	// Check if directory exists
	if _, err := os.Stat(privateModelDir); os.IsNotExist(err) {
		return &api.Mapping{Models: []api.Model{}}, nil
	}

	entries, err := os.ReadDir(privateModelDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read private model directory %s: %w", privateModelDir, err)
	}

	var allModels []api.Model

	// Process each file with "private-model-" prefix
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasPrefix(filename, "private-model-") {
			continue
		}

		// Only process YAML files
		ext := filepath.Ext(filename)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		filePath := filepath.Join(privateModelDir, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			// Log error but continue processing other files
			continue
		}

		var mapping api.Mapping
		if err := yaml.Unmarshal(data, &mapping); err != nil {
			// Log error but continue processing other files
			continue
		}

		// Only include active models
		for _, model := range mapping.Models {
			if model.Active {
				allModels = append(allModels, model)
			}
		}
	}

	return &api.Mapping{Models: allModels}, nil
}

// checkPrivateModels searches for a private Azure OpenAI model by name
// Results are cached for cacheTTL duration
// Returns the model, true if found (and active), otherwise returns nil, false
// This function is used during model resolution for Azure OpenAI routes
func (r *TargetResolver) checkPrivateModels(ctx context.Context, modelName string) (*api.Model, bool, error) {
	if r.privateModelDir == "" {
		return nil, false, nil
	}

	// Check cache first
	r.privateModelsMu.RLock()
	if time.Now().Before(r.privateModelsCacheExpiry) && r.privateModelsCache != nil {
		model, found := findModelInMapping(r.privateModelsCache, modelName)
		r.privateModelsMu.RUnlock()
		return model, found, nil
	}
	r.privateModelsMu.RUnlock()

	// Load from disk
	privateMapping, err := loadPrivateModels(ctx, r.privateModelDir)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load private models: %w", err)
	}

	// Update cache
	r.privateModelsMu.Lock()
	r.privateModelsCache = privateMapping
	r.privateModelsCacheExpiry = time.Now().Add(r.cacheTTL)
	r.privateModelsMu.Unlock()

	model, found := findModelInMapping(privateMapping, modelName)
	return model, found, nil
}
