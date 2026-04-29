/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package processors

import (
	"context"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/metadata"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/processors/extensions"
)

// RequestProcessor defines the main interface for processing requests and responses
type RequestProcessor interface {
	ProcessRequest(ctx context.Context, body []byte) (*ProcessedRequest, error)
	ProcessResponse(ctx context.Context, resp *http.Response) (*extensions.ProcessedResponse, error)
	UpdateMetrics(metadata *metadata.RequestMetadata, req *ProcessedRequest, resp *extensions.ProcessedResponse) error
}

// ProviderExtension defines provider-specific behavior
type ProviderExtension interface {
	GetConfiguration() extensions.ExtensionConfiguration
	ParseStreamingResponse(responseBody []byte) (*extensions.ProcessedResponse, error)
	ValidateProcessingConfig(config *extensions.ProcessingConfig) error
}

// ProcessedRequest contains the result of request processing
type ProcessedRequest struct {
	ModifiedBody    []byte
	OriginalTokens  *int
	ModifiedTokens  *int
	HasSystemPrompt bool
}
