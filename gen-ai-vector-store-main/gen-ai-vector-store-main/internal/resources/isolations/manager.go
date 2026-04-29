/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package isolations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"go.uber.org/zap"
)

type IsoManager interface {
	IsolationExists(ctx context.Context, isolationID string) (bool, error)
	CreateIsolation(ctx context.Context, isolationID, maxStorageSize, pdcEndpointURL string) error
	UpdateIsolation(ctx context.Context, isolationID, maxStorageSize, pdcEndpointURL string) error
	GetIsolation(ctx context.Context, isolationID string) (*Details, error)
	GetIsolations(ctx context.Context) ([]*Details, error)
	DeleteIsolation(ctx context.Context, isolationID string) error
	GetIsolationProfiles(ctx context.Context, isolationID string) ([]EmbeddingProfile, error)
}

type isoManager struct {
	Embedder          embedders.Embedder
	database          db.Database
	dbSchemaName      string
	isolationsTableV2 string
	logger            *zap.Logger
}

func NewManager(database db.Database, logger *zap.Logger) IsoManager {
	schemaName := "vector_store"
	mgr := &isoManager{
		database:          database,
		dbSchemaName:      schemaName,
		isolationsTableV2: fmt.Sprintf("%s.isolations", schemaName),
		logger:            logger,
	}
	return &tracedIsolationsManager{next: mgr}
}

func (m *isoManager) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return m.database.GetConn().Query(query, args...)
}

func (m *isoManager) execTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return tx.ExecContext(ctx, query, args...)
}
