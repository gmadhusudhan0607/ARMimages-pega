/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"context"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/gin-gonic/gin"
)

func SaxRequestEnrichment(ctx context.Context) gin.HandlerFunc {

	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		if cntx.IsUseSax(ctx) {
			//add new authorization header with SAX token
			l.Debugf("SaxRequestEnrichment: Adding new authorization header with SAX token")

			var saxJwt string
			var err error
			if saxJwt, err = cntx.IssueSaxClientToken(ctx); err != nil {
				l.Error(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"StatusCode": http.StatusInternalServerError,
					"Message":    err.Error(),
				})
				return
			}

			l.Debugf("SaxRequestEnrichment: Setting Authorization header with JWT token %s", maskValueTruncateAt(saxJwt, 5))
			c.Request.Header.Set("Authorization", "Bearer "+saxJwt)
		}
	}
}

func maskValueTruncateAt(v string, trunc int) string {
	t := v
	if len(t) > trunc {
		t = t[:trunc] + "[truncated]"
	}
	return t
}
