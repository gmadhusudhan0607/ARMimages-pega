/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

// ParameterSpec represents a parameter specification
type ParameterSpec struct {
	Title       string      `json:"title" yaml:"title"`
	Description string      `json:"description" yaml:"description"`
	Type        string      `json:"type" yaml:"type"`
	Default     interface{} `json:"default" yaml:"default"`
	Maximum     interface{} `json:"maximum" yaml:"maximum"`
	Minimum     interface{} `json:"minimum" yaml:"minimum"`
	Required    bool        `json:"required" yaml:"required"`
}

// ModelParameters represents a collection of model parameters
type ModelParameters map[string]ParameterSpec
