/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"github.com/gin-gonic/gin"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/mapping"
)

func HandleGetMappingsRequest(ctx context.Context, mappings *mapping.SyncMappingStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Debug("Serving GET /mappings request")
		r := mappings.Read()
		c.JSON(200, r)
		l.Debugf("Fetched mappings: %v", r)
	}
}
