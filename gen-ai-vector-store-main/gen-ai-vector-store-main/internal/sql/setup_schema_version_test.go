// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package sql

import (
	"context"
	"os"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db/mocks"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema/migrations"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestValidateSchemaVersion_WithForcedMinVersion(t *testing.T) {
	// Setup test registry with a higher version
	testRegistry := migrations.NewMigrationRegistry()
	testMigration := &mockMigration{version: "v1.0.0"}
	_ = testRegistry.Register(testMigration)

	// Save original registry and replace it
	originalRegistry := migrations.DefaultRegistry
	migrations.DefaultRegistry = testRegistry
	defer func() {
		migrations.DefaultRegistry = originalRegistry
	}()

	tests := []struct {
		name                 string
		currentSchemaVersion string
		forcedMinVersion     string
		expectError          bool
		errorContains        string
	}{
		{
			name:                 "Schema at forced minimum - should pass",
			currentSchemaVersion: "v0.16.0",
			forcedMinVersion:     "v0.16.0",
			expectError:          false,
		},
		{
			name:                 "Schema above forced minimum - should pass",
			currentSchemaVersion: "v0.18.0",
			forcedMinVersion:     "v0.16.0",
			expectError:          false,
		},
		{
			name:                 "Schema below forced minimum - should fail",
			currentSchemaVersion: "v0.15.0",
			forcedMinVersion:     "v0.16.0",
			expectError:          true,
			errorContains:        "less than required",
		},
		{
			name:                 "Invalid forced version format - should fail",
			currentSchemaVersion: "v0.16.0",
			forcedMinVersion:     "invalid-version",
			expectError:          true,
			errorContains:        "invalid DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION format",
		},
		{
			name:                 "No forced version - uses latest from registry",
			currentSchemaVersion: "v1.0.0",
			forcedMinVersion:     "",
			expectError:          false,
		},
		{
			name:                 "No forced version - schema below latest - should fail",
			currentSchemaVersion: "v0.16.0",
			forcedMinVersion:     "",
			expectError:          true,
			errorContains:        "less than required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.forcedMinVersion != "" {
				os.Setenv("DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION", tt.forcedMinVersion)
			} else {
				os.Unsetenv("DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION")
			}
			defer os.Unsetenv("DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION")

			// Create mock database using sqlmock
			mockDB := mocks.NewMockDb()
			defer mockDB.SqlDB.Close()

			// Mock configuration query to return current schema version
			// Note: GetVsConfiguration uses "SELECT * FROM..." not "SELECT key, value FROM..."
			rows := mockDB.Mock.NewRows([]string{"key", "value"}).
				AddRow("schema_version", tt.currentSchemaVersion)

			mockDB.Mock.ExpectQuery("SELECT \\* FROM vector_store.configuration").
				WillReturnRows(rows)

			// Execute validation
			err := validateSchemaVersion(mockDB)

			// Verify expectations
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			if err := mockDB.Mock.ExpectationsWereMet(); err != nil && !tt.expectError {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

// mockMigration is a simple mock implementation of the Migration interface for testing
type mockMigration struct {
	version string
}

func (m *mockMigration) Version() string {
	return m.version
}

func (m *mockMigration) SourceVersion() string {
	return "v0.0.0"
}

func (m *mockMigration) Description() string {
	return "test-migration"
}

func (m *mockMigration) Apply(ctx context.Context, logger *zap.Logger, database db.Database) error {
	return nil
}

func (m *mockMigration) Dependencies() []string {
	return []string{}
}
