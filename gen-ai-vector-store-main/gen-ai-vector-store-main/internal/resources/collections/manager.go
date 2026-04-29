/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package collections

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"go.uber.org/zap"
)

var ErrCollectionNotFound = errors.New("collection not found")

type ColManager interface {
	CollectionExists(ctx context.Context, collectionID string) (exists bool, err error)
	CreateCollection(ctx context.Context, collectionID string) (*Collection, error)
	GetCollections(ctx context.Context) ([]Collection, error)
	GetCollection(ctx context.Context, collectionID string) (*Collection, error)
	DeleteCollection(ctx context.Context, collectionID string) error

	getIsolationID() string
}

type colManager struct {
	IsolationID      string
	Ada              embedders.Embedder
	database         db.Database
	tx               *sql.Tx
	tableCollections string
	logger           *zap.Logger
}

func NewManager(database db.Database, isolationID string, logger *zap.Logger) ColManager {
	mgr := &colManager{
		IsolationID:      isolationID,
		logger:           logger,
		database:         database,
		tableCollections: db.GetTableCollections(isolationID),
	}
	return &tracedCollectionsManager{next: mgr}
}

func NewManagerTx(tx *sql.Tx, isolationID string, logger *zap.Logger) ColManager {
	mgr := &colManager{
		IsolationID:      isolationID,
		tx:               tx,
		tableCollections: db.GetTableCollections(isolationID),
		logger:           logger,
	}
	return &tracedCollectionsManager{next: mgr}
}

func (m *colManager) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.QueryContext(ctx, query, args...)
	} else {
		return m.database.GetConn().QueryContext(ctx, query, args...)
	}
}

func (m *colManager) execTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return tx.ExecContext(ctx, query, args...)
}

func (m *colManager) queryTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return tx.QueryContext(ctx, query, args...)
}

func (m *colManager) rollbackTransactionIfError(err *error) {
	if m.tx != nil {
		defer func() {
			if *err != nil {
				_ = m.tx.Rollback()
			}
		}()
	}
}

func (m *colManager) getIsolationID() string {
	return m.IsolationID
}
