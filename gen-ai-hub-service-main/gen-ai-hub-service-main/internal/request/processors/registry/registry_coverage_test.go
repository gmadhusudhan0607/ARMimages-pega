/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package registry

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/loader"
)

// TestAllModelsHaveProcessors validates that every model in the specs
// has a corresponding processor registered in the registry
func TestAllModelsHaveProcessors(t *testing.T) {
	// Load all models from embedded specs
	modelLoader := loader.NewModelLoader()
	modelRegistry, err := modelLoader.LoadModelsIntoRegistry(context.Background())
	require.NoError(t, err, "Failed to load model registry")

	// Initialize processor registry
	InitializeProcessorRegistry()
	processorRegistry := GetGlobalRegistry()

	// Track missing processors
	var missingProcessors []string
	var checkedCombinations []string

	// Check each model has a corresponding processor
	for _, model := range modelRegistry.GetAllModels() {
		// Create processor key for this model
		key := CreateProcessorKey(model)

		combination := fmt.Sprintf("%s/%s/%s/%s",
			model.Provider, model.Infrastructure, model.Creator, model.Version)
		checkedCombinations = append(checkedCombinations, combination)

		if !processorRegistry.HasProcessor(key) {
			missingProcessors = append(missingProcessors,
				fmt.Sprintf("  - Model: %s (Version: %s)\n    Key: %s\n    File: %s",
					model.Name, model.Version, key.String(), model.SourceFile))
		}
	}

	// Report results
	t.Logf("Checked %d model combinations", len(checkedCombinations))

	if len(missingProcessors) > 0 {
		t.Errorf("❌ MISSING PROCESSOR REGISTRATIONS\n\n"+
			"The following %d models do not have registered processors:\n\n%s\n\n"+
			"🔧 TO FIX:\n"+
			"1. Open internal/request/processors/registry/registry.go\n"+
			"2. Add processor registration(s) in registerAllProcessors() function\n"+
			"3. Use the ProcessorKey format shown above\n"+
			"4. Ensure the processor extension exists in internal/request/processors/extensions/\n\n"+
			"📖 Example registration:\n"+
			"  _ = registry.Register(ProcessorKey{\n"+
			"    Provider:       \"provider\",\n"+
			"    Infrastructure: \"infrastructure\",\n"+
			"    Creator:        \"creator\",\n"+
			"    Version:        \"version\",\n"+
			"  }, func() interface{} {\n"+
			"    return extensions.NewYourExtension()\n"+
			"  })",
			len(missingProcessors),
			strings.Join(missingProcessors, "\n\n"))
	} else {
		t.Logf("✅ All %d models have registered processors", len(checkedCombinations))
	}
}

// TestNoOrphanedProcessors warns about processors that don't match any models
func TestNoOrphanedProcessors(t *testing.T) {
	// Load all models
	modelLoader := loader.NewModelLoader()
	modelRegistry, err := modelLoader.LoadModelsIntoRegistry(context.Background())
	require.NoError(t, err)

	// Get all registered processors
	InitializeProcessorRegistry()
	processorRegistry := GetGlobalRegistry()
	registeredKeys := processorRegistry.GetRegisteredKeys()

	var orphanedProcessors []string

	for _, key := range registeredKeys {
		found := false
		for _, model := range modelRegistry.GetAllModels() {
			modelKey := CreateProcessorKey(model)
			if modelKey == key {
				found = true
				break
			}
		}

		if !found {
			orphanedProcessors = append(orphanedProcessors,
				fmt.Sprintf("  - %s", key.String()))
		}
	}

	if len(orphanedProcessors) > 0 {
		t.Logf("⚠️  WARNING: Found %d registered processors without corresponding models:\n\n%s\n\n"+
			"This might indicate:\n"+
			"- Legacy processors that can be removed\n"+
			"- Processors for deprecated models\n"+
			"- Test/development processors\n"+
			"- API version mappings (multiple versions -> same processor)",
			len(orphanedProcessors),
			strings.Join(orphanedProcessors, "\n"))
	} else {
		t.Logf("✅ No orphaned processors found")
	}
}

// TestProcessorRegistryIntegrity performs additional validation checks
func TestProcessorRegistryIntegrity(t *testing.T) {
	InitializeProcessorRegistry()
	processorRegistry := GetGlobalRegistry()

	// Test that we can create instances of all registered processors
	registeredKeys := processorRegistry.GetRegisteredKeys()

	for _, key := range registeredKeys {
		t.Run(fmt.Sprintf("CreateProcessor_%s", key.String()), func(t *testing.T) {
			processor, err := processorRegistry.CreateProcessor(key)
			require.NoError(t, err, "Failed to create processor for key: %s", key.String())
			require.NotNil(t, processor, "Processor instance is nil for key: %s", key.String())
		})
	}

	t.Logf("✅ Successfully created instances for all %d registered processors", len(registeredKeys))
}
