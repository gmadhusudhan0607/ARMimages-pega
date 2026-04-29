CREATE OR REPLACE FUNCTION
    vector_store.get_collection_document_count(isolation_id TEXT, collection_id TEXT DEFAULT NULL)
    RETURNS TABLE
            (
                col_id TEXT,
                default_embedding_profile TEXT,
                documents_total INTEGER
            )
AS
$$
DECLARE
    table_collections TEXT;
    table_collection_emb_profiles TEXT;
    schema_name TEXT;
    collection_query TEXT;
    collection_record RECORD;
    doc_table_name TEXT;
    docs_count INTEGER;
BEGIN
    schema_name = format('vector_store_%s', md5(isolation_id));
    table_collections = format('%s.collections', schema_name);
    table_collection_emb_profiles = format('%s.collection_emb_profiles', schema_name);

       
        collection_query := format('
            SELECT c.col_id, c.default_emb_profile, ce.tables_prefix 
            FROM %1$s c
            INNER JOIN %2$s ce ON c.col_id = ce.col_id AND c.default_emb_profile = ce.profile_id
            WHERE %3$L IS NULL OR c.col_id = %3$L
            ',
            table_collections,
            table_collection_emb_profiles,
            collection_id
        );
        
        FOR collection_record IN EXECUTE collection_query LOOP
            doc_table_name := format('%s.t_%s_doc', schema_name, collection_record.tables_prefix);
    
            EXECUTE format('SELECT COUNT(*) FROM %s', doc_table_name) INTO docs_count;

            col_id := collection_record.col_id;
            default_embedding_profile := collection_record.default_emb_profile;
            documents_total := docs_count;
            RETURN NEXT;
        END LOOP;

    RETURN;
END
$$
LANGUAGE plpgsql;
