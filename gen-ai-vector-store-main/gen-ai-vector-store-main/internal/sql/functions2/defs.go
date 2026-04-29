/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package functions

import _ "embed"

//go:embed table_exists.sql
var Function_table_exists string

//go:embed attributes_as_jsonb_by_ids.sql
var Function_attributes_as_jsonb_by_ids string

//go:embed embedding_statuses_as_json.sql
var Function_embedding_statuses_as_json string

//go:embed calculate_document_status.sql
var Function_calculate_document_status string

//go:embed drop_all_triggers_on_table.sql
var Function_drop_all_triggers_on_table string

//go:embed drop_all_triggers_on_collection.sql
var Function_drop_all_triggers_on_collection string

//go:embed lookup_resources_metadata.sql
var Function_lookup_resources_metadata string

//go:embed copy_schema.sql
var Function_copy_schema string

//go:embed schema_info.sql
var Function_schema_info string

//go:embed get_collection_document_count.sql
var Function_get_collection_document_count string

//go:embed get_db_metrics.sql
var Function_get_db_metrics string

//go:embed attribute_migration.sql
var Function_attribute_migration string
