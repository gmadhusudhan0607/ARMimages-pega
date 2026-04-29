/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Path Normalization
 * -----------------------------
 * Purpose: Normalizes incoming request paths for consistent metrics labeling and monitoring.
 * Usage: Add PathNormalizationMiddleware to your Gin middleware chain to automatically normalize paths
 *        based on configured patterns. The normalized path is set in the Gin context under the key "normalizedPath".
 * Configuration: Update urlPatterns to define which paths should be normalized and how.
 */

package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ucarion/urlpath"
)

type urlPatternDef struct {
	pattern     urlpath.Path
	normalized  string
	appendTrail bool
}

var urlPatterns = []urlPatternDef{
	// V1 patterns - most specific first
	{
		pattern:     urlpath.New("/v1/:isolationID/collections/:collectionName/documents/:documentID"),
		normalized:  "/v1/<isolationID>/collections/<collectionName>/documents/<documentID>",
		appendTrail: false,
	},
	{
		pattern:     urlpath.New("/v1/:isolationID/collections/:collectionName/documents"),
		normalized:  "/v1/<isolationID>/collections/<collectionName>/documents",
		appendTrail: false,
	},
	{
		pattern:     urlpath.New("/v1/:isolationID/collections/:collectionName/*"),
		normalized:  "/v1/<isolationID>/collections/<collectionName>/",
		appendTrail: true,
	},
	{
		pattern:     urlpath.New("/v1/:isolationID/smart-attributes-group/:groupID"),
		normalized:  "/v1/<isolationID>/smart-attributes-group/<groupID>",
		appendTrail: false,
	},
	{
		pattern:     urlpath.New("/v1/:isolationID/smart-attributes-group"),
		normalized:  "/v1/<isolationID>/smart-attributes-group",
		appendTrail: false,
	},
	// V2 patterns - most specific first
	{
		pattern:     urlpath.New("/v2/:isolationID/collections/:collectionID/documents/:documentID/chunks"),
		normalized:  "/v2/<isolationID>/collections/<collectionID>/documents/<documentID>/chunks",
		appendTrail: false,
	},
	{
		pattern:     urlpath.New("/v2/:isolationID/collections/:collectionID/find-documents"),
		normalized:  "/v2/<isolationID>/collections/<collectionID>/find-documents",
		appendTrail: false,
	},
	{
		pattern:     urlpath.New("/v2/:isolationID/collections/:collectionID"),
		normalized:  "/v2/<isolationID>/collections/<collectionID>",
		appendTrail: false,
	},
	{
		pattern:     urlpath.New("/v2/:isolationID/collections"),
		normalized:  "/v2/<isolationID>/collections",
		appendTrail: false,
	},
}

// PathNormalizationMiddleware extracts and normalizes the request path for metrics labeling.
// It matches the request path against configured patterns and sets the normalized path in the Gin context.
// If no pattern matches, the original path is used.
func PathNormalizationMiddleware(c *gin.Context) {

	path := c.Request.URL.Path
	for _, def := range urlPatterns {
		match, ok := def.pattern.Match(path)
		if ok {
			if def.appendTrail {
				path = def.normalized + match.Trailing
			} else {
				path = def.normalized
			}
			c.Set("normalizedPath", path)
			c.Next()
			return
		}
	}
	c.Set("normalizedPath", path)
	c.Next()
}
