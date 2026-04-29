//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

// ModelConfig represents configuration for a specific model variant
type ModelConfig struct {
	// Model identification
	Name        string // e.g., "gpt-35-turbo", "gpt-4o"
	DisplayName string // e.g., "gpt-35-turbo"

	// API configuration
	DeploymentName string // e.g., "gpt-35-turbo-1106", "gpt-4o-2024-11-20"
	URLPath        string // e.g., "/openai/deployments/gpt-35-turbo-1106/chat/completions"

	// Response configuration
	ResponseModelName string // Model name used in mock responses
	ResponseID        string // ID prefix for responses (e.g., "chatcmpl-test")
}

// Global model registry
var modelRegistry = map[string]*ModelConfig{
	"gpt-35-turbo": {
		Name:              "gpt-35-turbo",
		DisplayName:       "gpt-35-turbo",
		DeploymentName:    "gpt-35-turbo-1106",
		URLPath:           "/openai/deployments/gpt-35-turbo-1106/chat/completions",
		ResponseModelName: "gpt-35-turbo",
		ResponseID:        "chatcmpl-test",
	},
	"gpt-4o": {
		Name:              "gpt-4o",
		DisplayName:       "gpt-4o",
		DeploymentName:    "gpt-4o",
		URLPath:           "/openai/deployments/gpt-4o/chat/completions",
		ResponseModelName: "gpt-4o",
		ResponseID:        "chatcmpl-test-gpt4o",
	},
	"gpt-4o-mini": {
		Name:              "gpt-4o-mini",
		DisplayName:       "gpt-4o-mini",
		DeploymentName:    "gpt-4o-mini",
		URLPath:           "/openai/deployments/gpt-4o-mini/chat/completions",
		ResponseModelName: "gpt-4o-mini",
		ResponseID:        "chatcmpl-test-gpt4o-mini",
	},
	"gpt-4-preview": {
		Name:              "gpt-4-preview",
		DisplayName:       "gpt-4-preview",
		DeploymentName:    "gpt-4-1106-preview",
		URLPath:           "/openai/deployments/gpt-4-1106-preview/chat/completions",
		ResponseModelName: "gpt-4-preview",
		ResponseID:        "chatcmpl-test-gpt4-preview",
	},
}

// GetModelConfig retrieves model configuration by display name
func GetModelConfig(displayName string) *ModelConfig {
	if config, exists := modelRegistry[displayName]; exists {
		return config
	}
	return nil
}

// GetAllModelConfigs returns all available model configurations
func GetAllModelConfigs() []*ModelConfig {
	var configs []*ModelConfig
	for _, config := range modelRegistry {
		configs = append(configs, config)
	}
	return configs
}
