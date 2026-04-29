/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package embeddings_sql

import _ "embed"

// Embedding queue

//go:embed find_chunks.sql
var FindChunksSqlQueryTemplate string
