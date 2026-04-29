/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package httpmetrics

import (
	"testing"
	"time"
)

// mockHTTPMetricsCollector implements HTTPMetricsCollector for testing
type mockHTTPMetricsCollector struct {
	requests    []requestRecord
	connections []connectionRecord
	dbQueries   []dbQueryRecord
	items       []itemsReturnedRecord
}

type requestRecord struct {
	path, method, code string
	duration           time.Duration
}

type connectionRecord struct {
	path  string
	delta int
}

type dbQueryRecord struct {
	path, method, code string
	duration           time.Duration
}

type itemsReturnedRecord struct {
	path, method, code string
	itemCount          int
}

func (m *mockHTTPMetricsCollector) RecordRequest(path, method, code string, duration time.Duration) {
	m.requests = append(m.requests, requestRecord{path, method, code, duration})
}

func (m *mockHTTPMetricsCollector) RecordActiveConnection(path string, delta int) {
	m.connections = append(m.connections, connectionRecord{path, delta})
}

func (m *mockHTTPMetricsCollector) RecordDBQueryDuration(path, method, code string, duration time.Duration) {
	m.dbQueries = append(m.dbQueries, dbQueryRecord{path, method, code, duration})
}

func (m *mockHTTPMetricsCollector) RecordReturnedItems(path, method, code string, itemCount int) {
	m.items = append(m.items, itemsReturnedRecord{path, method, code, itemCount})
}

func TestNewHTTPMetrics(t *testing.T) {
	path := "/api/test"
	method := "GET"

	metrics := NewHTTPMetrics(path, method)

	if metrics.Path != path {
		t.Errorf("Expected path %s, got %s", path, metrics.Path)
	}

	if metrics.Method != method {
		t.Errorf("Expected method %s, got %s", method, metrics.Method)
	}

	if metrics.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}
}

func TestHTTPMetrics_Finish(t *testing.T) {
	metrics := NewHTTPMetrics("/api/test", "GET")

	// Wait a bit to ensure duration is non-zero
	time.Sleep(1 * time.Millisecond)

	statusCode := 200
	metrics.Finish(statusCode)

	if metrics.StatusCode != "200" {
		t.Errorf("Expected status code '200', got '%s'", metrics.StatusCode)
	}

	if metrics.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

func TestHTTPMetrics_SetDBQueryTime(t *testing.T) {
	metrics := NewHTTPMetrics("/api/test", "GET")
	duration := 50 * time.Millisecond

	metrics.SetDBQueryTime(duration)

	if metrics.DBQueryTime != duration {
		t.Errorf("Expected DB query time %v, got %v", duration, metrics.DBQueryTime)
	}
}

func TestCollector_RecordRequest(t *testing.T) {
	mock := &mockHTTPMetricsCollector{}
	config := DefaultConfig()
	collector := NewCollector(config, mock)

	metrics := HTTPMetrics{
		Path:       "/api/test",
		Method:     "GET",
		StatusCode: "200",
		Duration:   100 * time.Millisecond,
	}

	collector.RecordRequest(metrics)

	if len(mock.requests) != 1 {
		t.Errorf("Expected 1 request recorded, got %d", len(mock.requests))
	}

	req := mock.requests[0]
	if req.path != "/api/test" || req.method != "GET" || req.code != "200" {
		t.Errorf("Request recorded incorrectly: %+v", req)
	}

	if req.duration != 100*time.Millisecond {
		t.Errorf("Expected duration 100ms, got %v", req.duration)
	}
}

func TestCollector_RecordActiveConnection(t *testing.T) {
	mock := &mockHTTPMetricsCollector{}
	config := DefaultConfig()
	collector := NewCollector(config, mock)

	path := "/api/test"

	// Test increment
	collector.RecordActiveConnection(path, 1)

	if len(mock.connections) != 1 {
		t.Errorf("Expected 1 connection record, got %d", len(mock.connections))
	}

	conn := mock.connections[0]
	if conn.path != path || conn.delta != 1 {
		t.Errorf("Connection recorded incorrectly: %+v", conn)
	}

	// Test decrement
	collector.RecordActiveConnection(path, -1)

	if len(mock.connections) != 2 {
		t.Errorf("Expected 2 connection records, got %d", len(mock.connections))
	}

	conn = mock.connections[1]
	if conn.path != path || conn.delta != -1 {
		t.Errorf("Connection recorded incorrectly: %+v", conn)
	}
}

func TestCollector_RecordDBQueryDuration(t *testing.T) {
	mock := &mockHTTPMetricsCollector{}
	config := DefaultConfig()
	collector := NewCollector(config, mock)

	path := "/api/test"
	method := "GET"
	code := "200"
	duration := 25 * time.Millisecond

	collector.RecordDBQueryDuration(path, method, code, duration)

	if len(mock.dbQueries) != 1 {
		t.Errorf("Expected 1 DB query record, got %d", len(mock.dbQueries))
	}

	query := mock.dbQueries[0]
	if query.path != path || query.method != method || query.code != code {
		t.Errorf("DB query recorded incorrectly: %+v", query)
	}

	if query.duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, query.duration)
	}
}

func TestCollector_RecordItemsReturnedCount(t *testing.T) {
	mock := &mockHTTPMetricsCollector{}
	config := DefaultConfig()
	collector := NewCollector(config, mock)

	collector.RecordReturnedItems("/api/test", "GET", "200", 5)

	if len(mock.items) != 1 {
		t.Errorf("Expected 1 items record, got %d", len(mock.items))
	}

	item := mock.items[0]
	if item.path != "/api/test" || item.method != "GET" || item.code != "200" || item.itemCount != 5 {
		t.Errorf("Items record incorrectly: %+v", item)
	}
}

func TestCollector_DisabledConfig(t *testing.T) {
	mock := &mockHTTPMetricsCollector{}
	config := Config{
		Enabled:          false,
		IncludeDBMetrics: true,
	}
	collector := NewCollector(config, mock)

	metrics := HTTPMetrics{
		Path:       "/api/test",
		Method:     "GET",
		StatusCode: "200",
		Duration:   100 * time.Millisecond,
	}

	// These should not record anything when disabled
	collector.RecordRequest(metrics)
	collector.RecordActiveConnection("/api/test", 1)
	collector.RecordDBQueryDuration("/api/test", "GET", "200", 25*time.Millisecond)

	if len(mock.requests) != 0 {
		t.Errorf("Expected 0 requests when disabled, got %d", len(mock.requests))
	}

	if len(mock.connections) != 0 {
		t.Errorf("Expected 0 connections when disabled, got %d", len(mock.connections))
	}

	if len(mock.dbQueries) != 0 {
		t.Errorf("Expected 0 DB queries when disabled, got %d", len(mock.dbQueries))
	}
}

func TestCollector_DBMetricsDisabled(t *testing.T) {
	mock := &mockHTTPMetricsCollector{}
	config := Config{
		Enabled:          true,
		IncludeDBMetrics: false,
	}
	collector := NewCollector(config, mock)

	collector.RecordDBQueryDuration("/api/test", "GET", "200", 25*time.Millisecond)

	if len(mock.dbQueries) != 0 {
		t.Errorf("Expected 0 DB queries when DB metrics disabled, got %d", len(mock.dbQueries))
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("Expected default config to be enabled")
	}

	if !config.IncludeDBMetrics {
		t.Error("Expected default config to include DB metrics")
	}

	if len(config.RequestBuckets) == 0 {
		t.Error("Expected default config to have request buckets")
	}

	if len(config.DBQueryBuckets) == 0 {
		t.Error("Expected default config to have DB query buckets")
	}
}

func TestCollector_IsEnabled(t *testing.T) {
	mock := &mockHTTPMetricsCollector{}

	// Test enabled
	config := Config{Enabled: true}
	collector := NewCollector(config, mock)

	if !collector.IsEnabled() {
		t.Error("Expected collector to be enabled")
	}

	// Test disabled
	config = Config{Enabled: false}
	collector = NewCollector(config, mock)

	if collector.IsEnabled() {
		t.Error("Expected collector to be disabled")
	}
}

func TestCollector_Config(t *testing.T) {
	mock := &mockHTTPMetricsCollector{}
	originalConfig := DefaultConfig()
	collector := NewCollector(originalConfig, mock)

	retrievedConfig := collector.Config()

	if retrievedConfig.Enabled != originalConfig.Enabled {
		t.Error("Config mismatch: Enabled")
	}

	if retrievedConfig.IncludeDBMetrics != originalConfig.IncludeDBMetrics {
		t.Error("Config mismatch: IncludeDBMetrics")
	}
}
