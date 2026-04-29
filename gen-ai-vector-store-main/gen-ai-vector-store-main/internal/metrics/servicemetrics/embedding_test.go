/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicEmbeddingMeasurement(t *testing.T) {
	embedding := &servicemetrics.Embedding{}

	measurement := embedding.NewMeasurement("model1", "v1")
	measurement.Start()
	time.Sleep(10 * time.Millisecond)
	measurement.Stop()

	metrics := embedding.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, "model1", metrics[0].ModelID)
	assert.Equal(t, "v1", metrics[0].ModelVersion)
	assert.Equal(t, 1, metrics[0].TotalMeasurementCount)
	assert.Greater(t, metrics[0].TotalExecutionTime, 5*time.Millisecond)
}

func TestMultipleMeasurements(t *testing.T) {
	embedding := &servicemetrics.Embedding{}

	// Create measurements for different models
	models := []struct {
		id, version string
		count       int
	}{
		{"model1", "v1", 3},
		{"model1", "v2", 2},
		{"model2", "v1", 4},
	}

	for _, m := range models {
		for i := 0; i < m.count; i++ {
			measurement := embedding.NewMeasurement(m.id, m.version)
			measurement.Start()
			time.Sleep(5 * time.Millisecond)
			measurement.Stop()
		}
	}

	metrics := embedding.GetMetrics()
	require.Len(t, metrics, 3)

	// Check counts match what we expect
	for _, metric := range metrics {
		for _, m := range models {
			if metric.ModelID == m.id && metric.ModelVersion == m.version {
				assert.Equal(t, m.count, metric.TotalMeasurementCount)
			}
		}
	}
}

func TestEmbeddingEdgeCases(t *testing.T) {
	embedding := &servicemetrics.Embedding{}

	// Stop without start should not panic
	measurement := embedding.NewMeasurement("model1", "v1")
	measurement.Stop()
	assert.Zero(t, measurement.Duration())

	// Metrics should be empty
	metrics := embedding.GetMetrics()
	assert.Empty(t, metrics)

	// Multiple stops should not affect anything
	measurement.Start()
	time.Sleep(5 * time.Millisecond)
	measurement.Stop()
	duration := measurement.Duration()
	measurement.Stop() // Second stop
	assert.Equal(t, duration, measurement.Duration())
}

func TestConcurrentEmbeddingMeasurements(t *testing.T) {
	embedding := &servicemetrics.Embedding{}
	const goroutines = 50
	const measurementsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()

			// Use different model combinations
			modelID := fmt.Sprintf("model%d", i%3)
			modelVersion := fmt.Sprintf("v%d", i%2)

			for j := 0; j < measurementsPerGoroutine; j++ {
				measurement := embedding.NewMeasurement(modelID, modelVersion)
				measurement.Start()
				time.Sleep(time.Millisecond)
				measurement.Stop()
			}
		}(i)
	}

	wg.Wait()

	metrics := embedding.GetMetrics()
	require.Len(t, metrics, 6) // 3 models × 2 versions

	// Calculate total measurements
	totalMeasurements := 0
	for _, metric := range metrics {
		totalMeasurements += metric.TotalMeasurementCount
	}

	assert.Equal(t, goroutines*measurementsPerGoroutine, totalMeasurements)
}

func TestEmbeddingRaceConditions(t *testing.T) {
	embedding := &servicemetrics.Embedding{}
	const concurrent = 100

	var wg sync.WaitGroup
	wg.Add(concurrent)

	// Start all measurements at roughly the same time
	for i := 0; i < concurrent; i++ {
		go func() {
			defer wg.Done()
			m := embedding.NewMeasurement("sameModel", "v1")
			m.Start()
			time.Sleep(5 * time.Millisecond)
			m.Stop()
		}()
	}

	wg.Wait()

	metrics := embedding.GetMetrics()
	require.Len(t, metrics, 1)
	assert.Equal(t, concurrent, metrics[0].TotalMeasurementCount)
}
