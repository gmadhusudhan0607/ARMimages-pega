// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package servicemetrics

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestClampToZero(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected int64
	}{
		{"positive value", 10, 10},
		{"zero", 0, 0},
		{"negative value", -5, 0},
		{"large negative", -1000000, 0},
		{"large positive", 1000000, 1000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampToZero(tt.input)
			if result != tt.expected {
				t.Errorf("clampToZero(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseGatewayResponseTimeMs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"empty string", "", 0},
		{"minus one (default no response)", "-1", 0},
		{"zero", "0", 0},
		{"valid positive", "150", 150},
		{"large positive", "99999", 99999},
		{"negative other than -1", "-5", 0},
		{"non-numeric", "abc", 0},
		{"float string", "1.5", 0},
		{"whitespace", " ", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGatewayResponseTimeMs(tt.input)
			if result != tt.expected {
				t.Errorf("parseGatewayResponseTimeMs(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCalculateProcessingOverheads(t *testing.T) {
	tests := []struct {
		name                     string
		requestMs                int64
		dbMs                     int64
		embMs                    int64
		gwMs                     int64
		expectedProcessingMs     int64
		expectedOverheadMs       int64
		expectedEmbNetOverheadMs int64
	}{
		{
			name:                     "all zeroes",
			requestMs:                0,
			dbMs:                     0,
			embMs:                    0,
			gwMs:                     0,
			expectedProcessingMs:     0,
			expectedOverheadMs:       0,
			expectedEmbNetOverheadMs: 0,
		},
		{
			name:                     "no embedding",
			requestMs:                100,
			dbMs:                     30,
			embMs:                    0,
			gwMs:                     0,
			expectedProcessingMs:     70,
			expectedOverheadMs:       70,
			expectedEmbNetOverheadMs: 0,
		},
		{
			name:                     "with embedding and gateway",
			requestMs:                200,
			dbMs:                     30,
			embMs:                    100,
			gwMs:                     60,
			expectedProcessingMs:     110, // overhead(70) + embNetOverhead(40)
			expectedOverheadMs:       70,  // 200 - 30 - 100
			expectedEmbNetOverheadMs: 40,  // 100 - 60
		},
		{
			name:                     "with embedding no gateway response",
			requestMs:                200,
			dbMs:                     30,
			embMs:                    100,
			gwMs:                     0,   // no gateway response (-1 or empty)
			expectedProcessingMs:     170, // overhead(70) + embNetOverhead(100)
			expectedOverheadMs:       70,  // 200 - 30 - 100
			expectedEmbNetOverheadMs: 100, // 100 - 0
		},
		{
			name:                     "clamp: requestMs < dbMs + embMs",
			requestMs:                50,
			dbMs:                     30,
			embMs:                    40,
			gwMs:                     0,
			expectedProcessingMs:     40, // overhead(0, clamped) + embNetOverhead(40)
			expectedOverheadMs:       0,  // clamped from -20
			expectedEmbNetOverheadMs: 40,
		},
		{
			name:                     "clamp: gwMs > embMs",
			requestMs:                200,
			dbMs:                     30,
			embMs:                    50,
			gwMs:                     80,  // gateway took longer than embedding (shouldn't happen but handle it)
			expectedProcessingMs:     120, // overhead(120) + embNetOverhead(0, clamped)
			expectedOverheadMs:       120, // 200 - 30 - 50
			expectedEmbNetOverheadMs: 0,   // clamped from -30
		},
		{
			name:                     "typical query request",
			requestMs:                150,
			dbMs:                     20,
			embMs:                    80,
			gwMs:                     50,
			expectedProcessingMs:     80, // overhead(50) + embNetOverhead(30)
			expectedOverheadMs:       50, // 150 - 20 - 80
			expectedEmbNetOverheadMs: 30, // 80 - 50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := buildServiceMetrics(tt.requestMs, tt.dbMs, tt.embMs, tt.gwMs)

			processingMs, overheadMs, embNetOverheadMs := sm.CalculateProcessingOverheads()

			if processingMs != tt.expectedProcessingMs {
				t.Errorf("processingDurationMs = %d, want %d", processingMs, tt.expectedProcessingMs)
			}
			if overheadMs != tt.expectedOverheadMs {
				t.Errorf("overheadMs = %d, want %d", overheadMs, tt.expectedOverheadMs)
			}
			if embNetOverheadMs != tt.expectedEmbNetOverheadMs {
				t.Errorf("embNetOverheadMs = %d, want %d", embNetOverheadMs, tt.expectedEmbNetOverheadMs)
			}
		})
	}
}

// buildServiceMetrics constructs a ServiceMetrics with controlled timing values for testing.
// Since tests are in the same package, we can access unexported fields directly.
func buildServiceMetrics(requestMs, dbMs, embMs, gwMs int64) *ServiceMetrics {
	sm := &ServiceMetrics{}

	// Set request duration via unexported fields (same package access)
	if requestMs > 0 {
		now := time.Now()
		sm.RequestMetrics.start = now.Add(-time.Duration(requestMs) * time.Millisecond)
		sm.RequestMetrics.stop = now
	}

	// Set DB query time: set queryStartTime in the past, then Stop() accumulates the duration
	if dbMs > 0 {
		dbM := sm.DbMetrics.NewMeasurement()
		dbM.queryStartTime = time.Now().Add(-time.Duration(dbMs) * time.Millisecond)
		dbM.Stop()
	}

	// Set embedding time: set startTime in the past, then Stop() records the duration
	if embMs > 0 {
		m := sm.EmbeddingMetrics.NewMeasurement("test-model", "v1")
		m.startTime = time.Now().Add(-time.Duration(embMs) * time.Millisecond)
		m.Stop()
	}

	// Set gateway response time header
	if gwMs > 0 {
		resp := &http.Response{
			Header: map[string][]string{
				"X-Genai-Gateway-Response-Time-Ms": {strconv.FormatInt(gwMs, 10)},
			},
		}
		sm.GatewayMetrics.SetGenaiHeadersFromResponse(resp)
	}

	return sm
}
