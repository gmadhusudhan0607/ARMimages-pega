
SELECT DOC.doc_id, DOC.status, DOC.error_message
FROM %[1]s.%[2]s_doc DOC
%[3]s
