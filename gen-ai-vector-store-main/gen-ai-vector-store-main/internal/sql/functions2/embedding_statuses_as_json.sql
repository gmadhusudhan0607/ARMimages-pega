CREATE OR REPLACE FUNCTION vector_store.embedding_statuses_as_json(embTable TEXT, docID TEXT) RETURNS json AS
$$
DECLARE
    result   json;
BEGIN
    EXECUTE format('
        SELECT coalesce(json_agg(statuses)::TEXT, ''{}'') as statuses
        FROM (
               SELECT json_build_object(
                ''status'',  status,
                ''code'',    response_code,
                ''message'', error_message,
                ''count'',   count(*)) as statuses
               FROM %1$s
               WHERE doc_id = %2$L
               GROUP BY status, response_code, error_message
        ) AS T0
    ', embTable, docID) INTO result;
    RETURN result;
END
$$ LANGUAGE plpgsql;
