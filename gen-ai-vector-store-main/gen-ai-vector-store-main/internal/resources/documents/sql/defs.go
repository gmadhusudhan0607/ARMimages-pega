/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package documents_sql

import _ "embed"

//go:embed find_documents.sql
var FindDocumentsSqlQueryTemplate string

//go:embed list_documents.sql
var ListDocumentsSqlQueryTemplate string

//go:embed list_documents_without_filters.sql
var ListDocumentsWithoutFiltersSqlQueryTemplate string

//go:embed get_document.sql
var GetDocumentSqlQueryTemplate string

//go:embed delete_document.sql
var DeleteDocumentSqlQueryTemplate string

//go:embed delete_documents.sql
var DeleteDocumentsSqlQueryTemplate string

//go:embed list_documents_paginated.sql
var ListDocumentsPaginatedSqlQueryTemplate string

//go:embed list_documents_paginated_without_filters.sql
var ListDocumentsPaginatedWithoutFiltersSqlQueryTemplate string

//go:embed count_documents_by_status.sql
var CountDocumentsByStatusSqlQueryTemplate string

//go:embed count_documents_chunks_pending.sql
var CountDocumentsChunksPendingSqlQueryTemplate string

//go:embed count_document_chunks_processed.sql
var CountDocumentChunksProcessedSqlQueryTemplate string
