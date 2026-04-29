/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/resolvers/target"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models"
	modeltypes "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/config"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metrics"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors"
	processorconfig "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Global TargetResolver instance
var globalTargetResolver *target.TargetResolver

// SetGlobalTargetResolver sets the global TargetResolver instance
func SetGlobalTargetResolver(resolver *target.TargetResolver) {
	globalTargetResolver = resolver
}

// ExtractIsolationIDFromToken extracts the GUID from JWT token in Authorization header
func ExtractIsolationIDFromToken(authHeader string) (string, error) {
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "", fmt.Errorf("authorization header missing Bearer prefix")
	}

	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("empty token after Bearer prefix")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format: expected 3 parts, got %d", len(parts))
	}

	payload := parts[1]
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	if guid, ok := claims["guid"].(string); ok && guid != "" {
		return guid, nil
	}

	if isolationId, ok := claims["isolationId"].(string); ok && isolationId != "" {
		return isolationId, nil
	}

	availableClaims := make([]string, 0, len(claims))
	for key := range claims {
		availableClaims = append(availableClaims, key)
	}

	return "", fmt.Errorf("no valid isolation ID found in JWT claims (available claims: %v)", availableClaims)
}

// injectRequestMetadata handles metadata injection including model detection and metrics setup
func injectRequestMetadata(c *gin.Context) error {
	logger := cntx.LoggerFromContext(c.Request.Context()).Sugar()

	// Extract isolation ID from Authorization header
	authHeader := c.GetHeader("Authorization")
	isolationID, err := ExtractIsolationIDFromToken(authHeader)
	if err != nil {
		logger.Debugf("injectRequestMetadata: Failed to extract isolation ID: %v", err)
	}

	// Use TargetResolver to resolve target information
	var resolvedTarget *target.ResolvedTarget
	var resolveErr error
	if globalTargetResolver != nil {
		resolvedTarget, resolveErr = globalTargetResolver.Resolve(c.Request.Context(), c)
	} else {
		resolveErr = fmt.Errorf("TargetResolver not initialized")
	}

	// Determine original model name for metrics
	originalModelNameForMetrics := determineOriginalModelNameFromResolved(c, resolvedTarget)

	// Get target model from registry if this is an LLM request
	var targetModel *modeltypes.Model
	var modelLookupErr error
	if resolvedTarget != nil && resolvedTarget.TargetType == target.TargetTypeLLM {
		targetModel, modelLookupErr = resolveModelFromTarget(c.Request.Context(), resolvedTarget, logger)
	}

	// Track model recognition metrics
	if resolveErr != nil {
		logger.Debugf("injectRequestMetadata: Target resolution failed: %v", resolveErr)
		metrics.IncrementModelRecognition(isolationID, "unrecognized", originalModelNameForMetrics)
	} else if resolvedTarget != nil && resolvedTarget.TargetType == target.TargetTypeLLM {
		// For LLM targets, determine recognition status
		// Models resolved from MAPPING_ENDPOINT are considered "recognized" even if not in static registry
		// Models from static mapping need to be found in the registry to be "recognized"
		if resolvedTarget.IsFromMappingEndpoint {
			// Models from MAPPING_ENDPOINT are always recognized since they're dynamically fetched
			logger.Debugf("injectRequestMetadata: Target resolved from MAPPING_ENDPOINT - %s", resolvedTarget.ModelName)
			metrics.IncrementModelRecognition(isolationID, "recognized", originalModelNameForMetrics)
		} else if targetModel != nil && modelLookupErr == nil {
			// Static mapping models must be found in registry to be recognized
			logger.Debugf("injectRequestMetadata: Target resolved and model found in registry - %s", resolvedTarget.ModelName)
			metrics.IncrementModelRecognition(isolationID, "recognized", originalModelNameForMetrics)
		} else {
			logger.Debugf("injectRequestMetadata: Target resolved but model not found in registry - %s", resolvedTarget.ModelName)
			metrics.IncrementModelRecognition(isolationID, "unrecognized", originalModelNameForMetrics)
		}
	} else {
		metrics.IncrementModelRecognition(isolationID, "unrecognized", originalModelNameForMetrics)
	}

	// Create RequestMetadata
	md := metadata.RequestMetadata{
		IsolationID:       isolationID,
		TargetModel:       targetModel,
		RequestMetrics:    metrics.NewRequestMetrics(),
		OriginalModelName: originalModelNameForMetrics,
	}

	// Extract maxOutputTokens
	if targetModel != nil {
		if maxTokens := targetModel.GetMaxOutputTokens(); maxTokens != nil {
			md.RequestMetrics.TokenMetrics.Maximum = maxTokens
		}
	}

	// Store ResolvedTarget in context for downstream use
	if resolvedTarget != nil {
		ctx := context.WithValue(c.Request.Context(), target.ResolvedTargetContextKey, resolvedTarget)
		c.Request = c.Request.WithContext(ctx)
	}

	// Inject metadata into context
	ctx := context.WithValue(c.Request.Context(), metrics.RequestMetadataContextKey{}, &md)
	c.Request = c.Request.WithContext(ctx)

	return nil
}

// determineOriginalModelNameFromResolved determines the original model name from resolved target
// This should return the original model name from the request (may be an alias) for metrics tracking
func determineOriginalModelNameFromResolved(c *gin.Context, target *target.ResolvedTarget) string {
	// First try to get the original model name from resolved target (as it appears in the request)
	if target != nil && target.OriginalModelName != "" {
		return target.OriginalModelName
	}
	// Fallback: use URL path parameter
	modelID := c.Param("modelId")
	if modelID != "" {
		return modelID
	}
	return "unknown"
}

// resolveModelFromTarget looks up a model in the registry based on resolved target information
// This function implements a flexible matching strategy to handle variations in model IDs
func resolveModelFromTarget(ctx context.Context, target *target.ResolvedTarget, logger *zap.SugaredLogger) (*modeltypes.Model, error) {
	// Get models registry
	modelsRegistry, err := models.GetGlobalRegistry(ctx)
	if err != nil {
		if logger != nil {
			logger.Debugf("resolveModelFromTarget: Failed to get models registry: %v", err)
		}
		return nil, fmt.Errorf("failed to get models registry: %w", err)
	}

	// Convert string types to typed values
	infrastructure := modeltypes.Infrastructure(target.Infrastructure)
	provider := modeltypes.Provider(target.Provider)
	creator := modeltypes.Creator(target.Creator)

	if logger != nil {
		logger.Debugf("resolveModelFromTarget: Looking up model in registry: Infrastructure=%s, Provider=%s, Creator=%s, ModelName=%s, ModelVersion=%s, ModelID=%s, IsFromMappingEndpoint=%v",
			infrastructure, provider, creator, target.ModelName, target.ModelVersion, target.ModelID, target.IsFromMappingEndpoint)
	}

	// Strategy 1: Try exact match with model name and version from target
	if target.ModelVersion != "" {
		model, err := modelsRegistry.FindModel(
			infrastructure,
			provider,
			creator,
			target.ModelName,
			target.ModelVersion,
		)
		if err == nil && model != nil {
			if logger != nil {
				logger.Debugf("resolveModelFromTarget: Found model with exact match: %s/%s/%s/%s version %s",
					infrastructure, provider, creator, target.ModelName, target.ModelVersion)
			}
			return model, nil
		}
		if logger != nil {
			logger.Debugf("resolveModelFromTarget: Exact match failed: %v", err)
		}
	}

	// Strategy 2: Try pattern matching if modelID is available
	// This supports wildcard patterns in the registry (e.g., "anthropic.claude-3-7-sonnet-*-v1:0")
	if target.ModelID != "" {
		model, err := modelsRegistry.FindModelByIDPattern(
			infrastructure,
			provider,
			creator,
			target.ModelName,
			target.ModelID,
		)
		if err == nil && model != nil {
			if logger != nil {
				logger.Debugf("resolveModelFromTarget: Found model using ID pattern matching: %s/%s/%s/%s (pattern: %s, modelID: %s)",
					infrastructure, provider, creator, target.ModelName, model.KEY, target.ModelID)
			}
			return model, nil
		}
		if logger != nil {
			logger.Debugf("resolveModelFromTarget:: ID pattern matching failed: %v", err)
		}
	}

	// Strategy 3: Try to find latest version with the model name from target
	model, err := modelsRegistry.FindLatestModel(
		infrastructure,
		provider,
		creator,
		target.ModelName,
	)
	if err == nil && model != nil {
		if logger != nil {
			logger.Debugf("resolveModelFromTarget:: Found model using latest version lookup: %s/%s/%s/%s version %s",
				infrastructure, provider, creator, target.ModelName, model.Version)
		}
		return model, nil
	}
	if logger != nil {
		logger.Debugf("resolveModelFromTarget:: Latest version lookup failed: %v", err)
	}

	// Strategy 4: Try to extract base model name from model ID and find latest version
	// This handles cases where the model name in the mapping differs from the registry
	// For example: "claude-3-7-sonnet" in mapping might be "claude-3-sonnet" in registry
	if target.ModelID != "" {
		// Extract base model name from the full model ID
		// This removes regional prefixes, dates, and version suffixes
		baseModelName := extractBaseModelNameFromID(target.ModelID)
		if baseModelName != "" && baseModelName != target.ModelName {
			if logger != nil {
				logger.Debugf("resolveModelFromTarget:: Trying fallback with base model name extracted from ID: %s (original: %s)",
					baseModelName, target.ModelName)
			}
			model, err := modelsRegistry.FindLatestModel(
				infrastructure,
				provider,
				creator,
				baseModelName,
			)
			if err == nil && model != nil {
				if logger != nil {
					logger.Debugf("resolveModelFromTarget:: Found model using base name from ID: %s/%s/%s/%s version %s",
						infrastructure, provider, creator, baseModelName, model.Version)
				}
				return model, nil
			}
			if logger != nil {
				logger.Debugf("resolveModelFromTarget:: Fallback with base name failed: %v", err)
			}
		}
	}

	// All strategies failed
	if logger != nil {
		logger.Debugf("resolveModelFromTarget: All strategies failed - model not found in registry: %s/%s/%s/%s",
			infrastructure, provider, creator, target.ModelName)
	}
	return nil, fmt.Errorf("model not found in registry: %s/%s/%s/%s",
		infrastructure, provider, creator, target.ModelName)
}

// extractBaseModelNameFromID extracts the base model name from a model ID
// This is a helper function that wraps the logic from stages.go
func extractBaseModelNameFromID(modelID string) string {
	if modelID == "" {
		return ""
	}

	// Remove regional prefix (e.g., "us.", "eu.")
	normalized := modelID
	if len(modelID) > 3 && modelID[2] == '.' {
		// Check if first two chars are lowercase letters (regional prefix)
		if (modelID[0] >= 'a' && modelID[0] <= 'z') && (modelID[1] >= 'a' && modelID[1] <= 'z') {
			normalized = modelID[3:]
		}
	}

	// Remove vendor prefix (e.g., "anthropic.", "amazon.")
	if idx := strings.Index(normalized, "."); idx != -1 && idx < len(normalized)-1 {
		normalized = normalized[idx+1:]
	}

	// Remove version suffix after colon (e.g., ":0")
	if idx := strings.Index(normalized, ":"); idx != -1 {
		normalized = normalized[:idx]
	}

	// Remove date pattern and version suffix
	// Pattern: -YYYYMMDD-vX or -YYYY-MM-DD-vX
	dateVersionPattern := `(-\d{4}[-]?\d{2}[-]?\d{2})(-v\d+)?$`
	if matched, _ := regexp.MatchString(dateVersionPattern, normalized); matched {
		re := regexp.MustCompile(dateVersionPattern)
		normalized = re.ReplaceAllString(normalized, "")
	}

	// Remove standalone version suffix (e.g., "-v1", "-v2")
	versionSuffixPattern := `-v\d+$`
	if matched, _ := regexp.MatchString(versionSuffixPattern, normalized); matched {
		re := regexp.MustCompile(versionSuffixPattern)
		normalized = re.ReplaceAllString(normalized, "")
	}

	// Remove numeric suffix (e.g., "-002", "-001")
	numericSuffixPattern := `-\d{3}$`
	if matched, _ := regexp.MatchString(numericSuffixPattern, normalized); matched {
		re := regexp.MustCompile(numericSuffixPattern)
		normalized = re.ReplaceAllString(normalized, "")
	}

	return normalized
}

// setupResponseWriter wraps the response writer with RequestModificationResponseWriter for metrics collection
func setupResponseWriter(c *gin.Context) error {
	logger := cntx.LoggerFromContext(c.Request.Context()).Sugar()

	// Get existing RequestMetadata from context
	_, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
	if err != nil {
		logger.Errorf("setupResponseWriter: RequestMetadata not found in context: %v", err)
		c.AbortWithStatusJSON(500, gin.H{"error": "Request metadata not properly initialized"})
		return err
	}

	// Use RequestModificationResponseWriter directly for all cases
	// Trailer support has been removed as it's not used in the current implementation
	logger.Debugf("setupResponseWriter: Creating RequestModificationResponseWriter")
	customWriter := metrics.NewRequestModificationResponseWriter(c.Writer, c, logger)
	c.Writer = customWriter

	return nil
}

// modifyRequest handles request modification including body processing and processor creation
func modifyRequest(c *gin.Context) error {
	logger := cntx.LoggerFromContext(c.Request.Context()).Sugar()

	// Get RequestMetadata from context
	md, err := metadata.GetRequestMetadataFromContext(c.Request.Context())
	if err != nil {
		logger.Errorf("modifyRequest: RequestMetadata not found in context: %v", err)
		return nil // Continue processing (graceful degradation)
	}

	// Read request body
	body, err := readRequestBody(c, logger)
	if err != nil || body == nil {
		return nil // No body to process or error already handled
	}

	// Detect and store streaming status early
	md.IsStreaming = isStreamingRequestFromBody(body) || isBedrockStreamingApi(c)
	md.RequestMetrics.IsStreaming = md.IsStreaming
	if md.IsStreaming {
		logger.Debug("modifyRequest: Detected streaming request")
	}

	// Check if this is a chat-completion endpoint
	if !shouldModifyEndpoint(c, md, body, logger) {
		return nil
	}

	// Create processor
	processor, err := createRequestProcessor(c, md, body, logger)
	if err != nil {
		return nil // Error already logged and handled
	}

	// Store original request body for retry logic
	ctx := context.WithValue(c.Request.Context(), OriginalRequestBodyKey, body)
	c.Request = c.Request.WithContext(ctx)

	// Process request with processor
	processedRequest, err := processor.ProcessRequest(c.Request.Context(), body)
	if err != nil {
		logger.Errorf("modifyRequest: Failed to process request: %v", err)
		restoreRequestBody(c, body)
		return nil
	}

	// Update request body if modified
	if processedRequest != nil && processedRequest.ModifiedBody != nil {
		c.Request.Body = io.NopCloser(bytes.NewReader(processedRequest.ModifiedBody))
		logger.Debugf("modifyRequest: Request body modified, new size: %d bytes", len(processedRequest.ModifiedBody))
	} else {
		restoreRequestBody(c, body)
	}

	// Update metadata with processing results
	if processedRequest != nil {
		if processedRequest.OriginalTokens != nil {
			originalFloat := float64(*processedRequest.OriginalTokens)
			md.RequestMetrics.TokenMetrics.Requested = &originalFloat
		}
		if processedRequest.ModifiedTokens != nil {
			adjustedFloat := float64(*processedRequest.ModifiedTokens)
			md.RequestMetrics.TokenMetrics.Adjusted = &adjustedFloat
		}
	}

	// Store processor in context for post-processing
	if processor != nil {
		ctx := context.WithValue(c.Request.Context(), ProcessorContextKey, processor)
		c.Request = c.Request.WithContext(ctx)
	}

	return nil
}

func isBedrockStreamingApi(c *gin.Context) bool {
	return strings.Contains(c.Param("targetApi"), "converse-stream") || strings.Contains(c.Param("targetApi"), "invoke-stream")
}

// Helper functions for modifyRequest

func readRequestBody(c *gin.Context, logger *zap.SugaredLogger) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Errorf("modifyRequest: Failed to read request body: %v", err)
		return nil, err
	}
	_ = c.Request.Body.Close()

	return body, nil
}

func shouldModifyEndpoint(c *gin.Context, md *metadata.RequestMetadata, body []byte, logger *zap.SugaredLogger) bool {
	if md.TargetModel == nil {
		return true // Continue to next step
	}

	// Check if path contains chat completion endpoint
	path := c.Request.URL.Path
	if !strings.Contains(path, "chat/completions") && !strings.Contains(path, "messages") {
		if logger != nil {
			logger.Debugf("modifyRequest: Path '%s' does not appear to be chat-completion endpoint, skipping modification", path)
		}
		restoreRequestBody(c, body)
		return false
	}

	return true
}

func createRequestProcessor(c *gin.Context, md *metadata.RequestMetadata, body []byte, logger *zap.SugaredLogger) (processors.RequestProcessor, error) {
	if md.TargetModel == nil {
		logger.Debug("modifyRequest: No target model available, skipping processing")
		restoreRequestBody(c, body)
		return nil, fmt.Errorf("no target model")
	}

	// Load configuration from REQUEST_PROCESSING_* environment variables
	reqConfig, err := config.GetConfig(c.Request.Context())
	if err != nil {
		logger.Errorf("modifyRequest: Failed to load REQ configuration from environment: %v", err)
		return createDefaultProcessor(md, body, c, logger)
	}

	// Skip all processing for DISABLED strategy
	if reqConfig.GetOutputTokensStrategy() == "DISABLED" {
		logger.Debugf("modifyRequest: Strategy is DISABLED, skipping all processing")
		restoreRequestBody(c, body)
		return nil, fmt.Errorf("processing disabled")
	}

	processingConfig := processorconfig.CreateProcessingConfigFromReqConfig(reqConfig)

	// Create processor with strategy if needed
	if reqConfig.GetOutputTokensStrategy() != config.OutputTokensStrategyMonitoringOnly {
		processor, err := processors.NewChatProcessorWithStrategy(c.Request.Context(), md.TargetModel, processingConfig, reqConfig)
		if err != nil {
			logger.Errorf("modifyRequest: Failed to create processor with strategy: %v", err)
			return createDefaultProcessor(md, body, c, logger)
		}
		return processor, nil
	}

	// Create processor with standard configuration
	processor, err := processors.NewProcessor(c.Request.Context(), md.TargetModel, processingConfig)
	if err != nil {
		logger.Errorf("modifyRequest: Failed to create processor with environment config: %v", err)
		return createDefaultProcessor(md, body, c, logger)
	}
	return processor, nil
}

func createDefaultProcessor(md *metadata.RequestMetadata, body []byte, c *gin.Context, logger *zap.SugaredLogger) (processors.RequestProcessor, error) {
	processor, err := processors.NewDefaultProcessor(md.TargetModel)
	if err != nil {
		if logger != nil {
			logger.Errorf("modifyRequest: Failed to create default processor: %v", err)
		}
		restoreRequestBody(c, body)
		return nil, err
	}
	return processor, nil
}

func restoreRequestBody(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
}

func isStreamingRequestFromBody(body []byte) bool {
	// Fast-path: avoid JSON work if the key doesn't even appear.
	if !bytes.Contains(body, []byte(`"stream"`)) {
		return false
	}

	var payload struct {
		Stream bool `json:"stream"`
	}

	// If JSON is invalid or not an object, treat as non-streaming.
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	return payload.Stream
}
