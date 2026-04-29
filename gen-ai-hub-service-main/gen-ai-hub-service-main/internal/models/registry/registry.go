/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package registry

import (
	"context"
	"fmt"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"regexp"
	"sort"
	"strings"
	"sync"

	modelerrors "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/errors"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/utils"
)

// Registry provides fast 5-parameter model lookups with enhanced performance
type Registry struct {
	mu                sync.RWMutex
	models            map[types.ModelKey]*types.Model
	versionComparator *utils.VersionComparator

	// Pre-computed indexes for performance
	indexes struct {
		byInfrastructure map[types.Infrastructure][]*types.Model
		byProvider       map[types.Provider][]*types.Model
		byCreator        map[types.Creator][]*types.Model
		byModelName      map[string][]*types.Model
		byComposite      map[string][]*types.Model // Combined keys for faster lookup
	}
}

// NewRegistry creates a new registry
func NewRegistry() *Registry {
	r := &Registry{
		models:            make(map[types.ModelKey]*types.Model),
		versionComparator: utils.NewVersionComparator(),
	}

	// Initialize indexes
	r.indexes.byInfrastructure = make(map[types.Infrastructure][]*types.Model)
	r.indexes.byProvider = make(map[types.Provider][]*types.Model)
	r.indexes.byCreator = make(map[types.Creator][]*types.Model)
	r.indexes.byModelName = make(map[string][]*types.Model)
	r.indexes.byComposite = make(map[string][]*types.Model)

	return r
}

// RegisterModel registers a single model in the registry
func (r *Registry) RegisterModel(model *types.Model) error {
	if model == nil {
		return fmt.Errorf("model cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := model.GetModelKey()

	// Check for duplicates
	if _, exists := r.models[key]; exists {
		return fmt.Errorf("model already exists: %s", key.String())
	}

	// Store the model
	r.models[key] = r.copyModel(model)

	return nil
}

// RegisterModels registers multiple models in the registry (batch operation)
func (r *Registry) RegisterModels(models []*types.Model) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing data for batch load
	r.models = make(map[types.ModelKey]*types.Model)
	r.indexes.byInfrastructure = make(map[types.Infrastructure][]*types.Model)
	r.indexes.byProvider = make(map[types.Provider][]*types.Model)
	r.indexes.byCreator = make(map[types.Creator][]*types.Model)
	r.indexes.byModelName = make(map[string][]*types.Model)
	r.indexes.byComposite = make(map[string][]*types.Model)

	// Load all models
	for _, model := range models {
		if model == nil {
			continue
		}

		key := model.GetModelKey()
		r.models[key] = r.copyModel(model)
	}

	return nil
}

// FindModel performs exact 5-parameter model lookup - version is required
func (r *Registry) FindModel(
	infrastructure types.Infrastructure,
	provider types.Provider,
	creator types.Creator,
	modelName string,
	version string,
) (*types.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Validate version is provided
	if strings.TrimSpace(version) == "" {
		return nil, fmt.Errorf("version parameter is required and cannot be empty for model %s/%s/%s",
			infrastructure, provider, creator)
	}

	// Exact match only
	key := types.ModelKey{
		Infrastructure: infrastructure,
		Provider:       provider,
		Creator:        creator,
		ModelName:      modelName,
		Version:        version,
	}
	if model, exists := r.models[key]; exists {
		return r.copyModel(model), nil
	}

	// Return error with available versions
	return nil, r.createVersionNotFoundError(infrastructure, provider, creator, modelName, version)
}

// FindLatestModel finds the latest version of a model - explicit function for latest version lookup
func (r *Registry) FindLatestModel(
	infrastructure types.Infrastructure,
	provider types.Provider,
	creator types.Creator,
	modelName string,
) (*types.Model, error) {
	return r.findLatestVersion(infrastructure, provider, creator, modelName)
}

// FindModelByIDPattern finds a model where the model's ID pattern matches the given modelID
// This supports wildcard patterns in the registry (e.g., "anthropic.claude-3-7-sonnet-*-v1:0")
// Returns the first matching model for the given infrastructure/provider/creator/modelName combination
func (r *Registry) FindModelByIDPattern(
	infrastructure types.Infrastructure,
	provider types.Provider,
	creator types.Creator,
	modelName string,
	modelID string,
) (*types.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find all models matching infrastructure/provider/creator/modelName
	var candidates []*types.Model
	for key, model := range r.models {
		if key.Infrastructure == infrastructure &&
			key.Provider == provider &&
			key.Creator == creator &&
			key.ModelName == modelName {
			// Check if the model's ID pattern matches the incoming modelID
			// model.KEY is the model ID string itself
			if matchesModelIDPattern(modelID, model.KEY) {
				candidates = append(candidates, model)
			}
		}
	}

	if len(candidates) == 0 {
		return nil, &modelerrors.ModelNotFoundError{
			Provider:  provider,
			Creator:   creator,
			ModelName: modelName,
		}
	}

	// If multiple matches, return the one with the latest version
	if len(candidates) > 1 {
		sort.Slice(candidates, func(i, j int) bool {
			return r.versionComparator.IsNewer(candidates[i].Version, candidates[j].Version)
		})
	}

	return r.copyModel(candidates[0]), nil
}

// findLatestVersion finds the latest version of a model
func (r *Registry) findLatestVersion(
	infrastructure types.Infrastructure,
	provider types.Provider,
	creator types.Creator,
	modelName string,
) (*types.Model, error) {
	var candidates []*types.Model

	// Find all matching models
	for key, model := range r.models {
		if key.Infrastructure == infrastructure &&
			key.Provider == provider &&
			key.Creator == creator &&
			key.ModelName == modelName {
			candidates = append(candidates, model)
		}
	}

	if len(candidates) == 0 {
		return nil, &modelerrors.ModelNotFoundError{
			Provider:  provider,
			Creator:   creator,
			ModelName: modelName,
		}
	}

	// Sort by version (descending) to get the latest using intelligent version comparison
	sort.Slice(candidates, func(i, j int) bool {
		return r.versionComparator.IsNewer(candidates[i].Version, candidates[j].Version)
	})

	return r.copyModel(candidates[0]), nil
}

// createVersionNotFoundError creates a VersionNotFoundError with available versions
func (r *Registry) createVersionNotFoundError(
	infrastructure types.Infrastructure,
	provider types.Provider,
	creator types.Creator,
	modelName, requestedVersion string,
) error {
	var availableVersions []string

	// Find all versions for this model
	for key := range r.models {
		if key.Infrastructure == infrastructure &&
			key.Provider == provider &&
			key.Creator == creator &&
			key.ModelName == modelName {
			availableVersions = append(availableVersions, key.Version)
		}
	}

	return &modelerrors.VersionNotFoundError{
		Provider:          provider,
		Creator:           creator,
		ModelName:         modelName,
		RequestedVersion:  requestedVersion,
		AvailableVersions: availableVersions,
	}
}

// GetModelsByInfrastructure returns all models for a specific infrastructure
func (r *Registry) GetModelsByInfrastructure(infrastructure types.Infrastructure) ([]*types.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if models, exists := r.indexes.byInfrastructure[infrastructure]; exists {
		result := make([]*types.Model, len(models))
		for i, model := range models {
			result[i] = r.copyModel(model)
		}
		return result, nil
	}

	return nil, fmt.Errorf("no models found for infrastructure: %s", infrastructure)
}

// GetModelsByProvider returns all models for a specific provider
func (r *Registry) GetModelsByProvider(provider types.Provider) ([]*types.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if models, exists := r.indexes.byProvider[provider]; exists {
		result := make([]*types.Model, len(models))
		for i, model := range models {
			result[i] = r.copyModel(model)
		}
		return result, nil
	}

	return nil, fmt.Errorf("no models found for provider: %s", provider)
}

// GetModelsByCreator returns all models for a specific creator
func (r *Registry) GetModelsByCreator(creator types.Creator) ([]*types.Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if models, exists := r.indexes.byCreator[creator]; exists {
		result := make([]*types.Model, len(models))
		for i, model := range models {
			result[i] = r.copyModel(model)
		}
		return result, nil
	}

	return nil, fmt.Errorf("no models found for creator: %s", creator)
}

// GetAllModels returns all registered models
func (r *Registry) GetAllModels() []*types.Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*types.Model, 0, len(r.models))
	for _, model := range r.models {
		result = append(result, r.copyModel(model))
	}

	return result
}

// GetModelCount returns the total number of registered models
func (r *Registry) GetModelCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.models)
}

// IsModelSupported checks if a model is supported for the given parameters
func (r *Registry) IsModelSupported(
	infrastructure types.Infrastructure,
	provider types.Provider,
	creator types.Creator,
	modelName string,
) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if any version of this model exists
	for key := range r.models {
		if key.Infrastructure == infrastructure &&
			key.Provider == provider &&
			key.Creator == creator &&
			key.ModelName == modelName {
			return true
		}
	}
	return false
}

// GetAvailableVersions returns all available versions for a model
func (r *Registry) GetAvailableVersions(
	infrastructure types.Infrastructure,
	provider types.Provider,
	creator types.Creator,
	modelName string,
) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var versions []string
	for key := range r.models {
		if key.Infrastructure == infrastructure &&
			key.Provider == provider &&
			key.Creator == creator &&
			key.ModelName == modelName {
			versions = append(versions, key.Version)
		}
	}

	// Sort versions using intelligent comparison
	sort.Slice(versions, func(i, j int) bool {
		return r.versionComparator.IsNewer(versions[i], versions[j])
	})

	return versions
}

// GetLatestVersion returns the latest version for a model
func (r *Registry) GetLatestVersion(
	infrastructure types.Infrastructure,
	provider types.Provider,
	creator types.Creator,
	modelName string,
) (string, bool) {
	versions := r.GetAvailableVersions(infrastructure, provider, creator, modelName)
	if len(versions) == 0 {
		return "", false
	}
	return versions[0], true
}

// RebuildIndexes creates all necessary indexes for fast lookups
func (r *Registry) RebuildIndexes() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing indexes
	r.clearIndexes()

	// Build indexes
	r.buildModelIndexes()

	// Sort all indexes
	r.sortAllIndexes()

	return nil
}

// clearIndexes clears all existing indexes
func (r *Registry) clearIndexes() {
	r.indexes.byInfrastructure = make(map[types.Infrastructure][]*types.Model)
	r.indexes.byProvider = make(map[types.Provider][]*types.Model)
	r.indexes.byCreator = make(map[types.Creator][]*types.Model)
	r.indexes.byModelName = make(map[string][]*types.Model)
	r.indexes.byComposite = make(map[string][]*types.Model)
}

// buildModelIndexes builds indexes for all models
func (r *Registry) buildModelIndexes() {
	for _, model := range r.models {
		r.addModelToIndexes(model)
	}
}

// addModelToIndexes adds a single model to all indexes
func (r *Registry) addModelToIndexes(model *types.Model) {
	// Index by infrastructure
	r.indexes.byInfrastructure[model.Infrastructure] = append(
		r.indexes.byInfrastructure[model.Infrastructure], model)

	// Index by provider
	r.indexes.byProvider[model.Provider] = append(
		r.indexes.byProvider[model.Provider], model)

	// Index by creator
	r.indexes.byCreator[model.Creator] = append(
		r.indexes.byCreator[model.Creator], model)

	// Index by model name
	r.indexes.byModelName[model.Name] = append(
		r.indexes.byModelName[model.Name], model)

	// Build composite indexes
	r.addToCompositeIndexes(model)
}

// addToCompositeIndexes adds model to composite indexes
func (r *Registry) addToCompositeIndexes(model *types.Model) {
	compositeKeys := []string{
		fmt.Sprintf("%s/%s/%s", model.Infrastructure, model.Provider, model.Creator),
		fmt.Sprintf("%s/%s/%s/%s", model.Infrastructure, model.Provider, model.Creator, model.Name),
	}

	for _, compositeKey := range compositeKeys {
		r.indexes.byComposite[compositeKey] = append(
			r.indexes.byComposite[compositeKey], model)
	}
}

// sortAllIndexes sorts all indexes by version
func (r *Registry) sortAllIndexes() {
	r.sortIndexMap(r.indexes.byInfrastructure)
	r.sortIndexMap(r.indexes.byProvider)
	r.sortIndexMap(r.indexes.byCreator)
	r.sortIndexMap(r.indexes.byModelName)
	r.sortIndexMap(r.indexes.byComposite)
}

// sortIndexMap sorts any index map by version
func (r *Registry) sortIndexMap(indexMap interface{}) {
	switch idx := indexMap.(type) {
	case map[types.Infrastructure][]*types.Model:
		for _, models := range idx {
			r.sortModelsByVersion(models)
		}
	case map[types.Provider][]*types.Model:
		for _, models := range idx {
			r.sortModelsByVersion(models)
		}
	case map[types.Creator][]*types.Model:
		for _, models := range idx {
			r.sortModelsByVersion(models)
		}
	case map[string][]*types.Model:
		for _, models := range idx {
			r.sortModelsByVersion(models)
		}
	default:
		// Unexpected index map type - log for debugging but don't fail
		// This should not happen in normal operation
		log := cntx.LoggerFromContext(context.Background()).Sugar()
		log.Warnf("Failed to sort index map: unexpected type %T", indexMap)

	}
}

// sortModelsByVersion sorts models by version using intelligent comparison
func (r *Registry) sortModelsByVersion(models []*types.Model) {
	sort.Slice(models, func(i, j int) bool {
		return r.versionComparator.IsNewer(models[i].Version, models[j].Version)
	})
}

// matchesModelIDPattern checks if a model ID matches a pattern that may contain wildcards
// Supports wildcards: * (matches any sequence of characters)
// Example: "anthropic.claude-3-7-sonnet-*-v1:0" matches "anthropic.claude-3-7-sonnet-20250219-v1:0"
func matchesModelIDPattern(modelID, pattern string) bool {
	if pattern == "" || modelID == "" {
		return false
	}

	// If no wildcard, do exact match
	if !strings.Contains(pattern, "*") {
		return modelID == pattern
	}

	// Convert wildcard pattern to regex
	// Escape special regex characters except *
	regexPattern := regexp.QuoteMeta(pattern)
	// Replace escaped \* with .* (match any characters)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, `.*`)
	// Anchor the pattern to match the entire string
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, modelID)
	if err != nil {
		return false
	}
	return matched
}

// copyModel creates a deep copy of a model to prevent external modifications
func (r *Registry) copyModel(model *types.Model) *types.Model {
	if model == nil {
		return nil
	}

	copy := &types.Model{
		KEY:             model.KEY,
		Name:            model.Name,
		Version:         model.Version,
		Label:           model.Label,
		Capabilities:    model.Capabilities,
		DeprecationDate: model.DeprecationDate,
		Lifecycle:       model.Lifecycle,
		Creator:         model.Creator,
		Provider:        model.Provider,
		Infrastructure:  model.Infrastructure,
	}

	// Deep copy functional capabilities
	if model.FunctionalCapabilities != nil {
		copy.FunctionalCapabilities = make([]types.FunctionalCapability, len(model.FunctionalCapabilities))
		for i, capability := range model.FunctionalCapabilities {
			copy.FunctionalCapabilities[i] = capability
		}
	}

	// Deep copy parameters
	if model.Parameters != nil {
		copy.Parameters = make(map[string]types.ParameterSpec, len(model.Parameters))
		for k, v := range model.Parameters {
			copy.Parameters[k] = v
		}
	}

	// Deep copy endpoints
	if model.Endpoints != nil {
		copy.Endpoints = make([]types.Endpoint, len(model.Endpoints))
		for i, endpoint := range model.Endpoints {
			copy.Endpoints[i] = endpoint
		}
	}

	// Deep copy capabilities arrays
	if model.Capabilities.Features != nil {
		copy.Capabilities.Features = make([]string, len(model.Capabilities.Features))
		for i, feature := range model.Capabilities.Features {
			copy.Capabilities.Features[i] = feature
		}
	}
	if model.Capabilities.InputModalities != nil {
		copy.Capabilities.InputModalities = make([]string, len(model.Capabilities.InputModalities))
		for i, modality := range model.Capabilities.InputModalities {
			copy.Capabilities.InputModalities[i] = modality
		}
	}
	if model.Capabilities.OutputModalities != nil {
		copy.Capabilities.OutputModalities = make([]string, len(model.Capabilities.OutputModalities))
		for i, modality := range model.Capabilities.OutputModalities {
			copy.Capabilities.OutputModalities[i] = modality
		}
	}
	if model.Capabilities.MimeTypes != nil {
		copy.Capabilities.MimeTypes = make([]string, len(model.Capabilities.MimeTypes))
		for i, mimeType := range model.Capabilities.MimeTypes {
			copy.Capabilities.MimeTypes[i] = mimeType
		}
	}

	return copy
}
