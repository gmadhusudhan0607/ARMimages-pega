/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package db

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
)

var logger = log.GetNamedLogger("db")

type Query struct {
	SQL  string
	Args []any
}

// TODO (US-622684-1): Refactor make it more generic (UpsertDoc* methods)
type Database interface {
	GetConn() SQLDB
	GetSingleConn(ctx context.Context) (*sql.Conn, error)
	Query(ctx context.Context, query string) (*sql.Rows, error)
	UpsertDocStatus(ctx context.Context, table, docID, status, e string) error
	UpsertDocStatusTx(ctx context.Context, tx *sql.Tx, table, docID, status, e string) error
	// UpdateDocAttributesTx uses ` interface{}` because of circular dependency. To resolve this circular dependency, significant refactoring is needed.
	UpdateDocAttributesTx(ctx context.Context, tx *sql.Tx, table, docID string, attrIDs []int64, docAttributes interface{}) error
	UpdateEmbAttributesTx(ctx context.Context, tx *sql.Tx, table, docID string, attrIDs, attrIDs2 []int64) error
	DeleteDocMetadataTx(ctx context.Context, tx *sql.Tx, tableDocMeta, docID string) error
	InsertDocMetadataTx(ctx context.Context, tx *sql.Tx, tableDocMeta, docID, metadataKey, metadataValue string) error
}

type database struct {
	sqldb SQLDB
}

func (db *database) GetConn() SQLDB {
	return db.sqldb
}

func (db *database) GetSingleConn(ctx context.Context) (*sql.Conn, error) {
	return db.sqldb.Conn(ctx)
}

func NewDatabase(ctx context.Context, dbConfig *config.DatabaseConfig) (Database, error) {

	s, err := NewSQLDB(ctx, dbConfig)
	if err != nil {
		return nil, err
	}
	return &database{
		sqldb: s,
	}, nil
}

func GetTableName(isolationID, collectionName string, suffix string) string {
	dbname := fmt.Sprintf("i_%s_%s_%s", isolationID, collectionName, suffix)
	return strings.Replace(dbname, "-", "_", -1)
}

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func GetSchema(isolationID string) string {
	return fmt.Sprintf("vector_store_%s", GetMD5Hash(isolationID))
}

func GetTableIsolations() string {
	return "vector_store.isolations"
}
func GetTableCollections(isolationID string) string {
	return fmt.Sprintf("%s.collections", GetSchema(isolationID))
}
func GetTableEmbeddingProfiles(isolationID string) string {
	return fmt.Sprintf("%s.emb_profiles", GetSchema(isolationID))
}
func GetTableCollectionEmbeddingProfiles(isolationID string) string {
	return fmt.Sprintf("%s.collection_emb_profiles", GetSchema(isolationID))
}
func GetTableDoc(isolationID, collectionID string) string {
	return fmt.Sprintf("%s.t_%s_doc", GetSchema(isolationID), GetMD5Hash(collectionID))
}
func GetTableDocWithPrefix(isolationID, tablePrefix string) string {
	return fmt.Sprintf("%s.t_%s_doc", GetSchema(isolationID), tablePrefix)
}
func GetTableEmb(isolationID, collectionID string) string {
	return fmt.Sprintf("%s.t_%s_emb", GetSchema(isolationID), GetMD5Hash(collectionID))
}
func GetTableAttr(isolationID, collectionID string) string {
	return fmt.Sprintf("%s.t_%s_attr", GetSchema(isolationID), GetMD5Hash(collectionID))
}
func GetTableSmartAttrGroup(isolationID string) string {
	return fmt.Sprintf("%s.smart_attributes_group", GetSchema(isolationID))
}
func GetTableDocMeta(isolationID, collectionID string) string {
	return fmt.Sprintf("%s.t_%s_doc_meta", GetSchema(isolationID), GetMD5Hash(collectionID))
}
func GetTableEmbMeta(isolationID, collectionID string) string {
	return fmt.Sprintf("%s.t_%s_emb_meta", GetSchema(isolationID), GetMD5Hash(collectionID))
}
func GetTableDocProcessing(isolationID, collectionID string) string {
	return fmt.Sprintf("%s.t_%s_doc_processing", GetSchema(isolationID), GetMD5Hash(collectionID))
}
func GetTableEmbProcessing(isolationID, collectionID string) string {
	return fmt.Sprintf("%s.t_%s_emb_processing", GetSchema(isolationID), GetMD5Hash(collectionID))
}
func GetTableEmbStatistics(isolationID, collectionID string) string {
	return fmt.Sprintf("%s.t_%s_emb_statistics", GetSchema(isolationID), GetMD5Hash(collectionID))
}

func (db *database) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return db.GetConn().Query(query)
}

// TODO (US-622684-1): Refactor make it more generic
func (db *database) UpsertDocStatus(ctx context.Context, table, docID, status, e string) (err error) {
	query := fmt.Sprintf(`
		INSERT INTO %[1]s
          (doc_id, status, error_message, created_at, modified_at, record_timestamp)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) 
		ON conflict (doc_id) DO UPDATE SET 
			status=$2, error_message=$3, modified_at=CURRENT_TIMESTAMP, record_timestamp=CURRENT_TIMESTAMP
        WHERE %[1]s.doc_id=$1
    `, table)
	_, err = db.GetConn().ExecContext(ctx, query, docID, status, e)
	if err != nil {
		return fmt.Errorf("error while updating document status [%s]: %w", query, err)
	}
	return err
}

// TODO (US-622684-1): Refactor make it more generic
func (db *database) UpsertDocStatusTx(ctx context.Context, tx *sql.Tx, table, docID, status, e string) (err error) {
	query := fmt.Sprintf(`
		INSERT INTO %[1]s
          (doc_id, status, error_message, created_at, modified_at, record_timestamp)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) 
		ON conflict (doc_id) DO UPDATE SET 
			status=$2, error_message=$3, modified_at=CURRENT_TIMESTAMP, record_timestamp=CURRENT_TIMESTAMP
        WHERE %[1]s.doc_id=$1
    `, table)
	_, err = tx.ExecContext(ctx, query, docID, status, e)
	if err != nil {
		return fmt.Errorf("error while updating document status [%s]: %w", query, err)
	}
	return err
}

// TODO (US-622684-1): Move to documents package
// UpdateDocAttributesTx uses ` interface{}` because of circular dependency. To resolve this circular dependency, significant refactoring is needed.
func (db *database) UpdateDocAttributesTx(ctx context.Context, tx *sql.Tx, table, docID string, attrIDs []int64, docAttributes interface{}) error {
	var err error
	var docAttributesJSON []byte

	// Handle different input types for backward compatibility
	switch v := docAttributes.(type) {
	case []byte:
		// If it's already JSON bytes, use directly
		docAttributesJSON = v
	case string:
		// If it's a JSON string, convert to bytes
		docAttributesJSON = []byte(v)
	default:
		// For any other type (including AttributesV2), marshal to JSON
		docAttributesJSON, err = json.Marshal(docAttributes)
		if err != nil {
			return fmt.Errorf("error marshalling document attributes: %w", err)
		}
	}

	query := fmt.Sprintf(`UPDATE %[1]s SET attr_ids = $2::bigint[], doc_attributes = $3::jsonb, record_timestamp=CURRENT_TIMESTAMP WHERE doc_id = $1 `, table)
	_, err = tx.ExecContext(ctx, query, docID, attrIDs, docAttributesJSON)
	return err
}

// TODO (US-622684-1): Move to documents package
func (db *database) UpdateEmbAttributesTx(ctx context.Context, tx *sql.Tx, table, docID string, attrIDs, attrIDs2 []int64) (err error) {
	sort.Slice(attrIDs, func(i, j int) bool { return attrIDs[i] < attrIDs[j] })
	sort.Slice(attrIDs2, func(i, j int) bool { return attrIDs2[i] < attrIDs2[j] })
	query := fmt.Sprintf(`UPDATE %[1]s SET attr_ids = $2::bigint[], attr_ids_2 = $3::bigint[], record_timestamp=CURRENT_TIMESTAMP WHERE doc_id = $1 `, table)
	_, err = tx.ExecContext(ctx, query, docID, attrIDs, attrIDs2)
	if err != nil {
		return fmt.Errorf("error while updating emb attributes (docID: %s, attrIDs: %v, attrIDs2: %v) [%s]: %w",
			docID, attrIDs, attrIDs2, query, err)
	}
	return nil
}

func (db *database) DeleteDocMetadataTx(ctx context.Context, tx *sql.Tx, tableDocMeta, docID string) (err error) {
	query := fmt.Sprintf(`DELETE FROM %[1]s WHERE doc_id = $1`, tableDocMeta)
	_, err = tx.ExecContext(ctx, query, docID)
	if err != nil {
		return fmt.Errorf("error while removing document metadata [%s]: %w", query, err)
	}
	return nil
}

func (db *database) InsertDocMetadataTx(ctx context.Context, tx *sql.Tx, tableDocMeta, docID, metadataKey, metadataValue string) (err error) {
	query := fmt.Sprintf(`
			INSERT INTO %[1]s (doc_id, metadata_key, metadata_value, modified_at) 
		    VALUES ($1, $2, $3, CURRENT_TIMESTAMP)`, tableDocMeta)
	_, err = tx.ExecContext(ctx, query, docID, metadataKey, metadataValue)
	if err != nil {
		return fmt.Errorf("error while inserting document metadata [%s]: %w", query, err)
	}
	return nil
}
