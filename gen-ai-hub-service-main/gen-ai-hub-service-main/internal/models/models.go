/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package models

import (
	"context"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/loader"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/registry"
)

var (
	globalRegistry *registry.Registry
	registryOnce   sync.Once
	registryError  error
)

// ResetGlobalRegistryForTest resets the global registry for testing purposes
// This function should only be used in tests
func ResetGlobalRegistryForTest(reg *registry.Registry) {
	// Set the global registry directly
	globalRegistry = reg
	// Mark sync.Once as already done to prevent re-initialization
	if reg != nil {
		registryOnce.Do(func() {
			// This empty Do() marks the Once as done
		})
	} else {
		// Reset sync.Once to allow re-initialization with nil
		registryOnce = sync.Once{}
	}
	registryError = nil
}

// GetGlobalRegistry returns the global registry instance
func GetGlobalRegistry(ctx context.Context) (*registry.Registry, error) {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	registryOnce.Do(func() {
		start := time.Now()
		logger.Debug("Initializing global model registry")

		l := loader.NewModelLoader()
		globalRegistry, registryError = l.LoadModelsIntoRegistry(ctx)

		if registryError != nil {
			logger.Errorf("Failed to load model registry: %v", registryError)
		} else {
			modelCount := len(globalRegistry.GetAllModels())
			logger.Debugf("Global model registry initialized - %d models loaded in %v",
				modelCount, time.Since(start))
		}
	})

	return globalRegistry, registryError
}
