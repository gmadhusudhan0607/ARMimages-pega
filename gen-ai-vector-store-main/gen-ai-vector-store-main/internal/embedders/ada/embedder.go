/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package ada

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"go.uber.org/zap"
)

const (
	embeddingModel        = "text-embedding-ada-002"
	embeddingModelVersion = "2"
)

// AdaEmbedder implements TextEmbedder for OpenAI Ada models
type AdaEmbedder struct {
	*embedders.EmbedderBase
	processor embedders.EmbeddingProcessor
}

// AdaProcessor handles Ada-specific request/response processing
type AdaProcessor struct {
	base *embedders.EmbedderBase
}

// NewAdaEmbedder creates a new Ada embedder instance
func NewAdaEmbedder(
	uri string,
	httpHeaders map[string]string,
	cfg http_client.HTTPClientConfig,
	logger *zap.Logger,
) (embedders.TextEmbedder, error) {
	baseEmbedder, err := embedders.NewEmbedderBase(uri, httpHeaders, cfg, embeddingModel, embeddingModelVersion, "ada", logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base embedder: %w", err)
	}

	processor := &AdaProcessor{base: baseEmbedder}

	return &AdaEmbedder{
		EmbedderBase: baseEmbedder,
		processor:    processor,
	}, nil
}

func (a *AdaEmbedder) GetEmbedding(ctx context.Context, chunk string) ([]float32, int, error) {
	return a.EmbedderBase.GetEmbeddingWithProcessor(ctx, chunk, a.processor)
}

func (p *AdaProcessor) CreateRequest(ctx context.Context, chunk string) (*http.Request, error) {
	requestBody := EmbeddingRequest{Input: chunk}
	return p.base.CreateJSONRequest(ctx, http.MethodPost, requestBody)
}

func (p *AdaProcessor) ProcessResponse(resp *http.Response) ([]float32, error) {
	var response EmbeddingResponse
	if err := p.base.UnmarshalJSONResponse(resp, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	if len(response.Data) > 1 {
		return nil, fmt.Errorf("embeddings count > 1, case not yet handled")
	}

	return response.Data[0].Embedding, nil
}
