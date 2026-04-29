/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"text/template"
)

//go:embed templates/chat-completion.json.tmpl
var chatCompletionTemplateStr string

//go:embed templates/chat-completion-google.json.tmpl
var chatCompletionGoogleTemplateStr string

//go:embed templates/converse.json.tmpl
var converseTemplateStr string

//go:embed templates/embeddings-openai.json.tmpl
var embeddingsOpenAITemplateStr string

//go:embed templates/embeddings-amazon.json.tmpl
var embeddingsAmazonTemplateStr string

//go:embed templates/embeddings-google.json.tmpl
var embeddingsGoogleTemplateStr string

//go:embed templates/embeddings-amazon-nova.json.tmpl
var embeddingsAmazonNovaTemplateStr string

//go:embed templates/chat-completion-stream.json.tmpl
var chatCompletionStreamTemplateStr string

//go:embed templates/chat-completion-google-stream.json.tmpl
var chatCompletionGoogleStreamTemplateStr string

//go:embed templates/image-generation-google.json.tmpl
var imageGenerationGoogleTemplateStr string

//go:embed templates/image-generation-google-imagen.json.tmpl
var imageGenerationGoogleImagenTemplateStr string

//go:embed templates/image-generation-openai.json.tmpl
var imageGenerationOpenAITemplateStr string

//go:embed templates/image-generation-openai-gpt-image.json.tmpl
var imageGenerationOpenAIGPTImageTemplateStr string

var chatCompletionTemplate = template.Must(
	template.New("chat-completion.json").Parse(chatCompletionTemplateStr),
)

var chatCompletionGoogleTemplate = template.Must(
	template.New("chat-completion-google.json").Parse(chatCompletionGoogleTemplateStr),
)

var converseTemplate = template.Must(
	template.New("converse.json").Parse(converseTemplateStr),
)

var embeddingsOpenAITemplate = template.Must(
	template.New("embeddings-openai.json").Parse(embeddingsOpenAITemplateStr),
)

var embeddingsAmazonTemplate = template.Must(
	template.New("embeddings-amazon.json").Parse(embeddingsAmazonTemplateStr),
)

var embeddingsGoogleTemplate = template.Must(
	template.New("embeddings-google.json").Parse(embeddingsGoogleTemplateStr),
)

var embeddingsAmazonNovaTemplate = template.Must(
	template.New("embeddings-amazon-nova.json").Parse(embeddingsAmazonNovaTemplateStr),
)

var chatCompletionStreamTemplate = template.Must(
	template.New("chat-completion-stream.json").Parse(chatCompletionStreamTemplateStr),
)

var chatCompletionGoogleStreamTemplate = template.Must(
	template.New("chat-completion-google-stream.json").Parse(chatCompletionGoogleStreamTemplateStr),
)

var imageGenerationGoogleTemplate = template.Must(
	template.New("image-generation-google.json").Parse(imageGenerationGoogleTemplateStr),
)

var imageGenerationGoogleImagenTemplate = template.Must(
	template.New("image-generation-google-imagen.json").Parse(imageGenerationGoogleImagenTemplateStr),
)

var imageGenerationOpenAITemplate = template.Must(
	template.New("image-generation-openai.json").Parse(imageGenerationOpenAITemplateStr),
)

var imageGenerationOpenAIGPTImageTemplate = template.Must(
	template.New("image-generation-openai-gpt-image.json").Parse(imageGenerationOpenAIGPTImageTemplateStr),
)

// chatCompletionData is the data passed to the chat completion templates.
type chatCompletionData struct {
	Model        string // JSON-encoded model string (Google only)
	SystemPrompt string // JSON-encoded system prompt
	UserPrompt   string // JSON-encoded user prompt
}

// readAndEncodeFile reads a file and JSON-encodes its content as a string.
// Returns the JSON-encoded string (including surrounding quotes) ready for template injection.
func readAndEncodeFile(t *testing.T, description, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read %s %s: %v", description, path, err)
	}
	encoded, err := json.Marshal(string(data))
	if err != nil {
		t.Fatalf("Failed to JSON-encode %s: %v", description, err)
	}
	return string(encoded)
}

// renderAndValidate executes a template with the given data, validates the result as JSON,
// and returns the rendered bytes.
func renderAndValidate(t *testing.T, tmpl *template.Template, data interface{}) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("Failed to render template %q: %v", tmpl.Name(), err)
	}
	var check map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Rendered payload for template %q is not valid JSON: %v\nPayload: %s",
			tmpl.Name(), err, buf.String())
	}
	return buf.Bytes()
}

// BuildChatCompletionPayload reads system and user prompt files, renders them
// into a chat completion JSON payload using the appropriate template for the
// given target, and returns the resulting bytes.
// For Google targets, the "model" field is automatically included with "google/{model}".
// For other providers, the generic template (without model) is used.
func BuildChatCompletionPayload(t *testing.T, target ModelTarget, systemPromptPath, userPromptPath string) []byte {
	t.Helper()
	return buildChatPayload(t, target, systemPromptPath, userPromptPath, false)
}

// BuildStreamingChatCompletionPayload reads system and user prompt files, renders them
// into a streaming chat completion JSON payload using the appropriate template for the
// given target, and returns the resulting bytes.
// For OpenAI/Google targets, the payload includes "stream": true.
// For Anthropic/Amazon targets, the converse-stream URL is used with the same body as non-streaming.
func BuildStreamingChatCompletionPayload(t *testing.T, target ModelTarget, systemPromptPath, userPromptPath string) []byte {
	t.Helper()
	return buildChatPayload(t, target, systemPromptPath, userPromptPath, true)
}

// buildChatPayload is the shared implementation for BuildChatCompletionPayload and
// BuildStreamingChatCompletionPayload. The streaming parameter controls which template
// is selected for OpenAI/Google providers.
func buildChatPayload(t *testing.T, target ModelTarget, systemPromptPath, userPromptPath string, streaming bool) []byte {
	t.Helper()

	data := chatCompletionData{
		SystemPrompt: readAndEncodeFile(t, "system prompt", systemPromptPath),
		UserPrompt:   readAndEncodeFile(t, "user prompt", userPromptPath),
	}

	var tmpl *template.Template
	switch target.Provider {
	case ProviderGoogle:
		modelJSON, err := json.Marshal(fmt.Sprintf("google/%s", target.Model))
		if err != nil {
			t.Fatalf("Failed to JSON-encode model: %v", err)
		}
		data.Model = string(modelJSON)
		if streaming {
			tmpl = chatCompletionGoogleStreamTemplate
		} else {
			tmpl = chatCompletionGoogleTemplate
		}
	case ProviderAnthropic, ProviderAmazon, ProviderMeta:
		// Streaming uses a different URL (converse-stream) but the same request body.
		tmpl = converseTemplate
	default:
		if streaming {
			tmpl = chatCompletionStreamTemplate
		} else {
			tmpl = chatCompletionTemplate
		}
	}

	return renderAndValidate(t, tmpl, data)
}

// embeddingData is the data passed to the embeddings template.
type embeddingData struct {
	Model string // JSON-encoded model string (Google only)
	Input string // JSON-encoded input string
}

// BuildEmbeddingPayload reads an input file, renders it into an embeddings
// JSON payload using the appropriate template for the given target, and returns the resulting bytes.
// For Google targets, the "model" field is automatically included with "google/{model}".
// For Amazon Nova models, the "inputs" array format is used.
// For Amazon Titan models, the "inputText" field is used.
// For other providers (OpenAI), the "input" field is used.
func BuildEmbeddingPayload(t *testing.T, target ModelTarget, inputPath string) []byte {
	t.Helper()

	data := embeddingData{
		Input: readAndEncodeFile(t, "embedding input", inputPath),
	}

	// Select template based on provider and model
	var tmpl *template.Template
	switch {
	case target.Provider == ProviderGoogle:
		modelJSON, err := json.Marshal(target.Model)
		if err != nil {
			t.Fatalf("Failed to JSON-encode model: %v", err)
		}
		data.Model = string(modelJSON)
		tmpl = embeddingsGoogleTemplate
	case target.Provider == ProviderAmazon && isNovaModel(target.Model):
		tmpl = embeddingsAmazonNovaTemplate
	case target.Provider == ProviderAmazon:
		tmpl = embeddingsAmazonTemplate
	default:
		tmpl = embeddingsOpenAITemplate
	}

	return renderAndValidate(t, tmpl, data)
}

// isNovaModel returns true if the model name indicates an Amazon Nova model.
func isNovaModel(model string) bool {
	return strings.HasPrefix(model, "nova") || strings.HasPrefix(model, "Nova")
}

// getFullImagenModelID maps short Imagen deployment names to full model IDs required by Vertex AI.
// These mappings come from the Helm configuration modelId fields.
func getFullImagenModelID(shortName string) string {
	imagenModelIDs := map[string]string{
		"imagen-3":              "imagen-3.0-generate-002",
		"imagen-3-next":         "imagen-3.0-generate-002",
		"imagen-3-deprecated":   "imagen-3.0-generate-001",
		"imagen-3-fast":         "imagen-3.0-fast-generate-001",
		"imagen-4.0":            "imagen-4.0-generate-001",
		"imagen-4.0-next":       "imagen-4.0-generate-001",
		"imagen-4.0-deprecated": "imagen-4.0-generate-001",
		"imagen-4.0-fast":       "imagen-4.0-fast-generate-001",
		"imagen-4.0-ultra":      "imagen-4.0-ultra-generate-001",
	}

	if fullID, ok := imagenModelIDs[shortName]; ok {
		return fullID
	}
	// If not found in mapping, return the original name
	return shortName
}

// imageGenerationData is the data passed to the image generation templates.
type imageGenerationData struct {
	Model      string // JSON-encoded model name (Gemini only)
	ModelID    string // JSON-encoded model ID (Imagen only)
	UserPrompt string // JSON-encoded user prompt
}

// BuildImageGenerationPayload reads a user prompt file, renders it into an image generation
// JSON payload using the appropriate template for the given target, and returns the resulting bytes.
// For Google targets, the template is selected based on the endpoint:
//   - /images/generations: Imagen API format (modelId + payload)
//   - /generateContent: Gemini native format (contents/parts)
func BuildImageGenerationPayload(t *testing.T, target ModelTarget, userPromptPath string) []byte {
	t.Helper()

	data := imageGenerationData{
		UserPrompt: readAndEncodeFile(t, "user prompt", userPromptPath),
	}

	var tmpl *template.Template
	switch target.Provider {
	case ProviderGoogle:
		// Detect API format from endpoint
		if strings.Contains(target.Endpoint, "/images/generations") {
			// Imagen API format - needs full modelId (e.g., imagen-3.0-generate-002)
			fullModelID := getFullImagenModelID(target.Model)
			modelIDJSON, err := json.Marshal(fullModelID)
			if err != nil {
				t.Fatalf("Failed to JSON-encode model ID: %v", err)
			}
			data.ModelID = string(modelIDJSON)
			tmpl = imageGenerationGoogleImagenTemplate
		} else {
			// Gemini GenerateContent API format - needs model name with google/ prefix
			modelName := fmt.Sprintf("google/%s", target.Model)
			modelJSON, err := json.Marshal(modelName)
			if err != nil {
				t.Fatalf("Failed to JSON-encode model name: %v", err)
			}
			data.Model = string(modelJSON)
			tmpl = imageGenerationGoogleTemplate
		}
	case ProviderOpenAI:
		if strings.HasPrefix(target.Model, "gpt-image") {
			tmpl = imageGenerationOpenAIGPTImageTemplate
		} else {
			tmpl = imageGenerationOpenAITemplate
		}
	default:
		t.Fatalf("Image generation not supported for provider: %s", target.Provider)
	}

	return renderAndValidate(t, tmpl, data)
}
