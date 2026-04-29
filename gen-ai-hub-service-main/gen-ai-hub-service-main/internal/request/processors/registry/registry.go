/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package registry

import (
	"fmt"
	"sync"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
)

// registryImpl implements the ProcessorRegistry interface
type registryImpl struct {
	mu         sync.RWMutex
	processors map[ProcessorKey]ProcessorFactory
}

// NewProcessorRegistry creates a new processor registry
func NewProcessorRegistry() ProcessorRegistry {
	return &registryImpl{
		processors: make(map[ProcessorKey]ProcessorFactory),
	}
}

// Register registers a processor factory for a given key
func (r *registryImpl) Register(key ProcessorKey, factory ProcessorFactory) error {
	if !key.IsValid() {
		return fmt.Errorf("invalid processor key: %s", key.String())
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.processors[key]; exists {
		return fmt.Errorf("processor already registered for key: %s", key.String())
	}

	r.processors[key] = factory
	return nil
}

// CreateProcessor creates a processor instance for the given key
func (r *registryImpl) CreateProcessor(key ProcessorKey) (interface{}, error) {
	r.mu.RLock()
	factory, exists := r.processors[key]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no processor registered for key: %s", key.String())
	}

	processor := factory()
	return processor, nil
}

// HasProcessor checks if a processor is registered for the given key
func (r *registryImpl) HasProcessor(key ProcessorKey) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.processors[key]
	return exists
}

// GetRegisteredKeys returns all registered processor keys
func (r *registryImpl) GetRegisteredKeys() []ProcessorKey {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]ProcessorKey, 0, len(r.processors))
	for key := range r.processors {
		keys = append(keys, key)
	}
	return keys
}

// GetSupportedCombinations returns all supported provider/model combinations
func (r *registryImpl) GetSupportedCombinations() map[ProcessorKey]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	combinations := make(map[ProcessorKey]string, len(r.processors))
	for key := range r.processors {
		combinations[key] = fmt.Sprintf("Provider: %s, Infrastructure: %s, Creator: %s, Endpoint: %s, Version: %s",
			key.Provider, key.Infrastructure, key.Creator, "fixme", key.Version)
	}
	return combinations
}

// Global registry instance
var globalRegistry ProcessorRegistry

// InitializeProcessorRegistry initializes the global processor registry
func InitializeProcessorRegistry() {
	globalRegistry = NewProcessorRegistry()
	registerAllProcessors()
}

// GetGlobalRegistry returns the global processor registry
func GetGlobalRegistry() ProcessorRegistry {
	if globalRegistry == nil {
		InitializeProcessorRegistry()
	}
	return globalRegistry
}

// registerAllProcessors registers all supported processor combinations
func registerAllProcessors() {
	registry := globalRegistry

	registerAzureOpenAIProcessors(registry)
	registerBedrockAmazonProcessors(registry)
	registerBedrockAnthropicProcessors(registry)
	registerBedrockMetaProcessors(registry)
	registerBedrockMistralProcessors(registry)
	registerVertexGoogleProcessors(registry)
}

// registerAzureOpenAIModel is a helper to register an Azure OpenAI model with a given extension factory.
// This reduces boilerplate for models sharing the same provider/infrastructure/creator.
func registerAzureOpenAIModel(registry ProcessorRegistry, modelID, version string, extensionFactory func() interface{}) {
	_ = registry.Register(ProcessorKey{
		Provider:       "azure",
		Infrastructure: "azure",
		Creator:        "openai",
		ModelID:        modelID,
		Version:        version,
	}, extensionFactory)
}

// registerAzureOpenAI20240201Model registers an Azure OpenAI model using the 2024-02-01 API extension.
func registerAzureOpenAI20240201Model(registry ProcessorRegistry, modelID, version string) {
	registerAzureOpenAIModel(registry, modelID, version, func() interface{} {
		return extensions.NewAzureOpenAI20240201Extension()
	})
}

// registerAzureOpenAI20250807Model registers an Azure OpenAI model using the 2025-08-07 API extension.
func registerAzureOpenAI20250807Model(registry ProcessorRegistry, modelID, version string) {
	registerAzureOpenAIModel(registry, modelID, version, func() interface{} {
		return extensions.NewAzureOpenAI20250807Extension()
	})
}

// registerAzureOpenAIProcessors registers all Azure OpenAI model processors
func registerAzureOpenAIProcessors(registry ProcessorRegistry) {
	registerAzureOpenAIGPTModels(registry)
	registerAzureOpenAIEmbeddingModels(registry)
	registerAzureOpenAIImageModels(registry)
	registerAzureOpenAIRealtimeModels(registry)
}

// registerAzureOpenAIGPTModels registers all Azure OpenAI GPT model processors
func registerAzureOpenAIGPTModels(registry ProcessorRegistry) {
	// GPT-3.5 Turbo models
	registerAzureOpenAI20240201Model(registry, "gpt-35-turbo", "0613")
	registerAzureOpenAI20240201Model(registry, "gpt-35-turbo", "0125")
	registerAzureOpenAI20240201Model(registry, "gpt-35-turbo", "1106")

	// GPT-4 models
	registerAzureOpenAI20240201Model(registry, "gpt-4", "0613")
	registerAzureOpenAI20240201Model(registry, "gpt-4", "0125-preview")
	registerAzureOpenAI20240201Model(registry, "gpt-4-preview", "1106")
	registerAzureOpenAI20240201Model(registry, "gpt-4-vision-preview", "1106")

	// GPT-4o models
	registerAzureOpenAIGPT4oModels(registry)

	// GPT-4o-mini models
	registerAzureOpenAI20240201Model(registry, "gpt-4o-mini", "2024-07-18")

	registerAzureOpenAIGPT41Models(registry)

	registerAzureOpenAIGPT5Models(registry)
}

// registerAzureOpenAIGPT4oModels registers all Azure OpenAI GPT-4o variant processors
func registerAzureOpenAIGPT4oModels(registry ProcessorRegistry) {
	registerAzureOpenAI20240201Model(registry, "gpt-4o", "2024-11-20")
	registerAzureOpenAI20240201Model(registry, "gpt-4o", "2024-08-06")
	registerAzureOpenAI20240201Model(registry, "gpt-4o", "2024-05-13")
	registerAzureOpenAI20240201Model(registry, "gpt-4o", "2024-02-01")
}

// registerAzureOpenAIGPT41Models registers all Azure OpenAI GPT-4.1 variant processors
func registerAzureOpenAIGPT41Models(registry ProcessorRegistry) {
	registerAzureOpenAI20240201Model(registry, "gpt-4.1", "2025-04-14")
	registerAzureOpenAI20240201Model(registry, "gpt-4.1-nano", "2025-04-14")
	registerAzureOpenAI20240201Model(registry, "gpt-4.1-mini", "2025-04-14")
}

// registerAzureOpenAIGPT5Models registers all Azure OpenAI GPT-5 variant processors
func registerAzureOpenAIGPT5Models(registry ProcessorRegistry) {
	// Base gpt-5 models
	registerAzureOpenAI20250807Model(registry, "gpt-5", "2025-08-07")
	// gpt-5-nano models
	registerAzureOpenAI20250807Model(registry, "gpt-5-nano", "2025-08-07")
	// gpt-5-mini models
	registerAzureOpenAI20250807Model(registry, "gpt-5-mini", "2025-08-07")
	// gpt-5-chat models
	registerAzureOpenAI20250807Model(registry, "gpt-5-chat", "2025-08-07")
	// Base gpt-5.1 models
	registerAzureOpenAI20250807Model(registry, "gpt-5.1", "2025-11-13")
	// Base gpt-5.2 models
	registerAzureOpenAI20250807Model(registry, "gpt-5.2", "2025-12-11")
}

// registerAzureOpenAIEmbeddingModels registers all Azure OpenAI embedding model processors
func registerAzureOpenAIEmbeddingModels(registry ProcessorRegistry) {
	registerAzureOpenAI20240201Model(registry, "text-embedding-ada-002", "2")
	registerAzureOpenAI20240201Model(registry, "text-embedding-3-large", "1")
	registerAzureOpenAI20240201Model(registry, "text-embedding-3-small", "1")
}

// registerAzureOpenAIImageModels registers all Azure OpenAI image generation model processors
func registerAzureOpenAIImageModels(registry ProcessorRegistry) {
	registerAzureOpenAI20240201Model(registry, "dall-e-3", "3.0")
	registerAzureOpenAI20240201Model(registry, "gpt-image-1.5", "2025-12-16")
}

// registerAzureOpenAIRealtimeModels registers all Azure OpenAI realtime model processors
func registerAzureOpenAIRealtimeModels(registry ProcessorRegistry) {
	registerAzureOpenAI20240201Model(registry, "gpt-realtime", "2025-08-28")
	registerAzureOpenAI20240201Model(registry, "gpt-realtime-mini", "2025-10-06")
	registerAzureOpenAI20240201Model(registry, "gpt-realtime-mini", "2025-12-15")
	registerAzureOpenAI20240201Model(registry, "gpt-realtime-1.5", "2026-02-23")
}

// registerBedrockModel is a helper to register a Bedrock model with a given creator and extension factory.
// This reduces boilerplate for models sharing the same provider/infrastructure.
func registerBedrockModel(registry ProcessorRegistry, creator types.Creator, modelID, version string, extensionFactory func() interface{}) {
	_ = registry.Register(ProcessorKey{
		Provider:       "bedrock",
		Infrastructure: "aws",
		Creator:        creator,
		ModelID:        modelID,
		Version:        version,
	}, extensionFactory)
}

// registerBedrockAmazonModel registers a Bedrock Amazon model using the 2023-06-01 API extension.
func registerBedrockAmazonModel(registry ProcessorRegistry, modelID, version string) {
	registerBedrockModel(registry, "amazon", modelID, version, func() interface{} {
		return extensions.NewBedrockAmazon20230601Extension()
	})
}

// registerBedrockAnthropicModel registers a Bedrock Anthropic model using the 2023-06-01 API extension.
func registerBedrockAnthropicModel(registry ProcessorRegistry, modelID, version string) {
	registerBedrockModel(registry, "anthropic", modelID, version, func() interface{} {
		return extensions.NewBedrockAnthropic20230601Extension()
	})
}

// registerBedrockMetaModel registers a Bedrock Meta model using the 2023-06-01 API extension.
func registerBedrockMetaModel(registry ProcessorRegistry, modelID, version string) {
	registerBedrockModel(registry, "meta", modelID, version, func() interface{} {
		return extensions.NewBedrockMeta20230601Extension()
	})
}

// registerBedrockMistralModel registers a Bedrock Mistral model using the Anthropic 2023-06-01 API extension.
func registerBedrockMistralModel(registry ProcessorRegistry, modelID, version string) {
	registerBedrockModel(registry, "mistral", modelID, version, func() interface{} {
		return extensions.NewBedrockAnthropic20230601Extension()
	})
}

// registerBedrockAmazonProcessors registers all Bedrock Amazon model processors
func registerBedrockAmazonProcessors(registry ProcessorRegistry) {
	registerBedrockAmazonModel(registry, "titan-embed-text", "v2")
	registerBedrockAmazonModel(registry, "nova-micro", "v1")
	registerBedrockAmazonModel(registry, "nova-lite-v1", "v1")
	registerBedrockAmazonModel(registry, "nova-pro", "v1")
	registerBedrockAmazonModel(registry, "nova-premier", "v1")
	registerBedrockAmazonModel(registry, "nova-2-lite-v1", "v1")
	registerBedrockAmazonModel(registry, "nova-2-pro-preview", "v1")
	registerBedrockAmazonModel(registry, "nova-2-omni-preview", "v1")
	registerBedrockAmazonModel(registry, "nova-2-multimodal-embeddings", "v1")
}

// registerBedrockAnthropicProcessors registers all Bedrock Anthropic model processors
func registerBedrockAnthropicProcessors(registry ProcessorRegistry) {
	registerBedrockAnthropicModel(registry, "claude-3-haiku", "v1")
	registerBedrockAnthropicModel(registry, "claude-3-7-sonnet", "v1")
	registerBedrockAnthropicModel(registry, "claude-3-5-haiku", "v1")
	registerBedrockAnthropicModel(registry, "claude-3-5-sonnet", "v2")
	registerBedrockAnthropicModel(registry, "claude-haiku-4-5", "1.0")
	registerBedrockAnthropicModel(registry, "claude-sonnet-4-5", "1.0")
	registerBedrockAnthropicModel(registry, "claude-sonnet-4-6", "v1")
	registerBedrockAnthropicModel(registry, "claude-opus-4-6", "v1")
}

// registerBedrockMetaProcessors registers all Bedrock Meta model processors
func registerBedrockMetaProcessors(registry ProcessorRegistry) {
	registerBedrockMetaModel(registry, "llama3-70b-instruct", "v1")
	registerBedrockMetaModel(registry, "llama3-8b-instruct", "v1")
	registerBedrockMetaModel(registry, "llama-3-2-90b-instruct", "v1:0")
}

// registerBedrockMistralProcessors registers all Bedrock Mistral model processors
func registerBedrockMistralProcessors(registry ProcessorRegistry) {
	registerBedrockMistralModel(registry, "mistral-large-3", "v1")
	registerBedrockMistralModel(registry, "ministral-14b", "v1")
	registerBedrockMistralModel(registry, "ministral-8b", "v1")
	registerBedrockMistralModel(registry, "ministral-3b", "v1")
}

// registerVertexGoogleModel is a helper to register a Vertex Google model with a given extension factory.
// This reduces boilerplate for models sharing the same provider/infrastructure/creator.
func registerVertexGoogleModel(registry ProcessorRegistry, modelID, version string, extensionFactory func() interface{}) {
	_ = registry.Register(ProcessorKey{
		Provider:       "vertex",
		Infrastructure: "gcp",
		Creator:        "google",
		ModelID:        modelID,
		Version:        version,
	}, extensionFactory)
}

// registerVertexGoogleOpenAIModel registers a Vertex Google model that uses the OpenAI-compatible API.
func registerVertexGoogleOpenAIModel(registry ProcessorRegistry, modelID, version string) {
	registerVertexGoogleModel(registry, modelID, version, func() interface{} {
		return extensions.NewVertexGoogleOpenAIExtension()
	})
}

// registerVertexGeminiGenerateContentModel registers a Vertex Google model that uses the Gemini generateContent API.
func registerVertexGeminiGenerateContentModel(registry ProcessorRegistry, modelID, version string) {
	registerVertexGoogleModel(registry, modelID, version, func() interface{} {
		return extensions.NewVertexGoogle20240101Extension()
	})
}

// registerVertexGoogleProcessors registers all Vertex Google model processors
func registerVertexGoogleProcessors(registry ProcessorRegistry) {
	// NOTE:
	// We are using OpenAI API provided by Vertex Google, not the native Vertex AI API.
	// See https://ai.google.dev/gemini-api/docs/openai for details.

	// Vertex Google Gemini models
	registerVertexGoogleOpenAIModel(registry, "gemini-1.0-pro", "v1")
	registerVertexGoogleOpenAIModel(registry, "gemini-1.5-pro", "002")
	registerVertexGoogleOpenAIModel(registry, "gemini-1.5-flash", "002")
	registerVertexGoogleOpenAIModel(registry, "gemini-2.0-flash", "001")
	registerVertexGoogleOpenAIModel(registry, "gemini-2.5-flash", "001")
	registerVertexGoogleOpenAIModel(registry, "gemini-2.5-flash-lite", "001")
	registerVertexGoogleOpenAIModel(registry, "gemini-2.5-pro", "001")
	registerVertexGoogleOpenAIModel(registry, "gemini-3.0-flash-preview", "3.0-flash-preview")
	registerVertexGoogleOpenAIModel(registry, "gemini-3.0-pro-preview", "3.0-pro-preview")

	// Vertex Gemini Image Models
	registerVertexGeminiGenerateContentModel(registry, "gemini-2.5-flash-image", "001")
	registerVertexGeminiGenerateContentModel(registry, "gemini-3.1-flash-image-preview", "3.1-flash-image-preview")

	// Vertex Google Embedding models
	registerVertexGoogleOpenAIModel(registry, "text-multilingual-embedding", "002")
	registerVertexGoogleOpenAIModel(registry, "gemini-embedding-001", "001")

	// Vertex Google Imagen 3 models
	registerVertexGoogleOpenAIModel(registry, "imagen-3.0", "generate-001")
	registerVertexGoogleOpenAIModel(registry, "imagen-3.0", "generate-002")
	registerVertexGoogleOpenAIModel(registry, "imagen-3.0-fast", "generate-001")

	// Vertex Google Imagen 4 models
	registerVertexGoogleOpenAIModel(registry, "imagen-4.0", "generate-001")
	registerVertexGoogleOpenAIModel(registry, "imagen-4.0-fast", "fast-generate-001")
	registerVertexGoogleOpenAIModel(registry, "imagen-4.0-ultra", "ultra-generate-001")
}
