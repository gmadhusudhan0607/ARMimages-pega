CREATE OR REPLACE FUNCTION vector_store.embeddings_queue_get_with_exception() RETURNS vector_store.embedding_queue AS
$BODY$
declare
    result vector_store.embedding_queue;
BEGIN
    WITH deleted_rows as (
        DELETE
            FROM vector_store.embedding_queue EQ
                WHERE EQ.id =
                      (SELECT EQ_inner.id
                       FROM vector_store.embedding_queue EQ_inner
                       WHERE EQ_inner.postpone_until <= current_timestamp
                       ORDER BY EQ_inner.created_at ASC
                           FOR UPDATE SKIP LOCKED
                       LIMIT 1)
                RETURNING EQ.id, EQ.created_at, EQ.postpone_until, EQ.content)
    SELECT into result id, created_at, postpone_until, content
    from deleted_rows
    where deleted_rows.id is not null;
    IF not FOUND THEN
        RAISE SQLSTATE '02000' -- 5 ASCII chars (	02000 = no_data) . See https://www.postgresql.org/docs/current/errcodes-appendix.html
            USING MESSAGE = 'Queue is empty';
    END IF;
    return result;
END;
$BODY$ LANGUAGE plpgsql;