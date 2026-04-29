/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package isolations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	attributesgroup "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes_group"
	"go.uber.org/zap"
)

func (m *isoManager) CreateIsolation(ctx context.Context, isolationID, maxStorageSize, pdcEndpointURL string) (err error) {

	tx, err := m.database.GetConn().Begin()
	if err != nil {
		return fmt.Errorf("error while opening transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = m.createSchemaV2(ctx, tx, isolationID); err != nil {
		return fmt.Errorf("error while creating schema: %w", err)
	}

	if err = m.createProfilesTable(ctx, tx, isolationID); err != nil {
		return fmt.Errorf("error while creating profiles table: %w", err)
	}

	if err = m.createCollectionsTable(ctx, tx, isolationID); err != nil {
		return fmt.Errorf("error while creating collections table: %w", err)
	}

	if err = m.createCollectionsProfilesTable(ctx, tx, isolationID); err != nil {
		return fmt.Errorf("error while creating collection profiles table: %w", err)
	}

	attrGrpMgr := attributesgroup.NewManagerTx(tx, isolationID, m.logger)
	if err = attrGrpMgr.CreateTables(ctx); err != nil {
		return fmt.Errorf("error while creating attributes group tables: %w", err)
	}

	err = m.insertIsolationDetails(ctx, tx, isolationID, maxStorageSize, pdcEndpointURL)
	if err != nil {
		return fmt.Errorf("error while inserting isolation data: %w", err)
	}

	return tx.Commit()
}

func (m *isoManager) IsolationExists(ctx context.Context, isolationID string) (bool, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE iso_id=$1", m.isolationsTableV2)
	rows, err := m.query(ctx, query, isolationID)
	if err != nil {
		return false, fmt.Errorf("error checking if isolation '%s' exists: %s", isolationID, err)
	}
	defer rows.Close() // Defer the close statement after checking for errors

	var counter int
	if rows.Next() {
		err = rows.Scan(&counter)
		if err != nil {
			return false, fmt.Errorf("error while reading rows from query [%s]: %w", query, err)
		}
	}
	return counter == 1, nil
}

func (m *isoManager) GetIsolation(ctx context.Context, isolationID string) (*Details, error) {
	var createdAt *time.Time
	var modifiedAt *time.Time
	var maxStorageSize *string
	var pdcEndpointURL *string
	query := "SELECT iso_id, max_storage_size, pdc_endpoint_url, created_at, modified_at FROM vector_store.isolations WHERE iso_id=$1"
	rows, err := m.query(ctx, query, isolationID)
	if err != nil || rows == nil {
		return nil, err
	}
	defer rows.Close()

	d := &Details{}
	for rows.Next() {
		err = rows.Scan(&d.ID, &maxStorageSize, &pdcEndpointURL, &createdAt, &modifiedAt)
		if err != nil {
			return nil, err
		}
	}
	if createdAt != nil {
		d.CreatedAt = *createdAt
	}
	if modifiedAt != nil {
		d.ModifiedAt = *modifiedAt
	}
	if maxStorageSize != nil {
		d.MaxStorageSize = *maxStorageSize
	}
	if pdcEndpointURL != nil {
		d.PDCEndpointURL = *pdcEndpointURL
	}
	return d, nil
}

func (m *isoManager) GetIsolations(ctx context.Context) ([]*Details, error) {
	var isolations []*Details

	query := "SELECT iso_id, max_storage_size, pdc_endpoint_url, created_at, modified_at FROM vector_store.isolations"
	rows, err := m.query(ctx, query)
	if err != nil || rows == nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		d := &Details{}
		var createdAt *time.Time
		var modifiedAt *time.Time
		var maxStorageSize *string
		var pdcEndpointURL *string
		err = rows.Scan(&d.ID, &maxStorageSize, &pdcEndpointURL, &createdAt, &modifiedAt)
		if err != nil {
			return nil, err
		}
		if createdAt != nil {
			d.CreatedAt = *createdAt
		}
		if modifiedAt != nil {
			d.ModifiedAt = *modifiedAt
		}
		if maxStorageSize != nil {
			d.MaxStorageSize = *maxStorageSize
		}
		if pdcEndpointURL != nil {
			d.PDCEndpointURL = *pdcEndpointURL
		}
		isolations = append(isolations, d)
	}
	return isolations, nil
}

func (m *isoManager) createCollectionsProfilesTable(ctx context.Context, tx *sql.Tx, isolationID string) (err error) {
	tableCollectionProfiles := db.GetTableCollectionEmbeddingProfiles(isolationID)
	tableCollections := db.GetTableCollections(isolationID)
	tableProfiles := db.GetTableEmbeddingProfiles(isolationID)

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
		    col_id           TEXT REFERENCES %[2]s (col_id) ON DELETE CASCADE NOT NULL,
		    profile_id       VARCHAR(127) REFERENCES %[3]s (profile_id) ON DELETE CASCADE NOT NULL,
		    tables_prefix    VARCHAR(40) NOT NULL,
		    status           VARCHAR(63),
		    details          TEXT,
		    reated_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		    PRIMARY KEY (col_id, profile_id),
		    UNIQUE (tables_prefix)
		    )
		`, tableCollectionProfiles, tableCollections, tableProfiles)

	_, err = m.execTx(ctx, tx, query)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("error while creating collection profiles table [%s]: %w", query, err)
	}
	return nil
}

func (m *isoManager) createCollectionsTable(ctx context.Context, tx *sql.Tx, isolationID string) (err error) {
	tableCollections := db.GetTableCollections(isolationID)
	tableProfiles := db.GetTableEmbeddingProfiles(isolationID)

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
		    col_id               TEXT NOT NULL,
		    col_prefix           VARCHAR(40) NOT NULL UNIQUE,
		    default_emb_profile  VARCHAR(127) REFERENCES %[2]s(profile_id) ON DELETE CASCADE NOT NULL,
		    record_timestamp     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		    PRIMARY KEY (col_id)
		    )
		`, tableCollections, tableProfiles)
	_, err = m.execTx(ctx, tx, query)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("error while creating collections table [%s]: %w", query, err)
	}
	return nil
}

func (m *isoManager) createProfilesTable(ctx context.Context, tx *sql.Tx, isolationID string) (err error) {
	tableProfiles := db.GetTableEmbeddingProfiles(isolationID)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s
			(
				profile_id    VARCHAR(127) NOT NULL,
				provider_name VARCHAR(63)  NOT NULL,
				model_name    VARCHAR(127) NOT NULL,
				model_version VARCHAR(63)  NOT NULL,
				vector_len    INT          NOT NULL,
				max_tokens    INT          NOT NULL DEFAULT 0,
				status 		  VARCHAR(63),
				details   	  TEXT,
		    	PRIMARY KEY (profile_id),
				UNIQUE (profile_id, model_name, model_version, vector_len)
			)
		`, tableProfiles)
	_, err = m.execTx(ctx, tx, query)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("error while creating profiles table [%s]: %w", query, err)
	}

	// Inserting data into the table
	query = fmt.Sprintf(`
		INSERT INTO %[1]s (profile_id, provider_name, model_name, model_version, vector_len, max_tokens)
		VALUES
			('openai-text-embedding-ada-002', 'openai', 'text-embedding-ada-002', '2', 1536, 8191),
			('openai-text-embedding-3-small', 'openai', 'text-embedding-3-small', '1', 1536, 8191),
			('openai-text-embedding-3-large', 'openai', 'text-embedding-3-large', '1', 3072, 8191),
			('amazon-titan-embed-text', 'amazon', 'titan-embed-text', '2', 1024, 8192),
			('google-text-multilingual-embedding-002', 'google', 'text-multilingual-embedding-002', '2', 768, 20000)
		ON CONFLICT (profile_id) DO NOTHING;
		`, tableProfiles)
	_, err = m.execTx(ctx, tx, query)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("error while inserting profiles [%s]: %w", query, err)
	}

	return nil
}

func (m *isoManager) createSchemaV2(ctx context.Context, tx *sql.Tx, isolationID string) error {
	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS vector_store_%s", db.GetMD5Hash(isolationID))
	_, err := m.execTx(ctx, tx, query)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("error while creating schema [%s]: %w", query, err)
	}
	return nil
}

func (m *isoManager) insertIsolationDetails(ctx context.Context, tx *sql.Tx, isolationID, maxStorageSize, pdcEndpointURL string) error {
	isolationsV2 := db.GetTableIsolations()
	query := fmt.Sprintf(
		"INSERT INTO %[1]s (iso_id, iso_prefix, max_storage_size, pdc_endpoint_url, created_at, modified_at, record_timestamp) "+
			"SELECT $1, md5($1), $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP "+
			"ON conflict (iso_id) DO UPDATE SET max_storage_size=$2, pdc_endpoint_url=$3, modified_at=CURRENT_TIMESTAMP, record_timestamp=CURRENT_TIMESTAMP", isolationsV2)
	_, err := m.execTx(ctx, tx, query, isolationID, maxStorageSize, pdcEndpointURL)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("error while inserting isolation data V2: %w", err)
	}

	return nil
}

func (m *isoManager) UpdateIsolation(ctx context.Context, isolationID, maxStorageSize, pdcEndpointURL string) (err error) {
	tx, err := m.database.GetConn().Begin()
	if err != nil {
		return fmt.Errorf("error while opening transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	err = m.insertIsolationDetails(ctx, tx, isolationID, maxStorageSize, pdcEndpointURL)
	if err != nil {
		return fmt.Errorf("error while updating isolation data: %w", err)
	}
	return tx.Commit()
}

func (m *isoManager) DeleteIsolation(ctx context.Context, isolationID string) error {
	exists, err := m.IsolationExists(ctx, isolationID)
	if err != nil {
		return fmt.Errorf("error while checking if isolation exists: %w", err)
	}
	if !exists {
		m.logger.Info("isolation does not exist. Nothing to do", zap.String("isolationID", isolationID))
		return nil
	}

	var tables []string

	collections, err := m.getCollections(ctx, isolationID)
	if err != nil {
		return err
	}

	tx, err := m.database.GetConn().Begin()
	if err != nil {
		return fmt.Errorf("error while opening transaction: %w", err)
	}
	defer func() {
		if err != nil {
			m.logger.Error("error during isolation deletion", zap.Error(err))
			_ = tx.Rollback()
		}
	}()

	// first delete all unprocessed items from embedding_queue
	_, err = m.execTx(ctx, tx, fmt.Sprintf("DELETE FROM %s WHERE content->>'iso_id'=$1", embeddingQueueTableV2), isolationID)
	if err != nil {
		return err
	}

	// wait for 1 second to allow the queue to be processed
	time.Sleep(1 * time.Second)

	// Drop V2 tables
	tables = []string{}
	for _, c := range collections {
		// TODO: EPIC-103866 / US-682862:
		// tables = append(tables, db.GetTableEmbStatistics(isolationID, c))
		tables = append(tables, db.GetTableEmbProcessing(isolationID, c))
		tables = append(tables, db.GetTableDocProcessing(isolationID, c))
		tables = append(tables, db.GetTableEmbMeta(isolationID, c))
		tables = append(tables, db.GetTableDocMeta(isolationID, c))
		tables = append(tables, db.GetTableAttr(isolationID, c))
		tables = append(tables, db.GetTableEmb(isolationID, c))
		tables = append(tables, db.GetTableDoc(isolationID, c))
	}
	// Cancel all backend processes querying the tables before dropping them
	for _, t := range tables {
		query := fmt.Sprintf(`
			BEGIN;
				-- look for any query locking the table and cancel it
				SELECT
					pg_cancel_backend(pid)
				FROM
					pg_stat_activity
				WHERE
					pid IN (SELECT pid FROM pg_locks)
				AND
					query LIKE '%%%[1]s%%'
				AND
					-- except this specific query
					query not like '%%pg_stat_activity%%';
				DROP TABLE IF EXISTS %[1]s;
			COMMIT;
        `, t)
		_, err = m.execTx(ctx, tx, query)
		if err != nil {
			return err
		}
	}

	attrGrpMgr := attributesgroup.NewManager(m.database, isolationID, m.logger)
	err = attrGrpMgr.DropTables(ctx)
	if err != nil {
		return fmt.Errorf("error while dropping attributes group tables: %w", err)
	}

	_, err = m.execTx(ctx, tx, fmt.Sprintf("DROP TABLE IF EXISTS %s", db.GetTableCollectionEmbeddingProfiles(isolationID)))
	if err != nil {
		return fmt.Errorf("error while dropping collection profiles table: %w", err)
	}

	_, err = m.execTx(ctx, tx, fmt.Sprintf("DROP TABLE IF EXISTS %s", db.GetTableCollections(isolationID)))
	if err != nil {
		return fmt.Errorf("error while dropping collections table: %w", err)
	}

	_, err = m.execTx(ctx, tx, fmt.Sprintf("DROP TABLE IF EXISTS %s", db.GetTableEmbeddingProfiles(isolationID)))
	if err != nil {
		return fmt.Errorf("error while dropping embedding profiles table: %w", err)
	}

	_, err = m.execTx(ctx, tx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s cascade", db.GetSchema(isolationID)))
	if err != nil {
		return fmt.Errorf("error while dropping schema: %w", err)
	}

	// then - delete isolation from isolations table
	_, err = m.execTx(ctx, tx, fmt.Sprintf("DELETE FROM %s WHERE iso_id=$1", m.isolationsTableV2), isolationID)
	if err != nil {
		return fmt.Errorf("error while deleting isolation from isolations table: %w", err)
	}

	return tx.Commit()
}

func (m *isoManager) getCollections(ctx context.Context, isolationID string) ([]string, error) {
	colTable := db.GetTableCollections(isolationID)
	query := fmt.Sprintf("SELECT col_id FROM %s", colTable)

	rows, err := m.query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error execuring query [%s]: %w", query, err)
	}
	defer rows.Close()

	var collections []string
	var coll string

	for rows.Next() {
		err = rows.Scan(&coll)
		if err != nil {
			return nil, fmt.Errorf("error while reading rows from query [%s]: %w", query, err)
		}
		collections = append(collections, coll)
	}
	return collections, nil
}
