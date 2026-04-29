/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package mocks

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/pgvector/pgvector-go"
)

type MockDatabase struct {
	Mock       sqlmock.Sqlmock
	SqlDB      *sql.DB
	BatchError error
	BatchQuery []db.Query
}

func (db *MockDatabase) UpsertDocStatus(_ context.Context, _, _, _, _ string) error {
	return nil
}
func (db *MockDatabase) UpsertDocStatusTx(_ context.Context, _ *sql.Tx, _, _, _, _ string) error {
	return nil
}
func (db *MockDatabase) UpdateDocAttributesTx(_ context.Context, _ *sql.Tx, _, _ string, _ []int64, _ interface{}) error {
	return nil
}
func (db *MockDatabase) UpdateEmbAttributesTx(_ context.Context, _ *sql.Tx, _, _ string, _, _ []int64) error {
	return nil
}
func (db *MockDatabase) DeleteDocMetadataTx(_ context.Context, _ *sql.Tx, _, _ string) error {
	return nil
}
func (db *MockDatabase) InsertDocMetadataTx(_ context.Context, _ *sql.Tx, _, _, _, _ string) error {
	return nil
}

func (db *MockDatabase) Query(_ context.Context, query string) (*sql.Rows, error) {
	return db.GetConn().Query(query)
	//rows, err := db.GetConn().Query(query)
	//if err != nil {
	//	//pgErr, ok := err.(*pgconn.PgError)
	//	//if ok && pgErr.Code == pgerrcode.UndefinedTable {
	//	//	return nil, nil // clear the error
	//	//}
	//	return nil, err
	//}
	//return rows, nil
}
func (db *MockDatabase) Begin() (*sql.Tx, error) {
	return nil, nil
}

func (db *MockDatabase) GetConn() db.SQLDB {
	return db.SqlDB
}

func (db *MockDatabase) GetSingleConn(ctx context.Context) (*sql.Conn, error) {
	return db.SqlDB.Conn(ctx)
}

func (db *MockDatabase) SetError(err error) *MockDatabase {
	db.BatchError = err
	return db
}

func NewMockDb() *MockDatabase {
	sqlDB, mock, err := sqlmock.New(sqlmock.ValueConverterOption(CustomTypeConvertor{}))
	if err != nil {
		panic(fmt.Sprintf("error creating mock db: %v", err))

	}
	if err != nil {
		return nil
	}
	return &MockDatabase{
		SqlDB: sqlDB,
		Mock:  mock,
	}
}

// AttrValues type Attributes required for proper rows.Scan() in GORM
type AttrValues []string

// Attributes required for proper rows.Scan() in GORM
type Attributes []Attribute

type Attribute struct {
	Name   string     `json:"name" binding:"required"`
	Type   string     `json:"type,omitempty"`
	Values AttrValues `json:"value" binding:"required"`
}

type CustomTypeConvertor struct {
}

func (s CustomTypeConvertor) ConvertValue(src interface{}) (driver.Value, error) {
	switch v := src.(type) {
	case string:
		return v, nil
	case []string:
		return v, nil
	case float64:
		return v, nil
	case int:
		return v, nil
	case attributes.Attributes:
		return v, nil
	case attributes.Attribute:
		return v, nil
	case attributes.AttrValues:
		return v, nil
	case pgvector.Vector:
		return v, nil
	case time.Time:
		return v, nil
	case []float32:
		return v, nil
	default:
		return nil, fmt.Errorf("CustomTypeConvertor: cannot convert value %v of type %T", src, src)
	}
}
