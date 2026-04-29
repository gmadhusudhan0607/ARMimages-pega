/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"unicode/utf8"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// logTruncateMaxLen controls the maximum length of JSON string values in debug logs.
// Set LOG_TRUNCATE_LONG_STRINGS to an integer (e.g., 500) to truncate strings longer
// than that limit. Values < 1 or unset mean no truncation (default).
var logTruncateMaxLen = parseIntEnv("LOG_TRUNCATE_LONG_STRINGS", 0)

func parseIntEnv(key string, defaultVal int) int {
	s := os.Getenv(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

type requestEntry struct {
	Method     string `json:"method,omitempty"`
	Headers    any    `json:"headers,omitempty"`
	RequestURI string `json:"requestUri,omitempty"`
	IsJson     bool   `json:"isJson,omitempty"`
}

type responseEntry struct {
	Status  int  `json:"status,omitempty"`
	Headers any  `json:"headers,omitempty"`
	IsJson  bool `json:"isJson,omitempty"`
}

func RequestLoggerMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()
		DoLogRequest(l, c.Request, "Received ")
	}
}

func ResponseLoggerMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		l := cntx.LoggerFromContext(ctx).Sugar()

		// Only capture response bodies when debug logging is enabled.
		// Response body buffering is expensive for large payloads and
		// serves no purpose when the debug log line will be discarded.
		if !l.Desugar().Core().Enabled(zap.DebugLevel) {
			c.Next()
			return
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()
		body := blw.body.String()
		respEntry := responseEntry{
			Status:  blw.Status(),
			Headers: blw.Header(),
			IsJson:  isJSON(body),
		}
		objJson, err := json.Marshal(respEntry)
		if err != nil {
			l.Debugf("failed to unmarshal object [%#v]: %s", respEntry, err.Error())
		}
		logBody := maybeTruncate(body)
		l.Debugf("Returning Response:\n%s\nwith Body:\n%s", objJson, logBody)
	}
}

func DoLogRequest(l *zap.SugaredLogger, r *http.Request, msgPrefix string) {
	body := GetCopyOfBodyBytes(&r.Body)
	reqEntry := requestEntry{
		Method:     r.Method,
		Headers:    r.Header,
		RequestURI: r.RequestURI,
		IsJson:     isJSON(string(body)),
	}
	objJson, err := json.Marshal(reqEntry)
	if err != nil {
		l.Debugf("failed to unmarshal object [%#v]: %s", reqEntry, err.Error())
	}
	logBody := maybeTruncate(string(body))
	l.Debugf("%sRequest:\n%s\nwith Body:\n%s\nLength:%d", msgPrefix, objJson, logBody, len(body))
}

func DoLogResponse(l *zap.SugaredLogger, r *http.Response, msgPrefix string) {
	body := string(GetCopyOfBodyBytes(&r.Body))
	respEntry := responseEntry{
		Status:  r.StatusCode,
		Headers: r.Header,
		IsJson:  isJSON(body),
	}
	objJson, err := json.MarshalIndent(respEntry, "", "  ")
	if err != nil {
		l.Debugf("failed to unmarshal object [%#v]: %s", respEntry, err.Error())
	}
	logBody := maybeTruncate(body)
	l.Debugf("%sResponse:\n%s\nwith Body:\n%s", msgPrefix, objJson, logBody)
}

// truncateBody parses a JSON body, truncates any string values longer than maxLen
// to their first maxLen characters with a truncation notice, and re-serializes.
// Non-JSON bodies are returned as-is.
func truncateBody(body string, maxLen int) string {
	var parsed any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return body
	}
	truncateValue(&parsed, maxLen, 0)
	out, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return body
	}
	return string(out)
}

const maxTruncateDepth = 20

// truncateString truncates s to at most maxLen bytes on a valid UTF-8 boundary
// and appends a notice showing the original length.
func truncateString(s string, maxLen int) string {
	t := s[:maxLen]
	for !utf8.ValidString(t) && len(t) > 0 {
		t = t[:len(t)-1]
	}
	return t + fmt.Sprintf("... [TRUNCATED: %d bytes]", len(s))
}

// truncateValue recursively walks a parsed JSON value and truncates long strings.
// Recursion is bounded by maxTruncateDepth to prevent stack overflow on deeply nested payloads.
func truncateValue(v *any, maxLen int, depth int) {
	if depth > maxTruncateDepth {
		return
	}
	switch val := (*v).(type) {
	case string:
		if len(val) > maxLen {
			*v = truncateString(val, maxLen)
		}
	case map[string]any:
		for k, child := range val {
			truncateValue(&child, maxLen, depth+1)
			val[k] = child
		}
	case []any:
		for i, child := range val {
			truncateValue(&child, maxLen, depth+1)
			val[i] = child
		}
	default:
		// Other JSON primitives (float64, bool, nil) need no truncation.
	}
}

func maybeTruncate(body string) string {
	if logTruncateMaxLen > 0 && isJSON(body) {
		return truncateBody(body, logTruncateMaxLen)
	}
	return body
}

// isJSON checks that the string value of the field can unmarshall into valid json (object or array)
func isJSON(s string) bool {
	return json.Valid([]byte(s))
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func GetCopyOfBodyBytes(srcBody *io.ReadCloser) []byte {
	var buf bytes.Buffer
	tee := io.TeeReader(*srcBody, &buf)
	body, _ := io.ReadAll(tee)
	*srcBody = io.NopCloser(&buf)
	return body
}
