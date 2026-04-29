CREATE OR REPLACE FUNCTION vector_store.attributes_as_jsonb_by_ids(attr_table TEXT, attr_ids bigint[]) RETURNS JSONB AS
$$
DECLARE
    attributes JSONB;
BEGIN
    EXECUTE FORMAT('
        SELECT jsonb_agg(row_to_json(attr_row))
        FROM (SELECT name, type, array_agg(DISTINCT ATTR.value ORDER BY ATTR.value) as value
              FROM %1$s ATTR
              WHERE ATTR.attr_id = ANY( %2$L )
              GROUP BY ATTR.name, ATTR.type
        ) attr_row
        ', attr_table, attr_ids ) INTO attributes;
    if attributes is null then
        attributes = '[]'::json;
    end if;
    RETURN attributes;
END
$$ LANGUAGE plpgsql COST 100;
