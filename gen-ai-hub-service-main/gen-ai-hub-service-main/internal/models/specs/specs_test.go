/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package specs

import (
	"embed"
	"testing"
)

func TestEmbeddedModels(t *testing.T) {
	fs := ModelsSpecs

	// Verify that we get a valid embed.FS
	if fs == (embed.FS{}) {
		t.Fatal("ModelsSpecs is empty embed.FS")
	}

	// Test that we can read directories
	entries, err := fs.ReadDir(".")
	if err != nil {
		t.Fatalf("Failed to read root directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("Models directory is empty")
	}

	// Verify expected infrastructure directories exist
	expectedDirs := []string{"aws", "gcp", "azure"}
	foundDirs := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			foundDirs[entry.Name()] = true
		}
	}

	for _, expectedDir := range expectedDirs {
		if !foundDirs[expectedDir] {
			t.Errorf("Expected directory %s not found in models", expectedDir)
		}
	}
}

func TestEmbeddedModelsContent(t *testing.T) {
	fs := ModelsSpecs

	// Test specific model files exist
	testFiles := []string{
		"aws/bedrock/anthropic/claude.yaml",
		"gcp/vertex/google/gemini-1.0.yaml",
		"azure/azure/openai/gpt-4.yaml",
	}

	for _, file := range testFiles {
		data, err := fs.ReadFile(file)
		if err != nil {
			t.Errorf("Failed to read embedded file %s: %v", file, err)
			continue
		}

		if len(data) == 0 {
			t.Errorf("Embedded file %s is empty", file)
		}

		// Verify it's a YAML file with expected structure
		content := string(data)
		if !contains(content, "infrastructure:") || !contains(content, "provider:") {
			t.Errorf("File %s doesn't contain expected YAML structure", file)
		}
	}
}

func TestEmbeddedModelsDirectoryStructure(t *testing.T) {
	fs := ModelsSpecs

	// Test infrastructure directories
	infrastructures := []string{"aws", "gcp", "azure"}

	for _, infra := range infrastructures {
		path := infra
		entries, err := fs.ReadDir(path)
		if err != nil {
			t.Errorf("Failed to read %s directory: %v", path, err)
			continue
		}

		if len(entries) == 0 {
			t.Errorf("Infrastructure directory %s is empty", path)
		}

		// Verify each entry is a directory (provider)
		for _, entry := range entries {
			if !entry.IsDir() {
				t.Errorf("Expected directory but found file in %s: %s", path, entry.Name())
			}
		}
	}
}

func BenchmarkEmbeddedModels(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ModelsSpecs
	}
}

func BenchmarkEmbeddedFileAccess(b *testing.B) {
	fs := ModelsSpecs

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fs.ReadFile("aws/bedrock/anthropic/claude.yaml")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
