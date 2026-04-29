/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package loader

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/registry"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/specs"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/validation"
)

// ModelLoader provides direct YAML loading from embedded filesystem
type ModelLoader struct {
	embedFS   fs.FS
	validator *validation.ModelValidator
}

// NewModelLoader creates a new model loader
func NewModelLoader() *ModelLoader {
	return &ModelLoader{
		embedFS:   specs.ModelsSpecs,
		validator: validation.NewModelValidator(),
	}
}

// LoadModelsIntoRegistry loads all models into a registry
func (ml *ModelLoader) LoadModelsIntoRegistry(ctx context.Context) (*registry.Registry, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	start := time.Now()
	logger.Info("Loading model registry from embedded filesystem")

	reg := registry.NewRegistry()

	// Load models from each infrastructure
	infrastructures := []types.Infrastructure{
		types.InfrastructureAWS,
		types.InfrastructureGCP,
		types.InfrastructureAzure,
	}

	totalModels := 0
	for _, infra := range infrastructures {
		infraStart := time.Now()
		logger.Debugf("Loading models for infrastructure: %s", infra)

		modelCount := len(reg.GetAllModels())
		if err := ml.loadInfrastructureModels(ctx, reg, infra); err != nil {
			return nil, fmt.Errorf("failed to load %s models: %w", infra, err)
		}

		newModelCount := len(reg.GetAllModels()) - modelCount
		totalModels += newModelCount
		logger.Debugf("Loaded %d models for infrastructure %s in %v",
			newModelCount, infra, time.Since(infraStart))
	}

	// Build indexes for performance
	logger.Debug("Building model registry indexes")
	if err := reg.RebuildIndexes(); err != nil {
		return nil, fmt.Errorf("failed to build indexes: %w", err)
	}

	logger.Infof("Model registry loaded successfully - %d models in %v",
		totalModels, time.Since(start))
	return reg, nil
}

// loadInfrastructureModels loads models for a specific infrastructure
func (ml *ModelLoader) loadInfrastructureModels(ctx context.Context, reg *registry.Registry, infra types.Infrastructure) error {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	basePath := strings.ToLower(string(infra))

	fileCount := 0
	return fs.WalkDir(ml.embedFS, basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		fileCount++
		logger.Debugf("Loading model file: %s", path)
		if err := ml.loadModelFile(ctx, reg, path, infra); err != nil {
			return err
		}

		return nil
	})
}

// loadModelFile loads a single YAML model file
func (ml *ModelLoader) loadModelFile(ctx context.Context, reg *registry.Registry, path string, infra types.Infrastructure) error {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	data, err := fs.ReadFile(ml.embedFS, path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	var group config.ModelGroup
	if err := yaml.Unmarshal(data, &group); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Use the new comprehensive validator
	if err := ml.validator.ValidateModelGroup(&group, path); err != nil {
		return fmt.Errorf("validation failed for %s: %w", path, err)
	}

	// Register all models in the group
	modelCount := 0
	for _, modelConfig := range group.Models {
		model := modelConfig.ToModel(group.Infrastructure, group.Provider, group.Creator)
		model.SourceFile = path // Track which file this model came from
		if err := reg.RegisterModel(model); err != nil {
			return fmt.Errorf("failed to register model %s: %w", model.Name, err)
		}
		modelCount++
	}

	logger.Debugf("Loaded %d models from file %s", modelCount, path)
	return nil
}
