SELECT COUNT(*)
FROM %s.%s_emb
WHERE doc_id = $1
AND status = 'COMPLETED';