/*
 * schema_info
 *
 * Returns information about the schema, collections and profiles in the vector store.
 *
 * Parameters:
 *  isolation_id: TEXT - The isolation id to filter by. If not provided, all isolations are returned.
 *  collection_id: TEXT - The collection id to filter by. If not provided, all collections are returned.
 *
 * Returns:
 *  iso_id: TEXT - The isolation id.
 *  col_id: TEXT - The collection id.
 *  profile_id: TEXT - The profile id.
 *  schema_name: TEXT - The schema name.
 *  tables_prefix: TEXT - The tables prefix.
 *  profile_status: TEXT - The profile status.
 *  is_default_profile: BOOL - Whether the profile is the default profile for the collection.
 *
*/
CREATE OR REPLACE FUNCTION vector_store.schema_info(isolation_id TEXT DEFAULT NULL, collection_id TEXT DEFAULT NULL)
    RETURNS TABLE
            (
                iso_id TEXT,
                col_id TEXT,
                profile_id TEXT,
                schema_name TEXT,
                tables_prefix TEXT,
                profile_status TEXT,
                is_default_profile BOOL
            )
AS
$$
DECLARE
    table_collections TEXT;
    table_collection_emb_profiles TEXT;
    _col_id TEXT;
    _col_count INT;
    _profile_id TEXT;
    _tables_prefix TEXT;
    _profile_status TEXT;
    _is_default TEXT;
    _query TEXT;
    _iso_prefix TEXT;
BEGIN
    -- Only process specified isolation if isolation_id is provided
    FOR iso_id, _iso_prefix IN SELECT I.iso_id, I.iso_prefix from vector_store.isolations I
                                WHERE (isolation_id IS NULL OR I.iso_id = isolation_id)
        LOOP
            col_id = null;
            profile_id = null;
            tables_prefix = null;

            schema_name := format('vector_store_%s', _iso_prefix);
            table_collections = format('%s.collections', schema_name);
            table_collection_emb_profiles = format('%s.collection_emb_profiles', schema_name);

            IF vector_store.table_exists(table_collections) AND
               vector_store.table_exists(table_collection_emb_profiles) THEN

                -- Count collections, filtered by collection_id if provided
                EXECUTE format('SELECT count(*) FROM %1$s WHERE (%2$L IS NULL OR col_id = %2$L)',
                               table_collections, collection_id) INTO _col_count;

                IF _col_count = 0 THEN
                    -- return empty row with iso_id for isolations without collections
                    -- if not searching for specific collection
                    RETURN NEXT;
                ELSE
                    -- Filter by collection_id if provided in main query
                    _query := format('
                    SELECT C.col_id, CEP.profile_id, CEP.tables_prefix, CEP.status,
                          (C.default_emb_profile = CEP.profile_id) AS is_default
                    FROM %1$s C JOIN %2$s CEP ON C.col_id = CEP.col_id
                    WHERE (%3$L IS NULL OR C.col_id = %3$L)',
                                     table_collections, table_collection_emb_profiles, collection_id);

                    FOR _col_id, _profile_id, _tables_prefix, _profile_status, _is_default IN
                        EXECUTE _query
                        LOOP
                            tables_prefix = format('t_%s', _tables_prefix);
                            col_id = _col_id;
                            profile_id = _profile_id;
                            profile_status = _profile_status;
                            is_default_profile = _is_default;
                            RETURN NEXT;
                        END LOOP;
                END IF;
            ELSE
                RETURN NEXT;
            END IF;
        END LOOP;
    RETURN;
END
$$ LANGUAGE plpgsql;
