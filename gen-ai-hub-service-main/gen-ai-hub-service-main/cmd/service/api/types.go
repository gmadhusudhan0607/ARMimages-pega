/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package api

type RespErr struct {
	StatusCode int64  `json:"statusCode"`
	Message    string `json:"message"`
}

type Capabilities struct {
	ChatCompletion  bool `yaml:"completions" json:"completions"`
	Embeddings      bool `yaml:"embeddings" json:"embeddings"`
	ImageGeneration bool `yaml:"image-generation" json:"image-generation"`
}

// this struct now need comments on it about how it is used both for Azure, GCP and AWS, and what is
// expected to be on each kind of mapping.
type Model struct {
	Name           string       `yaml:"name" json:"name"`
	ModelId        string       `yaml:"modelId" json:"modelId"`
	ModelUrl       string       `yaml:"modelUrl" json:"modelUrl"`
	RedirectURL    string       `yaml:"redirectUrl" json:"-"`
	Capabilities   Capabilities `yaml:"capabilities" json:"capabilities"`
	Provider       string       `yaml:"provider" json:"provider"`
	Creator        string       `yaml:"creator" json:"creator"`
	TargetAPI      string       `yaml:"targetAPI" json:"targetAPI"`
	Path           string       `yaml:"path" json:"path"`
	Infrastructure string       `yaml:"infrastructure" json:"infrastructure"`
	ModelMapping   string       `json:"ModelMapping,omitempty"`
	OIDCIAMRole    string       `json:"OIDCIAMRoleArn,omitempty"`
	ApiKey         string       `yaml:"apiKey" json:"apiKey"`
	Active         bool         `yaml:"active" json:"active"`
}

type Buddy struct {
	Name        string `yaml:"name" json:"name"`
	BuddyUrl    string `yaml:"buddyUrl" json:"buddyUrl"`
	RedirectURL string `yaml:"redirectUrl" json:"-"`
}

type AppError struct {
	Error   error
	Message string
}

type ModelUrlParams struct {
	ModelName   string
	IsolationId string
}

type BuddyUrlParams struct {
	BuddyId     string
	IsolationId string
}

type Mapping struct {
	Models  []Model `yaml:"models" json:"models"`
	Buddies []Buddy `yaml:"buddies" json:"buddies"`
}

type ReqBodyType struct {
	ModelId string `json:"modelId"`
}
type ModelInfo struct {
	// Autopilot-aligned fields (primary)
	Provider              string                   `json:"provider"`
	Creator               string                   `json:"creator"`
	ModelName             string                   `json:"model_name"`
	Description           string                   `json:"description"`
	ModelMappingId        string                   `json:"model_mapping_id,omitempty"`
	Name                  string                   `json:"name"`
	InputTokens           *int                     `json:"input_tokens,omitempty"`
	OutputTokens          *int                     `json:"output_tokens,omitempty"`
	Type                  string                   `json:"type"`
	ModelID               string                   `json:"model_id"`
	DefaultModel          bool                     `json:"default_model"`
	Version               string                   `json:"version"`
	DeprecationInfo       DeprecationInfo          `json:"deprecation_info"`
	SupportedCapabilities SupportedCapabilities    `json:"supported_capabilities"`
	Parameters            map[string]ParameterSpec `json:"parameters"`
	AlternateModelInfo    *AlternateModelInfo      `json:"alternate_model_info,omitempty"`
	Examples              []string                 `json:"examples,omitempty"`

	// Legacy/Internal fields (maintain compatibility)
	ModelLabel      string   `json:"model_label,omitempty"`
	ModelPath       []string `json:"model_path,omitempty"`
	Lifecycle       string   `json:"lifecycle,omitempty"`
	DeprecationDate string   `json:"deprecation_date,omitempty"`
}
type ModelMetadata struct {
	ModelCapabilities  ModelCapabilities        `yaml:"model_capabilities" json:"model_capabilities"`
	Lifecycle          string                   `yaml:"lifecycle,omitempty" json:"lifecycle,omitempty"`
	DeprecationDate    string                   `yaml:"deprecation_date,omitempty" json:"deprecation_date,omitempty"`
	AlternateModelInfo *AlternateModelInfo      `yaml:"alternate_model_info,omitempty" json:"alternate_model_info,omitempty"`
	Parameters         map[string]ParameterSpec `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	ModelLabel         string                   `yaml:"model_label" json:"model_label"`
	ModelName          string                   `yaml:"model_name" json:"model_name"`
	ModelDescription   string                   `yaml:"model_description" json:"model_description"`
	ModelMappingId     string                   `yaml:"model_mapping_id,omitempty" json:"model_mapping_id,omitempty"`
	Provider           string                   `yaml:"provider,omitempty" json:"provider,omitempty"`
	Creator            string                   `yaml:"creator,omitempty" json:"creator,omitempty"`
	Type               string                   `yaml:"type" json:"type"`
	Version            string                   `yaml:"version" json:"version"`
	ModelID            string                   `yaml:"model_id" json:"model_id"`
	InputTokens        *int                     `yaml:"input_tokens" json:"input_tokens"`
	Examples           []string                 `yaml:"examples,omitempty" json:"examples,omitempty"`
}
type ModelCapabilities struct {
	InputModalities  []string `json:"input_modalities,omitempty" yaml:"input_modalities"`
	OutputModalities []string `json:"output_modalities,omitempty" yaml:"output_modalities"`
	Features         []string `json:"features,omitempty" yaml:"features"`
	MimeTypes        []string `json:"mime_types,omitempty" yaml:"mime_types"`
}

type DefaultModels struct {
	Fast  *ModelInfo `json:"fast,omitempty"`
	Smart *ModelInfo `json:"smart,omitempty"`
	Pro   *ModelInfo `json:"pro,omitempty"`
}
type Parameter struct {
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
	Required    bool        `json:"required,omitempty"`
}

// ParameterSpec represents the enhanced parameter specification for Autopilot compatibility
type ParameterSpec struct {
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
	Maximum     *float64    `json:"maximum,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Title       string      `json:"title"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Examples    []string    `json:"examples,omitempty"`
}

// DeprecationInfo represents model deprecation information
type DeprecationInfo struct {
	IsDeprecated             bool   `json:"is_deprecated"`
	ScheduledDeprecationDate string `json:"scheduled_deprecation_date,omitempty"`
}

// SupportedCapabilities represents the capabilities supported by the model
type SupportedCapabilities struct {
	Streaming               bool     `json:"streaming"`
	Multimodal              []string `json:"multimodal"`
	Functions               bool     `json:"functions"`
	ParallelFunctionCalling bool     `json:"parallel_function_calling"`
	JSONMode                bool     `json:"json_mode"`
	IsMultimodal            bool     `json:"is_multimodal"`
}

// AlternateModelInfo represents alternate model information
type AlternateModelInfo struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Creator  string `json:"creator"`
}

// ModelsResponse represents the response structure for the GET /models endpoint
type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
	Errors []string    `json:"errors,omitempty"`
}
