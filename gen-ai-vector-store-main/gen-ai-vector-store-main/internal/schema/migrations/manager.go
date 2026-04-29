/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package migrations

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"
)

// MigrationManager handles the overall process of determining which migrations to run
// and running them in the correct order
type MigrationManager struct {
	logger   *zap.Logger
	database db.Database
	registry *MigrationRegistry
	runner   *MigrationRunner
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(logger *zap.Logger, database db.Database) *MigrationManager {
	runner := NewMigrationRunner(logger, database, DefaultRegistry)

	return &MigrationManager{
		logger:   logger,
		database: database,
		registry: DefaultRegistry,
		runner:   runner,
	}
}

// RegisterMigration registers a new migration with the manager's registry
func (m *MigrationManager) RegisterMigration(migration Migration) error {
	return m.registry.Register(migration)
}

// ValidateMigrationChain verifies that all migrations form a proper chain
// where each migration's dependency is satisfied by another migration
func (m *MigrationManager) ValidateMigrationChain() error {
	migrations := m.registry.GetAllMigrations()

	// Create a map of all available versions
	availableVersions := make(map[string]bool)
	for _, migration := range migrations {
		availableVersions[migration.Version()] = true
	}

	// Check each migration's dependencies
	for _, migration := range migrations {
		deps := migration.Dependencies()
		for _, dep := range deps {
			if !availableVersions[dep] && dep != "" {
				return fmt.Errorf("migration %s depends on %s which is not registered",
					migration.Version(), dep)
			}
		}
	}

	return nil
}

// GetCurrentSchemaVersion fetches the current schema version from the database
func (m *MigrationManager) GetCurrentSchemaVersion(ctx context.Context) (string, error) {
	version, err := GetConfiguration(ctx, m.logger, m.database.GetConn(), KeyVsSchemaVersion)
	if err != nil {
		m.logger.Warn("No schema version found in database", zap.Error(err))
		return "", nil
	}
	return version, nil
}

// SetCurrentSchemaVersion updates the current schema version in the database
func (m *MigrationManager) SetCurrentSchemaVersion(ctx context.Context, version string) error {
	m.logger.Debug("updating schema version", zap.String("version", version))
	_, err := UpsertConfiguration(ctx, m.logger, m.database.GetConn(), KeyVsSchemaVersion, version)
	if err != nil {
		return fmt.Errorf("failed to update schema version to %s: %w", version, err)
	}
	return nil
}

// RunPendingMigrations runs all pending migrations based on the current schema version
func (m *MigrationManager) RunPendingMigrations(ctx context.Context) error {
	if err := m.ValidateMigrationChain(); err != nil {
		return fmt.Errorf("invalid migration chain: %w", err)
	}

	currentVersion, err := m.GetCurrentSchemaVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	m.logger.Info("current schema version", zap.String("version", currentVersion))

	allMigrations := m.registry.GetAllMigrations()
	m.logger.Debug("found migrations to apply", zap.Int("count", len(allMigrations)))

	pendingMigrations := m.filterPendingMigrations(currentVersion, allMigrations)

	if len(pendingMigrations) == 0 {
		m.logger.Info("no pending migrations to apply")
		return nil
	}

	m.logger.Info("Preparing to apply pending migrations", zap.Int("count", len(pendingMigrations)))

	for _, migration := range pendingMigrations {
		if err := m.runner.RunMigration(ctx, migration); err != nil {
			return err
		}
		currentVersion = migration.Version()
		if err := m.SetCurrentSchemaVersion(ctx, currentVersion); err != nil {
			return fmt.Errorf("failed to update schema version to %s: %w", currentVersion, err)
		}
	}

	return nil
}

// filterPendingMigrations returns a list of migrations that need to be applied
// based on the current schema version
func (m *MigrationManager) filterPendingMigrations(currentVersion string, allMigrations []Migration) []Migration {
	var pendingMigrations []Migration

	// If no current version, we need to apply all migrations
	if currentVersion == "" {
		return allMigrations
	}

	// Get max version limit from environment (for testing purposes)
	maxVersion := os.Getenv("DB_SCHEMA_MAX_MIGRATION_VERSION")
	if maxVersion != "" {
		m.logger.Info("migration limit set", zap.String("maxVersion", maxVersion))
	}

	// Only include migrations with a version greater than the current version
	for _, migration := range allMigrations {
		if semver.Compare(migration.Version(), currentVersion) > 0 {
			// Check if we have a max version limit
			if maxVersion != "" && semver.Compare(migration.Version(), maxVersion) > 0 {
				m.logger.Debug("skipping migration beyond max version",
					zap.String("migration", migration.Version()),
					zap.String("maxVersion", maxVersion))
				continue // Skip migrations beyond the max version
			}
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	// Sort by version to ensure they are applied in order
	sort.Slice(pendingMigrations, func(i, j int) bool {
		return semver.Compare(pendingMigrations[i].Version(), pendingMigrations[j].Version()) < 0
	})

	return pendingMigrations
}
