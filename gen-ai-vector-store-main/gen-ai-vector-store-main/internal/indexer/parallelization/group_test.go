/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package parallelization

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimitedGroupConcurrency(t *testing.T) {
	const maxConcurrent = 2
	const totalTasks = 5

	group := Limited(maxConcurrent)
	subgroup, _ := group.WithContext(context.Background())

	var currentlyRunning int64
	var maxObservedConcurrency int64
	var completedTasks int64

	for i := 0; i < totalTasks; i++ {
		subgroup.Go(func(ctx context.Context) error {
			// Track concurrency
			current := atomic.AddInt64(&currentlyRunning, 1)
			defer atomic.AddInt64(&currentlyRunning, -1)

			// Update max observed concurrency
			for {
				max := atomic.LoadInt64(&maxObservedConcurrency)
				if current <= max || atomic.CompareAndSwapInt64(&maxObservedConcurrency, max, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(100 * time.Millisecond)

			atomic.AddInt64(&completedTasks, 1)
			return nil
		})
	}

	err := subgroup.Wait()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify all tasks completed
	if completedTasks != totalTasks {
		t.Errorf("Expected %d completed tasks, got %d", totalTasks, completedTasks)
	}

	// Verify concurrency was limited
	if maxObservedConcurrency > maxConcurrent {
		t.Errorf("Expected max concurrency %d, observed %d", maxConcurrent, maxObservedConcurrency)
	}

	// Verify at least some concurrency was achieved
	if maxObservedConcurrency < 1 {
		t.Error("Expected at least some concurrency, but none was observed")
	}
}

func TestSubgroupContextCancellation(t *testing.T) {
	group := Limited(10)
	ctx, cancel := context.WithCancel(context.Background())
	subgroup, _ := group.WithContext(ctx)

	var startedTasks int64
	var completedTasks int64

	// Start multiple tasks
	for i := 0; i < 3; i++ {
		subgroup.Go(func(ctx context.Context) error {
			atomic.AddInt64(&startedTasks, 1)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				atomic.AddInt64(&completedTasks, 1)
				return nil
			}
		})
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := subgroup.Wait()

	// Should get context cancellation error
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	// Verify tasks were started but not all completed due to cancellation
	if startedTasks == 0 {
		t.Error("Expected some tasks to start")
	}

	if completedTasks == startedTasks {
		t.Error("Expected some tasks to be cancelled before completion")
	}
}
