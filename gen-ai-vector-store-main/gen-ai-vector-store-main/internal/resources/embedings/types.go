/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package embedings

import (
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/filters"
	"github.com/pgvector/pgvector-go"
)

const (
	MetadataKeyStaticEmbeddingAttributes = "staticEmbeddingAttributes"
)

type QueryChunksRequest struct {
	Limit              int                   `json:"limit"`
	MaxDistance        *float64              `json:"maxDistance"`
	RetrieveVector     bool                  `json:"retrieveVector,omitempty"`
	RetrieveAttributes *[]string             `json:"retrieveAttributes,omitempty"`
	Filters            filters.RequestFilter `json:"filters" binding:"required"`
}

type Chunk struct {
	ID         string                `json:"id,omitempty"`
	Content    string                `json:"content"`
	Attributes attributes.Attributes `json:"attributes,omitempty"`
	Embedding  []float32             `json:"embedding,omitempty"`
	Vector     pgvector.Vector       `json:"-"`
	DocumentID string                `json:"documentID"`
	Distance   float64               `json:"distance"`
	Metadata   *ChunkMetadata        `json:"metadata,omitempty"`
}

type Embedding struct {
	ID          string                `json:"id"`
	DocumentID  string                `json:"documentID"`
	Content     string                `json:"content"`
	Attributes  attributes.Attributes `json:"attributes,omitempty"`
	Attributes2 attributes.Attributes `json:"attributes2,omitempty"`
	Vector      pgvector.Vector       `json:"-"`
	Embedding   []float32             `json:"embedding,omitempty"`
}

type ChunkMetadata struct {
	StaticEmbeddingAttributes   []string                     `json:"embeddingAttributes,omitempty"`
	SmartIndexContextAttributes *SmartIndexContextAttributes `json:"-"` // not exposed in API
	SmartAutoResolvedAttributes *SmartAutoResolvedAttributes `json:"-"` // not exposed in API
}

// Keep allign with smart_chunking.IndexAttributes
type SmartIndexContextAttributes struct {
	Attributes *[]string                             `json:"attributes,omitempty"`
	PageRange  *SmartIndexContextAttributesPageRange `json:"pageRange,omitempty"`
}

// Keep allign with smart_chunking.AutoResolvedAttributes
type SmartAutoResolvedAttributes struct {
	Attributes SmartAutoResolvedAttributesItems `json:"attributes,omitempty"`
}

// Keep allign with smart_chunking.AutoResolvedAttributesItems
type SmartAutoResolvedAttributesItems map[string]interface{}

// Keep allign with smart_chunking.PageRange
type SmartIndexContextAttributesPageRange struct {
	First *int `json:"first,omitempty"`
	Last  *int `json:"last,omitempty"`
}
