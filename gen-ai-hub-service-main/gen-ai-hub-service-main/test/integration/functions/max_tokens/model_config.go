//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

// TokenLimitField represents which field to use for token limits
type TokenLimitField string

const (
	// MaxTokensField uses the "max_tokens" field (legacy)
	MaxTokensField TokenLimitField = "max_tokens"
	// MaxCompletionTokensField uses the "max_completion_tokens" field (newer APIs/models)
	MaxCompletionTokensField TokenLimitField = "max_completion_tokens"
)

// RequestCapabilities defines what features a model supports for request building
type RequestCapabilities struct {
	// SupportsStreaming indicates if the model supports streaming responses
	SupportsStreaming bool
	// SupportsTools indicates if the model supports function/tool calling
	SupportsTools bool
	// SupportsVision indicates if the model supports image inputs
	SupportsVision bool
	// MaxContextTokens is the maximum context length for the model
	MaxContextTokens int
}

// APIVersionInfo contains API version specific configuration
type APIVersionInfo struct {
	// Version is the API version string (e.g., "2023-05-15")
	Version string
	// TokenLimitField specifies which field to use for token limits
	TokenLimitField TokenLimitField
	// RequiredFields lists fields that must be present in requests
	RequiredFields []string
	// SupportedFields lists all fields supported by this API version
	SupportedFields []string
}

// RequestModelConfig defines the complete configuration for a model variant for request building
type RequestModelConfig struct {
	// ModelName is the base model name (e.g., "gpt-35-turbo")
	ModelName string
	// ModelVersion is the specific version (e.g., "0613", "1106", empty for base)
	ModelVersion string
	// Capabilities defines what the model supports
	Capabilities RequestCapabilities
	// SupportedAPIVersions maps API versions to their specific configurations
	SupportedAPIVersions map[string]*APIVersionInfo
	// DefaultAPIVersion is the recommended API version for this model
	DefaultAPIVersion string
}

// GetFullModelName returns the complete model identifier
func (rmc *RequestModelConfig) GetFullModelName() string {
	if rmc.ModelVersion == "" {
		return rmc.ModelName
	}
	return rmc.ModelName + "-" + rmc.ModelVersion
}

// IsAPIVersionSupported checks if an API version is supported
func (rmc *RequestModelConfig) IsAPIVersionSupported(apiVersion string) bool {
	_, exists := rmc.SupportedAPIVersions[apiVersion]
	return exists
}

// GetAPIVersionInfo returns configuration for a specific API version
func (rmc *RequestModelConfig) GetAPIVersionInfo(apiVersion string) *APIVersionInfo {
	if apiVersion == "" {
		apiVersion = rmc.DefaultAPIVersion
	}
	return rmc.SupportedAPIVersions[apiVersion]
}

// ValidateRequest checks if a request configuration is valid for this model
func (rmc *RequestModelConfig) ValidateRequest(apiVersion string, hasMaxTokens bool, hasStreaming bool, hasTools bool) error {
	apiInfo := rmc.GetAPIVersionInfo(apiVersion)
	if apiInfo == nil {
		return &RequestModelConfigError{
			Type:    "unsupported_api_version",
			Message: "API version '" + apiVersion + "' not supported for model " + rmc.GetFullModelName(),
		}
	}

	if hasStreaming && !rmc.Capabilities.SupportsStreaming {
		return &RequestModelConfigError{
			Type:    "unsupported_feature",
			Message: "Streaming not supported for model " + rmc.GetFullModelName(),
		}
	}

	if hasTools && !rmc.Capabilities.SupportsTools {
		return &RequestModelConfigError{
			Type:    "unsupported_feature",
			Message: "Tools/functions not supported for model " + rmc.GetFullModelName(),
		}
	}

	return nil
}

// RequestModelConfigError represents configuration validation errors
type RequestModelConfigError struct {
	Type    string
	Message string
}

func (e *RequestModelConfigError) Error() string {
	return e.Message
}
