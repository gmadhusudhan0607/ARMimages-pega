/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"bytes"
	"encoding/json"
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

func setupPatchTestContext(t *testing.T, method, url string, body []byte, params gin.Params) (*httptest.ResponseRecorder, *gin.Context, *mocks.MockDatabase) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, url, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	mockDB := mocks.NewMockDb()
	c.Set(middleware.DBConnectionGeneric, mockDB)
	return recorder, c, mockDB
}

func TestPatchDocument_Returns404_WhenDocumentDoesNotExist(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]interface{}
	}{
		{
			name:    "status only",
			payload: map[string]interface{}{"status": "COMPLETED"},
		},
		{
			name:    "attributes only",
			payload: map[string]interface{}{"attributes": []map[string]interface{}{{"name": "key", "value": []string{"val"}, "type": "string"}}},
		},
		{
			name: "both status and attributes",
			payload: map[string]interface{}{
				"status":     "COMPLETED",
				"attributes": []map[string]interface{}{{"name": "key", "value": []string{"val"}, "type": "string"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			params := gin.Params{
				{Key: "isolationID", Value: "test-iso"},
				{Key: "collectionName", Value: "test-col"},
				{Key: "documentID", Value: "non-existent-doc"},
			}
			recorder, c, mockDB := setupPatchTestContext(t, http.MethodPatch, "/", bodyBytes, params)

			// Schema Load query — return the isolation + collection so handler passes the existence checks
			schemaRows := sqlmock.NewRows([]string{"iso_id", "col_id", "profile_id", "schema_name", "tables_prefix", "profile_status", "is_default_profile"}).
				AddRow("test-iso", "test-col", "prof-1", "vector_store_abc", "t_abc", "ACTIVE", true)
			mockDB.Mock.ExpectQuery("SELECT .+ FROM vector_store\\.schema_info").
				WithArgs("test-iso", "test-col").
				WillReturnRows(schemaRows)

			// BeginTx for the handler's transaction
			mockDB.Mock.ExpectBegin()

			// DocumentExists query — return no rows (document does not exist)
			existsRows := sqlmock.NewRows([]string{"exists"})
			mockDB.Mock.ExpectQuery("SELECT true FROM").
				WithArgs("non-existent-doc").
				WillReturnRows(existsRows)

			// Rollback expected since we return 404 before commit
			mockDB.Mock.ExpectRollback()

			PatchDocument(c)

			assert.Equal(t, http.StatusNotFound, recorder.Code)

			var body map[string]string
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			assert.Equal(t, "404", body["code"])
			assert.Contains(t, body["message"], "non-existent-doc")
			assert.Contains(t, body["message"], "not found")

			assert.NoError(t, mockDB.Mock.ExpectationsWereMet())
		})
	}
}

func TestPatchDocument_Returns200_WhenDocumentExists_StatusOnly(t *testing.T) {
	payload := map[string]interface{}{"status": "COMPLETED"}
	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	params := gin.Params{
		{Key: "isolationID", Value: "test-iso"},
		{Key: "collectionName", Value: "test-col"},
		{Key: "documentID", Value: "existing-doc"},
	}
	recorder, c, mockDB := setupPatchTestContext(t, http.MethodPatch, "/", bodyBytes, params)

	// Schema Load query
	schemaRows := sqlmock.NewRows([]string{"iso_id", "col_id", "profile_id", "schema_name", "tables_prefix", "profile_status", "is_default_profile"}).
		AddRow("test-iso", "test-col", "prof-1", "vector_store_abc", "t_abc", "ACTIVE", true)
	mockDB.Mock.ExpectQuery("SELECT .+ FROM vector_store\\.schema_info").
		WithArgs("test-iso", "test-col").
		WillReturnRows(schemaRows)

	// BeginTx
	mockDB.Mock.ExpectBegin()

	// DocumentExists — document exists
	existsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
	mockDB.Mock.ExpectQuery("SELECT true FROM").
		WithArgs("existing-doc").
		WillReturnRows(existsRows)

	// UpsertDocStatusTx is called on the MockDatabase directly (it's a no-op mock)
	// Commit
	mockDB.Mock.ExpectCommit()

	PatchDocument(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NoError(t, mockDB.Mock.ExpectationsWereMet())
}
