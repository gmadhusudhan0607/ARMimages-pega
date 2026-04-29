/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package google

type embeddingRequest struct {
	Model string   `json:"model"`
	Texts []string `json:"texts"`
}

type embeddingResponse struct {
	Embedding []embeddingResponseEmbedding `json:"embedding"`
	Usage     embeddingResponseUsage       `json:"usage"`
}

type embeddingResponseEmbedding struct {
	Values     []float32                   `json:"values"`
	Statistics embeddingResponseStatistics `json:"statistics"`
}

type embeddingResponseUsage struct {
	PromptTokens float32 `json:"prompt_tokens"`
}

type embeddingResponseStatistics struct {
	TokenCount float32 `json:"token_count"`
	Truncated  bool    `json:"truncated"`
}
