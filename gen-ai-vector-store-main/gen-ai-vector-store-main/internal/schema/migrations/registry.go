/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package migrations

import (
	"fmt"
	"sort"

	"golang.org/x/mod/semver"
)

// MigrationRegistry maintains a registry of available database migrations
type MigrationRegistry struct {
	migrations map[string]Migration
}

// NewMigrationRegistry creates a new migration registry
func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{
		migrations: make(map[string]Migration),
	}
}

// Register adds a migration to the registry
func (r *MigrationRegistry) Register(migration Migration) error {
	version := migration.Version()

	if !semver.IsValid(version) {
		return fmt.Errorf("invalid version format for migration: %s", version)
	}

	if _, exists := r.migrations[version]; exists {
		return fmt.Errorf("migration with version %s is already registered", version)
	}

	r.migrations[version] = migration
	return nil
}

// GetMigration retrieves a specific migration by version
func (r *MigrationRegistry) GetMigration(version string) (Migration, error) {
	migration, exists := r.migrations[version]
	if !exists {
		return nil, fmt.Errorf("migration with version %s not found", version)
	}

	return migration, nil
}

// GetAllMigrations returns all registered migrations sorted by version
func (r *MigrationRegistry) GetAllMigrations() []Migration {
	versions := make([]string, 0, len(r.migrations))
	for version := range r.migrations {
		versions = append(versions, version)
	}

	// Sort versions according to semver rules
	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) < 0
	})

	migrations := make([]Migration, 0, len(versions))
	for _, version := range versions {
		migrations = append(migrations, r.migrations[version])
	}

	return migrations
}

// GetMigrationsFrom returns all migrations with a version greater than or equal to fromVersion
func (r *MigrationRegistry) GetMigrationsFrom(fromVersion string) ([]Migration, error) {
	if fromVersion != "" && !semver.IsValid(fromVersion) {
		return nil, fmt.Errorf("invalid fromVersion format: %s", fromVersion)
	}

	versions := make([]string, 0, len(r.migrations))
	for version := range r.migrations {
		// Only include versions >= fromVersion
		if fromVersion == "" || semver.Compare(version, fromVersion) >= 0 {
			versions = append(versions, version)
		}
	}

	// Sort versions according to semver rules
	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) < 0
	})

	migrations := make([]Migration, 0, len(versions))
	for _, version := range versions {
		migrations = append(migrations, r.migrations[version])
	}

	return migrations, nil
}

// GetLatestVersionOrDefault returns the highest version from all registered migrations
// Returns the provided defaultVersion if no migrations are registered
func (r *MigrationRegistry) GetLatestVersionOrDefault(defaultVersion string) string {
	migrations := r.GetAllMigrations()
	if len(migrations) == 0 {
		return defaultVersion
	}
	// GetAllMigrations returns migrations sorted by version, so the last one is the latest
	return migrations[len(migrations)-1].Version()
}

// DefaultRegistry is the default migration registry used by the application
var DefaultRegistry = NewMigrationRegistry()

// RegisterMigration registers a migration with the default registry
func RegisterMigration(migration Migration) {
	err := DefaultRegistry.Register(migration)
	if err != nil {
		panic(fmt.Sprintf("Failed to register migration %s: %s", migration.Version(), err))
	}
}
