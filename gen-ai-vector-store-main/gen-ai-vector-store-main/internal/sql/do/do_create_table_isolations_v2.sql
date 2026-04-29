CREATE TABLE IF NOT EXISTS vector_store.isolations
(
    iso_id           text        NOT NULL PRIMARY KEY,
    iso_prefix       VARCHAR(40) NOT NULL UNIQUE,
    max_storage_size text,
    pdc_endpoint_url text,
    created_at       timestamp,
    modified_at      timestamp,
    record_timestamp TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);