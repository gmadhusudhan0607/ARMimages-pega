/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package collections

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"go.uber.org/zap"
)

const (
	EmbeddingProfileStatusUnknown = "UNKNOWN"
	EmbeddingProfileStatusReady   = "READY"
)

var DefaultEmbeddingProfileID = helpers.GetEnvOrDefault("DEFAULT_EMBEDDING_PROFILE", "openai-text-embedding-ada-002")

func (m *colManager) CollectionExists(ctx context.Context, collectionID string) (exists bool, err error) {
	defer m.rollbackTransactionIfError(&err)
	query := fmt.Sprintf("SELECT count(*) from %s where col_id = $1", m.tableCollections)
	rows, err := m.Query(ctx, query, collectionID)
	if err != nil || rows == nil {
		return false, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("error while reading rows from query: [%s]: %w", query, err)
		}
	}
	return exists, nil
}

func (m *colManager) initCollection(ctx context.Context, collectionID, defaultProfile string) (err error) {
	tablesPrefix := db.GetMD5Hash(collectionID)

	// always run in transaction. If we are in upstream transaction, use it instead otherwise open new one.
	var tx0 *sql.Tx
	if m.tx != nil {
		tx0 = m.tx
		defer m.rollbackTransactionIfError(&err)
	} else {
		tx0, err = m.database.GetConn().Begin()
		if err != nil {
			return fmt.Errorf("error while opening transaction when initializing collection: %w", err)
		}
		defer func() {
			if err != nil {
				m.logger.Warn("failed initializing collection",
					zap.String("error", err.Error()))
				_ = tx0.Rollback()
			}
		}()
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (col_id, col_prefix, default_emb_profile, record_timestamp)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON conflict (col_id) DO NOTHING
    `, m.tableCollections)

	_, err = m.execTx(ctx, tx0, query, collectionID, tablesPrefix, defaultProfile)
	if err != nil {
		return fmt.Errorf("error while initializing collection '%s' [%s]: %w", collectionID, query, err)
	}

	query = fmt.Sprintf(`
		INSERT INTO %s (col_id, profile_id, tables_prefix, status)
		VALUES ($1, $2, $3, $4)
		ON conflict (col_id, profile_id) DO NOTHING
	`, db.GetTableCollectionEmbeddingProfiles(m.IsolationID))

	// The default profile is always ready to use when the collection is created.
	_, err = m.execTx(ctx, tx0, query, collectionID, defaultProfile, tablesPrefix, EmbeddingProfileStatusReady)
	if err != nil {
		return fmt.Errorf("error while initializing collection '%s' [%s]: %w", collectionID, query, err)
	}

	if m.tx != nil {
		// if we are in upstream transaction, do not commit
		return nil
	} else {
		// if we are in our own transaction, commit
		return tx0.Commit()
	}
}

func (m *colManager) CreateCollection(ctx context.Context, collectionID string) (*Collection, error) {
	var err error
	var collectionExists bool
	defer m.rollbackTransactionIfError(&err)

	collectionExists, err = m.CollectionExists(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("error while checking if collection exists: %w", err)
	}
	if collectionExists {
		return nil, fmt.Errorf("collection %s already exists", collectionID)
	}

	err = m.createTables(ctx, collectionID)
	if err != nil {
		return nil, fmt.Errorf("error while creating DBs for collection: %w", err)
	}

	err = m.initCollection(ctx, collectionID, DefaultEmbeddingProfileID)
	if err != nil {
		return nil, fmt.Errorf("error while initializing collection: %w", err)
	}

	co := &Collection{
		ID:                      collectionID,
		DefaultEmbeddingProfile: DefaultEmbeddingProfileID,
	}

	return co, nil
}

func (m *colManager) GetCollections(ctx context.Context) ([]Collection, error) {
	var err error
	defer m.rollbackTransactionIfError(&err)

	query := fmt.Sprintf(
		`SELECT 
			c.col_id,
			c.default_embedding_profile,
			c.documents_total
		FROM 
			vector_store.get_collection_document_count('%s') c;`,
		m.IsolationID,
	)
	var rows *sql.Rows
	rows, err = m.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error executing query [%s]: %w", query, err)
	}
	defer rows.Close()

	collections := []Collection{}
	for rows.Next() {
		var collectionID string
		var defaultEmbProfile string
		var documentsTotal int
		err = rows.Scan(&collectionID, &defaultEmbProfile, &documentsTotal)
		if err != nil {
			return nil, fmt.Errorf("error while reading rows from query: [%s]: %w", query, err)
		}

		co := Collection{
			ID:                      collectionID,
			DefaultEmbeddingProfile: defaultEmbProfile,
			DocumentsTotal:          documentsTotal,
		}
		collections = append(collections, co)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return collections, nil
}

func (m *colManager) GetCollection(ctx context.Context, collectionID string) (*Collection, error) {
	var err error
	defer m.rollbackTransactionIfError(&err)

	query := fmt.Sprintf(
		`SELECT 
			c.col_id,
			c.default_embedding_profile,
			c.documents_total
		FROM 
			vector_store.get_collection_document_count('%s', '%s') c;`,
		m.IsolationID,
		collectionID,
	)
	var rows *sql.Rows
	rows, err = m.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error executing query [%s]: %w", query, err)
	}
	defer rows.Close()

	if rows.Next() {
		var collectionID string
		var defaultEmbProfile string
		var documentsTotal int
		err = rows.Scan(&collectionID, &defaultEmbProfile, &documentsTotal)
		if err != nil {
			return nil, fmt.Errorf("error while reading rows from query: [%s]: %w", query, err)
		}

		return &Collection{
			ID:                      collectionID,
			DefaultEmbeddingProfile: defaultEmbProfile,
			DocumentsTotal:          documentsTotal,
		}, nil
	}
	return nil, ErrCollectionNotFound
}

func (m *colManager) DeleteCollection(ctx context.Context, collectionID string) error {
	var err error
	var tx0 *sql.Tx
	if m.tx != nil {
		tx0 = m.tx
		defer m.rollbackTransactionIfError(&err)
	} else {
		tx0, err = m.database.GetConn().Begin()
		if err != nil {
			return fmt.Errorf("error while opening transaction when deletion tables for collection: %w", err)
		}
		defer func() {
			if err != nil {
				m.logger.Warn("failed deleting collection",
					zap.String("error", err.Error()))
				_ = tx0.Rollback()
			}
		}()
	}

	err = m.dropTables(ctx, tx0, collectionID)
	if err != nil {
		return fmt.Errorf("failed deleting DBs: %w", err)
	}
	colTable := db.GetTableCollections(m.IsolationID)
	query := fmt.Sprintf("DELETE FROM %s WHERE col_id = $1", colTable)
	_, err = m.execTx(ctx, tx0, query, collectionID)
	if err != nil {
		return fmt.Errorf("error executing query [%s]: %w", query, err)
	}

	if m.tx != nil {
		// if we are in upstream transaction, do not commit
		return nil
	} else {
		// if we are in our own transaction, commit
		return tx0.Commit()
	}
}
