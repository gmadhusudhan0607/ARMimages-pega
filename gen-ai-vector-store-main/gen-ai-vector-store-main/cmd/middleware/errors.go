/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Error Handling
 * -------------------------
 * Purpose: Converts errors from the Gin context into structured HTTP responses.
 * Usage: Add ErrorHandler() to your Gin middleware chain to automatically handle errors
 *        and return appropriate HTTP status codes and error messages.
 * Configuration: Uses internal error types (ResponseError) and converts other errors as needed.
 */

package middleware

import (
	"errors"

	respErrors "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/errors"
	"github.com/gin-gonic/gin"
)

// ErrorHandler creates a Gin middleware that processes errors collected in the Gin context.
// It converts recognized internal errors to structured HTTP responses, and falls back to a generic
// error response for unrecognized errors. Place this middleware after handlers that may set errors.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Skip if response already written
		if c.Writer.Written() {
			return
		}

		for _, err := range c.Errors {
			var respErr respErrors.ResponseError
			switch {
			case errors.As(err.Err, &respErr):
				c.AbortWithStatusJSON(respErr.Code, respErr)
			default:
				respErr = respErrors.ToResponseError(err)
				c.AbortWithStatusJSON(respErr.Code, respErr)
			}
		}
	}
}
