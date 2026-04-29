/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package isolations

import (
	"time"
)

const (
	serviceName           = "genai-vector-store"
	embeddingQueueTableV2 = "vector_store.embedding_queue"
)

type Details struct {
	ID              string    `json:"id" binding:"required"`
	MaxStorageSize  string    `json:"maxStorageSize,omitempty"`
	PDCEndpointURL  string    `json:"pdcEndpointURL,omitempty"`
	CreatedAt       time.Time `json:"createdAt,omitempty"`
	ModifiedAt      time.Time `json:"modifiedAt,omitempty"`
	CollectionNames []string  `json:"collectionNames,omitempty"`
	Error           string    `json:"error,omitempty"`
}

type EmbeddingProfile struct {
	ID           string `json:"profile_id" binding:"required"`
	ProviderName string `json:"providerName,omitempty"`
	ModelName    string `json:"modelName,omitempty"`
	ModelVersion string `json:"modelVersion,omitempty"`
	VectorLen    int    `json:"vectorLen,omitempty"`
	MaxTokens    int    `json:"maxTokens,omitempty"`
}
