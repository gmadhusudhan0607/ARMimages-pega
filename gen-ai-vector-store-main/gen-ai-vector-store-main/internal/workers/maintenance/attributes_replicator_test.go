// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package maintenance

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestNewAttributesReplicator(t *testing.T) {
	mockDB := &MockDatabase{}
	worker := NewAttributesReplicator(mockDB)

	assert.NotNil(t, worker)
	assert.Equal(t, DefaultBatchSize, worker.batchSize)
	assert.Equal(t, DefaultDelayMs, worker.batchDelayMs)
	assert.Equal(t, DefaultIterationDelaySec, worker.iterationDelayS)
	assert.Equal(t, attributesReplicatorLogger, worker.logger)
	assert.Equal(t, mockDB, worker.database)
}

func TestReplicationUnit_structure(t *testing.T) {
	unit := ReplicationUnit{
		IsolationID:      "test-iso",
		CollectionID:     "test-col",
		ProfileID:        "test-profile",
		TablePrefix:      "abc123",
		TotalRecords:     1000,
		ProcessedRecords: 300,
		RemainingRecords: 700,
		Status:           statusInProgress,
	}

	assert.Equal(t, "test-iso", unit.IsolationID)
	assert.Equal(t, "test-col", unit.CollectionID)
	assert.Equal(t, "test-profile", unit.ProfileID)
	assert.Equal(t, "abc123", unit.TablePrefix)
	assert.Equal(t, int64(1000), unit.TotalRecords)
	assert.Equal(t, int64(300), unit.ProcessedRecords)
	assert.Equal(t, int64(700), unit.RemainingRecords)
	assert.Equal(t, statusInProgress, unit.Status)
}

func TestUnitTableInfo_structure(t *testing.T) {
	table := UnitTableInfo{
		TableName:        "vector_store_test.t_abc123_doc",
		SourceColumn:     "attr_ids",
		TargetColumn:     "doc_attributes",
		AttrTable:        "vector_store_test.t_abc123_attr",
		TotalRecords:     500,
		ProcessedRecords: 200,
		RemainingRecords: 300,
	}

	assert.Equal(t, "vector_store_test.t_abc123_doc", table.TableName)
	assert.Equal(t, "attr_ids", table.SourceColumn)
	assert.Equal(t, "doc_attributes", table.TargetColumn)
	assert.Equal(t, "vector_store_test.t_abc123_attr", table.AttrTable)
	assert.Equal(t, int64(500), table.TotalRecords)
	assert.Equal(t, int64(200), table.ProcessedRecords)
	assert.Equal(t, int64(300), table.RemainingRecords)
}

func TestAttributesReplicator_getUnitConfigKey(t *testing.T) {
	mockDB := &MockDatabase{}
	worker := NewAttributesReplicator(mockDB)

	key := worker.getUnitConfigKey("test-iso", "test-col", "test-profile")
	expected := "attribute_replication_v0.19.0_test-iso_test-col_test-profile"

	assert.Equal(t, expected, key)
}

func TestAttributesReplicator_logIterationProgress(t *testing.T) {
	mockDB := &MockDatabase{}
	worker := NewAttributesReplicator(mockDB)

	units := []ReplicationUnit{
		{
			IsolationID:      "iso1",
			CollectionID:     "col1",
			ProfileID:        "prof1",
			TotalRecords:     1000,
			ProcessedRecords: 800,
			RemainingRecords: 200,
			Status:           statusInProgress,
		},
		{
			IsolationID:      "iso2",
			CollectionID:     "col2",
			ProfileID:        "prof2",
			TotalRecords:     500,
			ProcessedRecords: 500,
			RemainingRecords: 0,
			Status:           statusCompleted,
		},
	}

	// This should not panic and should log progress
	worker.logIterationProgress(units, 1)
}

func TestAttributesReplicator_logBatchProgress(t *testing.T) {
	mockDB := &MockDatabase{}
	worker := NewAttributesReplicator(mockDB)

	table := UnitTableInfo{
		TableName:        "vector_store_test.t_abc123_doc",
		SourceColumn:     "attr_ids",
		TargetColumn:     "doc_attributes",
		TotalRecords:     1000,
		ProcessedRecords: 300,
		RemainingRecords: 700,
	}

	unit := &ReplicationUnit{
		IsolationID:      "test-iso",
		CollectionID:     "test-col",
		ProfileID:        "test-profile",
		TotalRecords:     2000,
		ProcessedRecords: 600,
		RemainingRecords: 1400,
	}

	allUnits := []ReplicationUnit{*unit}

	// This should not panic and should log batch progress
	worker.logBatchProgress(table, unit, allUnits)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "attribute_replication_v0.19.0", ConfigKeyPrefix)
	assert.Equal(t, "completed", statusCompleted)
	assert.Equal(t, "in_progress", statusInProgress)
	assert.Equal(t, "failed", statusFailed)
	assert.Equal(t, 1000, DefaultBatchSize)
	assert.Equal(t, 250, DefaultDelayMs)
	assert.Equal(t, 300, DefaultIterationDelaySec)
}

// Mock implementations for testing

type MockDatabase struct {
	mockConn *MockSQLDB
}

func (m *MockDatabase) GetConn() db.SQLDB {
	if m.mockConn == nil {
		m.mockConn = &MockSQLDB{}
	}
	return m.mockConn
}

func (m *MockDatabase) GetSingleConn(ctx context.Context) (*sql.Conn, error) {
	return nil, nil
}

func (m *MockDatabase) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockDatabase) UpsertDocStatus(ctx context.Context, table, docID, status, e string) error {
	return nil
}

func (m *MockDatabase) UpsertDocStatusTx(ctx context.Context, tx *sql.Tx, table, docID, status, e string) error {
	return nil
}

func (m *MockDatabase) UpdateDocAttributesTx(ctx context.Context, tx *sql.Tx, table, docID string, attrIDs []int64, docAttributes interface{}) error {
	return nil
}

func (m *MockDatabase) UpdateEmbAttributesTx(ctx context.Context, tx *sql.Tx, table, docID string, attrIDs, attrIDs2 []int64) error {
	return nil
}

func (m *MockDatabase) DeleteDocMetadataTx(ctx context.Context, tx *sql.Tx, tableDocMeta, docID string) error {
	return nil
}

func (m *MockDatabase) InsertDocMetadataTx(ctx context.Context, tx *sql.Tx, tableDocMeta, docID, metadataKey, metadataValue string) error {
	return nil
}

type MockSQLDB struct{}

func (m *MockSQLDB) Begin() (*sql.Tx, error) {
	return nil, nil
}

func (m *MockSQLDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return nil, nil
}

func (m *MockSQLDB) Close() error {
	return nil
}

func (m *MockSQLDB) Query(query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockSQLDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}

func (m *MockSQLDB) Exec(query string, arguments ...any) (sql.Result, error) {
	return nil, nil
}

func (m *MockSQLDB) ExecContext(ctx context.Context, query string, arguments ...any) (sql.Result, error) {
	return nil, nil
}

func (m *MockSQLDB) Conn(ctx context.Context) (*sql.Conn, error) {
	return nil, nil
}
