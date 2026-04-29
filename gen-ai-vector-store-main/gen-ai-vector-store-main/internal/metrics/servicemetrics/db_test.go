/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics_test

import (
	"sync"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/stretchr/testify/assert"
)

func TestBasicDBMeasurement(t *testing.T) {
	db := &servicemetrics.DB{}

	measurement := db.NewMeasurement()
	measurement.Start()
	time.Sleep(10 * time.Millisecond)
	measurement.Stop()

	executionTime := db.QueryExecutionTime()
	assert.Greater(t, executionTime, 5*time.Millisecond)
}

func TestMultipleDBMeasurements(t *testing.T) {
	db := &servicemetrics.DB{}
	const numMeasurements = 5
	expectedDuration := time.Duration(0)

	for i := 0; i < numMeasurements; i++ {
		measurement := db.NewMeasurement()
		measurement.Start()
		sleepTime := 5 * time.Millisecond
		time.Sleep(sleepTime)
		measurement.Stop()
		expectedDuration += sleepTime
	}

	executionTime := db.QueryExecutionTime()
	assert.Greater(t, executionTime, expectedDuration/2)
	assert.Less(t, executionTime, expectedDuration*10)
}

func TestDBEdgeCases(t *testing.T) {
	db := &servicemetrics.DB{}

	// Stop without start should not panic
	measurement := db.NewMeasurement()
	measurement.Stop()
	assert.Zero(t, db.QueryExecutionTime())

	// Start time should be set after Start() is called
	measurement = db.NewMeasurement()
	assert.True(t, measurement.StartTime().IsZero())
	measurement.Start()
	assert.False(t, measurement.StartTime().IsZero())

	// Multiple stops should not affect anything
	measurement = db.NewMeasurement()
	measurement.Start()
	time.Sleep(5 * time.Millisecond)
	measurement.Stop()
	initialExecutionTime := db.QueryExecutionTime()
	measurement.Stop() // Second stop
	assert.Equal(t, initialExecutionTime, db.QueryExecutionTime())
}

func TestConcurrentDBMeasurements(t *testing.T) {
	db := &servicemetrics.DB{}
	const goroutines = 50
	const measurementsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < measurementsPerGoroutine; j++ {
				measurement := db.NewMeasurement()
				measurement.Start()
				time.Sleep(time.Millisecond)
				measurement.Stop()
			}
		}()
	}

	wg.Wait()

	// Total execution time should be approximately the sum of all measurements
	// (goroutines * measurementsPerGoroutine * 1ms)
	// but could vary due to scheduling and overhead
	totalExpectedTime := time.Duration(goroutines*measurementsPerGoroutine) * time.Millisecond
	actualTime := db.QueryExecutionTime()

	// We use a relaxed comparison because of timing variability in concurrent execution
	assert.Greater(t, actualTime, totalExpectedTime/2)
	assert.Less(t, actualTime, totalExpectedTime*10)
}

func TestDBRaceConditions(t *testing.T) {
	db := &servicemetrics.DB{}
	const concurrent = 100

	// Create a bunch of measurements but don't stop them yet
	measurements := make([]servicemetrics.DBMeasurement, concurrent)
	for i := 0; i < concurrent; i++ {
		measurements[i] = db.NewMeasurement()
		measurements[i].Start()
	}

	// Now stop them all concurrently
	var wg sync.WaitGroup
	wg.Add(concurrent)

	for i := 0; i < concurrent; i++ {
		go func(idx int) {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			measurements[idx].Stop()
		}(i)
	}

	wg.Wait()

	// All measurements contributed to the total time
	executionTime := db.QueryExecutionTime()
	assert.Greater(t, executionTime, 0*time.Millisecond)
}

func TestDBMeasurementWithoutLockContention(t *testing.T) {
	db := &servicemetrics.DB{}
	const concurrent = 50

	var wg sync.WaitGroup
	wg.Add(concurrent)

	// Half of goroutines just read the execution time
	for i := 0; i < concurrent/2; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = db.QueryExecutionTime()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Other half create and use measurements
	for i := 0; i < concurrent/2; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				measurement := db.NewMeasurement()
				measurement.Start()
				time.Sleep(2 * time.Millisecond)
				measurement.Stop()
			}
		}()
	}

	wg.Wait()

	// Make sure we captured something
	assert.Greater(t, db.QueryExecutionTime(), 0*time.Millisecond)
}
