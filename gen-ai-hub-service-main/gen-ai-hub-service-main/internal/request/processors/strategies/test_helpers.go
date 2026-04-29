/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package strategies

// Test helper functions shared across strategy tests

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

// floatPtr returns a pointer to a float64
func floatPtr(f float64) *float64 {
	return &f
}

// float64Ptr returns a pointer to a float64 (alias for compatibility)
func float64Ptr(f float64) *float64 {
	return &f
}
