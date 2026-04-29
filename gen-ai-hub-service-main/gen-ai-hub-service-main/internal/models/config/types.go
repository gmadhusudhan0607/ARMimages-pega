/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package config

import (
	"context"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

// CreatorConfig represents a creator-specific configuration
type CreatorConfig struct {
	Models []ModelConfig `yaml:"models" json:"models"`
}

// ProviderConfig represents a provider-specific configuration
type ProviderConfig struct {
	Provider string                   `yaml:"provider" json:"provider"`
	Creators map[string]CreatorConfig `yaml:"creators" json:"creators"`
}

// ModelGroup represents a collection of related models with enhanced metadata
type ModelGroup struct {
	Infrastructure types.Infrastructure  `yaml:"infrastructure" json:"infrastructure"`
	Provider       types.Provider        `yaml:"provider" json:"provider"`
	Creator        types.Creator         `yaml:"creator" json:"creator"`
	Models         []EnhancedModelConfig `yaml:"models" json:"models"`
	Metadata       GroupMetadata         `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// GroupMetadata contains metadata about a model group
type GroupMetadata struct {
	Description   string `yaml:"description,omitempty" json:"description,omitempty"`
	Documentation string `yaml:"documentation,omitempty" json:"documentation,omitempty"`
}

// EnhancedModelConfig extends the current ModelConfig with additional fields
type EnhancedModelConfig struct {
	KEY                    string                   `yaml:"key" json:"key" validate:"required"`
	Name                   string                   `yaml:"name" json:"name" validate:"required"`
	Version                string                   `yaml:"version" json:"version" validate:"required"`
	Label                  string                   `yaml:"label" json:"label"`
	FunctionalCapabilities []string                 `yaml:"functionalCapabilities" json:"functionalCapabilities" validate:"min=1"`
	Endpoints              []EndpointConfig         `yaml:"endpoints" json:"endpoints" validate:"required,min=1,dive"`
	Capabilities           ModelCapabilitiesConfig  `yaml:"capabilities" json:"capabilities"`
	Parameters             map[string]ParameterSpec `yaml:"parameters" json:"parameters" validate:"required,dive"`
	DeploymentInfo         DeploymentConfig         `yaml:"deployment,omitempty" json:"deployment,omitempty"`
	Lifecycle              LifecycleConfig          `yaml:"lifecycle,omitempty" json:"lifecycle,omitempty"`
}

// DeploymentConfig contains infrastructure-specific deployment information
type DeploymentConfig struct {
	Region       string            `yaml:"region,omitempty" json:"region,omitempty"`
	InstanceType string            `yaml:"instanceType,omitempty" json:"instanceType,omitempty"`
	Scaling      ScalingConfig     `yaml:"scaling,omitempty" json:"scaling,omitempty"`
	CustomConfig map[string]string `yaml:"customConfig,omitempty" json:"customConfig,omitempty"`
}

// ScalingConfig contains scaling configuration
type ScalingConfig struct {
	MinInstances int `yaml:"minInstances,omitempty" json:"minInstances,omitempty"`
	MaxInstances int `yaml:"maxInstances,omitempty" json:"maxInstances,omitempty"`
}

// LifecycleConfig contains model lifecycle information
type LifecycleConfig struct {
	Status          string `yaml:"status,omitempty" json:"status,omitempty"`
	DeprecationDate string `yaml:"deprecationDate,omitempty" json:"deprecationDate,omitempty"`
	EndOfLifeDate   string `yaml:"endOfLifeDate,omitempty" json:"endOfLifeDate,omitempty"`
}

// ModelConfig represents a model configuration from YAML
type ModelConfig struct {
	Name                   string                   `yaml:"name" json:"name" validate:"required"`
	Version                string                   `yaml:"version" json:"version" validate:"required"`
	Label                  string                   `yaml:"label" json:"label"`
	FunctionalCapabilities []string                 `yaml:"functionalCapabilities" json:"functionalCapabilities" validate:"min=1"` // New field for functional capabilities
	Endpoints              []EndpointConfig         `yaml:"endpoints" json:"endpoints" validate:"required,min=1,dive"`
	Capabilities           ModelCapabilitiesConfig  `yaml:"capabilities" json:"capabilities"`
	Parameters             map[string]ParameterSpec `yaml:"parameters" json:"parameters" validate:"required,dive"`
}

// EndpointConfig represents an endpoint configuration
type EndpointConfig struct {
	Path types.Endpoint `yaml:"path" json:"path"`
}

// ModelCapabilitiesConfig represents model capabilities from configuration
type ModelCapabilitiesConfig struct {
	Features         []string `yaml:"features,omitempty" json:"features,omitempty"`
	InputModalities  []string `yaml:"inputModalities,omitempty" json:"inputModalities,omitempty"`
	OutputModalities []string `yaml:"outputModalities,omitempty" json:"outputModalities,omitempty"`
	MimeTypes        []string `yaml:"mimeTypes,omitempty" json:"mimeTypes,omitempty"`
}

// ParameterSpec represents a parameter specification
type ParameterSpec struct {
	Title       string      `yaml:"title" json:"title"`
	Description string      `yaml:"description" json:"description"`
	Type        string      `yaml:"type" json:"type"`
	Default     interface{} `yaml:"default" json:"default"`
	Maximum     interface{} `yaml:"maximum" json:"maximum"`
	Minimum     interface{} `yaml:"minimum" json:"minimum"`
	Required    bool        `yaml:"required" json:"required"`
}

// ToModel converts ModelConfig to types.Model
func (mc *ModelConfig) ToModel(provider types.Provider, creator types.Creator) *types.Model {
	// Convert capabilities
	capabilities := types.ModelCapabilities{
		Features:         mc.Capabilities.Features,
		InputModalities:  mc.Capabilities.InputModalities,
		OutputModalities: mc.Capabilities.OutputModalities,
		MimeTypes:        mc.Capabilities.MimeTypes,
	}

	// Convert parameters
	parameters := make(map[string]types.ParameterSpec)
	for key, param := range mc.Parameters {
		parameters[key] = types.ParameterSpec{
			Title:       param.Title,
			Description: param.Description,
			Type:        param.Type,
			Default:     param.Default,
			Maximum:     param.Maximum,
			Minimum:     param.Minimum,
			Required:    param.Required,
		}
	}

	// Convert endpoints to string slice and determine endpoint type
	endpoints := make([]types.Endpoint, len(mc.Endpoints))
	for i, endpoint := range mc.Endpoints {
		// Normalize the endpoint to handle leading/trailing slashes
		if normalized, err := types.NormalizeEndpoint(string(endpoint.Path)); err == nil {
			endpoints[i] = normalized
		} else {
			// Log the error but keep the original value for backward compatibility
			log := cntx.LoggerFromContext(context.Background()).Sugar()
			log.Warnf("Failed to normalize endpoint '%s': %v, using original value", endpoint.Path, err)
			endpoints[i] = endpoint.Path
		}
	}

	// Convert functional capabilities
	var functionalCapabilities []types.FunctionalCapability
	for _, capStr := range mc.FunctionalCapabilities {
		if cap, err := types.ParseFunctionalCapability(capStr); err == nil {
			functionalCapabilities = append(functionalCapabilities, cap)
		} else {
			// Log the error but continue processing other capabilities
			log := cntx.LoggerFromContext(context.Background()).Sugar()
			log.Warnf("Failed to parse functional capability '%s': %v", capStr, err)
			continue
		}
	}

	return &types.Model{
		KEY:                    "",
		Name:                   mc.Name,
		Version:                mc.Version,
		Label:                  mc.Label,
		FunctionalCapabilities: functionalCapabilities,
		Provider:               provider,
		Creator:                creator,
		Capabilities:           capabilities,
		Parameters:             parameters,
		Endpoints:              endpoints,
	}
}

// ToModel converts EnhancedModelConfig to types.Model
func (emc *EnhancedModelConfig) ToModel(infrastructure types.Infrastructure, provider types.Provider, creator types.Creator) *types.Model {
	// Convert capabilities
	capabilities := types.ModelCapabilities{
		Features:         emc.Capabilities.Features,
		InputModalities:  emc.Capabilities.InputModalities,
		OutputModalities: emc.Capabilities.OutputModalities,
		MimeTypes:        emc.Capabilities.MimeTypes,
	}

	// Convert parameters
	parameters := make(map[string]types.ParameterSpec)
	for key, param := range emc.Parameters {
		parameters[key] = types.ParameterSpec{
			Title:       param.Title,
			Description: param.Description,
			Type:        param.Type,
			Default:     param.Default,
			Maximum:     param.Maximum,
			Minimum:     param.Minimum,
			Required:    param.Required,
		}
	}

	// Convert endpoints to string slice and determine endpoint type
	endpoints := make([]types.Endpoint, len(emc.Endpoints))
	for i, endpoint := range emc.Endpoints {
		// Normalize the endpoint to handle leading/trailing slashes
		if normalized, err := types.NormalizeEndpoint(string(endpoint.Path)); err == nil {
			endpoints[i] = normalized
		} else {
			// Log the error but keep the original value for backward compatibility
			log := cntx.LoggerFromContext(context.Background()).Sugar()
			log.Warnf("Failed to normalize endpoint '%s': %v, using original value", endpoint.Path, err)
			endpoints[i] = endpoint.Path
		}
	}

	// Convert functional capabilities
	var functionalCapabilities []types.FunctionalCapability
	for _, capStr := range emc.FunctionalCapabilities {
		if cap, err := types.ParseFunctionalCapability(capStr); err == nil {
			functionalCapabilities = append(functionalCapabilities, cap)
		} else {
			// Log the error but continue processing other capabilities
			log := cntx.LoggerFromContext(context.Background()).Sugar()
			log.Warnf("Failed to parse functional capability '%s': %v", capStr, err)
			continue
		}
	}

	return &types.Model{
		KEY:                    emc.KEY,
		Name:                   emc.Name,
		Version:                emc.Version,
		Label:                  emc.Label,
		FunctionalCapabilities: functionalCapabilities,
		Infrastructure:         infrastructure,
		Provider:               provider,
		Creator:                creator,
		Capabilities:           capabilities,
		Parameters:             parameters,
		Endpoints:              endpoints,
	}
}
