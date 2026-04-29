// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package saxmetrics

import (
	"testing"
)

// mockSAXMetricsCollector implements SAXMetricsCollector for testing
type mockSAXMetricsCollector struct {
	cacheHits   int
	cacheMisses int
	cacheSizes  []int
}

func (m *mockSAXMetricsCollector) RecordCacheHit() {
	m.cacheHits++
}

func (m *mockSAXMetricsCollector) RecordCacheMiss() {
	m.cacheMisses++
}

func (m *mockSAXMetricsCollector) RecordCacheSize(size int) {
	m.cacheSizes = append(m.cacheSizes, size)
}

func TestNewCollector(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, true)

	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	if !collector.IsEnabled() {
		t.Error("Expected collector to be enabled")
	}
}

func TestCollector_IsEnabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    bool
	}{
		{
			name:    "enabled collector",
			enabled: true,
			want:    true,
		},
		{
			name:    "disabled collector",
			enabled: false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSAXMetricsCollector{}
			collector := NewCollector(mock, tt.enabled)

			if got := collector.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollector_RecordCacheHit_Enabled(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, true)

	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheHit()

	if mock.cacheHits != 3 {
		t.Errorf("Expected 3 cache hits, got %d", mock.cacheHits)
	}
}

func TestCollector_RecordCacheHit_Disabled(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, false)

	collector.RecordCacheHit()
	collector.RecordCacheHit()

	if mock.cacheHits != 0 {
		t.Errorf("Expected 0 cache hits when disabled, got %d", mock.cacheHits)
	}
}

func TestCollector_RecordCacheHit_NilCollector(t *testing.T) {
	collector := NewCollector(nil, true)

	// Should not panic
	collector.RecordCacheHit()
}

func TestCollector_RecordCacheMiss_Enabled(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, true)

	collector.RecordCacheMiss()
	collector.RecordCacheMiss()

	if mock.cacheMisses != 2 {
		t.Errorf("Expected 2 cache misses, got %d", mock.cacheMisses)
	}
}

func TestCollector_RecordCacheMiss_Disabled(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, false)

	collector.RecordCacheMiss()
	collector.RecordCacheMiss()
	collector.RecordCacheMiss()

	if mock.cacheMisses != 0 {
		t.Errorf("Expected 0 cache misses when disabled, got %d", mock.cacheMisses)
	}
}

func TestCollector_RecordCacheMiss_NilCollector(t *testing.T) {
	collector := NewCollector(nil, true)

	// Should not panic
	collector.RecordCacheMiss()
}

func TestCollector_RecordCacheSize_Enabled(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, true)

	collector.RecordCacheSize(10)
	collector.RecordCacheSize(20)
	collector.RecordCacheSize(30)

	if len(mock.cacheSizes) != 3 {
		t.Errorf("Expected 3 cache size records, got %d", len(mock.cacheSizes))
	}

	expectedSizes := []int{10, 20, 30}
	for i, expected := range expectedSizes {
		if mock.cacheSizes[i] != expected {
			t.Errorf("Expected cache size %d at index %d, got %d", expected, i, mock.cacheSizes[i])
		}
	}
}

func TestCollector_RecordCacheSize_Disabled(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, false)

	collector.RecordCacheSize(10)
	collector.RecordCacheSize(20)

	if len(mock.cacheSizes) != 0 {
		t.Errorf("Expected 0 cache size records when disabled, got %d", len(mock.cacheSizes))
	}
}

func TestCollector_RecordCacheSize_NilCollector(t *testing.T) {
	collector := NewCollector(nil, true)

	// Should not panic
	collector.RecordCacheSize(42)
}

func TestCollector_RecordCacheSize_ZeroValue(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, true)

	collector.RecordCacheSize(0)

	if len(mock.cacheSizes) != 1 {
		t.Errorf("Expected 1 cache size record, got %d", len(mock.cacheSizes))
	}

	if mock.cacheSizes[0] != 0 {
		t.Errorf("Expected cache size 0, got %d", mock.cacheSizes[0])
	}
}

func TestCollector_MixedOperations(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, true)

	// Perform mixed operations
	collector.RecordCacheHit()
	collector.RecordCacheMiss()
	collector.RecordCacheSize(5)
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheMiss()
	collector.RecordCacheSize(10)

	if mock.cacheHits != 3 {
		t.Errorf("Expected 3 cache hits, got %d", mock.cacheHits)
	}

	if mock.cacheMisses != 2 {
		t.Errorf("Expected 2 cache misses, got %d", mock.cacheMisses)
	}

	if len(mock.cacheSizes) != 2 {
		t.Errorf("Expected 2 cache size records, got %d", len(mock.cacheSizes))
	}
}

// Checks for thread safe concurrent access
func TestCollector_ConcurrentAccess(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, true)

	done := make(chan bool)

	// Goroutine 1: Record hits
	go func() {
		for i := 0; i < 50; i++ {
			collector.RecordCacheHit()
		}
		done <- true
	}()

	// Goroutine 2: Record misses
	go func() {
		for i := 0; i < 30; i++ {
			collector.RecordCacheMiss()
		}
		done <- true
	}()

	// Goroutine 3: Record cache sizes
	go func() {
		for i := 0; i < 20; i++ {
			collector.RecordCacheSize(i)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestCollector_EnabledWithNilCollector(t *testing.T) {
	collector := NewCollector(nil, true)

	if !collector.IsEnabled() {
		t.Error("Expected collector to report as enabled even with nil underlying collector")
	}

	// These should not panic
	collector.RecordCacheHit()
	collector.RecordCacheMiss()
	collector.RecordCacheSize(42)
}

func TestCollector_DisabledWithValidCollector(t *testing.T) {
	mock := &mockSAXMetricsCollector{}
	collector := NewCollector(mock, false)

	if collector.IsEnabled() {
		t.Error("Expected collector to report as disabled")
	}

	// Perform operations - they should be no-ops
	collector.RecordCacheHit()
	collector.RecordCacheMiss()
	collector.RecordCacheSize(42)

	// Verify nothing was recorded
	if mock.cacheHits != 0 {
		t.Errorf("Expected 0 cache hits when disabled, got %d", mock.cacheHits)
	}

	if mock.cacheMisses != 0 {
		t.Errorf("Expected 0 cache misses when disabled, got %d", mock.cacheMisses)
	}

	if len(mock.cacheSizes) != 0 {
		t.Errorf("Expected 0 cache size records when disabled, got %d", len(mock.cacheSizes))
	}
}
