/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

// Model represents a model configuration that matches the structure in model-metadata.yaml
type Model struct {
	KEY                    string                   `json:"KEY" yaml:"KEY" validate:"required"` // Required field for model identification
	FunctionalCapabilities []FunctionalCapability   `json:"functionalCapabilities" yaml:"functionalCapabilities" validate:"required,min=1"`
	Name                   string                   `json:"modelName" yaml:"modelName" validate:"required"` // Required field
	Version                string                   `json:"version" yaml:"version" validate:"required"`     // Required field
	Label                  string                   `json:"modelLabel" yaml:"modelLabel"`
	Capabilities           ModelCapabilities        `json:"modelCapabilities" yaml:"modelCapabilities"`
	Parameters             map[string]ParameterSpec `json:"parameters,omitempty" yaml:"parameters,omitempty" validate:"required,dive"` // Required field with required parameters
	DeprecationDate        string                   `json:"deprecationDate,omitempty" yaml:"deprecationDate,omitempty"`
	Lifecycle              string                   `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`
	Creator                Creator                  `json:"creator,omitempty" yaml:"creator,omitempty"`                               // Model creator (for internal use)
	Provider               Provider                 `json:"provider,omitempty" yaml:"provider,omitempty"`                             // AI provider (for internal use)
	Infrastructure         Infrastructure           `json:"infrastructure,omitempty" yaml:"infrastructure,omitempty"`                 // Deployment infrastructure (for internal use)
	Endpoints              []Endpoint               `json:"endpoints,omitempty" yaml:"endpoints,omitempty" validate:"required,min=1"` // Required field with at least 1 element
	SourceFile             string                   `json:"source_file,omitempty"`                                                    // Path to the YAML file that defined this model
}

// GetMaxOutputTokens extracts maximum output tokens from model parameters
func (m *Model) GetMaxOutputTokens() *float64 {
	if m == nil {
		return nil
	}

	maxOutputTokensParam, exists := m.Parameters["maxOutputTokens"]
	if !exists {
		return nil
	}

	if maximum := maxOutputTokensParam.Maximum; maximum != nil {
		if maxFloat, ok := maximum.(float64); ok {
			return &maxFloat
		} else if maxInt, ok := maximum.(int); ok {
			maxFloat := float64(maxInt)
			return &maxFloat
		}
	}

	return nil
}
