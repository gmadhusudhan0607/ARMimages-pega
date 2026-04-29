/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Loader Tests

func TestLoadStaticMapping(t *testing.T) {
	tempDir := t.TempDir()
	configFile := tempDir + "/test-config.yaml"

	// Create test YAML content
	yamlContent := `
models:
  - name: test-model
    infrastructure: azure
    provider: azure
    creator: openai
    modelId: test-model-id
    redirectURL: https://test.example.com
    active: true
buddies:
  - name: test-buddy
    redirectURL: https://buddy.example.com
    active: true
`
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	mapping, err := loadStaticMapping(configFile)
	require.NoError(t, err)
	require.NotNil(t, mapping)
	assert.Len(t, mapping.Models, 1)
	assert.Len(t, mapping.Buddies, 1)
	assert.Equal(t, "test-model", mapping.Models[0].Name)
	assert.Equal(t, "test-buddy", mapping.Buddies[0].Name)
}

func TestLoadStaticMapping_FileNotFound(t *testing.T) {
	_, err := loadStaticMapping("/nonexistent/path/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read configuration file")
}

func TestLoadStaticMapping_EmptyPath(t *testing.T) {
	_, err := loadStaticMapping("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration file path is empty")
}

func TestFindModelInMapping(t *testing.T) {
	mapping := createTestMapping()

	// Test finding existing model
	model, found := findModelInMapping(mapping, "gpt-4o")
	assert.True(t, found)
	require.NotNil(t, model)
	assert.Equal(t, "gpt-4o", model.Name)

	// Test model not found
	model, found = findModelInMapping(mapping, "nonexistent")
	assert.False(t, found)
	assert.Nil(t, model)

	// Test with nil mapping
	model, found = findModelInMapping(nil, "gpt-4o")
	assert.False(t, found)
	assert.Nil(t, model)
}

func TestFindBuddyInMapping(t *testing.T) {
	mapping := createTestMapping()

	// Test finding existing buddy
	buddy, found := findBuddyInMapping(mapping, "selfstudybuddy")
	assert.True(t, found)
	require.NotNil(t, buddy)
	assert.Equal(t, "selfstudybuddy", buddy.Name)

	// Test buddy not found
	buddy, found = findBuddyInMapping(mapping, "nonexistent")
	assert.False(t, found)
	assert.Nil(t, buddy)

	// Test with nil mapping
	buddy, found = findBuddyInMapping(nil, "selfstudybuddy")
	assert.False(t, found)
	assert.Nil(t, buddy)
}
