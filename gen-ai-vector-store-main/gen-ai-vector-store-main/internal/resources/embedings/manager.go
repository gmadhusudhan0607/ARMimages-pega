/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package embedings

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"go.uber.org/zap"
)

const serviceName = "genai-vector-store"

type EmbManager interface {
	FindChunks4(ctx context.Context, query *QueryChunksRequest) ([]*Chunk, error) // Uses JSON attributes for filtering
	FindChunks2(ctx context.Context, query *QueryChunksRequest) ([]*Chunk, error) // Uses attr_ids for filtering
	GetDocumentChunksPaginated(ctx context.Context, documentID string, cursor string, limit int) (chunks []*Chunk, itemsTotal, itemsLeft int, err error)

	getIsolationID() string
	getCollectionID() string
}

type embManager struct {
	IsolationID  string
	CollectionID string
	Embedder     embedders.Embedder
	database     db.Database
	schemaName   string
	tablesPrefix string
	logger       zap.Logger
}

func NewManager(
	database db.Database,
	embedder embedders.Embedder,
	isolationID,
	collectionID string,
	logger *zap.Logger,
) EmbManager {
	mgr := &embManager{
		IsolationID:  isolationID,
		CollectionID: collectionID,
		Embedder:     embedder,
		database:     database,
		schemaName:   db.GetSchema(isolationID),
		tablesPrefix: fmt.Sprintf("t_%s", db.GetMD5Hash(collectionID)),
		logger:       *logger,
	}
	return &tracedEmbeddingsManager{next: mgr}
}

func (m *embManager) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return m.database.GetConn().QueryContext(ctx, query, args...)
}

func (m *embManager) queryTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return tx.QueryContext(ctx, query, args...)
}

func (m *embManager) execTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return tx.ExecContext(ctx, query, args...)
}

func (m *embManager) getIsolationID() string {
	return m.IsolationID
}
func (m *embManager) getCollectionID() string {
	return m.CollectionID
}
