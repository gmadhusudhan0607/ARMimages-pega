/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockDbDefault creates a MockDatabase without the CustomTypeConvertor so
// that nil arguments (used by schema.Load for collectionID) are handled by
// the default database/sql converter.
func newMockDbDefault(t *testing.T) *mocks.MockDatabase {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	return &mocks.MockDatabase{
		SqlDB: sqlDB,
		Mock:  mock,
	}
}

func setupPutDocTestContext(t *testing.T, body []byte, params gin.Params) (*httptest.ResponseRecorder, *gin.Context, *mocks.MockDatabase) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	mockDB := newMockDbDefault(t)
	c.Set(middleware.DBConnectionGeneric, mockDB)
	c.Set(middleware.DBConnectionIngest, mockDB)
	return recorder, c, mockDB
}

// expectSchemaAndCollection mocks the ensureIsolation schema query and the
// CollectionExists query that run before the empty-chunks path.
func expectSchemaAndCollection(mockDB *mocks.MockDatabase, isoID, colName string) {
	// ensureIsolation → schema.NewVsSchemaManager.Load(ctx, isolationID, nil)
	schemaRows := sqlmock.NewRows(
		[]string{"iso_id", "col_id", "profile_id", "schema_name", "tables_prefix", "profile_status", "is_default_profile"},
	).AddRow(isoID, colName, "prof-1", "vector_store_abc", "t_abc", "ACTIVE", true)
	mockDB.Mock.ExpectQuery("SELECT .+ FROM vector_store\\.schema_info").
		WithArgs(isoID, nil).
		WillReturnRows(schemaRows)

	// collections.CollectionExists → SELECT count(*) from <schema>.collections
	colRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mockDB.Mock.ExpectQuery("SELECT count").
		WithArgs(colName).
		WillReturnRows(colRows)
}

var putDocParams = gin.Params{
	{Key: "isolationID", Value: "test-iso"},
	{Key: "collectionName", Value: "test-col"},
}

func TestPutDocument_EmptyChunks_NoAttributes_Returns202(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"id":     "doc-1",
		"chunks": []interface{}{},
	})
	require.NoError(t, err)

	_, c, mockDB := setupPutDocTestContext(t, body, putDocParams)
	expectSchemaAndCollection(mockDB, "test-iso", "test-col")

	// Transaction: begin → (UpsertDocStatusTx is MockDatabase no-op) → commit
	mockDB.Mock.ExpectBegin()
	mockDB.Mock.ExpectCommit()

	PutDocument(c)

	assert.Equal(t, http.StatusAccepted, c.Writer.Status())
	assert.NoError(t, mockDB.Mock.ExpectationsWereMet(),
		"transaction must be started and committed even with zero attributes")
}

func TestPutDocument_EmptyChunks_WithAttributes_Returns202(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"id":     "doc-1",
		"chunks": []interface{}{},
		"attributes": []map[string]interface{}{
			{"name": "category", "value": []string{"test"}, "type": "string"},
		},
	})
	require.NoError(t, err)

	_, c, mockDB := setupPutDocTestContext(t, body, putDocParams)
	expectSchemaAndCollection(mockDB, "test-iso", "test-col")

	// Transaction: begin → attr upsert → attr id select → commit
	mockDB.Mock.ExpectBegin()

	// UpsertAttributes2 → upsertAttributeItems: INSERT INTO <attr_table> ... ON CONFLICT
	mockDB.Mock.ExpectExec("INSERT INTO .+_attr").
		WithArgs("category", "string", "test").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// UpsertAttributes2 → getAttributeItemIds2: SELECT attr_id FROM <attr_table>
	attrIDRows := sqlmock.NewRows([]string{"attr_id"}).AddRow(int64(42))
	mockDB.Mock.ExpectQuery("SELECT attr_id FROM").
		WillReturnRows(attrIDRows)

	// UpdateDocAttributesTx is MockDatabase no-op
	mockDB.Mock.ExpectCommit()

	PutDocument(c)

	assert.Equal(t, http.StatusAccepted, c.Writer.Status())
	assert.NoError(t, mockDB.Mock.ExpectationsWereMet(),
		"attributes must be persisted within the same transaction as the status upsert")
}

func TestPutDocument_EmptyChunks_BeginTxFails_Returns500(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"id":     "doc-1",
		"chunks": []interface{}{},
	})
	require.NoError(t, err)

	recorder, c, mockDB := setupPutDocTestContext(t, body, putDocParams)
	expectSchemaAndCollection(mockDB, "test-iso", "test-col")

	mockDB.Mock.ExpectBegin().WillReturnError(fmt.Errorf("connection lost"))

	PutDocument(c)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)

	var respBody map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &respBody))
	assert.Equal(t, "500", respBody["code"])
	assert.Contains(t, respBody["message"], "failed to begin transaction")
	assert.NoError(t, mockDB.Mock.ExpectationsWereMet())
}

func TestPutDocument_EmptyChunks_CommitFails_Returns500AndRollback(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"id":     "doc-1",
		"chunks": []interface{}{},
	})
	require.NoError(t, err)

	recorder, c, mockDB := setupPutDocTestContext(t, body, putDocParams)
	expectSchemaAndCollection(mockDB, "test-iso", "test-col")

	mockDB.Mock.ExpectBegin()
	// UpsertDocStatusTx is MockDatabase no-op
	mockDB.Mock.ExpectCommit().WillReturnError(fmt.Errorf("disk full"))
	// Note: database/sql marks the tx as done after Commit() even on failure,
	// so the deferred Rollback() gets sql.ErrTxDone before reaching sqlmock.

	PutDocument(c)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)

	var respBody map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &respBody))
	assert.Equal(t, "500", respBody["code"])
	assert.Contains(t, respBody["message"], "failed to commit")
	assert.NoError(t, mockDB.Mock.ExpectationsWereMet())
}

func TestPutDocument_EmptyChunks_AttributeUpsertFails_Returns500AndRollback(t *testing.T) {
	body, err := json.Marshal(map[string]interface{}{
		"id":     "doc-1",
		"chunks": []interface{}{},
		"attributes": []map[string]interface{}{
			{"name": "category", "value": []string{"test"}, "type": "string"},
		},
	})
	require.NoError(t, err)

	recorder, c, mockDB := setupPutDocTestContext(t, body, putDocParams)
	expectSchemaAndCollection(mockDB, "test-iso", "test-col")

	mockDB.Mock.ExpectBegin()
	// UpsertDocStatusTx is MockDatabase no-op (placeholder is already persisted before attribute step)

	// Attribute INSERT fails mid-transaction
	mockDB.Mock.ExpectExec("INSERT INTO .+_attr").
		WithArgs("category", "string", "test").
		WillReturnError(fmt.Errorf("constraint violation"))

	// Handler must rollback (leaving no partial writes from the failed attribute insert)
	mockDB.Mock.ExpectRollback()

	PutDocument(c)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)

	var respBody map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &respBody))
	assert.Equal(t, "500", respBody["code"])
	assert.Contains(t, respBody["message"], "failed to upsert attributes")
	assert.NoError(t, mockDB.Mock.ExpectationsWereMet(),
		"transaction must be rolled back after attribute upsert failure")
}
