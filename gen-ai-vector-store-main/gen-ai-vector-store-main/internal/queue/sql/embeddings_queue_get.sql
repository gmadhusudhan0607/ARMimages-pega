CREATE OR REPLACE FUNCTION  vector_store.embeddings_queue_get() RETURNS SETOF vector_store.embedding_queue AS
$func$
BEGIN
    RETURN QUERY
        SELECT * from vector_store.embeddings_queue_get_with_exception();
EXCEPTION
    WHEN SQLSTATE '02000' THEN
        NULL; -- ignore the error
END
$func$ LANGUAGE plpgsql;
