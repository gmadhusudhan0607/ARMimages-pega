// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMetric is a simple type for testing PDCSender with generics
type TestMetric struct {
	ID    string `json:"id"`
	Value int    `json:"value"`
}

// createTestPDCSender creates a PDCSender instance for testing
func createTestPDCSender(maxPayloadSize int) *PDCSender[TestMetric] {
	return NewPDCSender[TestMetric](PDCSenderConfig{
		RequestTimeoutSeconds: 30,
		MaxPayloadSizeBytes:   maxPayloadSize,
	}, "test.pdc")
}

// TestSend_EmptyMetrics tests that sending empty metrics returns nil without making HTTP calls
func TestSend_EmptyMetrics(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := createTestPDCSender(1000)
	ctx := context.Background()

	err := sender.Send(ctx, server.URL, []TestMetric{})

	// Should not return an error
	if err != nil {
		t.Errorf("Expected no error for empty metrics, got: %v", err)
	}

	// Should not make any HTTP requests
	if requestCount != 0 {
		t.Errorf("Expected 0 HTTP requests for empty metrics, got %d", requestCount)
	}
}

// TestSend_SingleChunk_Success tests successful send of a small payload in a single chunk
func TestSend_SingleChunk_Success(t *testing.T) {
	requestCount := 0
	var receivedPayload UsageDataPayload[TestMetric]

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Verify Content-Type header
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("Expected Content-Type: application/octet-stream, got: %s", r.Header.Get("Content-Type"))
		}

		// Decode payload
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := createTestPDCSender(1000)
	ctx := context.Background()

	metrics := []TestMetric{
		{ID: "m1", Value: 100},
		{ID: "m2", Value: 200},
	}

	err := sender.Send(ctx, server.URL, metrics)

	// Should not return an error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should make exactly 1 HTTP request
	if requestCount != 1 {
		t.Errorf("Expected 1 HTTP request, got %d", requestCount)
	}

	// Verify payload structure
	if len(receivedPayload.Data) != 2 {
		t.Errorf("Expected 2 metrics in payload, got %d", len(receivedPayload.Data))
	}

	if receivedPayload.Metadata.SegmentNumber != 1 {
		t.Errorf("Expected segment number 1, got %d", receivedPayload.Metadata.SegmentNumber)
	}

	if receivedPayload.Metadata.SegmentsTotal != 1 {
		t.Errorf("Expected segments total 1, got %d", receivedPayload.Metadata.SegmentsTotal)
	}

	if receivedPayload.Metadata.Source != metricSourceGenAIVectorStore {
		t.Errorf("Expected source %s, got %s", metricSourceGenAIVectorStore, receivedPayload.Metadata.Source)
	}
}

// TestSend_SingleChunk_Failure tests that HTTP errors are properly returned
func TestSend_SingleChunk_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	sender := createTestPDCSender(1000)
	ctx := context.Background()

	metrics := []TestMetric{
		{ID: "m1", Value: 100},
	}

	err := sender.Send(ctx, server.URL, metrics)

	// Should return an error
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}

	// Error should mention the status code
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

// TestSend_MultipleChunks_AllSuccess tests successful chunking and sending of large payload
func TestSend_MultipleChunks_AllSuccess(t *testing.T) {
	requestCount := 0
	var receivedSegments []int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		var payload UsageDataPayload[TestMetric]
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}

		receivedSegments = append(receivedSegments, payload.Metadata.SegmentNumber)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := createTestPDCSender(300) // Small max size to force chunking

	ctx := context.Background()

	// Create enough metrics to require chunking
	metrics := make([]TestMetric, 20)
	for i := range metrics {
		metrics[i] = TestMetric{ID: "metric", Value: i}
	}

	err := sender.Send(ctx, server.URL, metrics)

	// Should not return an error
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should make multiple HTTP requests (chunked)
	if requestCount <= 1 {
		t.Errorf("Expected more than 1 HTTP request for large payload, got %d", requestCount)
	}

	// Verify segments are sequential
	for i, segment := range receivedSegments {
		expectedSegment := i + 1
		if segment != expectedSegment {
			t.Errorf("Expected segment %d, got %d", expectedSegment, segment)
		}
	}
}

// TestSend_MultipleChunks_PartialFailure tests that partial failures are properly reported
func TestSend_MultipleChunks_PartialFailure(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// First chunk succeeds, second fails
		if requestCount == 1 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	sender := createTestPDCSender(300) // Small max size to force chunking

	ctx := context.Background()

	// Create enough metrics to require chunking (at least 2 chunks)
	metrics := make([]TestMetric, 20)
	for i := range metrics {
		metrics[i] = TestMetric{ID: "metric", Value: i}
	}

	err := sender.Send(ctx, server.URL, metrics)

	// Should return a PartialFailureError
	if err == nil {
		t.Error("Expected PartialFailureError, got nil")
	}

	partialErr, ok := err.(*PartialFailureError)
	if !ok {
		t.Errorf("Expected *PartialFailureError, got %T", err)
		return
	}

	// Should have sent some metrics successfully (first chunk)
	if partialErr.SuccessfullySent == 0 {
		t.Error("Expected some metrics to be sent successfully")
	}

	// Should not have sent all metrics
	if partialErr.SuccessfullySent >= len(metrics) {
		t.Errorf("Expected fewer than %d metrics sent, got %d", len(metrics), partialErr.SuccessfullySent)
	}

	// Total metrics should match
	if partialErr.TotalMetrics != len(metrics) {
		t.Errorf("Expected total metrics %d, got %d", len(metrics), partialErr.TotalMetrics)
	}

	// Failed segment should be > 1 (since first succeeded)
	if partialErr.FailedSegment <= 1 {
		t.Errorf("Expected failed segment > 1, got %d", partialErr.FailedSegment)
	}
}
