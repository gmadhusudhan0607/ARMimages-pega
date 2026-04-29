/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package google

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"go.uber.org/zap"
)

const (
	modelID      = "text-multilingual-embedding-002"
	modelVersion = "2"
)

// GoogleEmbedder implements TextEmbedder for Google multilingual models
type GoogleEmbedder struct {
	*embedders.EmbedderBase
	processor embedders.EmbeddingProcessor
}

// GoogleProcessor handles Google-specific request/response processing
type GoogleProcessor struct {
	base *embedders.EmbedderBase
}

var _ embedders.TextEmbedder = (*GoogleEmbedder)(nil)

// NewGoogleEmbedder creates a new Google embedder instance
func NewGoogleEmbedder(uri string, httpHeaders map[string]string, cfg http_client.HTTPClientConfig, logger *zap.Logger) (embedders.TextEmbedder, error) {
	baseEmbedder, err := embedders.NewEmbedderBase(uri, httpHeaders, cfg, modelID, modelVersion, "google", logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base embedder: %w", err)
	}

	processor := &GoogleProcessor{base: baseEmbedder}

	return &GoogleEmbedder{
		EmbedderBase: baseEmbedder,
		processor:    processor,
	}, nil
}

func (c *GoogleEmbedder) GetEmbedding(ctx context.Context, chunk string) ([]float32, int, error) {
	return c.EmbedderBase.GetEmbeddingWithProcessor(ctx, chunk, c.processor)
}

func (p *GoogleProcessor) CreateRequest(ctx context.Context, chunk string) (*http.Request, error) {
	requestBody := embeddingRequest{
		Model: modelID,
		Texts: []string{chunk},
	}
	return p.base.CreateJSONRequest(ctx, http.MethodPost, requestBody)
}

func (p *GoogleProcessor) ProcessResponse(resp *http.Response) ([]float32, error) {
	var response embeddingResponse
	if err := p.base.UnmarshalJSONResponse(resp, &response); err != nil {
		return nil, err
	}

	if len(response.Embedding) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	if len(response.Embedding) > 1 {
		return nil, fmt.Errorf("embeddings count > 1, case not yet handled")
	}

	return response.Embedding[0].Values, nil
}
