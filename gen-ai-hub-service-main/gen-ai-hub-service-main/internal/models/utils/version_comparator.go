/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// VersionComparator provides intelligent version comparison for different model version formats
type VersionComparator struct {
	// GPT model version to release date mapping
	gptVersionDates map[string]string
}

// NewVersionComparator creates a new version comparator
func NewVersionComparator() *VersionComparator {
	return &VersionComparator{
		// I "love" Microsoft :)
		gptVersionDates: map[string]string{
			"0301": "2023-03-01", // gpt-3.5-turbo-0301
			"0613": "2023-06-13", // gpt-3.5-turbo-0613, gpt-3.5-turbo-16k-0613
			"0914": "2023-09-14", // gpt-3.5-turbo-instruct-0914
			"1106": "2023-11-01", // gpt-3.5-turbo-1106 (November 2023)
			"0125": "2024-01-25", // gpt-3.5-turbo-0125
		},
	}
}

// Compare compares two version strings and returns:
// -1 if v1 < v2 (v1 is older)
//
//	0 if v1 == v2 (versions are equal)
//	1 if v1 > v2 (v1 is newer)
func (vc *VersionComparator) Compare(v1, v2 string) int {
	if v1 == v2 {
		return 0
	}

	// Handle GPT model versions - both must be GPT versions for special comparison
	if vc.isGPTVersion(v1) && vc.isGPTVersion(v2) {
		return vc.compareGPTVersions(v1, v2)
	}

	// Handle semantic versions - both must be semantic versions for special comparison
	if vc.isSemanticVersion(v1) && vc.isSemanticVersion(v2) {
		return vc.compareSemanticVersions(v1, v2)
	}

	// Handle numeric versions - both must be numeric versions for special comparison
	if vc.isNumericVersion(v1) && vc.isNumericVersion(v2) {
		return vc.compareNumericVersions(v1, v2)
	}

	// Fallback to lexicographic comparison for mixed types or unknown formats
	if v1 < v2 {
		return -1
	}
	return 1
}

// IsNewer returns true if v1 is newer than v2
func (vc *VersionComparator) IsNewer(v1, v2 string) bool {
	return vc.Compare(v1, v2) > 0
}

// isGPTVersion checks if a version string matches GPT model version format
func (vc *VersionComparator) isGPTVersion(version string) bool {
	// Check if the version is in our known GPT versions map
	_, exists := vc.gptVersionDates[version]
	return exists
}

// compareGPTVersions compares GPT model versions based on their actual release dates
func (vc *VersionComparator) compareGPTVersions(v1, v2 string) int {
	date1, exists1 := vc.gptVersionDates[v1]
	date2, exists2 := vc.gptVersionDates[v2]

	// If either version is not found, fall back to string comparison
	if !exists1 || !exists2 {
		if v1 < v2 {
			return -1
		} else if v1 > v2 {
			return 1
		}
		return 0
	}

	// Compare release dates
	if date1 < date2 {
		return -1
	} else if date1 > date2 {
		return 1
	}
	return 0
}

// isSemanticVersion checks if a version string matches semantic versioning format
func (vc *VersionComparator) isSemanticVersion(version string) bool {
	// Basic semantic version pattern: X.Y.Z or X.Y
	matched, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?(-[a-zA-Z0-9\-\.]+)?(\+[a-zA-Z0-9\-\.]+)?$`, version)
	return matched
}

// compareSemanticVersions compares semantic versions
func (vc *VersionComparator) compareSemanticVersions(v1, v2 string) int {
	// Split versions into parts
	parts1 := vc.parseSemanticVersion(v1)
	parts2 := vc.parseSemanticVersion(v2)

	// Compare each part
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		if parts1[i] > parts2[i] {
			return 1
		} else if parts1[i] < parts2[i] {
			return -1
		}
	}

	// If all compared parts are equal, longer version is newer
	if len(parts1) > len(parts2) {
		return 1
	} else if len(parts1) < len(parts2) {
		return -1
	}
	return 0
}

// parseSemanticVersion parses a semantic version into numeric parts
func (vc *VersionComparator) parseSemanticVersion(version string) []int {
	// Remove pre-release and build metadata
	version = strings.Split(version, "-")[0]
	version = strings.Split(version, "+")[0]

	parts := strings.Split(version, ".")
	result := make([]int, len(parts))
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			num = 0
		}
		result[i] = num
	}
	return result
}

// isNumericVersion checks if a version string is purely numeric
func (vc *VersionComparator) isNumericVersion(version string) bool {
	_, err := strconv.Atoi(version)
	return err == nil
}

// compareNumericVersions compares purely numeric versions
func (vc *VersionComparator) compareNumericVersions(v1, v2 string) int {
	num1, _ := strconv.Atoi(v1)
	num2, _ := strconv.Atoi(v2)

	if num1 > num2 {
		return 1
	} else if num1 < num2 {
		return -1
	}
	return 0
}
