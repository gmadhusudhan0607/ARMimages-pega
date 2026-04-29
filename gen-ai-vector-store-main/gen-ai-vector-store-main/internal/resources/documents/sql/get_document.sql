
SELECT doc_id, status, error_message
FROM %[1]s.%[2]s_doc
WHERE doc_id = $1
