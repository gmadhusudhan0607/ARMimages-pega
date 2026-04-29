// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package sql

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestGetPostgreSQLVersion tests getPostgreSQLVersion with database mocking
func TestGetPostgreSQLVersion(t *testing.T) {
	logger := zap.NewNop()

	t.Run("successful version retrieval", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		expectedVersion := "PostgreSQL 17.2"

		rows := sqlmock.NewRows([]string{"version"}).
			AddRow(expectedVersion)

		mock.ExpectQuery("SELECT version\\(\\)").WillReturnRows(rows)

		version, err := getPostgreSQLVersion(logger, mockDB)

		assert.NoError(t, err)
		assert.Equal(t, expectedVersion, version)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		expectedErr := errors.New("connection failed")
		mock.ExpectQuery("SELECT version\\(\\)").WillReturnError(expectedErr)

		version, err := getPostgreSQLVersion(logger, mockDB)

		assert.Error(t, err)
		assert.Empty(t, version)
		assert.Contains(t, err.Error(), "failed to query PostgreSQL version")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows returned", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		rows := sqlmock.NewRows([]string{"version"})
		mock.ExpectQuery("SELECT version\\(\\)").WillReturnRows(rows)

		version, err := getPostgreSQLVersion(logger, mockDB)

		assert.Error(t, err)
		assert.Empty(t, version)
		assert.Contains(t, err.Error(), "no PostgreSQL version returned")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		rows := sqlmock.NewRows([]string{"version"}).
			AddRow(nil) // This will cause a scan error

		mock.ExpectQuery("SELECT version\\(\\)").WillReturnRows(rows)

		version, err := getPostgreSQLVersion(logger, mockDB)

		assert.Error(t, err)
		assert.Empty(t, version)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestCheckAndRunPostgreSQLAnalyze tests checkAndRunPostgreSQLAnalyze with database mocking
func TestCheckAndRunPostgreSQLAnalyze(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("first time - stores version only", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		// Optional: Set DATABASE_ENGINE_VERSION for logging
		os.Setenv("DATABASE_ENGINE_VERSION", "17.2")
		defer os.Unsetenv("DATABASE_ENGINE_VERSION")

		// Mock getPostgreSQLVersion
		pgVersion := "PostgreSQL 17.0 on x86_64-pc-linux-gnu"
		versionRows := sqlmock.NewRows([]string{"version"}).AddRow(pgVersion)
		mock.ExpectQuery("SELECT version\\(\\)").WillReturnRows(versionRows)

		// Mock GetVsConfiguration - empty config (first time)
		configRows := sqlmock.NewRows([]string{"key", "value"})
		mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows)

		// Mock UpsertConfiguration for postgres_version
		mock.ExpectExec("INSERT INTO vector_store.configuration").
			WithArgs("postgres_version", pgVersion).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock GetVsConfiguration again (after upsert)
		configRows2 := sqlmock.NewRows([]string{"key", "value"}).
			AddRow("postgres_version", pgVersion)
		mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows2)

		err := checkAndRunPostgreSQLAnalyze(ctx, logger, mockDB)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("version changed - runs ANALYZE automatically", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		// Set DATABASE_ENGINE_VERSION (triggers pod restart in real scenario)
		os.Setenv("DATABASE_ENGINE_VERSION", "17.2")
		defer os.Unsetenv("DATABASE_ENGINE_VERSION")

		oldVersion := "PostgreSQL 14.10 on x86_64-pc-linux-gnu"
		newVersion := "PostgreSQL 17.2 on x86_64-pc-linux-gnu"

		// Mock getPostgreSQLVersion
		versionRows := sqlmock.NewRows([]string{"version"}).AddRow(newVersion)
		mock.ExpectQuery("SELECT version\\(\\)").WillReturnRows(versionRows)

		// Mock GetVsConfiguration - has old analyzed version
		configRows := sqlmock.NewRows([]string{"key", "value"}).
			AddRow("postgres_version", oldVersion).
			AddRow("postgres_version_analyzed", oldVersion)
		mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows)

		// Mock ANALYZE execution
		mock.ExpectExec("SET default_statistics_target=100").WillReturnResult(sqlmock.NewResult(0, 0))

		// Mock UpsertConfiguration for postgres_version
		mock.ExpectExec("INSERT INTO vector_store.configuration").
			WithArgs("postgres_version", newVersion).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock GetVsConfiguration after first upsert
		configRows2 := sqlmock.NewRows([]string{"key", "value"}).
			AddRow("postgres_version", newVersion).
			AddRow("postgres_version_analyzed", oldVersion)
		mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows2)

		// Mock UpsertConfiguration for postgres_version_analyzed
		mock.ExpectExec("INSERT INTO vector_store.configuration").
			WithArgs("postgres_version_analyzed", newVersion).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock GetVsConfiguration after second upsert
		configRows3 := sqlmock.NewRows([]string{"key", "value"}).
			AddRow("postgres_version", newVersion).
			AddRow("postgres_version_analyzed", newVersion)
		mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows3)

		err := checkAndRunPostgreSQLAnalyze(ctx, logger, mockDB)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("same version - skips ANALYZE", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		pgVersion := "PostgreSQL 17.2 on x86_64-pc-linux-gnu"

		// Mock getPostgreSQLVersion
		versionRows := sqlmock.NewRows([]string{"version"}).AddRow(pgVersion)
		mock.ExpectQuery("SELECT version\\(\\)").WillReturnRows(versionRows)

		// Mock GetVsConfiguration - already analyzed for this version
		configRows := sqlmock.NewRows([]string{"key", "value"}).
			AddRow("postgres_version", pgVersion).
			AddRow("postgres_version_analyzed", pgVersion)
		mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows)

		// No ANALYZE should be executed
		// No UpsertConfiguration should be called

		err := checkAndRunPostgreSQLAnalyze(ctx, logger, mockDB)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("multiple version upgrades", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		// Simulate upgrade from 14 -> 17 -> 18
		versions := []string{
			"PostgreSQL 14.10 on x86_64-pc-linux-gnu",
			"PostgreSQL 17.2 on x86_64-pc-linux-gnu",
			"PostgreSQL 18.0 on x86_64-pc-linux-gnu",
		}

		for i := 1; i < len(versions); i++ {
			oldVer := versions[i-1]
			newVer := versions[i]

			// Mock getPostgreSQLVersion
			versionRows := sqlmock.NewRows([]string{"version"}).AddRow(newVer)
			mock.ExpectQuery("SELECT version\\(\\)").WillReturnRows(versionRows)

			// Mock GetVsConfiguration
			configRows := sqlmock.NewRows([]string{"key", "value"}).
				AddRow("postgres_version", oldVer).
				AddRow("postgres_version_analyzed", oldVer)
			mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows)

			// Mock ANALYZE execution
			mock.ExpectExec("SET default_statistics_target=100").WillReturnResult(sqlmock.NewResult(0, 0))

			// Mock UpsertConfiguration for postgres_version
			mock.ExpectExec("INSERT INTO vector_store.configuration").
				WithArgs("postgres_version", newVer).
				WillReturnResult(sqlmock.NewResult(1, 1))

			// Mock GetVsConfiguration after first upsert
			configRows2 := sqlmock.NewRows([]string{"key", "value"}).
				AddRow("postgres_version", newVer).
				AddRow("postgres_version_analyzed", oldVer)
			mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows2)

			// Mock UpsertConfiguration for postgres_version_analyzed
			mock.ExpectExec("INSERT INTO vector_store.configuration").
				WithArgs("postgres_version_analyzed", newVer).
				WillReturnResult(sqlmock.NewResult(1, 1))

			// Mock GetVsConfiguration after second upsert
			configRows3 := sqlmock.NewRows([]string{"key", "value"}).
				AddRow("postgres_version", newVer).
				AddRow("postgres_version_analyzed", newVer)
			mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").WillReturnRows(configRows3)

			err := checkAndRunPostgreSQLAnalyze(ctx, logger, mockDB)
			assert.NoError(t, err)
		}

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error during version query", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		// Mock getPostgreSQLVersion with error
		mock.ExpectQuery("SELECT version\\(\\)").WillReturnError(errors.New("connection failed"))

		err := checkAndRunPostgreSQLAnalyze(ctx, logger, mockDB)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get PostgreSQL version")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error during config retrieval", func(t *testing.T) {
		mockDB := mocks.NewMockDb()
		defer mockDB.SqlDB.Close()
		mock := mockDB.Mock

		pgVersion := "PostgreSQL 17.0 on x86_64-pc-linux-gnu"

		// Mock getPostgreSQLVersion
		versionRows := sqlmock.NewRows([]string{"version"}).AddRow(pgVersion)
		mock.ExpectQuery("SELECT version\\(\\)").WillReturnRows(versionRows)

		// Mock GetVsConfiguration with error
		mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").
			WillReturnError(errors.New("config table not found"))

		err := checkAndRunPostgreSQLAnalyze(ctx, logger, mockDB)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get VS Configuration")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestConfigurationKeys tests the configuration key constants
func TestConfigurationKeys(t *testing.T) {
	assert.Equal(t, "postgres_version", KeyPostgresVersion)
	assert.Equal(t, "postgres_version_analyzed", KeyPostgresVersionAnalyzed)
	assert.NotEqual(t, KeyPostgresVersion, KeyPostgresVersionAnalyzed)
}
