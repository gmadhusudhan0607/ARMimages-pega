/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package collections

const (
	serviceName = "genai-vector-store"
)

type Collection struct {
	ID                      string `json:"id" binding:"required"`
	DefaultEmbeddingProfile string `json:"defaultEmbeddingProfile" binding:"required"`
	DocumentsTotal          int    `json:"documentsTotal" binding:"required"`
}
