/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/gin-gonic/gin"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/mapping"
)

var helperSuite = helpers.HelperSuite

func HandleGetDefaultsRequest(ctx context.Context, credsProvider mapping.CredentialsProvider, clientFactory mapping.ClientFactory) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Debug("Serving GET /models/defaults request")

		// Check feature flag for Pro model inclusion
		enableProModel := helpers.GetEnvOrFalse("ENABLE_PRO_MODEL_DEFAULT")
		l.Debugf("EnableProModelDefault flag: %v", enableProModel)

		//Check if GenAI Infra is enabled - if not, return empty response
		if !helpers.GetEnvOrFalse("USE_GENAI_INFRA") {
			l.Warnf("UseGenAIInfra is disabled. Please configure it in order to fetch default models.")
			emptyConfig := infra.DefaultModelConfig{
				Fast:  "",
				Smart: "",
				Pro:   "",
			}
			c.JSON(http.StatusOK, emptyConfig.ToResponse(enableProModel))
			return
		}

		stage := helperSuite.GetEnvOrPanic("STAGE_NAME")
		saxCell := helperSuite.GetEnvOrPanic("SAX_CELL")
		region := helperSuite.GetEnvOrPanic("LLM_MODELS_REGION")

		creds, err := credsProvider.GetCredentials()
		if err != nil {
			l.Errorf("Failed to get credentials: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error occurred while fetching aws credentials"})
			return
		}

		client := clientFactory(creds, region)

		config, err := mapping.LoadDefaultModelMapping(client, stage, saxCell)
		if err != nil {
			l.Errorf("Failed to load default model mapping: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load smart and fast defaults from aws secret"})
			return
		}

		// Return response based on feature flag
		response := config.ToResponse(enableProModel)
		c.JSON(http.StatusOK, response)
		l.Debugf("Fetched default model mapping (EnableProModel=%v): %+v", enableProModel, response)
	}
}
