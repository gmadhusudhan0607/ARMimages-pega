CREATE OR REPLACE FUNCTION vector_store.copy_schema(src_schema text, dst_schema text) RETURNS void AS
$$
DECLARE
    table_record RECORD;
BEGIN
    EXECUTE format('CREATE SCHEMA %I', dst_schema);
    FOR table_record IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = src_schema
        LOOP
            EXECUTE format('CREATE TABLE %2$s.%3$s (LIKE %1$s.%3$s INCLUDING ALL)', src_schema,  dst_schema, table_record.tablename);
            EXECUTE format('INSERT INTO  %2$s.%3$s SELECT * FROM %1$s.%3$s', src_schema, dst_schema, table_record.tablename);
        END LOOP;
END;
$$ LANGUAGE plpgsql;
