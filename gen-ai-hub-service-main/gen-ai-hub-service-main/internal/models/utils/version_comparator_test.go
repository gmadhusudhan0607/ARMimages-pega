/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package utils

import (
	"testing"
)

func TestVersionComparator_GPTVersions(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "0125 is newer than 1106",
			v1:       "0125", // January 25, 2024
			v2:       "1106", // November 1, 2023
			expected: 1,      // v1 > v2
		},
		{
			name:     "1106 is older than 0125",
			v1:       "1106", // November 1, 2023
			v2:       "0125", // January 25, 2024
			expected: -1,     // v1 < v2
		},
		{
			name:     "Same versions are equal",
			v1:       "0125",
			v2:       "0125",
			expected: 0,
		},
		{
			name:     "0613 is older than 1106",
			v1:       "0613", // June 13, 2023
			v2:       "1106", // November 1, 2023
			expected: -1,     // v1 < v2
		},
		{
			name:     "0301 is oldest",
			v1:       "0301", // March 1, 2023
			v2:       "0613", // June 13, 2023
			expected: -1,     // v1 < v2
		},
		{
			name:     "0914 is newer than 0613",
			v1:       "0914", // September 14, 2023
			v2:       "0613", // June 13, 2023
			expected: 1,      // v1 > v2
		},
		{
			name:     "0914 is older than 1106",
			v1:       "0914", // September 14, 2023
			v2:       "1106", // November 1, 2023
			expected: -1,     // v1 < v2
		},
		{
			name:     "Complete chronological order test",
			v1:       "0301", // March 1, 2023 (oldest)
			v2:       "0125", // January 25, 2024 (newest)
			expected: -1,     // v1 < v2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vc.Compare(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("Compare(%s, %s) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestVersionComparator_IsNewer(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
	}{
		{
			name:     "0125 is newer than 1106",
			v1:       "0125", // January 25, 2024
			v2:       "1106", // November 1, 2023
			expected: true,
		},
		{
			name:     "1106 is not newer than 0125",
			v1:       "1106", // November 1, 2023
			v2:       "0125", // January 25, 2024
			expected: false,
		},
		{
			name:     "Same versions are not newer",
			v1:       "0125",
			v2:       "0125",
			expected: false,
		},
		{
			name:     "0914 is newer than 0613",
			v1:       "0914", // September 14, 2023
			v2:       "0613", // June 13, 2023
			expected: true,
		},
		{
			name:     "0301 is not newer than any other version",
			v1:       "0301", // March 1, 2023 (oldest)
			v2:       "0613", // June 13, 2023
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vc.IsNewer(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("IsNewer(%s, %s) = %t, expected %t", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestVersionComparator_SemanticVersions(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "2.0.0 is newer than 1.0.0",
			v1:       "2.0.0",
			v2:       "1.0.0",
			expected: 1,
		},
		{
			name:     "1.1.0 is newer than 1.0.0",
			v1:       "1.1.0",
			v2:       "1.0.0",
			expected: 1,
		},
		{
			name:     "1.0.1 is newer than 1.0.0",
			v1:       "1.0.1",
			v2:       "1.0.0",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vc.Compare(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("Compare(%s, %s) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestVersionComparator_NumericVersions(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "10 is newer than 2",
			v1:       "10",
			v2:       "2",
			expected: 1,
		},
		{
			name:     "2 is older than 10",
			v1:       "2",
			v2:       "10",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vc.Compare(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("Compare(%s, %s) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestVersionComparator_isGPTVersion(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		version  string
		expected bool
	}{
		{"0125", true},   // Valid GPT version (gpt-3.5-turbo-0125)
		{"1106", true},   // Valid GPT version (gpt-3.5-turbo-1106)
		{"0613", true},   // Valid GPT version (gpt-3.5-turbo-0613)
		{"0301", true},   // Valid GPT version (gpt-3.5-turbo-0301)
		{"0914", true},   // Valid GPT version (gpt-3.5-turbo-instruct-0914)
		{"1301", false},  // Not a known GPT version
		{"0001", false},  // Not a known GPT version
		{"1212", false},  // Not a known GPT version
		{"0314", false},  // Not a known GPT version (should be 0301)
		{"123", false},   // Too short
		{"12345", false}, // Too long
		{"abcd", false},  // Non-numeric
		{"1.0", false},   // Semantic version
		{"2024", false},  // Not a known GPT version
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := vc.isGPTVersion(tt.version)
			if result != tt.expected {
				t.Errorf("isGPTVersion(%s) = %t, expected %t", tt.version, result, tt.expected)
			}
		})
	}
}

func TestVersionComparator_GPTVersionsChronologicalOrder(t *testing.T) {
	vc := NewVersionComparator()

	// Test complete chronological ordering based on actual release dates
	// Expected order from oldest to newest:
	// 0301 (March 1, 2023) -> 0613 (June 13, 2023) -> 0914 (September 14, 2023) -> 1106 (November 1, 2023) -> 0125 (January 25, 2024)
	versions := []string{"0301", "0613", "0914", "1106", "0125"}

	for i := 0; i < len(versions); i++ {
		for j := i + 1; j < len(versions); j++ {
			older := versions[i]
			newer := versions[j]

			// Test that older version is indeed older
			result := vc.Compare(older, newer)
			if result != -1 {
				t.Errorf("Expected %s to be older than %s, but Compare(%s, %s) = %d", older, newer, older, newer, result)
			}

			// Test that newer version is indeed newer
			result = vc.Compare(newer, older)
			if result != 1 {
				t.Errorf("Expected %s to be newer than %s, but Compare(%s, %s) = %d", newer, older, newer, older, result)
			}

			// Test IsNewer method
			if !vc.IsNewer(newer, older) {
				t.Errorf("Expected IsNewer(%s, %s) to be true", newer, older)
			}

			if vc.IsNewer(older, newer) {
				t.Errorf("Expected IsNewer(%s, %s) to be false", older, newer)
			}
		}
	}
}

func TestVersionComparator_MixedVersionTypes(t *testing.T) {
	vc := NewVersionComparator()

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "GPT version vs semantic version falls back to lexicographic",
			v1:       "0125",  // GPT version
			v2:       "1.0.0", // Semantic version
			expected: -1,      // "0125" < "1.0.0" lexicographically
		},
		{
			name:     "GPT version vs numeric version falls back to lexicographic",
			v1:       "0125", // GPT version
			v2:       "2",    // Numeric version
			expected: 1,      // "0125" > "2" lexicographically (string comparison)
		},
		{
			name:     "Unknown version format falls back to lexicographic",
			v1:       "unknown1",
			v2:       "unknown2",
			expected: -1, // "unknown1" < "unknown2" lexicographically
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vc.Compare(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("Compare(%s, %s) = %d, expected %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}
