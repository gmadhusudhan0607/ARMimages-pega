/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics_test

import (
	"sync"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/stretchr/testify/assert"
)

func TestBasicResponseMetrics(t *testing.T) {
	response := &servicemetrics.Response{}

	// Test initial value
	assert.Equal(t, 0, response.ItemsReturned())

	// Test setting a value
	response.SetItemsReturned(10)
	assert.Equal(t, 10, response.ItemsReturned())

	// Test updating the value
	response.SetItemsReturned(20)
	assert.Equal(t, 20, response.ItemsReturned())
}

func TestResponseRaceConditions(t *testing.T) {
	response := &servicemetrics.Response{}
	const concurrent = 10

	var wg sync.WaitGroup
	wg.Add(concurrent * 2)

	// Half of goroutines set values, half read values
	for i := 0; i < concurrent; i++ {
		// Writers
		go func(val int) {
			defer wg.Done()
			response.SetItemsReturned(val)
		}(i)

		// Readers
		go func() {
			defer wg.Done()
			_ = response.ItemsReturned()
		}()
	}

	wg.Wait()

	// Final value should be in valid range
	finalValue := response.ItemsReturned()
	assert.GreaterOrEqual(t, finalValue, 0)
	assert.Less(t, finalValue, concurrent)
}
