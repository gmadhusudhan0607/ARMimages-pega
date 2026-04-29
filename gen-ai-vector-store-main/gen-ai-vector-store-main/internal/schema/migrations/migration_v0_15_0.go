/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package migrations

import (
	"context"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
)

const (
	VersionV0x15x0 = "v0.15.0"
)

// init registers the v0.15.0 migration with the default registry
func init() {
	RegisterMigration(NewMigrationV0x15x0())
}

// MigrationV0x15x0 represents the migration to version v0.15.0
type MigrationV0x15x0 struct {
	*BaseMigration

	// Runtime dependencies - set during execution
	logger   *zap.Logger
	database db.Database
}

// NewMigrationV0x15x0 creates a new v0.15.0 migration
func NewMigrationV0x15x0() *MigrationV0x15x0 {
	migration := &MigrationV0x15x0{
		BaseMigration: NewBaseMigration(
			VersionV0x15x0,
			"Migration to v0.15.0 - Initial migration",
			nil, // Will be set below
			"",  // Empty string means no source version dependency (this is the first migration)
		),
	}
	// Set the apply function to be the method on this struct
	migration.BaseMigration.applyFunc = migration.Apply
	return migration
}

// Apply applies all the v0.15.0 schema changes (that are nothing in this case)
func (m *MigrationV0x15x0) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
	// Store dependencies for use during migration
	m.logger = logger
	m.database = database

	// Defer cleanup to ensure we don't keep references
	defer func() {
		m.logger = nil
		m.database = nil
	}()

	m.logger.Info("v0.15.0 initial migration setup - no schema changes to apply", zap.String("migration", VersionV0x15x0))
	return nil
}
