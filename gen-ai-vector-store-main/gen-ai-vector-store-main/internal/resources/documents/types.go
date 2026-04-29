/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/filters"
)

const (
	OperatorEq = "eq"
	OperatorIn = "in"
)

const (
	MetadataKeyStaticEmbeddingAttributes = "staticEmbeddingAttributes"
)

type PatchDocumentRequest struct {
	Attributes   []attributes.Attribute `json:"attributes,omitempty" binding:"dive"`
	Status       *string                `json:"status,omitempty"`
	ErrorMessage *string                `json:"errorMessage,omitempty"`
}

type DeleteDocumentRequest struct {
	Items []attributes.AttributeFilter
}

type DeleteDocumentByIdRequest struct {
	ID string `json:"id" binding:"required"`
}

type PutDocumentRequest struct {
	ID         string                 `json:"id" binding:"required"`
	Chunks     []embedings.Chunk      `json:"chunks" binding:"required"`
	Attributes []attributes.Attribute `json:"attributes,omitempty" binding:"dive"`
	Metadata   *DocumentMetadata      `json:"metadata,omitempty"`
}

type Document struct {
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

type GetDocumentResponse struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

type RetrieveDocumentQueryRequest struct {
	Limit              int                   `json:"limit"`
	MaxDistance        *float64              `json:"maxDistance"`
	RetrieveAttributes *[]string             `json:"retrieveAttributes,omitempty"`
	Filters            filters.RequestFilter `json:"filters" binding:"required"`
}

type DocumentQueryResponse struct {
	DocumentID string                `json:"documentID"`
	Attributes attributes.Attributes `json:"attributes,omitempty"`
	Distance   float64               `json:"distance"`
}

type QueryDocumentsRequest struct {
	Limit              int                   `json:"limit"`
	MaxDistance        *float64              `json:"maxDistance"`
	RetrieveAttributes *[]string             `json:"retrieveAttributes,omitempty"`
	Filters            filters.RequestFilter `json:"filters" binding:"required"`
	EnableSecondScan   bool                  `json:"enableSecondScan,omitempty"`
}

type PutFileTextRequest struct {
	DocumentID         string                 `json:"documentID"`
	DocumentAttributes []attributes.Attribute `json:"documentAttributes"`
	DocumentContent    string                 `json:"documentContent"`
	DocumentMetadata   *DocumentMetadata      `json:"documentMetadata,omitempty"`
}

type PutFileRequest struct {
	DocumentID         string                 `json:"documentID"`
	DocumentAttributes []attributes.Attribute `json:"documentAttributes"`
	DocumentContent    string                 `json:"documentContent"`
	DocumentMetadata   *DocumentMetadata      `json:"documentMetadata,omitempty"`
}

type DocumentMetadata struct {
	StaticEmbeddingAttributes []string  `json:"embeddingAttributes,omitempty"`
	ExtraAttributesKinds      []string  `json:"extraAttributesKinds,omitempty"`
	EnableSmartAttribution    *bool     `json:"enableSmartAttribution,omitempty"`
	EmbedSmartAttributes      *bool     `json:"embedSmartAttributes,omitempty"`
	EnableOCR                 *bool     `json:"enableOCR,omitempty"`
	ExcludeSmartAttributes    *[]string `json:"excludeSmartAttributes,omitempty"`
}

type DocumentStatus struct {
	DocumentID              string                 `json:"documentID" binding:"required"`
	Status                  string                 `json:"ingestionStatus,omitempty"`
	IngestionStart          string                 `json:"ingestionStart,omitempty"`
	LastSuccessfulIngestion string                 `json:"lastSuccessfulIngestion,omitempty"`
	ErrorMessage            string                 `json:"errorMessage,omitempty"`
	TotalChunks             int                    `json:"totalChunks,omitempty"`
	ChunksIngested          int                    `json:"chunksIngested,omitempty"`
	ChunksPending           int                    `json:"chunksPending,omitempty"`
	ErrorChunks             int                    `json:"errorChunks,omitempty"`
	ChunkStatus             map[string]int         `json:"chunkStatus"`
	DocumentAttributes      []attributes.Attribute `json:"documentAttributes,omitempty"`
}
