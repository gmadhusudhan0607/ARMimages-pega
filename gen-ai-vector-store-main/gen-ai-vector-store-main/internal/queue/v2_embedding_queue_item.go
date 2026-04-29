/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package queue

import (
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
)

type EmbeddingQueueItem struct {
	IsolationID    string                           `json:"iso_id" binding:"required"`
	CollectionID   string                           `json:"col_id" binding:"required"`
	DocumentID     string                           `json:"doc_id" binding:"required"`
	EmbeddingID    string                           `json:"emb_id" binding:"required"`
	RetryCount     int                              `json:"retry_count" binding:"required"`
	Data           *EmbeddingQueueItemData          `json:"data,omitempty"`
	AdditionalData EmbeddingQueueItemAdditionalData `json:"additional_data,omitempty"`
}

type EmbeddingQueueItemData struct {
	ID           string             `json:"id" binding:"required"`
	DocumentID   string             `json:"documentID" binding:"required"`
	Status       string             `json:"status,omitempty"`
	ResponseCode int                `json:"response_code,omitempty"`
	ErrorMessage string             `json:"error_message,omitempty"`
	ModifiedAt   time.Time          `json:"modified_at,omitempty"`
	Content      string             `json:"content,omitempty"`
	Embedding    *pgvector.Vector   `json:"embedding,omitempty"`
	Metadata     EmbeddingAttribute `json:"metadata,omitempty"`
	DocMetadata  EmbeddingAttribute `json:"doc_metadata,omitempty"`
}
type EmbeddingAttribute struct {
	Attribute []string `json:"embeddingAttributes"`
}

type EmbeddingQueueItemAdditionalData struct {
	TraceID   string `json:"trace_id,omitempty"`
	SpanID    string `json:"span_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func (e *EmbeddingQueueItem) GetEmbPath() string {
	return fmt.Sprintf("%s/%s/%s/%s", e.IsolationID, e.CollectionID, e.DocumentID, e.EmbeddingID)
}
func (e *EmbeddingQueueItem) GetDocPath() string {
	return fmt.Sprintf("%s/%s/%s", e.IsolationID, e.CollectionID, e.DocumentID)
}

func (e *EmbeddingQueueItem) isValid() bool {
	return e.IsolationID != "" && e.CollectionID != "" && e.DocumentID != "" && e.EmbeddingID != ""
}
