// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package tools

import (
	"os"
	"testing"
	"time"
)

func TestParseKeepMode(t *testing.T) {
	tests := []struct {
		name           string
		envValue       string
		expectError    bool
		expectEnabled  bool
		expectDuration time.Duration
	}{
		{
			name:           "Empty KEEP variable",
			envValue:       "",
			expectError:    false,
			expectEnabled:  false,
			expectDuration: 0,
		},
		{
			name:           "Valid duration 5m",
			envValue:       "5m",
			expectError:    false,
			expectEnabled:  true,
			expectDuration: 5 * time.Minute,
		},
		{
			name:           "Valid duration 30s",
			envValue:       "30s",
			expectError:    false,
			expectEnabled:  true,
			expectDuration: 30 * time.Second,
		},
		{
			name:           "Valid duration 1h30m",
			envValue:       "1h30m",
			expectError:    false,
			expectEnabled:  true,
			expectDuration: 90 * time.Minute,
		},
		{
			name:        "Invalid duration - just number",
			envValue:    "5",
			expectError: true,
		},
		{
			name:        "Invalid duration - true",
			envValue:    "true",
			expectError: true,
		},
		{
			name:        "Invalid duration - invalid string",
			envValue:    "invalid",
			expectError: true,
		},
		{
			name:        "Negative duration",
			envValue:    "-5m",
			expectError: true,
		},
		{
			name:        "Zero duration",
			envValue:    "0s",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("KEEP", tt.envValue)
			} else {
				os.Unsetenv("KEEP")
			}
			defer os.Unsetenv("KEEP")

			// Parse KEEP mode
			config, err := ParseKeepMode()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			// Check no error when not expected
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check enabled flag
			if config.Enabled != tt.expectEnabled {
				t.Errorf("Expected Enabled=%v, got %v", tt.expectEnabled, config.Enabled)
			}

			// Check duration
			if config.Duration != tt.expectDuration {
				t.Errorf("Expected Duration=%v, got %v", tt.expectDuration, config.Duration)
			}
		})
	}
}
