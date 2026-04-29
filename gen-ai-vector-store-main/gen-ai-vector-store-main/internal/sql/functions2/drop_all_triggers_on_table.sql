CREATE OR REPLACE FUNCTION vector_store.drop_all_triggers_on_table(schema_name TEXT, table_name TEXT) RETURNS VOID AS
$$
DECLARE
    r               RECORD;
    full_table_name TEXT;
    triggers_count  INTEGER;
BEGIN
    full_table_name := format('%s.%s', schema_name, table_name);
    IF NOT vector_store.table_exists(full_table_name) THEN
        RAISE NOTICE 'skip %. table not found', full_table_name;
        RETURN;
    END IF;

    triggers_count = (SELECT coalesce(sum(count), 0)
                      FROM (SELECT count(trigger_name) as count
                            FROM information_schema.triggers
                            WHERE event_object_schema = schema_name AND event_object_table = table_name
                            GROUP BY trigger_name) T0 );
    if coalesce(triggers_count, 0) = 0 then
        RAISE NOTICE 'skip %. no triggers found', full_table_name;
        RETURN;
    else
        RAISE NOTICE 'found % triggers on %. dropping', triggers_count, full_table_name;
    end if;

    FOR r IN (SELECT trigger_name
              FROM information_schema.triggers
              WHERE event_object_schema = schema_name
                AND event_object_table = table_name
              GROUP BY trigger_name)
        LOOP
            EXECUTE format('DROP TRIGGER %s ON %s.%s', r.trigger_name, schema_name, table_name);
            RAISE NOTICE 'dropped triggers xx_% on table %', r.trigger_name, full_table_name;
        END LOOP;
END
$$ LANGUAGE plpgsql;