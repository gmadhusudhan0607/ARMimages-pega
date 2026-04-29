/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

// ModelCapabilities represents the capabilities of a model
type ModelCapabilities struct {
	Features         []string `json:"features,omitempty" yaml:"features,omitempty"`
	InputModalities  []string `json:"inputModalities,omitempty" yaml:"inputModalities,omitempty"`
	OutputModalities []string `json:"outputModalities,omitempty" yaml:"outputModalities,omitempty"`
	MimeTypes        []string `json:"mimeTypes,omitempty" yaml:"mimeTypes,omitempty"`
}

// HasCapability checks if the model has a specific capability
func (m *Model) HasCapability(capability FunctionalCapability) bool {
	for _, capa := range m.FunctionalCapabilities {
		if capa == capability {
			return true
		}
	}
	return false
}
