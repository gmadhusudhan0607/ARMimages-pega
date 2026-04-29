/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

/*
 * Middleware: Request Logging
 * --------------------------
 * Purpose: Logs incoming HTTP request details, including headers and optionally the body (if small).
 * Usage: Add RequestLoggerMiddleware(ctx) to your Gin middleware chain to enable detailed request logging.
 * Configuration: Logging is conditional on debug level and body size (MaxBodySizeForLogging).
 *                Uses a named logger and supports JSON formatting for request/response entries.
 */

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// MaxBodySizeForLogging - Body size limits for logging
	MaxBodySizeForLogging = 10 * 1024 // 10 KB
)

type requestEntry struct {
	Method     string      `json:"method,omitempty"`
	Headers    interface{} `json:"headers,omitempty"`
	RequestURI string      `json:"requestUri,omitempty"`
	IsJson     bool        `json:"isJson,omitempty"`
}

type responseEntry struct {
	Status  int         `json:"status,omitempty"`
	Headers interface{} `json:"headers,omitempty"`
	IsJson  bool        `json:"isJson,omitempty"`
}

// RequestLoggerMiddleware returns a Gin middleware that logs incoming request details.
// Logging is performed only if debug level is enabled. For small request bodies, the body is also logged.
// The logger used is named "middleware" and supports structured logging via zap.
func RequestLoggerMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {

		logger := log.GetLoggerFromContext(c.Request.Context())

		// Skip logging if debug level is not enabled
		if !logger.Core().Enabled(log.DebugLevel) {
			c.Next()
			return
		}

		// Only perform expensive body copy if debug logging is enabled and body is small
		var bodyCopy []byte
		if c.Request.ContentLength > 0 && c.Request.ContentLength < MaxBodySizeForLogging {
			bodyCopy = GetCopyOfBodyBytes(&c.Request.Body)
		}

		entry := requestEntry{
			Method:     c.Request.Method,
			Headers:    c.Request.Header,
			RequestURI: c.Request.RequestURI,
			IsJson:     strings.Contains(c.Request.Header.Get("Content-Type"), "json"),
		}
		if len(bodyCopy) > 0 {
			logger.Debug("Incoming request",
				zap.Any("entry", entry),
				zap.ByteString("body", bodyCopy),
			)
		} else {
			logger.Debug("Incoming request",
				zap.Any("entry", entry),
			)
		}

		c.Next()
	}
}

// ResponseLoggerMiddleware logs response details
func ResponseLoggerMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {

		logger := log.GetLoggerFromContext(c.Request.Context())

		// Skip expensive logging setup if debug is not enabled
		if !logger.Core().Enabled(log.DebugLevel) {
			c.Next()
			return
		}

		blw := &bodyLogWriter{
			body:           bytes.NewBufferString(""),
			ResponseWriter: c.Writer,
			maxSize:        MaxBodySizeForLogging,
		}
		c.Writer = blw
		c.Next()

		// Log asynchronously after response has been sent
		status := blw.Status()
		headers := blw.Header().Clone()

		// Capture body only if it's not too large
		var responseBody string
		if blw.body.Len() <= MaxBodySizeForLogging {
			responseBody = blw.body.String()
		}

		go func(status int, headers http.Header, body string, logger *zap.Logger) {
			logResponseDetails(status, headers, body, logger)
		}(status, headers, responseBody, logger)
	}
}

// logResponseDetails logs response details
func logResponseDetails(status int, headers http.Header, body string, logger *zap.Logger) {
	isJsonBody := len(body) > 0 && isJSON([]byte(body))

	respEntry := responseEntry{
		Status:  status,
		Headers: headers,
		IsJson:  isJsonBody,
	}

	objJson, err := json.Marshal(respEntry)
	if err != nil {
		logger.Debug("failed to marshal response object", zap.Error(err))
		return
	}

	if len(body) > 0 {
		bodyTxt := strings.ReplaceAll(body, "\n", "\\n")
		logger.Debug("Returning Response", zap.ByteString("json", objJson), zap.String("body", helpers.ToTruncatedString(bodyTxt)))
	} else {
		logger.Debug("Returning Response (body logging skipped or empty)", zap.ByteString("json", objJson))
	}
}

// isJSON is a faster JSON validation that avoids full unmarshalling
func isJSON(data []byte) bool {
	// Quick check: valid JSON must start with { or [
	if len(data) == 0 || (data[0] != '{' && data[0] != '[') {
		return false
	}

	// Use a more lightweight check than full unmarshalling
	var j json.RawMessage
	return json.Unmarshal(data, &j) == nil
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body     *bytes.Buffer
	maxSize  int
	sizeLock sync.Mutex
	tooLarge bool
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	// Only buffer up to maxSize bytes for logging
	w.sizeLock.Lock()
	if !w.tooLarge && w.body.Len() < w.maxSize {
		remaining := w.maxSize - w.body.Len()
		if len(b) <= remaining {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remaining])
			w.tooLarge = true
		}
	}
	w.sizeLock.Unlock()

	// Always write everything to the actual response
	return w.ResponseWriter.Write(b)
}

// GetCopyOfBodyBytes reads the body and provides a copy while preserving the original
func GetCopyOfBodyBytes(srcBody *io.ReadCloser) []byte {
	if *srcBody == nil {
		return []byte{}
	}

	var buf bytes.Buffer
	tee := io.TeeReader(*srcBody, &buf)
	body, _ := io.ReadAll(tee)
	*srcBody = io.NopCloser(&buf)
	return body
}
