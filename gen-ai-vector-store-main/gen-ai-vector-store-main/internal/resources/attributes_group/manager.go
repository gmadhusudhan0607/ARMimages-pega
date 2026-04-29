/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package attributesgroup

import (
	"context"
	"database/sql"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"go.uber.org/zap"
)

type Manager interface {
	CreateTables(ctx context.Context) error
	DropTables(ctx context.Context) error

	CreateAttributesGroup(ctx context.Context, description string, attrs []string) (ag *AttributesGroup, err error)
	GetAttributesGroup(ctx context.Context, groupID string) (ag *AttributesGroup, err error)
	GetAttributesGroupDescriptions(ctx context.Context) (agDescriptions map[string]string, err error)
	DeleteAttributesGroup(ctx context.Context, groupID string) error

	getIsolationID() string
}

type attrGrpManager struct {
	database     db.Database
	tx           *sql.Tx
	IsolationID  string
	schemaName   string
	attrGrpTable string
	logger       *zap.Logger
}

func NewManager(database db.Database, isolationID string, logger *zap.Logger) Manager {
	mgr := &attrGrpManager{
		database:     database,
		IsolationID:  isolationID,
		schemaName:   db.GetSchema(isolationID),
		attrGrpTable: db.GetTableSmartAttrGroup(isolationID),
		logger:       logger,
	}
	return &tracedAttrGrpManager{next: mgr}
}

func NewManagerTx(tx *sql.Tx, isolationID string, logger *zap.Logger) Manager {
	mgr := &attrGrpManager{
		tx:           tx,
		IsolationID:  isolationID,
		schemaName:   db.GetSchema(isolationID),
		attrGrpTable: db.GetTableSmartAttrGroup(isolationID),
		logger:       logger,
	}
	return &tracedAttrGrpManager{next: mgr}
}

func (m *attrGrpManager) rollbackTransactionIfError(err *error) {
	if m.tx != nil {
		defer func() {
			if *err != nil {
				_ = m.tx.Rollback()
			}
		}()
	}
}

func (m *attrGrpManager) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.QueryContext(ctx, query, args...)
	} else {
		return m.database.GetConn().QueryContext(ctx, query, args...)
	}
}

func (m *attrGrpManager) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	measurement := servicemetrics.FromContext(ctx).DbMetrics.NewMeasurement()
	measurement.Start()
	defer measurement.Stop()

	if m.tx != nil {
		return m.tx.ExecContext(ctx, query, args...)
	} else {
		return m.database.GetConn().ExecContext(ctx, query, args...)
	}
}
func (m *attrGrpManager) getIsolationID() string {
	return m.IsolationID
}
