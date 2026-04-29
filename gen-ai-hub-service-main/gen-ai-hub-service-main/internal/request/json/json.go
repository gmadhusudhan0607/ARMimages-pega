/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// GetValueByPath extracts a value from JSON data using a dot-separated path
// Example: GetValueByPath(data, "generationConfig.maxOutputTokens")
func GetValueByPath(requestBody []byte, path string) (interface{}, error) {
	if len(requestBody) == 0 {
		return nil, fmt.Errorf("empty request body")
	}

	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	// Parse JSON using streaming decoder for performance
	decoder := json.NewDecoder(bytes.NewReader(requestBody))
	decoder.UseNumber() // Use json.Number to preserve numeric precision

	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return getValueFromPath(data, strings.Split(path, "."))
}

// SetValueByPath sets or overrides a value in JSON data using a dot-separated path
// Returns the modified JSON bytes
func SetValueByPath(requestBody []byte, path string, newValue interface{}) ([]byte, error) {
	if len(requestBody) == 0 {
		return nil, fmt.Errorf("empty request body")
	}

	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	// Parse JSON using streaming decoder for performance
	decoder := json.NewDecoder(bytes.NewReader(requestBody))
	decoder.UseNumber() // Use json.Number to preserve numeric precision

	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Ensure we have a map to work with
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("root JSON must be an object")
	}

	// Set the value at the specified path
	if err := setValueAtPath(dataMap, strings.Split(path, "."), newValue); err != nil {
		return nil, err
	}

	// Encode back to JSON using streaming encoder
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // Don't escape HTML characters for better performance

	if err := encoder.Encode(dataMap); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	// Remove the trailing newline added by encoder
	result := buf.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}

	return result, nil
}

// getValueFromPath recursively traverses the data structure following the path
func getValueFromPath(data interface{}, pathParts []string) (interface{}, error) {
	if len(pathParts) == 0 {
		return data, nil
	}

	currentKey := pathParts[0]
	remainingPath := pathParts[1:]

	switch v := data.(type) {
	case map[string]interface{}:
		value, exists := v[currentKey]
		if !exists {
			return nil, fmt.Errorf("key '%s' not found", currentKey)
		}
		return getValueFromPath(value, remainingPath)

	case []interface{}:
		// Handle array access with numeric index
		index, err := strconv.Atoi(currentKey)
		if err != nil {
			return nil, fmt.Errorf("invalid array index '%s': %w", currentKey, err)
		}
		if index < 0 || index >= len(v) {
			return nil, fmt.Errorf("array index %d out of bounds (length: %d)", index, len(v))
		}
		return getValueFromPath(v[index], remainingPath)

	default:
		return nil, fmt.Errorf("cannot traverse path '%s' on non-object/non-array value", currentKey)
	}
}

// setValueAtPath recursively sets a value at the specified path, creating intermediate objects as needed
func setValueAtPath(data map[string]interface{}, pathParts []string, value interface{}) error {
	if len(pathParts) == 0 {
		return fmt.Errorf("empty path parts")
	}

	currentKey := pathParts[0]

	// If this is the last part of the path, set the value
	if len(pathParts) == 1 {
		data[currentKey] = value
		return nil
	}

	// Navigate or create intermediate path
	remainingPath := pathParts[1:]

	// Check if the key exists
	if existing, exists := data[currentKey]; exists {
		// If it exists and is a map, continue traversing
		if existingMap, ok := existing.(map[string]interface{}); ok {
			return setValueAtPath(existingMap, remainingPath, value)
		}
		// If it exists but is not a map, we need to replace it with a new map
	}

	// Create a new map for this key
	newMap := make(map[string]interface{})
	data[currentKey] = newMap

	return setValueAtPath(newMap, remainingPath, value)
}

// StreamingGetValueByPath extracts a value from JSON stream using a dot-separated path
// This is optimized for large JSON payloads by using streaming parsing
func StreamingGetValueByPath(reader io.Reader, path string) (interface{}, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	decoder := json.NewDecoder(reader)
	decoder.UseNumber() // Use json.Number to preserve numeric precision

	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return getValueFromPath(data, strings.Split(path, "."))
}

// StreamingSetValueByPath sets a value in JSON stream and writes the result to the writer
// This is optimized for large JSON payloads by using streaming parsing and encoding
func StreamingSetValueByPath(reader io.Reader, writer io.Writer, path string, newValue interface{}) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	decoder := json.NewDecoder(reader)
	decoder.UseNumber() // Use json.Number to preserve numeric precision

	var data interface{}
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Ensure we have a map to work with
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("root JSON must be an object")
	}

	// Set the value at the specified path
	if err := setValueAtPath(dataMap, strings.Split(path, "."), newValue); err != nil {
		return err
	}

	// Encode directly to the writer
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false) // Don't escape HTML characters for better performance

	return encoder.Encode(dataMap)
}
