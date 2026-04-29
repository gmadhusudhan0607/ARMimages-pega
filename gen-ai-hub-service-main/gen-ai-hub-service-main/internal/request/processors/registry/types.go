/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package registry

import (
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

// ProcessorKey represents a unique identifier for a processor based on model characteristics
type ProcessorKey struct {
	Provider       types.Provider       `json:"provider"`
	Infrastructure types.Infrastructure `json:"infrastructure"`
	Creator        types.Creator        `json:"creator"`
	ModelID        string               `json:"modelId"` // Model identifier
	Version        string               `json:"version"` // For API versioning
}

// String returns a string representation of the ProcessorKey
func (pk ProcessorKey) String() string {
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		pk.Provider, pk.Infrastructure, pk.Creator, pk.ModelID, pk.Version)
}

// IsValid checks if the ProcessorKey has all required fields
func (pk ProcessorKey) IsValid() bool {
	return pk.Provider != ""
}

// ProcessorFactory is a function that creates a new ProviderExtension instance
type ProcessorFactory func() interface{}

// ProcessorRegistry defines the interface for processor registration and creation
type ProcessorRegistry interface {
	// Register registers a processor factory for a given key
	Register(key ProcessorKey, factory ProcessorFactory) error

	// CreateProcessor creates a processor instance for the given key
	CreateProcessor(key ProcessorKey) (interface{}, error)

	// HasProcessor checks if a processor is registered for the given key
	HasProcessor(key ProcessorKey) bool

	// GetRegisteredKeys returns all registered processor keys
	GetRegisteredKeys() []ProcessorKey

	// GetSupportedCombinations returns all supported provider/model combinations
	GetSupportedCombinations() map[ProcessorKey]string
}

// CreateProcessorKey creates a ProcessorKey from a model
func CreateProcessorKey(model *types.Model) ProcessorKey {
	return ProcessorKey{
		Provider:       model.Provider,
		Infrastructure: model.Infrastructure,
		Creator:        model.Creator,
		ModelID:        model.Name, // Use model name instead of full ID to avoid duplication
		Version:        model.Version,
	}
}
