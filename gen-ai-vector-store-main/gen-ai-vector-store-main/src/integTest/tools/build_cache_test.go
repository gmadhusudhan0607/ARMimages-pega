// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildCache(t *testing.T) {
	// Create a test context
	ctx := context.Background()

	// Reset build cache for clean test state
	ResetBuildCache()

	// Get the build cache
	cache := GetBuildCache()

	// Create a test config
	config := ServiceConfig{
		SourcePath:  "./cmd/background",
		BinaryPath:  "bin/background-test",
		ServiceName: "background-test",
	}

	// Test 1: First build should create binary
	t.Run("FirstBuild", func(t *testing.T) {
		binaryPath, err := cache.EnsureBinary(ctx, config)
		if err != nil {
			t.Fatalf("First build failed: %v", err)
		}

		// Verify binary exists
		if _, err := os.Stat(binaryPath); err != nil {
			t.Fatalf("Binary not created: %v", err)
		}

		t.Logf("First build succeeded: %s", binaryPath)
	})

	// Test 2: Second build should use cache (no rebuild)
	t.Run("CachedBuild", func(t *testing.T) {
		// Get the binary modification time before
		projectRoot, _ := findProjectRoot()
		binaryPath := filepath.Join(projectRoot, config.BinaryPath)
		beforeStat, _ := os.Stat(binaryPath)
		beforeModTime := beforeStat.ModTime()

		// Small delay to ensure time difference would be detectable
		time.Sleep(100 * time.Millisecond)

		// Try to build again
		_, err := cache.EnsureBinary(ctx, config)
		if err != nil {
			t.Fatalf("Cached build failed: %v", err)
		}

		// Verify binary was NOT rebuilt (modification time unchanged)
		afterStat, _ := os.Stat(binaryPath)
		afterModTime := afterStat.ModTime()

		if !beforeModTime.Equal(afterModTime) {
			t.Errorf("Binary was rebuilt when it should have been cached (before: %v, after: %v)",
				beforeModTime, afterModTime)
		} else {
			t.Logf("Binary correctly reused from cache")
		}
	})

	// Test 3: Force rebuild
	t.Run("ForceRebuild", func(t *testing.T) {
		// Get the binary modification time before
		projectRoot, _ := findProjectRoot()
		binaryPath := filepath.Join(projectRoot, config.BinaryPath)
		beforeStat, _ := os.Stat(binaryPath)
		beforeModTime := beforeStat.ModTime()

		// Small delay to ensure time difference
		time.Sleep(100 * time.Millisecond)

		// Enable force rebuild
		cache.SetForceRebuild(true)
		defer cache.SetForceRebuild(false)

		// Try to build again
		_, err := cache.EnsureBinary(ctx, config)
		if err != nil {
			t.Fatalf("Force rebuild failed: %v", err)
		}

		// Verify binary WAS rebuilt (modification time changed)
		afterStat, _ := os.Stat(binaryPath)
		afterModTime := afterStat.ModTime()

		if beforeModTime.Equal(afterModTime) {
			t.Errorf("Binary was not rebuilt when force rebuild was enabled")
		} else {
			t.Logf("Binary correctly rebuilt with force flag")
		}
	})

	// Test 4: Check build info
	t.Run("BuildInfo", func(t *testing.T) {
		info := cache.GetBuildInfo()
		if len(info) == 0 {
			t.Errorf("Build info should not be empty")
		}

		for service, status := range info {
			t.Logf("Service %s: %s", service, status)
		}
	})

	// Test 5: Clear cache
	t.Run("ClearCache", func(t *testing.T) {
		cache.ClearCache()

		info := cache.GetBuildInfo()
		if len(info) != 0 {
			t.Errorf("Build info should be empty after clear, got %d entries", len(info))
		} else {
			t.Logf("Cache cleared successfully")
		}
	})
}

func TestBuildCacheConcurrency(t *testing.T) {
	// This test verifies that the build cache is thread-safe
	ctx := context.Background()
	ResetBuildCache()

	cache := GetBuildCache()

	config := ServiceConfig{
		SourcePath:  "./cmd/background",
		BinaryPath:  "bin/background-test",
		ServiceName: "background-test",
	}

	// Try to build the same binary from multiple goroutines
	done := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			_, err := cache.EnsureBinary(ctx, config)
			done <- err
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		if err := <-done; err != nil {
			t.Errorf("Goroutine failed: %v", err)
		}
	}

	t.Logf("Concurrent access to build cache succeeded")
}
