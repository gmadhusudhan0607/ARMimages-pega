SELECT COUNT(*) 
FROM vector_store.embedding_queue 
WHERE (content->'iso_id')::jsonb ? $1 
AND (content->'col_id')::jsonb ? $2 
AND (content->'doc_id')::jsonb ? $3