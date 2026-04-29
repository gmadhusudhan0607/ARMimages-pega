//go:build ignore
// +build ignore

/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 *
 * Cross-platform script to add new GCP Vertex AI models to the gen-ai-hub-service
 * Usage:
 *   Single model:  go run scripts/add-vertex-model.go -model-name gemini-2.5-flash-lite -model-type chat_completion [-template gemini-2.5-flash]
 *   Batch mode:    go run scripts/add-vertex-model.go -config models.json
 *   Dry run:       go run scripts/add-vertex-model.go -model-name gemini-2.5-flash-lite -model-type chat_completion -dry-run
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ANSI color codes (disabled on Windows cmd)
var (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

// File paths
const (
	SpecYAML              = "apidocs/spec.yaml"
	MainTestGo            = "cmd/service/main_test.go"
	ModelConfigDir        = "distribution/genai-hub-service-helm/src/main/helm/configuration/models"
	ModelMetadataYAML     = "distribution/genai-hub-service-helm/src/main/helm/templates/model-metadata.yaml"
	IntegrationMappingsGo = "test/integration/service/mappings_test.go"
	SpecsBaseDir          = "internal/models/specs/gcp/vertex"
	RegistryGo            = "internal/request/processors/registry/registry.go"
)

// ModelConfig represents the configuration for a single model
type ModelConfig struct {
	ModelName string `json:"model_name"`
	ModelType string `json:"model_type"`
	Template  string `json:"template,omitempty"`
	Preview   bool   `json:"preview,omitempty"`
}

// BatchConfig represents the JSON configuration file format
type BatchConfig struct {
	Models []ModelConfig `json:"models"`
}

// ParsedModel contains parsed model information
type ParsedModel struct {
	ModelName      string
	ModelType      string
	ModelLabel     string
	Version        string
	Template       string
	ConfigFileName string
	Preview        bool
}

// Script state
var (
	dryRun        bool
	modelsAdded   int
	modelsSkipped int
)

func init() {
	// Disable colors on Windows unless TERM is set
	if os.Getenv("TERM") == "" && os.Getenv("WT_SESSION") == "" {
		if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
			colorReset = ""
			colorRed = ""
			colorGreen = ""
			colorYellow = ""
			colorBlue = ""
		}
	}
}

func printInfo(msg string) {
	fmt.Printf("%sℹ️  %s%s\n", colorBlue, msg, colorReset)
}

func printSuccess(msg string) {
	fmt.Printf("%s✅ %s%s\n", colorGreen, msg, colorReset)
}

func printWarning(msg string) {
	fmt.Printf("%s⚠️  %s%s\n", colorYellow, msg, colorReset)
}

func printError(msg string) {
	fmt.Printf("%s❌ %s%s\n", colorRed, msg, colorReset)
}

func printHeader(msg string) {
	fmt.Println()
	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorBlue, colorReset)
	fmt.Printf("%s  %s%s\n", colorBlue, msg, colorReset)
	fmt.Printf("%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorBlue, colorReset)
}

// parseModelName parses the model name and extracts components
func parseModelName(modelName, modelType string) (*ParsedModel, error) {
	if modelName == "" {
		return nil, fmt.Errorf("model_name is required")
	}

	validTypes := []string{"chat_completion", "embedding", "image"}
	validType := false
	for _, t := range validTypes {
		if modelType == t {
			validType = true
			break
		}
	}
	if !validType {
		return nil, fmt.Errorf("invalid model_type: %s (must be one of: %s)", modelType, strings.Join(validTypes, ", "))
	}

	// Generate model label (Title Case With Spaces, handling dots and hyphens)
	modelLabel := toModelLabel(modelName)

	// Extract version from model name (e.g., "2.5" from "gemini-2.5-flash")
	version := extractVersion(modelName)

	// Generate config file name (replace dots with hyphens)
	configFileName := strings.ReplaceAll(modelName, ".", "-") + ".yaml"

	return &ParsedModel{
		ModelName:      modelName,
		ModelType:      modelType,
		ModelLabel:     modelLabel,
		Version:        version,
		ConfigFileName: configFileName,
	}, nil
}

// toModelLabel converts model name to human-readable label
// e.g., "gemini-2.5-flash-lite" -> "Gemini 2.5 Flash-Lite"
func toModelLabel(s string) string {
	// Split by hyphens but preserve version numbers
	words := strings.Split(s, "-")
	result := []string{}

	for _, word := range words {
		// Check if it's a version number (like "2.5" or "1.5")
		if matched, _ := regexp.MatchString(`^\d+(\.\d+)?$`, word); matched {
			result = append(result, word)
		} else if len(word) > 0 {
			// Title case the word
			result = append(result, strings.ToUpper(string(word[0]))+strings.ToLower(word[1:]))
		}
	}

	return strings.Join(result, " ")
}

// extractVersion extracts version from model name
func extractVersion(modelName string) string {
	// Try to find version pattern like "2.5", "1.5", "2.0"
	re := regexp.MustCompile(`(\d+\.\d+)`)
	matches := re.FindStringSubmatch(modelName)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to find single digit version
	re = regexp.MustCompile(`-(\d+)(?:-|$)`)
	matches = re.FindStringSubmatch(modelName)
	if len(matches) > 1 {
		return matches[1]
	}

	return "1"
}

// fileContains checks if a file contains a specific string
func fileContains(filePath, search string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), search)
}

// checkSpecYAML checks if model exists in spec.yaml
func checkSpecYAML(modelName string) bool {
	return fileContains(SpecYAML, fmt.Sprintf("- %s", modelName))
}

// checkMainTestGo checks if model exists in main_test.go
func checkMainTestGo(modelName string) bool {
	return fileContains(MainTestGo, fmt.Sprintf(`/google/deployments/%s/`, modelName))
}

// checkModelConfig checks if model config file exists
func checkModelConfig(configFileName string) bool {
	_, err := os.Stat(fmt.Sprintf("%s/%s", ModelConfigDir, configFileName))
	return err == nil
}

// checkModelMetadata checks if model exists in model-metadata.yaml
func checkModelMetadata(modelName string) bool {
	return fileContains(ModelMetadataYAML, fmt.Sprintf("    %s:", modelName))
}

// checkIntegrationTests checks if model exists in integration tests
func checkIntegrationTests(modelName string) bool {
	return fileContains(IntegrationMappingsGo, fmt.Sprintf(`"%s"`, modelName))
}

// getSpecFile determines the spec file path based on model name
func getSpecFile(modelName string) string {
	// Extract version family from model name (e.g., "2.5" from "gemini-2.5-flash")
	re := regexp.MustCompile(`(\d+\.\d+)`)
	matches := re.FindStringSubmatch(modelName)
	if len(matches) > 1 {
		versionFamily := matches[1]
		return fmt.Sprintf("%s/google/gemini-%s.yaml", SpecsBaseDir, versionFamily)
	}
	// Default to gemini-2.5.yaml for unknown versions
	return fmt.Sprintf("%s/google/gemini-2.5.yaml", SpecsBaseDir)
}

// checkModelSpecYAML checks if model exists in the spec YAML file
func checkModelSpecYAML(modelName string) bool {
	specFile := getSpecFile(modelName)
	return fileContains(specFile, fmt.Sprintf("  - name: %s", modelName))
}

// checkRegistryGo checks if model exists in registry.go
func checkRegistryGo(modelName string) bool {
	content, err := os.ReadFile(RegistryGo)
	if err != nil {
		return false
	}
	// Use regex to match ModelID with the model name (handles various whitespace)
	pattern := fmt.Sprintf(`ModelID:\s*"%s"`, regexp.QuoteMeta(modelName))
	matched, _ := regexp.MatchString(pattern, string(content))
	return matched
}

// addToModelSpecYAML adds model to the internal specs YAML file
func addToModelSpecYAML(model *ParsedModel) error {
	specFile := getSpecFile(model.ModelName)

	if dryRun {
		templateInfo := "none"
		if model.Template != "" {
			templateInfo = model.Template
		}
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' to %s (template: %s)", model.ModelName, specFile, templateInfo))
		return nil
	}

	// Check if spec file exists
	if _, err := os.Stat(specFile); os.IsNotExist(err) {
		printWarning(fmt.Sprintf("Spec file does not exist: %s. Creating new file.", specFile))

		if err := os.MkdirAll(getSpecFile(model.ModelName)[:strings.LastIndex(getSpecFile(model.ModelName), "/")], 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		header := `infrastructure: gcp
provider: vertex
creator: google
metadata:
  description: "Google Gemini models on GCP Vertex AI"
  documentation: "https://cloud.google.com/vertex-ai/docs/generative-ai/model-reference/gemini"
models:
`

		if err := os.WriteFile(specFile, []byte(header), 0644); err != nil {
			return fmt.Errorf("failed to create spec file: %w", err)
		}
	}

	var yamlBlock string

	if model.Template != "" && fileContains(specFile, fmt.Sprintf("  - name: %s", model.Template)) {
		printInfo(fmt.Sprintf("Using template '%s' for model spec", model.Template))

		// Read file and extract template block
		content, err := os.ReadFile(specFile)
		if err != nil {
			return fmt.Errorf("failed to read spec file: %w", err)
		}

		// Extract template block - match from "  - name: template" to next "  - name:" or end
		pattern := fmt.Sprintf(`(?s)(  - name: %s\n.*?)(?:  - name:|\z)`, regexp.QuoteMeta(model.Template))
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(string(content))

		if len(matches) > 1 {
			templateBlock := matches[1]
			// Replace identifiers
			yamlBlock = strings.Replace(templateBlock, fmt.Sprintf("  - name: %s", model.Template), fmt.Sprintf("  - name: %s", model.ModelName), 1)
			yamlBlock = regexp.MustCompile(fmt.Sprintf(`label: [^\n]+`)).ReplaceAllString(yamlBlock, fmt.Sprintf("label: %s", model.ModelLabel))
			// Remove trailing whitespace
			yamlBlock = strings.TrimRight(yamlBlock, " \t\n") + "\n"
		}
	}

	if yamlBlock == "" {
		// Create scaffold YAML block with proper structure
		yamlBlock = generateSpecScaffold(model)
	}

	// Append to file
	f, err := os.OpenFile(specFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open spec file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(yamlBlock); err != nil {
		return fmt.Errorf("failed to write to spec file: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' to %s", model.ModelName, specFile))
	if model.Template == "" {
		printWarning("Review and update TODO markers in spec file")
	}
	return nil
}

// generateSpecScaffold creates a scaffold spec block with TODO markers
func generateSpecScaffold(model *ParsedModel) string {
	// Determine version code (e.g., "001" for gemini models)
	versionCode := "001"

	switch model.ModelType {
	case "embedding":
		return fmt.Sprintf(`  - name: %s
    version: '%s'
    label: %s
    functionalCapabilities: ["embedding"]
    endpoints:
      - path: /embedContent
    capabilities:
      inputModalities: [text]
      outputModalities: [embedding]
    parameters:
      maxInputTokens:
        title: Max Input Tokens
        description: Maximum number of tokens for input
        type: integer
        maximum: 2048  # TODO: Set correct maximum
        required: false
`, model.ModelName, versionCode, model.ModelLabel)

	case "image":
		return fmt.Sprintf(`  - name: %s
    version: '%s'
    label: %s
    functionalCapabilities: ["image"]
    endpoints:
      - path: /predict
    capabilities:
      inputModalities: [text]
      outputModalities: [image]
    parameters:
      aspectRatio:
        title: Aspect Ratio
        description: The aspect ratio for the generated output image
        type: string
        default: null
        required: false
`, model.ModelName, versionCode, model.ModelLabel)

	default: // chat_completion
		return fmt.Sprintf(`  - name: %s
    version: '%s'
    label: %s
    functionalCapabilities: ["chat_completion"]
    endpoints:
      - path: /generateContent
      - path: :streamGenerateContent
    capabilities:
      features: [streaming, functionCalling, jsonMode, structuredOutput, promptCaching]  # TODO: Verify features
      inputModalities: [text, code, image, audio, video]  # TODO: Verify modalities
      outputModalities: [text]
      mimeTypes:
        - application/pdf
        - text/csv
        - text/plain
        - text/markdown
        - text/html
        - text/xml
        - audio/wav
        - audio/mp3
        - audio/aiff
        - audio/aac
        - audio/ogg
        - audio/flac
        - image/png
        - image/jpeg
        - image/webp
        - image/heic
        - image/heif
        - video/mp4
        - video/mpeg
        - video/mov
        - video/avi
        - video/x-flv
        - video/mpg
        - video/webm
        - video/wmv
        - video/3gpp
    parameters:
      maxInputTokens:
        title: Max Input Tokens
        description: Maximum number of tokens (words or subwords) for input
        type: integer
        maximum: 1048576  # TODO: Set correct maximum
        required: false
      maxOutputTokens:
        title: Max Output Tokens
        description: Maximum number of tokens to generate
        type: integer
        default: null
        maximum: 65536  # TODO: Set correct maximum
        required: false
      temperature:
        title: Temperature
        description: Controls randomness of the generated output
        type: float
        default: 1.0
        maximum: 2.0
        minimum: 0.0
        required: false
      topP:
        title: Top P
        description: Nucleus sampling parameter controlling diversity of the output
        type: float
        default: 0.95
        minimum: 0.0
        maximum: 1.0
        required: false
      topK:
        title: Top K
        description: Top K parameter controlling predictability/diversity of output
        type: integer
        default: 64
        minimum: 64
        maximum: 64
        required: false
      stopSequences:
        title: Stop Sequences
        description: Allows to define sequences causing model to terminate generation
        type: array
        items:
          type: string
        default: null
        required: false
`, model.ModelName, versionCode, model.ModelLabel)
	}
}

// addToRegistryGo adds model to registry.go
func addToRegistryGo(model *ParsedModel) error {
	if dryRun {
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' to registry.go in registerVertexGoogleProcessors", model.ModelName))
		return nil
	}

	content, err := os.ReadFile(RegistryGo)
	if err != nil {
		return fmt.Errorf("failed to read registry.go: %w", err)
	}

	// Determine version code (e.g., "001" for gemini models)
	versionCode := "001"

	// Build the Go code block
	goBlock := fmt.Sprintf(`
	_ = registry.Register(ProcessorKey{
		Provider:       "vertex",
		Infrastructure: "gcp",
		Creator:        "google",
		ModelID:        "%s",
		Version:        "%s",
	}, func() interface{} {
		// NOTE:
		// We are using OpenAI API provided by Vertex Google, not the native Vertex AI API.
		//return extensions.NewVertexGoogle20240101Extension()
		return extensions.NewVertexGoogleOpenAIExtension()
	})
`, model.ModelName, versionCode)

	// Find the registerVertexGoogleProcessors function and insert before its closing brace
	funcPattern := `(?sm)(func registerVertexGoogleProcessors\(registry ProcessorRegistry\) \{.*?)(^\})`
	re := regexp.MustCompile(funcPattern)

	newContent := re.ReplaceAllString(string(content), "${1}"+goBlock+"$2")

	if newContent == string(content) {
		return fmt.Errorf("failed to find function registerVertexGoogleProcessors in registry.go")
	}

	if err := os.WriteFile(RegistryGo, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write registry.go: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' to registry.go", model.ModelName))
	return nil
}

// addToSpecYAML adds model to spec.yaml Models-VertexAIOpenAI enum
func addToSpecYAML(model *ParsedModel) error {
	if dryRun {
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' to spec.yaml", model.ModelName))
		return nil
	}

	content, err := os.ReadFile(SpecYAML)
	if err != nil {
		return fmt.Errorf("failed to read spec.yaml: %w", err)
	}

	contentStr := string(content)

	// Find Models-VertexAIOpenAI enum section
	enumPattern := regexp.MustCompile(`(?s)(Models-VertexAIOpenAI:.*?enum:\s*\n)((?:\s*- [^\n]+\n)+)`)
	match := enumPattern.FindStringSubmatchIndex(contentStr)

	if match == nil {
		return fmt.Errorf("could not find Models-VertexAIOpenAI enum in spec.yaml")
	}

	// Get the enum list end position
	enumListStart := match[4]
	enumListEnd := match[5]

	// Find proper indentation
	enumList := contentStr[enumListStart:enumListEnd]
	lines := strings.Split(enumList, "\n")
	indent := "        " // default
	for _, line := range lines {
		if strings.Contains(line, "- ") {
			indent = line[:len(line)-len(strings.TrimLeft(line, " "))]
			break
		}
	}

	// Insert new entry at the end of enum list
	newEntry := fmt.Sprintf("%s- %s\n", indent, model.ModelName)
	newContent := contentStr[:enumListEnd] + newEntry + contentStr[enumListEnd:]

	if err := os.WriteFile(SpecYAML, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write spec.yaml: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' to spec.yaml", model.ModelName))
	return nil
}

// addToMainTestGo adds test scenario to main_test.go
func addToMainTestGo(model *ParsedModel) error {
	if dryRun {
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' test to main_test.go", model.ModelName))
		return nil
	}

	content, err := os.ReadFile(MainTestGo)
	if err != nil {
		return fmt.Errorf("failed to read main_test.go: %w", err)
	}

	contentStr := string(content)

	// Determine URL path based on model type
	urlPath := "chat/completions"
	testName := "POST call VertexAI model"
	switch model.ModelType {
	case "embedding":
		urlPath = "embeddings"
		testName = "POST call VertexAI embed model"
	case "image":
		urlPath = "images/generations"
		testName = "POST call VertexAI image model"
	}

	// Add test case for the model
	testCase := fmt.Sprintf(`		{
			name:              "%s",
			method:            http.MethodPost,
			uri:               "/google/deployments/%s/%s",
			reqBody:           `+"`{}`"+`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
`, testName, model.ModelName, urlPath)

	// Find insertion point - look for the VertexAI embed model test case
	// We'll insert before it for chat_completion models
	insertPattern := regexp.MustCompile(`(\s*\{\s*\n\s*name:\s*"POST call VertexAI embed model")`)
	match := insertPattern.FindStringIndex(contentStr)

	if match == nil {
		// Fallback: try to find any VertexAI test and insert after it
		insertPattern = regexp.MustCompile(`(expectedHeaders:\s*successHeaders,\s*\},)(\s*\{[^}]*"POST embeddings with invalid api-version")`)
		match = insertPattern.FindStringIndex(contentStr)
		if match == nil {
			return fmt.Errorf("could not find insertion point for test case in main_test.go")
		}
		// Find end of first capture group
		subMatch := insertPattern.FindStringSubmatchIndex(contentStr)
		if subMatch != nil {
			insertPos := subMatch[3] // End of first capture group
			newContent := contentStr[:insertPos] + "\n" + testCase + contentStr[insertPos:]
			contentStr = newContent
		}
	} else {
		// Insert before the embedding test
		newContent := contentStr[:match[0]] + testCase + contentStr[match[0]:]
		contentStr = newContent
	}

	// Add to getModelsYamlContent function
	modelEntry := fmt.Sprintf(`- name: %s
  redirectUrl: `+"`"+` + redirectUrl + `+"`"+`
  provider: google
  creator: google
`, model.ModelName)

	// Find getModelsYamlContent function and add entry before "buddies:"
	buddiesPattern := regexp.MustCompile(`(buddies:\n- name: selfstudybuddy)`)
	buddiesMatch := buddiesPattern.FindStringIndex(contentStr)
	if buddiesMatch != nil {
		contentStr = contentStr[:buddiesMatch[0]] + modelEntry + contentStr[buddiesMatch[0]:]
	}

	if err := os.WriteFile(MainTestGo, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("failed to write main_test.go: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' test to main_test.go", model.ModelName))
	return nil
}

// addModelConfig creates the model configuration file
func addModelConfig(model *ParsedModel) error {
	configPath := fmt.Sprintf("%s/%s", ModelConfigDir, model.ConfigFileName)

	if dryRun {
		printInfo(fmt.Sprintf("[DRY-RUN] Would create model config at %s", configPath))
		return nil
	}

	configContent := fmt.Sprintf(`- name: %s
  modelId: ""
  modelUrl: "http://{{ .Release.Namespace }}.genai-hub-service.svc.cluster.local:443/google/deployments/%s"
  redirectUrl: "{{ .Values.DemoGcpVertexURL }}/google/deployments/%s"
  provider: google
  creator: google
  targetAPI: "/chat/completions"
  path: "/google/deployments/%s/chat/completions"
  infrastructure: gcp
  capabilities:
    completions: false
    embeddings: false
    image-generation: false
`,
		model.ModelName, model.ModelName, model.ModelName, model.ModelName,
	)

	// Adjust targetAPI based on model type
	switch model.ModelType {
	case "embedding":
		configContent = strings.ReplaceAll(configContent, `targetAPI: "/chat/completions"`, `targetAPI: "/embeddings"`)
		configContent = strings.ReplaceAll(configContent, `/chat/completions`, `/embeddings`)
	case "image":
		configContent = strings.ReplaceAll(configContent, `targetAPI: "/chat/completions"`, `targetAPI: "/image/generation"`)
		configContent = strings.ReplaceAll(configContent, `/chat/completions`, `/image/generation`)
	}

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create model config: %w", err)
	}

	printSuccess(fmt.Sprintf("Created model config at %s", configPath))
	return nil
}

// addToModelMetadata adds model metadata to model-metadata.yaml
func addToModelMetadata(model *ParsedModel) error {
	if dryRun {
		templateInfo := "none (scaffold)"
		if model.Template != "" {
			templateInfo = model.Template
		}
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' to model-metadata.yaml (template: %s)", model.ModelName, templateInfo))
		return nil
	}

	var yamlBlock string

	if model.Template != "" && fileContains(ModelMetadataYAML, fmt.Sprintf("    %s:", model.Template)) {
		printInfo(fmt.Sprintf("Using template '%s' for model-metadata.yaml", model.Template))

		// Read file and extract template block
		content, err := os.ReadFile(ModelMetadataYAML)
		if err != nil {
			return fmt.Errorf("failed to read model-metadata.yaml: %w", err)
		}

		// Extract template block
		pattern := fmt.Sprintf(`(?s)(    %s:.*?)(\n    [a-zA-Z]|\z)`, regexp.QuoteMeta(model.Template))
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(string(content))

		if len(matches) > 1 {
			templateBlock := matches[1]
			// Replace identifiers
			yamlBlock = strings.Replace(templateBlock, fmt.Sprintf("    %s:", model.Template), fmt.Sprintf("    %s:", model.ModelName), 1)
			yamlBlock = regexp.MustCompile(fmt.Sprintf(`model_mapping_id: %s`, regexp.QuoteMeta(model.Template))).ReplaceAllString(yamlBlock, fmt.Sprintf("model_mapping_id: %s", model.ModelName))
			yamlBlock = regexp.MustCompile(`model_name: [^\n]+`).ReplaceAllString(yamlBlock, fmt.Sprintf("model_name: %s", toModelName(model.ModelName)))
			yamlBlock = regexp.MustCompile(`model_label: [^\n]+`).ReplaceAllString(yamlBlock, fmt.Sprintf("model_label: %s", model.ModelLabel))
			yamlBlock = regexp.MustCompile(`model_id: [^\n]+`).ReplaceAllString(yamlBlock, fmt.Sprintf("model_id: %s", model.ModelName))
			yamlBlock = regexp.MustCompile(`version: '[^']*'`).ReplaceAllString(yamlBlock, fmt.Sprintf("version: '%s'", model.Version))
		}
	}

	if yamlBlock == "" {
		// Create minimal scaffold with TODO markers
		yamlBlock = generateMetadataScaffold(model)
	}

	// Append to file
	f, err := os.OpenFile(ModelMetadataYAML, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open model-metadata.yaml: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(yamlBlock + "\n"); err != nil {
		return fmt.Errorf("failed to write to model-metadata.yaml: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' to model-metadata.yaml", model.ModelName))
	if model.Template == "" {
		printWarning("Review and update TODO markers in model-metadata.yaml")
	}
	return nil
}

// toModelName converts model name to Pascal-Hyphen case for model_name field
// e.g., "gemini-2.5-flash-lite" -> "Gemini-2.5-Flash-Lite"
func toModelName(s string) string {
	words := strings.Split(s, "-")
	result := []string{}
	for _, word := range words {
		if matched, _ := regexp.MatchString(`^\d+(\.\d+)?$`, word); matched {
			result = append(result, word)
		} else if len(word) > 0 {
			result = append(result, strings.ToUpper(string(word[0]))+strings.ToLower(word[1:]))
		}
	}
	return strings.Join(result, "-")
}

// generateMetadataScaffold creates a scaffold metadata block with TODO markers
func generateMetadataScaffold(model *ParsedModel) string {
	switch model.ModelType {
	case "embedding":
		return fmt.Sprintf(`    %s:
      model_name: %s
      model_description: Google Embedding model  # TODO: Update description
      model_mapping_id: %s
      version: '%s'
      model_label: %s
      model_id: %s
      type: embedding
      input_tokens: 2048  # TODO: Verify max input tokens
      model_capabilities:
        input_modalities:
          - text
        output_modalities:
          - embedding
`, model.ModelName, toModelName(model.ModelName), model.ModelName, model.Version, model.ModelLabel, model.ModelName)

	case "image":
		return fmt.Sprintf(`    %s:
      model_name: %s
      model_description: Google Image Generation model  # TODO: Update description
      model_mapping_id: %s
      version: '%s'
      model_label: %s
      model_id: %s
      type: image
      input_tokens: 480  # TODO: Verify max input tokens
      model_capabilities:
        input_modalities:
          - text
        output_modalities:
          - image
      parameters:
        aspect_ratio:
          default: null
          description: The aspect ratio for the generated output image
          examples:
            - 1:1
            - 9:16
            - 16:9
            - 3:4
            - 4:3
          title: Aspect Ratio
          type: string
          required: false
`, model.ModelName, toModelName(model.ModelName), model.ModelName, model.Version, model.ModelLabel, model.ModelName)

	default: // chat_completion
		return fmt.Sprintf(`    %s:
      model_name: %s
      model_description: Gemini Chat Completions model  # TODO: Update description
      model_mapping_id: %s
      version: '%s'
      model_label: %s
      model_id: %s
      type: chat_completion
      input_tokens: 1048576  # TODO: Verify max input tokens
      model_capabilities:
        features:
          - functionCalling  # TODO: Verify features
          - jsonMode
          - structuredOutput
          - promptCaching
        input_modalities:
          - text
          - code
          - image
          - audio
          - video
        output_modalities:
          - text
        mime_types:  # TODO: Verify supported mime types
          - application/pdf
          - text/csv
          - text/plain
          - text/markdown
          - text/html
          - text/xml
          - audio/wav
          - audio/mp3
          - audio/aiff
          - audio/aac
          - audio/ogg
          - audio/flac
          - image/png
          - image/jpeg
          - image/webp
          - image/heic
          - image/heif
          - video/mp4
          - video/mpeg
          - video/mov
          - video/avi
          - video/x-flv
          - video/mpg
          - video/webm
          - video/wmv
          - video/3gpp
      parameters:
        max_tokens:
          title: Max Output Tokens
          description: Maximum number of tokens to generate
          type: integer
          default: null
          maximum: 65535  # TODO: Verify max output tokens
          required: false
        temperature:
          title: Temperature
          description: Controls randomness of the generated output
          type: float
          default: 1.0
          maximum: 2.0
          minimum: 0.0
          required: false
        top_p:
          title: Top P
          description: Nucleus sampling parameter controlling diversity of the output
          type: float
          default: 0.95
          minimum: 0.0
          maximum: 1.0
          required: false
        stop:
          title: Stop Sequences
          description: Allows to define sequences causing model to terminate generation
          type: array
          default: null
          required: false
        response_format:
          default: null
          description: response_format
          examples:
            - text
            - json_object
          title: Response Format
          type: object
          required: false
        n:
          title: N
          description: number of completions to generate
          type: integer
          default: null
          required: false
        tool_choice:
          default: null
          description: auto, none literals or valid function name string
          examples:
            - auto
            - none
            - <tool_name>
          title: Tool Choice
          type: string
          required: false
      lifecycle: Generally Available  # TODO: Verify lifecycle status
`, model.ModelName, toModelName(model.ModelName), model.ModelName, model.Version, model.ModelLabel, model.ModelName)
	}
}

// addToIntegrationTests adds integration test scenarios
func addToIntegrationTests(model *ParsedModel) error {
	if dryRun {
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' integration tests to mappings_test.go", model.ModelName))
		return nil
	}

	content, err := os.ReadFile(IntegrationMappingsGo)
	if err != nil {
		return fmt.Errorf("failed to read mappings_test.go: %w", err)
	}

	contentStr := string(content)

	// Determine URL path based on model type
	urlPath := "URLPathChatCompletions"
	switch model.ModelType {
	case "embedding":
		urlPath = "UrlPathEmbeddings"
	case "image":
		urlPath = "URLPathImageGenerations"
	}

	// Generate test block for the model
	testBlocks := fmt.Sprintf(`
		It("must redirect to model %s", func() {
			model := GetModelByName(mappings.Models, "%s")
			prepareAndInvokeModel(model, %s, "", testID)
		})
`,
		model.ModelName, model.ModelName, urlPath,
	)

	// Find insertion point - at the end of the Context block, before the closing })
	// Look for the last test case and insert after it
	insertPattern := regexp.MustCompile(`(?s)(It\("must redirect to model [^"]+", func\(\) \{[^}]+\}\)[^}]*\}\))(\s*\}\))`)
	matches := insertPattern.FindAllStringSubmatchIndex(contentStr, -1)

	if len(matches) == 0 {
		return fmt.Errorf("could not find insertion point for integration tests in mappings_test.go")
	}

	// Get the last match
	lastMatch := matches[len(matches)-1]
	insertPos := lastMatch[3] // Position after the last test's closing })

	newContent := contentStr[:insertPos] + testBlocks + contentStr[insertPos:]

	if err := os.WriteFile(IntegrationMappingsGo, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write mappings_test.go: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' integration tests to mappings_test.go", model.ModelName))
	return nil
}

// processModel processes a single model
func processModel(modelName, modelType, template string, preview bool) error {
	// Apply preview suffix if requested
	if preview && !strings.HasSuffix(modelName, "-preview") {
		modelName = modelName + "-preview"
	}

	model, err := parseModelName(modelName, modelType)
	if err != nil {
		return err
	}
	model.Template = template
	model.Preview = preview

	printHeader(fmt.Sprintf("Processing Model: %s", model.ModelName))
	fmt.Printf("  Model Name:    %s\n", model.ModelName)
	fmt.Printf("  Model Type:    %s\n", model.ModelType)
	fmt.Printf("  Model Label:   %s\n", model.ModelLabel)
	fmt.Printf("  Version:       %s\n", model.Version)
	fmt.Printf("  Config File:   %s\n", model.ConfigFileName)
	if preview {
		fmt.Printf("  Preview:       yes\n")
	}
	if template != "" {
		fmt.Printf("  Template:      %s\n", template)
	}

	// Check for existing entries and track what needs to be added
	var existsIn []string
	var addedTo []string
	var errors []string

	// Define all configuration locations with their check and add functions
	type configLocation struct {
		name      string
		checkFunc func() bool
		addFunc   func() error
	}

	locations := []configLocation{
		{
			name:      "spec.yaml",
			checkFunc: func() bool { return checkSpecYAML(model.ModelName) },
			addFunc:   func() error { return addToSpecYAML(model) },
		},
		{
			name:      "main_test.go",
			checkFunc: func() bool { return checkMainTestGo(model.ModelName) },
			addFunc:   func() error { return addToMainTestGo(model) },
		},
		{
			name:      "model config",
			checkFunc: func() bool { return checkModelConfig(model.ConfigFileName) },
			addFunc:   func() error { return addModelConfig(model) },
		},
		{
			name:      "model-metadata.yaml",
			checkFunc: func() bool { return checkModelMetadata(model.ModelName) },
			addFunc:   func() error { return addToModelMetadata(model) },
		},
		{
			name:      "mappings_test.go",
			checkFunc: func() bool { return checkIntegrationTests(model.ModelName) },
			addFunc:   func() error { return addToIntegrationTests(model) },
		},
		{
			name:      "model spec YAML",
			checkFunc: func() bool { return checkModelSpecYAML(model.ModelName) },
			addFunc:   func() error { return addToModelSpecYAML(model) },
		},
		{
			name:      "registry.go",
			checkFunc: func() bool { return checkRegistryGo(model.ModelName) },
			addFunc:   func() error { return addToRegistryGo(model) },
		},
	}

	// Process each location independently
	for _, loc := range locations {
		if loc.checkFunc() {
			existsIn = append(existsIn, loc.name)
		} else {
			if err := loc.addFunc(); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", loc.name, err))
			} else {
				addedTo = append(addedTo, loc.name)
			}
		}
	}

	// Report results
	fmt.Println()
	if len(existsIn) > 0 {
		printWarning(fmt.Sprintf("Already exists in: %s", strings.Join(existsIn, ", ")))
	}

	if len(errors) > 0 {
		for _, errMsg := range errors {
			printError(errMsg)
		}
	}

	// Determine outcome
	if len(addedTo) > 0 {
		modelsAdded++
		printSuccess(fmt.Sprintf("Added '%s' to: %s", model.ModelName, strings.Join(addedTo, ", ")))
	} else if len(existsIn) == len(locations) {
		modelsSkipped++
		printWarning(fmt.Sprintf("Skipped model: %s (already fully configured)", model.ModelName))
	} else if len(errors) > 0 {
		return fmt.Errorf("failed to add model to some locations")
	}

	return nil
}

func main() {
	modelName := flag.String("model-name", "", "Model name to add (e.g., gemini-2.5-flash-lite)")
	modelType := flag.String("model-type", "", "Model type: chat_completion, embedding, image")
	template := flag.String("template", "", "Template model to copy metadata from (optional)")
	configFile := flag.String("config", "", "JSON config file with model list")
	preview := flag.Bool("preview", false, "Add model as preview (appends -preview suffix)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	flag.Parse()

	printHeader("Add Vertex AI Model Script")

	if dryRun {
		printWarning("DRY-RUN MODE - No changes will be made")
	}

	if *configFile != "" {
		// Batch mode from JSON
		content, err := os.ReadFile(*configFile)
		if err != nil {
			printError(fmt.Sprintf("Failed to read config file: %v", err))
			os.Exit(1)
		}

		var config BatchConfig
		if err := json.Unmarshal(content, &config); err != nil {
			printError(fmt.Sprintf("Failed to parse config file: %v", err))
			os.Exit(1)
		}

		printInfo(fmt.Sprintf("Found %d model(s) in config file", len(config.Models)))

		for _, m := range config.Models {
			if err := processModel(m.ModelName, m.ModelType, m.Template, m.Preview); err != nil {
				printError(fmt.Sprintf("Error processing model %s: %v", m.ModelName, err))
			}
		}

	} else if *modelName != "" && *modelType != "" {
		// Single model mode
		if err := processModel(*modelName, *modelType, *template, *preview); err != nil {
			printError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	} else {
		printError("Missing required parameters")
		fmt.Println("Use -model-name and -model-type, or -config to specify models")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run scripts/add-vertex-model.go -model-name <name> -model-type <type> [-template <template>] [-preview] [-dry-run]")
		fmt.Println("  go run scripts/add-vertex-model.go -config <config.json> [-dry-run]")
		fmt.Println()
		fmt.Println("Model types: chat_completion, embedding, image")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -preview    Add model as preview (appends -preview suffix to model name)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run scripts/add-vertex-model.go -model-name gemini-2.5-flash-lite -model-type chat_completion")
		fmt.Println("  go run scripts/add-vertex-model.go -model-name gemini-2.5-flash-lite -model-type chat_completion -template gemini-2.5-flash")
		fmt.Println("  go run scripts/add-vertex-model.go -model-name gemini-3.0-flash -model-type chat_completion -preview")
		fmt.Println("  go run scripts/add-vertex-model.go -config scripts/vertex-model-sample.json")
		os.Exit(1)
	}

	// Print summary
	printHeader("Summary")
	fmt.Println()
	fmt.Printf("  Models added:     %d\n", modelsAdded)
	fmt.Printf("  Models skipped:   %d\n", modelsSkipped)

	if dryRun {
		fmt.Println()
		printInfo("This was a dry run. No changes were made.")
		printInfo("Run without -dry-run to apply changes.")
	} else if modelsAdded > 0 {
		fmt.Println()
		printSuccess("Models scaffolded successfully!")
		printWarning("Review generated files for any TODO markers")
		printInfo("Files modified:")
		printInfo(fmt.Sprintf("  - %s", SpecYAML))
		printInfo(fmt.Sprintf("  - %s", MainTestGo))
		printInfo(fmt.Sprintf("  - %s/<model>.yaml", ModelConfigDir))
		printInfo(fmt.Sprintf("  - %s", ModelMetadataYAML))
		printInfo(fmt.Sprintf("  - %s", IntegrationMappingsGo))
		printInfo(fmt.Sprintf("  - %s/google/<spec>.yaml", SpecsBaseDir))
		printInfo(fmt.Sprintf("  - %s", RegistryGo))
	}
}
