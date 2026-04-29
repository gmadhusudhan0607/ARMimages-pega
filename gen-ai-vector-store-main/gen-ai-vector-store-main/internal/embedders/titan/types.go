/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package titan

type EmbeddingRequest struct {
	Input      string `json:"inputText"`
	Dimensions int    `json:"dimensions"`
}

type EmbeddingResponse struct {
	Embedding           []float32        `json:"embedding"`
	EmbeddingsByType    EmbeddingsByType `json:"embeddingsByType"`
	InputTextTokenCount int              `json:"inputTextTokenCount"`
}

type EmbeddingsByType struct {
	Float []float32 `json:"float"`
}
