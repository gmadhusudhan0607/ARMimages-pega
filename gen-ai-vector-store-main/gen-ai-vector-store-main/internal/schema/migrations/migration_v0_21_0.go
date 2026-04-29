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

const (
	ConfigurationPrefixV0x21x0 = "schema_migration_v0.21.0"
	VersionV0x21x0             = "v0.21.0"
)

type MigrationV0x21x0 struct {
	*BaseMigration

	// Runtime dependencies - set during execution
	logger   *zap.Logger
	database db.Database
}

func init() {
	RegisterMigration(NewMigrationV0x21x0())
}

func NewMigrationV0x21x0() *MigrationV0x21x0 {
	migration := &MigrationV0x21x0{
		BaseMigration: NewBaseMigration(
			VersionV0x21x0,
			"Migration to v0.21.0 - Add PDC endpoint URL support to isolations table",
			nil,            // Will be set below
			VersionV0x19x0, // Source version (explicitly indicates migration from v0.19.0 to v0.21.0)
		),
	}
	// Set the apply function to be the method on this struct
	migration.BaseMigration.applyFunc = migration.Apply
	return migration
}

func (m *MigrationV0x21x0) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
	// Store dependencies for use during migration
	m.logger = logger
	m.database = database

	// Defer cleanup to ensure we don't keep references
	defer func() {
		m.logger = nil
		m.database = nil
	}()

	// Add pdc_endpoint_url column to isolations table
	if err := m.addPdcEndpointUrlColumn(ctx); err != nil {
		return err
	}

	return nil
}

// addPdcEndpointUrlColumn adds the pdc_endpoint_url column to the isolations table
func (m *MigrationV0x21x0) addPdcEndpointUrlColumn(ctx context.Context) error {
	m.logger.Info("Adding pdc_endpoint_url column to isolations table")

	key := fmt.Sprintf("%s_add_pdc_endpoint_url", ConfigurationPrefixV0x21x0)
	query := "ALTER TABLE vector_store.isolations ADD COLUMN IF NOT EXISTS pdc_endpoint_url TEXT"

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add pdc_endpoint_url column to isolations table: %w", err)
	}

	if changed {
		m.logger.Info("Successfully added pdc_endpoint_url column to isolations table")
	} else {
		m.logger.Debug("pdc_endpoint_url column already exists in isolations table")
	}

	return nil
}
