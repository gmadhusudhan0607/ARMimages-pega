/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package specs

import (
	"embed"
)

// ModelsSpecs contains all model specification files
//
//go:embed aws gcp azure
var ModelsSpecs embed.FS
