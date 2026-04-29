CREATE OR REPLACE FUNCTION vector_store.calculate_document_status(embTable TEXT, docID TEXT) RETURNS TEXT AS
$$
DECLARE
    result jsonb;
BEGIN
    EXECUTE format('
        SELECT jsonb_object_agg(status, count) AS status_counts
        FROM (
             SELECT status, count(status) AS count
             FROM %1$s
             WHERE doc_id = %2$L
             GROUP BY status
        ) AS status_counts;
    ', embTable, docID) INTO result;

    -- return In Progress if at least one status is in progress
    IF result ? 'IN_PROGRESS' THEN RETURN 'IN_PROGRESS'; END IF;

    -- else return Error if at least one status is error
    IF result ? 'ERROR' THEN RETURN 'ERROR'; END IF;

    -- else return Completed if at least one status is completed
    IF result ? 'COMPLETED' THEN RETURN 'COMPLETED'; END IF;

    -- else return Unknown
    RETURN 'UNKNOWN';
END
$$ LANGUAGE plpgsql;
