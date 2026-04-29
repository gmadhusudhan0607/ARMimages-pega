/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package migrations

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers/hnsw"
	"go.uber.org/zap"
)

const (
	ConfigurationPrefixV0x16x2 = "schema_migration_v0.16.2"
	VersionV0x16x2             = "v0.16.2"
)

// MigrationV0x16x2 represents the migration to version v0.16.2
type MigrationV0x16x2 struct {
	*BaseMigration

	// Runtime dependencies - set during execution
	logger   *zap.Logger
	database db.Database
}

// init registers the v0.16.2 migration with the default registry
func init() {
	RegisterMigration(NewMigrationV0x16x2())
}

// NewMigrationV0x16x0 creates a new v0.16.2 migration
func NewMigrationV0x16x2() *MigrationV0x16x2 {
	migration := &MigrationV0x16x2{
		BaseMigration: NewBaseMigration(
			VersionV0x16x2,
			"Migration to v0.16.2 - Handle invalid indexes.",
			nil,            // Will be set below
			VersionV0x16x0, // Source version (explicitly indicates migration from v0.16.0 to v0.16.2)
		),
	}
	// Set the apply function to be the method on this struct
	migration.BaseMigration.applyFunc = migration.Apply
	return migration
}

// Apply applies all the v0.16.2 schema changes
func (m *MigrationV0x16x2) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
	// Store dependencies for use during migration
	m.logger = logger
	m.database = database

	// Defer cleanup to ensure we don't keep references
	defer func() {
		m.logger = nil
		m.database = nil
	}()

	isoIDs, err := GetIsolationIDs(ctx, logger, database.GetConn())
	if err != nil {
		return fmt.Errorf("failed to get isolation IDs: %w", err)
	}
	for _, isoId := range isoIDs {
		colIDs, err := GetCollectionIDs(ctx, logger, database.GetConn(), isoId)
		if err != nil {
			return fmt.Errorf("failed to get collection IDs for isolation %s: %w", isoId, err)
		}

		// Iterate over each collection ID and apply the changes (alter table and create HNSW index)
		for _, colId := range colIDs {
			colProfiles, err := GetCollectionProfiles(ctx, logger, database.GetConn(), isoId, colId)
			if err != nil {
				return fmt.Errorf("failed to get collection profiles for isolation %s and collection %s: %w", isoId, colId, err)
			}
			// Process each profile for this collection
			for _, profileId := range colProfiles {
				err = m.recreateInvalidHNSWIndex(ctx, isoId, colId, profileId)
				if err != nil {
					return fmt.Errorf("failed to manage HNSW index for isolation %s, collection %s, profile %s: %w", isoId, colId, profileId, err)
				}
			}
		}
	}
	return nil
}

func (m *MigrationV0x16x2) recreateInvalidHNSWIndex(ctx context.Context, isoId, colId, profileId string) error {
	m.logger.Info("checking HNSW index",
		zap.String("collection", colId),
		zap.String("profile", profileId),
		zap.String("isolation", isoId),
	)

	idxInfo, err := m.getHNSWIndexInfo(isoId, colId)
	if err != nil {
		return fmt.Errorf("failed to get HNSW index info: %w", err)
	}

	valid, err := m.checkIfIndexIsValid(ctx, idxInfo)
	if err != nil {
		return err
	}

	if valid {
		m.logger.Info("HNSW index already exists and is valid",
			zap.String("collection", colId),
			zap.String("profile", profileId),
			zap.String("isolation", isoId),
		)
		return nil
	}

	// recreate index
	if err := m.dropInvalidIndex(ctx, isoId, colId, profileId, idxInfo); err != nil {
		return err
	}

	return m.createHNSWIndex(ctx, isoId, colId, profileId)
}

// getHNSWIndexInfo retrieves information about the HNSW index
func (m *MigrationV0x16x2) getHNSWIndexInfo(isoId, colId string) (hnswIndexInfo, error) {
	tableEmb := db.GetTableEmb(isoId, colId)
	_, tableEmbWithoutSchema := helpers.SplitTableName(tableEmb)
	idxName := hnsw.GetIdxName(tableEmbWithoutSchema)

	schemaName, tableName := helpers.SplitTableName(tableEmb)

	return hnswIndexInfo{
		schema:        schemaName,
		table:         tableName,
		indexName:     idxName,
		fullTableName: tableEmb,
	}, nil
}

// hnswIndexInfo holds information about the HNSW index
type hnswIndexInfo struct {
	schema        string
	table         string
	indexName     string
	fullTableName string
}

// checkIfIndexIsValid checks if the HNSW index is valid
func (m *MigrationV0x16x2) checkIfIndexIsValid(ctx context.Context, info hnswIndexInfo) (bool, error) {
	validQuery := `
		SELECT i.indisvalid
		FROM pg_index i
		JOIN pg_class c ON i.indexrelid = c.oid
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE n.nspname = $1 AND c.relname = $2
	`

	var isValid bool
	rows, err := m.database.GetConn().QueryContext(ctx, validQuery, info.schema, info.indexName)
	if err != nil {
		return false, fmt.Errorf("failed to check if index is valid: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&isValid)
		if err != nil {
			return false, fmt.Errorf("failed to scan index validity result: %w", err)
		}
	}

	return isValid, nil
}

// dropInvalidIndex drops an invalid HNSW index
func (m *MigrationV0x16x2) dropInvalidIndex(ctx context.Context, isoId, colId, profileId string, info hnswIndexInfo) error {
	m.logger.Info("dropping invalid HNSW index",
		zap.String("index", info.indexName),
		zap.String("collection", colId),
		zap.String("profile", profileId),
		zap.String("isolation", isoId),
	)

	dropQuery := fmt.Sprintf("DROP INDEX CONCURRENTLY IF EXISTS %s.%s", info.schema, info.indexName)

	_, err := m.database.GetConn().ExecContext(ctx, dropQuery)
	if err != nil {
		return fmt.Errorf("failed to drop invalid HNSW index: %w", err)
	}

	m.logger.Info("dropped invalid HNSW index",
		zap.String("index", info.indexName),
		zap.String("collection", colId),
		zap.String("profile", profileId),
		zap.String("isolation", isoId),
	)

	return nil
}

func (m *MigrationV0x16x2) createHNSWIndex(ctx context.Context, isoId, colId, profileId string) error {
	m.logger.Info("creating HNSW index",
		zap.String("collection", colId),
		zap.String("profile", profileId),
		zap.String("isolation", isoId),
	)

	// Need single connection to run both setting parameter and creating index in one session
	conn, err := m.database.GetSingleConn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get single database connection: %w", err)
	}
	defer conn.Close()

	setParamsQuery, err := hnsw.BuildSetParametersQuery(ctx, m.logger, m.database.GetConn(), isoId, colId, profileId)
	if err != nil {
		return fmt.Errorf("failed to build set parameters query: %w", err)
	}

	_, err = conn.ExecContext(ctx, setParamsQuery)
	if err != nil {
		return fmt.Errorf("failed to set parameters: %w", err)
	}

	createIndexQuery, err := hnsw.BuildCreateIndexQuery(ctx, m.logger, m.database.GetConn(), isoId, colId, profileId)
	if err != nil {
		return fmt.Errorf("failed to build create index query: %w", err)
	}

	// Execute the index creation
	_, err = conn.ExecContext(ctx, createIndexQuery)
	if err != nil {
		return fmt.Errorf("failed to create HNSW index: %w", err)
	}

	m.logger.Info("created HNSW index",
		zap.String("collection", colId),
		zap.String("profile", profileId),
		zap.String("isolation", isoId),
	)

	return nil
}
