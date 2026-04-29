WITH all_documents AS (
SELECT
	DOC.doc_id,
	DOC.status,
	DOC.created_at,
	DOC.record_timestamp,
	DOC.error_message,
    vector_store.attributes_as_jsonb_by_ids('%[1]s.%[2]s_attr', DOC.attr_ids) as attributes,
	vector_store.attributes_as_jsonb_by_ids('%[1]s.%[2]s_attr', DOC_PROC.attr_ids) as processing_attributes
FROM
	%[1]s.%[2]s_doc DOC
LEFT JOIN %[1]s.%[2]s_doc_processing DOC_PROC
ON
	DOC.doc_id = DOC_PROC.doc_id
%[3]s
),
completed_chunks AS (
SELECT
	COUNT(*) as chunks,
	EMB.doc_id
FROM
	%[1]s.%[2]s_emb EMB
WHERE
	EMB.status = 'COMPLETED'
	AND EMB.doc_id in (SELECT doc_id FROM all_documents)
GROUP BY
	EMB.doc_id
),
pending_chunks AS (
SELECT
	COUNT(*) AS chunks,
	(content->>'doc_id')::text AS doc_id
FROM
	vector_store.embedding_queue
WHERE
	(content->>'iso_id') = $1
		AND (content->>'col_id') = $2
			AND (content->>'doc_id')::text IN (SELECT doc_id::text FROM all_documents)
		GROUP BY
			(content->>'doc_id')::text
),
error_chunks AS (
SELECT
	COUNT(*) AS chunks,
	EMB.doc_id
FROM
	%[1]s.%[2]s_emb EMB
WHERE
	EMB.status = 'ERROR'
	AND EMB.doc_id IN (SELECT doc_id FROM all_documents)
GROUP BY
	EMB.doc_id
)
SELECT
	AD.doc_id,
	AD.status,
	AD.created_at,
	AD.record_timestamp,
	AD.error_message,
	AD.attributes,
	AD.processing_attributes,
	COALESCE(CC.chunks, 0) AS completed_chunks,
	COALESCE(PC.chunks, 0) AS pending_chunks,
	COALESCE(EC.chunks, 0) AS error_chunks
FROM all_documents AD
LEFT JOIN completed_chunks CC ON AD.doc_id = CC.doc_id
LEFT JOIN pending_chunks PC ON AD.doc_id = PC.doc_id
LEFT JOIN error_chunks EC ON AD.doc_id = EC.doc_id
ORDER BY AD.doc_id
%[4]s;