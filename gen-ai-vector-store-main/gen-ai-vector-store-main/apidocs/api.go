//
// Copyright (c) 2023 Pegasystems Inc.
// All rights reserved.
//

package apidocs

import (
	"embed"
)

//go:embed service.yaml
//go:embed ops.yaml
var FS embed.FS
