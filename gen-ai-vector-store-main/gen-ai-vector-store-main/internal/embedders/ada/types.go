/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package ada

type EmbeddingRequest struct {
	Input string `json:"input"`
}

type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []Data `json:"data"`
	Model  string `json:"model"`
	Usage  Usage  `json:"usage"`
}

type Data struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
