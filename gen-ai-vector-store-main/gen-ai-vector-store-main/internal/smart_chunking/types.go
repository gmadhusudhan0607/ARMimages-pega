/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package smart_chunking

import (
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
)

// JobRequestOptions represents the "options" JSON sent to SC POST /v1/{isolationID}/jobs.
// Matches SC's JobRequestOptions schema.
type JobRequestOptions struct {
	Tasks         []string        `json:"tasks"`
	InlineResults []string        `json:"inlineResults,omitempty"`
	TaskOptions   *JobTaskOptions `json:"taskOptions,omitempty"`
}

// JobTaskOptions contains per-task configuration options.
// Matches SC's JobTaskOptions schema.
type JobTaskOptions struct {
	Extraction *ExtractionOptions `json:"extraction,omitempty"`
	Chunking   *ChunkingOptions   `json:"chunking,omitempty"`
	Indexing   *IndexingOptions   `json:"indexing,omitempty"`
}

// ExtractionOptions configures the extraction task.
// Matches SC's ExtractionOptions schema.
type ExtractionOptions struct {
	ExtractionMethod string `json:"extractionMethod,omitempty"`
	EnableOCR        *bool  `json:"enableOCR,omitempty"`
	OutputFormat     string `json:"outputFormat,omitempty"`
}

// ChunkingOptions configures the chunking task.
// Matches SC's ChunkingOptions schema.
type ChunkingOptions struct {
	ChunkingMethod         string    `json:"chunkingMethod,omitempty"`
	ChunkSize              int       `json:"chunkSize,omitempty"`
	ChunkOverlap           int       `json:"chunkOverlap,omitempty"`
	SemanticThreshold      *float64  `json:"semanticThreshold,omitempty"`
	StructureAware         *bool     `json:"structureAware,omitempty"`
	AddContext             *bool     `json:"addContext,omitempty"`
	UseAdaptiveThreshold   *bool     `json:"useAdaptiveThreshold,omitempty"`
	SemanticModel          string    `json:"semanticModel,omitempty"`
	EnableSmartAttribution *bool     `json:"enableSmartAttribution,omitempty"`
	ExcludeSmartAttributes *[]string `json:"excludeSmartAttributes,omitempty"`
}

// IndexingOptions configures the indexing task.
// Matches SC's JobIndexingOptions schema.
// documentID is a VS-specific extension: SC reads it to use as the VS document ID
// (falling back to SC's own operationID if absent).
type IndexingOptions struct {
	CollectionName       string                 `json:"collectionName"`
	DocumentID           string                 `json:"documentID,omitempty"`
	DocumentAttributes   []attributes.Attribute `json:"documentAttributes,omitempty"`
	EmbeddingAttributes  []string               `json:"embeddingAttributes,omitempty"`
	EmbedSmartAttributes *bool                  `json:"embedSmartAttributes,omitempty"`
}

// JobSubmittedResponse represents the SC /job API 202 response.
// Matches SC's JobSubmittedResponse schema.
type JobSubmittedResponse struct {
	OperationID        string   `json:"operationID"`
	IsolationID        string   `json:"isolationID"`
	Status             string   `json:"status"`
	RequestedTasks     []string `json:"requestedTasks,omitempty"`
	InlineResults      []string `json:"inlineResults,omitempty"`
	Message            string   `json:"message,omitempty"`
	CallbackRegistered bool     `json:"callbackRegistered"`
}
