CREATE OR REPLACE FUNCTION vector_store.drop_all_triggers_on_collection(iso_id TEXT, col_id TEXT) RETURNS VOID AS
$$
DECLARE
    schema_name  TEXT;
    tables_array text[] := '{}';
BEGIN

    -- V1 tables
    tables_array := '{}';
    schema_name := current_schema;
    tables_array := array_append(tables_array, REPLACE('i_' || iso_id || '_' || col_id || '_attributes', '-', '_'));
    tables_array := array_append(tables_array, REPLACE('i_' || iso_id || '_' || col_id || '_embeddings', '-', '_'));
    tables_array := array_append(tables_array, REPLACE('i_' || iso_id || '_' || col_id || '_documents', '-', '_'));
    FOR i IN 1..array_length(tables_array, 1)
        LOOP
            PERFORM vector_store.drop_all_triggers_on_table(schema_name, tables_array[i]);
        END LOOP;

    -- V2 tables
    tables_array := '{}';
    schema_name := format('vector_store_%s', md5(iso_id));
    tables_array := array_append(tables_array, format('t_%s_attr', md5(col_id)));
    tables_array := array_append(tables_array, format('t_%s_emb', md5(col_id)));
    tables_array := array_append(tables_array, format('t_%s_doc', md5(col_id)));
    FOR i IN 1..array_length(tables_array, 1)
        LOOP
            PERFORM vector_store.drop_all_triggers_on_table(schema_name, tables_array[i]);
        END LOOP;

END
$$ LANGUAGE plpgsql;