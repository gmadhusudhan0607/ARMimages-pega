/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */
package attributes

import (
	"reflect"
	"testing"
)

func TestConvertAttributesV1ToV2(t *testing.T) {
	tests := []struct {
		name     string
		input    []Attribute
		expected AttributesV2
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    []Attribute{},
			expected: AttributesV2{},
		},
		{
			name: "single attribute with kind",
			input: []Attribute{
				{
					Name:   "category",
					Values: AttrValues{"electronics", "gadgets"},
					Type:   "string", // This should be ignored in V2
					Kind:   "dynamic",
				},
			},
			expected: AttributesV2{
				"category": AttributeObject{
					Kind:   "dynamic",
					Values: []string{"electronics", "gadgets"},
				},
			},
		},
		{
			name: "single attribute without kind",
			input: []Attribute{
				{
					Name:   "brand",
					Values: AttrValues{"apple", "samsung"},
					Type:   "string",
				},
			},
			expected: AttributesV2{
				"brand": AttributeObject{
					Kind:   "", // Kind should be empty when not specified
					Values: []string{"apple", "samsung"},
				},
			},
		},
		{
			name: "multiple attributes with different kinds",
			input: []Attribute{
				{
					Name:   "category",
					Values: AttrValues{"electronics"},
					Type:   "string",
					Kind:   "static",
				},
				{
					Name:   "price_range",
					Values: AttrValues{"100-500", "500-1000"},
					Type:   "numeric", // This should be ignored
					Kind:   "dynamic",
				},
				{
					Name:   "color",
					Values: AttrValues{"red", "blue", "green"},
					Type:   "string",
					Kind:   "",
				},
			},
			expected: AttributesV2{
				"category": AttributeObject{
					Kind:   "static",
					Values: []string{"electronics"},
				},
				"price_range": AttributeObject{
					Kind:   "dynamic",
					Values: []string{"100-500", "500-1000"},
				},
				"color": AttributeObject{
					Kind:   "",
					Values: []string{"red", "blue", "green"},
				},
			},
		},
		{
			name: "attribute with single value",
			input: []Attribute{
				{
					Name:   "status",
					Values: AttrValues{"active"},
					Type:   "string",
					Kind:   "static",
				},
			},
			expected: AttributesV2{
				"status": AttributeObject{
					Kind:   "static",
					Values: []string{"active"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertAttributesV1ToV2(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ConvertAttributesV1ToV2() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestConvertAttributesV2ToV1(t *testing.T) {
	tests := []struct {
		name     string
		input    AttributesV2
		expected []Attribute
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    AttributesV2{},
			expected: []Attribute{},
		},
		{
			name: "single attribute with kind",
			input: AttributesV2{
				"category": AttributeObject{
					Kind:   "dynamic",
					Values: []string{"electronics", "gadgets"},
				},
			},
			expected: []Attribute{
				{
					Name:   "category",
					Values: AttrValues{"electronics", "gadgets"},
					Type:   "string", // Should default to "string"
					Kind:   "dynamic",
				},
			},
		},
		{
			name: "attribute with single value",
			input: AttributesV2{
				"status": AttributeObject{
					Kind:   "static",
					Values: []string{"active"},
				},
			},
			expected: []Attribute{
				{
					Name:   "status",
					Values: AttrValues{"active"},
					Type:   "string",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertAttributesV2ToV1(tt.input)

			// For non-nil cases, we need to compare each attribute individually
			// since the order might vary due to map iteration
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ConvertAttributesV2ToV1() = %v, expected nil", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ConvertAttributesV2ToV1() returned %d attributes, expected %d", len(result), len(tt.expected))
				return
			}

			// Create maps for comparison since order might differ
			resultMap := make(map[string]Attribute)
			for _, attr := range result {
				resultMap[attr.Name] = attr
			}

			expectedMap := make(map[string]Attribute)
			for _, attr := range tt.expected {
				expectedMap[attr.Name] = attr
			}

			if !reflect.DeepEqual(resultMap, expectedMap) {
				t.Errorf("ConvertAttributesV2ToV1() = %v, expected %v", resultMap, expectedMap)
			}
		})
	}
}

func TestConvertAttributesV2ToV1_MultipleAttributes(t *testing.T) {
	// Test with multiple attributes to ensure all are converted properly
	input := AttributesV2{
		"category": AttributeObject{
			Kind:   "static",
			Values: []string{"electronics"},
		},
		"price_range": AttributeObject{
			Kind:   "dynamic",
			Values: []string{"100-500", "500-1000"},
		},
		"color": AttributeObject{
			Kind:   "", // Should default to "static"
			Values: []string{"red", "blue", "green"},
		},
	}

	result := ConvertAttributesV2ToV1(input)

	// Check that we have the right number of attributes
	if len(result) != 3 {
		t.Errorf("Expected 3 attributes, got %d", len(result))
		return
	}

	// Create a map for easier verification
	resultMap := make(map[string]Attribute)
	for _, attr := range result {
		resultMap[attr.Name] = attr
	}

	// Verify each attribute
	expectedAttrs := map[string]Attribute{
		"category": {
			Name:   "category",
			Values: AttrValues{"electronics"},
			Type:   "string",
			Kind:   "", // Empty Kind since V2 had "static"
		},
		"price_range": {
			Name:   "price_range",
			Values: AttrValues{"100-500", "500-1000"},
			Type:   "string",
			Kind:   "dynamic",
		},
		"color": {
			Name:   "color",
			Values: AttrValues{"red", "blue", "green"},
			Type:   "string",
			Kind:   "", // Empty Kind since V2 had empty Kind
		},
	}

	for name, expected := range expectedAttrs {
		if actual, exists := resultMap[name]; !exists {
			t.Errorf("Missing attribute %s in result", name)
		} else if !reflect.DeepEqual(actual, expected) {
			t.Errorf("Attribute %s: got %v, expected %v", name, actual, expected)
		}
	}
}

func TestConvertAttributesV1ToV2toV1_Roundtrip(t *testing.T) {
	// Test round-trip conversion (V1 -> V2 -> V1)
	// Note: Type information is lost in the round-trip, and Kind defaults to "static" if empty
	original := []Attribute{
		{
			Name:   "category",
			Values: AttrValues{"electronics", "gadgets"},
			Kind:   "dynamic",
		},
		{
			Name:   "brand",
			Values: AttrValues{"apple"},
			Kind:   "static",
		},
	}

	// Convert V1 -> V2 -> V1
	v2 := ConvertAttributesV1ToV2(original)
	roundtrip := ConvertAttributesV2ToV1(v2)

	// Expected result after roundtrip (Type defaults to "string")
	expected := []Attribute{
		{
			Name:   "category",
			Values: AttrValues{"electronics", "gadgets"},
			Type:   "string", // Defaulted
			Kind:   "dynamic",
		},
		{
			Name:   "brand",
			Values: AttrValues{"apple"},
			Type:   "string", // Defaulted
			Kind:   "",       // Empty Kind since original had "static" which converts to empty in V1
		},
	}

	// Create maps for comparison due to potential order differences
	roundtripMap := make(map[string]Attribute)
	for _, attr := range roundtrip {
		roundtripMap[attr.Name] = attr
	}

	expectedMap := make(map[string]Attribute)
	for _, attr := range expected {
		expectedMap[attr.Name] = attr
	}

	if !reflect.DeepEqual(roundtripMap, expectedMap) {
		t.Errorf("Round-trip conversion failed. Got %v, expected %v", roundtripMap, expectedMap)
	}
}
