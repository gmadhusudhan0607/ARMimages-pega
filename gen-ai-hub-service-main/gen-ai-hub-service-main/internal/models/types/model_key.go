/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

import "fmt"

// ModelKey represents a 5-parameter unique model identifier
type ModelKey struct {
	Infrastructure Infrastructure `json:"infrastructure"`
	Provider       Provider       `json:"provider"`
	Creator        Creator        `json:"creator"`
	ModelName      string         `json:"modelName"`
	Version        string         `json:"version"`
}

// String returns a string representation of the ModelKey
func (mk ModelKey) String() string {
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		mk.Infrastructure, mk.Provider, mk.Creator, mk.ModelName, mk.Version)
}

// GetModelKey returns the 5-parameter ModelKey for this model
func (m *Model) GetModelKey() ModelKey {
	return ModelKey{
		Infrastructure: m.Infrastructure,
		Provider:       m.Provider,
		Creator:        m.Creator,
		ModelName:      m.Name,
		Version:        m.Version,
	}
}
