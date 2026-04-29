/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics

import (
	"sync"
	"time"
)

type DB struct {
	queryExecutionTime time.Duration
	mutex              sync.RWMutex
}

type DBMeasurement struct {
	queryStartTime time.Time
	db             *DB
}

func (db *DB) QueryExecutionTime() time.Duration {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	return db.queryExecutionTime
}

func (db *DB) NewMeasurement() DBMeasurement {
	return DBMeasurement{
		db: db,
	}
}

func (m *DBMeasurement) Start() {
	m.queryStartTime = time.Now()
}

func (m *DBMeasurement) Stop() {
	if m.queryStartTime.IsZero() {
		return
	}

	m.db.mutex.Lock()
	defer m.db.mutex.Unlock()

	m.db.queryExecutionTime += time.Since(m.queryStartTime)

	m.queryStartTime = time.Time{}
}

func (m *DBMeasurement) StartTime() time.Time {
	return m.queryStartTime
}
