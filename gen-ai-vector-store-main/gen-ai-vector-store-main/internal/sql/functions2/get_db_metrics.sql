CREATE OR REPLACE FUNCTION vector_store.get_db_metrics()
    RETURNS TABLE
            (
                iso_id TEXT,
                col_id TEXT,
                profile_id TEXT,
                schema_prefix TEXT,
                tables_prefix TEXT,
                doc_count BIGINT,
                emb_count BIGINT,
                attr_count BIGINT,
                emb_queue_count BIGINT
            )
AS
$$
DECLARE
    isolation_rec RECORD;
    config_rec RECORD;
    table_collections TEXT;
    table_doc TEXT;
    table_emb TEXT;
    table_attr TEXT;
    schema_name TEXT;
BEGIN
    -- Single query to get all isolation data
    FOR isolation_rec IN SELECT I.iso_id, I.iso_prefix FROM vector_store.isolations I
        LOOP
            schema_name := format('vector_store_%s', isolation_rec.iso_prefix);
            table_collections := format('%s.collections', schema_name);

            -- Check if collections table exists before querying
            IF vector_store.table_exists(table_collections) THEN
                -- Single query to get all collection/profile configurations
                FOR config_rec IN EXECUTE format('
                SELECT c.col_id, cep.profile_id, cep.tables_prefix
                FROM %s c
                JOIN %s.collection_emb_profiles cep ON c.col_id = cep.col_id
            ', table_collections, schema_name)
                    LOOP
                        -- Initialize output record
                        iso_id := isolation_rec.iso_id;
                        col_id := config_rec.col_id;
                        profile_id := config_rec.profile_id;
                        schema_prefix := isolation_rec.iso_prefix;
                        tables_prefix := config_rec.tables_prefix;
                        doc_count := 0;
                        emb_count := 0;
                        attr_count := 0;
                        emb_queue_count := 0;

                        -- Build table names
                        table_doc := format('%s.t_%s_doc', schema_name, config_rec.tables_prefix);
                        table_emb := format('%s.t_%s_emb', schema_name, config_rec.tables_prefix);
                        table_attr := format('%s.t_%s_attr', schema_name, config_rec.tables_prefix);

                        -- Count rows in each table if it exists
                        BEGIN
                            IF vector_store.table_exists(table_doc) THEN
                                EXECUTE format('SELECT count(*) FROM %s', table_doc) INTO doc_count;
                            END IF;
                            IF vector_store.table_exists(table_emb) THEN
                                EXECUTE format('SELECT count(*) FROM %s', table_emb) INTO emb_count;
                            END IF;
                            IF vector_store.table_exists(table_attr) THEN
                                EXECUTE format('SELECT count(*) FROM %s', table_attr) INTO attr_count;
                            END IF;
                            -- Count embedding queue entries for this collection
                            EXECUTE format('SELECT count(*) FROM vector_store.embedding_queue' ||
                                           ' WHERE (content->''iso_id'')::jsonb ? %L AND (content->''col_id'')::jsonb ? %L',
                                           isolation_rec.iso_id, config_rec.col_id) INTO emb_queue_count;
                        EXCEPTION
                            WHEN OTHERS THEN
                                -- Log error but continue processing
                                RAISE NOTICE 'Error counting rows for tables_prefix %: %', config_rec.tables_prefix, SQLERRM;
                                doc_count := -1;
                                emb_count := -1;
                                attr_count := -1;
                                emb_queue_count := -1;
                        END;

                        RETURN NEXT;
                    END LOOP;
            END IF;
        END LOOP;
    RETURN;
END
$$ LANGUAGE plpgsql;
