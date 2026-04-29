/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package sql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	queuesql "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/queue/sql"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/collections"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema/migrations"
	sqldo "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql/do"
	functions2 "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql/functions2"
	functions2metrics "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql/functions2/metrics"

	"go.uber.org/zap"
	"golang.org/x/mod/semver"
)

const (
	VsSchemaChangeCompleted = "completed"

	KeyVsSchemaVersion         = "schema_version"
	KeyVsSchemaVersionPrev     = "schema_version_prev"
	KeyPostgresVersion         = "postgres_version"
	KeyPostgresVersionAnalyzed = "postgres_version_analyzed"

	VsSchemaVersionDefault = "v0.14.0" // Assume this is the version of the DB schema if not found
	VsSchemaVersionLegacy  = "v0.15.0" // Legacy schema version before the migration manager was introduced. Do not change this value.

	V11CreateMissedCollectionsV2Table = "V11_create_missed-collections_v2_table"
	V12CreateDocMetaTable             = "V12_create_doc_meta_table"
	V12CreateEmbMetaTable             = "V12_create_emb_meta_table"

	V13CreateEmbeddingProfiles     = "V13_create_emb_profiles_table"
	V13AlterCollectionsTable       = "V13_alter_collections_table"
	V13CreateCollectionEmbProfiles = "V13_create_collection_emb_profiles_table"
	V13InitCollectionEmbProfiles   = "V13_init_collection_emb_profiles_table"

	V15UpdateEmbeddingProfiles = "V15_update_emb_profiles_table"
)

func SetupDatabaseForService(ctx context.Context, logger *zap.Logger, database db.Database) (err error) {

	// Check if the service runs in ReadOnly mode
	if helpers.IsReadOnlyMode() {
		logger.Info("ReadOnly mode enabled, skipping base tables creation")

		// Check if the configuration table exists
		for {
			exists, err := CheckConfigurationExists(database, "vector_store.configuration")
			if err != nil {
				logger.Warn("configuration table does not exist, waiting...", zap.Error(err))
				time.Sleep(15 * time.Second)
			}
			if exists {
				logger.Info("configuration table exists")
				break
			}
		}
	} else {
		if err := createBaseTables(ctx, logger, database); err != nil {
			return fmt.Errorf("failed to setup DB (createBaseTables): %w", err)
		}
	}

	// Wait until schema version is valid
	for ok := true; ok; ok = err != nil {
		err = validateSchemaVersion(database)
		if err != nil {
			logger.Warn("wait.... not supported db schema version", zap.Error(err))
			time.Sleep(5 * time.Second)
		}
	}
	return nil
}

func SetupDatabaseForOps(ctx context.Context, logger *zap.Logger, database db.Database) (err error) {

	// Check if the service runs in ReadOnly mode
	if helpers.IsReadOnlyMode() {
		logger.Info("ReadOnly mode enabled, skipping base tables creation")

		// Check if the configuration table exists
		for {
			exists, err := CheckConfigurationExists(database, "vector_store.configuration")
			if err != nil {
				logger.Warn("configuration table does not exist, waiting...", zap.Error(err))
				time.Sleep(15 * time.Second)
			}
			if exists {
				logger.Info("configuration table exists")
				break
			}
		}
	} else {
		if err := createBaseTables(ctx, logger, database); err != nil {
			return fmt.Errorf("failed to setup DB (createBaseTables): %w", err)
		}
	}

	// Wait until schema version is valid
	for ok := true; ok; ok = err != nil {
		err = validateSchemaVersion(database)
		if err != nil {
			logger.Warn("wait.... not supported db schema version", zap.Error(err))
			time.Sleep(5 * time.Second)
		}
	}
	if !helpers.IsReadOnlyMode() {
		if err := createMetricsFunctions(logger, database); err != nil {
			return fmt.Errorf("failed to setup DB (createMetricsFunctions): %w", err)
		}
	}

	logger.Info("successfully initialized DB")
	return nil
}

func SetupDatabaseForBackground(ctx context.Context, logger *zap.Logger, database db.Database) (err error) {
	if helpers.IsReadOnlyMode() {
		logger.Info("ReadOnly mode enabled, skipping database setup for background")
		return nil
	}
	if err = createBaseTables(ctx, logger, database); err != nil {
		return fmt.Errorf("failed to setup DB (createBaseTables): %w", err)
	}
	if err = applyDbChanges(ctx, logger, database); err != nil {
		return fmt.Errorf("failed to setup DB (applyDbChanges): %w", err)
	}
	if err = checkAndRunPostgreSQLAnalyze(ctx, logger, database); err != nil {
		return fmt.Errorf("failed to check and run PostgreSQL ANALYZE: %w", err)
	}
	logger.Info("successfully initialized DB")
	return nil
}

func applyDbChanges(ctx context.Context, logger *zap.Logger, database db.Database) (err error) {
	var config map[string]string
	config, err = GetVsConfiguration(database)
	if err != nil {
		return fmt.Errorf("failed to get VS Configuration: %w", err)
	}
	logger.Info("using schema version", zap.String("version", config[KeyVsSchemaVersion]))

	// Set previous schema version if not found
	if config[KeyVsSchemaVersionPrev] == "" {
		logger.Debug("schema version (prev) not found")
		config, err = UpsertConfiguration(logger, database, KeyVsSchemaVersionPrev, VsSchemaVersionDefault) //nolint:ineffassign,staticcheck
		if err != nil {
			return fmt.Errorf("failed to set %s configuration: %w", KeyVsSchemaVersionPrev, err)
		}
	}

	if err = installPgVectorPlugin(logger, database); err != nil {
		return fmt.Errorf("failed to install PgVectorPlugin: %w", err)
	}
	if err = createDbFunctions(logger, database); err != nil {
		return fmt.Errorf("failed to create/update DB Functions: %w", err)
	}
	if err = createMetricsFunctions(logger, database); err != nil {
		return fmt.Errorf("failed to create/update DB Metrics Functions: %w", err)
	}
	if err = applySchemaChanges(ctx, logger, database); err != nil {
		return fmt.Errorf("failed to apply DB schema changes: %w", err)
	}

	return nil
}

func installPgVectorPlugin(logger *zap.Logger, database db.Database) (err error) {
	if err = sqlExecute(logger, database, sqldo.Do_create_extension_vector); err != nil {
		return err
	}
	logger.Info("successfully installed PGVector plugin")
	return nil
}

func createBaseTables(ctx context.Context, logger *zap.Logger, database db.Database) (err error) {
	sqlQueries := []string{
		// V2 Tables
		sqldo.Do_create_schema_vector_store,
		sqldo.Do_create_table_isolations_v2,
		sqldo.Do_create_table_configuration_v2,

		// V2 Queue
		queuesql.Do_create_table,
		queuesql.Do_do_create_index,
	}
	for _, query := range sqlQueries {
		if err = sqlExecute(logger, database, query); err != nil {
			return err
		}
	}
	logger.Info("successfully created base tables")
	return nil
}

func createDbFunctions(logger *zap.Logger, database db.Database) (err error) {
	sqlQueries := []string{

		// SQL functions in Database (Vector Store schema) -
		functions2.Function_table_exists,
		functions2.Function_attributes_as_jsonb_by_ids,
		functions2.Function_embedding_statuses_as_json,
		functions2.Function_calculate_document_status,
		functions2.Function_drop_all_triggers_on_table,
		functions2.Function_drop_all_triggers_on_collection,
		functions2.Function_lookup_resources_metadata,
		functions2.Function_schema_info,
		functions2.Function_copy_schema,
		functions2.Function_get_collection_document_count,
		functions2.Function_get_db_metrics,
		functions2.Function_attribute_migration,

		// SQL functions in Database (Vector Store schema) - Metrics
		functions2metrics.Function_metrics_tables_size,
		functions2metrics.Function_metrics_document_count,
		functions2metrics.Function_metrics_last_modified_time,
		functions2metrics.Function_metrics_iso_size,

		// Queue2
		queuesql.Do_create_table,
		queuesql.Do_do_create_index,
		queuesql.Function_embeddings_queue_get,
		queuesql.Function_embeddings_queue_put,
		queuesql.Function_embeddings_queue_get_with_exception,
	}
	for _, query := range sqlQueries {
		if err = sqlExecute(logger, database, query); err != nil {
			return err
		}
	}
	logger.Info("successfully created core DB functions")
	return nil
}

func createMetricsFunctions(logger *zap.Logger, database db.Database) (err error) {
	sqlQueries := []string{
		// Functions for metrics retrieval
		functions2metrics.Function_metrics_tables_size,
		functions2metrics.Function_metrics_document_count,
		functions2metrics.Function_metrics_last_modified_time,
	}
	for _, query := range sqlQueries {
		if err = sqlExecute(logger, database, query); err != nil {
			return err
		}
	}
	logger.Info("successfully created metrics DB functions")
	return nil
}

func applySchemaChanges(ctx context.Context, logger *zap.Logger, database db.Database) (err error) {
	// Initialize migration manager
	migrationManager := migrations.NewMigrationManager(logger, database)

	// Get the current schema version from configuration
	config, err := GetVsConfiguration(database)
	if err != nil {
		return fmt.Errorf("failed to get VS Configuration: %w", err)
	}

	currentVersion := config[KeyVsSchemaVersion]
	logger.Info("loaded current schema version", zap.String("version", currentVersion))

	// For versions before v0.15.0, apply legacy migrations first
	if currentVersion == "" || semver.Compare(currentVersion, VsSchemaVersionLegacy) < 0 {
		// Apply legacy migrations for versions before v0.15.0
		err = applyLegacySchemaChanges(ctx, logger, database)
		if err != nil {
			return fmt.Errorf("failed to apply legacy schema changes: %w", err)
		}
	}

	// Run all pending migrations using the improved migration manager
	err = migrationManager.RunPendingMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply pending schema migrations: %w", err)
	}

	return nil
}

// applyLegacySchemaChanges contains all the legacy schema change logic for versions before v0.16.0
func applyLegacySchemaChanges(ctx context.Context, logger *zap.Logger, database db.Database) (err error) {
	// Fix for BUG-914465 - Create missed collections table for isolation
	err = createV2CollectionsTableIfMissed(ctx, logger, database)
	if err != nil {
		return fmt.Errorf("failed to create missed collections table: %w", err)
	}

	err = addMetadataTables(logger, database)
	if err != nil {
		return fmt.Errorf("failed to add metadata tables: %w", err)
	}

	err = addProfilesTables(logger, database)
	if err != nil {
		return fmt.Errorf("failed to add embedding profile tables: %w", err)
	}

	err = insertTextMultilingualEmbedding002EmbeddingProfile(logger, database)
	if err != nil {
		return fmt.Errorf("failed to add 'text-multilingual-embedding-002' embedding profile: %w", err)
	}

	return nil
}

func addProfilesTables(logger *zap.Logger, database db.Database) (err error) {
	ctx := context.Background()
	schemaMgr, err := schema.NewVsSchemaManager(database, logger).Load(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get schema manager: %w", err)
	}
	for _, iso := range schemaMgr.GetIsolations() {
		err = addEmbProfilesTable(logger, database, iso.ID)
		if err != nil {
			return fmt.Errorf("failed to create profiles table for %s: %w", iso.ID, err)
		}
		err = addCollectionsEmbeddingProfilesTable(logger, database, iso.ID)
		if err != nil {
			return fmt.Errorf("failed to create collection profiles table for '%s': %w", iso.ID, err)
		}
		err = alterCollectionsTable(logger, database, iso.ID)
		if err != nil {
			return fmt.Errorf("failed to alter collections table for %s: %w", iso.ID, err)
		}
	}

	isoIDs, err := getIsolationIDs(logger, database)
	if err != nil {
		return fmt.Errorf("failed to get isolation IDs: %w", err)
	}
	for _, isoId := range isoIDs {
		colIDs, err := getCollectionIDs(logger, database, isoId)
		if err != nil {
			return fmt.Errorf("failed to get collection IDs for isolation %s: %w", isoId, err)
		}
		for _, colId := range colIDs {
			err = initCollectionsEmbeddingProfilesTableForCollection(logger, database, isoId, colId)
			if err != nil {
				return fmt.Errorf("failed to initialize collection embedding profiles table for %s/%s: %w",
					isoId, colId, err)
			}
		}
	}

	return nil

}

func addEmbProfilesTable(logger *zap.Logger, database db.Database, isolationID string) (err error) {

	logger.Debug("creating embedding profile table", zap.String("isolationID", isolationID))
	key := fmt.Sprintf("%s__%s_create", V13CreateEmbeddingProfiles, isolationID)
	tableProfiles := db.GetTableEmbeddingProfiles(isolationID)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s
			(
                profile_id    VARCHAR(127) NOT NULL,
                provider_name VARCHAR(63)  NOT NULL,
                model_name    VARCHAR(127) NOT NULL,
                model_version VARCHAR(63)  NOT NULL,
                vector_len    INT          NOT NULL,
                max_tokens    INT          NOT NULL DEFAULT 0,
                status 		  VARCHAR(63),
                details   	  TEXT,
                PRIMARY KEY (profile_id),
                UNIQUE (profile_id, model_name, model_version, vector_len)
			)
		`, tableProfiles)
	changed, err := ExecuteOnce(logger, database, key, query)
	if err != nil {
		return fmt.Errorf("failed to create profiles table [%s]: %w", query, err)
	}
	if changed {
		logger.Info("created embedding profile table", zap.String("isolationID", isolationID))
	}

	logger.Debug("Initializing embedding profile table", zap.String("isolationID", isolationID))
	key = fmt.Sprintf("%s__%s_init", V13CreateEmbeddingProfiles, isolationID)
	query = fmt.Sprintf(`
		INSERT INTO %[1]s (profile_id, provider_name, model_name, model_version, vector_len, max_tokens)
		VALUES
            ('openai-text-embedding-ada-002', 'openai', 'text-embedding-ada-002', '2', 1536, 8191),
            ('openai-text-embedding-3-small', 'openai', 'text-embedding-3-small', '1', 1536, 8191),
            ('openai-text-embedding-3-large', 'openai', 'text-embedding-3-large', '1', 3072, 8191),
            ('amazon-titan-embed-text', 'amazon', 'titan-embed-text', '2', 0, 8192)
		ON CONFLICT (profile_id) DO NOTHING;
		`, tableProfiles)

	changed, err = ExecuteOnce(logger, database, key, query)
	if err != nil {
		return fmt.Errorf("failed to init profiles [%s]: %w", query, err)
	}
	if changed {
		logger.Info("initialized embedding profile table", zap.String("isolationID", isolationID))
	}
	return nil
}

func addCollectionsEmbeddingProfilesTable(logger *zap.Logger, database db.Database, isolationID string) (err error) {

	logger.Debug("creating embedding profile table", zap.String("isolationID", isolationID))
	key := fmt.Sprintf("%s__%s_create", V13CreateCollectionEmbProfiles, isolationID)
	tableCollectionProfiles := db.GetTableCollectionEmbeddingProfiles(isolationID)
	tableCollections := db.GetTableCollections(isolationID)
	tableProfiles := db.GetTableEmbeddingProfiles(isolationID)

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
		    col_id           TEXT REFERENCES %[2]s (col_id) ON DELETE CASCADE NOT NULL,
		    profile_id       VARCHAR(127) REFERENCES %[3]s (profile_id) ON DELETE CASCADE NOT NULL,
		    tables_prefix    VARCHAR(40) NOT NULL,
		    status           VARCHAR(63),
		    details          TEXT,
		    reated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		    PRIMARY KEY (col_id, profile_id),
		    UNIQUE (tables_prefix)
		    )
		`, tableCollectionProfiles, tableCollections, tableProfiles)

	changed, err := ExecuteOnce(logger, database, key, query)
	if err != nil {
		return fmt.Errorf("failed to create collection profiles table [%s]: %w", query, err)
	}
	if changed {
		logger.Info("created collection profiles table", zap.String("isolationID", isolationID))
	}

	return nil
}

func initCollectionsEmbeddingProfilesTableForCollection(logger *zap.Logger, database db.Database,
	isolationID, collectionID string) (err error) {

	key := fmt.Sprintf("%s__%s_%s", V13InitCollectionEmbProfiles, isolationID, collectionID)
	tableCollectionProfiles := db.GetTableCollectionEmbeddingProfiles(isolationID)
	tableCollections := db.GetTableCollections(isolationID)

	query := fmt.Sprintf(`
        INSERT INTO %[1]s (col_id, profile_id, tables_prefix, status)
        VALUES (
            $1,
            COALESCE(
              (SELECT default_emb_profile from %[2]s WHERE col_id = $1 ),
              $2),
            md5($1),
            $3)
		ON CONFLICT (col_id, profile_id) DO NOTHING
	`, tableCollectionProfiles, tableCollections)

	// The default profile is always ready to use when the collection is created.
	changed, err := ExecuteOnce(logger, database, key, query, collectionID,
		collections.DefaultEmbeddingProfileID, collections.EmbeddingProfileStatusReady)
	if err != nil {
		return fmt.Errorf("failed to init collection profiles table for '%s'/'%s' [%s]: %w",
			isolationID, collectionID, query, err)
	}
	if changed {
		logger.Info("initialized collection profiles table", zap.String("isolationID", isolationID), zap.String("collectionID", collectionID))
	}

	return nil
}

func alterCollectionsTable(logger *zap.Logger, database db.Database, isolationID string) (err error) {
	tableCollections := db.GetTableCollections(isolationID)
	tableProfiles := db.GetTableEmbeddingProfiles(isolationID)

	key := fmt.Sprintf("%s__%s_add_column", V13AlterCollectionsTable, isolationID)
	query := fmt.Sprintf(`
		ALTER TABLE %[1]s
			ADD COLUMN IF NOT EXISTS default_emb_profile VARCHAR(127)
				REFERENCES %[2]s (profile_id) ON DELETE CASCADE
		`, tableCollections, tableProfiles)
	changed, err := ExecuteOnce(logger, database, key, query)
	if err != nil {
		return fmt.Errorf("failed to add column to collections table: %w", err)
	}
	if changed {
		logger.Info("added column to collections table", zap.String("isolationID", isolationID))
	}

	key = fmt.Sprintf("%s__%s_insert_data", V13AlterCollectionsTable, isolationID)
	query = fmt.Sprintf(`
		UPDATE %[1]s
		  SET default_emb_profile = '%[2]s'
		WHERE default_emb_profile IS NULL
	`, tableCollections, collections.DefaultEmbeddingProfileID)

	changed, err = ExecuteOnce(logger, database, key, query)
	if err != nil {
		return fmt.Errorf("failed to insert default_emb_profile to collections table: %w", err)
	}
	if changed {
		logger.Info("inserted default_emb_profile to collections table", zap.String("isolationID", isolationID))
	}

	key = fmt.Sprintf("%s__%s_set_not_null", V13AlterCollectionsTable, isolationID)
	query = fmt.Sprintf(`
       ALTER TABLE  %[1]s
         ALTER COLUMN default_emb_profile SET NOT NULL
	`, tableCollections)

	changed, err = ExecuteOnce(logger, database, key, query)
	if err != nil {
		return fmt.Errorf("failed set 'not null to default_emb_profile in collections table: %w", err)
	}
	if changed {
		logger.Info("set 'not null to default_emb_profile in collections table", zap.String("isolationID", isolationID))
	}
	return nil
}

func addMetadataTables(logger *zap.Logger, database db.Database) (err error) {
	ctx := context.Background()
	schemaMgr, err := schema.NewVsSchemaManager(database, logger).Load(ctx, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get schema manager: %w", err)
	}
	for _, col := range schemaMgr.GetCollections() {
		err = createDocMetaTable(logger, database, col)
		if err != nil {
			return fmt.Errorf("failed to create documents metadata tables for %s / %s: %w",
				col.IsolationID, col.CollectionID, err)
		}
		err = createEmbMetaTable(logger, database, col)
		if err != nil {
			return fmt.Errorf("failed to create embeddings metadata tables for %s / %s: %w",
				col.IsolationID, col.CollectionID, err)
		}
	}
	return nil
}

func createDocMetaTable(logger *zap.Logger, database db.Database, c *schema.Collection) (err error) {
	logger.Debug("creating documents metadata tables", zap.String("isolationID", c.IsolationID), zap.String("collectionID", c.CollectionID))
	key := fmt.Sprintf("%s__%s_%s", V12CreateDocMetaTable, c.IsolationID, c.CollectionID)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s.%[2]s_doc_meta (
			doc_id           text references %[1]s.%[2]s_doc(doc_id) on delete cascade,
			metadata_key     text not null,
			metadata_value   text,
			modified_at      timestamp default CURRENT_TIMESTAMP not null,
			unique (doc_id, metadata_key),
			primary key (doc_id, metadata_key)
			)`, c.SchemaName, c.TablesPrefix)
	changed, err := ExecuteOnce(logger, database, key, query)
	if err != nil {
		return fmt.Errorf("failed to create documents metadata tables: %w", err)
	}
	if changed {
		logger.Info("created documents metadata tables", zap.String("isolationID", c.IsolationID), zap.String("collectionID", c.CollectionID))
	}
	return nil
}

func createEmbMetaTable(logger *zap.Logger, database db.Database, c *schema.Collection) (err error) {
	logger.Debug("creating embeddings metadata tables", zap.String("isolationID", c.IsolationID), zap.String("collectionID", c.CollectionID))
	key := fmt.Sprintf("%s__%s_%s", V12CreateEmbMetaTable, c.IsolationID, c.CollectionID)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s.%[2]s_emb_meta (
			emb_id           text references %[1]s.%[2]s_emb(emb_id) on delete cascade,
			metadata_key     text not null,
			metadata_value   text,
			modified_at      timestamp default CURRENT_TIMESTAMP not null,
			unique (emb_id, metadata_key),
			primary key (emb_id, metadata_key)
			)`, c.SchemaName, c.TablesPrefix)
	changed, err := ExecuteOnce(logger, database, key, query)
	if err != nil {
		return fmt.Errorf("failed to create embeddings metadata tables: %w", err)
	}
	if changed {
		logger.Info("created embeddings metadata tables", zap.String("isolationID", c.IsolationID), zap.String("collectionID", c.CollectionID))
	}
	return nil
}

func ExecuteOnce(logger *zap.Logger, database db.Database, configKey, query string, args ...interface{}) (bool, error) {
	defer func() {
		_ = logger.Sync()
	}()
	config, err := GetVsConfiguration(database)
	if err != nil {
		return false, fmt.Errorf("failed to get VS Configuration: %w", err)
	}
	if config[configKey] != VsSchemaChangeCompleted {
		paramsList := ""
		for i, arg := range args {
			paramsList += fmt.Sprintf(" %d: %+v", i, arg)
		}
		logger.Debug("configuration not completed", zap.String("configKey", configKey))
		if err = sqlExecute(logger, database, query, args...); err != nil {
			return false, fmt.Errorf("failed to execute query [%s] with params [%s]: %w", query, paramsList, err)
		}
		logger.Debug("set configuration to 'completed'", zap.String("configKey", configKey))
		query = `
			INSERT INTO vector_store.configuration (key, value)
			VALUES ($1, 'completed')
			ON CONFLICT (key) DO UPDATE SET value = 'completed'
        `
		if err = sqlExecute(logger, database, query, configKey); err != nil {
			return false, fmt.Errorf("failed to set configuration %s to 'completed': %w", configKey, err)
		}
		config, err = GetVsConfiguration(database)
		logger.Debug("reload configuration")
		if err != nil {
			return false, fmt.Errorf("failed to get updated VS Configuration: %w", err)
		}
		if config[configKey] != VsSchemaChangeCompleted {
			return false, fmt.Errorf("failed to complete %s", configKey)
		}
		logger.Debug("set to 'completed'", zap.String("configKey", configKey))
		return true, nil
	} else {
		logger.Debug("already completed", zap.String("configKey", configKey))
		return false, nil
	}
}

func validateSchemaVersion(database db.Database) error {
	// If there are any data in DB check if migration completed successfully (Expect a proper schema version)
	configs, err := GetVsConfiguration(database)
	if err != nil {
		return fmt.Errorf("failed to setup DB (GetVsConfiguration): %w", err)
	}

	schemaVersion := configs[KeyVsSchemaVersion]
	if schemaVersion == "" {
		schemaVersion = VsSchemaVersionDefault
	}
	// compatibility with legacy schema versions without 'v' prefix
	if !strings.HasPrefix(schemaVersion, "v") {
		schemaVersion = fmt.Sprintf("v%s", schemaVersion)
	}

	if !semver.IsValid(schemaVersion) {
		return fmt.Errorf("invalid schema version (actual version): %s", schemaVersion)
	}

	// Check if forced minimum version is set via environment variable
	var requiredVersion string
	forcedMinVersion := helpers.GetForcedMinRequiredSchemaVersion()
	if forcedMinVersion != "" {
		// Validate the forced version format
		if !semver.IsValid(forcedMinVersion) {
			return fmt.Errorf("invalid DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION format: %s", forcedMinVersion)
		}
		requiredVersion = forcedMinVersion
	} else {
		// Get the latest migration version dynamically from the migration registry
		// If no migrations are registered, use VsSchemaVersionDefault as fallback
		requiredVersion = migrations.DefaultRegistry.GetLatestVersionOrDefault(VsSchemaVersionDefault)
	}

	if semver.Compare(schemaVersion, requiredVersion) < 0 {
		return fmt.Errorf("schema version %s is less than required %s", schemaVersion, requiredVersion)
	}
	return nil
}

func sqlExecute(logger *zap.Logger, d db.Database, sqlQuery string, args ...interface{}) error {
	sqlQueryTxt := strings.ReplaceAll(sqlQuery, "\n", "\\n")
	logger.Debug("executing sql query", zap.String("query", sqlQueryTxt), zap.Any("args", args))
	_, err := d.GetConn().Exec(sqlQuery, args...)
	if err != nil {
		logger.Error("failed to execute sql query", zap.String("query", sqlQuery), zap.Error(err))
		return err
	}
	return nil
}

func createV2CollectionsTableIfMissed(ctx context.Context, logger *zap.Logger, database db.Database) (err error) {

	isoList, err := isolations.NewManager(database, logger).GetIsolations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get isolations: %w", err)
	}

	for _, iso := range isoList {
		key := fmt.Sprintf("%s__%s_schema", V11CreateMissedCollectionsV2Table, iso.ID)
		query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS vector_store_%s", db.GetMD5Hash(iso.ID))
		changed, err := ExecuteOnce(logger, database, key, query)
		if err != nil {
			return fmt.Errorf("failed to create missed collections schema for isolation '%s': %w", iso.ID, err)
		}
		if changed {
			logger.Info("checked/created missed collections schema for isolation", zap.String("isolationID", iso.ID))
		}

		query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
		    col_id           TEXT NOT NULL,
		    col_prefix       VARCHAR(40) NOT NULL UNIQUE,
		    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		    PRIMARY KEY (col_id)
		    )
		`, db.GetTableCollections(iso.ID))

		key = fmt.Sprintf("%s__%s", V11CreateMissedCollectionsV2Table, iso.ID)
		changed, err = ExecuteOnce(logger, database, key, query)
		if err != nil {
			return fmt.Errorf("failed to create missed collections table for isolation %s: %w", iso.ID, err)
		}
		if changed {
			logger.Info("checked/created missed collections table for isolation", zap.String("isolationID", iso.ID))
		}
	}
	return nil
}

func insertTextMultilingualEmbedding002EmbeddingProfile(logger *zap.Logger, database db.Database) (err error) {
	isolationIDs, err := getIsolationIDs(logger, database)
	if err != nil {
		return fmt.Errorf("failed to get isolation IDs: %w", err)
	}

	for _, isoId := range isolationIDs {
		tableProfiles := db.GetTableEmbeddingProfiles(isoId)
		key := fmt.Sprintf("%s__%s_%s", V15UpdateEmbeddingProfiles, isoId, "text-multilingual-embedding-002")
		query := fmt.Sprintf(`
			INSERT INTO %[1]s (profile_id, provider_name, model_name, model_version, vector_len, max_tokens)
			VALUES
				('google-text-multilingual-embedding-002', 'google', 'text-multilingual-embedding-002', '2', 768, 20000)
			ON CONFLICT (profile_id) DO NOTHING;
			`, tableProfiles)

		changed, err := ExecuteOnce(logger, database, key, query)
		if err != nil {
			return fmt.Errorf("failed to add 'text-multilingual-embedding-002' profile [%s]: %w", query, err)
		}
		if changed {
			logger.Info("added 'text-multilingual-embedding-002' profile", zap.String("isolationID", isoId))
		}
	}

	return nil
}

func getIsolationIDs(logger *zap.Logger, database db.Database) (isoIDs []string, err error) {
	query := "SELECT iso_id from vector_store.isolations"
	rows, err := database.GetConn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("error while executing query [%s]: %w", query, err)
	}
	defer rows.Close()

	for rows.Next() {
		var isoID string
		err = rows.Scan(&isoID)
		if err != nil {
			return nil, fmt.Errorf("error while reading rows from query: [%s]: %w", query, err)
		}
		isoIDs = append(isoIDs, isoID)
	}
	return isoIDs, nil
}

func getCollectionIDs(logger *zap.Logger, database db.Database, isoID string) (colIDs []string, err error) {
	schemaName := db.GetSchema(isoID)
	query := fmt.Sprintf("SELECT col_id from %s.collections", schemaName)
	rows, err := database.GetConn().Query(query)
	if err != nil {
		return nil, fmt.Errorf("error while executing query [%s]: %w", query, err)
	}
	defer rows.Close()

	for rows.Next() {
		var colID string
		err = rows.Scan(&colID)
		if err != nil {
			return nil, fmt.Errorf("error while reading rows from query: [%s]: %w", query, err)
		}
		colIDs = append(colIDs, colID)
	}
	return colIDs, nil
}

// checkAndRunPostgreSQLAnalyze checks if PostgreSQL version has changed and runs ANALYZE if needed
// to rebuild database statistics after PostgreSQL upgrade. ANALYZE runs at most once per PostgreSQL version.
// The DatabaseEngineVersion from DBInstance SCE is used to trigger pod restarts, but the actual ANALYZE
// decision is based on the detected PostgreSQL version from the database.
func checkAndRunPostgreSQLAnalyze(ctx context.Context, logger *zap.Logger, database db.Database) error {
	logger.Info("Checking PostgreSQL version for ANALYZE requirement")

	// Get DatabaseEngineVersion from environment (set by DBInstance SCE) - this triggers pod restart
	databaseEngineVersion := helpers.GetEnvOrDefault("DATABASE_ENGINE_VERSION", "")
	if databaseEngineVersion != "" {
		logger.Info("DatabaseEngineVersion from DBInstance", zap.String("engineVersion", databaseEngineVersion))
	}

	// Get current PostgreSQL version from database
	currentPgVersion, err := getPostgreSQLVersion(logger, database)
	if err != nil {
		return fmt.Errorf("failed to get PostgreSQL version: %w", err)
	}
	logger.Info("Current PostgreSQL version", zap.String("version", currentPgVersion))

	// Get stored configuration
	config, err := GetVsConfiguration(database)
	if err != nil {
		return fmt.Errorf("failed to get VS Configuration: %w", err)
	}

	storedPgVersion := config[KeyPostgresVersion]
	lastAnalyzedVersion := config[KeyPostgresVersionAnalyzed]

	logger.Info("PostgreSQL version check",
		zap.String("currentVersion", currentPgVersion),
		zap.String("storedVersion", storedPgVersion),
		zap.String("lastAnalyzedVersion", lastAnalyzedVersion))

	// Determine if ANALYZE should run based on PostgreSQL version
	shouldRunAnalyze := false
	reason := ""

	if storedPgVersion == "" {
		// First time setup - store version without running ANALYZE
		logger.Info("First time PostgreSQL version detection, storing version without running ANALYZE")
		_, err = UpsertConfiguration(logger, database, KeyPostgresVersion, currentPgVersion)
		if err != nil {
			return fmt.Errorf("failed to store PostgreSQL version: %w", err)
		}
		return nil
	} else if currentPgVersion != lastAnalyzedVersion {
		// PostgreSQL version changed since last ANALYZE
		shouldRunAnalyze = true
		reason = fmt.Sprintf("PostgreSQL version changed from %s to %s", lastAnalyzedVersion, currentPgVersion)
	}

	if shouldRunAnalyze {
		logger.Info("Running ANALYZE to rebuild database statistics", zap.String("reason", reason))

		// Run ANALYZE with standard sampling
		query := `
			SET default_statistics_target=100;
			ANALYZE;
		`
		if err = sqlExecute(logger, database, query); err != nil {
			return fmt.Errorf("failed to execute ANALYZE: %w", err)
		}

		logger.Info("Successfully completed ANALYZE")

		// Update stored versions
		_, err = UpsertConfiguration(logger, database, KeyPostgresVersion, currentPgVersion)
		if err != nil {
			return fmt.Errorf("failed to update PostgreSQL version: %w", err)
		}

		_, err = UpsertConfiguration(logger, database, KeyPostgresVersionAnalyzed, currentPgVersion)
		if err != nil {
			return fmt.Errorf("failed to update analyzed PostgreSQL version: %w", err)
		}

		logger.Info("Updated PostgreSQL version tracking",
			zap.String("currentVersion", currentPgVersion),
			zap.String("analyzedVersion", currentPgVersion))
	} else {
		logger.Info("No ANALYZE required - already completed for current PostgreSQL version")
	}

	return nil
}

// getPostgreSQLVersion retrieves the current PostgreSQL version string
func getPostgreSQLVersion(logger *zap.Logger, database db.Database) (string, error) {
	query := "SELECT version()"
	rows, err := database.GetConn().Query(query)
	if err != nil {
		logger.Error("failed to query PostgreSQL version", zap.Error(err))
		return "", fmt.Errorf("failed to query PostgreSQL version: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		var version string
		err = rows.Scan(&version)
		if err != nil {
			logger.Error("failed to scan PostgreSQL version", zap.Error(err))
			return "", fmt.Errorf("failed to scan PostgreSQL version: %w", err)
		}
		return version, nil
	}

	return "", fmt.Errorf("no PostgreSQL version returned")
}
