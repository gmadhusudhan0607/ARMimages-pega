/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"context"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/models/types"
	"github.com/gin-gonic/gin"
)

// Context key for storing ResolvedTarget in gin context
type resolvedTargetContextKeyType string

const ResolvedTargetContextKey resolvedTargetContextKeyType = "resolved_target"

// TargetType represents the type of target endpoint
type TargetType string

const (
	// TargetTypeLLM represents LLM model endpoints (Azure OpenAI, AWS Bedrock, GCP Vertex)
	TargetTypeLLM TargetType = "LLM"
	// TargetTypeBuddy represents Buddy endpoints
	TargetTypeBuddy TargetType = "Buddy"
	// TargetTypeUnknown represents unrecognized endpoint types (local endpoints, unknown routes, etc.)
	TargetTypeUnknown TargetType = "Unknown"
)

// ResolvedTarget represents the complete resolved target information
// This is the primary output of the TargetResolver
type ResolvedTarget struct {
	// Required fields
	TargetURL  string     `json:"targetURL"`
	TargetType TargetType `json:"targetType"`

	// Optional fields - populated based on route type and configuration
	Infrastructure        types.Infrastructure `json:"infrastructure,omitempty"`           // azure, gcp, bedrock
	Provider              types.Provider       `json:"provider,omitempty"`                 // Azure, Bedrock, etc.
	Creator               types.Creator        `json:"creator,omitempty"`                  // openai, anthropic, meta, etc.
	OriginalModelName     string               `json:"original-model-name,omitempty"`      // Model name from request (may be an alias)
	ModelName             string               `json:"model-name,omitempty"`               // Canonical model name (aliases resolved)
	ModelVersion          string               `json:"model-version,omitempty"`            // Extracted a version from model ID
	ModelID               string               `json:"model-id,omitempty"`                 // Full model ID/ARN
	IsFromMappingEndpoint bool                 `json:"is-from-mapping-endpoint,omitempty"` // True if resolved from MAPPING_ENDPOINT, false if from static mapping
}

// ResolutionRequest represents the internal working context during resolution
// It accumulates information through the enrichment pipeline stages
type ResolutionRequest struct {
	GinContext *gin.Context
	Target     *ResolvedTarget
	Metadata   map[string]interface{} // Stage-specific metadata for passing data between stages
}

// EnrichmentStage represents a function that processes one stage of the resolution pipeline
// Each stage can read from and write to the ResolutionRequest
type EnrichmentStage func(ctx context.Context, req *ResolutionRequest) error

// ResolutionError represents an error that occurred during target resolution
type ResolutionError struct {
	Stage   string // Which stage failed
	Reason  string // Why it failed
	Details string // Additional context
}

// Error implements the error interface
func (e *ResolutionError) Error() string {
	if e.Details != "" {
		return e.Stage + ": " + e.Reason + " (" + e.Details + ")"
	}
	return e.Stage + ": " + e.Reason
}

// NewResolutionError creates a new ResolutionError
func NewResolutionError(stage, reason, details string) *ResolutionError {
	return &ResolutionError{
		Stage:   stage,
		Reason:  reason,
		Details: details,
	}
}
