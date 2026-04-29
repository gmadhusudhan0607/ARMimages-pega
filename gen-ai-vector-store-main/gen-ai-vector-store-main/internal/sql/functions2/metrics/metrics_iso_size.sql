CREATE OR REPLACE FUNCTION vector_store.metrics_iso_size(isolation_id TEXT DEFAULT NULL)
    RETURNS TABLE
            (
                iso_id TEXT,
                iso_prefix TEXT,
                profile_id TEXT,
                tables_prefix TEXT,
                disk_usage_bytes   bigint
            )
AS
$$
DECLARE
    _table_collections TEXT;
    _table_collection_emb_profiles TEXT;
    _iso_id TEXT;
    _iso_prefix TEXT;
    _col_id TEXT;
    _profile_id TEXT;
    _tables_prefix TEXT;
    _schema_name TEXT;
    _col_md5 TEXT;
    _table_doc TEXT;
    _table_emb TEXT;
    _table_doc_processing TEXT;
    _table_emb_processing TEXT;
    _table_attr TEXT;
    _total_size bigint;
    col_profiles RECORD;
BEGIN
    -- Only process specified isolation if isolation_id is provided
    FOR _iso_id, _iso_prefix IN SELECT I.iso_id, I.iso_prefix from vector_store.isolations I
                                WHERE (isolation_id IS NULL OR I.iso_id = isolation_id)
        LOOP
            _schema_name := format('vector_store_%s', _iso_prefix);
            _table_collections := format('%s.collections', _schema_name);
            _table_collection_emb_profiles := format('%s.collection_emb_profiles', _schema_name);

            IF vector_store.table_exists(_table_collections) AND
               vector_store.table_exists(_table_collection_emb_profiles) THEN
                
                -- Loop through each collection and profile combination
                FOR col_profiles IN 
                    EXECUTE format('SELECT c.col_id, p.profile_id FROM %s c JOIN %s p ON c.col_id = p.col_id', 
                                   _table_collections, _table_collection_emb_profiles)
                LOOP
                    _col_id := col_profiles.col_id;
                    _profile_id := col_profiles.profile_id;
                    _col_md5 := md5(_col_id);
                    _tables_prefix := format('t_%s', _col_md5);
                    _total_size := 0;

                    -- Calculate size of _doc table
                    _table_doc := format('%s.t_%s_doc', _schema_name, _col_md5);
                    IF vector_store.table_exists(_table_doc) THEN
                        _total_size := _total_size + pg_total_relation_size(_table_doc);
                    END IF;

                    -- Calculate size of _emb table
                    _table_emb := format('%s.t_%s_emb', _schema_name, _col_md5);
                    IF vector_store.table_exists(_table_emb) THEN
                        _total_size := _total_size + pg_total_relation_size(_table_emb);
                    END IF;

                    -- Calculate size of _attr table
                    _table_attr := format('%s.t_%s_attr', _schema_name, _col_md5);
                    IF vector_store.table_exists(_table_attr) THEN
                        _total_size := _total_size + pg_total_relation_size(_table_attr);
                    END IF;

                    -- Calculate size of _doc_processing table
                    _table_doc_processing := format('%s.t_%s_doc_processing', _schema_name, _col_md5);
                    IF vector_store.table_exists(_table_doc_processing) THEN
                        _total_size := _total_size + pg_total_relation_size(_table_doc_processing);
                    END IF;

                    -- Calculate size of _emb_processing table
                    _table_emb_processing := format('%s.t_%s_emb_processing', _schema_name, _col_md5);
                    IF vector_store.table_exists(_table_emb_processing) THEN
                        _total_size := _total_size + pg_total_relation_size(_table_emb_processing);
                    END IF;

                    -- Return the aggregated row
                    RETURN QUERY SELECT _iso_id, _iso_prefix, _profile_id, _tables_prefix, _total_size;
                END LOOP;
                
            END IF;
        END LOOP;
    RETURN;
END
$$ LANGUAGE plpgsql;