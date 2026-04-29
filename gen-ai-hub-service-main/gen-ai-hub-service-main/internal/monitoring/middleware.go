/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package monitoring

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/jwt"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/repository"
)

func RequestReporter(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(c)
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Changed to info, because it is too noisy, during tests without SAX token
			l.Info("unable to monitor request - no Authorization header found")
			return
		}

		authToken, err := helpers.ExtractTokenValue(authHeader)
		if err != nil {
			l.Warn("unable to monitor request - JWT token not found")
			return
		}

		token, err := jwt.ParseString(authToken)
		if err != nil {
			l.Warn("unable to monitor request - JWT token cannot be parsed")
			return
		}

		var guid string
		if res, ok := token.Get("guid"); ok {
			guid = res.(string)
		}

		if len(guid) == 0 {
			l.Warn("unable to monitor request - JWT token have no claim with key guid")
			return
		}

		evt := repository.NewEvent(guid, time.Now().Unix())
		publishEvent(ctx, evt)
	}
}
