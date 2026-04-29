/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package attributes

import (
	"sort"
	"strings"
)

// ConvertAttributesV1ToV2 converts a slice of V1 Attributes to a slice of V2 AttributesV2
// Note: The function signature in the task suggested []Attribute => AttributesV2 (single),
// but logically it makes more sense to convert each V1 attribute to a V2 attribute
func ConvertAttributesV1ToV2(source []Attribute) AttributesV2 {
	if source == nil {
		return nil
	}
	target := make(AttributesV2)
	for _, attr := range source {
		target[attr.Name] = AttributeObject{
			Kind:   attr.Kind,
			Values: []string(attr.Values),
		}
		// Note: attr.Type is ignored as specified in the requirements
	}
	return target
}

// ConvertAttributesV2ToV1 converts a slice of V2 AttributesV2 to a slice of V1 Attributes
func ConvertAttributesV2ToV1(source AttributesV2) []Attribute {
	if source == nil {
		return nil
	}

	target := make([]Attribute, 0, len(source))
	for name, obj := range source {
		kind := ""
		// Only preserve non-default kind values (anything except empty or "static")
		if obj.Kind != "" && obj.Kind != "static" {
			kind = obj.Kind
		}

		target = append(target, Attribute{
			Name:   name,
			Values: AttrValues(obj.Values),
			Kind:   kind,
			Type:   "string", // Defaulting to "string" for backward compatibility
		})
	}

	// Sort attributes by name to ensure consistent ordering (maps don't preserve order)
	// Use case-insensitive comparison to maintain consistent order
	sort.Slice(target, func(i, j int) bool {
		return strings.ToLower(target[i].Name) < strings.ToLower(target[j].Name)
	})

	return target
}

// ConvertAttributesV2ToV1WithFilter converts V2 AttributesV2 to V1 Attributes, filtering by attribute names
// This optimized version filters attributes before sorting to reduce unnecessary work
func ConvertAttributesV2ToV1WithFilter(source AttributesV2, includedNamesFilter []string) []Attribute {
	if source == nil {
		return nil
	}

	// If no filter provided, use the standard conversion
	if len(includedNamesFilter) == 0 {
		return ConvertAttributesV2ToV1(source)
	}

	// Build a set of filter names for O(1) lookup
	filterSet := make(map[string]bool, len(includedNamesFilter))
	for _, name := range includedNamesFilter {
		filterSet[name] = true
	}

	// Pre-allocate with filter size hint
	target := make([]Attribute, 0, len(includedNamesFilter))
	for name, obj := range source {
		// Skip attributes not in the filter
		if !filterSet[name] {
			continue
		}

		kind := ""
		// Only preserve non-default kind values (anything except empty or "static")
		if obj.Kind != "" && obj.Kind != "static" {
			kind = obj.Kind
		}

		target = append(target, Attribute{
			Name:   name,
			Values: AttrValues(obj.Values),
			Kind:   kind,
			Type:   "string", // Defaulting to "string" for backward compatibility
		})
	}

	// Sort filtered attributes by name
	sort.Slice(target, func(i, j int) bool {
		return strings.ToLower(target[i].Name) < strings.ToLower(target[j].Name)
	})

	return target
}

// MergeAttributes merges two slices of attributes
// When merging attributes with the same name and type:
// - Values are merged, keeping only unique values
// - Values are sorted within each attribute
// - Final list is sorted by attribute name
func MergeAttributes(docAttrs, chunkAttrs []Attribute) []Attribute {
	if len(docAttrs) == 0 && len(chunkAttrs) == 0 {
		return nil
	}

	// Create a map to track unique attributes by name+type
	attrMap := make(map[string]*Attribute)

	// Helper function to add attribute to map
	addToMap := func(attr Attribute) {
		key := attr.Name + "|" + attr.Type
		if existing, exists := attrMap[key]; exists {
			// Merge values, keeping unique ones
			valueSet := make(map[string]bool)
			for _, v := range existing.Values {
				valueSet[v] = true
			}
			for _, v := range attr.Values {
				valueSet[v] = true
			}

			// Convert back to slice and sort
			mergedValues := make([]string, 0, len(valueSet))
			for v := range valueSet {
				mergedValues = append(mergedValues, v)
			}

			// Sort values using sort.Strings
			sort.Strings(mergedValues)

			existing.Values = AttrValues(mergedValues)
		} else {
			// Create a copy to avoid modifying original
			newAttr := Attribute{
				Name:   attr.Name,
				Type:   attr.Type,
				Kind:   attr.Kind,
				Values: make(AttrValues, len(attr.Values)),
			}
			copy(newAttr.Values, attr.Values)

			// Sort values in the new attribute
			values := []string(newAttr.Values)
			sort.Strings(values)
			newAttr.Values = AttrValues(values)

			attrMap[key] = &newAttr
		}
	}

	// Add document attributes
	for _, attr := range docAttrs {
		addToMap(attr)
	}

	// Add chunk attributes
	for _, attr := range chunkAttrs {
		addToMap(attr)
	}

	// Convert map back to slice
	result := make([]Attribute, 0, len(attrMap))
	for _, attr := range attrMap {
		result = append(result, *attr)
	}

	// Sort by attribute name, then by type using sort.Slice
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].Type < result[j].Type
	})

	return result
}
