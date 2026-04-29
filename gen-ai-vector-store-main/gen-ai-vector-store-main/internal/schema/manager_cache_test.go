/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package schema

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db/mocks"
// 	"github.com/DATA-DOG/go-sqlmock"
// 	"github.com/stretchr/testify/assert"
// )

// func TestVsSchemaManager_Cache(t *testing.T) {
// 	db, mock, err := sqlmock.New()
// 	assert.NoError(t, err)
// 	defer db.Close()

// 	mockDB := &mocks.MockDatabase{SqlDB: db, Mock: mock}
// 	mgr := &vsSchemaManager{
// 		database:   mockDB,
// 		isolations: make(map[string]*Isolation),
// 	}
// 	ctx := context.Background()
// 	CacheTTL = 100 * time.Millisecond

// 	// Setup expected query and result
// 	mock.ExpectQuery(`SELECT COALESCE\(iso_id, ''\)`).WillReturnRows(
// 		sqlmock.NewRows([]string{"iso_id", "col_id", "profile_id", "schema_name", "tables_prefix", "profile_status", "is_default_profile"}).
// 			AddRow("iso1", "col1", "prof1", "schema1", "prefix1", "status1", true),
// 	)

// 	// First load should call DB
// 	_, err = mgr.Load(ctx, "iso1", "col1")
// 	assert.NoError(t, err)
// 	assert.NotNil(t, mgr.isolations["iso1"])
// 	assert.Equal(t, 1, len(mgr.isolations["iso1"].Collections))

// 	// Second load within TTL should use cache (no DB call expected)
// 	_, err = mgr.Load(ctx, "iso1", "col1")
// 	assert.NoError(t, err)
// 	assert.NotNil(t, mgr.isolations["iso1"])

// 	// Wait for cache to expire
// 	time.Sleep(120 * time.Millisecond)
// 	mock.ExpectQuery(`SELECT COALESCE\(iso_id, ''\)`).WillReturnRows(
// 		sqlmock.NewRows([]string{"iso_id", "col_id", "profile_id", "schema_name", "tables_prefix", "profile_status", "is_default_profile"}).
// 			AddRow("iso1", "col1", "prof1", "schema1", "prefix1", "status1", true),
// 	)
// 	_, err = mgr.Load(ctx, "iso1", "col1")
// 	assert.NoError(t, err)
// 	assert.NotNil(t, mgr.isolations["iso1"])
// }
