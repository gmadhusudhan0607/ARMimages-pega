/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package errors

import (
	"errors"
	"testing"
)

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "context.DeadlineExceeded",
			err:      errors.New("context deadline exceeded"),
			expected: true,
		},
		{
			name:     "client timeout exceeded",
			err:      errors.New("net/http: request canceled (Client.Timeout exceeded while awaiting headers)"),
			expected: true,
		},
		{
			name:     "timeout in error message",
			err:      errors.New("operation timeout"),
			expected: true,
		},
		{
			name:     "timed out in error message",
			err:      errors.New("request timed out"),
			expected: true,
		},
		{
			name:     "deadline exceeded in error message",
			err:      errors.New("deadline exceeded"),
			expected: true,
		},
		{
			name:     "uppercase timeout",
			err:      errors.New("TIMEOUT occurred"),
			expected: true,
		},
		{
			name:     "non-timeout error",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			result := IsTimeout(tt.err)
			if result != tt.expected {
				t.Errorf("IsTimeout() = %v, expected %v for error: %v", result, tt.expected, tt.err)
			}
		})
	}
}
