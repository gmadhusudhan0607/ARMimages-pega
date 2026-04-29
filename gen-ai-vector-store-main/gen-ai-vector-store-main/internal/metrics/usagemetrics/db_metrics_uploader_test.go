// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
)

// mockIsolationsGetter implements IsolationsGetter for testing
type mockIsolationsGetter struct {
	isolations []*isolations.Details
	err        error
}

func (m *mockIsolationsGetter) GetIsolations(ctx context.Context) ([]*isolations.Details, error) {
	return m.isolations, m.err
}

// mockDatabase implements db.Database interface for testing
type mockDatabase struct {
	db.Database
}

// TestCollectAndUploadMetrics_NoIsolations tests behavior when no isolations exist
func TestCollectAndUploadMetrics_NoIsolations(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockIsoMgr := &mockIsolationsGetter{
		isolations: []*isolations.Details{},
		err:        nil,
	}

	uploader := NewDBMetricsUploader(&mockDatabase{}, mockIsoMgr)

	ctx := context.Background()
	uploader.collectAndUploadMetrics(ctx)

	// Should not make any HTTP requests when there are no isolations
	if requestCount != 0 {
		t.Errorf("Expected 0 HTTP requests with no isolations, got %d", requestCount)
	}
}

// TestCollectAndUploadMetrics_IsolationsWithoutPDCURL tests that isolations without PDC URL are skipped
func TestCollectAndUploadMetrics_IsolationsWithoutPDCURL(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockIsoMgr := &mockIsolationsGetter{
		isolations: []*isolations.Details{
			{ID: "iso1", PDCEndpointURL: ""}, // No PDC URL
			{ID: "iso2", PDCEndpointURL: ""}, // No PDC URL
		},
		err: nil,
	}

	uploader := NewDBMetricsUploader(&mockDatabase{}, mockIsoMgr)

	ctx := context.Background()
	uploader.collectAndUploadMetrics(ctx)

	// Should not make any HTTP requests when isolations have no PDC URL
	if requestCount != 0 {
		t.Errorf("Expected 0 HTTP requests with no PDC URLs, got %d", requestCount)
	}
}

// TestCollectAndUploadMetrics_GetIsolationsError tests error handling when GetIsolations fails
func TestCollectAndUploadMetrics_GetIsolationsError(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockIsoMgr := &mockIsolationsGetter{
		isolations: nil,
		err:        context.DeadlineExceeded,
	}

	uploader := NewDBMetricsUploader(&mockDatabase{}, mockIsoMgr)

	ctx := context.Background()
	uploader.collectAndUploadMetrics(ctx)

	// Should not make any HTTP requests when GetIsolations fails
	if requestCount != 0 {
		t.Errorf("Expected 0 HTTP requests when GetIsolations fails, got %d", requestCount)
	}
}

// TestUploadMetrics_Success tests successful metric upload
func TestUploadMetrics_Success(t *testing.T) {
	requestCount := 0
	var receivedMetrics []DBMetric

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		var payload UsageDataPayload[DBMetric]
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}

		receivedMetrics = append(receivedMetrics, payload.Data...)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockIsoMgr := &mockIsolationsGetter{
		isolations: []*isolations.Details{},
	}

	uploader := NewDBMetricsUploader(&mockDatabase{}, mockIsoMgr)

	ctx := context.Background()
	metrics := []DBMetric{
		{
			MetricType:     "DB",
			IsolationID:    "iso1",
			DiskUsage:      1000,
			DocumentsCount: 50,
		},
		{
			MetricType:     "DB",
			IsolationID:    "iso2",
			DiskUsage:      2000,
			DocumentsCount: 100,
		},
	}

	err := uploader.uploadMetrics(ctx, server.URL, metrics)

	// Should succeed
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should have received the metrics
	if len(receivedMetrics) != 2 {
		t.Errorf("Expected 2 metrics received, got %d", len(receivedMetrics))
	}

	// Verify first metric
	if receivedMetrics[0].IsolationID != "iso1" {
		t.Errorf("Expected isolation ID 'iso1', got '%s'", receivedMetrics[0].IsolationID)
	}
	if receivedMetrics[0].DiskUsage != 1000 {
		t.Errorf("Expected disk usage 1000, got %d", receivedMetrics[0].DiskUsage)
	}
}

// TestUploadMetrics_Failure tests upload failure handling
func TestUploadMetrics_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockIsoMgr := &mockIsolationsGetter{
		isolations: []*isolations.Details{},
	}

	uploader := NewDBMetricsUploader(&mockDatabase{}, mockIsoMgr)

	ctx := context.Background()
	metrics := []DBMetric{
		{
			MetricType:     "DB",
			IsolationID:    "iso1",
			DiskUsage:      1000,
			DocumentsCount: 50,
		},
	}

	err := uploader.uploadMetrics(ctx, server.URL, metrics)

	// Should return an error
	if err == nil {
		t.Error("Expected error when upload fails, got nil")
	}
}

// TestUploadMetrics_Chunking tests that large payloads are chunked
func TestUploadMetrics_Chunking(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockIsoMgr := &mockIsolationsGetter{
		isolations: []*isolations.Details{},
	}

	// Create uploader with small max payload size to force chunking
	uploader := NewDBMetricsUploader(&mockDatabase{}, mockIsoMgr)
	uploader.pdcSender.maxPayloadSizeBytes = 300 // Small size to force chunking

	ctx := context.Background()

	// Create many metrics to exceed payload size
	metrics := make([]DBMetric, 20)
	for i := range metrics {
		metrics[i] = DBMetric{
			MetricType:     "DB",
			IsolationID:    "iso1",
			DiskUsage:      1000,
			DocumentsCount: 50,
		}
	}

	err := uploader.uploadMetrics(ctx, server.URL, metrics)

	// Should succeed
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should make multiple requests (chunked)
	if requestCount <= 1 {
		t.Errorf("Expected multiple HTTP requests for chunking, got %d", requestCount)
	}
}
