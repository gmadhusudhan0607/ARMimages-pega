SELECT DOC.doc_id, DOC.status, DOC.created_at, DOC.record_timestamp, DOC.error_message
FROM %[1]s DOC
%[2]s
ORDER BY DOC.doc_id