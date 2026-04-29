/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package sql

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"go.uber.org/zap"
)

var (
	// replicationCompletedCache stores the cached state of attribute replication completion
	replicationCompletedCache atomic.Bool
	// replicationCheckMutex ensures only one goroutine queries the database at a time
	replicationCheckMutex sync.Mutex
)

func GetVsConfiguration(database db.Database) (configs map[string]string, err error) {
	sqlQuery := "SELECT * FROM vector_store.configuration"
	rows, err := database.GetConn().Query(sqlQuery)
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

func UpsertConfiguration(logger *zap.Logger, database db.Database, key string, value string) (configs map[string]string, err error) {
	sqlQuery := "INSERT INTO vector_store.configuration (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2"
	res, err := database.GetConn().Exec(sqlQuery, key, value)
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
	return GetVsConfiguration(database)
}

func CheckConfigurationExists(database db.Database, tableName string) (bool, error) {
	sqlQuery := fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", tableName)
	rows, err := database.GetConn().Query(sqlQuery)
	if err != nil {
		if err.Error() == fmt.Sprintf("pq: relation \"%s\" does not exist", tableName) {
			return false, nil
		}
		return false, err
	}
	defer rows.Close()
	return rows.Next(), nil
}

func IsAttributeReplicationCompleted(database db.Database) bool {
	// Fast path: check if already cached as completed
	if replicationCompletedCache.Load() {
		return true
	}

	// Slow path: query database and cache result if true
	// Use mutex to ensure only one goroutine queries at a time
	replicationCheckMutex.Lock()
	defer replicationCheckMutex.Unlock()

	// Double-check after acquiring lock - another goroutine might have set it
	if replicationCompletedCache.Load() {
		return true
	}

	// Query database
	sqlQuery := "SELECT value FROM vector_store.configuration WHERE key = $1"
	rows, err := database.GetConn().Query(sqlQuery, "attribute_replication_v0.19.0_completed")
	if err != nil {
		// Error reading - assume not completed (fail-safe behavior)
		return false
	}
	defer rows.Close()

	var completedValue string
	if rows.Next() {
		err = rows.Scan(&completedValue)
		if err != nil {
			// Error scanning - assume not completed
			return false
		}
	} else {
		// Key doesn't exist - assume not completed
		return false
	}

	isCompleted := completedValue == "true"

	// Cache the result permanently if replication is completed
	// Once true, it never goes back to false
	if isCompleted {
		replicationCompletedCache.Store(true)
	}

	return isCompleted
}

// ResetAttributeReplicationCache resets the cached replication completion state
// This function should only be used in tests
func ResetAttributeReplicationCache() {
	replicationCompletedCache.Store(false)
}
