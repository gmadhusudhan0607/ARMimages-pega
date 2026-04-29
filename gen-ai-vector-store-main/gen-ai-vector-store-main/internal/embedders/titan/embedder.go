/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package titan

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"go.uber.org/zap"
)

const (
	embeddingModel        = "amazon-titan-embed-text"
	embeddingModelVersion = "2"
)

// TitanEmbedder implements TextEmbedder for Amazon Titan models
type TitanEmbedder struct {
	*embedders.EmbedderBase
	processor embedders.EmbeddingProcessor
	vectorLen int
}

// TitanProcessor handles Titan-specific request/response processing
type TitanProcessor struct {
	base      *embedders.EmbedderBase
	vectorLen int
}

// NewTitanEmbedder creates a new Titan embedder instance
func NewTitanEmbedder(
	uri string,
	vectorLen int,
	httpHeaders map[string]string,
	cfg http_client.HTTPClientConfig,
	logger *zap.Logger,
) (embedders.TextEmbedder, error) {
	// Validate vector length
	if vectorLen != 1024 && vectorLen != 512 && vectorLen != 256 {
		return nil, fmt.Errorf("invalid vector length: %d, must be one of 1024, 512, or 256", vectorLen)
	}

	baseEmbedder, err := embedders.NewEmbedderBase(uri, httpHeaders, cfg, embeddingModel, embeddingModelVersion, "titan", logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base embedder: %w", err)
	}

	processor := &TitanProcessor{
		base:      baseEmbedder,
		vectorLen: vectorLen,
	}

	return &TitanEmbedder{
		EmbedderBase: baseEmbedder,
		processor:    processor,
		vectorLen:    vectorLen,
	}, nil
}

func (t *TitanEmbedder) GetEmbedding(ctx context.Context, chunk string) ([]float32, int, error) {
	return t.EmbedderBase.GetEmbeddingWithProcessor(ctx, chunk, t.processor)
}

func (p *TitanProcessor) CreateRequest(ctx context.Context, chunk string) (*http.Request, error) {
	requestBody := EmbeddingRequest{
		Input:      chunk,
		Dimensions: p.vectorLen,
	}
	return p.base.CreateJSONRequest(ctx, http.MethodPost, requestBody)
}

func (p *TitanProcessor) ProcessResponse(resp *http.Response) ([]float32, error) {
	var response EmbeddingResponse
	if err := p.base.UnmarshalJSONResponse(resp, &response); err != nil {
		return nil, err
	}

	if len(response.Embedding) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return response.Embedding, nil
}
