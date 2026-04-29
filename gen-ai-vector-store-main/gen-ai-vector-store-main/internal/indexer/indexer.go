/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package indexer

import (
	"context"
	"database/sql"
	"sync"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer/parallelization"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
)

const (
	serviceName                          = "genai-vector-store"
	defaultMaxConcurrentEmbedderRequests = 50
	maxConcurrentEmbedderRequestsEnvVar  = "EMBEDDER_MAX_CONCURRENT_REQUESTS"
)

var (
	globalParallelizationGroup *parallelization.Group
	initOnce                   sync.Once
)

// getParallelizationGroup returns the global parallelization group, initializing it once
func getParallelizationGroup() *parallelization.Group {
	initOnce.Do(func() {
		maxConcurrent := int(helpers.GetEnvOrDefaultInt64(maxConcurrentEmbedderRequestsEnvVar, defaultMaxConcurrentEmbedderRequests))
		globalParallelizationGroup = parallelization.Limited(maxConcurrent)
	})
	return globalParallelizationGroup
}

// TODO: Move Index2() to documents.Manager
type Indexer interface {
	Index(ctx context.Context, docID string, chunks []embedings.Chunk, attributes []attributes.Attribute, docMetadata *documents.DocumentMetadata, consistencyLevel string, extraAttributesKinds []string) error
	MoveDataToPermanentTablesTx(ctx context.Context, tx *sql.Tx, docID string) error
	getIsolationID() string
	getCollectionID() string
}

type indexer struct {
	IsolationID    string
	CollectionName string
	Embedder       embedders.Embedder
	dbConn         db.Database
	genericDbConn  db.Database // lightweight pool for quick, non-transactional ops (status updates, queue writes)
	logger         *zap.Logger
}

// NewIndexer creates a new Indexer.
//
// dbConn is the primary pool used for ingestion transactions (both sync and
// async processing).  genericDbConn is a separate, lightweight pool used for
// quick, non-transactional operations such as document-status updates and
// re-embedding queue writes.  Keeping these on a dedicated pool prevents them
// from being blocked when the ingestion pool is under heavy load.
//
// If genericDbConn is nil it defaults to dbConn (useful for callers that do
// not need pool separation, e.g. background workers).
func NewIndexer(dbConn db.Database, genericDbConn db.Database, embedder embedders.Embedder, isolationID, collectionName string, logger *zap.Logger) Indexer {
	if genericDbConn == nil {
		genericDbConn = dbConn
	}
	idx := &indexer{
		IsolationID:    isolationID,
		CollectionName: collectionName,
		Embedder:       embedder,
		dbConn:         dbConn,
		genericDbConn:  genericDbConn,
		logger:         logger,
	}

	return &tracedIndexer{next: idx}
}

func (i *indexer) getIsolationID() string {
	return i.IsolationID
}
func (i *indexer) getCollectionID() string {
	return i.CollectionName
}
