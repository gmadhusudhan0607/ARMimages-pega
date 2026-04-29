// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
)

// mockIsoManager is a test-only mock implementation of the IsoManager interface
type mockIsoManager struct {
	isolation    *isolations.Details
	isolationErr error
}

func (m *mockIsoManager) GetIsolation(ctx context.Context, isolationID string) (*isolations.Details, error) {
	return m.isolation, m.isolationErr
}

// createTestUploader creates an uploader instance for testing
func createTestUploader(isoManager IsolationGetter) *Uploader {
	collector := NewCollector(Config{
		Enabled:               true,
		UploadIntervalSeconds: 60,
		MaxPayloadSizeBytes:   1000,
		RetryCount:            3,
		RequestTimeoutSeconds: 30,
	})

	pdcSender := NewPDCSender[SemanticSearchMetric](PDCSenderConfig{
		RequestTimeoutSeconds: 30,
		MaxPayloadSizeBytes:   1000,
	}, "usagemetrics.uploader.test.pdc")

	return &Uploader{
		collector:  collector,
		isoManager: isoManager,
		pdcSender:  pdcSender,
		logger:     log.GetNamedLogger("usagemetrics.uploader.test"),
	}
}

// TestRequeueMetrics_BelowMaxRetryCount tests that metrics below max retry count are re-queued
func TestRequeueMetrics_BelowMaxRetryCount(t *testing.T) {
	uploader := createTestUploader(&mockIsoManager{})

	// Create metrics with various retry counts below the limit
	metrics := []SemanticSearchMetric{
		{IsolationID: "iso1", CollectionID: "col1", Endpoint: "query_documents", retryCount: 0},
		{IsolationID: "iso1", CollectionID: "col2", Endpoint: "query_chunks", retryCount: 5},
		{IsolationID: "iso1", CollectionID: "col3", Endpoint: "query_documents", retryCount: 9},
	}

	uploader.requeueMetrics(metrics)

	// Verify all metrics were added back to the collector
	queue := uploader.collector.GetAndClearQueue()
	if len(queue["iso1"]) != 3 {
		t.Errorf("Expected 3 metrics in queue, got %d", len(queue["iso1"]))
	}

	// Verify retry counts were incremented
	for i, metric := range queue["iso1"] {
		expectedRetryCount := metrics[i].retryCount + 1
		if metric.retryCount != expectedRetryCount {
			t.Errorf("Metric %d: expected retryCount %d, got %d", i, expectedRetryCount, metric.retryCount)
		}
	}
}

// TestRequeueMetrics_AtMaxRetryCount tests that metrics at max retry count are discarded
func TestRequeueMetrics_AtMaxRetryCount(t *testing.T) {
	uploader := createTestUploader(&mockIsoManager{})

	// Create metrics at max retry count
	metrics := []SemanticSearchMetric{
		{IsolationID: "iso1", CollectionID: "col1", Endpoint: "query_documents", retryCount: maxRetryCount},
		{IsolationID: "iso1", CollectionID: "col2", Endpoint: "query_chunks", retryCount: maxRetryCount},
	}

	uploader.requeueMetrics(metrics)

	// Verify no metrics were added back to the collector
	queue := uploader.collector.GetAndClearQueue()
	if len(queue["iso1"]) != 0 {
		t.Errorf("Expected 0 metrics in queue (all should be discarded), got %d", len(queue["iso1"]))
	}
}

// TestRequeueMetrics_MixedRetryCount tests mixed retry counts (some discarded, some re-queued)
func TestRequeueMetrics_MixedRetryCount(t *testing.T) {
	uploader := createTestUploader(&mockIsoManager{})

	// Create metrics with mixed retry counts
	metrics := []SemanticSearchMetric{
		{IsolationID: "iso1", CollectionID: "col1", Endpoint: "query_documents", retryCount: 0},
		{IsolationID: "iso1", CollectionID: "col2", Endpoint: "query_chunks", retryCount: maxRetryCount},
		{IsolationID: "iso1", CollectionID: "col3", Endpoint: "query_documents", retryCount: 9},
		{IsolationID: "iso1", CollectionID: "col4", Endpoint: "query_chunks", retryCount: maxRetryCount},
		{IsolationID: "iso1", CollectionID: "col5", Endpoint: "query_documents", retryCount: 5},
	}

	uploader.requeueMetrics(metrics)

	// Verify only metrics below max retry count were re-queued (3 out of 5)
	queue := uploader.collector.GetAndClearQueue()
	if len(queue["iso1"]) != 3 {
		t.Errorf("Expected 3 metrics in queue, got %d", len(queue["iso1"]))
	}

	// Verify the re-queued metrics have incremented retry counts
	for _, metric := range queue["iso1"] {
		if metric.retryCount > maxRetryCount {
			t.Errorf("Metric has invalid retryCount %d (should be <= %d)", metric.retryCount, maxRetryCount)
		}
	}
}

// TestRequeueMetrics_IncrementRetryCount tests that retry count is properly incremented
func TestRequeueMetrics_IncrementRetryCount(t *testing.T) {
	uploader := createTestUploader(&mockIsoManager{})

	testCases := []struct {
		name               string
		initialRetryCount  int
		expectedRetryCount int
		shouldBeQueued     bool
	}{
		{"Zero to One", 0, 1, true},
		{"Five to Six", 5, 6, true},
		{"Nine to Ten", 9, 10, true},
		{"At Max", maxRetryCount, maxRetryCount, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear queue before each test
			uploader.collector.GetAndClearQueue()

			metrics := []SemanticSearchMetric{
				{IsolationID: "iso1", CollectionID: "col1", Endpoint: "query_documents", retryCount: tc.initialRetryCount},
			}

			uploader.requeueMetrics(metrics)

			queue := uploader.collector.GetAndClearQueue()
			if tc.shouldBeQueued {
				if len(queue["iso1"]) != 1 {
					t.Errorf("Expected 1 metric in queue, got %d", len(queue["iso1"]))
				}
				if queue["iso1"][0].retryCount != tc.expectedRetryCount {
					t.Errorf("Expected retryCount %d, got %d", tc.expectedRetryCount, queue["iso1"][0].retryCount)
				}
			} else {
				if len(queue["iso1"]) != 0 {
					t.Errorf("Expected 0 metrics in queue (should be discarded), got %d", len(queue["iso1"]))
				}
			}
		})
	}
}

// TestUploadMetrics_ReturnsMetricsOnFailure tests that upload failure returns all metrics
func TestUploadMetrics_ReturnsMetricsOnFailure(t *testing.T) {
	// Create HTTP server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockIso := &mockIsoManager{
		isolation: &isolations.Details{
			ID:             "iso1",
			PDCEndpointURL: server.URL,
		},
	}

	uploader := createTestUploader(mockIso)
	uploader.collector.config.RetryCount = 1 // Reduce retry count for faster test

	metrics := []SemanticSearchMetric{
		{IsolationID: "iso1", CollectionID: "col1", Endpoint: "query_documents", StatusCode: 200},
		{IsolationID: "iso1", CollectionID: "col2", Endpoint: "query_chunks", StatusCode: 200},
	}

	ctx := context.Background()
	err, missedMetrics := uploader.uploadMetrics(ctx, server.URL, metrics)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error from upload failure")
	}

	// Verify all metrics were returned as missed
	if len(missedMetrics) != len(metrics) {
		t.Errorf("Expected %d missed metrics, got %d", len(metrics), len(missedMetrics))
	}
}

// TestUploadMetrics_SuccessfulSingleChunk tests successful single chunk upload
func TestUploadMetrics_SuccessfulSingleChunk(t *testing.T) {
	// Create HTTP server that always succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockIso := &mockIsoManager{
		isolation: &isolations.Details{
			ID:             "iso1",
			PDCEndpointURL: server.URL,
		},
	}

	uploader := createTestUploader(mockIso)

	metrics := []SemanticSearchMetric{
		{IsolationID: "iso1", CollectionID: "col1", Endpoint: "query_documents", StatusCode: 200},
	}

	ctx := context.Background()
	err, missedMetrics := uploader.uploadMetrics(ctx, server.URL, metrics)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify no missed metrics
	if len(missedMetrics) != 0 {
		t.Errorf("Expected 0 missed metrics, got %d", len(missedMetrics))
	}
}

// TestRequeueMetrics_EmptySlice tests requeuing with empty metrics slice
func TestRequeueMetrics_EmptySlice(t *testing.T) {
	uploader := createTestUploader(&mockIsoManager{})

	uploader.requeueMetrics([]SemanticSearchMetric{})

	// Verify queue remains empty
	queue := uploader.collector.GetAndClearQueue()
	if len(queue) != 0 {
		t.Errorf("Expected empty queue, got %d items", len(queue))
	}
}

// TestRequeueMetrics_MultipleIsolations tests requeuing metrics for multiple isolations
func TestRequeueMetrics_MultipleIsolations(t *testing.T) {
	uploader := createTestUploader(&mockIsoManager{})

	metrics := []SemanticSearchMetric{
		{IsolationID: "iso1", CollectionID: "col1", Endpoint: "query_documents", retryCount: 0},
		{IsolationID: "iso2", CollectionID: "col2", Endpoint: "query_chunks", retryCount: 1},
		{IsolationID: "iso1", CollectionID: "col3", Endpoint: "query_documents", retryCount: 2},
	}

	uploader.requeueMetrics(metrics)

	// Verify metrics were added to their respective isolation queues
	queue := uploader.collector.GetAndClearQueue()
	if len(queue["iso1"]) != 2 {
		t.Errorf("Expected 2 metrics for iso1, got %d", len(queue["iso1"]))
	}
	if len(queue["iso2"]) != 1 {
		t.Errorf("Expected 1 metric for iso2, got %d", len(queue["iso2"]))
	}
}

// TestUploadMetrics_PartialSuccess_FirstChunkFails tests that when first chunk fails, all metrics are returned
func TestUploadMetrics_PartialSuccess_FirstChunkFails(t *testing.T) {
	requestCount := 0

	// Create HTTP server that fails on first request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockIso := &mockIsoManager{
		isolation: &isolations.Details{
			ID:             "iso1",
			PDCEndpointURL: server.URL,
		},
	}

	uploader := createTestUploader(mockIso)
	uploader.collector.config.RetryCount = 1
	uploader.collector.config.MaxPayloadSizeBytes = 200 // Force chunking

	// Create enough metrics to require chunking
	metrics := make([]SemanticSearchMetric, 10)
	for i := range metrics {
		metrics[i] = SemanticSearchMetric{
			IsolationID:  "iso1",
			CollectionID: "col1",
			Endpoint:     "query_documents",
			StatusCode:   200,
		}
	}

	ctx := context.Background()
	err, missedMetrics := uploader.uploadMetrics(ctx, server.URL, metrics)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error when first chunk fails")
	}

	// When first chunk fails, all metrics should be returned
	if len(missedMetrics) != len(metrics) {
		t.Errorf("Expected all %d metrics as missed, got %d", len(metrics), len(missedMetrics))
	}
}

// TestUploadMetrics_PartialSuccess_MiddleChunkFails tests partial success with middle chunk failure
func TestUploadMetrics_PartialSuccess_MiddleChunkFails(t *testing.T) {
	requestCount := 0

	// Create HTTP server that succeeds on first request, fails on second
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusOK) // First chunk succeeds
		} else {
			w.WriteHeader(http.StatusInternalServerError) // Second chunk fails
		}
	}))
	defer server.Close()

	mockIso := &mockIsoManager{
		isolation: &isolations.Details{
			ID:             "iso1",
			PDCEndpointURL: server.URL,
		},
	}

	uploader := createTestUploader(mockIso)
	uploader.collector.config.RetryCount = 1
	uploader.collector.config.MaxPayloadSizeBytes = 400 // Force chunking into at least 2 chunks

	// Create enough metrics to require chunking
	metrics := make([]SemanticSearchMetric, 20)
	for i := range metrics {
		metrics[i] = SemanticSearchMetric{
			IsolationID:       "iso1",
			CollectionID:      "col1",
			Endpoint:          "query_documents",
			StatusCode:        200,
			RequestDurationMs: 100,
		}
	}

	ctx := context.Background()
	err, missedMetrics := uploader.uploadMetrics(ctx, server.URL, metrics)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error when middle chunk fails")
	}

	// Should have fewer missed metrics than total (first chunk succeeded)
	if len(missedMetrics) >= len(metrics) {
		t.Errorf("Expected fewer than %d missed metrics (partial success), got %d", len(metrics), len(missedMetrics))
	}

	// Should have some missed metrics (second chunk failed)
	if len(missedMetrics) == 0 {
		t.Error("Expected some missed metrics from failed chunk")
	}
}

// TestUploadMetrics_PartialSuccess_RetryWithOnlyUnsentMetrics tests that retry only sends unsent metrics
func TestUploadMetrics_PartialSuccess_RetryWithOnlyUnsentMetrics(t *testing.T) {
	requestCount := 0
	totalMetricsSentInRetry := 0

	// Create HTTP server that tracks retry behavior
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Decode to count metrics in request
		var payload struct {
			Data []SemanticSearchMetric `json:"data"`
		}
		json.NewDecoder(r.Body).Decode(&payload)

		switch requestCount {
		case 1:
			// First chunk succeeds (5 metrics)
			w.WriteHeader(http.StatusOK)
		case 2:
			// Second chunk fails first time (trying to send next 5 metrics)
			w.WriteHeader(http.StatusInternalServerError)
		default:
			// All subsequent chunks should succeed (these are retry attempts with remaining metrics)
			totalMetricsSentInRetry += len(payload.Data)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	mockIso := &mockIsoManager{
		isolation: &isolations.Details{
			ID:             "iso1",
			PDCEndpointURL: server.URL,
		},
	}

	uploader := createTestUploader(mockIso)
	uploader.collector.config.RetryCount = 2
	uploader.collector.config.MaxPayloadSizeBytes = 400

	// Create metrics that will be chunked
	metrics := make([]SemanticSearchMetric, 20)
	for i := range metrics {
		metrics[i] = SemanticSearchMetric{
			IsolationID:       "iso1",
			CollectionID:      "col1",
			Endpoint:          "query_documents",
			StatusCode:        200,
			RequestDurationMs: 100,
		}
	}

	ctx := context.Background()
	err, missedMetrics := uploader.uploadMetrics(ctx, server.URL, metrics)

	// Should succeed on retry
	if err != nil {
		t.Errorf("Expected success on retry, got error: %v", err)
	}

	// Should have no missed metrics after successful retry
	if len(missedMetrics) != 0 {
		t.Errorf("Expected 0 missed metrics after successful retry, got %d", len(missedMetrics))
	}

	// Verify retry sent fewer total metrics than original
	// First chunk sent 5, then remaining 15 should be sent on retry
	if totalMetricsSentInRetry != 15 {
		t.Errorf("Expected 15 metrics sent in retry, got %d", totalMetricsSentInRetry)
	}

	// Should have made multiple requests: chunk1 (success), chunk2 (fail), then retry chunks (success)
	if requestCount < 3 {
		t.Errorf("Expected at least 3 HTTP requests, got %d", requestCount)
	}
}
