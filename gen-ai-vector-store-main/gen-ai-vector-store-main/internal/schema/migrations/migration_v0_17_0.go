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
	"go.uber.org/zap"
)

const (
	ConfigurationPrefixV0x17x0 = "schema_migration_v0.17.0"
	VersionV0x17x0             = "v0.17.0"
)

type MigrationV0x17x0 struct {
	*BaseMigration

	// Runtime dependencies - set during execution
	logger   *zap.Logger
	database db.Database
}

func init() {
	RegisterMigration(NewMigrationV0x17x0())
}

func NewMigrationV0x17x0() *MigrationV0x17x0 {
	migration := &MigrationV0x17x0{
		BaseMigration: NewBaseMigration(
			VersionV0x17x0,
			"Migration to v0.17.0 - Process chunks in background.",
			nil,            // Will be set below
			VersionV0x16x2, // Source version (explicitly indicates migration from v0.16.0 to v0.17.0)
		),
	}
	// Set the apply function to be the method on this struct
	migration.BaseMigration.applyFunc = migration.Apply
	return migration
}

func (m *MigrationV0x17x0) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
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
			err = m.createDocProcessingTable(ctx, database, isoId, colId)
			if err != nil {
				return err
			}

			err = m.createEmbProcessingTable(ctx, database, isoId, colId)
			if err != nil {
				return err
			}

			// TODO: EPIC-103866 / US-682862:
			// err = addEmbStatisticsTable(logger, database)
			// if err != nil {
			// 	return fmt.Errorf("failed to add stats tables: %w", err)
			// }

			err = m.fixAttrIds2Index(ctx, isoId, colId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MigrationV0x17x0) createDocProcessingTable(ctx context.Context, database db.Database, isolationID, collectionID string) (err error) {
	m.logger.Debug(
		"creating doc processing table",
		zap.String("isolationID", isolationID),
		zap.String("collectionID", collectionID),
	)
	key := fmt.Sprintf("%s_create_table_doc_processing_%s_%s", ConfigurationPrefixV0x17x0, isolationID, collectionID)

	tableDocProcessing := db.GetTableDocProcessing(isolationID, collectionID)
	tableDoc := db.GetTableDoc(isolationID, collectionID)
	_, tableDocProcessingWithoutSchema := helpers.SplitTableName(tableDocProcessing)

	// create index on attr_ids
	idxName := fmt.Sprintf("idx_%s__attrids", tableDocProcessingWithoutSchema)

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
			doc_id             TEXT REFERENCES %[2]s (doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
			created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			heartbeat          TIMESTAMP,
			record_timestamp   TIMESTAMP,
			error_message      TEXT,
			retry_count        INTEGER,
		    attr_ids       BIGINT[],
            doc_metadata       JSONB,
			file               BYTEA,
			PRIMARY KEY (doc_id)
		);

		CREATE INDEX IF NOT EXISTS %[3]s ON %[1]s USING GIN (attr_ids) WHERE attr_ids IS NOT NULL;
		`, tableDocProcessing, tableDoc, idxName)
	changed, err := ExecuteOnce(ctx, m.logger, database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add doc processing table: %w", err)
	}
	if changed {
		m.logger.Info(
			"added doc processing table",
			zap.String("isolationID", isolationID),
			zap.String("collectionID", collectionID),
		)
	}
	return nil
}

func (m *MigrationV0x17x0) createEmbProcessingTable(ctx context.Context, database db.Database, isolationID, collectionID string) (err error) {
	m.logger.Debug(
		"creating emb processing table",
		zap.String("isolationID", isolationID),
		zap.String("collectionID", collectionID),
	)
	key := fmt.Sprintf("%s_create_table_emb_processing_%s_%s", ConfigurationPrefixV0x17x0, isolationID, collectionID)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
			emb_id             TEXT NOT NULL,
			doc_id             TEXT REFERENCES %[2]s (doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
			content            TEXT,
			embedding          vector,
			status             TEXT,
			attr_ids		   BIGINT[],
			metadata		   JSONB,
			response_code      INT DEFAULT 0,
			error_message      TEXT,
			retry_count        INTEGER,
			start_time         TIMESTAMP,
			end_time           TIMESTAMP,
			record_timestamp   TIMESTAMP,
			token_count        INTEGER,
			PRIMARY KEY (emb_id)
		)`,
		db.GetTableEmbProcessing(isolationID, collectionID),
		db.GetTableDocProcessing(isolationID, collectionID),
	)

	changed, err := ExecuteOnce(ctx, m.logger, database.GetConn(), key, query)
	if err != nil {
		return fmt.Errorf("failed to add emb processing table: %w", err)
	}
	if changed {
		m.logger.Info(
			"added emb processing table",
			zap.String("isolationID", isolationID),
			zap.String("collectionID", collectionID),
		)
	}
	return nil
}

// In previous versions, the attr_ids2 index could be created with a condition that might have been incorrect
// Namely, with wrong column in `WHERE` clause - `gin (attr_ids2) WHERE attr_ids IS NOT NULL`
func (m *MigrationV0x17x0) fixAttrIds2Index(ctx context.Context, isolationID, collectionID string) (err error) {
	m.logger.Debug(
		"fixing attr_ids2 index",
		zap.String("isolationID", isolationID),
		zap.String("collectionID", collectionID),
	)

	tableEmb := db.GetTableEmb(isolationID, collectionID)
	schema, tableEmbWithoutSchema := helpers.SplitTableName(tableEmb)

	idxName := fmt.Sprintf("idx_%s__attrids2", tableEmbWithoutSchema)

	checkQuery := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1
			FROM pg_index i
			JOIN pg_class idx ON idx.oid = i.indexrelid
			JOIN pg_class tbl ON tbl.oid = i.indrelid
			JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
			WHERE ns.nspname || '.' || tbl.relname = '%s'
			AND idx.relname = '%s'
			AND pg_get_expr(i.indpred, i.indrelid) LIKE '%%attr_ids IS NOT NULL%%'
		)
	`, tableEmb, idxName)

	var incorrectIndexExists bool
	rows, err := m.database.Query(ctx, checkQuery)
	if err != nil {
		return fmt.Errorf("failed to check if incorrect index exists: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err = rows.Scan(&incorrectIndexExists); err != nil {
			return fmt.Errorf("failed to scan result of index check: %w", err)
		}
	}

	if incorrectIndexExists {
		m.logger.Info(
			"found incorrectly created index with condition 'attr_ids IS NOT NULL', recreating with proper condition",
			zap.String("indexName", idxName),
			zap.String("table", tableEmb),
		)

		key := fmt.Sprintf("%s_fix_index_attrids2_%s_%s", ConfigurationPrefixV0x17x0, isolationID, collectionID)

		// Drop and recreate the index with the correct condition
		query := fmt.Sprintf(`
			DROP INDEX IF EXISTS %[1]s.%[2]s;
			CREATE INDEX %[2]s ON %[3]s USING GIN (attr_ids2) WHERE attr_ids2 IS NOT NULL;
		`, schema, idxName, tableEmb)

		changed, err := ExecuteOnce(ctx, m.logger, m.database.GetConn(), key, query)
		if err != nil {
			return fmt.Errorf("failed to fix attr_ids2 index: %w", err)
		}

		if changed {
			m.logger.Info(
				"fixed attr_ids2 index",
				zap.String("isolationID", isolationID),
				zap.String("collectionID", collectionID),
			)
		}
	} else {
		m.logger.Debug(
			"index appears to be correct or doesn't exist",
			zap.String("indexName", idxName),
			zap.String("table", tableEmb),
		)
	}

	return nil
}

// TODO: EPIC-103866 / US-682862:
// func addEmbStatisticsTable(logger *zap.SugaredLogger, database db.Database) error {
// 	isoIDs, err := getIsolationIDs(logger, database)
// 	if err != nil {
// 		return err
// 	}

// 	for _, isoID := range isoIDs {
// 		colIDs, err := getCollectionIDs(logger, database, isoID)
// 		if err != nil {
// 			return err
// 		}

// 		for _, colID := range colIDs {
// 			key := fmt.Sprintf("%s__%s_%s", V16AddEmbStatisticsTable, isoID, colID)
// 			query := fmt.Sprintf(`
// 				CREATE TABLE IF NOT EXISTS %[1]s (
// 						doc_id               TEXT REFERENCES %[2]s (doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
// 						emb_id               TEXT REFERENCES %[3]s (emb_id) ON DELETE CASCADE ON UPDATE CASCADE,
// 						retry_count          INT,
// 						token_count          INT,
// 						start_time           TIMESTAMP,
// 						end_time             TIMESTAMP,
// 						last_llm_duration_ms INT,
// 						last_error_message   TEXT
// 					)`,
// 				db.GetTableEmbStatistics(isoID, colID),
// 				db.GetTableDoc(isoID, colID),
// 				db.GetTableEmb(isoID, colID),
// 			)

// 			changed, err := ExecuteOnce(logger, database, key, query)
// 			if err != nil {
// 				return fmt.Errorf("failed to add emb stats table [%s]: %w", query, err)
// 			}
// 			if changed {
// 				logger.Infof("added emb stats table for iso %q col %q", isoID, colID)
// 			}
// 		}
// 	}

// 	return nil
// }
