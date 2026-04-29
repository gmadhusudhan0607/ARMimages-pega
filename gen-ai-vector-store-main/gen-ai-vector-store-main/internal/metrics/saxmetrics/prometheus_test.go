// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package saxmetrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestNewPrometheusCollector(t *testing.T) {
	collector := NewPrometheusCollector()

	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	if collector.saxCacheHits == nil {
		t.Error("Expected saxCacheHits to be initialized")
	}

	if collector.saxCacheMisses == nil {
		t.Error("Expected saxCacheMisses to be initialized")
	}

	if collector.saxCacheSize == nil {
		t.Error("Expected saxCacheSize to be initialized")
	}

	if collector.saxCacheHitRatio == nil {
		t.Error("Expected saxCacheHitRatio to be initialized")
	}
}

func TestPrometheusCollector_RecordCacheHit(t *testing.T) {
	collector := NewPrometheusCollector()

	// Record some hits
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheHit()

	// Verify the counter was incremented
	var metric dto.Metric
	if err := collector.saxCacheHits.Write(&metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	hits := metric.GetCounter().GetValue()
	if hits != 3 {
		t.Errorf("Expected 3 hits, got %f", hits)
	}
}

func TestPrometheusCollector_RecordCacheMiss(t *testing.T) {
	collector := NewPrometheusCollector()

	// Record some misses
	collector.RecordCacheMiss()
	collector.RecordCacheMiss()

	// Verify the counter was incremented
	var metric dto.Metric
	if err := collector.saxCacheMisses.Write(&metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	misses := metric.GetCounter().GetValue()
	if misses != 2 {
		t.Errorf("Expected 2 misses, got %f", misses)
	}
}

func TestPrometheusCollector_RecordCacheSize(t *testing.T) {
	collector := NewPrometheusCollector()

	// Record cache size
	collector.RecordCacheSize(42)

	// Verify the gauge was set
	var metric dto.Metric
	if err := collector.saxCacheSize.Write(&metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	size := metric.GetGauge().GetValue()
	if size != 42 {
		t.Errorf("Expected cache size 42, got %f", size)
	}

	// Update cache size
	collector.RecordCacheSize(100)

	metric.Reset()
	if err := collector.saxCacheSize.Write(&metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}

	size = metric.GetGauge().GetValue()
	if size != 100 {
		t.Errorf("Expected cache size 100, got %f", size)
	}
}

func TestPrometheusCollector_CalculateCacheHitRatio_NoData(t *testing.T) {
	collector := NewPrometheusCollector()

	// With no hits or misses, ratio should be 0
	ratio := collector.calculateCacheHitRatio()

	if ratio != 0 {
		t.Errorf("Expected ratio 0 with no data, got %f", ratio)
	}
}

func TestPrometheusCollector_CalculateCacheHitRatio_OnlyHits(t *testing.T) {
	collector := NewPrometheusCollector()

	// Record only hits
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheHit()

	ratio := collector.calculateCacheHitRatio()

	if ratio != 1.0 {
		t.Errorf("Expected ratio 1.0 with only hits, got %f", ratio)
	}
}

func TestPrometheusCollector_CalculateCacheHitRatio_OnlyMisses(t *testing.T) {
	collector := NewPrometheusCollector()

	// Record only misses
	collector.RecordCacheMiss()
	collector.RecordCacheMiss()

	ratio := collector.calculateCacheHitRatio()

	if ratio != 0.0 {
		t.Errorf("Expected ratio 0.0 with only misses, got %f", ratio)
	}
}

func TestPrometheusCollector_CalculateCacheHitRatio_Mixed(t *testing.T) {
	collector := NewPrometheusCollector()

	// Record 3 hits and 1 miss (75% hit rate)
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheMiss()

	ratio := collector.calculateCacheHitRatio()

	expected := 0.75
	if ratio != expected {
		t.Errorf("Expected ratio %f, got %f", expected, ratio)
	}
}

func TestPrometheusCollector_CalculateCacheHitRatio_EqualHitsAndMisses(t *testing.T) {
	collector := NewPrometheusCollector()

	// Record equal hits and misses (50% hit rate)
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheMiss()
	collector.RecordCacheMiss()

	ratio := collector.calculateCacheHitRatio()

	expected := 0.5
	if ratio != expected {
		t.Errorf("Expected ratio %f, got %f", expected, ratio)
	}
}

func TestPrometheusCollector_CalculateCacheHitRatio_LargeNumbers(t *testing.T) {
	collector := NewPrometheusCollector()

	// Record large numbers of hits and misses
	for i := 0; i < 1000; i++ {
		collector.RecordCacheHit()
	}
	for i := 0; i < 500; i++ {
		collector.RecordCacheMiss()
	}

	ratio := collector.calculateCacheHitRatio()

	// Expected: 1000 / (1000 + 500) = 0.6666...
	expected := 1000.0 / 1500.0
	if ratio < expected-0.0001 || ratio > expected+0.0001 {
		t.Errorf("Expected ratio approximately %f, got %f", expected, ratio)
	}
}

func TestPrometheusCollector_Register(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	registry := prometheus.NewRegistry()

	collector := NewPrometheusCollector()

	// Register with custom registry
	registry.MustRegister(collector.saxCacheHits)
	registry.MustRegister(collector.saxCacheMisses)
	registry.MustRegister(collector.saxCacheSize)
	registry.MustRegister(collector.saxCacheHitRatio)

	// Verify metrics are registered by gathering them
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Should have 4 metric families
	if len(metricFamilies) != 4 {
		t.Errorf("Expected 4 metric families, got %d", len(metricFamilies))
	}

	// Verify metric names
	expectedNames := map[string]bool{
		"vector_store_sax_validation_cache_hits_total":   false,
		"vector_store_sax_validation_cache_misses_total": false,
		"vector_store_sax_validation_cache_size":         false,
		"vector_store_sax_validation_cache_hit_ratio":    false,
	}

	for _, mf := range metricFamilies {
		name := mf.GetName()
		if _, exists := expectedNames[name]; exists {
			expectedNames[name] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Expected metric %s not found", name)
		}
	}
}

func TestPrometheusCollector_GaugeFuncIntegration(t *testing.T) {
	// Create a new registry for this test
	registry := prometheus.NewRegistry()

	collector := NewPrometheusCollector()

	// Register with custom registry
	registry.MustRegister(collector.saxCacheHits)
	registry.MustRegister(collector.saxCacheMisses)
	registry.MustRegister(collector.saxCacheSize)
	registry.MustRegister(collector.saxCacheHitRatio)

	// Record some metrics
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheHit()
	collector.RecordCacheMiss()

	// Gather metrics - this should trigger the GaugeFunc calculation
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Find the hit ratio metric
	var hitRatioValue float64
	var found bool

	for _, mf := range metricFamilies {
		if mf.GetName() == "vector_store_sax_validation_cache_hit_ratio" {
			if len(mf.GetMetric()) > 0 {
				hitRatioValue = mf.GetMetric()[0].GetGauge().GetValue()
				found = true
				break
			}
		}
	}

	if !found {
		t.Fatal("Hit ratio metric not found in gathered metrics")
	}

	// Expected ratio: 3 hits / (3 hits + 1 miss) = 0.75
	expected := 0.75
	if hitRatioValue != expected {
		t.Errorf("Expected hit ratio %f, got %f", expected, hitRatioValue)
	}
}

func TestPrometheusCollector_ConcurrentAccess(t *testing.T) {
	collector := NewPrometheusCollector()

	// Test concurrent access to ensure no race conditions
	done := make(chan bool)

	// Goroutine 1: Record hits
	go func() {
		for i := 0; i < 100; i++ {
			collector.RecordCacheHit()
		}
		done <- true
	}()

	// Goroutine 2: Record misses
	go func() {
		for i := 0; i < 50; i++ {
			collector.RecordCacheMiss()
		}
		done <- true
	}()

	// Goroutine 3: Calculate ratio
	go func() {
		for i := 0; i < 10; i++ {
			_ = collector.calculateCacheHitRatio()
		}
		done <- true
	}()

	// Goroutine 4: Record cache size
	go func() {
		for i := 0; i < 20; i++ {
			collector.RecordCacheSize(i)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify final counts
	var metric dto.Metric
	if err := collector.saxCacheHits.Write(&metric); err != nil {
		t.Fatalf("Failed to write hits metric: %v", err)
	}
	hits := metric.GetCounter().GetValue()

	metric.Reset()
	if err := collector.saxCacheMisses.Write(&metric); err != nil {
		t.Fatalf("Failed to write misses metric: %v", err)
	}
	misses := metric.GetCounter().GetValue()

	if hits != 100 {
		t.Errorf("Expected 100 hits, got %f", hits)
	}

	if misses != 50 {
		t.Errorf("Expected 50 misses, got %f", misses)
	}

	// Verify ratio calculation
	ratio := collector.calculateCacheHitRatio()
	expected := 100.0 / 150.0
	if ratio < expected-0.0001 || ratio > expected+0.0001 {
		t.Errorf("Expected ratio approximately %f, got %f", expected, ratio)
	}
}

func TestPrometheusCollector_ImplementsInterface(t *testing.T) {
	var _ SAXMetricsCollector = (*PrometheusCollector)(nil)
}
