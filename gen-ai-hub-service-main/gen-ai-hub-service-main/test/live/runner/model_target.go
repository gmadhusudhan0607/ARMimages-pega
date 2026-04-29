/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Provider represents an LLM provider name.
type Provider string

// Provider constants for supported LLM providers.
const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGoogle    Provider = "google"
	ProviderAmazon    Provider = "amazon"
	ProviderMeta      Provider = "meta"
	ProviderMistral   Provider = "mistral"
)

// Model type constants for filtering /models responses.
const (
	ModelTypeChatCompletion  = "chat_completion"
	ModelTypeEmbedding       = "embedding"
	ModelTypeImageGeneration = "image"
	ModelTypeRealtime        = "realtime"
)

// ModelTarget identifies a model by provider and deployment name,
// matching the URL pattern: /{provider}/deployments/{model}/...
type ModelTarget struct {
	Provider  Provider
	Model     string
	Endpoint  string // Endpoint suffix extracted from model_path (e.g., "/generateContent", "/images/generations")
	Lifecycle string // Lifecycle status from /models response (e.g., "Generally Available", "Deprecated", "Preview")
}

// String returns "provider/model" for use in test descriptions and logs.
func (mt ModelTarget) String() string {
	return fmt.Sprintf("%s/%s", mt.Provider, mt.Model)
}

// IsDeprecated reports whether the model has a "Deprecated" lifecycle status.
func (mt ModelTarget) IsDeprecated() bool {
	return strings.EqualFold(mt.Lifecycle, "Deprecated")
}

// ChatCompletionsPath returns the URL path for chat completions.
func (mt ModelTarget) ChatCompletionsPath() string {
	return fmt.Sprintf("/%s/deployments/%s/chat/completions", mt.Provider, mt.Model)
}

// EmbeddingsPath returns the URL path for embeddings.
// Amazon embedding models use /invoke endpoint, while OpenAI/Azure use /embeddings.
// Note: api-version is accepted for backward compatibility but is always overridden
// by the service to the governed version (2024-10-21).
func (mt ModelTarget) EmbeddingsPath() string {
	if mt.Provider == ProviderAmazon {
		return fmt.Sprintf("/%s/deployments/%s/invoke", mt.Provider, mt.Model)
	}
	return fmt.Sprintf("/%s/deployments/%s/embeddings?api-version=2024-10-21", mt.Provider, mt.Model)
}

// ImageGenerationPath returns the URL path for image generation.
// Uses the Endpoint field extracted from the model_path during discovery.
// Examples:
//   - Gemini: /google/deployments/gemini-3.1-flash-image-preview/generateContent
//   - Imagen: /google/deployments/imagen-3/images/generations
func (mt ModelTarget) ImageGenerationPath() string {
	return fmt.Sprintf("/%s/deployments/%s%s", mt.Provider, mt.Model, mt.Endpoint)
}

// ConversePath returns the URL path for the Bedrock converse endpoint.
func (mt ModelTarget) ConversePath() string {
	return fmt.Sprintf("/%s/deployments/%s/converse", mt.Provider, mt.Model)
}

// ConverseStreamPath returns the URL path for the Bedrock converse-stream endpoint.
func (mt ModelTarget) ConverseStreamPath() string {
	return fmt.Sprintf("/%s/deployments/%s/converse-stream", mt.Provider, mt.Model)
}

// isConverseProvider reports whether the target uses the Bedrock converse API
// (Anthropic and Amazon), as opposed to the OpenAI-compatible chat completions API.
func (mt ModelTarget) isConverseProvider() bool {
	return mt.Provider == ProviderAnthropic || mt.Provider == ProviderAmazon || mt.Provider == ProviderMeta || mt.Provider == ProviderMistral
}

// RequestPath returns the provider-appropriate URL path.
// Anthropic and Amazon targets use the converse endpoint; all others use chat completions.
func (mt ModelTarget) RequestPath() string {
	if mt.isConverseProvider() {
		return mt.ConversePath()
	}
	return mt.ChatCompletionsPath()
}

// StreamingRequestPath returns the provider-appropriate URL path for streaming.
// Anthropic and Amazon targets use the converse-stream endpoint; all others use chat completions
// (streaming is controlled via the "stream": true field in the request body).
func (mt ModelTarget) StreamingRequestPath() string {
	if mt.isConverseProvider() {
		return mt.ConverseStreamPath()
	}
	return mt.ChatCompletionsPath()
}

// modelInfoDTO is a minimal struct for parsing the /models JSON response.
// Only the fields needed for target discovery are included.
type modelInfoDTO struct {
	Type      string   `json:"type"`
	ModelPath []string `json:"model_path"`
	Lifecycle string   `json:"lifecycle"`
}

// providerFromPathSegment maps the first path segment to a Provider constant.
func providerFromPathSegment(segment string) (Provider, bool) {
	switch segment {
	case "openai":
		return ProviderOpenAI, true
	case "anthropic":
		return ProviderAnthropic, true
	case "google":
		return ProviderGoogle, true
	case "amazon":
		return ProviderAmazon, true
	case "meta":
		return ProviderMeta, true
	case "mistral":
		return ProviderMistral, true
	default:
		return "", false
	}
}

// parseTargetFromPath extracts Provider, Model, and Endpoint from a model_path entry.
// Expected format: /{provider}/deployments/{model}/{endpoint}
// For example: /google/deployments/gemini-3.1-flash-image-preview/generateContent
// or: /google/deployments/imagen-3/images/generations
func parseTargetFromPath(path string) (ModelTarget, bool) {
	// Trim leading slash and split: ["provider", "deployments", "model", "endpoint", ...]
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 || parts[1] != "deployments" {
		return ModelTarget{}, false
	}
	provider, ok := providerFromPathSegment(parts[0])
	if !ok {
		return ModelTarget{}, false
	}

	// Extract endpoint suffix (everything after model name)
	endpoint := ""
	if len(parts) > 3 {
		endpoint = "/" + strings.Join(parts[3:], "/")
	}

	return ModelTarget{Provider: provider, Model: parts[2], Endpoint: endpoint}, true
}

// FetchAllModels calls GET /models once and returns the raw parsed model list.
// Callers can then use FilterByType to extract targets for specific model types
// without making additional HTTP requests.
func FetchAllModels(baseURL, token string) ([]modelInfoDTO, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create /models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /models request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read /models response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /models returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response — can be a bare array or {"models": [...]}
	var models []modelInfoDTO
	if err := json.Unmarshal(body, &models); err != nil {
		var wrapped struct {
			Models []modelInfoDTO `json:"models"`
		}
		if err2 := json.Unmarshal(body, &wrapped); err2 != nil {
			return nil, fmt.Errorf("failed to parse /models response: %w", err2)
		}
		models = wrapped.Models
	}

	return models, nil
}

// FilterByType filters an already-fetched model list by type and returns a deduplicated
// sorted list of ModelTarget values. Use with FetchAllModels to avoid repeated HTTP calls.
func FilterByType(models []modelInfoDTO, modelType string) []ModelTarget {
	skipDeprecated := os.Getenv("INCLUDE_DEPRECATED_MODELS") != "true"
	seen := make(map[string]bool)
	var targets []ModelTarget
	var deprecatedSkipped []string
	for _, m := range models {
		if m.Type != modelType {
			continue
		}
		for _, path := range m.ModelPath {
			target, ok := parseTargetFromPath(path)
			if !ok {
				continue
			}
			target.Lifecycle = m.Lifecycle
			key := target.String()
			if seen[key] {
				continue
			}
			seen[key] = true

			if skipDeprecated && target.IsDeprecated() {
				deprecatedSkipped = append(deprecatedSkipped, key)
				continue
			}
			targets = append(targets, target)
		}
	}
	if len(deprecatedSkipped) > 0 {
		logVerbosef("  Skipped %d deprecated %s model(s): %s\n",
			len(deprecatedSkipped), modelType, strings.Join(deprecatedSkipped, ", "))
		logVerbose("  Set INCLUDE_DEPRECATED_MODELS=true to include them")
	}

	// Sort for deterministic ordering: by provider then model
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Provider != targets[j].Provider {
			return targets[i].Provider < targets[j].Provider
		}
		return targets[i].Model < targets[j].Model
	})

	return targets
}

// fetchTargetsByType calls GET /models on the running service, filters models by the given
// modelType (e.g. ModelTypeChatCompletion or ModelTypeEmbedding), and returns a deduplicated
// sorted list of ModelTarget values. This makes a fresh HTTP call each time — prefer
// FetchAllModels + FilterByType when discovering multiple model types.
func fetchTargetsByType(baseURL, token, modelType string) ([]ModelTarget, error) {
	models, err := FetchAllModels(baseURL, token)
	if err != nil {
		return nil, err
	}
	return FilterByType(models, modelType), nil
}

// FetchChatCompletionTargets calls GET /models on the running service,
// filters models with "type": "chat_completion", and returns a deduplicated
// sorted list of ModelTarget values.
func FetchChatCompletionTargets(baseURL, token string) ([]ModelTarget, error) {
	return fetchTargetsByType(baseURL, token, ModelTypeChatCompletion)
}

// FetchEmbeddingTargets calls GET /models on the running service,
// filters models with "type": "embedding", and returns a deduplicated
// sorted list of ModelTarget values.
func FetchEmbeddingTargets(baseURL, token string) ([]ModelTarget, error) {
	return fetchTargetsByType(baseURL, token, ModelTypeEmbedding)
}

// FetchImageGenerationTargets calls GET /models on the running service,
// filters models with "type": "image", and returns a deduplicated
// sorted list of ModelTarget values.
func FetchImageGenerationTargets(baseURL, token string) ([]ModelTarget, error) {
	return fetchTargetsByType(baseURL, token, ModelTypeImageGeneration)
}

// FetchRealtimeTargets calls GET /models on the running service,
// filters models with "type": "realtime", and returns a deduplicated
// sorted list of ModelTarget values.
func FetchRealtimeTargets(baseURL, token string) ([]ModelTarget, error) {
	return fetchTargetsByType(baseURL, token, ModelTypeRealtime)
}
