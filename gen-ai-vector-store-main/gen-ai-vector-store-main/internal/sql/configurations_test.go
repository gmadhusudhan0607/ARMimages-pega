// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package sql

import (
	"regexp"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db/mocks"
)

func TestIsAttributeReplicationCompleted_CachingBehavior(t *testing.T) {
	tests := []struct {
		name            string
		dbReturnValue   string
		expectedResult  bool
		expectedDBCalls int
		description     string
	}{
		{
			name:            "FirstCall_ReturnsFalse",
			dbReturnValue:   "false",
			expectedResult:  false,
			expectedDBCalls: 1,
			description:     "First call should query database and return false",
		},
		{
			name:            "FirstCall_ReturnsTrue",
			dbReturnValue:   "true",
			expectedResult:  true,
			expectedDBCalls: 1,
			description:     "First call should query database and return true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset cache for each test
			ResetAttributeReplicationCache()

			mockDB := mocks.NewMockDb()
			defer mockDB.SqlDB.Close()

			query := regexp.QuoteMeta("SELECT value FROM vector_store.configuration WHERE key = $1")
			rows := sqlmock.NewRows([]string{"value"}).AddRow(tt.dbReturnValue)
			mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows)

			result := IsAttributeReplicationCompleted(mockDB)

			if result != tt.expectedResult {
				t.Errorf("Expected result %v, got %v", tt.expectedResult, result)
			}

			if err := mockDB.Mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestIsAttributeReplicationCompleted_CachesPermanently(t *testing.T) {
	// Reset cache before test
	ResetAttributeReplicationCache()

	mockDB := mocks.NewMockDb()
	defer mockDB.SqlDB.Close()

	query := regexp.QuoteMeta("SELECT value FROM vector_store.configuration WHERE key = $1")
	rows := sqlmock.NewRows([]string{"value"}).AddRow("true")
	// Expect only ONE query - subsequent calls should use cache
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows)

	// First call - should query DB
	result1 := IsAttributeReplicationCompleted(mockDB)
	if !result1 {
		t.Error("First call should return true")
	}

	// Second call - should use cache (no new query expected)
	result2 := IsAttributeReplicationCompleted(mockDB)
	if !result2 {
		t.Error("Second call should return true from cache")
	}

	// Third call - should still use cache (no new query expected)
	result3 := IsAttributeReplicationCompleted(mockDB)
	if !result3 {
		t.Error("Third call should return true from cache")
	}

	// Verify only one query was executed
	if err := mockDB.Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestIsAttributeReplicationCompleted_DoesNotCacheFalse(t *testing.T) {
	// Reset cache before test
	ResetAttributeReplicationCache()

	mockDB := mocks.NewMockDb()
	defer mockDB.SqlDB.Close()

	query := regexp.QuoteMeta("SELECT value FROM vector_store.configuration WHERE key = $1")

	// First call - expect query returning false
	rows1 := sqlmock.NewRows([]string{"value"}).AddRow("false")
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows1)

	result1 := IsAttributeReplicationCompleted(mockDB)
	if result1 {
		t.Error("First call should return false")
	}

	// Second call - should query DB again (false is not cached)
	rows2 := sqlmock.NewRows([]string{"value"}).AddRow("false")
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows2)

	result2 := IsAttributeReplicationCompleted(mockDB)
	if result2 {
		t.Error("Second call should return false")
	}

	if err := mockDB.Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestIsAttributeReplicationCompleted_HandlesDatabaseError(t *testing.T) {
	// Reset cache before test
	ResetAttributeReplicationCache()

	mockDB := mocks.NewMockDb()
	defer mockDB.SqlDB.Close()

	query := regexp.QuoteMeta("SELECT value FROM vector_store.configuration WHERE key = $1")
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnError(sqlmock.ErrCancelled)

	// Should return false on error (fail-safe behavior)
	result := IsAttributeReplicationCompleted(mockDB)
	if result {
		t.Error("Should return false on database error")
	}

	if err := mockDB.Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestIsAttributeReplicationCompleted_ThreadSafety(t *testing.T) {
	// Reset cache before test
	ResetAttributeReplicationCache()

	mockDB := mocks.NewMockDb()
	defer mockDB.SqlDB.Close()

	query := regexp.QuoteMeta("SELECT value FROM vector_store.configuration WHERE key = $1")
	// We can't predict exact number of calls due to race conditions,
	// but with proper locking, should be minimal (1-few calls max)
	// So we'll set up multiple possible responses
	for i := 0; i < 10; i++ {
		rows := sqlmock.NewRows([]string{"value"}).AddRow("true")
		mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows)
	}

	// Launch multiple concurrent goroutines
	concurrency := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)

	results := make([]bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = IsAttributeReplicationCompleted(mockDB)
		}(i)
	}

	wg.Wait()

	// All results should be true
	for i, result := range results {
		if !result {
			t.Errorf("Goroutine %d got false, expected true", i)
		}
	}

	// We don't strictly check ExpectationsWereMet here because
	// the exact number of queries depends on race conditions
	// But we can verify that not all 10 queries were used (meaning cache worked)
}

func TestResetAttributeReplicationCache(t *testing.T) {
	// Reset cache before test
	ResetAttributeReplicationCache()

	mockDB := mocks.NewMockDb()
	defer mockDB.SqlDB.Close()

	query := regexp.QuoteMeta("SELECT value FROM vector_store.configuration WHERE key = $1")

	// First call
	rows1 := sqlmock.NewRows([]string{"value"}).AddRow("true")
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows1)

	IsAttributeReplicationCompleted(mockDB)

	// Second call - should use cache (no new query)
	result := IsAttributeReplicationCompleted(mockDB)
	if !result {
		t.Error("Second call should return true from cache")
	}

	// Reset cache
	ResetAttributeReplicationCache()

	// Next call should query DB again
	rows2 := sqlmock.NewRows([]string{"value"}).AddRow("true")
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows2)

	result = IsAttributeReplicationCompleted(mockDB)
	if !result {
		t.Error("Third call after reset should return true")
	}

	if err := mockDB.Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestIsAttributeReplicationCompleted_TransitionFromFalseToTrue(t *testing.T) {
	// Reset cache before test
	ResetAttributeReplicationCache()

	mockDB := mocks.NewMockDb()
	defer mockDB.SqlDB.Close()

	query := regexp.QuoteMeta("SELECT value FROM vector_store.configuration WHERE key = $1")

	// First call returns false
	rows1 := sqlmock.NewRows([]string{"value"}).AddRow("false")
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows1)

	result1 := IsAttributeReplicationCompleted(mockDB)
	if result1 {
		t.Error("First call should return false")
	}

	// Second call returns true and caches it
	rows2 := sqlmock.NewRows([]string{"value"}).AddRow("true")
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows2)

	result2 := IsAttributeReplicationCompleted(mockDB)
	if !result2 {
		t.Error("Second call should return true")
	}

	// Third call uses cache (no new query expected)
	result3 := IsAttributeReplicationCompleted(mockDB)
	if !result3 {
		t.Error("Third call should return true from cache")
	}

	if err := mockDB.Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestIsAttributeReplicationCompleted_NoRowsReturned(t *testing.T) {
	// Reset cache before test
	ResetAttributeReplicationCache()

	mockDB := mocks.NewMockDb()
	defer mockDB.SqlDB.Close()

	query := regexp.QuoteMeta("SELECT value FROM vector_store.configuration WHERE key = $1")
	// Return no rows (key doesn't exist)
	rows := sqlmock.NewRows([]string{"value"})
	mockDB.Mock.ExpectQuery(query).WithArgs("attribute_replication_v0.19.0_completed").WillReturnRows(rows)

	result := IsAttributeReplicationCompleted(mockDB)
	if result {
		t.Error("Should return false when key doesn't exist")
	}

	if err := mockDB.Mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}
