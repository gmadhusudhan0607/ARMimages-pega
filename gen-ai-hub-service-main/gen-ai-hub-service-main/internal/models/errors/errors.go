/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package errors

import (
	"fmt"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

// VersionNotFoundError is returned when a specific version of a model is not found
type VersionNotFoundError struct {
	Provider          types.Provider
	Creator           types.Creator
	ModelName         string
	RequestedVersion  string
	AvailableVersions []string
}

// Error implements the error interface
func (e *VersionNotFoundError) Error() string {
	if len(e.AvailableVersions) == 0 {
		return fmt.Sprintf("model %s/%s/%s not found", e.Provider, e.Creator, e.ModelName)
	}
	return fmt.Sprintf("version %s not found for model %s/%s/%s, available versions: %v",
		e.RequestedVersion, e.Provider, e.Creator, e.ModelName, e.AvailableVersions)
}

// ModelNotFoundError is returned when a model is not found
type ModelNotFoundError struct {
	Provider  types.Provider
	Creator   types.Creator
	ModelName string
}

// Error implements the error interface
func (e *ModelNotFoundError) Error() string {
	return fmt.Sprintf("model %s/%s/%s not found", e.Provider, e.Creator, e.ModelName)
}

// IsVersionNotFoundError checks if an error is a VersionNotFoundError
func IsVersionNotFoundError(err error) bool {
	_, ok := err.(*VersionNotFoundError)
	return ok
}

// IsModelNotFoundError checks if an error is a ModelNotFoundError
func IsModelNotFoundError(err error) bool {
	_, ok := err.(*ModelNotFoundError)
	return ok
}
