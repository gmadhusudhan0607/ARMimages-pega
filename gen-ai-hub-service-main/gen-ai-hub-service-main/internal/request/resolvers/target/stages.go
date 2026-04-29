/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"regexp"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
)

// extractBasicInfo extracts basic information from the request URL and path
// This stage extracts: route pattern, model name, operation path, query parameters, etc.
func (r *TargetResolver) extractBasicInfo(ctx context.Context, req *ResolutionRequest) error {
	c := req.GinContext
	fullPath := c.Request.URL.Path
	rawQuery := c.Request.URL.RawQuery

	req.Metadata["fullPath"] = fullPath
	req.Metadata["rawQuery"] = rawQuery

	// Extract route pattern from path
	// Patterns: /openai, /anthropic, /meta, /amazon, /google, /buddies, /models, /swagger, /health
	routePattern := extractRoutePattern(fullPath)
	req.Metadata["routePattern"] = routePattern

	// Extract model name or buddy ID from path
	if routePattern == "buddies" {
		// Pattern: /v1/{isolationId}/buddies/{buddyId}/...
		buddyID, isolationID := extractBuddyInfo(fullPath)
		req.Metadata["buddyId"] = buddyID
		req.Metadata["isolationId"] = isolationID
	} else if routePattern != "" && routePattern != "models" && routePattern != "swagger" && routePattern != "health" {
		// Pattern: /{provider}/deployments/{modelName}/...
		modelName := extractModelName(fullPath)
		req.Metadata["modelName"] = modelName
	}

	// Extract operation path (everything after model name or buddy ID)
	operation := extractOperationPath(fullPath, routePattern)
	req.Metadata["operation"] = operation

	return nil
}

// determineTargetType classifies the request based on the route pattern
func (r *TargetResolver) determineTargetType(ctx context.Context, req *ResolutionRequest) error {
	routePattern, ok := req.Metadata["routePattern"].(string)
	if !ok {
		return NewResolutionError("determineTargetType", "route pattern not found", "")
	}

	switch routePattern {
	case "openai", "anthropic", "meta", "amazon", "google":
		req.Target.TargetType = TargetTypeLLM
	case "buddies":
		req.Target.TargetType = TargetTypeBuddy
	case "models", "swagger", "health":
		req.Target.TargetType = TargetTypeUnknown
	case "":
		req.Target.TargetType = TargetTypeUnknown
	default:
		req.Target.TargetType = TargetTypeUnknown
	}

	return nil
}

// fetchModelConfiguration fetches model or buddy configuration from appropriate source
func (r *TargetResolver) fetchModelConfiguration(ctx context.Context, req *ResolutionRequest) error {
	logger := cntx.LoggerFromContext(ctx).Sugar()
	routePattern, _ := req.Metadata["routePattern"].(string)
	modelName, _ := req.Metadata["modelName"].(string)

	switch routePattern {
	case "openai":
		logger.Debugf("fetchModelConfiguration: Using static mapping for OpenAI model: %s", modelName)
		return r.fetchFromStaticMapping(ctx, req, "azure")
	case "google":
		logger.Debugf("fetchModelConfiguration: Using static mapping for Google model: %s", modelName)
		return r.fetchFromStaticMapping(ctx, req, "gcp")
	case "anthropic", "meta", "amazon":
		// Check if we should use infra models (MAPPING_ENDPOINT)
		useGenAiInfra := cntx.IsUseGenAiInfraModels(ctx)
		logger.Debugf("fetchModelConfiguration: Provider=%s, Model=%s, UseGenAiInfra=%v", routePattern, modelName, useGenAiInfra)

		if useGenAiInfra {
			logger.Debugf("fetchModelConfiguration: Using MAPPING_ENDPOINT for model: %s", modelName)
			return r.fetchFromMappingEndpoint(ctx, req)
		}
		// TODO: Remove bedrock static mapping
		logger.Debugf("fetchModelConfiguration: Falling back to static mapping for Bedrock model: %s", modelName)
		return r.fetchFromStaticMapping(ctx, req, "bedrock")
	case "buddies":
		return r.fetchBuddyConfiguration(ctx, req)
	default:
		// Local endpoints don't need configuration
		return nil
	}
}

// fetchFromStaticMapping fetches model configuration from CONFIGURATION_FILE
func (r *TargetResolver) fetchFromStaticMapping(ctx context.Context, req *ResolutionRequest, infra string) error {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	modelName, ok := req.Metadata["modelName"].(string)
	if !ok || modelName == "" {
		logger.Errorf("fetchFromStaticMapping: model name not found in metadata")
		return NewResolutionError("fetchFromStaticMapping", "model name not found", "")
	}

	if r.staticMapping == nil {
		logger.Errorf("fetchFromStaticMapping: static mapping is nil")
		return NewResolutionError("fetchFromStaticMapping", "static mapping not loaded", "")
	}

	logger.Debugf("fetchFromStaticMapping: Looking for model '%s' (infra=%s) in static mapping with %d models", modelName, infra, len(r.staticMapping.Models))

	// For Azure, check private models first
	if infra == "azure" {
		logger.Debugf("fetchFromStaticMapping: Checking private models for Azure model '%s'", modelName)
		privateModel, found, err := r.checkPrivateModels(ctx, modelName)
		if err == nil && found && privateModel != nil {
			logger.Debugf("fetchFromStaticMapping: Found private model '%s'", modelName)
			req.Metadata["modelConfig"] = privateModel
			req.Metadata["isPrivateModel"] = true
			return nil
		}
		logger.Debugf("fetchFromStaticMapping: No private model found, searching in static mapping")
	}

	// Search in static mapping
	model, found := findModelInMapping(r.staticMapping, modelName)
	if !found {
		// Log all available model names for debugging
		availableModels := make([]string, 0, len(r.staticMapping.Models))
		for _, m := range r.staticMapping.Models {
			availableModels = append(availableModels, m.Name)
		}
		logger.Errorf("fetchFromStaticMapping: Model '%s' not found in static mapping. Available models: %v", modelName, availableModels)
		return NewResolutionError("fetchFromStaticMapping", "model not found", modelName)
	}

	logger.Debugf("fetchFromStaticMapping: Found model '%s' with redirectURL=%s", model.Name, model.RedirectURL)
	req.Metadata["modelConfig"] = model
	req.Metadata["isPrivateModel"] = false

	return nil
}

// fetchFromMappingEndpoint fetches model configuration from MAPPING_ENDPOINT for AWS Bedrock
func (r *TargetResolver) fetchFromMappingEndpoint(ctx context.Context, req *ResolutionRequest) error {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	if r.mappingClient == nil {
		logger.Errorf("fetchFromMappingEndpoint: mapping client not initialized")
		return NewResolutionError("fetchFromMappingEndpoint", "mapping client not initialized", "")
	}

	modelName, ok := req.Metadata["modelName"].(string)
	if !ok || modelName == "" {
		logger.Errorf("fetchFromMappingEndpoint: model name not found in metadata")
		return NewResolutionError("fetchFromMappingEndpoint", "model name not found", "")
	}

	// Fetch models from mapping endpoint
	logger.Debugf("fetchFromMappingEndpoint: Fetching models from MAPPING_ENDPOINT for model: %s", modelName)
	models, err := r.mappingClient.GetModels(ctx)
	if err != nil {
		logger.Errorf("fetchFromMappingEndpoint: Failed to fetch models from endpoint: %v", err)
		return NewResolutionError("fetchFromMappingEndpoint", "failed to fetch models", err.Error())
	}

	logger.Debugf("fetchFromMappingEndpoint: Retrieved %d models from MAPPING_ENDPOINT", len(models))

	// Extract targetApi from operation path (e.g., /converse, /invoke)
	operation, _ := req.Metadata["operation"].(string)
	targetApi := extractTargetApi(operation)

	logger.Debugf("fetchFromMappingEndpoint: Looking for model=%s with targetApi=%s", modelName, targetApi)

	// Use FindBestMatch logic to find matching model
	found, matchedModel := infra.FindBestMatch(ctx, models, modelName, targetApi)
	if !found || matchedModel == nil {
		logger.Warnf("fetchFromMappingEndpoint: No matching model found for name=%s, targetApi=%s among %d available models",
			modelName, targetApi, len(models))

		// Log available model mappings for troubleshooting
		availableMappings := make([]string, 0, len(models))
		for _, m := range models {
			availableMappings = append(availableMappings, m.ModelMapping)
		}
		logger.Debugf("fetchFromMappingEndpoint: Available model mappings: %v", availableMappings)

		return NewResolutionError("fetchFromMappingEndpoint", "no matching model found", modelName)
	}

	logger.Debugf("fetchFromMappingEndpoint: Successfully matched model=%s to ModelMapping=%s, ModelId=%s, TargetApi=%s, UseRegionalInferenceProfile=%v",
		modelName, matchedModel.ModelMapping, matchedModel.ModelId, matchedModel.TargetApi, matchedModel.UseRegionalInferenceProfile)

	req.Metadata["infraConfig"] = matchedModel
	return nil
}

// fetchBuddyConfiguration fetches buddy configuration from CONFIGURATION_FILE
func (r *TargetResolver) fetchBuddyConfiguration(ctx context.Context, req *ResolutionRequest) error {
	buddyID, ok := req.Metadata["buddyId"].(string)
	if !ok || buddyID == "" {
		return NewResolutionError("fetchBuddyConfiguration", "buddy ID not found", "")
	}

	if r.staticMapping == nil {
		return NewResolutionError("fetchBuddyConfiguration", "static mapping not loaded", "")
	}

	buddy, found := findBuddyInMapping(r.staticMapping, buddyID)
	if !found {
		return NewResolutionError("fetchBuddyConfiguration", "buddy not found", buddyID)
	}

	req.Metadata["buddyConfig"] = buddy
	return nil
}

// enrichWithInfrastructure extracts infrastructure, provider, and creator information
func (r *TargetResolver) enrichWithInfrastructure(ctx context.Context, req *ResolutionRequest) error {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	// Check if we have infra config (from MAPPING_ENDPOINT)
	if infraConfig, ok := req.Metadata["infraConfig"].(*infra.ModelConfig); ok {
		// Extract infrastructure from endpoint URL hostname
		req.Target.Infrastructure = extractInfrastructureFromEndpoint(infraConfig.Endpoint)
		// Extract provider from endpoint URL
		req.Target.Provider = extractProviderFromEndpoint(infraConfig.Endpoint)
		// Extract creator from ModelId (e.g., "anthropic.claude-3-5-sonnet..." -> "anthropic")
		req.Target.Creator = types.Creator(extractCreatorFromModelId(infraConfig.ModelId))

		// Fallback: If endpoint extraction failed (e.g., localhost), try to infer from ModelId
		if req.Target.Infrastructure == "" || req.Target.Provider == "" {
			infra, provider := inferInfrastructureFromModelId(infraConfig.ModelId)
			if req.Target.Infrastructure == "" {
				req.Target.Infrastructure = infra
			}
			if req.Target.Provider == "" {
				req.Target.Provider = provider
			}
			logger.Debugf("enrichWithInfrastructure: Inferred from ModelId: Infrastructure=%s, Provider=%s",
				req.Target.Infrastructure, req.Target.Provider)
		}

		logger.Debugf("enrichWithInfrastructure: From MAPPING_ENDPOINT: Infrastructure=%s, Provider=%s, Creator=%s, ModelId=%s",
			req.Target.Infrastructure, req.Target.Provider, req.Target.Creator, infraConfig.ModelId)

		// MAPPING_ENDPOINT models are fully resolved, no need for StaticTargetsByModelName
		return nil
	}

	// Check if we have model config (from CONFIGURATION_FILE)
	if modelConfig, ok := req.Metadata["modelConfig"].(*api.Model); ok {
		req.Target.Infrastructure = types.Infrastructure(modelConfig.Infrastructure)
		req.Target.Provider = types.Provider(modelConfig.Provider)
		req.Target.Creator = types.Creator(modelConfig.Creator)

		// Normalize provider for legacy mapping files
		// Legacy Azure OpenAI models may have provider="openai" instead of "azure"
		req.Target.Provider = normalizeProvider(req.Target.Infrastructure, req.Target.Provider)

		// Extract creator from ModelId if not provided in mapping file (for AWS Bedrock models only)
		if req.Target.Creator == "" && req.Target.Infrastructure == types.InfrastructureAWS && modelConfig.ModelId != "" {
			req.Target.Creator = types.Creator(extractCreatorFromModelId(modelConfig.ModelId))
		}

		// Enrich with static info if any fields are missing (only for Azure/GCP models)
		modelName, _ := req.Metadata["modelName"].(string)
		enrichFromStaticInfo(req.Target, modelName)
		return nil
	}

	// Buddies don't have infrastructure info
	if req.Target.TargetType == TargetTypeBuddy {
		return nil
	}

	// Local/unknown endpoints don't have infrastructure info
	if req.Target.TargetType == TargetTypeUnknown {
		return nil
	}

	return nil
}

// enrichWithModelMetadata extracts model name, version, and ID
func (r *TargetResolver) enrichWithModelMetadata(ctx context.Context, req *ResolutionRequest) error {
	logger := cntx.LoggerFromContext(ctx).Sugar()

	// Get original model name from URL (may be an alias)
	originalModelName, _ := req.Metadata["modelName"].(string)

	// Check if we have infra config (from MAPPING_ENDPOINT)
	if infraConfig, ok := req.Metadata["infraConfig"].(*infra.ModelConfig); ok {
		req.Target.OriginalModelName = originalModelName
		req.Target.ModelName = infraConfig.ModelMapping
		req.Target.ModelID = infraConfig.ModelId
		req.Target.ModelVersion = extractVersion(infraConfig.ModelId)
		req.Target.IsFromMappingEndpoint = true // Mark as resolved from MAPPING_ENDPOINT

		logger.Debugf("enrichWithModelMetadata: From MAPPING_ENDPOINT: OriginalModelName=%s, ModelName=%s, ModelID=%s, ModelVersion=%s, IsFromMappingEndpoint=%v",
			req.Target.OriginalModelName, req.Target.ModelName, req.Target.ModelID, req.Target.ModelVersion, req.Target.IsFromMappingEndpoint)

		return nil
	}

	// Check if we have model config (from CONFIGURATION_FILE)
	if modelConfig, ok := req.Metadata["modelConfig"].(*api.Model); ok {
		req.Target.OriginalModelName = originalModelName
		req.Target.ModelName = modelConfig.Name
		req.Target.ModelID = modelConfig.ModelId
		req.Target.ModelVersion = extractVersion(modelConfig.ModelId)
		req.Target.IsFromMappingEndpoint = false // Mark as resolved from static mapping

		// Enrich with static info to resolve aliases to canonical names
		// This will override ModelName if it's an alias
		enrichFromStaticInfo(req.Target, originalModelName)
		return nil
	}

	return nil
}

// constructTargetURL builds the final target URL based on target type and configuration
func (r *TargetResolver) constructTargetURL(ctx context.Context, req *ResolutionRequest) error {
	switch req.Target.TargetType {
	case TargetTypeLLM:
		return r.constructLLMTargetURL(ctx, req)
	case TargetTypeBuddy:
		return r.constructBuddyTargetURL(ctx, req)
	case TargetTypeUnknown:
		// Local/unknown endpoints don't have target URL
		req.Target.TargetURL = ""
		return nil
	default:
		return NewResolutionError("constructTargetURL", "cannot construct URL for target type", string(req.Target.TargetType))
	}
}

// constructLLMTargetURL constructs target URL for LLM endpoints
func (r *TargetResolver) constructLLMTargetURL(ctx context.Context, req *ResolutionRequest) error {
	operation, _ := req.Metadata["operation"].(string)
	rawQuery, _ := req.Metadata["rawQuery"].(string)

	var redirectURL string

	// Get redirect URL from configuration
	if infraConfig, ok := req.Metadata["infraConfig"].(*infra.ModelConfig); ok {
		// For infra models, construct URL from Endpoint + Path
		redirectURL = infraConfig.Endpoint
		if infraConfig.Path != "" {
			redirectURL += infraConfig.Path
		}
	} else if modelConfig, ok := req.Metadata["modelConfig"].(*api.Model); ok {
		redirectURL = modelConfig.RedirectURL
	} else {
		return NewResolutionError("constructLLMTargetURL", "no configuration found", "")
	}

	// Build target URL
	targetURL := redirectURL + operation

	// Add query parameters if present
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	req.Target.TargetURL = targetURL
	return nil
}

// constructBuddyTargetURL constructs target URL for Buddy endpoints
func (r *TargetResolver) constructBuddyTargetURL(ctx context.Context, req *ResolutionRequest) error {
	buddyConfig, ok := req.Metadata["buddyConfig"].(*api.Buddy)
	if !ok {
		return NewResolutionError("constructBuddyTargetURL", "buddy configuration not found", "")
	}

	operation, _ := req.Metadata["operation"].(string)
	rawQuery, _ := req.Metadata["rawQuery"].(string)

	// Build target URL
	targetURL := buddyConfig.RedirectURL + operation

	// Add query parameters if present
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	req.Target.TargetURL = targetURL
	return nil
}

// Helper functions

// extractRoutePattern extracts the route pattern from the path
// Examples:
//
//	/openai/deployments/gpt-4o/... -> "openai"
//	/anthropic/deployments/claude/... -> "anthropic"
//	/buddies/selfstudybuddy/... -> "buddies"
//	/models -> "models"
func extractRoutePattern(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	firstPart := parts[0]

	// Check for version prefix in buddies path: /v1/{isolationId}/buddies/...
	if firstPart == "v1" && len(parts) >= 3 && parts[2] == "buddies" {
		return "buddies"
	}

	// Check for known patterns
	switch firstPart {
	case "openai", "anthropic", "meta", "amazon", "google", "buddies", "models", "swagger", "health":
		return firstPart
	default:
		return ""
	}
}

// extractModelName extracts model name from deployment path
// Pattern: /{provider}/deployments/{modelName}/...
func extractModelName(path string) string {
	// Pattern: /{provider}/deployments/{modelName}/...
	re := regexp.MustCompile(`^/[^/]+/deployments/([^/]+)`)
	matches := re.FindStringSubmatch(path)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractBuddyInfo extracts buddy ID and isolation ID from path
// Pattern: /v1/{isolationId}/buddies/{buddyId}/...
func extractBuddyInfo(path string) (buddyID, isolationID string) {
	// Pattern: /v1/{isolationId}/buddies/{buddyId}/...
	re := regexp.MustCompile(`^/v1/([^/]+)/buddies/([^/]+)`)
	matches := re.FindStringSubmatch(path)
	if len(matches) > 2 {
		return matches[2], matches[1]
	}

	// Alternative pattern without version: /buddies/{buddyId}/...
	re2 := regexp.MustCompile(`^/buddies/([^/]+)`)
	matches2 := re2.FindStringSubmatch(path)
	if len(matches2) > 1 {
		return matches2[1], ""
	}

	return "", ""
}

// extractOperationPath extracts the operation path (everything after model/buddy identifier)
// Examples:
//
//	/openai/deployments/gpt-4o/chat/completions -> /chat/completions
//	/v1/tenant123/buddies/selfstudybuddy/question -> /question
func extractOperationPath(fullPath, routePattern string) string {
	if routePattern == "buddies" {
		// Pattern: /v1/{isolationId}/buddies/{buddyId}/{operation}
		re := regexp.MustCompile(`^/v1/[^/]+/buddies/[^/]+(/.*)?$`)
		matches := re.FindStringSubmatch(fullPath)
		if len(matches) > 1 {
			return matches[1]
		}

		// Alternative pattern: /buddies/{buddyId}/{operation}
		re2 := regexp.MustCompile(`^/buddies/[^/]+(/.*)?$`)
		matches2 := re2.FindStringSubmatch(fullPath)
		if len(matches2) > 1 {
			return matches2[1]
		}
	} else if routePattern != "" && routePattern != "models" && routePattern != "swagger" && routePattern != "health" {
		// Pattern: /{provider}/deployments/{modelName}/{operation}
		re := regexp.MustCompile(`^/[^/]+/deployments/[^/]+(/.*)?$`)
		matches := re.FindStringSubmatch(fullPath)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// extractVersion extracts version information from model ID for registry lookup
// This is a router function that delegates to provider-specific version extraction
func extractVersion(modelID string) string {
	if modelID == "" {
		return ""
	}

	// Remove regional prefix if present (us., eu., etc.) to normalize the ID
	normalizedID := removeRegionalPrefix(modelID)

	// Try AWS Bedrock format first (has most specific patterns)
	if version := extractVersionBedrock(normalizedID); version != "" {
		return version
	}

	// Try Azure OpenAI format
	if version := extractVersionAzure(normalizedID); version != "" {
		return version
	}

	// Try GCP Vertex format
	if version := extractVersionVertex(normalizedID); version != "" {
		return version
	}

	return ""
}

// extractVersionBedrock extracts version from AWS Bedrock model IDs
// Bedrock models use version patterns like "-v1", "-v2" before a colon
// Examples:
//
//	"anthropic.claude-3-5-sonnet-20241022-v2:0" -> "v2"
//	"anthropic.claude-3-7-sonnet-20250219-v1:0" -> "v1"
//	"amazon.titan-embed-text-v2:0" -> "v2"
func extractVersionBedrock(modelID string) string {
	if modelID == "" {
		return ""
	}

	// Pattern: version with v prefix before colon (e.g., -v2:0, -v1:0)
	versionPattern := regexp.MustCompile(`[-]v(\d+)(?::|$)`)
	if matches := versionPattern.FindStringSubmatch(modelID); len(matches) > 1 {
		return "v" + matches[1]
	}

	return ""
}

// extractVersionAzure extracts version from Azure OpenAI model IDs
// Azure models use date-based versions
// Examples:
//
//	"gpt-4o-2024-11-20" -> "2024-11-20"
//	"gpt-35-turbo-0613" -> "0613"
//	"dall-e-3" -> "" (no version)
func extractVersionAzure(modelID string) string {
	if modelID == "" {
		return ""
	}

	// Pattern: Full date format (YYYY-MM-DD)
	datePattern := regexp.MustCompile(`(\d{4}[-]\d{2}[-]\d{2})$`)
	if matches := datePattern.FindStringSubmatch(modelID); len(matches) > 1 {
		return matches[1]
	}

	// Pattern: Short version format (e.g., -0613, -1106)
	shortVersionPattern := regexp.MustCompile(`[-](\d{4})$`)
	if matches := shortVersionPattern.FindStringSubmatch(modelID); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// extractVersionVertex extracts version from GCP Vertex AI model IDs
// Vertex models use numeric or compound versions
// Examples:
//
//	"gemini-1.5-pro-002" -> "002"
//	"imagen-3.0-generate-001" -> "generate-001"
//	"text-multilingual-embedding-002" -> "002"
func extractVersionVertex(modelID string) string {
	if modelID == "" {
		return ""
	}

	// Pattern: Compound version (e.g., -generate-001)
	compoundVersionPattern := regexp.MustCompile(`[-](generate-\d{3})$`)
	if matches := compoundVersionPattern.FindStringSubmatch(modelID); len(matches) > 1 {
		return matches[1]
	}

	// Pattern: Numeric version at the end (e.g., -002)
	numericVersionPattern := regexp.MustCompile(`[-](\d{3})$`)
	if matches := numericVersionPattern.FindStringSubmatch(modelID); len(matches) > 1 {
		return matches[1]
	}

	// Pattern: Version with v prefix (e.g., -v1)
	versionPattern := regexp.MustCompile(`[-]v(\d+)$`)
	if matches := versionPattern.FindStringSubmatch(modelID); len(matches) > 1 {
		return "v" + matches[1]
	}

	return ""
}

// removeRegionalPrefix removes regional prefixes from AWS model IDs
// Examples:
//
//	"us.anthropic.claude-3-7-sonnet-20250219-v1:0" -> "anthropic.claude-3-7-sonnet-20250219-v1:0"
//	"eu.amazon.titan-text-v1:0" -> "amazon.titan-text-v1:0"
func removeRegionalPrefix(modelID string) string {
	if modelID == "" {
		return ""
	}

	// Check for regional prefix pattern (two lowercase letters followed by a dot)
	regionalPrefixPattern := regexp.MustCompile(`^[a-z]{2}\.`)
	if regionalPrefixPattern.MatchString(modelID) {
		// Remove the prefix (first 3 characters: "xx.")
		return modelID[3:]
	}

	return modelID
}

// extractTargetApi extracts the target API from the operation path
// and maps OpenAI-style paths to Bedrock API names
// Examples:
//   - /converse -> converse
//   - /invoke -> invoke
//   - /chat/completions -> converse (OpenAI-style mapped to Bedrock API)
//   - /embeddings -> invoke (OpenAI-style mapped to Bedrock API)
func extractTargetApi(operation string) string {
	if operation == "" {
		return ""
	}

	parts := strings.Split(strings.TrimPrefix(operation, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	apiPath := parts[0]

	// Map OpenAI-style API paths to Bedrock API names
	switch apiPath {
	case "chat":
		// /chat/completions -> converse
		return "converse"
	case "embeddings":
		// /embeddings -> invoke
		return "invoke"
	default:
		// Return as-is for native Bedrock paths (converse, invoke, etc.)
		return apiPath
	}
}

// extractCreatorFromModelId extracts the creator/vendor from AWS Bedrock model ID
// Examples:
//
//	"anthropic.claude-3-5-sonnet-20241022-v2:0" -> "anthropic"
//	"amazon.titan-text-express-v1" -> "amazon"
//	"meta.llama3-70b-instruct-v1:0" -> "meta"
//	"us.anthropic.claude-3-5-sonnet-20241022-v2:0" -> "anthropic"
func extractCreatorFromModelId(modelId string) string {
	if modelId == "" {
		return ""
	}

	// Remove regional prefix first (e.g., "us.", "eu.") before extracting creator
	normalizedId := removeRegionalPrefix(modelId)

	// Bedrock model IDs typically follow pattern: {creator}.{model-name}...
	parts := strings.SplitN(normalizedId, ".", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// extractInfrastructureFromEndpoint extracts infrastructure from endpoint URL hostname
// Examples:
//   - "https://bedrock-runtime.us-east-1.amazonaws.com" -> "aws"
//   - "https://vertex-ai.googleapis.com" -> "gcp"
//   - "https://something.azure-api.net" -> "azure"
func extractInfrastructureFromEndpoint(endpoint string) types.Infrastructure {
	if endpoint == "" {
		return ""
	}

	endpointLower := strings.ToLower(endpoint)

	// Check for AWS endpoints (amazonaws.com)
	if strings.Contains(endpointLower, "amazonaws.com") {
		return types.InfrastructureAWS
	}

	// Check for GCP endpoints (googleapis.com)
	if strings.Contains(endpointLower, "googleapis.com") {
		return types.InfrastructureGCP
	}

	// Check for Azure endpoints (azure, windows.net)
	if strings.Contains(endpointLower, "azure") || strings.Contains(endpointLower, "windows.net") {
		return types.InfrastructureAzure
	}

	// Default to empty if cannot determine
	return ""
}

// extractProviderFromEndpoint extracts provider from endpoint URL
// Examples:
//   - "https://bedrock-runtime.us-east-1.amazonaws.com" -> "bedrock"
//   - "https://vertex-ai.googleapis.com" -> "vertex"
//   - "https://something.openai.azure.com" -> "azure"
func extractProviderFromEndpoint(endpoint string) types.Provider {
	if endpoint == "" {
		return ""
	}

	endpointLower := strings.ToLower(endpoint)

	// Check for AWS Bedrock (bedrock-runtime or bedrock)
	if strings.Contains(endpointLower, "bedrock-runtime") || strings.Contains(endpointLower, "bedrock") {
		return types.ProviderBedrock
	}

	// Check for GCP Vertex AI
	if strings.Contains(endpointLower, "vertex") || strings.Contains(endpointLower, "aiplatform.googleapis.com") {
		return types.ProviderVertex
	}

	// Check for Azure
	if strings.Contains(endpointLower, "azure") {
		return types.ProviderAzure
	}

	// Default to empty if cannot determine
	return ""
}

// normalizeProvider normalizes provider values from legacy mapping files
// This handles cases where legacy configuration files may have incorrect provider values
// that don't match the model registry's expected values.
// For example, Azure OpenAI models in legacy mapping files may have provider="openai"
// but the model registry expects provider="azure"
// Similarly, AWS Bedrock models may have provider="amazon", "anthropic", or "meta"
// but the model registry expects provider="bedrock"
func normalizeProvider(infrastructure types.Infrastructure, provider types.Provider) types.Provider {
	// Normalize Azure OpenAI models: if infrastructure is "azure" but provider is "openai",
	// correct it to "azure" to match the model registry
	if infrastructure == types.InfrastructureAzure && provider == "openai" {
		return types.ProviderAzure
	}

	// Normalize AWS Bedrock models: if infrastructure is "aws" but provider is a creator
	// (amazon, anthropic, meta), correct it to "bedrock" to match the model registry
	if infrastructure == types.InfrastructureAWS {
		if provider == "amazon" || provider == "anthropic" || provider == "meta" {
			return types.ProviderBedrock
		}
	}

	// Return provider as-is for all other cases
	return provider
}

func isGPT4oModel(modelName string) bool {
	switch modelName {
	case "gpt-4o":
		return true
	default:
		return false
	}
}

// inferInfrastructureFromModelId attempts to infer infrastructure and provider from AWS Bedrock model ID
// This is useful when the endpoint URL doesn't provide enough information (e.g., localhost in tests)
// Examples:
//   - "us.anthropic.claude-3-7-sonnet-20250219-v1:0" -> (aws, bedrock) - regional inference profile
//   - "anthropic.claude-3-5-sonnet-20241022-v2:0" -> (aws, bedrock) - standard model
//   - "amazon.titan-embed-text-v2:0" -> (aws, bedrock) - standard model
func inferInfrastructureFromModelId(modelId string) (types.Infrastructure, types.Provider) {
	if modelId == "" {
		return "", ""
	}

	// Check for AWS Bedrock model ID patterns:
	// 1. Regional inference profile: "us.anthropic.claude-...", "eu.amazon.titan-..."
	// 2. Standard format: "anthropic.claude-...", "amazon.titan-...", "meta.llama-..."

	// Known AWS Bedrock creators
	bedrockCreators := map[string]bool{
		"anthropic": true,
		"amazon":    true,
		"meta":      true,
		"cohere":    true,
		"ai21":      true,
		"mistral":   true,
	}

	modelIdLower := strings.ToLower(modelId)

	// Check for regional prefix pattern (e.g., "us.", "eu.")
	regionalPrefixPattern := regexp.MustCompile(`^[a-z]{2}\.`)
	if regionalPrefixPattern.MatchString(modelIdLower) {
		// This is a regional inference profile - definitely AWS Bedrock
		return types.InfrastructureAWS, types.ProviderBedrock
	}

	// Check if model ID starts with a known Bedrock creator
	for creator := range bedrockCreators {
		if strings.HasPrefix(modelIdLower, creator+".") {
			return types.InfrastructureAWS, types.ProviderBedrock
		}
	}

	// Unable to infer
	return "", ""
}
