CREATE OR REPLACE FUNCTION vector_store.embeddings_queue_put(_content json, _delay_sec int default 0) RETURNS void AS
$BODY$
DECLARE
    cur_time        TIMESTAMP;
    _postpone_until TIMESTAMP;
    ts              integer;
BEGIN
    cur_time = (SELECT current_timestamp);
    if _delay_sec > 0 then
        ts = (select cast(extract(epoch from current_timestamp) as integer));
        _postpone_until = TO_TIMESTAMP(ts + _delay_sec);
    else
        _postpone_until = cur_time;
    end if;
    INSERT INTO vector_store.embedding_queue (id, created_at, postpone_until, content)
    VALUES (gen_random_uuid(), cur_time, _postpone_until, _content);
END;
$BODY$ LANGUAGE plpgsql;