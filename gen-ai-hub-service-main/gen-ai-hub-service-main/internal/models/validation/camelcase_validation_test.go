/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"gopkg.in/yaml.v3"
)

// isCamelCase checks if a string follows camelCase naming convention
func isCamelCase(s string) bool {
	if s == "" {
		return false
	}

	// First character must be lowercase letter
	firstRune := rune(s[0])
	if !unicode.IsLower(firstRune) || !unicode.IsLetter(firstRune) {
		return false
	}

	// Check remaining characters - only letters and digits allowed
	// Also check for consecutive uppercase letters which are not allowed in camelCase
	runes := []rune(s)
	for i := 1; i < len(runes); i++ {
		r := runes[i]
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}

		// Check for consecutive uppercase letters
		if unicode.IsUpper(r) && i > 1 && unicode.IsUpper(runes[i-1]) {
			return false
		}
	}

	return true
}

// validatePropertyNames recursively validates all property names in YAML data for camelCase compliance
func validatePropertyNames(data interface{}, path string) error {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if !isCamelCase(key) {
				return fmt.Errorf("property '%s' at path '%s' is not in camelCase", key, buildPath(path, key))
			}

			newPath := buildPath(path, key)
			if err := validatePropertyNames(value, newPath); err != nil {
				return err
			}
		}
	case []interface{}:
		for i, item := range v {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if err := validatePropertyNames(item, itemPath); err != nil {
				return err
			}
		}
		// For primitive types (string, int, bool, etc.) no validation needed
	}
	return nil
}

// buildPath constructs the property path for error reporting
func buildPath(currentPath, key string) string {
	if currentPath == "" {
		return key
	}
	return currentPath + "." + key
}

// findYAMLFiles recursively finds all *.yaml files in the given directory
func findYAMLFiles(root string) ([]string, error) {
	var yamlFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".yaml") {
			yamlFiles = append(yamlFiles, path)
		}

		return nil
	})

	return yamlFiles, err
}

// TestYAMLSpecifications_CamelCaseValidation validates that all property names in YAML specification files follow camelCase convention
func TestYAMLSpecifications_CamelCaseValidation(t *testing.T) {
	specsDir := "../specs"

	// Find all YAML files
	yamlFiles, err := findYAMLFiles(specsDir)
	if err != nil {
		t.Fatalf("Failed to find YAML files: %v", err)
	}

	if len(yamlFiles) == 0 {
		t.Fatal("No YAML files found in specs directory")
	}

	// Validate each YAML file
	for _, filePath := range yamlFiles {
		t.Run(fmt.Sprintf("CamelCase validation for %s", filepath.Base(filePath)), func(t *testing.T) {
			// Read file
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", filePath, err)
			}

			// Parse YAML
			var yamlData interface{}
			if err := yaml.Unmarshal(data, &yamlData); err != nil {
				t.Fatalf("Failed to parse YAML file %s: %v", filePath, err)
			}

			// Validate property names
			if err := validatePropertyNames(yamlData, ""); err != nil {
				t.Errorf("CamelCase validation failed for file %s: %v", filePath, err)
			}
		})
	}
}

// TestIsCamelCase tests the camelCase validation function
func TestIsCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		desc     string
	}{
		// Valid camelCase examples
		{"camelCase", true, "standard camelCase"},
		{"myProperty", true, "camelCase with multiple words"},
		{"maxTokens", true, "camelCase from existing YAML"},
		{"topP", true, "camelCase with single letter"},
		{"id", true, "simple lowercase"},
		{"a", true, "single lowercase letter"},
		{"test123", true, "camelCase with numbers"},
		{"version", true, "single word lowercase"},
		{"functionalCapabilities", true, "long camelCase"},

		// Invalid examples
		{"", false, "empty string"},
		{"CamelCase", false, "starts with uppercase"},
		{"UPPERCASE", false, "all uppercase"},
		{"snake_case", false, "contains underscore"},
		{"kebab-case", false, "contains hyphen"},
		{"with space", false, "contains space"},
		{"with.dot", false, "contains dot"},
		{"123number", false, "starts with number"},
		{"special@char", false, "contains special character"},
		{"mixedUPPER", false, "contains consecutive uppercase"},
		{"endsWith_", false, "ends with underscore"},
		{"middle_underscore", false, "contains underscore in middle"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s: '%s'", test.desc, test.input), func(t *testing.T) {
			result := isCamelCase(test.input)
			if result != test.expected {
				t.Errorf("isCamelCase('%s') = %v, expected %v", test.input, result, test.expected)
			}
		})
	}
}

// TestValidatePropertyNames tests the recursive property validation function
func TestValidatePropertyNames(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid nested structure",
			data: map[string]interface{}{
				"name":    "test",
				"version": "v1",
				"parameters": map[string]interface{}{
					"maxTokens":   1000,
					"temperature": 0.7,
				},
				"endpoints": []interface{}{
					map[string]interface{}{
						"path": "/test",
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid property name at root",
			data: map[string]interface{}{
				"invalid_name": "test",
			},
			expectError: true,
			errorMsg:    "property 'invalid_name' at path 'invalid_name' is not in camelCase",
		},
		{
			name: "invalid property name in nested object",
			data: map[string]interface{}{
				"validName": map[string]interface{}{
					"invalid_nested": "value",
				},
			},
			expectError: true,
			errorMsg:    "property 'invalid_nested' at path 'validName.invalid_nested' is not in camelCase",
		},
		{
			name: "invalid property name in array item",
			data: map[string]interface{}{
				"validArray": []interface{}{
					map[string]interface{}{
						"invalid-item": "value",
					},
				},
			},
			expectError: true,
			errorMsg:    "property 'invalid-item' at path 'validArray[0].invalid-item' is not in camelCase",
		},
		{
			name:        "primitive values should pass",
			data:        "just a string",
			expectError: false,
		},
		{
			name:        "array of primitives should pass",
			data:        []interface{}{"string1", "string2", 123},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePropertyNames(test.data, "")

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if test.errorMsg != "" && !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
