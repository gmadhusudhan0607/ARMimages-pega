WITH filtered_embeddings_with_distance AS (
    SELECT EMB.emb_id, emb.embedding <=> $1 as distance
    FROM %[1]s.%[2]s_emb EMB
    %[3]s
    /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
    ORDER BY distance
    LIMIT %[5]d
)
SELECT
    EMB.emb_id,
    EMB.doc_id,
    content,
    vector_store.attributes_as_jsonb_by_ids( '%[1]s.%[2]s_attr' , EMB.attr_ids2 ) as attributes,
    distance
FROM filtered_embeddings_with_distance FDEMB LEFT JOIN %[1]s.%[2]s_emb EMB ON FDEMB.emb_id = EMB.emb_id
WHERE distance <= %[4]s
ORDER BY distance, emb_id
LIMIT %[5]d
