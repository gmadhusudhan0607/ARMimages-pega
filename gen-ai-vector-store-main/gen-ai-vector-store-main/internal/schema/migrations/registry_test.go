// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationRegistry_GetLatestVersionOrDefault(t *testing.T) {
	t.Run("returns default version when no migrations registered", func(t *testing.T) {
		registry := NewMigrationRegistry()
		defaultVersion := "v0.14.0"
		latestVersion := registry.GetLatestVersionOrDefault(defaultVersion)
		assert.Equal(t, defaultVersion, latestVersion)
	})

	t.Run("returns the only version when single migration registered", func(t *testing.T) {
		registry := NewMigrationRegistry()
		defaultVersion := "v0.14.0"
		migration := NewBaseMigration("v1.0.0", "Test migration", nil, "")
		err := registry.Register(migration)
		require.NoError(t, err)

		latestVersion := registry.GetLatestVersionOrDefault(defaultVersion)
		assert.Equal(t, "v1.0.0", latestVersion)
	})

	t.Run("returns highest version when multiple migrations registered", func(t *testing.T) {
		registry := NewMigrationRegistry()
		defaultVersion := "v0.14.0"

		migrations := []Migration{
			NewBaseMigration("v1.0.0", "Migration 1", nil, ""),
			NewBaseMigration("v1.2.0", "Migration 2", nil, "v1.0.0"),
			NewBaseMigration("v1.1.0", "Migration 3", nil, "v1.0.0"),
			NewBaseMigration("v2.0.0", "Migration 4", nil, "v1.2.0"),
		}

		for _, migration := range migrations {
			err := registry.Register(migration)
			require.NoError(t, err)
		}

		latestVersion := registry.GetLatestVersionOrDefault(defaultVersion)
		assert.Equal(t, "v2.0.0", latestVersion)
	})

	t.Run("ignores default version when migrations are present", func(t *testing.T) {
		registry := NewMigrationRegistry()
		defaultVersion := "v0.14.0"

		migrations := []Migration{
			NewBaseMigration("v0.15.0", "Migration 1", nil, ""),
			NewBaseMigration("v0.19.0", "Migration 2", nil, "v0.15.0"),
		}

		for _, migration := range migrations {
			err := registry.Register(migration)
			require.NoError(t, err)
		}

		// Should return v0.19.0, not the default v0.14.0
		latestVersion := registry.GetLatestVersionOrDefault(defaultVersion)
		assert.Equal(t, "v0.19.0", latestVersion)
	})
}
