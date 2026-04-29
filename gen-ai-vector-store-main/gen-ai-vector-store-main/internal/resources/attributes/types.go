/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package attributes

// AttrValues type Attributes required for proper rows.Scan() in GORM
type AttrValues []string

// Dedicated type Attributes required for proper rows.Scan() in GORM
type Attributes []Attribute

type Attribute struct {
	Name   string     `json:"name" binding:"required"`
	Values AttrValues `json:"value" binding:"required"`
	Type   string     `json:"type,omitempty"`
	Kind   string     `json:"kind,omitempty"`
}

type RetrieveAttributesRequest struct {
	RetrieveAttributes []string `json:"retrieveAttributes,omitempty"`
}

type AttributeFilter struct {
	Name     string     `json:"name,omitempty"`
	Type     string     `json:"type,omitempty"`
	Operator string     `json:"operator,omitempty"`
	Values   AttrValues `json:"value,omitempty"`
}

// AttributeItem represents an attribute item with one value only
type AttributeItem struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
	Type  string `json:"type,omitempty"`
}

type Filter struct {
	Operator string            `json:"operator,omitempty"`
	Items    []AttributeFilter `json:"items,omitempty"`
}

// The AttributesV2 is only for storing attributes in DB. Not used in API requests/responses.
type AttributesV2 map[string]AttributeObject

// The AttributeObject is only for storing attributes in DB. Not used in API requests/responses.
type AttributeObject struct {
	Kind   string   `json:"kind,omitempty"`
	Values []string `json:"values" binding:"required"`
}
