CREATE OR REPLACE FUNCTION vector_store.sort_bigint_array (bigint[])
    RETURNS bigint[] LANGUAGE SQL
AS $$
SELECT ARRAY(SELECT unnest($1) ORDER BY 1)
$$;
