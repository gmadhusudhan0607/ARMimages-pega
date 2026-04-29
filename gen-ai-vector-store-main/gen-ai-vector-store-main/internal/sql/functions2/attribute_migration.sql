-- Copyright (c) 2025 Pegasystems Inc.
-- All rights reserved.

-- Convert attr_ids to AttributesV2 JSONB format
CREATE OR REPLACE FUNCTION vector_store.migration_19_convert_attr_ids_to_jsonb_attributes(
    attr_table TEXT, 
    attr_ids bigint[]
) RETURNS JSONB AS
$$
DECLARE
    attributes JSONB;
BEGIN
    -- Handle NULL or empty array case
    IF attr_ids IS NULL OR array_length(attr_ids, 1) IS NULL THEN
        RETURN '{}'::JSONB;
    END IF;
    
    -- Convert attribute IDs to AttributesV2 format
    EXECUTE FORMAT('
        WITH attr_values AS (
            SELECT 
                ATTR.name,
                array_agg(DISTINCT ATTR.value ORDER BY ATTR.value) as values
            FROM %1$s ATTR
            WHERE ATTR.attr_id = ANY(%2$L)
            GROUP BY ATTR.name
        )
        SELECT COALESCE(
            jsonb_object_agg(
                name,
                jsonb_build_object(
                    ''kind'', ''static'',
                    ''values'', values
                )
            ),
            ''{}''::JSONB
        )
        FROM attr_values
        ', attr_table, attr_ids) INTO attributes;
    
    RETURN COALESCE(attributes, '{}'::JSONB);
END
$$ LANGUAGE plpgsql COST 100;

-- Batch migration function for a specific table
CREATE OR REPLACE FUNCTION vector_store.migration_19_migrate_attributes_batch(
    table_name TEXT,
    source_column TEXT,
    target_column TEXT,
    attr_table TEXT,
    batch_size INT
) RETURNS INT AS
$$
DECLARE
    updated_count INT := 0;
    column_exists BOOLEAN := FALSE;
    target_schema TEXT;
    target_table TEXT;
BEGIN
    -- Parse schema and table name
    target_schema := split_part(table_name, '.', 1);
    target_table := split_part(table_name, '.', 2);
    
    -- Check if source column exists in the table
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns c
        WHERE c.table_schema = target_schema
        AND c.table_name = target_table
        AND c.column_name = source_column
    ) INTO column_exists;
    
    -- If source column doesn't exist, return 0 (no migration needed)
    IF NOT column_exists THEN
        RETURN 0;
    END IF;
    
    -- Process only one batch per function call
    EXECUTE FORMAT('
        WITH batch_records AS (
            SELECT ctid, %2$s
            FROM %1$s 
            WHERE %3$s IS NULL AND %2$s IS NOT NULL
            LIMIT %4$s
        )
        UPDATE %1$s 
        SET %3$s = vector_store.migration_19_convert_attr_ids_to_jsonb_attributes(%5$L, batch_records.%2$s)
        FROM batch_records
        WHERE %1$s.ctid = batch_records.ctid
        ', table_name, source_column, target_column, batch_size, attr_table);
    
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    
    RETURN updated_count;
END
$$ LANGUAGE plpgsql;

-- Get count of processed records (where target column is NOT NULL AND source column is NOT NULL)
CREATE OR REPLACE FUNCTION vector_store.migration_19_get_processed_count(
    table_name TEXT,
    target_column TEXT,
    source_column TEXT DEFAULT NULL
) RETURNS INT AS
$$
DECLARE
    processed_count INT;
    target_column_exists BOOLEAN := FALSE;
    source_column_exists BOOLEAN := TRUE;
    target_schema TEXT;
    target_table TEXT;
BEGIN
    -- Parse schema and table name
    target_schema := split_part(table_name, '.', 1);
    target_table := split_part(table_name, '.', 2);
    
    -- Check if target column exists in the table
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns c
        WHERE c.table_schema = target_schema
        AND c.table_name = target_table
        AND c.column_name = target_column
    ) INTO target_column_exists;

    -- If target column doesn't exist, raise an error
    IF NOT target_column_exists THEN
        RAISE EXCEPTION 'Target column "%" does not exist in table "%"', target_column, table_name;
    END IF;
    
    -- If source column is specified, check if it exists and count only migrated records
    IF source_column IS NOT NULL THEN
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns c
            WHERE c.table_schema = target_schema
            AND c.table_name = target_table
            AND c.column_name = source_column
        ) INTO source_column_exists;
        
        -- If source column doesn't exist, return 0 (no migration scope)
        IF NOT source_column_exists THEN
            RETURN 0;
        END IF;
        
        -- Count records where BOTH source and target are NOT NULL (successfully migrated)
        EXECUTE FORMAT('SELECT COUNT(*) FROM %1$s WHERE %2$s IS NOT NULL AND %3$s IS NOT NULL', 
            table_name, target_column, source_column) INTO processed_count;
    ELSE
        -- If no source column specified, count all records with target data (legacy behavior)
        EXECUTE FORMAT('SELECT COUNT(*) FROM %1$s WHERE %2$s IS NOT NULL', 
            table_name, target_column) INTO processed_count;
    END IF;
    
    RETURN processed_count;
END
$$ LANGUAGE plpgsql;

-- Get count of remaining records (where target column is NULL and source column has data)
CREATE OR REPLACE FUNCTION vector_store.migration_19_get_remaining_count(
    table_name TEXT,
    target_column TEXT,
    source_column TEXT DEFAULT NULL
) RETURNS INT AS
$$
DECLARE
    remaining_count INT;
    target_column_exists BOOLEAN := FALSE;
    source_column_exists BOOLEAN := TRUE;
    target_schema TEXT;
    target_table TEXT;
BEGIN
    -- Parse schema and table name
    target_schema := split_part(table_name, '.', 1);
    target_table := split_part(table_name, '.', 2);
    
    -- Check if target column exists in the table
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns c
        WHERE c.table_schema = target_schema
        AND c.table_name = target_table
        AND c.column_name = target_column
    ) INTO target_column_exists;

    -- If target column doesn't exist, raise an error
    IF NOT target_column_exists THEN
        RAISE EXCEPTION 'Target column "%" does not exist in table "%"', target_column, table_name;
    END IF;
    
    -- If source column is specified, check if it exists
    IF source_column IS NOT NULL THEN
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns c
            WHERE c.table_schema = target_schema
            AND c.table_name = target_table
            AND c.column_name = source_column
        ) INTO source_column_exists;
        
        -- If source column doesn't exist, return 0 (no migration needed)
        IF NOT source_column_exists THEN
            RETURN 0;
        END IF;
        
        -- Count records where target is NULL AND source is NOT NULL (matches migration logic)
        EXECUTE FORMAT('SELECT COUNT(*) FROM %1$s WHERE %2$s IS NULL AND %3$s IS NOT NULL', table_name, target_column, source_column) INTO remaining_count;
    ELSE
        -- If no source column specified, just count NULL target records
        EXECUTE FORMAT('SELECT COUNT(*) FROM %1$s WHERE %2$s IS NULL', table_name, target_column) INTO remaining_count;
    END IF;
    
    RETURN remaining_count;
END
$$ LANGUAGE plpgsql;

-- Get count of total migration scope (all records with source column data, regardless of target status)
CREATE OR REPLACE FUNCTION vector_store.migration_19_get_migration_scope_count(
    table_name TEXT,
    source_column TEXT
) RETURNS INT AS
$$
DECLARE
    scope_count INT;
    source_column_exists BOOLEAN := FALSE;
    target_schema TEXT;
    target_table TEXT;
BEGIN
    -- Parse schema and table name
    target_schema := split_part(table_name, '.', 1);
    target_table := split_part(table_name, '.', 2);
    
    -- Check if source column exists in the table
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns c
        WHERE c.table_schema = target_schema
        AND c.table_name = target_table
        AND c.column_name = source_column
    ) INTO source_column_exists;
    
    -- If source column doesn't exist, return 0 (no migration scope)
    IF NOT source_column_exists THEN
        RETURN 0;
    END IF;
    
    -- Count all records where source is NOT NULL (total migration scope)
    EXECUTE FORMAT('SELECT COUNT(*) FROM %1$s WHERE %2$s IS NOT NULL', table_name, source_column) INTO scope_count;
    
    RETURN scope_count;
END
$$ LANGUAGE plpgsql;

-- Bulk task discovery function to replace N+1 query pattern
CREATE OR REPLACE FUNCTION vector_store.migration_19_get_bulk_replication_tasks()
RETURNS TABLE (
    isolation_id TEXT,
    collection_id TEXT,
    profile_id TEXT,
    table_prefix TEXT,
    task_status TEXT
) AS
$$
DECLARE
    iso_record RECORD;
    col_record RECORD;
    profile_record RECORD;
    target_schema TEXT;
    schema_exists BOOLEAN;
    table_exists BOOLEAN;
BEGIN
    -- Iterate through isolations from vector_store.isolations
    FOR iso_record IN
        SELECT iso_id, iso_prefix
        FROM vector_store.isolations
    LOOP
        -- Use correct schema naming pattern: vector_store_<iso_prefix>
        target_schema := CONCAT('vector_store_', iso_record.iso_prefix);

        -- Check if schema exists
        SELECT EXISTS (
            SELECT 1 FROM information_schema.schemata s
            WHERE s.schema_name = target_schema
        ) INTO schema_exists;

        -- Skip this isolation if schema doesn't exist
        IF NOT schema_exists THEN
            CONTINUE;
        END IF;

        -- Check if collections table exists in this schema
        SELECT EXISTS (
            SELECT 1 FROM information_schema.tables t
            WHERE t.table_schema = target_schema
            AND t.table_name = 'collections'
        ) INTO table_exists;

        -- Skip this isolation if collections table doesn't exist
        IF NOT table_exists THEN
            CONTINUE;
        END IF;

        -- For each isolation, get collections from its schema (with exception handling)
        BEGIN
            FOR col_record IN
                EXECUTE FORMAT('SELECT col_id FROM %I.collections', target_schema)
            LOOP
                -- Check if collection_emb_profiles table exists
                SELECT EXISTS (
                    SELECT 1 FROM information_schema.tables t
                    WHERE t.table_schema = target_schema
                    AND t.table_name = 'collection_emb_profiles'
                ) INTO table_exists;

                -- Skip if collection_emb_profiles table doesn't exist
                IF NOT table_exists THEN
                    CONTINUE;
                END IF;

                -- For each collection, get profiles from collection_emb_profiles (with exception handling)
                BEGIN
                    FOR profile_record IN
                        EXECUTE FORMAT('SELECT profile_id, tables_prefix FROM %I.collection_emb_profiles WHERE col_id = %L',
                                      target_schema, col_record.col_id)
                    LOOP
                        -- Check task status from configuration
                        isolation_id := iso_record.iso_id;
                        collection_id := col_record.col_id;
                        profile_id := profile_record.profile_id;
                        table_prefix := profile_record.tables_prefix;

                        -- Determine task status
                        SELECT COALESCE(
                            (SELECT c.value
                             FROM vector_store.configuration c
                             WHERE c.key LIKE CONCAT('attribute_replication_v0.19.0_', iso_record.iso_id, '_', col_record.col_id, '_', profile_record.profile_id, '_%')
                             AND c.value = 'completed'
                             LIMIT 1),
                            'pending'
                        ) INTO task_status;

                        RETURN NEXT;
                    END LOOP;
                EXCEPTION
                    WHEN OTHERS THEN
                        -- Skip this collection if there's an error accessing collection_emb_profiles
                        CONTINUE;
                END;
            END LOOP;
        EXCEPTION
            WHEN OTHERS THEN
                -- Skip this isolation if there's an error accessing collections
                CONTINUE;
        END;
    END LOOP;
END
$$ LANGUAGE plpgsql;

-- Bulk statistics collection function  (helper)
CREATE OR REPLACE FUNCTION vector_store.migration_19_get_bulk_replication_statistics()
RETURNS TABLE (
    total_scope BIGINT,
    total_processed BIGINT,
    total_remaining BIGINT
) AS $$
BEGIN
    RETURN QUERY
    WITH pending_tasks AS (
        SELECT isolation_id, table_prefix
        FROM vector_store.migration_19_get_bulk_replication_tasks()
        WHERE task_status != 'completed'
    ),
    task_details AS (
        SELECT 
            t.isolation_id,
            t.table_prefix,
            i.iso_prefix,
            'vector_store_' || i.iso_prefix as schema_name
        FROM pending_tasks t
        JOIN vector_store.isolations i ON i.iso_id = t.isolation_id
    ),
    all_table_stats AS (
        -- doc tables
        SELECT 
            vector_store.migration_19_get_migration_scope_count(schema_name || '.t_' || table_prefix || '_doc', 'attr_ids') as scope_count,
            vector_store.migration_19_get_processed_count(schema_name || '.t_' || table_prefix || '_doc', 'doc_attributes') as processed_count,
            vector_store.migration_19_get_remaining_count(schema_name || '.t_' || table_prefix || '_doc', 'doc_attributes', 'attr_ids') as remaining_count
        FROM task_details
        UNION ALL
        -- emb tables  
        SELECT 
            vector_store.migration_19_get_migration_scope_count(schema_name || '.t_' || table_prefix || '_emb', 'attr_ids') as scope_count,
            vector_store.migration_19_get_processed_count(schema_name || '.t_' || table_prefix || '_emb', 'emb_attributes') as processed_count,
            vector_store.migration_19_get_remaining_count(schema_name || '.t_' || table_prefix || '_emb', 'emb_attributes', 'attr_ids') as remaining_count
        FROM task_details
        UNION ALL
        -- emb tables attr2 (using attr_ids2 -> attributes)
        SELECT 
            vector_store.migration_19_get_migration_scope_count(schema_name || '.t_' || table_prefix || '_emb', 'attr_ids2') as scope_count,
            vector_store.migration_19_get_processed_count(schema_name || '.t_' || table_prefix || '_emb', 'attributes') as processed_count,
            vector_store.migration_19_get_remaining_count(schema_name || '.t_' || table_prefix || '_emb', 'attributes', 'attr_ids2') as remaining_count
        FROM task_details
        UNION ALL
        -- doc_processing tables
        SELECT 
            vector_store.migration_19_get_migration_scope_count(schema_name || '.t_' || table_prefix || '_doc_processing', 'attr_ids') as scope_count,
            vector_store.migration_19_get_processed_count(schema_name || '.t_' || table_prefix || '_doc_processing', 'doc_attributes') as processed_count,
            vector_store.migration_19_get_remaining_count(schema_name || '.t_' || table_prefix || '_doc_processing', 'doc_attributes', 'attr_ids') as remaining_count
        FROM task_details
        UNION ALL
        -- emb_processing tables
        SELECT 
            vector_store.migration_19_get_migration_scope_count(schema_name || '.t_' || table_prefix || '_emb_processing', 'attr_ids') as scope_count,
            vector_store.migration_19_get_processed_count(schema_name || '.t_' || table_prefix || '_emb_processing', 'emb_attributes') as processed_count,
            vector_store.migration_19_get_remaining_count(schema_name || '.t_' || table_prefix || '_emb_processing', 'emb_attributes', 'attr_ids') as remaining_count
        FROM task_details
        UNION ALL
        -- emb_processing tables attr2 (using attr_ids2 -> attributes)
        SELECT 
            vector_store.migration_19_get_migration_scope_count(schema_name || '.t_' || table_prefix || '_emb_processing', 'attr_ids2') as scope_count,
            vector_store.migration_19_get_processed_count(schema_name || '.t_' || table_prefix || '_emb_processing', 'attributes') as processed_count,
            vector_store.migration_19_get_remaining_count(schema_name || '.t_' || table_prefix || '_emb_processing', 'attributes', 'attr_ids2') as remaining_count
        FROM task_details
    )
    SELECT 
        COALESCE(SUM(scope_count), 0)::BIGINT,
        COALESCE(SUM(processed_count), 0)::BIGINT, 
        COALESCE(SUM(remaining_count), 0)::BIGINT
    FROM all_table_stats;
END
$$ LANGUAGE plpgsql;

-- Get all replication units with their overall progress
CREATE OR REPLACE FUNCTION vector_store.migration_19_get_replication_units_with_progress()
RETURNS TABLE (
    isolation_id TEXT,
    collection_id TEXT,
    profile_id TEXT,
    table_prefix TEXT,
    total_records BIGINT,
    processed_records BIGINT,
    remaining_records BIGINT,
    status TEXT
) AS $$
DECLARE
    iso_record RECORD;
    col_record RECORD;
    profile_record RECORD;
    target_schema TEXT;
    schema_exists BOOLEAN;
    table_exists BOOLEAN;
    unit_key TEXT;
    unit_status TEXT;
    unit_total BIGINT;
    unit_processed BIGINT;
    unit_remaining BIGINT;
BEGIN
    -- Iterate through isolations from vector_store.isolations
    FOR iso_record IN 
        SELECT iso_id, iso_prefix 
        FROM vector_store.isolations
    LOOP
        -- Use correct schema naming pattern: vector_store_<iso_prefix>
        target_schema := CONCAT('vector_store_', iso_record.iso_prefix);
        
        -- Check if schema exists
        SELECT EXISTS (
            SELECT 1 FROM information_schema.schemata s
            WHERE s.schema_name = target_schema
        ) INTO schema_exists;
        
        -- Skip this isolation if schema doesn't exist
        IF NOT schema_exists THEN
            CONTINUE;
        END IF;
        
        -- Check if collections table exists in this schema
        SELECT EXISTS (
            SELECT 1 FROM information_schema.tables t
            WHERE t.table_schema = target_schema 
            AND t.table_name = 'collections'
        ) INTO table_exists;
        
        -- Skip this isolation if collections table doesn't exist
        IF NOT table_exists THEN
            CONTINUE;
        END IF;
        
        -- For each isolation, get collections from its schema (with exception handling)
        BEGIN
            FOR col_record IN 
                EXECUTE FORMAT('SELECT col_id FROM %I.collections', target_schema)
            LOOP
                -- Check if collection_emb_profiles table exists
                SELECT EXISTS (
                    SELECT 1 FROM information_schema.tables t
                    WHERE t.table_schema = target_schema 
                    AND t.table_name = 'collection_emb_profiles'
                ) INTO table_exists;
                
                -- Skip if collection_emb_profiles table doesn't exist
                IF NOT table_exists THEN
                    CONTINUE;
                END IF;
                
                -- For each collection, get profiles from collection_emb_profiles (with exception handling)
                BEGIN
                    FOR profile_record IN 
                        EXECUTE FORMAT('SELECT profile_id, tables_prefix FROM %I.collection_emb_profiles WHERE col_id = %L', 
                                      target_schema, col_record.col_id)
                    LOOP
                        -- Create unit configuration key
                        unit_key := FORMAT('attribute_replication_v0.19.0_%s_%s_%s',
                                          iso_record.iso_id, col_record.col_id, profile_record.profile_id);
                        
                        -- Check unit completion status
                        SELECT COALESCE(
                            (SELECT c.value FROM vector_store.configuration c WHERE c.key = unit_key),
                            'pending'
                        ) INTO unit_status;
                        
                        -- Calculate unit statistics (sum of all 6 table types for this unit)
                        unit_total := 0;
                        unit_processed := 0;
                        unit_remaining := 0;
                        
                        -- Doc table
                        unit_total := unit_total + vector_store.migration_19_get_migration_scope_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_doc', 'attr_ids');
                        unit_processed := unit_processed + vector_store.migration_19_get_processed_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_doc', 'doc_attributes', 'attr_ids');
                        unit_remaining := unit_remaining + vector_store.migration_19_get_remaining_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_doc', 'doc_attributes', 'attr_ids');
                        
                        -- Emb table (attr_ids -> emb_attributes)
                        unit_total := unit_total + vector_store.migration_19_get_migration_scope_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb', 'attr_ids');
                        unit_processed := unit_processed + vector_store.migration_19_get_processed_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb', 'emb_attributes', 'attr_ids');
                        unit_remaining := unit_remaining + vector_store.migration_19_get_remaining_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb', 'emb_attributes', 'attr_ids');
                        
                        -- Emb table (attr_ids2 -> attributes)
                        unit_total := unit_total + vector_store.migration_19_get_migration_scope_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb', 'attr_ids2');
                        unit_processed := unit_processed + vector_store.migration_19_get_processed_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb', 'attributes', 'attr_ids2');
                        unit_remaining := unit_remaining + vector_store.migration_19_get_remaining_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb', 'attributes', 'attr_ids2');
                        
                        -- Doc processing table
                        unit_total := unit_total + vector_store.migration_19_get_migration_scope_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_doc_processing', 'attr_ids');
                        unit_processed := unit_processed + vector_store.migration_19_get_processed_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_doc_processing', 'doc_attributes', 'attr_ids');
                        unit_remaining := unit_remaining + vector_store.migration_19_get_remaining_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_doc_processing', 'doc_attributes', 'attr_ids');
                        
                        -- Emb processing table (attr_ids -> emb_attributes)
                        unit_total := unit_total + vector_store.migration_19_get_migration_scope_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb_processing', 'attr_ids');
                        unit_processed := unit_processed + vector_store.migration_19_get_processed_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb_processing', 'emb_attributes', 'attr_ids');
                        unit_remaining := unit_remaining + vector_store.migration_19_get_remaining_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb_processing', 'emb_attributes', 'attr_ids');
                        
                        -- Emb processing table (attr_ids2 -> attributes)
                        unit_total := unit_total + vector_store.migration_19_get_migration_scope_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb_processing', 'attr_ids2');
                        unit_processed := unit_processed + vector_store.migration_19_get_processed_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb_processing', 'attributes', 'attr_ids2');
                        unit_remaining := unit_remaining + vector_store.migration_19_get_remaining_count(
                            target_schema || '.t_' || profile_record.tables_prefix || '_emb_processing', 'attributes', 'attr_ids2');
                        
                        -- Return this unit
                        isolation_id := iso_record.iso_id;
                        collection_id := col_record.col_id;
                        profile_id := profile_record.profile_id;
                        table_prefix := profile_record.tables_prefix;
                        total_records := unit_total;
                        processed_records := unit_processed;
                        remaining_records := unit_remaining;
                        status := unit_status;
                        
                        RETURN NEXT;
                    END LOOP;
                EXCEPTION
                    WHEN OTHERS THEN
                        -- Skip this collection if there's an error accessing collection_emb_profiles
                        CONTINUE;
                END;
            END LOOP;
        EXCEPTION
            WHEN OTHERS THEN
                -- Skip this isolation if there's an error accessing collections
                CONTINUE;
        END;
    END LOOP;
END
$$ LANGUAGE plpgsql;

-- Get table details for a specific replication unit
CREATE OR REPLACE FUNCTION vector_store.migration_19_get_unit_table_details(
    p_isolation_id TEXT,
    p_collection_id TEXT,
    p_profile_id TEXT
)
RETURNS TABLE (
    table_name TEXT,
    source_column TEXT,
    target_column TEXT,
    attr_table TEXT,
    total_records BIGINT,
    processed_records BIGINT,
    remaining_records BIGINT
) AS $$
DECLARE
    target_schema TEXT;
    table_prefix_val TEXT;
    iso_prefix_val TEXT;
BEGIN
    -- Get isolation prefix and table prefix for this unit
    SELECT i.iso_prefix INTO iso_prefix_val
    FROM vector_store.isolations i 
    WHERE i.iso_id = p_isolation_id;
    
    IF iso_prefix_val IS NULL THEN
        RETURN;
    END IF;
    
    target_schema := CONCAT('vector_store_', iso_prefix_val);
    
    -- Get table prefix for this unit
    EXECUTE FORMAT('SELECT tables_prefix FROM %I.collection_emb_profiles WHERE col_id = %L AND profile_id = %L',
                   target_schema, p_collection_id, p_profile_id) INTO table_prefix_val;
    
    IF table_prefix_val IS NULL THEN
        RETURN;
    END IF;
    
    -- Return details for all 6 table types in this unit
    
    -- Doc table
    table_name := target_schema || '.t_' || table_prefix_val || '_doc';
    source_column := 'attr_ids';
    target_column := 'doc_attributes';
    attr_table := target_schema || '.t_' || table_prefix_val || '_attr';
    total_records := vector_store.migration_19_get_migration_scope_count(table_name, source_column);
    processed_records := vector_store.migration_19_get_processed_count(table_name, target_column, source_column);
    remaining_records := vector_store.migration_19_get_remaining_count(table_name, target_column, source_column);
    RETURN NEXT;
    
    -- Emb table (attr_ids -> emb_attributes)
    table_name := target_schema || '.t_' || table_prefix_val || '_emb';
    source_column := 'attr_ids';
    target_column := 'emb_attributes';
    attr_table := target_schema || '.t_' || table_prefix_val || '_attr';
    total_records := vector_store.migration_19_get_migration_scope_count(table_name, source_column);
    processed_records := vector_store.migration_19_get_processed_count(table_name, target_column, source_column);
    remaining_records := vector_store.migration_19_get_remaining_count(table_name, target_column, source_column);
    RETURN NEXT;
    
    -- Emb table (attr_ids2 -> attributes)  
    table_name := target_schema || '.t_' || table_prefix_val || '_emb';
    source_column := 'attr_ids2';
    target_column := 'attributes';
    attr_table := target_schema || '.t_' || table_prefix_val || '_attr';
    total_records := vector_store.migration_19_get_migration_scope_count(table_name, source_column);
    processed_records := vector_store.migration_19_get_processed_count(table_name, target_column, source_column);
    remaining_records := vector_store.migration_19_get_remaining_count(table_name, target_column, source_column);
    RETURN NEXT;
    
    -- Doc processing table
    table_name := target_schema || '.t_' || table_prefix_val || '_doc_processing';
    source_column := 'attr_ids';
    target_column := 'doc_attributes';
    attr_table := target_schema || '.t_' || table_prefix_val || '_attr';
    total_records := vector_store.migration_19_get_migration_scope_count(table_name, source_column);
    processed_records := vector_store.migration_19_get_processed_count(table_name, target_column, source_column);
    remaining_records := vector_store.migration_19_get_remaining_count(table_name, target_column, source_column);
    RETURN NEXT;
    
    -- Emb processing table (attr_ids -> emb_attributes)
    table_name := target_schema || '.t_' || table_prefix_val || '_emb_processing';
    source_column := 'attr_ids';
    target_column := 'emb_attributes';
    attr_table := target_schema || '.t_' || table_prefix_val || '_attr';
    total_records := vector_store.migration_19_get_migration_scope_count(table_name, source_column);
    processed_records := vector_store.migration_19_get_processed_count(table_name, target_column, source_column);
    remaining_records := vector_store.migration_19_get_remaining_count(table_name, target_column, source_column);
    RETURN NEXT;
    
    -- Emb processing table (attr_ids2 -> attributes)
    table_name := target_schema || '.t_' || table_prefix_val || '_emb_processing';
    source_column := 'attr_ids2';
    target_column := 'attributes';
    attr_table := target_schema || '.t_' || table_prefix_val || '_attr';
    total_records := vector_store.migration_19_get_migration_scope_count(table_name, source_column);
    processed_records := vector_store.migration_19_get_processed_count(table_name, target_column, source_column);
    remaining_records := vector_store.migration_19_get_remaining_count(table_name, target_column, source_column);
    RETURN NEXT;
    
EXCEPTION
    WHEN OTHERS THEN
        -- If there's any error, return empty result set
        RETURN;
END
$$ LANGUAGE plpgsql;
