/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/gin-gonic/gin"
)

// TargetResolver analyzes incoming HTTP requests and resolves comprehensive
// target routing information including targetURL, targetType, infrastructure,
// provider, creator, model metadata, etc.
type TargetResolver struct {
	// Configuration sources
	configFile      string
	staticMapping   *api.Mapping
	privateModelDir string

	// HTTP clients for dynamic configuration
	mappingClient  *MappingClient
	defaultsClient *DefaultsClient

	// Pipeline stages for enrichment
	enrichmentStages []EnrichmentStage

	// Optional caching for infra models
	cacheTTL time.Duration

	// Cache for private models
	privateModelsCache       *api.Mapping
	privateModelsCacheExpiry time.Time
	privateModelsMu          sync.RWMutex
}

// NewTargetResolver creates a new TargetResolver with all configuration sources initialized
func NewTargetResolver(configFile string, mappingEndpoint string, defaultsEndpoint string, privateModelDir string) (*TargetResolver, error) {
	resolver := &TargetResolver{
		configFile:      configFile,
		privateModelDir: privateModelDir,
		cacheTTL:        5 * time.Minute,
	}

	// Load static mapping from CONFIGURATION_FILE
	if configFile != "" {
		mapping, err := loadStaticMapping(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load static mapping: %w", err)
		}
		resolver.staticMapping = mapping
	}

	// Initialize HTTP clients for dynamic configuration
	if mappingEndpoint != "" {
		resolver.mappingClient = NewMappingClient(mappingEndpoint)
	}

	if defaultsEndpoint != "" {
		resolver.defaultsClient = NewDefaultsClient(defaultsEndpoint)
	}

	// Initialize enrichment pipeline stages
	resolver.enrichmentStages = []EnrichmentStage{
		resolver.extractBasicInfo,
		resolver.determineTargetType,
		resolver.fetchModelConfiguration,
		resolver.enrichWithInfrastructure,
		resolver.enrichWithModelMetadata,
		resolver.constructTargetURL,
	}

	return resolver, nil
}

// Resolve performs the complete target resolution process
// This is the main entry point for resolving a request
func (r *TargetResolver) Resolve(ctx context.Context, c *gin.Context) (*ResolvedTarget, error) {
	// Add gin context to ctx so it's available throughout the resolution pipeline
	// This allows access to request headers (including JWT tokens) in downstream functions
	ctx = cntx.ContextWithGinContext(ctx, c)

	// Create resolution request context
	req := &ResolutionRequest{
		GinContext: c,
		Target: &ResolvedTarget{
			TargetType: TargetTypeUnknown,
		},
		Metadata: make(map[string]interface{}),
	}

	if err := r.executeBasicStages(ctx, req); err != nil {
		return nil, err
	}

	// Early return for Unknown types (local endpoints like /models, /swagger, /health)
	// These endpoints don't need configuration fetching, enrichment, or URL construction
	if req.Target.TargetType == TargetTypeUnknown {
		return req.Target, nil
	}

	// Stage 3: Fetch model or buddy configuration
	if err := r.fetchModelConfiguration(ctx, req); err != nil {
		return nil, err
	}

	if r.hasConfiguration(req) {
		if err := r.executeEnrichmentStages(ctx, req); err != nil {
			return nil, err
		}
	}

	// Stage 6: Construct target URL
	if err := r.constructTargetURL(ctx, req); err != nil {
		return nil, err
	}

	return req.Target, nil
}

// executeBasicStages runs the fundamental stages that all requests must go through
func (r *TargetResolver) executeBasicStages(ctx context.Context, req *ResolutionRequest) error {
	// Extract basic information from request (always required)
	if err := r.extractBasicInfo(ctx, req); err != nil {
		return err
	}

	// Determine target type (always required)
	if err := r.determineTargetType(ctx, req); err != nil {
		return err
	}

	return nil
}

// hasConfiguration checks if the request has any configuration to enrich with
func (r *TargetResolver) hasConfiguration(req *ResolutionRequest) bool {
	return req.Metadata["modelConfig"] != nil ||
		req.Metadata["infraConfig"] != nil ||
		req.Metadata["buddyConfig"] != nil
}

// executeEnrichmentStages runs all enrichment stages for configured models
func (r *TargetResolver) executeEnrichmentStages(ctx context.Context, req *ResolutionRequest) error {
	// Enrich with infrastructure information
	if err := r.enrichWithInfrastructure(ctx, req); err != nil {
		return err
	}

	// Trigger lazy initialization for special model types
	r.handleSpecialModelInitialization(ctx, req)

	// Enrich with model metadata (only for LLM endpoints)
	// Buddy endpoints don't have model metadata
	if req.Target.TargetType == TargetTypeLLM {
		if err := r.enrichWithModelMetadata(ctx, req); err != nil {
			return err
		}
	}

	return nil
}

// handleSpecialModelInitialization triggers lazy initialization for models requiring special handling
func (r *TargetResolver) handleSpecialModelInitialization(ctx context.Context, req *ResolutionRequest) {
	// Trigger lazy initialization for gpt-4o version if this is an Azure gpt-4o model
	modelName, ok := req.Metadata["modelName"].(string)
	if !ok {
		return
	}

	if req.Target.Infrastructure == types.InfrastructureAzure && isGPT4oModel(modelName) {
		genaiURL := r.getGenAIURL()
		LazyInitGPT4oVersion(ctx, genaiURL)
	}
}

// GetStaticMapping returns the loaded static mapping configuration
func (r *TargetResolver) GetStaticMapping() *api.Mapping {
	return r.staticMapping
}

// GetMappingClient returns the mapping endpoint client
func (r *TargetResolver) GetMappingClient() *MappingClient {
	return r.mappingClient
}

// GetDefaultsClient returns the defaults endpoint client
func (r *TargetResolver) GetDefaultsClient() *DefaultsClient {
	return r.defaultsClient
}

// getGenAIURL returns the GENAI_URL from environment variables
func (r *TargetResolver) getGenAIURL() string {
	return helpers.GetEnvOrDefault("GENAI_URL", "")
}
