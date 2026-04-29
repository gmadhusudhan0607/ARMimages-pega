/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

const (
	realtimePathPrefix = "/openai/deployments/"

	// maxRealtimeBodyBytes caps the request body size for realtime endpoints.
	// SDP bodies are typically a few KB; 1 MB is generous headroom.
	maxRealtimeBodyBytes int64 = 1 << 20 // 1 MB
)

// sensitiveQueryParams lists query parameter names that must not be forwarded
// to the upstream provider.
var sensitiveQueryParams = map[string]struct{}{
	"api-key":          {},
	"subscription-key": {},
}

// sanitizeLogValue replaces control characters (newlines, tabs, etc.) in s
// so that user-controlled input cannot inject fake log lines.
func sanitizeLogValue(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '_'
		}
		return r
	}, s)
}

// stripSensitiveQuery returns the raw query string with sensitive parameters removed.
func stripSensitiveQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		// Malformed query — drop it entirely rather than forwarding garbage.
		return ""
	}
	for param := range values {
		if _, ok := sensitiveQueryParams[strings.ToLower(param)]; ok {
			values.Del(param)
		}
	}
	return values.Encode()
}

// HandleRealtimeProxyRequest proxies realtime WebRTC requests to the Azure OpenAI
// APIM endpoint. Unlike chat/completions, realtime models are not in the model mapping,
// so this handler constructs the target URL directly from GENAI_URL.
func HandleRealtimeProxyRequest(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		defer l.Sync() //nolint:errcheck

		l.Infow("Serving request",
			"method", c.Request.Method,
			"uri", sanitizeLogValue(c.Request.RequestURI),
		)

		baseURL := cntx.AzureGenAIURL(ctx)
		if baseURL == "" {
			l.Error("GENAI_URL is not configured")
			c.JSON(http.StatusInternalServerError, RespErr{
				StatusCode: http.StatusInternalServerError,
				Message:    "service configuration error",
			})
			return
		}

		// Validate and clean the request path to prevent path traversal.
		cleanPath := path.Clean(c.Request.URL.Path)
		if !strings.HasPrefix(cleanPath, realtimePathPrefix) {
			l.Errorw("Invalid realtime path",
				"path", sanitizeLogValue(c.Request.URL.Path),
			)
			c.JSON(http.StatusBadRequest, RespErr{
				StatusCode: http.StatusBadRequest,
				Message:    "invalid realtime request path",
			})
			return
		}

		// Limit request body size to prevent abuse.
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRealtimeBodyBytes)

		baseURL = strings.TrimRight(baseURL, "/")
		query := stripSensitiveQuery(c.Request.URL.RawQuery)

		targetURL := baseURL + cleanPath
		if query != "" {
			targetURL += "?" + query
		}

		l.Infow("Redirecting request",
			"method", c.Request.Method,
			"target", sanitizeLogValue(targetURL),
		)

		CallTarget(c, ctx, targetURL, cntx.IsUseSax(ctx))

		l.Infow("Received response",
			"target", sanitizeLogValue(targetURL),
		)
	}
}
