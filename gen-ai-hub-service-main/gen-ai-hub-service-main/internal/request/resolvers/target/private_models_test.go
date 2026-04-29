/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Private Model Tests

func TestCheckPrivateModels(t *testing.T) {
	// Create temporary directory for private models
	tempDir := t.TempDir()

	// Create a private model file
	privateModel := api.Mapping{
		Models: []api.Model{
			{
				Name:           "private-gpt-4o",
				Infrastructure: "azure",
				Provider:       "azure",
				Creator:        "openai",
				ModelId:        "private-gpt-4o-deployment",
				RedirectURL:    "https://private-azure.openai.azure.com",
				Active:         true,
			},
		},
	}

	data, err := json.Marshal(privateModel)
	require.NoError(t, err)

	// Write YAML file (simplified for test - using JSON)
	privateFile := tempDir + "/private-model-test.yaml"
	err = os.WriteFile(privateFile, data, 0644)
	require.NoError(t, err)

	resolver := &TargetResolver{
		privateModelDir: tempDir,
	}

	// Test finding private model
	model, found, err := resolver.checkPrivateModels(context.Background(), "private-gpt-4o")
	require.NoError(t, err)
	assert.True(t, found)
	assert.NotNil(t, model)
	assert.Equal(t, "private-gpt-4o", model.Name)

	// Test model not found
	model, found, err = resolver.checkPrivateModels(context.Background(), "nonexistent")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, model)
}

func TestLoadPrivateModels(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple private model files
	models := []api.Mapping{
		{
			Models: []api.Model{
				{
					Name:    "private-model-1",
					Active:  true,
					ModelId: "model-1",
				},
			},
		},
		{
			Models: []api.Model{
				{
					Name:    "private-model-2",
					Active:  true,
					ModelId: "model-2",
				},
				{
					Name:    "inactive-model",
					Active:  false,
					ModelId: "inactive",
				},
			},
		},
	}

	for i, m := range models {
		data, err := json.Marshal(m)
		require.NoError(t, err)
		filename := tempDir + "/private-model-" + string(rune('a'+i)) + ".yaml"
		err = os.WriteFile(filename, data, 0644)
		require.NoError(t, err)
	}

	// Load private models
	mapping, err := loadPrivateModels(context.Background(), tempDir)
	require.NoError(t, err)
	require.NotNil(t, mapping)

	// Should only load active models
	assert.Len(t, mapping.Models, 2)
	names := []string{mapping.Models[0].Name, mapping.Models[1].Name}
	assert.Contains(t, names, "private-model-1")
	assert.Contains(t, names, "private-model-2")
}
