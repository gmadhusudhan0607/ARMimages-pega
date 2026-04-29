CREATE OR REPLACE FUNCTION vector_store.table_exists(name TEXT) RETURNS boolean AS
$$
DECLARE
    result   boolean;
    tbl_path text[];
    sch_name text;
    tbl_name text;
BEGIN
    tbl_path = string_to_array(name, '.');
    IF array_length(tbl_path, 1) > 1 THEN
        sch_name = tbl_path[1];
        tbl_name = tbl_path[2];
    ELSE
        sch_name = 'public';
        tbl_name = name;
    END IF;

    SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = sch_name AND table_name = tbl_name) INTO result;
    RETURN result;
END
$$ LANGUAGE plpgsql;
