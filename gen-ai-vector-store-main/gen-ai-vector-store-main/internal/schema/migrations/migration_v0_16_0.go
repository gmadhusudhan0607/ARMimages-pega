/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package migrations

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers/hnsw"
	"go.uber.org/zap"
)

const (
	ConfigurationPrefixV0x16x0 = "schema_migration_v0.16.0"
	VersionV0x16x0             = "v0.16.0"
)

// MigrationV0x16x0 represents the migration to version v0.16.0
type MigrationV0x16x0 struct {
	*BaseMigration

	// Runtime dependencies - set during execution
	logger   *zap.Logger
	database db.Database
}

// init registers the v0.16.0 migration with the default registry
func init() {
	RegisterMigration(NewMigrationV0x16x0())
}

// NewMigrationV0x16x0 creates a new v0.16.0 migration
func NewMigrationV0x16x0() *MigrationV0x16x0 {
	migration := &MigrationV0x16x0{
		BaseMigration: NewBaseMigration(
			VersionV0x16x0,
			"Migration to v0.16.0 - Add HNSW Indexes.",
			nil,            // Will be set below
			VersionV0x15x0, // Source version (explicitly indicates migration from v0.15.0 to v0.16.0)
		),
	}
	// Set the apply function to be the method on this struct
	migration.BaseMigration.applyFunc = migration.Apply
	return migration
}

// Apply applies all the v0.16.0 schema changes
func (m *MigrationV0x16x0) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
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

		// Titan model needs to be updated first to have the correct vector length
		if err = m.UpdateTitanLen(ctx, isoId); err != nil {
			return fmt.Errorf("failed to update Titan model length for isolation %s: %w", isoId, err)
		}

		// Iterate over each collection ID and apply the changes (alter table and create HNSW index)
		for _, colId := range colIDs {
			colProfiles, err := GetCollectionProfiles(ctx, logger, database.GetConn(), isoId, colId)
			if err != nil {
				return fmt.Errorf("failed to get collection profiles for isolation %s and collection %s: %w", isoId, colId, err)
			}
			// print colProfile in loop
			for _, profileId := range colProfiles {
				err2 := m.AlterEmbTable(ctx, isoId, colId, profileId)
				if err2 != nil {
					return err2
				}
				err2 = m.CreateHNSWIndex(ctx, isoId, colId, profileId)
				if err2 != nil {
					return err2
				}
			}
		}
	}
	return nil
}

// UpdateTitanLen updates the vector length for the Titan model if it's null or zero
func (m *MigrationV0x16x0) UpdateTitanLen(ctx context.Context, isoId string) error {
	vectorLen := 1024 // Default vector length for Amazon Titan embedding model

	// Get the embedding profiles table for this isolation
	embProfilesTable := db.GetTableEmbeddingProfiles(isoId)

	// SQL to update vectorLen for amazon-titan-embed-text profile where it's 0 or NULL
	query := fmt.Sprintf(
		"UPDATE %s SET vector_len = $1 WHERE profile_id = 'amazon-titan-embed-text' AND (vector_len = 0 OR vector_len IS NULL)",
		embProfilesTable)

	// Create a migration key for this specific update
	key := fmt.Sprintf("%s_update_titan_vector_len_%s", ConfigurationPrefixV0x16x0, isoId)

	// Execute the update query
	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query, vectorLen)
	if err != nil {
		return fmt.Errorf("failed to update Amazon Titan embedding model vector length for isolation %s: %w", isoId, err)
	}

	if changed {
		m.logger.Info("successfully updated Amazon Titan embedding model vector length", zap.Int("vectorLen", vectorLen), zap.String("isolationID", isoId))
	} else {
		m.logger.Info("Amazon Titan embedding model vector length already set correctly", zap.String("isolationID", isoId))
	}

	return nil
}

// AlterEmbTable alters the embedding table to set the correct vector size
func (m *MigrationV0x16x0) AlterEmbTable(ctx context.Context, isoId string, colId string, profileId string) error {
	m.logger.Info("processing collection", zap.String("collectionID", colId), zap.String("profileID", profileId))

	tablePrefix, err := GetCollectionProfileTablesPrefix(ctx, m.logger, m.database.GetConn(), isoId, colId, profileId)
	if err != nil {
		return fmt.Errorf("failed to get table prefix for profile %s in isolation %s: %w", profileId, isoId, err)
	}

	tableEmb := fmt.Sprintf("%s.t_%s_emb", db.GetSchema(isoId), tablePrefix)

	vectorLen, err := m.GetVectorLengthForCollection(isoId, profileId)
	if err != nil {
		return fmt.Errorf("failed to get vector length for collection %s in isolation %s: %w", colId, isoId, err)
	}

	query := fmt.Sprintf("ALTER TABLE %[1]s ALTER COLUMN embedding TYPE vector(%d)", tableEmb, vectorLen)

	key := fmt.Sprintf("%s_alter_table_set_vector_len_%s_%s", ConfigurationPrefixV0x16x0, isoId, colId)
	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to execute query for collection %s in isolation %s: %w", colId, isoId, err)
	}
	if !changed {
		m.logger.Info("no changes made for collection, vector size already set", zap.String("collectionID", colId), zap.String("isolationID", isoId))
	} else {
		m.logger.Info("successfully set vector size", zap.Int("vectorLen", vectorLen), zap.String("collectionID", colId), zap.String("isolationID", isoId))
	}
	return nil
}

// GetVectorLengthForCollection retrieves the vector length for a collection from its embedding profile
func (m *MigrationV0x16x0) GetVectorLengthForCollection(isoId, profileId string) (int, error) {
	// Get the vector_len from the emb_profiles table for this profile
	embProfilesTable := db.GetTableEmbeddingProfiles(isoId)
	vectorLenQuery := fmt.Sprintf("SELECT vector_len FROM %[1]s WHERE profile_id = $1", embProfilesTable)

	var vectorLen int
	rows, err := m.database.GetConn().Query(vectorLenQuery, profileId)
	if err != nil {
		return 0, fmt.Errorf("failed to get vector_len for profile %s: %w", profileId, err)
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&vectorLen)
		if err != nil {
			return 0, fmt.Errorf("failed to scan vector_len for profile %s: %w", profileId, err)
		}
	} else {
		return 0, fmt.Errorf("no vector_len found for profile %s", profileId)
	}

	// Validate that vectorLen is greater than 0
	if vectorLen <= 0 {
		return 0, fmt.Errorf("invalid vector_len value %d for profile %s: must be greater than 0", vectorLen, profileId)
	}

	return vectorLen, nil
}

// CreateHNSWIndex creates an HNSW index for the given collection profile
func (m *MigrationV0x16x0) CreateHNSWIndex(ctx context.Context, isoId string, colId string, profileId string) error {
	m.logger.Info("creating HNSW index for collection", zap.String("collectionID", colId), zap.String("profileID", profileId))

	// need single connection to run both setting parameter and creating index in one session
	conn, err := m.database.GetSingleConn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get single connection for isolation %s, collection %s and profile %s: %w", isoId, colId, profileId, err)
	}
	defer conn.Close()

	setParamsQuery, err := hnsw.BuildSetParametersQuery(
		ctx,
		m.logger.With(zap.String("isolationID", isoId), zap.String("collectionID", colId)),
		m.database.GetConn(),
		isoId,
		colId,
		profileId,
	)
	if err != nil {
		return fmt.Errorf("failed to build set parameters query for isolation %s, collection %s and profile %s: %w", isoId, colId, profileId, err)
	}
	m.logger.Debug("Setting HNSW parameters with query", zap.String("query", setParamsQuery))
	// Execute the set parameters query
	_, err = conn.ExecContext(ctx, setParamsQuery)
	if err != nil {
		return fmt.Errorf("failed to set HNSW parameters for collection %s with profile %s in isolation %s: %w",
			colId, profileId, isoId, err)
	}

	// Create the SQL for index creation with dynamic parameters
	query, err := hnsw.BuildCreateIndexQuery(
		ctx,
		m.logger.With(zap.String("isolationID", isoId), zap.String("collectionID", colId)),
		conn,
		isoId,
		colId,
		profileId,
	)
	if err != nil {
		return fmt.Errorf("failed to build create index query for isolation %s, collection %s and profile %s: %w", isoId, colId, profileId, err)
	}
	m.logger.Info("Creating HNSW index with query", zap.String("query", query))

	// Create a migration key for this specific index creation
	key := fmt.Sprintf("%s_create_hnsw_index_%s_%s_%s", ConfigurationPrefixV0x16x0, isoId, colId, profileId)

	// Execute the index creation
	changed, err := ExecuteOnce(ctx, m.logger, conn, key, query)
	if err != nil {
		return fmt.Errorf("failed to create HNSW index for collection %s with profile %s in isolation %s: %w",
			colId, profileId, isoId, err)
	}

	if !changed {
		m.logger.Info("HNSW index already exists for collection", zap.String("collectionID", colId), zap.String("profileID", profileId), zap.String("isolationID", isoId))
	} else {
		m.logger.Info("successfully created HNSW index for collection", zap.String("collectionID", colId), zap.String("profileID", profileId), zap.String("isolationID", isoId))
	}

	return nil
}
