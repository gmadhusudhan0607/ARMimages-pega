// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package migrations

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
)

const (
	ConfigurationPrefixV0x18x0 = "schema_migration_v0.18.0"
	VersionV0x18x0             = "v0.18.0"
)

// MigrationV0x18x0 represents the migration to version v0.18.0
type MigrationV0x18x0 struct {
	*BaseMigration

	// Runtime dependencies - set during execution
	logger   *zap.Logger
	database db.Database
}

// init registers the v0.18.0 migration with the default registry
func init() {
	RegisterMigration(NewMigrationV0x18x0())
}

// NewMigrationV0x18x0 creates a new v0.18.0 migration
func NewMigrationV0x18x0() *MigrationV0x18x0 {
	migration := &MigrationV0x18x0{
		BaseMigration: NewBaseMigration(
			VersionV0x18x0,
			"Migration to v0.18.0 - Rebuild database statistics after PostgreSQL 14 to 17 upgrade.",
			nil,            // Will be set below
			VersionV0x17x0, // Source version (explicitly indicates migration from v0.17.0 to v0.18.0)
		),
	}
	// Set the apply function to be the method on this struct
	migration.BaseMigration.applyFunc = migration.Apply
	return migration
}

// Apply applies all the v0.18.0 schema changes
func (m *MigrationV0x18x0) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
	// Store dependencies for use during migration
	m.logger = logger
	m.database = database

	// Defer cleanup to ensure we don't keep references
	defer func() {
		m.logger = nil
		m.database = nil
	}()

	m.logger.Info("Starting database statistics rebuild for PostgreSQL 14 to 17 upgrade", zap.String("migration", VersionV0x18x0))

	// Execute the statistics rebuild
	if err := m.rebuildStatistics(ctx); err != nil {
		return fmt.Errorf("failed to execute statistics rebuild: %w", err)
	}

	m.logger.Info("Successfully completed database statistics rebuild", zap.String("migration", VersionV0x18x0))
	return nil
}

// rebuildStatistics executes statistics rebuild with standard sampling
func (m *MigrationV0x18x0) rebuildStatistics(ctx context.Context) error {
	m.logger.Info("Executing statistics rebuild (comprehensive rebuild with default_statistics_target=100)")

	key := fmt.Sprintf("%s_rebuild_statistics", ConfigurationPrefixV0x18x0)
	query := `
		SET default_statistics_target=100;
		ANALYZE;
	`

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to execute statistics rebuild: %w", err)
	}

	if changed {
		m.logger.Info("Completed statistics rebuild (default_statistics_target=100)")
	} else {
		m.logger.Debug("Statistics rebuild already completed")
	}

	return nil
}
