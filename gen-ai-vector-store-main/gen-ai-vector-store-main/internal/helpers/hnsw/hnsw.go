/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package hnsw

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.uber.org/zap"
)

const (
	// pgVectorHnswBuildMaintenanceMemoryMbEnvVarName is the environment variable name for the memory limit used during HNSW index build.
	pgVectorHnswBuildMaintenanceMemoryMbEnvVarName = "PGVECTOR_HNSW_BUILD_MAINTENANCE_MEMORY_MB"
)

type DBTX interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func BuildSetParametersQuery(ctx context.Context, logger *zap.Logger, dbtx DBTX, isoID string, colID string, profileID string) (string, error) {
	memoryParam, err := getMemoryParam(ctx, logger, dbtx, isoID, colID, profileID)
	if err != nil {
		return "", fmt.Errorf("failed to get memory parameter for isolation %s, collection %s, profile %s: %w", isoID, colID, profileID, err)
	}

	logger.Debug("postgres memory parameter", zap.String("memoryParam", memoryParam))

	maintanenceWorkers, workers := getWorkersParam()
	logger.Debug("postgres maintanence workers and workers", zap.Int("maintanenceWorkers", maintanenceWorkers), zap.Int("workers", workers))

	query := fmt.Sprintf(`
SET maintenance_work_mem = '%s';
SET max_parallel_maintenance_workers = %d;
SET max_parallel_workers = %d;`, memoryParam, maintanenceWorkers, workers)
	return query, nil
}

func BuildCreateIndexQuery(ctx context.Context, logger *zap.Logger, dbtx DBTX, isoID, colID, profileID string) (string, error) {
	mValue, efConstruction, err := getHNSWIndexParams(ctx, dbtx, isoID, profileID)
	if err != nil {
		return "", fmt.Errorf("failed to get HNSW index parameters for isolation %s, profile %s: %w", isoID, profileID, err)
	}
	logger.Debug("hnsw will be created with", zap.Int("m", mValue), zap.Int("ef_construction", efConstruction))

	// `CONCURRENTLY` cannot be used inside a transaction.
	concurrently := "CONCURRENTLY"
	if _, ok := dbtx.(*sql.Tx); ok {
		concurrently = ""
	}

	// create index query
	tableEmb := db.GetTableEmb(isoID, colID)
	_, tableEmbWithoutSchema := helpers.SplitTableName(tableEmb)
	idxName := GetIdxName(tableEmbWithoutSchema)
	query := fmt.Sprintf(`
CREATE INDEX %[5]s IF NOT EXISTS %[1]s 
ON %[2]s
USING hnsw (embedding vector_cosine_ops) WITH (m = %[3]d, ef_construction = %[4]d);`,
		idxName,
		tableEmb,
		mValue,
		efConstruction,
		concurrently,
	)

	return query, nil
}

func getMemoryParam(ctx context.Context, logger *zap.Logger, db DBTX, isoID, colID, profileID string) (string, error) {
	upperMemoryLimitsMb := int64(helpers.GetEnvOrDefaultInt64(pgVectorHnswBuildMaintenanceMemoryMbEnvVarName, 2*1024)) // 2GB by default, not to exhaust the system memory
	upperMemoryLimitBytes := int64(upperMemoryLimitsMb * 1024 * 1024)
	pgDefaultMemoryBytes := int64(64 * 1024 * 1024) // 64MB, default value for maintenance_work_mem in PostgreSQL

	vectorLen, err := getVectorLength(ctx, db, isoID, profileID)
	if err != nil {
		return "", fmt.Errorf("failed to get vector length for profile %s in isolation %s: %w", profileID, isoID, err)
	}

	vectorsNum, err := getVectorsNum(ctx, db, isoID, colID)
	if err != nil {
		return "", fmt.Errorf("failed to get vectors number for collection %s in isolation %s: %w", colID, isoID, err)
	}

	// https://github.com/pgvector/pgvector/issues/844
	approximatedIndexBuildSizeBytes := int64(float64(vectorsNum*(8+(vectorLen*4))) * 1.3)

	// if approximation less then default, use default value
	approximatedIndexBuildSizeBytes = max(approximatedIndexBuildSizeBytes, pgDefaultMemoryBytes)

	var workingMemoryApproximationBytes int64
	if approximatedIndexBuildSizeBytes < upperMemoryLimitBytes {
		workingMemoryApproximationBytes = approximatedIndexBuildSizeBytes
		logger.Debug("index build size approximation fits in memory limit", zap.Int64("workingMemoryApproximationBytes", workingMemoryApproximationBytes), zap.Int64("upperMemoryLimitBytes", upperMemoryLimitBytes))
	} else {
		logger.Warn("index build size approximation does not fit in memory limit, index build time may be increased", zap.Int64("workingMemoryApproximationBytes", workingMemoryApproximationBytes), zap.Int64("upperMemoryLimitBytes", upperMemoryLimitBytes))
		workingMemoryApproximationBytes = upperMemoryLimitBytes
	}

	workingMemoryApproximationMB := workingMemoryApproximationBytes / 1024 / 1024

	return fmt.Sprintf("%dMB", workingMemoryApproximationMB), nil
}

func getVectorLength(ctx context.Context, dbtx DBTX, isoID, profileID string) (int64, error) {
	vecLengthQuery := fmt.Sprintf("SELECT vector_len FROM %s WHERE profile_id = $1", db.GetTableEmbeddingProfiles(isoID))
	rows, err := dbtx.QueryContext(ctx, vecLengthQuery, profileID)
	if err != nil {
		return 0, fmt.Errorf("error while executing query [%s] for profile %s: %w", vecLengthQuery, profileID, err)
	}

	defer rows.Close()

	if !rows.Next() {
		return 0, fmt.Errorf("no vector length found for profile %s in isolation %s", profileID, isoID)
	}

	var vectorLength int64
	err = rows.Scan(&vectorLength)
	if err != nil {
		return 0, fmt.Errorf("error while reading row from query [%s] for profile %s and isolation %s: %w", vecLengthQuery, profileID, isoID, err)
	}

	return vectorLength, nil
}

func getVectorsNum(ctx context.Context, dbtx DBTX, isoID, colID string) (int64, error) {
	vectorsNumQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", db.GetTableEmb(isoID, colID))
	rows, err := dbtx.QueryContext(ctx, vectorsNumQuery)
	if err != nil {
		return 0, fmt.Errorf("error while executing query [%s] for iso %s and col %s: %w", vectorsNumQuery, isoID, colID, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, fmt.Errorf("no vectors found for collection %s in isolation %s", colID, isoID)
	}

	var vectorsNum int64
	err = rows.Scan(&vectorsNum)
	if err != nil {
		return 0, fmt.Errorf("error while reading row from query [%s] for isolation %s and collection %s: %w", vectorsNumQuery, isoID, colID, err)
	}

	return vectorsNum, nil
}

// https://github.com/pgvector/pgvector#index-build-time
func getWorkersParam() (maintenanceWorkers int, workers int) {
	maintenanceWorkers = 4

	defaultPgWorkers := 8

	return maintenanceWorkers, defaultPgWorkers
}

func getHNSWIndexParams(ctx context.Context, dbtx DBTX, isoID, profileID string) (mValue, efConstruction int, err error) {
	vectorLen, err := getVectorLength(ctx, dbtx, isoID, profileID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get vector length for profile %s in isolation %s: %w", profileID, isoID, err)
	}

	// Select the appropriate parameters based on vector dimension
	switch {
	case vectorLen >= 1 && vectorLen < 1000:
		return 16, 100, nil // "tiny"
	case vectorLen >= 1000 && vectorLen < 1500:
		return 24, 200, nil // "small"
	case vectorLen >= 1500 && vectorLen <= 3000:
		return 32, 350, nil // "medium"
	case vectorLen > 3000 && vectorLen <= 5000:
		return 48, 500, nil // "large"
	default:
		return 0, 0, fmt.Errorf("unsupported vector length %d: must be between 1 and 5000", vectorLen)
	}
}

func GetIdxName(tableName string) string {
	return fmt.Sprintf("idx_%s__hnsw", tableName)
}
