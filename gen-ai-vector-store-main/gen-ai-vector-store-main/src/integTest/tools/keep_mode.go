// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package tools

import (
	"context"
	"fmt"
	"os"
	"time"
)

// KeepModeConfig holds the parsed KEEP mode configuration
type KeepModeConfig struct {
	Enabled  bool
	Duration time.Duration
}

// ParseKeepMode parses the KEEP environment variable and returns the configuration.
// Returns an error if KEEP is set but cannot be parsed as a valid duration.
// Supported formats:
//   - "5m" - keep for 5 minutes
//   - "30s" - keep for 30 seconds
//   - "1h30m" - keep for 1 hour 30 minutes
//   - Any valid Go duration format
//
// If KEEP is not set or empty, returns a config with Enabled=false.
func ParseKeepMode() (*KeepModeConfig, error) {
	keepValue := os.Getenv("KEEP")

	// If KEEP is not set or empty, return disabled config
	if keepValue == "" {
		return &KeepModeConfig{
			Enabled:  false,
			Duration: 0,
		}, nil
	}

	// Try to parse as duration
	duration, err := time.ParseDuration(keepValue)
	if err != nil {
		return nil, fmt.Errorf("invalid KEEP duration '%s': %w. Valid examples: 5m, 30s, 1h30m", keepValue, err)
	}

	// Validate that duration is positive
	if duration <= 0 {
		return nil, fmt.Errorf("KEEP duration must be positive, got: %s", keepValue)
	}

	return &KeepModeConfig{
		Enabled:  true,
		Duration: duration,
	}, nil
}

// ScheduleCleanup schedules a cleanup function to run after the specified duration
// Returns immediately, running the cleanup in a background goroutine
func (kmc *KeepModeConfig) ScheduleCleanup(ctx context.Context, resourceName string, cleanupFunc func() error) {
	if !kmc.Enabled {
		return
	}

	go func() {
		fmt.Println("  ================================================================================")
		fmt.Printf("    KEEP mode: %s will remain running for %s\n", resourceName, kmc.Duration)
		fmt.Printf("    Cleanup scheduled at: %s\n", time.Now().Add(kmc.Duration).Format(time.RFC3339))

		// Wait for the duration or context cancellation
		select {
		case <-time.After(kmc.Duration):
			fmt.Printf("\nKEEP timeout reached for %s, cleaning up...\n", resourceName)
			if err := cleanupFunc(); err != nil {
				fmt.Printf("Error during scheduled cleanup of %s: %v\n", resourceName, err)
			} else {
				fmt.Printf("Successfully cleaned up %s after KEEP timeout\n", resourceName)
			}
		case <-ctx.Done():
			fmt.Printf("\nContext cancelled, skipping scheduled cleanup of %s\n", resourceName)
		}
	}()
}
