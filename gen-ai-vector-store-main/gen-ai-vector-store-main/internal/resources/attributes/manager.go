/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package attributes

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"go.uber.org/zap"
)

const (
	serviceName         = "genai-vector-store"
	isolationIDParam    = "isolationID"
	collectionNameParam = "collectionName"
)

type Manager interface {
	UpsertAttributes2(ctx context.Context, attrs []Attribute, extraAttributesKinds []string) (attrItemIds []int64, err error)
	GetAttributesByIDs(ctx context.Context, attrIDs []int64) ([]Attribute, error)
	GetAttributesIDs(ctx context.Context, attrs []Attribute) (attrIDs []int64, err error)
	FindAttributes(ctx context.Context, names []string) ([]Attribute, error)
	GetEmbeddingAttributes(ctx context.Context, docId, embID string, filterNames []string) ([]Attribute, error)
	GetEmbeddingAttributesProcessing(ctx context.Context, docId, embID string, filterNames []string) ([]Attribute, error)
	getIsolationID() string
	getCollectionID() string
}

type attrManager struct {
	database     db.Database
	tx           *sql.Tx
	IsolationID  string
	CollectionID string
	schemaName   string
	prefix       string
	tableAttr    string
	logger       *zap.Logger
}

func NewManager(database db.Database, isolationID, collectionID string, logger *zap.Logger) Manager {
	mgr := &attrManager{
		database:     database,
		IsolationID:  isolationID,
		CollectionID: collectionID,
		schemaName:   db.GetSchema(isolationID),
		prefix:       fmt.Sprintf("t_%s", db.GetMD5Hash(collectionID)),
		tableAttr:    db.GetTableAttr(isolationID, collectionID),
		logger:       logger,
	}
	return &tracedAttributesManager{next: mgr}
}

func NewManagerTx(tx *sql.Tx, isolationID, collectionID string, logger *zap.Logger) Manager {
	mgr := &attrManager{
		tx:           tx,
		IsolationID:  isolationID,
		CollectionID: collectionID,
		schemaName:   db.GetSchema(isolationID),
		prefix:       fmt.Sprintf("t_%s", db.GetMD5Hash(collectionID)),
		tableAttr:    db.GetTableAttr(isolationID, collectionID),
		logger:       logger,
	}
	return &tracedAttributesManager{next: mgr}
}

func (m *attrManager) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.QueryContext(ctx, query, args...)
	} else {
		return m.database.GetConn().QueryContext(ctx, query, args...)
	}
}

func (m *attrManager) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.ExecContext(ctx, query, args...)
	} else {
		return m.database.GetConn().ExecContext(ctx, query, args...)
	}
}

func (m *attrManager) getIsolationID() string {
	return m.IsolationID
}
func (m *attrManager) getCollectionID() string {
	return m.CollectionID
}
