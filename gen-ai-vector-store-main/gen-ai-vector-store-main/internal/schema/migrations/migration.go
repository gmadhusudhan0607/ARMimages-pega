/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package migrations

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
)

// Migration represents a database schema migration that can be applied
// to update the database schema from one version to another.
type Migration interface {
	// Version returns the target version of this migration (e.g., "v0.16.0")
	Version() string

	// SourceVersion returns the source version that this migration upgrades from
	SourceVersion() string

	// Description returns a human-readable description of the migration
	Description() string

	// Apply applies the migration to the database
	Apply(ctx context.Context, logger *zap.Logger, database db.Database) error

	// Dependencies returns a list of versions that must be applied before this migration
	Dependencies() []string
}

// KeyVsSchemaVersion is the configuration key for the current schema version
const KeyVsSchemaVersion = "schema_version"

// KeyVsSchemaVersionPrev is the configuration key for the previous schema version
const KeyVsSchemaVersionPrev = "schema_version_prev"

// MigrationRunner is responsible for running migrations in the correct order
type MigrationRunner struct {
	logger   *zap.Logger
	database db.Database
	registry *MigrationRegistry
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(logger *zap.Logger, database db.Database, registry *MigrationRegistry) *MigrationRunner {
	return &MigrationRunner{
		logger:   logger,
		database: database,
		registry: registry,
	}
}

// RunMigration runs a specific migration
func (r *MigrationRunner) RunMigration(ctx context.Context, migration Migration) error {
	r.logger.Info("Applying migration",
		zap.String("from", migration.SourceVersion()),
		zap.String("to", migration.Version()),
		zap.String("description", migration.Description()))

	// Validate that the current version is the source version this migration expects
	currentVersion, err := GetConfiguration(ctx, r.logger, r.database.GetConn(), KeyVsSchemaVersion)
	if err == nil && currentVersion != migration.SourceVersion() && currentVersion != "" {
		r.logger.Warn("Current version does not match expected source version for migration",
			zap.String("currentVersion", currentVersion),
			zap.String("expectedSourceVersion", migration.SourceVersion()),
			zap.String("targetVersion", migration.Version()))
		// We still proceed as this might be a deliberate upgrade path
	}

	if err := migration.Apply(ctx, r.logger, r.database); err != nil {
		return fmt.Errorf("failed to apply migration %s: %w", migration.Version(), err)
	}

	// Update the schema versions in the configuration after successful migration
	if err := r.updateSchemaVersions(ctx, currentVersion, migration.Version()); err != nil {
		r.logger.Warn("Failed to update schema versions",
			zap.String("currentVersion", currentVersion),
			zap.String("newVersion", migration.Version()),
			zap.Error(err))
		// We don't return an error here as the migration itself was successful
	} else {
		r.logger.Info("Updated schema version",
			zap.String("newVersion", migration.Version()),
			zap.String("previousVersion", currentVersion))
	}

	r.logger.Info("Successfully applied migration", zap.String("version", migration.Version()))
	return nil
}

// updateSchemaVersions updates both schema_version and schema_version_prev in the configuration table
func (r *MigrationRunner) updateSchemaVersions(ctx context.Context, previousVersion, currentVersion string) error {
	// First update the previous version
	if previousVersion != "" {
		_, err := UpsertConfiguration(ctx, r.logger, r.database.GetConn(), KeyVsSchemaVersionPrev, previousVersion)
		if err != nil {
			return fmt.Errorf("failed to update previous schema version: %w", err)
		}
	}

	// Then update the current version
	_, err := UpsertConfiguration(ctx, r.logger, r.database.GetConn(), KeyVsSchemaVersion, currentVersion)
	if err != nil {
		return fmt.Errorf("failed to update current schema version: %w", err)
	}

	return nil
}

// RunMigrationsFrom runs all migrations starting from the specified version
func (r *MigrationRunner) RunMigrationsFrom(ctx context.Context, fromVersion string) error {
	migrations, err := r.registry.GetMigrationsFrom(fromVersion)
	if err != nil {
		return fmt.Errorf("failed to get migrations from version %s: %w", fromVersion, err)
	}

	for _, migration := range migrations {
		if err := r.RunMigration(ctx, migration); err != nil {
			return err
		}
	}

	return nil
}

// BaseMigration provides a basic implementation of the Migration interface
type BaseMigration struct {
	version       string
	sourceVersion string
	description   string
	deps          []string
	applyFunc     func(ctx context.Context, logger *zap.Logger, database db.Database) error
}

// NewBaseMigration creates a new base migration
func NewBaseMigration(version, description string, applyFunc func(ctx context.Context, logger *zap.Logger, database db.Database) error, sourceVersion string, additionalDependencies ...string) *BaseMigration {
	deps := append([]string{sourceVersion}, additionalDependencies...)
	return &BaseMigration{
		version:       version,
		sourceVersion: sourceVersion,
		description:   description,
		deps:          deps,
		applyFunc:     applyFunc,
	}
}

// Version returns the target version of this migration
func (m *BaseMigration) Version() string {
	return m.version
}

// SourceVersion returns the source version that this migration upgrades from
func (m *BaseMigration) SourceVersion() string {
	return m.sourceVersion
}

// Description returns a human-readable description of the migration
func (m *BaseMigration) Description() string {
	return m.description
}

// Dependencies returns a list of versions that must be applied before this migration
func (m *BaseMigration) Dependencies() []string {
	return m.deps
}

// Apply applies the migration to the database
func (m *BaseMigration) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
	return m.applyFunc(ctx, logger, database)
}
