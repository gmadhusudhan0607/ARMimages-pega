/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package queries

import _ "embed"

//go:embed metrics_tables_size.sql
var Function_metrics_tables_size string

//go:embed metrics_last_modified_time.sql
var Function_metrics_last_modified_time string

//go:embed metrics_document_count.sql
var Function_metrics_document_count string

//go:embed metrics_iso_size.sql
var Function_metrics_iso_size string
