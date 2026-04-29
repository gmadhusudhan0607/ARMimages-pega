WITH filtered_embeddings_with_distance AS (
    SELECT EMB.doc_id, EMB.emb_id, emb.embedding <=>
        $1
        as distance
    FROM %[1]s.%[2]s_emb EMB
        %[3]s
    /* cannot order by emb_id, because this prevents HNSW index usage. Therefore, returned items can be not deterministic. */
    ORDER BY distance
    LIMIT %[6]d
), runked_filtered_embeddings AS (
    SELECT
        emb_id,
        distance,
        ROW_NUMBER() OVER (partition by doc_id ORDER BY distance) as rank
    FROM filtered_embeddings_with_distance
    ORDER BY distance, emb_id
)
SELECT
    DOC.doc_id,
    distance,
    vector_store.attributes_as_jsonb_by_ids( '%[1]s.%[2]s_attr', EMB.attr_ids2 ) as attributes
FROM runked_filtered_embeddings RFE
    LEFT JOIN %[1]s.%[2]s_emb EMB ON RFE.emb_id = EMB.emb_id
    LEFT JOIN %[1]s.%[2]s_doc DOC ON DOC.doc_id = EMB.doc_id
WHERE distance <= %[4]s AND rank = 1
ORDER BY distance, doc_id
LIMIT %[5]d
