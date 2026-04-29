CREATE OR REPLACE FUNCTION vector_store.lookup_resources_metadata()
    RETURNS TABLE
            (
                iso_id TEXT,
                schema_name TEXT,
                col_id TEXT,
                tables_prefix TEXT
            )
    AS
$$
DECLARE
    col_ids   TEXT[];
    table_collections TEXT;
BEGIN
    FOR iso_id IN SELECT I.iso_id from vector_store.isolations I
    LOOP
        schema_name := format('vector_store_%s', md5(iso_id));
        table_collections = format('%s.collections', schema_name);
        IF vector_store.table_exists(table_collections) THEN
            EXECUTE format('SELECT array_agg(col_id) FROM %s ', table_collections) INTO col_ids;
            IF col_ids is null THEN
                ---return empty row so that we will have a row for each iso_id even if there are no collections
                col_id = null;
                tables_prefix = null;
                RETURN NEXT;
            ELSE
                FOR col_id IN SELECT * FROM unnest(col_ids)
                    LOOP
                        tables_prefix = format('t_%s', md5(col_id));
                        RETURN NEXT;
                    END LOOP;
            END IF;
        END IF;
    END LOOP;
    RETURN;
END
$$ LANGUAGE plpgsql;
