/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package queries

import _ "embed"

//go:embed do_create_extension_vector.sql
var Do_create_extension_vector string

//go:embed do_create_table_isolations_v2.sql
var Do_create_table_isolations_v2 string

//go:embed do_create_table_configuration_v2.sql
var Do_create_table_configuration_v2 string

//go:embed do_create_schema_vector_store.sql
var Do_create_schema_vector_store string
