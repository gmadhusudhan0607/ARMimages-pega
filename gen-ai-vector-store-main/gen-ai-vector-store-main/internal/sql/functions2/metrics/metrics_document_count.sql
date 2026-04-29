CREATE OR REPLACE FUNCTION vector_store.metrics_document_count(iso_id text)
    RETURNS TABLE
            (
                col_id   text,
                doc_count bigint
            )
AS
$func$
DECLARE
    iso_md5 text;
    table_v2_collections text;
    table_v2_doc text;
    count         bigint;
    c_id          text;
    col_ids text[];
BEGIN
    iso_md5 = md5(iso_id);
    table_v2_collections =  FORMAT('vector_store_%s.collections',iso_md5);
    if not vector_store.table_exists(table_v2_collections) then
        return;
    end if;

    EXECUTE FORMAT('SELECT array_agg(col_id) FROM %s ', table_v2_collections) into col_ids;
    FOR c_id IN SELECT * FROM unnest(col_ids) LOOP
            table_v2_doc = FORMAT('vector_store_%s.t_%s_doc', iso_md5, md5(c_id));
            IF vector_store.table_exists(table_v2_doc) THEN
                EXECUTE format('SELECT COUNT(*) FROM %s', table_v2_doc) INTO count;
                RETURN QUERY SELECT c_id, count;
            END IF;
        END LOOP;
    RETURN;
END
$func$ LANGUAGE plpgsql;

SELECT * FROM vector_store.metrics_document_count('iso-awzrcastnh');