/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
)

// Configuration constants
const (
	VsSchemaChangeCompleted = "completed"
)

type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// GetIsolationIDs retrieves all isolation IDs from the database
func GetIsolationIDs(ctx context.Context, logger *zap.Logger, database DB) (isoIDs []string, err error) {
	query := "SELECT iso_id from vector_store.isolations"
	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		logger.Error("error while executing query", zap.String("query", query), zap.Error(err))
		return nil, fmt.Errorf("error while executing query [%s]: %w", query, err)
	}
	defer rows.Close()

	for rows.Next() {
		var isoID string
		err = rows.Scan(&isoID)
		if err != nil {
			logger.Error("error while reading rows from query", zap.String("query", query), zap.Error(err))
			return nil, fmt.Errorf("error while reading rows from query: [%s]: %w", query, err)
		}
		isoIDs = append(isoIDs, isoID)
	}
	return isoIDs, nil
}

// GetCollectionIDs retrieves all collection IDs for a given isolation from the database
func GetCollectionIDs(ctx context.Context, logger *zap.Logger, database DB, isoID string) (colIDs []string, err error) {
	schemaName := db.GetSchema(isoID)
	query := fmt.Sprintf("SELECT col_id from %s.collections", schemaName)
	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		logger.Error("error while executing query", zap.String("query", query), zap.Error(err))
		return nil, fmt.Errorf("error while executing query [%s]: %w", query, err)
	}
	defer rows.Close()

	for rows.Next() {
		var colID string
		err = rows.Scan(&colID)
		if err != nil {
			logger.Error("error while reading rows from query", zap.String("query", query), zap.Error(err))
			return nil, fmt.Errorf("error while reading rows from query: [%s]: %w", query, err)
		}
		colIDs = append(colIDs, colID)
	}
	return colIDs, nil
}

// GetVsConfiguration retrieves all configuration key-value pairs from the database
func GetVsConfiguration(ctx context.Context, database DB) (configs map[string]string, err error) {
	sqlQuery := "SELECT * FROM vector_store.configuration"
	rows, err := database.QueryContext(ctx, sqlQuery)
	if err != nil || rows == nil {
		return nil, err
	}
	defer rows.Close()

	configs = make(map[string]string)
	for rows.Next() {
		key := ""
		value := ""
		err = rows.Scan(&key, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row while running GetVsConfiguration: %w", err)
		}
		configs[key] = value
	}
	return configs, nil
}

// UpsertConfiguration updates or inserts a configuration key-value pair in the database
func UpsertConfiguration(ctx context.Context, logger *zap.Logger, database DB, key string, value string) (configs map[string]string, err error) {
	sqlQuery := "INSERT INTO vector_store.configuration (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2"
	res, err := database.ExecContext(ctx, sqlQuery, key, value)
	if err != nil {
		logger.Error("failed to execute sql query", zap.String("query", sqlQuery), zap.String("key", key), zap.String("value", value), zap.Error(err))
		return nil, fmt.Errorf("failed to execute sql query [%s] (key=%s, value=%s) : %w", sqlQuery, key, value, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		logger.Error("failed to get rows affected", zap.Error(err))
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		logger.Info("updated configuration", zap.String("key", key), zap.String("value", value))
	}
	return GetVsConfiguration(ctx, database)
}

// ExecuteOnce executes a query only once and marks it as completed in the configuration
func ExecuteOnce(ctx context.Context, logger *zap.Logger, database DB, configKey, query string, args ...interface{}) (bool, error) {
	config, err := GetVsConfiguration(ctx, database)
	if err != nil {
		logger.Error("failed to get VS Configuration", zap.Error(err))
		return false, fmt.Errorf("failed to get VS Configuration: %w", err)
	}

	if config[configKey] != VsSchemaChangeCompleted {
		paramsList := ""
		for i, arg := range args {
			paramsList += fmt.Sprintf(" %d: %+v", i, arg)
		}
		logger.Debug("configuration not completed", zap.String("configKey", configKey))

		// Execute the SQL query
		_, err := database.ExecContext(ctx, query, args...)
		if err != nil {
			logger.Error("failed to execute query", zap.String("query", query), zap.String("params", paramsList), zap.Error(err))
			return false, fmt.Errorf("failed to execute query [%s] with params [%s]: %w", query, paramsList, err)
		}

		logger.Debug("set configuration to 'completed'", zap.String("configKey", configKey))
		markQuery := `
			INSERT INTO vector_store.configuration (key, value)
			VALUES ($1, 'completed')
			ON CONFLICT (key) DO UPDATE SET value = 'completed'
		`
		_, err = database.ExecContext(ctx, markQuery, configKey)
		if err != nil {
			logger.Error("failed to set configuration to 'completed'", zap.String("configKey", configKey), zap.Error(err))
			return false, fmt.Errorf("failed to set configuration %s to 'completed': %w", configKey, err)
		}

		config, err = GetVsConfiguration(ctx, database)
		logger.Debug("reload configuration")
		if err != nil {
			logger.Error("failed to get updated VS Configuration", zap.Error(err))
			return false, fmt.Errorf("failed to get updated VS Configuration: %w", err)
		}
		if config[configKey] != VsSchemaChangeCompleted {
			logger.Error("failed to complete configuration", zap.String("configKey", configKey))
			return false, fmt.Errorf("failed to complete %s", configKey)
		}
		logger.Debug("set to 'completed'", zap.String("configKey", configKey))
		return true, nil
	} else {
		logger.Debug("already completed", zap.String("configKey", configKey))
		return false, nil
	}
}

// GetConfiguration retrieves a specific configuration value by key from the database
func GetConfiguration(ctx context.Context, logger *zap.Logger, database DB, key string) (string, error) {
	configs, err := GetVsConfiguration(ctx, database)
	if err != nil {
		logger.Error("failed to get configurations", zap.Error(err))
		return "", fmt.Errorf("failed to get configurations: %w", err)
	}

	value, exists := configs[key]
	if !exists {
		logger.Error("configuration key not found", zap.String("key", key))
		return "", fmt.Errorf("configuration key '%s' not found", key)
	}

	logger.Debug("Retrieved configuration", zap.String("key", key), zap.String("value", value))
	return value, nil
}

// GetCollectionProfiles retrieves all profile IDs for a given collection in a specific isolation
func GetCollectionProfiles(ctx context.Context, logger *zap.Logger, database DB, isoID string, colID string) (profileIDs []string, err error) {
	// Get the collection_emb_profiles table for this isolation
	collectionEmbProfilesTable := db.GetTableCollectionEmbeddingProfiles(isoID)
	query := fmt.Sprintf("SELECT profile_id FROM %s WHERE col_id = $1", collectionEmbProfilesTable)

	rows, err := database.QueryContext(ctx, query, colID)
	if err != nil {
		logger.Error("error while executing query", zap.String("query", query), zap.Error(err))
		return nil, fmt.Errorf("error while executing query [%s] for collection %s: %w", query, colID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var profileID string
		err = rows.Scan(&profileID)
		if err != nil {
			logger.Error("error while reading rows from query", zap.String("query", query), zap.Error(err))
			return nil, fmt.Errorf("error while reading rows from query [%s] for collection %s: %w", query, colID, err)
		}
		profileIDs = append(profileIDs, profileID)
	}

	if err = rows.Err(); err != nil {
		logger.Error("error after iterating rows for collection", zap.String("collectionID", colID), zap.Error(err))
		return nil, fmt.Errorf("error after iterating rows for collection %s: %w", colID, err)
	}

	logger.Debug("Found profiles for collection", zap.Int("count", len(profileIDs)), zap.String("collectionID", colID), zap.String("isolationID", isoID))
	return profileIDs, nil
}

// GetCollectionProfileTablesPrefix retrieves the table prefix for a given profile ID
func GetCollectionProfileTablesPrefix(ctx context.Context, logger *zap.Logger, database DB, isoID string, collectionID string, profileID string) (string, error) {
	// Get the collection_emb_profiles table for this isolation
	collectionEmbProfilesTable := db.GetTableCollectionEmbeddingProfiles(isoID)
	query := fmt.Sprintf("SELECT tables_prefix FROM %s WHERE profile_id = $1 AND col_id = $2", collectionEmbProfilesTable)

	rows, err := database.QueryContext(ctx, query, profileID, collectionID)
	if err != nil {
		logger.Error("error while executing query", zap.String("query", query), zap.Error(err))
		return "", fmt.Errorf("error while executing query [%s] for profile %s: %w", query, profileID, err)
	}
	defer rows.Close()

	if rows.Next() {
		var tablesPrefix string
		err = rows.Scan(&tablesPrefix)
		if err != nil {
			logger.Error("error while reading row from query", zap.String("query", query), zap.Error(err))
			return "", fmt.Errorf("error while reading row from query [%s] for profile %s and collection %s: %w", query, profileID, collectionID, err)
		}

		if err = rows.Err(); err != nil {
			return "", fmt.Errorf("error after reading row for profile %s and collection %s: %w", profileID, collectionID, err)
		}

		return tablesPrefix, nil
	}

	return "", fmt.Errorf("no tables prefix found for profile %s in isolation %s", profileID, isoID)
}
