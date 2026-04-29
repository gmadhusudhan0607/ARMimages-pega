/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package health

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/mapping"
)

type RespErr struct {
	StatusCode int64  `json:"statusCode"`
	Message    string `json:"message"`
}

func GetLiveness(c *gin.Context) {
	c.Status(http.StatusOK)
}

func GetOpsReadiness(ctx context.Context, m *mapping.SyncMappingStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		if cntx.IsUseGenAiInfraModels(ctx) && helpers.HelperSuite.GetEnvOrFalse("USE_AUTO_MAPPING") {
			r := m.Read()
			if len(r) == 0 {
				l.Errorf("Service Unavailable. No GenAI Infra mappings available to be served from LLMAccountID=%s SaxCell=%s",
					helpers.HelperSuite.GetEnvOrDefault("LLM_ACCOUNT_ID", "undefined"),
					helpers.HelperSuite.GetEnvOrDefault("SAX_CELL", "undefined"))
				c.JSON(http.StatusServiceUnavailable, RespErr{
					StatusCode: http.StatusInternalServerError,
					Message:    "No GenAI Infra mappings available. Verify GenAI Gateway Service configuration.",
				})
				return
			}
		}
		c.Status(http.StatusOK)
	}
}

func GetReadiness(c *gin.Context) {
	c.Status(http.StatusOK)
}

func GetReadinessDependingOnMappings(ctx context.Context, mappingsGetter infra.ConfigLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		// can read mappings from gateway ops service
		l := cntx.LoggerFromContext(ctx).Sugar()

		m, e := mappingsGetter(ctx)
		msg := ""
		if e != nil {
			msg = "Service Not Ready: error retrieving GenAI Infra mappings: " + e.Error()
			l.Error(msg)
			c.JSON(http.StatusServiceUnavailable, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    msg,
			})
			return
		}
		if len(m) == 0 {
			msg = "Service Not Ready: No GenAI Infra mappings available for routing GenAI requests to AWS Bedrock"
			l.Error(msg)
			c.JSON(http.StatusServiceUnavailable, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    msg,
			})
			return
		}
		c.Status(http.StatusOK)
	}
}
