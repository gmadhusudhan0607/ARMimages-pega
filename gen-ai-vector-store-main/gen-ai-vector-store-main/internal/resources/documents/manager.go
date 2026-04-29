/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	"go.uber.org/zap"
)

const (
	serviceName         = "genai-vector-store"
	isolationIDParam    = "isolationID"
	collectionNameParam = "collectionName"
)

type Manager interface {
	FindDocuments2(ctx context.Context, query *QueryDocumentsRequest) ([]*DocumentQueryResponse, error) // Uses attr_ids for filtering
	FindDocuments4(ctx context.Context, query *QueryDocumentsRequest) ([]*DocumentQueryResponse, error) // Uses JSON attributes for filtering
	ListDocuments2(ctx context.Context, status string, filters []attributes.AttributeFilter) ([]Document, error)
	ListDocuments3(ctx context.Context, status string, filters []attributes.AttributeFilter) ([]Document, error) // Uses JSONB doc_attributes for filtering
	DocumentExists(ctx context.Context, docID string) (bool, error)
	GetDocument2(ctx context.Context, docID string) (Document, error)
	DeleteDocument2(ctx context.Context, docID string) (int64, error)
	DeleteDocumentsByFilters(ctx context.Context, filters []attributes.AttributeFilter) (int64, error)
	DeleteDocumentsByFilters3(ctx context.Context, filters []attributes.AttributeFilter) (int64, error) // Uses JSONB doc_attributes for filtering
	SetAttributes(ctx context.Context, docID string, attrs attributes.Attributes) error
	GetChunksContent2(ctx context.Context, docID string) ([]embedings.Chunk, error)
	GetAttributeIDs(ctx context.Context, docID string) ([]int64, error)
	CalculateDocumentStatus2(ctx context.Context, documentID string) (status, msg string, err error)
	SetDocumentStatus(ctx context.Context, documentID, status, msg string) (err error)
	GetDocumentStatuses(ctx context.Context, status string, fields []string, filter attributes.Filter, cursor string, limit int) (documentStatuses []DocumentStatus, itemsTotal int, itemsLeft int, err error)
	GetDocumentStatuses3(ctx context.Context, status string, fields []string, filter attributes.Filter, cursor string, limit int) (documentStatuses []DocumentStatus, itemsTotal int, itemsLeft int, err error) // Uses JSONB doc_attributes for filtering

	getIsolationID() string
	getCollectionID() string
}

type docManager struct {
	database     db.Database
	tx           *sql.Tx
	Embedder     embedders.Embedder
	IsolationID  string
	CollectionID string
	schemaName   string
	prefix       string
	tableDoc     string
	attrMgr      attributes.Manager
	logger       *zap.Logger
}

func NewManager(
	dbConn db.Database,
	embedder embedders.Embedder,
	isolationID,
	collectionID string,
	logger *zap.Logger,
) Manager {
	mgr := &docManager{
		Embedder:     embedder, // TODO: make private
		database:     dbConn,
		IsolationID:  isolationID,
		CollectionID: collectionID,
		schemaName:   db.GetSchema(isolationID),
		prefix:       fmt.Sprintf("t_%s", db.GetMD5Hash(collectionID)),
		tableDoc:     db.GetTableDoc(isolationID, collectionID),
		attrMgr:      attributes.NewManager(dbConn, isolationID, collectionID, logger),
		logger:       logger,
	}
	return &tracedDocumentsManager{next: mgr}
}

func NewManagerTx(
	tx *sql.Tx,
	embedder embedders.Embedder,
	isolationID,
	collectionID string,
	logger *zap.Logger,
) Manager {
	mgr := &docManager{
		Embedder:     embedder, // TODO: make private
		tx:           tx,
		IsolationID:  isolationID,
		CollectionID: collectionID,
		schemaName:   db.GetSchema(isolationID),
		prefix:       fmt.Sprintf("t_%s", db.GetMD5Hash(collectionID)),
		tableDoc:     db.GetTableDoc(isolationID, collectionID),
		attrMgr:      attributes.NewManagerTx(tx, isolationID, collectionID, logger),
		logger:       logger,
	}
	return &tracedDocumentsManager{next: mgr}
}

func (m *docManager) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.QueryContext(ctx, query, args...)
	} else {
		return m.database.GetConn().QueryContext(ctx, query, args...)
	}
}

func (m *docManager) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.Exec(query, args...)
	} else {
		return m.database.GetConn().Exec(query, args...)
	}
}

func (m *docManager) queryTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return tx.QueryContext(ctx, query, args...)
}

func (m *docManager) execTx(ctx context.Context, tx *sql.Tx, query string, args ...any) (sql.Result, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	return tx.Exec(query, args...)
}

func (m *docManager) getIsolationID() string {
	return m.IsolationID
}
func (m *docManager) getCollectionID() string {
	return m.CollectionID
}

func (m *docManager) getTxOrStartNew() (tx *sql.Tx, rollback func() error, commit func() error, err error) {
	if m.tx != nil {
		return m.tx, func() error { return nil }, func() error { return nil }, nil
	}

	tx, err = m.database.GetConn().Begin()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start transaction: %s", err)
	}

	return tx, tx.Rollback, tx.Commit, nil
}
