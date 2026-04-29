/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

type mockSQLDBPool struct {
	beginFunc        func() (*sql.Tx, error)
	beginTxFunc      func(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	closeFunc        func() error
	queryFunc        func(query string, args ...any) (*sql.Rows, error)
	queryContextFunc func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	execFunc         func(query string, args ...any) (sql.Result, error)
	execContextFunc  func(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (m *mockSQLDBPool) Conn(ctx context.Context) (*sql.Conn, error) {
	return nil, nil
}

func (m *mockSQLDBPool) Begin() (*sql.Tx, error) {
	return m.beginFunc()
}

func (m *mockSQLDBPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if m.beginTxFunc != nil {
		return m.beginTxFunc(ctx, opts)
	}
	// Fall back to non-context version if context version not provided
	return m.beginFunc()
}

func (m *mockSQLDBPool) Close() error {
	return m.closeFunc()
}

func (m *mockSQLDBPool) Query(query string, args ...any) (*sql.Rows, error) {
	return m.queryFunc(query, args...)
}

func (m *mockSQLDBPool) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return m.queryContextFunc(ctx, query, args...)
}

func (m *mockSQLDBPool) Exec(query string, args ...any) (sql.Result, error) {
	return m.execFunc(query, args...)
}

func (m *mockSQLDBPool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if m.execContextFunc != nil {
		return m.execContextFunc(ctx, query, args...)
	}
	// Fall back to non-context version if context version not provided
	return m.execFunc(query, args...)
}

func TestSQLDB_Begin(t *testing.T) {
	ctx := context.Background()
	successfulTx := &sql.Tx{}
	errorTx := fmt.Errorf("begin error")

	tests := []struct {
		name    string
		dbPool  SQLDB
		want    *sql.Tx
		wantErr bool
	}{
		{
			name: "successful call to Begin",
			dbPool: &mockSQLDBPool{
				beginFunc: func() (*sql.Tx, error) {
					return successfulTx, nil
				},
			},
			want:    successfulTx,
			wantErr: false,
		},
		{
			name: "error in Begin",
			dbPool: &mockSQLDBPool{
				beginFunc: func() (*sql.Tx, error) {
					return nil, errorTx
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &sqlDB{
				ctx:    ctx,
				dbPool: tt.dbPool,
			}
			got, err := p.Begin()
			if (err != nil) != tt.wantErr {
				t.Errorf("Begin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Begin() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLDB_Close(t *testing.T) {
	tests := []struct {
		name    string
		dbPool  SQLDB
		wantErr bool
	}{
		{
			name: "successful call to Close",
			dbPool: &mockSQLDBPool{
				closeFunc: func() error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "error in Close",
			dbPool: &mockSQLDBPool{
				closeFunc: func() error {
					return fmt.Errorf("close error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &sqlDB{
				dbPool: tt.dbPool,
			}
			err := p.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLDB_Query(t *testing.T) {
	ctx := context.Background()
	successfulRows := &sql.Rows{}
	errorQuery := fmt.Errorf("query error")

	tests := []struct {
		name    string
		dbPool  SQLDB
		query   string
		args    []any
		want    *sql.Rows
		wantErr bool
	}{
		{
			name: "successful call to Query",
			dbPool: &mockSQLDBPool{
				queryFunc: func(query string, args ...any) (*sql.Rows, error) {
					return successfulRows, nil
				},
			},
			query:   "SELECT * FROM table",
			args:    []any{},
			want:    successfulRows,
			wantErr: false,
		},
		{
			name: "error in Query",
			dbPool: &mockSQLDBPool{
				queryFunc: func(query string, args ...any) (*sql.Rows, error) {
					return nil, errorQuery
				},
			},
			query:   "SELECT * FROM table",
			args:    []any{},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &sqlDB{
				ctx:    ctx,
				dbPool: tt.dbPool,
			}
			got, err := p.Query(tt.query, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Query() got = %v, want %v", got, tt.want)
			}
		})
	}
}
