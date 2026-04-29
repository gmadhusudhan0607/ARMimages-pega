CREATE TABLE IF NOT EXISTS vector_store.embedding_queue
(
    id              UUID PRIMARY KEY,
    created_at      TIMESTAMP,
    postpone_until  TIMESTAMP,
    content         JSON
);