/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package helpers

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileExists(t *testing.T) {
	// Use os.CreateTemp instead of ioutil.TempFile.
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Error creating temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	exists, err := fileExists(tmpFile.Name())
	if err != nil {
		t.Fatalf("Unexpected error calling fileExists: %v", err)
	}
	if !exists {
		t.Errorf("Expected fileExists to return true for existing file")
	}

	// Test a non-existent file.
	exists, err = fileExists("non_existing_file_1234567.txt")
	if err == nil {
		t.Errorf("Expected an error for non-existent file, got nil")
	}
	if exists {
		t.Errorf("Expected fileExists to return false for a non-existent file")
	}
}

func TestSelectValue(t *testing.T) {
	// When first non-empty string is in the middle.
	result := selectValue("", "first", "second")
	if result != "first" {
		t.Errorf("Expected 'first', got '%s'", result)
	}

	// When the first input is non-empty.
	result = selectValue("non-empty", "first", "")
	if result != "non-empty" {
		t.Errorf("Expected 'non-empty', got '%s'", result)
	}

	// When all inputs are empty.
	result = selectValue("", "", "")
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func Test_GetEnabledProviders(t *testing.T) {
	// Create a test logger that we can verify
	ctx := context.Background()

	tests := []struct {
		name                string
		environmentVariable string
		expectedProviders   []string
	}{
		{
			name:                "Multiple providers",
			environmentVariable: "aws,azure,gcp",
			expectedProviders:   []string{"aws", "azure", "gcp"},
		},
		{
			name:                "Single provider",
			environmentVariable: "aws",
			expectedProviders:   []string{"aws"},
		},
		{
			name:                "Empty string",
			environmentVariable: "",
			expectedProviders:   []string{"Azure", "Bedrock", "Vertex"},
		},
		{
			name:                "Providers with whitespace",
			environmentVariable: "aws, azure, gcp",
			expectedProviders:   []string{"aws", "azure", "gcp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment variable value
			originalValue := os.Getenv("ENABLED_PROVIDERS")
			defer func() {
				// Restore original environment variable value after test
				os.Setenv("ENABLED_PROVIDERS", originalValue)
			}()

			// Set environment variable for test
			os.Setenv("ENABLED_PROVIDERS", tt.environmentVariable)

			// Call the function
			result := GetEnabledProviders(ctx)

			// Verify the result, ignoring order of elements
			assert.ElementsMatch(t, tt.expectedProviders, result)
		})
	}
}
