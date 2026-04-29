/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

package smart_chunking

import (
	"context"
	"io"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
)

// Manager provides a high-level interface for submitting jobs to Smart Chunking.
type Manager interface {
	GetServiceURL() string
	SubmitJob(ctx context.Context, authToken string, fileReader io.Reader, fileName string, options JobRequestOptions) (*JobSubmittedResponse, error)
}

type smartChunkingManager struct {
	serviceURL   string
	isolationID  string
	collectionID string
	client       SmartChunkingClient
}

func NewManager(client SmartChunkingClient, isolationID, collectionID string) Manager {
	return &smartChunkingManager{
		serviceURL:   helpers.GetEnvOrDefault("GENAI_SMART_CHUNKING_SERVICE_URL", ""),
		isolationID:  isolationID,
		collectionID: collectionID,
		client:       client,
	}
}

func (m *smartChunkingManager) GetServiceURL() string {
	return m.serviceURL
}

func (m *smartChunkingManager) SubmitJob(ctx context.Context, authToken string, fileReader io.Reader, fileName string, options JobRequestOptions) (*JobSubmittedResponse, error) {
	return m.client.SubmitJob(ctx, authToken, m.isolationID, fileReader, fileName, options)
}
