/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

// ModelValidator provides comprehensive validation for model configurations
type ModelValidator struct{}

// NewModelValidator creates a new model validator
func NewModelValidator() *ModelValidator {
	return &ModelValidator{}
}

// ValidateModelGroup validates a model group and enriches it with computed values
func (v *ModelValidator) ValidateModelGroup(group *config.ModelGroup, filePath string) error {
	// Extract path components for validation
	pathInfo, err := v.extractPathInfo(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path structure: %w", err)
	}

	// Validate infrastructure/provider/creator match path
	if err := v.validatePathConsistency(group, pathInfo); err != nil {
		return fmt.Errorf("path consistency validation failed: %w", err)
	}

	// Validate group-level requirements
	if err := v.validateGroupRequirements(group); err != nil {
		return fmt.Errorf("group validation failed: %w", err)
	}

	// Validate and enrich each model
	for i, model := range group.Models {
		if err := v.validateAndEnrichModel(&group.Models[i], pathInfo); err != nil {
			return fmt.Errorf("model validation failed for model %d (%s): %w", i, model.Name, err)
		}
	}

	return nil
}

// PathInfo contains extracted path information
type PathInfo struct {
	Infrastructure types.Infrastructure
	Provider       types.Provider
	Creator        types.Creator
	SpecFile       string
}

// extractPathInfo extracts infrastructure/provider/creator from file path
func (v *ModelValidator) extractPathInfo(filePath string) (*PathInfo, error) {
	// Expected format: infrastructure/provider/creator/model-name.yaml
	// Example: aws/bedrock/amazon/nova.yaml

	// Remove .yaml extension
	path := strings.TrimSuffix(filePath, ".yaml")

	// Split path components
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("path must have at least 3 components (infrastructure/provider/creator), got %d: %s", len(parts), filePath)
	}

	// Extract components
	infrastructure := types.Infrastructure(parts[0])
	provider := types.Provider(parts[1])
	creator := types.Creator(parts[2])

	// Model name is the last component (filename without extension)
	modelName := parts[len(parts)-1]

	return &PathInfo{
		Infrastructure: infrastructure,
		Provider:       provider,
		Creator:        creator,
		SpecFile:       modelName,
	}, nil
}

// validatePathConsistency validates that YAML properties match the file path
func (v *ModelValidator) validatePathConsistency(group *config.ModelGroup, pathInfo *PathInfo) error {
	// Validate infrastructure matches
	if group.Infrastructure != pathInfo.Infrastructure {
		return fmt.Errorf("infrastructure mismatch: path has '%s' but YAML has '%s'",
			pathInfo.Infrastructure, group.Infrastructure)
	}

	// Validate provider matches
	if group.Provider != pathInfo.Provider {
		return fmt.Errorf("provider mismatch: path has '%s' but YAML has '%s'",
			pathInfo.Provider, group.Provider)
	}

	// Validate creator matches
	if group.Creator != pathInfo.Creator {
		return fmt.Errorf("creator mismatch: path has '%s' but YAML has '%s'",
			pathInfo.Creator, group.Creator)
	}

	return nil
}

// validateGroupRequirements validates group-level requirements
func (v *ModelValidator) validateGroupRequirements(group *config.ModelGroup) error {
	if group.Infrastructure == "" {
		return fmt.Errorf("infrastructure is required")
	}
	if group.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if group.Creator == "" {
		return fmt.Errorf("creator is required")
	}
	if len(group.Models) == 0 {
		return fmt.Errorf("at least one model is required")
	}

	return nil
}

// validateAndEnrichModel validates a single model and enriches it with computed values
func (v *ModelValidator) validateAndEnrichModel(model *config.EnhancedModelConfig, pathInfo *PathInfo) error {
	// Check for forbidden properties using reflection
	if err := v.validateForbiddenProperties(model); err != nil {
		return err
	}

	// Validate required properties
	if err := v.validateRequiredProperties(model); err != nil {
		return err
	}

	// Validate parameters structure
	if err := v.validateParameters(model); err != nil {
		return err
	}

	// Automatically calculate and set Model.KEY
	model.KEY = v.calculateModelKEY(pathInfo, model)

	// Validate Model.KEY is required (should be auto-calculated)
	if model.KEY == "" {
		return fmt.Errorf("model.modelKEY is required (should be auto-calculated)")
	}

	// Validate Model.KEY format
	if err := v.validateModelKEYFormat(model.KEY, pathInfo, model); err != nil {
		return err
	}

	return nil
}

// validateForbiddenProperties checks for forbidden properties (case insensitive)
func (v *ModelValidator) validateForbiddenProperties(model *config.EnhancedModelConfig) error {
	// Check for forbidden 'key' property at top level (case insensitive)
	// Note: The 'key' field is auto-calculated and should not be present in YAML
	if err := v.checkForbiddenField(model, []string{"key"}); err != nil {
		return err
	}

	// Check for forbidden 'modelKey' property at top level (case insensitive)
	if err := v.checkForbiddenField(model, []string{"modelKey"}); err != nil {
		return err
	}

	return nil
}

// checkForbiddenField checks for forbidden field names at a specific path (case insensitive)
func (v *ModelValidator) checkForbiddenField(model *config.EnhancedModelConfig, forbiddenPath []string) error {
	if len(forbiddenPath) == 0 {
		return fmt.Errorf("forbidden path cannot be empty")
	}

	return v.checkForbiddenFieldAtPath(reflect.ValueOf(model).Elem(), forbiddenPath, false, "")
}

// checkForbiddenFieldExact checks for exact forbidden field names at a specific path
func (v *ModelValidator) checkForbiddenFieldExact(model *config.EnhancedModelConfig, forbiddenPath []string) error {
	if len(forbiddenPath) == 0 {
		return fmt.Errorf("forbidden path cannot be empty")
	}

	return v.checkForbiddenFieldAtPath(reflect.ValueOf(model).Elem(), forbiddenPath, true, "")
}

// checkForbiddenFieldAtPath recursively checks for forbidden fields at the specified path
func (v *ModelValidator) checkForbiddenFieldAtPath(value reflect.Value, path []string, exactMatch bool, currentPath string) error {
	if len(path) == 0 {
		return nil
	}

	// Handle pointer at the top level
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		return v.checkForbiddenFieldAtPath(value.Elem(), path, exactMatch, currentPath)
	}

	// Handle interface at the top level
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		return v.checkForbiddenFieldAtPath(value.Elem(), path, exactMatch, currentPath)
	}

	// If this is the last element in the path, check if it exists at this level
	if len(path) == 1 {
		return v.checkFieldExistsAtLevel(value, path[0], exactMatch, currentPath)
	}

	// Navigate to the next level in the path
	return v.navigateToNextLevel(value, path, exactMatch, currentPath)
}

// navigateToNextLevel navigates to the next level in the path
func (v *ModelValidator) navigateToNextLevel(value reflect.Value, path []string, exactMatch bool, currentPath string) error {
	nextField := path[0]
	remainingPath := path[1:]

	// Get the field at this level
	fieldValue, fieldName, found := v.getFieldByName(value, nextField, exactMatch)
	if !found {
		return nil // Field doesn't exist at this level
	}

	newCurrentPath := v.buildCurrentPath(currentPath, fieldName)

	return v.handleFieldType(fieldValue, remainingPath, exactMatch, newCurrentPath, nextField)
}

// buildCurrentPath builds the current path string
func (v *ModelValidator) buildCurrentPath(currentPath, fieldName string) string {
	if currentPath != "" {
		return currentPath + "." + fieldName
	}
	return fieldName
}

// handleFieldType handles different field types
func (v *ModelValidator) handleFieldType(fieldValue reflect.Value, remainingPath []string, exactMatch bool, currentPath, nextField string) error {
	switch fieldValue.Kind() {
	case reflect.Struct:
		return v.checkForbiddenFieldAtPath(fieldValue, remainingPath, exactMatch, currentPath)
	case reflect.Map:
		return v.handleMapField(fieldValue, remainingPath, exactMatch, currentPath)
	case reflect.Ptr:
		return v.handlePointerField(fieldValue, remainingPath, exactMatch, currentPath)
	case reflect.Interface:
		return v.handleInterfaceField(fieldValue, remainingPath, exactMatch, currentPath)
	default:
		// Can't navigate further into non-struct/non-map types
		// Log unexpected type for debugging
		return nil
	}
}

// handleMapField handles map field types
func (v *ModelValidator) handleMapField(fieldValue reflect.Value, remainingPath []string, exactMatch bool, currentPath string) error {
	if len(remainingPath) == 0 {
		return nil
	}

	nextKey := remainingPath[0]
	mapValue := fieldValue.MapIndex(reflect.ValueOf(nextKey))
	if !mapValue.IsValid() {
		return nil // Key doesn't exist in map
	}

	// For maps, if this is the final element in the path, report it as found
	if len(remainingPath) == 1 {
		return v.createForbiddenPropertyError(nextKey, currentPath+"."+nextKey, exactMatch)
	}

	// Continue with the value from the map
	return v.checkForbiddenFieldAtPath(mapValue, remainingPath[1:], exactMatch, currentPath+"."+nextKey)
}

// handlePointerField handles pointer field types
func (v *ModelValidator) handlePointerField(fieldValue reflect.Value, remainingPath []string, exactMatch bool, currentPath string) error {
	if fieldValue.IsNil() {
		return nil
	}
	return v.checkForbiddenFieldAtPath(fieldValue.Elem(), remainingPath, exactMatch, currentPath)
}

// handleInterfaceField handles interface field types
func (v *ModelValidator) handleInterfaceField(fieldValue reflect.Value, remainingPath []string, exactMatch bool, currentPath string) error {
	if fieldValue.IsNil() {
		return nil
	}
	return v.checkForbiddenFieldAtPath(fieldValue.Elem(), remainingPath, exactMatch, currentPath)
}

// createForbiddenPropertyError creates a forbidden property error
func (v *ModelValidator) createForbiddenPropertyError(fieldName, fullPath string, exactMatch bool) error {
	if exactMatch {
		return fmt.Errorf("forbidden property '%s' found in YAML at path '%s' (exact match)", fieldName, fullPath)
	}
	return fmt.Errorf("forbidden property '%s' found in YAML at path '%s' (case insensitive check)", fieldName, fullPath)
}

// checkFieldExistsAtLevel checks if a field exists at the current level and has a non-zero value
func (v *ModelValidator) checkFieldExistsAtLevel(value reflect.Value, fieldName string, exactMatch bool, currentPath string) error {
	fieldValue, actualFieldName, found := v.getFieldByName(value, fieldName, exactMatch)
	if found {
		// Check if the field has a non-zero value (i.e., it was explicitly set)
		if !fieldValue.IsZero() {
			fullPath := currentPath
			if fullPath != "" {
				fullPath += "."
			}
			fullPath += actualFieldName

			if exactMatch {
				return fmt.Errorf("forbidden property '%s' found in YAML at path '%s' (exact match)", fieldName, fullPath)
			} else {
				return fmt.Errorf("forbidden property '%s' found in YAML at path '%s' (case insensitive check)", fieldName, fullPath)
			}
		}
	}
	return nil
}

// getFieldByName finds a field by name in a struct or map, supporting both exact and case-insensitive matching
func (v *ModelValidator) getFieldByName(value reflect.Value, fieldName string, exactMatch bool) (reflect.Value, string, bool) {
	switch value.Kind() {
	case reflect.Struct:
		valueType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			field := valueType.Field(i)
			yamlTag := field.Tag.Get("yaml")

			// Extract the yaml field name (before any options like omitempty)
			yamlName := strings.Split(yamlTag, ",")[0]
			if yamlName == "" || yamlName == "-" {
				continue
			}

			var matches bool
			if exactMatch {
				matches = yamlName == fieldName
			} else {
				matches = strings.EqualFold(yamlName, fieldName)
			}

			if matches {
				return value.Field(i), yamlName, true
			}
		}
	case reflect.Map:
		// For maps, check if the key exists
		for _, key := range value.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			var matches bool
			if exactMatch {
				matches = keyStr == fieldName
			} else {
				matches = strings.EqualFold(keyStr, fieldName)
			}

			if matches {
				return value.MapIndex(key), keyStr, true
			}
		}
	}

	return reflect.Value{}, "", false
}

// validateRequiredProperties validates all required properties
func (v *ModelValidator) validateRequiredProperties(model *config.EnhancedModelConfig) error {
	if model.Name == "" {
		return fmt.Errorf("model.name is required")
	}
	if model.Version == "" {
		return fmt.Errorf("model.version is required")
	}

	// Validate functional capabilities are required
	if len(model.FunctionalCapabilities) == 0 {
		return fmt.Errorf("model.functionalCapabilities is required")
	}

	// Validate functional capabilities values
	for _, capStr := range model.FunctionalCapabilities {
		if _, err := types.ParseFunctionalCapability(capStr); err != nil {
			return fmt.Errorf("invalid functional capability '%s': %w", capStr, err)
		}
	}

	if len(model.Endpoints) == 0 {
		return fmt.Errorf("model.endpoints is required and must have at least one endpoint")
	}

	return nil
}

// validateParameters validates the parameters structure
func (v *ModelValidator) validateParameters(model *config.EnhancedModelConfig) error {
	if model.Parameters == nil {
		return fmt.Errorf("model.parameters is required")
	}

	// Validate maxInputTokens
	if err := v.validateMaxInputTokens(model.Parameters); err != nil {
		return err
	}

	// Validate max_tokens if required
	if v.requiresMaxOutputTokens(model) {
		if err := v.validateMaxOutputTokens(model.Parameters); err != nil {
			return err
		}
	}

	return nil
}

// validateMaxInputTokens validates maxInputTokens parameter
func (v *ModelValidator) validateMaxInputTokens(parameters map[string]config.ParameterSpec) error {
	maxInputTokens, exists := parameters["maxInputTokens"]
	if !exists {
		return fmt.Errorf("model.parameters.maxInputTokens is required")
	}

	if maxInputTokens.Type == "" {
		return fmt.Errorf("model.parameters.maxInputTokens.type is required")
	}

	return nil
}

// validateMaxOutputTokens validates max_tokens parameter
func (v *ModelValidator) validateMaxOutputTokens(parameters map[string]config.ParameterSpec) error {
	maxTokens, exists := parameters["maxOutputTokens"]
	if !exists {
		return fmt.Errorf("model.parameters.maxOutputTokens is required")
	}

	if maxTokens.Type == "" {
		return fmt.Errorf("model.parameters.maxOutputTokens.type is required")
	}

	if maxTokens.Maximum == nil {
		return fmt.Errorf("model.parameters.maxOutputTokens.maximum is required")
	}

	return nil
}

// validateModelKEYFormat validates the auto-calculated Model.KEY format
func (v *ModelValidator) validateModelKEYFormat(modelKEY string, pathInfo *PathInfo, model *config.EnhancedModelConfig) error {
	expectedKEY := fmt.Sprintf("%s/%s/%s/%s/%s",
		pathInfo.Infrastructure,
		pathInfo.Provider,
		pathInfo.Creator,
		model.Name,
		model.Version,
	)

	if modelKEY != expectedKEY {
		return fmt.Errorf("model.modelKEY format is invalid: expected '%s', got '%s'", expectedKEY, modelKEY)
	}

	return nil
}

// calculateModelKEY automatically calculates the Model.KEY based on path and model info
func (v *ModelValidator) calculateModelKEY(pathInfo *PathInfo, model *config.EnhancedModelConfig) string {
	// Format: infrastructure/provider/creator/model-name/model-version
	return fmt.Sprintf("%s/%s/%s/%s/%s",
		pathInfo.Infrastructure,
		pathInfo.Provider,
		pathInfo.Creator,
		model.Name,
		model.Version,
	)
}

// isEmbeddingModel determines if a model is an embedding model based on its functional capabilities or endpoints
func (v *ModelValidator) isEmbeddingModel(model *config.EnhancedModelConfig) bool {
	// Check functional capabilities for embedding
	for _, capability := range model.FunctionalCapabilities {
		if strings.Contains(strings.ToLower(capability), "embedding") {
			return true
		}
	}

	// Check endpoints for embedding-related paths
	for _, endpoint := range model.Endpoints {
		if strings.Contains(strings.ToLower(string(endpoint.Path)), "embedding") {
			return true
		}
	}

	return false
}

// requiresMaxOutputTokens determines if a model requires max_tokens.maximum based on outputModalities
// If outputModalities contain modalities other than "image", "audio", or "embedding", then max_tokens.maximum is required
func (v *ModelValidator) requiresMaxOutputTokens(model *config.EnhancedModelConfig) bool {
	// If no outputModalities are specified, check if it's an embedding model
	if len(model.Capabilities.OutputModalities) == 0 {
		// Embedding models don't require max_tokens
		if v.isEmbeddingModel(model) {
			return false
		}
		// Otherwise, assume it requires max_tokens
		return true
	}

	// Check if any outputModality is not "image", "audio", or "embedding"
	for _, modality := range model.Capabilities.OutputModalities {
		modalityLower := strings.ToLower(modality)
		if modalityLower != "image" && modalityLower != "audio" && modalityLower != "embedding" {
			return true
		}
	}

	// All outputModalities are "image", "audio", or "embedding", so max_tokens.maximum is optional
	return false
}
