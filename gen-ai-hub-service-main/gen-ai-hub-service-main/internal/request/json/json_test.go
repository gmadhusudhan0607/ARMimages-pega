/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package json

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestGetValueByPath(t *testing.T) {
	testJSON := `{
		"generationConfig": {
			"maxOutputTokens": 1024,
			"temperature": 0.7,
			"nested": {
				"deep": {
					"value": "found"
				}
			}
		},
		"model": "gpt-4",
		"messages": [
			{"role": "user", "content": "hello"},
			{"role": "assistant", "content": "hi"}
		],
		"numbers": {
			"integer": 42,
			"float": 3.14,
			"zero": 0
		}
	}`

	tests := []struct {
		name     string
		path     string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "simple path",
			path:     "model",
			expected: "gpt-4",
			wantErr:  false,
		},
		{
			name:     "nested path",
			path:     "generationConfig.maxOutputTokens",
			expected: json.Number("1024"),
			wantErr:  false,
		},
		{
			name:     "deep nested path",
			path:     "generationConfig.nested.deep.value",
			expected: "found",
			wantErr:  false,
		},
		{
			name:     "array access",
			path:     "messages.0.role",
			expected: "user",
			wantErr:  false,
		},
		{
			name:     "array access second element",
			path:     "messages.1.content",
			expected: "hi",
			wantErr:  false,
		},
		{
			name:     "integer value",
			path:     "numbers.integer",
			expected: json.Number("42"),
			wantErr:  false,
		},
		{
			name:     "float value",
			path:     "numbers.float",
			expected: json.Number("3.14"),
			wantErr:  false,
		},
		{
			name:     "zero value",
			path:     "numbers.zero",
			expected: json.Number("0"),
			wantErr:  false,
		},
		{
			name:    "non-existent key",
			path:    "nonexistent",
			wantErr: true,
		},
		{
			name:    "non-existent nested key",
			path:    "generationConfig.nonexistent",
			wantErr: true,
		},
		{
			name:    "array index out of bounds",
			path:    "messages.5.role",
			wantErr: true,
		},
		{
			name:    "invalid array index",
			path:    "messages.invalid.role",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetValueByPath([]byte(testJSON), tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetValueByPath() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GetValueByPath() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("GetValueByPath() = %v (%T), expected %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestGetValueByPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		requestBody []byte
		path        string
		wantErr     bool
	}{
		{
			name:        "empty request body",
			requestBody: []byte{},
			path:        "test",
			wantErr:     true,
		},
		{
			name:        "invalid JSON",
			requestBody: []byte(`{"invalid": json}`),
			path:        "test",
			wantErr:     true,
		},
		{
			name:        "null JSON",
			requestBody: []byte(`null`),
			path:        "test",
			wantErr:     true,
		},
		{
			name:        "array root",
			requestBody: []byte(`[1, 2, 3]`),
			path:        "0",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetValueByPath(tt.requestBody, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValueByPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetValueByPath(t *testing.T) {
	baseJSON := `{
		"generationConfig": {
			"maxOutputTokens": 1024,
			"temperature": 0.7
		},
		"model": "gpt-4",
		"existing": {
			"nested": "value"
		}
	}`

	tests := []struct {
		name     string
		path     string
		newValue interface{}
		wantErr  bool
		validate func(t *testing.T, result []byte)
	}{
		{
			name:     "update existing value",
			path:     "generationConfig.maxOutputTokens",
			newValue: 2048,
			wantErr:  false,
			validate: func(t *testing.T, result []byte) {
				value, err := GetValueByPath(result, "generationConfig.maxOutputTokens")
				if err != nil {
					t.Errorf("Failed to get updated value: %v", err)
					return
				}
				// Value could be json.Number or float64 depending on how it was parsed
				var numVal float64
				switch v := value.(type) {
				case json.Number:
					var err error
					numVal, err = v.Float64()
					if err != nil {
						t.Errorf("Failed to convert json.Number to float64: %v", err)
						return
					}
				case float64:
					numVal = v
				default:
					t.Errorf("Expected numeric value, got %v (%T)", value, value)
					return
				}
				if numVal != 2048 {
					t.Errorf("Expected 2048, got %v", numVal)
				}
			},
		},
		{
			name:     "create new nested path",
			path:     "newConfig.newValue",
			newValue: "test",
			wantErr:  false,
			validate: func(t *testing.T, result []byte) {
				value, err := GetValueByPath(result, "newConfig.newValue")
				if err != nil {
					t.Errorf("Failed to get new value: %v", err)
					return
				}
				if value != "test" {
					t.Errorf("Expected 'test', got %v", value)
				}
			},
		},
		{
			name:     "override existing nested object",
			path:     "existing.nested",
			newValue: map[string]interface{}{"new": "structure"},
			wantErr:  false,
			validate: func(t *testing.T, result []byte) {
				value, err := GetValueByPath(result, "existing.nested.new")
				if err != nil {
					t.Errorf("Failed to get overridden value: %v", err)
					return
				}
				if value != "structure" {
					t.Errorf("Expected 'structure', got %v", value)
				}
			},
		},
		{
			name:     "set root level value",
			path:     "newRoot",
			newValue: 123,
			wantErr:  false,
			validate: func(t *testing.T, result []byte) {
				value, err := GetValueByPath(result, "newRoot")
				if err != nil {
					t.Errorf("Failed to get root value: %v", err)
					return
				}
				// Value could be json.Number or float64 depending on how it was parsed
				var numVal float64
				switch v := value.(type) {
				case json.Number:
					var err error
					numVal, err = v.Float64()
					if err != nil {
						t.Errorf("Failed to convert json.Number to float64: %v", err)
						return
					}
				case float64:
					numVal = v
				default:
					t.Errorf("Expected numeric value, got %v (%T)", value, value)
					return
				}
				if numVal != 123 {
					t.Errorf("Expected 123, got %v", numVal)
				}
			},
		},
		{
			name:     "deep nested creation",
			path:     "level1.level2.level3.level4",
			newValue: "deep",
			wantErr:  false,
			validate: func(t *testing.T, result []byte) {
				value, err := GetValueByPath(result, "level1.level2.level3.level4")
				if err != nil {
					t.Errorf("Failed to get deep nested value: %v", err)
					return
				}
				if value != "deep" {
					t.Errorf("Expected 'deep', got %v", value)
				}
			},
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SetValueByPath([]byte(baseJSON), tt.path, tt.newValue)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SetValueByPath() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("SetValueByPath() unexpected error: %v", err)
				return
			}

			// Validate the result is valid JSON
			var parsed interface{}
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Errorf("SetValueByPath() produced invalid JSON: %v", err)
				return
			}

			// Run custom validation if provided
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestSetValueByPath_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		requestBody []byte
		path        string
		newValue    interface{}
		wantErr     bool
	}{
		{
			name:        "empty request body",
			requestBody: []byte{},
			path:        "test",
			newValue:    "value",
			wantErr:     true,
		},
		{
			name:        "invalid JSON",
			requestBody: []byte(`{"invalid": json}`),
			path:        "test",
			newValue:    "value",
			wantErr:     true,
		},
		{
			name:        "array root",
			requestBody: []byte(`[1, 2, 3]`),
			path:        "test",
			newValue:    "value",
			wantErr:     true,
		},
		{
			name:        "null root",
			requestBody: []byte(`null`),
			path:        "test",
			newValue:    "value",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SetValueByPath(tt.requestBody, tt.path, tt.newValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetValueByPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStreamingGetValueByPath(t *testing.T) {
	testJSON := `{
		"generationConfig": {
			"maxOutputTokens": 1024,
			"temperature": 0.7
		},
		"model": "gpt-4"
	}`

	tests := []struct {
		name     string
		path     string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "simple path",
			path:     "model",
			expected: "gpt-4",
			wantErr:  false,
		},
		{
			name:     "nested path",
			path:     "generationConfig.maxOutputTokens",
			expected: json.Number("1024"),
			wantErr:  false,
		},
		{
			name:    "non-existent key",
			path:    "nonexistent",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(testJSON)
			result, err := StreamingGetValueByPath(reader, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("StreamingGetValueByPath() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("StreamingGetValueByPath() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("StreamingGetValueByPath() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestStreamingSetValueByPath(t *testing.T) {
	testJSON := `{
		"generationConfig": {
			"maxOutputTokens": 1024,
			"temperature": 0.7
		},
		"model": "gpt-4"
	}`

	tests := []struct {
		name     string
		path     string
		newValue interface{}
		wantErr  bool
		validate func(t *testing.T, result string)
	}{
		{
			name:     "update existing value",
			path:     "generationConfig.maxOutputTokens",
			newValue: 2048,
			wantErr:  false,
			validate: func(t *testing.T, result string) {
				value, err := GetValueByPath([]byte(result), "generationConfig.maxOutputTokens")
				if err != nil {
					t.Errorf("Failed to get updated value: %v", err)
					return
				}
				// Value could be json.Number or float64 depending on how it was parsed
				var numVal float64
				switch v := value.(type) {
				case json.Number:
					var err error
					numVal, err = v.Float64()
					if err != nil {
						t.Errorf("Failed to convert json.Number to float64: %v", err)
						return
					}
				case float64:
					numVal = v
				default:
					t.Errorf("Expected numeric value, got %v (%T)", value, value)
					return
				}
				if numVal != 2048 {
					t.Errorf("Expected 2048, got %v", numVal)
				}
			},
		},
		{
			name:     "create new path",
			path:     "newConfig.newValue",
			newValue: "test",
			wantErr:  false,
			validate: func(t *testing.T, result string) {
				value, err := GetValueByPath([]byte(result), "newConfig.newValue")
				if err != nil {
					t.Errorf("Failed to get new value: %v", err)
					return
				}
				if value != "test" {
					t.Errorf("Expected 'test', got %v", value)
				}
			},
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(testJSON)
			var writer bytes.Buffer

			err := StreamingSetValueByPath(reader, &writer, tt.path, tt.newValue)

			if tt.wantErr {
				if err == nil {
					t.Errorf("StreamingSetValueByPath() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("StreamingSetValueByPath() unexpected error: %v", err)
				return
			}

			result := writer.String()

			// Validate the result is valid JSON
			var parsed interface{}
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("StreamingSetValueByPath() produced invalid JSON: %v", err)
				return
			}

			// Run custom validation if provided
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkGetValueByPath(b *testing.B) {
	testJSON := []byte(`{
		"generationConfig": {
			"maxOutputTokens": 1024,
			"temperature": 0.7,
			"nested": {
				"deep": {
					"value": "found"
				}
			}
		},
		"model": "gpt-4",
		"messages": [
			{"role": "user", "content": "hello"},
			{"role": "assistant", "content": "hi"}
		]
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetValueByPath(testJSON, "generationConfig.maxOutputTokens")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetValueByPath(b *testing.B) {
	testJSON := []byte(`{
		"generationConfig": {
			"maxOutputTokens": 1024,
			"temperature": 0.7
		},
		"model": "gpt-4"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SetValueByPath(testJSON, "generationConfig.maxOutputTokens", 2048)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamingGetValueByPath(b *testing.B) {
	testJSON := `{
		"generationConfig": {
			"maxOutputTokens": 1024,
			"temperature": 0.7
		},
		"model": "gpt-4"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(testJSON)
		_, err := StreamingGetValueByPath(reader, "generationConfig.maxOutputTokens")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamingSetValueByPath(b *testing.B) {
	testJSON := `{
		"generationConfig": {
			"maxOutputTokens": 1024,
			"temperature": 0.7
		},
		"model": "gpt-4"
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(testJSON)
		var writer bytes.Buffer
		err := StreamingSetValueByPath(reader, &writer, "generationConfig.maxOutputTokens", 2048)
		if err != nil {
			b.Fatal(err)
		}
	}
}
