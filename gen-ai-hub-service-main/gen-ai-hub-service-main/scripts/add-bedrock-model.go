//go:build ignore
// +build ignore

/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 *
 * Cross-platform script to add new AWS Bedrock models to the gen-ai-hub-service
 * Usage:
 *   Single model:  go run scripts/add-bedrock-model.go -model-id amazon.nova-new-v1:0 [-template nova-lite-v1]
 *   Batch mode:    go run scripts/add-bedrock-model.go -config models.json
 *   Dry run:       go run scripts/add-bedrock-model.go -model-id amazon.nova-new-v1:0 -dry-run
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	MetadataJSON      = "distribution/genai-awsbedrock-infra-sce/src/main/resources/metadata.json"
	ModelMetadataYAML = "distribution/genai-hub-service-helm/src/main/helm/templates/model-metadata.yaml"
	SpecsBaseDir      = "internal/models/specs/aws/bedrock"
	RegistryGo        = "internal/request/processors/registry/registry.go"
)

// ModelConfig represents the configuration for a single model
type ModelConfig struct {
	ModelID  string `json:"model_id"`
	Template string `json:"template,omitempty"`
}

// BatchConfig represents the JSON configuration file format
type BatchConfig struct {
	Models []ModelConfig `json:"models"`
}

// ParsedModel contains parsed model information
type ParsedModel struct {
	ModelID        string
	Creator        string
	ModelMappingID string
	ModelName      string
	ModelLabel     string
	Version        string
	SpecFile       string
	RegistryFunc   string
	Extension      string
	Template       string
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

// parseModelID parses the model ID and extracts components
func parseModelID(modelID string) (*ParsedModel, error) {
	parts := strings.SplitN(modelID, ".", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid model_id format: %s (expected: creator.model-name:version)", modelID)
	}

	creator := parts[0]
	rest := parts[1]

	// Split by colon to get version
	modelParts := strings.SplitN(rest, ":", 2)
	modelMappingID := modelParts[0]
	version := "v1"
	if len(modelParts) > 1 && modelParts[1] != "" {
		version = modelParts[1]
	}

	// Generate model name (Title-Case-With-Hyphens)
	modelName := toTitleCase(modelMappingID)

	// Generate label (Title Case With Spaces)
	modelLabel := strings.ReplaceAll(modelName, "-", " ")

	// Determine spec file and registry function based on creator
	var specFile, registryFunc, extension string
	switch creator {
	case "amazon":
		if strings.Contains(modelMappingID, "nova") {
			specFile = "amazon/nova.yaml"
		} else if strings.Contains(modelMappingID, "titan") {
			specFile = "amazon/embeddings.yaml"
		} else {
			specFile = "amazon/nova.yaml"
		}
		registryFunc = "registerBedrockAmazonProcessors"
		extension = "NewBedrockAmazon20230601Extension"
	case "anthropic":
		specFile = "anthropic/claude.yaml"
		registryFunc = "registerBedrockAnthropicProcessors"
		extension = "NewBedrockAnthropic20230601Extension"
	case "meta":
		specFile = "meta/llama.yaml"
		registryFunc = "registerBedrockMetaProcessors"
		extension = "NewBedrockMeta20230601Extension"
	default:
		return nil, fmt.Errorf("unknown creator: %s (must be: amazon, anthropic, or meta)", creator)
	}

	return &ParsedModel{
		ModelID:        modelID,
		Creator:        creator,
		ModelMappingID: modelMappingID,
		ModelName:      modelName,
		ModelLabel:     modelLabel,
		Version:        version,
		SpecFile:       specFile,
		RegistryFunc:   registryFunc,
		Extension:      extension,
	}, nil
}

// toTitleCase converts hyphenated string to Title-Case-With-Hyphens
func toTitleCase(s string) string {
	words := strings.Split(s, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, "-")
}

// fileContains checks if a file contains a specific string
func fileContains(filePath, search string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), search)
}

// checkMetadataJSON checks if model exists in metadata.json
func checkMetadataJSON(modelID string) bool {
	return fileContains(MetadataJSON, fmt.Sprintf(`"%s"`, modelID))
}

// checkModelMetadataYAML checks if model exists in model-metadata.yaml
func checkModelMetadataYAML(modelMappingID string) bool {
	return fileContains(ModelMetadataYAML, fmt.Sprintf("    %s:", modelMappingID))
}

// checkRegistryGo checks if model exists in registry.go
func checkRegistryGo(modelMappingID string) bool {
	return fileContains(RegistryGo, fmt.Sprintf(`ModelID:.*"%s"`, modelMappingID))
}

// addToMetadataJSON adds model ID to metadata.json allowedValues
// Uses text-based manipulation to preserve original formatting
func addToMetadataJSON(model *ParsedModel) error {
	if dryRun {
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' to metadata.json", model.ModelID))
		return nil
	}

	content, err := os.ReadFile(MetadataJSON)
	if err != nil {
		return fmt.Errorf("failed to read metadata.json: %w", err)
	}

	contentStr := string(content)

	// Find the ModelID parameter section and its allowedValues array
	// Pattern: find "name": "ModelID" section, then find the allowedValues array within it

	// First, locate the ModelID parameter block
	modelIDPattern := regexp.MustCompile(`(?s)"name":\s*"ModelID".*?"allowedValues":\s*\[([^\]]*)\]`)
	match := modelIDPattern.FindStringSubmatchIndex(contentStr)

	if match == nil {
		return fmt.Errorf("could not find ModelID allowedValues in metadata.json")
	}

	// Get the position of the closing bracket of allowedValues array
	// match[2] and match[3] are the start and end of the captured group (content inside [])
	arrayContentStart := match[2]
	arrayContentEnd := match[3]

	// Find the last entry in the array to determine proper indentation and comma placement
	arrayContent := contentStr[arrayContentStart:arrayContentEnd]

	// Check if there are existing entries
	lastQuoteIndex := strings.LastIndex(arrayContent, `"`)
	if lastQuoteIndex == -1 {
		return fmt.Errorf("could not find existing entries in allowedValues array")
	}

	// Find the position in the original string where we need to insert
	// We want to insert after the last entry (after its closing quote)
	insertPos := arrayContentStart + lastQuoteIndex + 1

	// Determine indentation by looking at existing entries
	// Find the indentation of the last line with a quote
	lines := strings.Split(arrayContent, "\n")
	indent := "          " // default indentation
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], `"`) {
			// Extract leading whitespace
			trimmed := strings.TrimLeft(lines[i], " \t")
			indent = lines[i][:len(lines[i])-len(trimmed)]
			break
		}
	}

	// Create the new entry: comma, newline, indentation, quoted model ID
	newEntry := fmt.Sprintf(",\n%s\"%s\"", indent, model.ModelID)

	// Insert the new entry
	newContent := contentStr[:insertPos] + newEntry + contentStr[insertPos:]

	if err := os.WriteFile(MetadataJSON, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write metadata.json: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' to metadata.json", model.ModelID))
	return nil
}

// addToModelMetadataYAML adds model to model-metadata.yaml
func addToModelMetadataYAML(model *ParsedModel) error {
	if dryRun {
		templateInfo := "none"
		if model.Template != "" {
			templateInfo = model.Template
		}
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' to model-metadata.yaml (template: %s)", model.ModelMappingID, templateInfo))
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

		// Simple extraction using regex
		pattern := fmt.Sprintf(`(?s)(    %s:.*?)(\n    [a-zA-Z]|\z)`, regexp.QuoteMeta(model.Template))
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(string(content))

		if len(matches) > 1 {
			templateBlock := matches[1]
			// Replace identifiers
			yamlBlock = strings.Replace(templateBlock, fmt.Sprintf("    %s:", model.Template), fmt.Sprintf("    %s:", model.ModelMappingID), 1)
			yamlBlock = regexp.MustCompile(fmt.Sprintf(`model_mapping_id: %s`, model.Template)).ReplaceAllString(yamlBlock, fmt.Sprintf("model_mapping_id: %s", model.ModelMappingID))
			yamlBlock = regexp.MustCompile(`model_name: .*`).ReplaceAllString(yamlBlock, fmt.Sprintf("model_name: %s", model.ModelName))
			yamlBlock = regexp.MustCompile(`model_label: .*`).ReplaceAllString(yamlBlock, fmt.Sprintf("model_label: %s", model.ModelLabel))
			yamlBlock = regexp.MustCompile(`version: .*`).ReplaceAllString(yamlBlock, fmt.Sprintf("version: %s", model.Version))
		}
	}

	if yamlBlock == "" {
		// Create minimal scaffold with TODO markers
		yamlBlock = fmt.Sprintf(`    %s:
      model_name: %s
      model_description: TODO - Add description
      model_mapping_id: %s
      version: %s
      model_label: %s
      type: chat_completion  # TODO: Verify type
      input_tokens: 0  # TODO: Set max input tokens
      model_capabilities:
        features:
          - streaming  # TODO: Verify features
        input_modalities:
          - text  # TODO: Verify modalities
        output_modalities:
          - text
      parameters:
        max_tokens:
          title: Max Output Tokens
          description: Maximum number of tokens to generate
          type: integer
          maximum: 4096  # TODO: Set correct maximum
          default: null
          required: false
        temperature:
          title: Temperature
          description: Controls randomness of the generated output
          type: float
          default: null
          maximum: 1.0
          minimum: 0.0
          required: false
        top_p:
          title: Top P
          description: Nucleus sampling parameter controlling diversity of the output.
          type: float
          default: null
          maximum: 1.0
          minimum: 0.0
          required: false
`, model.ModelMappingID, model.ModelName, model.ModelMappingID, model.Version, model.ModelLabel)
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

	printSuccess(fmt.Sprintf("Added '%s' to model-metadata.yaml", model.ModelMappingID))
	if model.Template == "" {
		printWarning("Review and update TODO markers in model-metadata.yaml")
	}
	return nil
}

// addToSpecYAML adds model to spec YAML file
func addToSpecYAML(model *ParsedModel) error {
	specFile := filepath.Join(SpecsBaseDir, model.SpecFile)

	if dryRun {
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' to %s", model.ModelMappingID, specFile))
		return nil
	}

	// Check if spec file exists, create if not
	if _, err := os.Stat(specFile); os.IsNotExist(err) {
		printWarning(fmt.Sprintf("Spec file does not exist: %s. Creating new file.", specFile))

		if err := os.MkdirAll(filepath.Dir(specFile), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		header := fmt.Sprintf(`infrastructure: aws
provider: bedrock
creator: %s
metadata:
  description: "%s model family on AWS Bedrock"
  documentation: "https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids-arns.html"
models:
`, model.Creator, strings.Title(model.Creator))

		if err := os.WriteFile(specFile, []byte(header), 0644); err != nil {
			return fmt.Errorf("failed to create spec file: %w", err)
		}
	}

	// Create scaffold YAML block
	yamlBlock := fmt.Sprintf(`  - name: %s
    version: %s
    id: %s
    label: %s
    functionalCapabilities: ["chat_completion"]  # TODO: Verify
    endpoints:
      - path: /converse
      - path: /converse-stream
      - path: /invoke
      - path: /invoke-with-response-stream
    capabilities:
      features: [streaming]  # TODO: Verify features
      inputModalities: [text]  # TODO: Verify modalities
      outputModalities: [text]
    deployment:
      region: us-east-1
      instanceType: bedrock-runtime
    parameters:
      maxInputTokens:
        title: Max Input Tokens
        description: Maximum number of tokens for input
        type: integer
        maximum: 128000  # TODO: Set correct maximum
        required: false
      maxOutputTokens:
        title: Max Output Tokens
        description: Maximum number of tokens to generate
        type: integer
        maximum: 4096  # TODO: Set correct maximum
        default: null
        required: false
      temperature:
        title: Temperature
        description: Controls randomness of the generated output
        type: float
        default: null
        maximum: 1.0
        minimum: 0.0
        required: false
      topP:
        title: Top P
        description: Nucleus sampling parameter
        type: float
        default: null
        maximum: 1.0
        minimum: 0.0
        required: false
`, model.ModelMappingID, model.Version, model.ModelID, model.ModelLabel)

	// Append to file
	f, err := os.OpenFile(specFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open spec file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(yamlBlock); err != nil {
		return fmt.Errorf("failed to write to spec file: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' to %s", model.ModelMappingID, specFile))
	return nil
}

// addToRegistryGo adds model to registry.go
func addToRegistryGo(model *ParsedModel) error {
	if dryRun {
		printInfo(fmt.Sprintf("[DRY-RUN] Would add '%s' to registry.go in %s", model.ModelMappingID, model.RegistryFunc))
		return nil
	}

	content, err := os.ReadFile(RegistryGo)
	if err != nil {
		return fmt.Errorf("failed to read registry.go: %w", err)
	}

	// Build the Go code block
	goBlock := fmt.Sprintf(`
	_ = registry.Register(ProcessorKey{
		Provider:       "bedrock",
		Infrastructure: "aws",
		Creator:        "%s",
		ModelID:        "%s",
		Version:        "%s",
	}, func() interface{} {
		return extensions.%s()
	})
`, model.Creator, model.ModelMappingID, model.Version, model.Extension)

	// Find the function and insert before its closing brace
	// Use (?sm) for single-line mode (. matches newlines) and multiline mode (^ matches line beginnings)
	funcPattern := fmt.Sprintf(`(?sm)(func %s\(registry ProcessorRegistry\) \{.*?)(^\})`, model.RegistryFunc)
	re := regexp.MustCompile(funcPattern)

	newContent := re.ReplaceAllString(string(content), "${1}"+goBlock+"$2")

	if newContent == string(content) {
		return fmt.Errorf("failed to find function %s in registry.go", model.RegistryFunc)
	}

	if err := os.WriteFile(RegistryGo, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write registry.go: %w", err)
	}

	printSuccess(fmt.Sprintf("Added '%s' to registry.go", model.ModelMappingID))
	return nil
}

// processModel processes a single model
func processModel(modelID, template string) error {
	model, err := parseModelID(modelID)
	if err != nil {
		return err
	}
	model.Template = template

	printHeader(fmt.Sprintf("Processing Model: %s", model.ModelMappingID))
	fmt.Printf("  Model ID:    %s\n", model.ModelID)
	fmt.Printf("  Creator:     %s\n", model.Creator)
	fmt.Printf("  Name:        %s\n", model.ModelName)
	fmt.Printf("  Label:       %s\n", model.ModelLabel)
	fmt.Printf("  Version:     %s\n", model.Version)
	fmt.Printf("  Spec File:   %s\n", model.SpecFile)
	if template != "" {
		fmt.Printf("  Template:    %s\n", template)
	}

	// Check for existing entries
	var existsIn []string

	if checkMetadataJSON(model.ModelID) {
		existsIn = append(existsIn, "metadata.json")
	}
	if checkModelMetadataYAML(model.ModelMappingID) {
		existsIn = append(existsIn, "model-metadata.yaml")
	}
	if checkRegistryGo(model.ModelMappingID) {
		existsIn = append(existsIn, "registry.go")
	}

	if len(existsIn) > 0 {
		printWarning(fmt.Sprintf("Model already exists in: %s", strings.Join(existsIn, ", ")))
		printWarning(fmt.Sprintf("Skipping model: %s", model.ModelMappingID))
		modelsSkipped++
		return nil
	}

	// Add to all files
	if err := addToMetadataJSON(model); err != nil {
		return err
	}
	if err := addToModelMetadataYAML(model); err != nil {
		return err
	}
	if err := addToSpecYAML(model); err != nil {
		return err
	}
	if err := addToRegistryGo(model); err != nil {
		return err
	}

	modelsAdded++
	printSuccess(fmt.Sprintf("Successfully scaffolded model: %s", model.ModelMappingID))
	return nil
}

func main() {
	modelID := flag.String("model-id", "", "Model ID to add (e.g., amazon.nova-new-v1:0)")
	template := flag.String("template", "", "Template model to copy from (optional)")
	configFile := flag.String("config", "", "JSON config file with model list")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	flag.Parse()

	printHeader("Add Bedrock Model Script")

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
			if err := processModel(m.ModelID, m.Template); err != nil {
				printError(fmt.Sprintf("Error processing model %s: %v", m.ModelID, err))
			}
		}

	} else if *modelID != "" {
		// Single model mode
		if err := processModel(*modelID, *template); err != nil {
			printError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

	} else {
		printError("No model specified")
		fmt.Println("Use -model-id or -config to specify models")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  go run scripts/add-bedrock-model.go -model-id <model_id> [-template <template>] [-dry-run]")
		fmt.Println("  go run scripts/add-bedrock-model.go -config <config.json> [-dry-run]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run scripts/add-bedrock-model.go -model-id amazon.nova-new-v1:0")
		fmt.Println("  go run scripts/add-bedrock-model.go -model-id amazon.nova-new-v1:0 -template nova-lite-v1")
		fmt.Println("  go run scripts/add-bedrock-model.go -config scripts/bedrock-model-sample.json")
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
		printWarning("Review and update TODO markers in the generated files")
		printInfo("Files modified:")
		printInfo(fmt.Sprintf("  - %s", MetadataJSON))
		printInfo(fmt.Sprintf("  - %s", ModelMetadataYAML))
		printInfo(fmt.Sprintf("  - %s/<spec>.yaml", SpecsBaseDir))
		printInfo(fmt.Sprintf("  - %s", RegistryGo))
	}
}
