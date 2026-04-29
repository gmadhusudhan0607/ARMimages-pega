CREATE OR REPLACE FUNCTION vector_store.metrics_last_modified_time(iso_id text)
    RETURNS TABLE
            (
                col_id        text,
                modified_time timestamp
            )
AS
$func$
DECLARE
    iso_md5              text      := md5(iso_id);
    table_v2_collections text      := FORMAT('vector_store_%s.collections', iso_md5);
    c_id                 text;
    table_v2_doc         text;
    col_ids              text[];
    default_time         timestamp := TO_TIMESTAMP('0001-01-01 00:00:00 +0000', 'YYYY-MM-DD HH24:MI:SS TZH:TZM');
    latest_modified      timestamp;
BEGIN
    IF NOT vector_store.table_exists(table_v2_collections) THEN
        RETURN;
    END IF;

    EXECUTE FORMAT('SELECT array_agg(col_id) FROM %s', table_v2_collections) INTO col_ids;
    FOR c_id IN SELECT * FROM unnest(col_ids)
        LOOP
            table_v2_doc = FORMAT('vector_store_%s.t_%s_doc', iso_md5, md5(c_id));
            IF vector_store.table_exists(table_v2_doc) THEN
                EXECUTE FORMAT('SELECT COALESCE(MAX(modified_at), %L) AS modified_time FROM %s', default_time,
                               table_v2_doc) INTO latest_modified;
                RETURN QUERY SELECT c_id, latest_modified;
            END IF;
        END LOOP;
    RETURN;
END
$func$ LANGUAGE plpgsql;