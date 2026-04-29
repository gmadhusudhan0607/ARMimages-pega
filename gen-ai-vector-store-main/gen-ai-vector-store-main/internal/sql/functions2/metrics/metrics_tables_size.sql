CREATE OR REPLACE FUNCTION vector_store.metrics_table_size(iso_id text)
    returns
        table
        (
            col_id       text,
            table_name   text,
            table_suffix text,
            table_size   bigint
        )
AS
$func$
DECLARE
    iso_md5              text;
    col_md5              text;
    table_v2_collections text;
    table_v2_doc         text;
    table_v2_emb         text;
    table_v2_attr        text;
    v1_iso_size          bigint;
    v1_col_size          bigint;
    col_ids              text[];
    c_id                 text;
BEGIN
    iso_md5 = md5(iso_id);

    -- Size of records in isolations table
    EXECUTE format('SELECT coalesce(SUM(pg_column_size(%1$s.*)), 0) as iso_size FROM %1$s WHERE iso_id = %2$L',
                   'vector_store.isolations', iso_id) INTO v1_iso_size;
    RETURN QUERY SELECT '', 'vector_store.isolations', 'v2_isolations', v1_iso_size;

    table_v2_collections = FORMAT('vector_store_%s.collections', iso_md5);
    IF NOT vector_store.table_exists(table_v2_collections) THEN
        RETURN ;
    END IF;

    EXECUTE FORMAT('SELECT array_agg(col_id) FROM %s ', table_v2_collections) into col_ids;
    FOR c_id IN SELECT * FROM unnest(col_ids)
        LOOP
            col_md5 = md5(c_id);
            EXECUTE format('SELECT coalesce(SUM(pg_column_size(%1$s.*)), 0) as col_size FROM %1$s WHERE col_id = %2$L'
                , table_v2_collections, c_id) INTO v1_col_size;
            RETURN QUERY SELECT c_id, 'vector_store.collections', 'v2_collections', v1_col_size;

            table_v2_doc = FORMAT('vector_store_%s.t_%s_doc', iso_md5, col_md5);
            IF vector_store.table_exists(table_v2_doc) THEN
                RETURN QUERY SELECT c_id, table_v2_doc, 'v2_doc', pg_total_relation_size(table_v2_doc);
            END IF;

            table_v2_emb = FORMAT('vector_store_%s.t_%s_emb', iso_md5, col_md5);
            IF vector_store.table_exists(table_v2_emb) THEN
                RETURN QUERY SELECT c_id, table_v2_emb, 'v2_emb', pg_total_relation_size(table_v2_emb);
            END IF;

            table_v2_attr = FORMAT('vector_store_%s.t_%s_attr', iso_md5, col_md5);
            IF vector_store.table_exists(table_v2_attr) THEN
                RETURN QUERY SELECT c_id, table_v2_attr, 'v2_attr', pg_total_relation_size(table_v2_attr);
            END IF;
        END LOOP;
    RETURN;
END
$func$ LANGUAGE plpgsql;
