// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
)

func RequestTimeoutMiddleware() gin.HandlerFunc {
	requestTimeout := helpers.GetRequestTimeout()

	logger := log.GetNamedLogger("timeout-middleware")
	logger.Debug("Request timeout middleware initialized",
		zap.Duration("timeout", requestTimeout))

	return timeout.New(
		timeout.WithTimeout(requestTimeout),
		timeout.WithResponse(func(c *gin.Context) {
			logger.Warn("request timeout exceeded",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Duration("timeout", requestTimeout),
			)

			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"code": "504",
				"message": fmt.Sprintf("operation timeout exceeded: request processing failed (limit: %.0fs)",
					requestTimeout.Seconds()),
				"timeout": requestTimeout.String(),
			})
		}),
	)
}
