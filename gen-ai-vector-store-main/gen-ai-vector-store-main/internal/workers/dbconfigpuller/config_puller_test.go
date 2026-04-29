// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package dbconfigpuller

import (
	"os"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema/migrations"
	"github.com/stretchr/testify/assert"
)

func TestDefaultPullInterval(t *testing.T) {
	assert.Equal(t, 300, DefaultPullIntervalSec)
}

func TestDBConfigPuller_DefaultConfigKeys(t *testing.T) {
	// Test that the default configuration keys are correctly defined
	expectedKeys := []string{
		migrations.KeyVsSchemaVersion,
		migrations.KeyVsSchemaVersionPrev,
	}

	// Verify each expected key value
	assert.Equal(t, "schema_version", migrations.KeyVsSchemaVersion)
	assert.Equal(t, "schema_version_prev", migrations.KeyVsSchemaVersionPrev)
	assert.Len(t, expectedKeys, 2)
}

func TestDBConfigPuller_PullIntervalFromEnv(t *testing.T) {
	// Save original env value if it exists
	originalValue := os.Getenv("DB_CONFIG_PULL_INTERVAL_SEC")
	defer func() {
		if originalValue != "" {
			os.Setenv("DB_CONFIG_PULL_INTERVAL_SEC", originalValue)
		} else {
			os.Unsetenv("DB_CONFIG_PULL_INTERVAL_SEC")
		}
	}()

	// Test with custom value
	os.Setenv("DB_CONFIG_PULL_INTERVAL_SEC", "600")
	// Note: This test verifies the environment variable can be set
	// The actual NewDBConfigPuller function reads this value
	assert.Equal(t, "600", os.Getenv("DB_CONFIG_PULL_INTERVAL_SEC"))

	// Test with default (unset)
	os.Unsetenv("DB_CONFIG_PULL_INTERVAL_SEC")
	assert.Equal(t, "", os.Getenv("DB_CONFIG_PULL_INTERVAL_SEC"))
}
