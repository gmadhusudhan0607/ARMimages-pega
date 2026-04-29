-- Copyright (c) 2025 Pegasystems Inc.
-- All rights reserved.
--
-- Test data for schema version v0.18.0
-- This represents the database state BEFORE migration to v0.19.0
-- IMPORTANT: Does NOT include columns added by NewMigrationV0x19x0:
--   - doc_attributes JSONB
--   - emb_attributes JSONB
--   - attributes JSONB

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create vector_store schema
CREATE SCHEMA IF NOT EXISTS vector_store;

-- Create base tables
CREATE TABLE IF NOT EXISTS vector_store.isolations (
    iso_id           TEXT        NOT NULL PRIMARY KEY,
    iso_prefix       VARCHAR(40) NOT NULL UNIQUE,
    max_storage_size TEXT,
    created_at       TIMESTAMP,
    modified_at      TIMESTAMP,
    record_timestamp TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS vector_store.configuration (
    key   TEXT NOT NULL PRIMARY KEY,
    value TEXT
);

-- Set schema version to v0.18.0
INSERT INTO vector_store.configuration (key, value) VALUES ('schema_version', 'v0.18.0');

-- Create test isolations
INSERT INTO vector_store.isolations (iso_id, iso_prefix, max_storage_size, created_at, modified_at)
VALUES 
    ('iso-empty', md5('iso-empty'), '1GB', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('iso-test-1', md5('iso-test-1'), '2GB', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('iso-test-2', md5('iso-test-2'), '2GB', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- Create schemas for test isolations
CREATE SCHEMA IF NOT EXISTS vector_store_e5af55acf1ba36f60b1e55a5d57f4b7d; -- iso-empty
CREATE SCHEMA IF NOT EXISTS vector_store_6f909e9b46455b62a7337a75311a25eb; -- iso-test-1
CREATE SCHEMA IF NOT EXISTS vector_store_76a3008adb7cb8f988ba492ad034e815; -- iso-test-2

-- Create collections table for iso-empty (no collections)
CREATE TABLE IF NOT EXISTS vector_store_e5af55acf1ba36f60b1e55a5d57f4b7d.collections (
    col_id           TEXT NOT NULL,
    col_prefix       VARCHAR(40) NOT NULL UNIQUE,
    default_emb_profile VARCHAR(127) NOT NULL,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id)
);

-- Create embedding profiles table for iso-empty
CREATE TABLE IF NOT EXISTS vector_store_e5af55acf1ba36f60b1e55a5d57f4b7d.embedding_profiles (
    profile_id    VARCHAR(127) NOT NULL,
    provider_name VARCHAR(63)  NOT NULL,
    model_name    VARCHAR(127) NOT NULL,
    model_version VARCHAR(63)  NOT NULL,
    vector_len    INT          NOT NULL,
    max_tokens    INT          NOT NULL DEFAULT 0,
    status        VARCHAR(63),
    details       TEXT,
    PRIMARY KEY (profile_id),
    UNIQUE (profile_id, model_name, model_version, vector_len)
);

-- Initialize default embedding profile for iso-empty
INSERT INTO vector_store_e5af55acf1ba36f60b1e55a5d57f4b7d.embedding_profiles 
    (profile_id, provider_name, model_name, model_version, vector_len, max_tokens)
VALUES 
    ('openai-text-embedding-ada-002', 'openai', 'text-embedding-ada-002', '2', 1536, 8191);

-- Create collection embedding profiles table for iso-empty
CREATE TABLE IF NOT EXISTS vector_store_e5af55acf1ba36f60b1e55a5d57f4b7d.collection_emb_profiles (
    col_id           TEXT NOT NULL,
    profile_id       VARCHAR(127) NOT NULL,
    tables_prefix    VARCHAR(40) NOT NULL,
    status           VARCHAR(63),
    details          TEXT,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id, profile_id),
    UNIQUE (tables_prefix)
);

-- Setup for iso-test-1 with 2 collections
CREATE TABLE IF NOT EXISTS vector_store_6f909e9b46455b62a7337a75311a25eb.collections (
    col_id           TEXT NOT NULL,
    col_prefix       VARCHAR(40) NOT NULL UNIQUE,
    default_emb_profile VARCHAR(127) NOT NULL,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id)
);

CREATE TABLE IF NOT EXISTS vector_store_6f909e9b46455b62a7337a75311a25eb.embedding_profiles (
    profile_id    VARCHAR(127) NOT NULL,
    provider_name VARCHAR(63)  NOT NULL,
    model_name    VARCHAR(127) NOT NULL,
    model_version VARCHAR(63)  NOT NULL,
    vector_len    INT          NOT NULL,
    max_tokens    INT          NOT NULL DEFAULT 0,
    status        VARCHAR(63),
    details       TEXT,
    PRIMARY KEY (profile_id),
    UNIQUE (profile_id, model_name, model_version, vector_len)
);

INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.embedding_profiles 
    (profile_id, provider_name, model_name, model_version, vector_len, max_tokens)
VALUES 
    ('openai-text-embedding-ada-002', 'openai', 'text-embedding-ada-002', '2', 1536, 8191);

CREATE TABLE IF NOT EXISTS vector_store_6f909e9b46455b62a7337a75311a25eb.collection_emb_profiles (
    col_id           TEXT NOT NULL,
    profile_id       VARCHAR(127) NOT NULL,
    tables_prefix    VARCHAR(40) NOT NULL,
    status           VARCHAR(63),
    details          TEXT,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id, profile_id),
    UNIQUE (tables_prefix)
);

-- Insert collections for iso-test-1
INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.collections 
    (col_id, col_prefix, default_emb_profile)
VALUES 
    ('col-1a', md5('col-1a'), 'openai-text-embedding-ada-002'),
    ('col-1b', md5('col-1b'), 'openai-text-embedding-ada-002');

INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.collection_emb_profiles 
    (col_id, profile_id, tables_prefix, status)
VALUES 
    ('col-1a', 'openai-text-embedding-ada-002', md5('col-1a'), 'READY'),
    ('col-1b', 'openai-text-embedding-ada-002', md5('col-1b'), 'READY');

-- Create tables for collection col-1a (iso-test-1)
-- Note: These tables do NOT have doc_attributes, emb_attributes, or attributes columns (v0.18.0)
CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc (
    doc_id           TEXT NOT NULL,
    status           TEXT,
    error_message    TEXT,
    attr_ids         BIGINT[],
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (doc_id)
);

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_emb (
    emb_id           TEXT NOT NULL,
    doc_id           TEXT NOT NULL,
    content          TEXT,
    embedding        vector(1536),
    status           TEXT,
    response_code    INT DEFAULT 0,
    error_message    TEXT,
    attr_ids         BIGINT[],
    attr_ids2        BIGINT[],
    modified_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (emb_id),
    FOREIGN KEY (doc_id) REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc(doc_id) ON DELETE CASCADE
);

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_attr (
    attr_id    BIGSERIAL NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    value      TEXT NOT NULL,
    value_hash VARCHAR(40) NOT NULL,
    PRIMARY KEY (attr_id),
    UNIQUE (name, type, value_hash)
);

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc_meta (
    doc_id         TEXT NOT NULL,
    metadata_key   TEXT NOT NULL,
    metadata_value TEXT,
    modified_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (doc_id, metadata_key),
    FOREIGN KEY (doc_id) REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc(doc_id) ON DELETE CASCADE
);

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_emb_meta (
    emb_id         TEXT NOT NULL,
    metadata_key   TEXT NOT NULL,
    metadata_value TEXT,
    modified_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (emb_id, metadata_key),
    FOREIGN KEY (emb_id) REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_emb(emb_id) ON DELETE CASCADE
);

-- Create processing tables for col-1a (added in v0.17.0)
CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc_processing (
    doc_id TEXT REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc(doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    heartbeat TIMESTAMP,
    record_timestamp TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER,
    attr_ids BIGINT[],
    doc_metadata JSONB,
    file BYTEA,
    PRIMARY KEY (doc_id)
);

CREATE INDEX idx_t_f5e462a02802922fa7e21ece51498c05_doc_processing__attrids 
    ON vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc_processing 
    USING GIN (attr_ids) WHERE attr_ids IS NOT NULL;

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_emb_processing (
    emb_id TEXT NOT NULL,
    doc_id TEXT REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc_processing(doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
    content TEXT,
    embedding vector(1536),
    status TEXT,
    attr_ids BIGINT[],
    metadata JSONB,
    response_code INT DEFAULT 0,
    error_message TEXT,
    retry_count INTEGER,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    record_timestamp TIMESTAMP,
    token_count INTEGER,
    PRIMARY KEY (emb_id)
);

-- Insert test data for col-1a with attributes in old format (metadata tables)
INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_attr 
    (attr_id, name, type, value, value_hash)
VALUES 
    (1, 'Document type', 'string', 'Article', md5('Article')),
    (2, 'Category', 'string', 'Technology', md5('Technology')),
    (3, 'Author', 'string', 'John Doe', md5('John Doe')),
    (4, 'Priority', 'string', 'High', md5('High')),
    (5, 'Priority', 'string', 'Medium', md5('Medium')),
    (6, 'Priority', 'string', 'Low', md5('Low')),
    (7, 'Status', 'string', 'Draft', md5('Draft')),
    (8, 'Status', 'string', 'Published', md5('Published')),
    (9, 'Department', 'string', 'Engineering', md5('Engineering')),
    (10, 'Department', 'string', 'Marketing', md5('Marketing'));

-- Insert many documents for testing progress metric during migration
-- This generates 100 documents to ensure migration takes enough time to capture intermediate progress
-- while still completing in a reasonable time frame (around 30-40 seconds with slow settings)
DO $$
DECLARE
    i INT;
    doc_id_val TEXT;
    emb_id_val TEXT;
    attr_set BIGINT[];
BEGIN
    FOR i IN 1..100 LOOP
        doc_id_val := 'doc-1a-bulk-' || i;
        emb_id_val := 'emb-1a-bulk-' || i;
        
        -- Vary attributes to create realistic data
        CASE (i % 4)
            WHEN 0 THEN attr_set := ARRAY[1, 2, 4, 9];
            WHEN 1 THEN attr_set := ARRAY[1, 2, 5, 10];
            WHEN 2 THEN attr_set := ARRAY[1, 2, 6, 9];
            ELSE attr_set := ARRAY[1, 2, 3];
        END CASE;
        
        -- Insert document
        INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc 
            (doc_id, status, attr_ids)
        VALUES 
            (doc_id_val, 'COMPLETED', attr_set);
        
        -- Insert document metadata
        INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc_meta 
            (doc_id, metadata_key, metadata_value)
        SELECT doc_id_val, a.name, a.value
        FROM vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_attr a
        WHERE a.attr_id = ANY(attr_set);
        
        -- Insert embedding (typically 3-5 embeddings per document)
        FOR j IN 1..3 LOOP
            INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_emb 
                (emb_id, doc_id, content, embedding, status, attr_ids, attr_ids2)
            VALUES 
                (emb_id_val || '-' || j, doc_id_val, 'Content for chunk ' || j,
                 (SELECT ARRAY_AGG(random()::real) FROM generate_series(1, 1536))::vector,
                 'COMPLETED', ARRAY[attr_set[1]], attr_set);
            
            -- Insert embedding metadata (1-2 attributes per embedding)
            INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_emb_meta 
                (emb_id, metadata_key, metadata_value)
            SELECT emb_id_val || '-' || j, a.name, a.value
            FROM vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_attr a
            WHERE a.attr_id = attr_set[1];
        END LOOP;
    END LOOP;
END $$;

-- Also insert the original test document for compatibility with existing tests
INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc 
    (doc_id, status, attr_ids)
VALUES 
    ('doc-1a-1', 'COMPLETED', ARRAY[1, 2, 3]);

INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_doc_meta 
    (doc_id, metadata_key, metadata_value)
VALUES 
    ('doc-1a-1', 'Document type', 'Article'),
    ('doc-1a-1', 'Category', 'Technology'),
    ('doc-1a-1', 'Author', 'John Doe');

INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_emb 
    (emb_id, doc_id, content, embedding, status, attr_ids, attr_ids2)
VALUES 
    ('emb-1a-1', 'doc-1a-1', 'Test content',
     (SELECT ARRAY_AGG(random()::real) FROM generate_series(1, 1536))::vector,
     'COMPLETED', ARRAY[2], ARRAY[1, 2, 3]);

INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.t_f5e462a02802922fa7e21ece51498c05_emb_meta 
    (emb_id, metadata_key, metadata_value)
VALUES 
    ('emb-1a-1', 'Category', 'Technology');

-- Create tables for collection col-1b (iso-test-1) - similar structure
CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_doc (
    doc_id           TEXT NOT NULL,
    status           TEXT,
    error_message    TEXT,
    attr_ids         BIGINT[],
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (doc_id)
);

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_emb (
    emb_id           TEXT NOT NULL,
    doc_id           TEXT NOT NULL,
    content          TEXT,
    embedding        vector(1536),
    status           TEXT,
    response_code    INT DEFAULT 0,
    error_message    TEXT,
    attr_ids         BIGINT[],
    attr_ids2        BIGINT[],
    modified_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (emb_id),
    FOREIGN KEY (doc_id) REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_doc(doc_id) ON DELETE CASCADE
);

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_attr (
    attr_id    BIGSERIAL NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    value      TEXT NOT NULL,
    value_hash VARCHAR(40) NOT NULL,
    PRIMARY KEY (attr_id),
    UNIQUE (name, type, value_hash)
);

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_doc_meta (
    doc_id         TEXT NOT NULL,
    metadata_key   TEXT NOT NULL,
    metadata_value TEXT,
    modified_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (doc_id, metadata_key),
    FOREIGN KEY (doc_id) REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_doc(doc_id) ON DELETE CASCADE
);

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_emb_meta (
    emb_id         TEXT NOT NULL,
    metadata_key   TEXT NOT NULL,
    metadata_value TEXT,
    modified_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (emb_id, metadata_key),
    FOREIGN KEY (emb_id) REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_emb(emb_id) ON DELETE CASCADE
);

-- Create processing tables for col-1b (added in v0.17.0)
CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_doc_processing (
    doc_id TEXT REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_doc(doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    heartbeat TIMESTAMP,
    record_timestamp TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER,
    attr_ids BIGINT[],
    doc_metadata JSONB,
    file BYTEA,
    PRIMARY KEY (doc_id)
);

CREATE INDEX idx_t_f7322bda811fd991d972a9651021157e_doc_processing__attrids 
    ON vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_doc_processing 
    USING GIN (attr_ids) WHERE attr_ids IS NOT NULL;

CREATE TABLE vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_emb_processing (
    emb_id TEXT NOT NULL,
    doc_id TEXT REFERENCES vector_store_6f909e9b46455b62a7337a75311a25eb.t_f7322bda811fd991d972a9651021157e_doc_processing(doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
    content TEXT,
    embedding vector(1536),
    status TEXT,
    attr_ids BIGINT[],
    metadata JSONB,
    response_code INT DEFAULT 0,
    error_message TEXT,
    retry_count INTEGER,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    record_timestamp TIMESTAMP,
    token_count INTEGER,
    PRIMARY KEY (emb_id)
);

-- Setup for iso-test-2 with 1 collection
CREATE TABLE IF NOT EXISTS vector_store_76a3008adb7cb8f988ba492ad034e815.collections (
    col_id           TEXT NOT NULL,
    col_prefix       VARCHAR(40) NOT NULL UNIQUE,
    default_emb_profile VARCHAR(127) NOT NULL,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id)
);

CREATE TABLE IF NOT EXISTS vector_store_76a3008adb7cb8f988ba492ad034e815.embedding_profiles (
    profile_id    VARCHAR(127) NOT NULL,
    provider_name VARCHAR(63)  NOT NULL,
    model_name    VARCHAR(127) NOT NULL,
    model_version VARCHAR(63)  NOT NULL,
    vector_len    INT          NOT NULL,
    max_tokens    INT          NOT NULL DEFAULT 0,
    status        VARCHAR(63),
    details       TEXT,
    PRIMARY KEY (profile_id),
    UNIQUE (profile_id, model_name, model_version, vector_len)
);

INSERT INTO vector_store_76a3008adb7cb8f988ba492ad034e815.embedding_profiles 
    (profile_id, provider_name, model_name, model_version, vector_len, max_tokens)
VALUES 
    ('openai-text-embedding-ada-002', 'openai', 'text-embedding-ada-002', '2', 1536, 8191);

CREATE TABLE IF NOT EXISTS vector_store_76a3008adb7cb8f988ba492ad034e815.collection_emb_profiles (
    col_id           TEXT NOT NULL,
    profile_id       VARCHAR(127) NOT NULL,
    tables_prefix    VARCHAR(40) NOT NULL,
    status           VARCHAR(63),
    details          TEXT,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id, profile_id),
    UNIQUE (tables_prefix)
);

INSERT INTO vector_store_76a3008adb7cb8f988ba492ad034e815.collections 
    (col_id, col_prefix, default_emb_profile)
VALUES 
    ('col-2a', md5('col-2a'), 'openai-text-embedding-ada-002');

INSERT INTO vector_store_76a3008adb7cb8f988ba492ad034e815.collection_emb_profiles 
    (col_id, profile_id, tables_prefix, status)
VALUES 
    ('col-2a', 'openai-text-embedding-ada-002', md5('col-2a'), 'READY');

-- Create tables for collection col-2a (iso-test-2)
CREATE TABLE vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc (
    doc_id           TEXT NOT NULL,
    status           TEXT,
    error_message    TEXT,
    attr_ids         BIGINT[],
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (doc_id)
);

CREATE TABLE vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_emb (
    emb_id           TEXT NOT NULL,
    doc_id           TEXT NOT NULL,
    content          TEXT,
    embedding        vector(1536),
    status           TEXT,
    response_code    INT DEFAULT 0,
    error_message    TEXT,
    attr_ids         BIGINT[],
    attr_ids2        BIGINT[],
    modified_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (emb_id),
    FOREIGN KEY (doc_id) REFERENCES vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc(doc_id) ON DELETE CASCADE
);

CREATE TABLE vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_attr (
    attr_id    BIGSERIAL NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    value      TEXT NOT NULL,
    value_hash VARCHAR(40) NOT NULL,
    PRIMARY KEY (attr_id),
    UNIQUE (name, type, value_hash)
);

CREATE TABLE vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc_meta (
    doc_id         TEXT NOT NULL,
    metadata_key   TEXT NOT NULL,
    metadata_value TEXT,
    modified_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (doc_id, metadata_key),
    FOREIGN KEY (doc_id) REFERENCES vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc(doc_id) ON DELETE CASCADE
);

CREATE TABLE vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_emb_meta (
    emb_id         TEXT NOT NULL,
    metadata_key   TEXT NOT NULL,
    metadata_value TEXT,
    modified_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (emb_id, metadata_key),
    FOREIGN KEY (emb_id) REFERENCES vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_emb(emb_id) ON DELETE CASCADE
);

-- Create processing tables for col-2a (added in v0.17.0)
CREATE TABLE vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc_processing (
    doc_id TEXT REFERENCES vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc(doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    heartbeat TIMESTAMP,
    record_timestamp TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER,
    attr_ids BIGINT[],
    doc_metadata JSONB,
    file BYTEA,
    PRIMARY KEY (doc_id)
);

CREATE INDEX idx_t_98ffca137d7d22e2a7af739a3f1a04b2_doc_processing__attrids 
    ON vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc_processing 
    USING GIN (attr_ids) WHERE attr_ids IS NOT NULL;

CREATE TABLE vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_emb_processing (
    emb_id TEXT NOT NULL,
    doc_id TEXT REFERENCES vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc_processing(doc_id) ON DELETE CASCADE ON UPDATE CASCADE,
    content TEXT,
    embedding vector(1536),
    status TEXT,
    attr_ids BIGINT[],
    metadata JSONB,
    response_code INT DEFAULT 0,
    error_message TEXT,
    retry_count INTEGER,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    record_timestamp TIMESTAMP,
    token_count INTEGER,
    PRIMARY KEY (emb_id)
);

-- Insert test data for col-2a with more comprehensive attributes
INSERT INTO vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_attr 
    (attr_id, name, type, value, value_hash)
VALUES 
    (1, 'Priority', 'string', 'High', md5('High')),
    (2, 'Status', 'string', 'Active', md5('Active')),
    (3, 'Tags', 'string', 'important', md5('important')),
    (4, 'Region', 'string', 'US-East', md5('US-East'));

INSERT INTO vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc 
    (doc_id, status, attr_ids)
VALUES 
    ('doc-2a-1', 'COMPLETED', ARRAY[1, 2]),
    ('doc-2a-2', 'COMPLETED', ARRAY[3, 4]);

INSERT INTO vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_doc_meta 
    (doc_id, metadata_key, metadata_value)
VALUES 
    ('doc-2a-1', 'Priority', 'High'),
    ('doc-2a-1', 'Status', 'Active'),
    ('doc-2a-2', 'Tags', 'important'),
    ('doc-2a-2', 'Region', 'US-East');

INSERT INTO vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_emb 
    (emb_id, doc_id, content, status, attr_ids, attr_ids2)
VALUES 
    ('emb-2a-1', 'doc-2a-1', 'Important content', 'COMPLETED', ARRAY[1], ARRAY[1, 2]),
    ('emb-2a-2', 'doc-2a-2', 'Regional content', 'COMPLETED', ARRAY[4], ARRAY[3, 4]);

INSERT INTO vector_store_76a3008adb7cb8f988ba492ad034e815.t_98ffca137d7d22e2a7af739a3f1a04b2_emb_meta 
    (emb_id, metadata_key, metadata_value)
VALUES 
    ('emb-2a-1', 'Priority', 'High'),
    ('emb-2a-2', 'Region', 'US-East');
