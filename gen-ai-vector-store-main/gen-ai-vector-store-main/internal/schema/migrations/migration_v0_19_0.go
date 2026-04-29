/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package migrations

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	functions "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql/functions2"
	"go.uber.org/zap"
)

const (
	ConfigurationPrefixV0x19x0 = "schema_migration_v0.19.0"
	VersionV0x19x0             = "v0.19.0"
)

type MigrationV0x19x0 struct {
	*BaseMigration

	// Runtime dependencies - set during execution
	logger   *zap.Logger
	database db.Database
}

func init() {
	RegisterMigration(NewMigrationV0x19x0())
}

func NewMigrationV0x19x0() *MigrationV0x19x0 {
	migration := &MigrationV0x19x0{
		BaseMigration: NewBaseMigration(
			VersionV0x19x0,
			"Migration to v0.19.0 - Support JSONB attributes",
			nil,            // Will be set below
			VersionV0x18x0, // Source version (explicitly indicates migration from v0.18.0 to v0.19.0)
		),
	}
	// Set the apply function to be the method on this struct
	migration.BaseMigration.applyFunc = migration.Apply
	return migration
}

func (m *MigrationV0x19x0) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
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

		// Iterate over each collection ID
		for _, colId := range colIDs {
			colProfiles, err := GetCollectionProfiles(ctx, logger, database.GetConn(), isoId, colId)
			if err != nil {
				return fmt.Errorf("failed to get collection profiles for isolation %s and collection %s: %w", isoId, colId, err)
			}

			// Iterate over each profile in the collection
			for _, profileId := range colProfiles {
				m.logger.Info("Processing collection profile",
					zap.String("isolationID", isoId),
					zap.String("collectionID", colId),
					zap.String("profileID", profileId))

				// Get the table prefix for this profile
				tablePrefix, err := GetCollectionProfileTablesPrefix(ctx, logger, database.GetConn(), isoId, colId, profileId)
				if err != nil {
					return fmt.Errorf("failed to get table prefix for profile %s in collection %s, isolation %s: %w", profileId, colId, isoId, err)
				}

				// Apply document table changes
				if err := m.alterDocTableAddDocAttributes(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.createDocAttributesPathIndex(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.createDocAttributesOpsIndex(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}

				// Apply embedding table changes
				if err := m.alterEmbTableAddEmbAttributes(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.createEmbAttributesPathIndex(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.createEmbAttributesOpsIndex(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.alterEmbTableAddAttributes(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.createAttributesPathIndex(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.createAttributesOpsIndex(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}

				// Apply processing table changes (no indexes for processing tables)
				if err := m.alterDocProcessingTableAddDocAttributes(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.alterEmbProcessingTableAddEmbAttributes(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}
				if err := m.alterEmbProcessingTableAddAttributes(ctx, isoId, colId, profileId, tablePrefix); err != nil {
					return err
				}

				m.logger.Info("Successfully processed collection profile",
					zap.String("isolationID", isoId),
					zap.String("collectionID", colId),
					zap.String("profileID", profileId))
			}
		}
	}

	// Deploy SQL functions for optimized attribute replication
	if err := m.deploySQLFunctions(ctx); err != nil {
		return err
	}

	return nil
}

// deploySQLFunctions deploys the SQL functions needed for optimized attribute replication
func (m *MigrationV0x19x0) deploySQLFunctions(ctx context.Context) error {
	m.logger.Info("Deploying SQL functions for optimized attribute replication")

	key := fmt.Sprintf("%s_deploy_sql_functions", ConfigurationPrefixV0x19x0)

	// Use the embedded SQL functions from the functions2 package
	sqlFunctions := functions.Function_attribute_migration

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, sqlFunctions)
	if err != nil {
		return fmt.Errorf("failed to deploy SQL functions for attribute replication: %w", err)
	}

	if changed {
		m.logger.Info("Successfully deployed SQL functions for optimized attribute replication")
	} else {
		m.logger.Debug("SQL functions for attribute replication already deployed")
	}

	return nil
}

// alterDocProcessingTableAddDocAttributes adds the doc_attributes JSONB column to the document processing table
func (m *MigrationV0x19x0) alterDocProcessingTableAddDocAttributes(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Adding doc_attributes column to document processing table",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_alter_doc_processing_add_doc_attributes_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableDocProcessing := fmt.Sprintf("%s.t_%s_doc_processing", db.GetSchema(isoId), tablePrefix)
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS doc_attributes JSONB DEFAULT NULL", tableDocProcessing)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add doc_attributes column to table %s: %w", tableDocProcessing, err)
	}

	if changed {
		m.logger.Info("Added doc_attributes column to document processing table",
			zap.String("table", tableDocProcessing))
	} else {
		m.logger.Debug("doc_attributes column already exists in document processing table",
			zap.String("table", tableDocProcessing))
	}

	return nil
}

// alterEmbProcessingTableAddEmbAttributes adds the emb_attributes JSONB column to the embedding processing table
func (m *MigrationV0x19x0) alterEmbProcessingTableAddEmbAttributes(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Adding emb_attributes column to embedding processing table",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_alter_emb_processing_add_emb_attributes_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableEmbProcessing := fmt.Sprintf("%s.t_%s_emb_processing", db.GetSchema(isoId), tablePrefix)
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS emb_attributes JSONB DEFAULT NULL", tableEmbProcessing)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add emb_attributes column to table %s: %w", tableEmbProcessing, err)
	}

	if changed {
		m.logger.Info("Added emb_attributes column to embedding processing table",
			zap.String("table", tableEmbProcessing))
	} else {
		m.logger.Debug("emb_attributes column already exists in embedding processing table",
			zap.String("table", tableEmbProcessing))
	}

	return nil
}

// alterEmbProcessingTableAddAttributes adds the attributes JSONB column to the embedding processing table
func (m *MigrationV0x19x0) alterEmbProcessingTableAddAttributes(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Adding attributes column to embedding processing table",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_alter_emb_processing_add_attributes_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableEmbProcessing := fmt.Sprintf("%s.t_%s_emb_processing", db.GetSchema(isoId), tablePrefix)
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS attributes JSONB DEFAULT NULL", tableEmbProcessing)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add attributes column to table %s: %w", tableEmbProcessing, err)
	}

	if changed {
		m.logger.Info("Added attributes column to embedding processing table",
			zap.String("table", tableEmbProcessing))
	} else {
		m.logger.Debug("attributes column already exists in embedding processing table",
			zap.String("table", tableEmbProcessing))
	}

	return nil
}

// alterDocTableAddDocAttributes adds the doc_attributes JSONB column to the document table
func (m *MigrationV0x19x0) alterDocTableAddDocAttributes(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Adding doc_attributes column to document table",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_alter_doc_add_doc_attributes_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableDoc := fmt.Sprintf("%s.t_%s_doc", db.GetSchema(isoId), tablePrefix)
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS doc_attributes JSONB DEFAULT NULL", tableDoc)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add doc_attributes column to table %s: %w", tableDoc, err)
	}

	if changed {
		m.logger.Info("Added doc_attributes column to document table",
			zap.String("table", tableDoc))
	} else {
		m.logger.Debug("doc_attributes column already exists in document table",
			zap.String("table", tableDoc))
	}

	return nil
}

// createDocAttributesPathIndex creates the GIN index for doc_attributes using jsonb_path_ops
func (m *MigrationV0x19x0) createDocAttributesPathIndex(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Creating doc_attributes path index",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_idx_doc_attributes_path_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableDoc := fmt.Sprintf("%s.t_%s_doc", db.GetSchema(isoId), tablePrefix)
	indexName := fmt.Sprintf("idx_%s_doc_attributes_path", tablePrefix)
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIN (doc_attributes jsonb_path_ops)", indexName, tableDoc)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to create doc_attributes path index %s on table %s: %w", indexName, tableDoc, err)
	}

	if changed {
		m.logger.Info("Created doc_attributes path index",
			zap.String("index", indexName),
			zap.String("table", tableDoc))
	} else {
		m.logger.Debug("doc_attributes path index already exists",
			zap.String("index", indexName),
			zap.String("table", tableDoc))
	}

	return nil
}

// createDocAttributesOpsIndex creates the GIN index for doc_attributes using jsonb_ops
func (m *MigrationV0x19x0) createDocAttributesOpsIndex(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Creating doc_attributes ops index",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_idx_doc_attributes_ops_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableDoc := fmt.Sprintf("%s.t_%s_doc", db.GetSchema(isoId), tablePrefix)
	indexName := fmt.Sprintf("idx_%s_doc_attributes_ops", tablePrefix)
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIN (doc_attributes jsonb_ops)", indexName, tableDoc)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to create doc_attributes ops index %s on table %s: %w", indexName, tableDoc, err)
	}

	if changed {
		m.logger.Info("Created doc_attributes ops index",
			zap.String("index", indexName),
			zap.String("table", tableDoc))
	} else {
		m.logger.Debug("doc_attributes ops index already exists",
			zap.String("index", indexName),
			zap.String("table", tableDoc))
	}

	return nil
}

// alterEmbTableAddEmbAttributes adds the emb_attributes JSONB column to the embedding table
func (m *MigrationV0x19x0) alterEmbTableAddEmbAttributes(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Adding emb_attributes column to embedding table",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_alter_emb_add_emb_attributes_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableEmb := fmt.Sprintf("%s.t_%s_emb", db.GetSchema(isoId), tablePrefix)
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS emb_attributes JSONB DEFAULT NULL", tableEmb)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add emb_attributes column to table %s: %w", tableEmb, err)
	}

	if changed {
		m.logger.Info("Added emb_attributes column to embedding table",
			zap.String("table", tableEmb))
	} else {
		m.logger.Debug("emb_attributes column already exists in embedding table",
			zap.String("table", tableEmb))
	}

	return nil
}

// createEmbAttributesPathIndex creates the GIN index for emb_attributes using jsonb_path_ops
func (m *MigrationV0x19x0) createEmbAttributesPathIndex(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Creating emb_attributes path index",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_idx_emb_attributes_path_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableEmb := fmt.Sprintf("%s.t_%s_emb", db.GetSchema(isoId), tablePrefix)
	indexName := fmt.Sprintf("idx_%s_emb_attributes_path", tablePrefix)
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIN (emb_attributes jsonb_path_ops)", indexName, tableEmb)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to create emb_attributes path index %s on table %s: %w", indexName, tableEmb, err)
	}

	if changed {
		m.logger.Info("Created emb_attributes path index",
			zap.String("index", indexName),
			zap.String("table", tableEmb))
	} else {
		m.logger.Debug("emb_attributes path index already exists",
			zap.String("index", indexName),
			zap.String("table", tableEmb))
	}

	return nil
}

// createEmbAttributesOpsIndex creates the GIN index for emb_attributes using jsonb_ops
func (m *MigrationV0x19x0) createEmbAttributesOpsIndex(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Creating emb_attributes ops index",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_idx_emb_attributes_ops_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableEmb := fmt.Sprintf("%s.t_%s_emb", db.GetSchema(isoId), tablePrefix)
	indexName := fmt.Sprintf("idx_%s_emb_attributes_ops", tablePrefix)
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIN (emb_attributes jsonb_ops)", indexName, tableEmb)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to create emb_attributes ops index %s on table %s: %w", indexName, tableEmb, err)
	}

	if changed {
		m.logger.Info("Created emb_attributes ops index",
			zap.String("index", indexName),
			zap.String("table", tableEmb))
	} else {
		m.logger.Debug("emb_attributes ops index already exists",
			zap.String("index", indexName),
			zap.String("table", tableEmb))
	}

	return nil
}

// alterEmbTableAddAttributes adds the attributes JSONB column to the embedding table
func (m *MigrationV0x19x0) alterEmbTableAddAttributes(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Adding attributes column to embedding table",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_alter_emb_add_attributes_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableEmb := fmt.Sprintf("%s.t_%s_emb", db.GetSchema(isoId), tablePrefix)
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS attributes JSONB DEFAULT NULL", tableEmb)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add attributes column to table %s: %w", tableEmb, err)
	}

	if changed {
		m.logger.Info("Added attributes column to embedding table",
			zap.String("table", tableEmb))
	} else {
		m.logger.Debug("attributes column already exists in embedding table",
			zap.String("table", tableEmb))
	}

	return nil
}

// createAttributesPathIndex creates the GIN index for attributes using jsonb_path_ops
func (m *MigrationV0x19x0) createAttributesPathIndex(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Creating attributes path index",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_idx_attributes_path_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableEmb := fmt.Sprintf("%s.t_%s_emb", db.GetSchema(isoId), tablePrefix)
	indexName := fmt.Sprintf("idx_%s_attributes_path", tablePrefix)
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIN (attributes jsonb_path_ops)", indexName, tableEmb)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to create attributes path index %s on table %s: %w", indexName, tableEmb, err)
	}

	if changed {
		m.logger.Info("Created attributes path index",
			zap.String("index", indexName),
			zap.String("table", tableEmb))
	} else {
		m.logger.Debug("attributes path index already exists",
			zap.String("index", indexName),
			zap.String("table", tableEmb))
	}

	return nil
}

// createAttributesOpsIndex creates the GIN index for attributes using jsonb_ops
func (m *MigrationV0x19x0) createAttributesOpsIndex(ctx context.Context, isoId, colId, profileId, tablePrefix string) error {
	m.logger.Debug("Creating attributes ops index",
		zap.String("isolationID", isoId),
		zap.String("collectionID", colId),
		zap.String("profileID", profileId),
		zap.String("tablePrefix", tablePrefix))

	key := fmt.Sprintf("%s_idx_attributes_ops_%s_%s_%s", ConfigurationPrefixV0x19x0, isoId, colId, profileId)
	tableEmb := fmt.Sprintf("%s.t_%s_emb", db.GetSchema(isoId), tablePrefix)
	indexName := fmt.Sprintf("idx_%s_attributes_ops", tablePrefix)
	query := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s USING GIN (attributes jsonb_ops)", indexName, tableEmb)

	changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to create attributes ops index %s on table %s: %w", indexName, tableEmb, err)
	}

	if changed {
		m.logger.Info("Created attributes ops index",
			zap.String("index", indexName),
			zap.String("table", tableEmb))
	} else {
		m.logger.Debug("attributes ops index already exists",
			zap.String("index", indexName),
			zap.String("table", tableEmb))
	}

	return nil
}
