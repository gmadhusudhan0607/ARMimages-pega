/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"fmt"
	"net/http"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/gin-gonic/gin"
)

func ProviderEnabled(provider string) gin.HandlerFunc {
	return func(c *gin.Context) {

		if !cntx.IsLLMProviderConfigured(c, provider) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Provider %s is not enabled", provider)})
			return
		}
		c.Next()
	}
}
