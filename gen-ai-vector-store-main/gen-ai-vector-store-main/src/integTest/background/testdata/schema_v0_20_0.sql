-- Copyright (c) 2025 Pegasystems Inc.
-- All rights reserved.
--
-- Test data for schema version v0.20.0
-- This represents the database state BEFORE migration to v0.21.0
-- IMPORTANT: Does NOT include the pdc_endpoint_url column (will be added by migration v0.21.0)

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create vector_store schema
CREATE SCHEMA IF NOT EXISTS vector_store;

-- Create base tables
-- Note: isolations table does NOT have pdc_endpoint_url column (v0.20.0 state, before v0.21.0)
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

-- Set schema version to v0.20.0 (before migration to v0.21.0)
INSERT INTO vector_store.configuration (key, value) VALUES ('schema_version', 'v0.20.0');
INSERT INTO vector_store.configuration (key, value) VALUES ('schema_version_prev', 'v0.19.0');

-- Create test isolations with various data
INSERT INTO vector_store.isolations (iso_id, iso_prefix, max_storage_size, created_at, modified_at)
VALUES 
    ('iso-empty', md5('iso-empty'), '1GB', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('iso-test-1', md5('iso-test-1'), '2GB', CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 hour'),
    ('iso-test-2', md5('iso-test-2'), '5GB', CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP),
    ('iso-test-3', md5('iso-test-3'), NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- Create schemas for test isolations
CREATE SCHEMA IF NOT EXISTS vector_store_e5af55acf1ba36f60b1e55a5d57f4b7d; -- iso-empty
CREATE SCHEMA IF NOT EXISTS vector_store_6f909e9b46455b62a7337a75311a25eb; -- iso-test-1
CREATE SCHEMA IF NOT EXISTS vector_store_76a3008adb7cb8f988ba492ad034e815; -- iso-test-2
CREATE SCHEMA IF NOT EXISTS vector_store_c5a5bdd11f93ee7c40c46a71351b83f0; -- iso-test-3

-- Create minimal collections tables for each isolation (to verify migration doesn't break existing structures)
CREATE TABLE IF NOT EXISTS vector_store_e5af55acf1ba36f60b1e55a5d57f4b7d.collections (
    col_id           TEXT NOT NULL,
    col_prefix       VARCHAR(40) NOT NULL UNIQUE,
    default_emb_profile VARCHAR(127) NOT NULL,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id)
);

CREATE TABLE IF NOT EXISTS vector_store_6f909e9b46455b62a7337a75311a25eb.collections (
    col_id           TEXT NOT NULL,
    col_prefix       VARCHAR(40) NOT NULL UNIQUE,
    default_emb_profile VARCHAR(127) NOT NULL,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id)
);

CREATE TABLE IF NOT EXISTS vector_store_76a3008adb7cb8f988ba492ad034e815.collections (
    col_id           TEXT NOT NULL,
    col_prefix       VARCHAR(40) NOT NULL UNIQUE,
    default_emb_profile VARCHAR(127) NOT NULL,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id)
);

CREATE TABLE IF NOT EXISTS vector_store_c5a5bdd11f93ee7c40c46a71351b83f0.collections (
    col_id           TEXT NOT NULL,
    col_prefix       VARCHAR(40) NOT NULL UNIQUE,
    default_emb_profile VARCHAR(127) NOT NULL,
    record_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (col_id)
);

-- Insert a collection for iso-test-1 to verify migration doesn't affect collection data
INSERT INTO vector_store_6f909e9b46455b62a7337a75311a25eb.collections 
    (col_id, col_prefix, default_emb_profile)
VALUES 
    ('col-1a', md5('col-1a'), 'openai-text-embedding-ada-002');
