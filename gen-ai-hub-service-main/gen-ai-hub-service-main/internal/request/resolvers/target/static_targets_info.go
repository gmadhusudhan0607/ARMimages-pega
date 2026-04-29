/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

// TargetInfo contains static metadata about a model target
type TargetInfo struct {
	Infrastructure types.Infrastructure
	Provider       types.Provider
	Creator        types.Creator
	ModelName      string
	ModelVersion   string
}

// StaticTargetsByModelName provides predefined target information for known models
// Key: model name as it appears in requests (e.g., "gpt-35-turbo")
// Value: Complete target metadata including infrastructure, provider, creator, model-name, and model-version
//
// This map is used to enrich ResolvedTarget when models are resolved from CONFIGURATION_FILE
// and lack complete metadata like model-version information.
var StaticTargetsByModelName = map[string]TargetInfo{
	// Azure OpenAI - GPT Models
	"gpt-35-turbo": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-35-turbo",
		ModelVersion:   "1106",
	},
	"gpt-35-turbo-1106": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-35-turbo",
		ModelVersion:   "1106",
	},
	"gpt-4o": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4o",
		ModelVersion:   "2024-11-20",
	},
	"gpt-4o-2024-11-20": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4o",
		ModelVersion:   "2024-11-20",
	},
	"gpt-4o-2024-05-13": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4o",
		ModelVersion:   "2024-05-13",
	},
	"gpt-4o-mini": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4o-mini",
		ModelVersion:   "2024-07-18",
	},
	"gpt-4-preview": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4-preview",
		ModelVersion:   "1106",
	},
	"gpt-4-1106-preview": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4-preview",
		ModelVersion:   "1106",
	},
	"gpt-4-vision-preview": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-4-vision-preview",
		ModelVersion:   "1106",
	},
	"dall-e-3": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "dall-e-3",
		ModelVersion:   "3.0",
	},

	// Azure OpenAI - Realtime Models
	"gpt-realtime": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-realtime",
		ModelVersion:   "2025-08-28",
	},
	"gpt-realtime-mini": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-realtime-mini",
		ModelVersion:   "2025-12-15",
	},
	"gpt-realtime-1.5": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "gpt-realtime-1.5",
		ModelVersion:   "2026-02-23",
	},

	// Azure OpenAI - Embedding Models
	"text-embedding-ada-002": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "text-embedding-ada-002",
		ModelVersion:   "2",
	},
	"text-embedding-3-large": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "text-embedding-3-large",
		ModelVersion:   "1",
	},
	"text-embedding-3-small": {
		Infrastructure: types.InfrastructureAzure,
		Provider:       types.ProviderAzure,
		Creator:        types.CreatorOpenAI,
		ModelName:      "text-embedding-3-small",
		ModelVersion:   "1",
	},

	// GCP Vertex AI - Google Gemini Models
	"gemini-1.5-flash": {
		Infrastructure: types.InfrastructureGCP,
		Provider:       types.ProviderVertex,
		Creator:        types.CreatorGoogle,
		ModelName:      "gemini-1.5-flash",
		ModelVersion:   "002",
	},
	"gemini-1.5-pro": {
		Infrastructure: types.InfrastructureGCP,
		Provider:       types.ProviderVertex,
		Creator:        types.CreatorGoogle,
		ModelName:      "gemini-1.5-pro",
		ModelVersion:   "002",
	},
	"gemini-2.0-flash": {
		Infrastructure: types.InfrastructureGCP,
		Provider:       types.ProviderVertex,
		Creator:        types.CreatorGoogle,
		ModelName:      "gemini-2.0-flash",
		ModelVersion:   "001",
	},

	// GCP Vertex AI - Google Imagen Models (populated via init())

	// GCP Vertex AI - Google Embedding Models
	"text-multilingual-embedding-002": {
		Infrastructure: types.InfrastructureGCP,
		Provider:       types.ProviderVertex,
		Creator:        types.CreatorGoogle,
		ModelName:      "text-multilingual-embedding",
		ModelVersion:   "002",
	},
}

// imagenModelDef defines an Imagen model family with its short name, dotted name, canonical model name,
// and version.
type imagenModelDef struct {
	shortName  string // e.g. "imagen-3", "imagen-4-fast"
	dottedName string // e.g. "imagen-3.0", "imagen-4.0-fast"
	modelName  string // canonical model name for TargetInfo
	version    string // model version
}

func init() {
	// Register Imagen models for both short names (e.g. "imagen-3") and dotted names (e.g. "imagen-3.0").
	imagenModels := []imagenModelDef{
		// Imagen 3 models
		{"imagen-3", "imagen-3.0", "imagen-3.0", "generate-002"},
		{"imagen-3-fast", "imagen-3.0-fast", "imagen-3.0-fast", "generate-001"},
		// Imagen 4 models
		{"imagen-4", "imagen-4.0", "imagen-4.0", "generate-001"},
		{"imagen-4-fast", "imagen-4.0-fast", "imagen-4.0-fast", "fast-generate-001"},
		{"imagen-4-ultra", "imagen-4.0-ultra", "imagen-4.0-ultra", "ultra-generate-001"},
	}

	for _, m := range imagenModels {
		registerImagenModel(m)
	}
}

// registerImagenModel adds entries for both short and dotted naming styles of an Imagen model
// into StaticTargetsByModelName.
func registerImagenModel(m imagenModelDef) {
	info := TargetInfo{
		Infrastructure: types.InfrastructureGCP,
		Provider:       types.ProviderVertex,
		Creator:        types.CreatorGoogle,
		ModelName:      m.modelName,
		ModelVersion:   m.version,
	}

	// Short name (e.g. imagen-4)
	StaticTargetsByModelName[m.shortName] = info

	// Dotted name (e.g. imagen-4.0)
	StaticTargetsByModelName[m.dottedName] = info
}

// enrichFromStaticInfo enriches the ResolvedTarget with information from StaticTargetsByModelName.
// For infrastructure, creator, and modelVersion: only fills in empty/missing fields.
// For provider and modelName: always overrides to ensure correct resolution.
func enrichFromStaticInfo(target *ResolvedTarget, originalModelName string) {
	if originalModelName == "" {
		return
	}

	info, found := StaticTargetsByModelName[originalModelName]
	if !found {
		return
	}

	// Only fill in missing values for these fields
	if target.Infrastructure == "" {
		target.Infrastructure = info.Infrastructure
	}

	// Always override Provider to ensure correct model lookup
	// Configuration files may have incorrect provider values that need correction
	target.Provider = info.Provider

	if target.Creator == "" {
		target.Creator = info.Creator
	}

	// Always override ModelName to resolve aliases to canonical names
	target.ModelName = info.ModelName

	if target.ModelVersion == "" {
		// Default fallback
		target.ModelVersion = info.ModelVersion

		// For gpt-4o, prefer dynamically fetched version
		if isGPT4oModel(info.ModelName) {
			if dynamicVersion := GetGPT4oVersion(); dynamicVersion != "" {
				target.ModelVersion = dynamicVersion
			}
		}
	}
}
