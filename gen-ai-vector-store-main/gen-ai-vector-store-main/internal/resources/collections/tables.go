/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package collections

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers/hnsw"
)

func (m *colManager) createTables(ctx context.Context, collectionID string) (err error) {
	//start transaction for document status
	tx0, err1 := m.database.GetConn().Begin()
	if err1 != nil {
		return fmt.Errorf("error while opening transaction when creating tables for collection: %w", err1)
	}
	defer func() {
		if err != nil {
			m.logger.Warn("failed creating tables for collection", zap.Error(err))
			_ = tx0.Rollback()
		}
	}()

	tableDoc := db.GetTableDoc(m.IsolationID, collectionID)
	if err = m.createTableDoc(ctx, tx0, tableDoc); err != nil {
		return err
	}

	tableEmb := db.GetTableEmb(m.IsolationID, collectionID)
	if err = m.createTableEmb(ctx, tx0, tableDoc, tableEmb, collectionID, DefaultEmbeddingProfileID); err != nil {
		return err
	}

	tableAttr := db.GetTableAttr(m.IsolationID, collectionID)
	if err = m.createTableAttrs(ctx, tx0, tableAttr); err != nil {
		return err
	}

	tableDocMeta := db.GetTableDocMeta(m.IsolationID, collectionID)
	if err = m.createTableDocMeta(ctx, tx0, tableDoc, tableDocMeta); err != nil {
		return err
	}

	tableEmbMeta := db.GetTableEmbMeta(m.IsolationID, collectionID)
	if err = m.createTableEmbMeta(ctx, tx0, tableEmb, tableEmbMeta); err != nil {
		return err
	}

	tableDocProcessing := db.GetTableDocProcessing(m.IsolationID, collectionID)
	if err = m.createTableDocProcessing(ctx, tx0, tableDocProcessing, tableDoc); err != nil {
		return err
	}

	tableEmbProcessing := db.GetTableEmbProcessing(m.IsolationID, collectionID)
	if err = m.createTableEmbProcessing(ctx, tx0, tableDocProcessing, tableEmbProcessing); err != nil {
		return err
	}

	// TODO: EPIC-103866 / US-682862:
	// tableEmbStats := db.GetTableEmbStatistics(m.IsolationID, collectionID)
	// if err = m.createTableEmbStats(ctx, tx0, tableDoc, tableEmb, tableEmbStats); err != nil {
	// 	return err
	// }

	return tx0.Commit()
}

func (m *colManager) createTableDoc(ctx context.Context, tx0 *sql.Tx, tableDoc string) (err error) {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
			doc_id             TEXT NOT NULL,
			status             TEXT,
			error_message      TEXT,
			attr_ids           bigint[],
			doc_attributes     JSONB DEFAULT NULL,
			created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			modified_at        TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			record_timestamp   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  PRIMARY KEY (doc_id)
		)`, tableDoc)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	_, tableDocWithoutSchema := helpers.SplitTableName(tableDoc)

	// create index on attr_ids
	idxName := fmt.Sprintf("idx_%s__attrids", tableDocWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (attr_ids) WHERE attr_ids IS NOT NULL
	`, tableDoc, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on doc_attributes (path ops)
	idxName = fmt.Sprintf("idx_%s_doc_attributes_path", tableDocWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (doc_attributes jsonb_path_ops)
	`, tableDoc, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on doc_attributes (ops)
	idxName = fmt.Sprintf("idx_%s_doc_attributes_ops", tableDocWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (doc_attributes jsonb_ops)
	`, tableDoc, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	return nil
}

func (m *colManager) createTableEmb(ctx context.Context, tx0 *sql.Tx, tableDoc, tableEmb string, collectionID string, profileID string) (err error) {
	vectorLength, err := m.getVectorLength(ctx, tx0, profileID)
	if err != nil {
		return fmt.Errorf("error while getting vector length for profile %s: %w", profileID, err)
	}

	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
			emb_id             TEXT NOT NULL,
			doc_id             TEXT REFERENCES %[2]s (doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
			content            TEXT,
			embedding          vector(%[3]d),
			status             TEXT,
			response_code      INT DEFAULT 0,
			error_message      TEXT,
			attr_ids           bigint[],
			attr_ids2          bigint[],  --> attribute IDs of ( embedding + document)
			emb_attributes     JSONB DEFAULT NULL,
			attributes         JSONB DEFAULT NULL,
			modified_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			record_timestamp   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  PRIMARY KEY (emb_id)
		)`, tableEmb, tableDoc, vectorLength)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	_, tableEmbWithoutSchema := helpers.SplitTableName(tableEmb)

	// create index on doc_id
	idxName := fmt.Sprintf("idx_%s__docid", tableEmbWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s (doc_id)
	`, tableEmb, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on record_timestamp
	idxName = fmt.Sprintf("idx_%s__rts", tableEmbWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s (record_timestamp) WHERE record_timestamp IS NOT NULL
	`, tableEmb, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on attr_ids
	idxName = fmt.Sprintf("idx_%s__attrids", tableEmbWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (attr_ids) WHERE attr_ids IS NOT NULL
	`, tableEmb, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on attr_ids2
	idxName = fmt.Sprintf("idx_%s__attrids2", tableEmbWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (attr_ids2) WHERE attr_ids2 IS NOT NULL
	`, tableEmb, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on emb_attributes (path ops)
	idxName = fmt.Sprintf("idx_%s_emb_attributes_path", tableEmbWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (emb_attributes jsonb_path_ops)
	`, tableEmb, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on emb_attributes (ops)
	idxName = fmt.Sprintf("idx_%s_emb_attributes_ops", tableEmbWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (emb_attributes jsonb_ops)
	`, tableEmb, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on attributes (path ops)
	idxName = fmt.Sprintf("idx_%s_attributes_path", tableEmbWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (attributes jsonb_path_ops)
	`, tableEmb, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on attributes (ops)
	idxName = fmt.Sprintf("idx_%s_attributes_ops", tableEmbWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (attributes jsonb_ops)
	`, tableEmb, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	// create index on embedding
	idxBuildParamsQuery, err := hnsw.BuildSetParametersQuery(ctx, m.logger, tx0, m.IsolationID, collectionID, profileID)
	if err != nil {
		return fmt.Errorf("failed to build hnsw set parameters query for isolation %s, collection %s, profile %s: %w", m.IsolationID, collectionID, profileID, err)
	}
	createIdxQuery, err := hnsw.BuildCreateIndexQuery(ctx, m.logger, tx0, m.IsolationID, collectionID, profileID)
	if err != nil {
		return fmt.Errorf("failed to build HNSW index query for isolation %s, collection %s, profile %s: %w", m.IsolationID, collectionID, profileID, err)
	}
	// it is fine to concatenate, since we run in a transaction explicitely via *sql.Tx
	query = fmt.Sprintf("%s\n%s", idxBuildParamsQuery, createIdxQuery)

	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	return nil
}

func (m *colManager) getVectorLength(ctx context.Context, tx *sql.Tx, profileID string) (int, error) {
	vecLengthQuery := fmt.Sprintf("SELECT vector_len FROM %s WHERE profile_id = $1", db.GetTableEmbeddingProfiles(m.IsolationID))
	rows, err := m.queryTx(ctx, tx, vecLengthQuery, profileID)
	if err != nil {
		return 0, fmt.Errorf("error while executing query [%s] for profile %s: %w", vecLengthQuery, profileID, err)
	}

	defer rows.Close()

	if !rows.Next() {
		return 0, fmt.Errorf("no vector length found for profile %s in isolation %s", profileID, m.IsolationID)
	}

	var vectorLength int
	err = rows.Scan(&vectorLength)
	if err != nil {
		return 0, fmt.Errorf("error while reading row from query [%s] for profile %s: %w", vecLengthQuery, profileID, err)
	}

	return vectorLength, nil
}

func (m *colManager) createTableAttrs(ctx context.Context, tx0 *sql.Tx, tableAttr string) (err error) {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[1]s (
			attr_id  BIGSERIAL NOT NULL UNIQUE,
			name     TEXT NOT NULL,
			type     TEXT NOT NULL,
			value    TEXT NOT NULL,
			value_hash VARCHAR(40) NOT NULL,
		  PRIMARY  KEY (attr_id),
		  UNIQUE (name, type, value_hash)
		)`, tableAttr)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	_, tableAttrWithoutSchema := helpers.SplitTableName(tableAttr)

	// create index on (name, type, value_hash) for fast lookup
	idxName := fmt.Sprintf("idx_%s_kth", tableAttrWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s (name, type, value_hash)
	`, tableAttr, idxName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	return nil
}

func (m *colManager) dropTables(ctx context.Context, tx0 *sql.Tx, collectionID string) (err error) {
	// TODO: EPIC-103866 / US-682862:
	// if err = m.dropTable(ctx, tx0, db.GetTableEmbStatistics(m.IsolationID, collectionID)); err != nil {
	// 	return err
	// }
	if err = m.dropTable(ctx, tx0, db.GetTableEmbProcessing(m.IsolationID, collectionID)); err != nil {
		return err
	}
	if err = m.dropTable(ctx, tx0, db.GetTableDocProcessing(m.IsolationID, collectionID)); err != nil {
		return err
	}
	if err = m.dropTable(ctx, tx0, db.GetTableEmbMeta(m.IsolationID, collectionID)); err != nil {
		return err
	}
	if err = m.dropTable(ctx, tx0, db.GetTableDocMeta(m.IsolationID, collectionID)); err != nil {
		return err
	}
	if err = m.dropTable(ctx, tx0, db.GetTableAttr(m.IsolationID, collectionID)); err != nil {
		return err
	}
	if err = m.dropTable(ctx, tx0, db.GetTableEmb(m.IsolationID, collectionID)); err != nil {
		return err
	}
	if err = m.dropTable(ctx, tx0, db.GetTableDoc(m.IsolationID, collectionID)); err != nil {
		return err
	}
	return nil
}

func (m *colManager) dropTable(ctx context.Context, tx0 *sql.Tx, tableName string) (err error) {
	query := fmt.Sprintf(`DROP TABLE IF EXISTS %[1]s`, tableName)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}
	return nil
}

func (m *colManager) createTableDocMeta(ctx context.Context, tx0 *sql.Tx, tableDoc, tableDocMeta string) (err error) {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[2]s (
			doc_id           text references %[1]s(doc_id) on delete cascade,
			metadata_key     text not null,
			metadata_value   text,
			modified_at      timestamp default CURRENT_TIMESTAMP not null,
			unique (doc_id, metadata_key),
			primary key (doc_id, metadata_key)
		)`, tableDoc, tableDocMeta)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}
	return nil
}

func (m *colManager) createTableEmbMeta(ctx context.Context, tx0 *sql.Tx, tableEmb, tableEmbMeta string) (err error) {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %[2]s (
			emb_id           text references %[1]s(emb_id) on delete cascade,
			metadata_key     text not null,
			metadata_value   text,
			modified_at      timestamp default CURRENT_TIMESTAMP not null,
			unique (emb_id, metadata_key),
			primary key (emb_id, metadata_key)
		)`, tableEmb, tableEmbMeta)
	if _, err = m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}
	return nil
}

func (m *colManager) createTableDocProcessing(ctx context.Context, tx0 *sql.Tx, tableDocProcessing, tableDoc string) error {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %[1]s (
		doc_id             TEXT REFERENCES %[2]s (doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
		created_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        heartbeat          TIMESTAMP,
		record_timestamp   TIMESTAMP,
		error_message      TEXT,
		retry_count        INTEGER,
		attr_ids	       BIGINT[],
		doc_attributes     JSONB DEFAULT NULL,
		doc_metadata       JSONB,
		file               BYTEA,
	    PRIMARY KEY (doc_id)
	)`, tableDocProcessing, tableDoc)

	if _, err := m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	_, tableDocProcessingWithoutSchema := helpers.SplitTableName(tableDocProcessing)

	// create index on attr_ids
	idxName := fmt.Sprintf("idx_%s__attrids", tableDocProcessingWithoutSchema)
	query = fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %[2]s ON %[1]s USING GIN (attr_ids) WHERE attr_ids IS NOT NULL
	`, tableDocProcessing, idxName)
	if _, err := m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	return nil
}

func (m *colManager) createTableEmbProcessing(ctx context.Context, tx0 *sql.Tx, tableDocProcessing, tableEmbProcessing string) error {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %[1]s (
		emb_id             TEXT NOT NULL,
		doc_id             TEXT REFERENCES %[2]s (doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
		content            TEXT,
		embedding          vector,
		status             TEXT,
		attr_ids		   BIGINT[],
		emb_attributes     JSONB DEFAULT NULL,
		attributes         JSONB DEFAULT NULL,
		metadata		   JSONB,
		response_code      INT DEFAULT 0,
		error_message      TEXT,
		retry_count        INTEGER,
		start_time         TIMESTAMP,
		end_time           TIMESTAMP,
		record_timestamp   TIMESTAMP,
		token_count        INTEGER,
	    PRIMARY KEY (emb_id)
	)`, tableEmbProcessing, tableDocProcessing)

	if _, err := m.execTx(ctx, tx0, query); err != nil {
		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
	}

	return nil
}

// TODO: EPIC-103866 / US-682862:
// func (m *colManager) createTableEmbStats(ctx context.Context, tx0 *sql.Tx, tableDoc, tableEmb, tableEmbStats string) error {
// 	query := fmt.Sprintf(`
// 	CREATE TABLE IF NOT EXISTS %[1]s (
// 		doc_id               TEXT REFERENCES %[2]s (doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
// 		emb_id               TEXT REFERENCES %[3]s (emb_id) ON DELETE CASCADE ON UPDATE CASCADE,
// 		retry_count          INT,
// 		token_count          INT,
// 		start_time           TIMESTAMP,
// 		end_time             TIMESTAMP,
// 		last_llm_duration_ms INT,
// 		last_error_message   TEXT
// 	)`, tableEmbStats, tableDoc, tableEmb)
// 	if _, err := m.execTx(ctx, tx0, query); err != nil {
// 		return fmt.Errorf("error while executing query [ %s ]: %w", query, err)
// 	}

// 	return nil
// }
