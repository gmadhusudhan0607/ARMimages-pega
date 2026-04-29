/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package queue_sql

import _ "embed"

//go:embed do_create_table.sql
var Do_create_table string

//go:embed do_create_index.sql
var Do_do_create_index string

//go:embed embeddings_queue_get.sql
var Function_embeddings_queue_get string

//go:embed embeddings_queue_get_with_exception.sql
var Function_embeddings_queue_get_with_exception string

//go:embed embeddings_queue_put.sql
var Function_embeddings_queue_put string
