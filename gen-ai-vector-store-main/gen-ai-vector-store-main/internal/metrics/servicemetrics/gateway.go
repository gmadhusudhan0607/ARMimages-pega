/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics

import (
	"net/http"
	"strings"
	"sync"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
)

// Gateway stores metrics related to GenAI Gateway responses
type Gateway struct {
	mu      sync.RWMutex
	headers map[string]string
}

// Required gateway headers and their default values
var requiredGatewayHeaders = map[string]string{
	headers.GatewayResponseTimeMs:  "-1",
	headers.GatewayInputTokens:     "-1",
	headers.GatewayOutputTokens:    "-1",
	headers.GatewayTokensPerSecond: "-1",
	headers.GatewayModelId:         "not-set",
	headers.GatewayRegion:          "not-set",
	headers.GatewayRetryCount:      "-1",
}

// SetGenaiHeadersFromResponse extracts all X-Genai-* headers from the HTTP response
func (g *Gateway) SetGenaiHeadersFromResponse(resp *http.Response) {
	if resp == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.headers == nil {
		g.headers = make(map[string]string)
	}

	// Extract all headers that start with "X-Genai-" (case-insensitive)
	for headerName, headerValues := range resp.Header {
		if strings.HasPrefix(strings.ToLower(headerName), "x-genai-") {
			// Take the first value if multiple values exist
			if len(headerValues) > 0 {
				g.headers[headerName] = headerValues[0]
			}
		}
	}

	// Ensure all required headers are set, using defaults if missing
	for reqHeader, defaultValue := range requiredGatewayHeaders {
		found := false
		for k := range g.headers {
			if strings.EqualFold(k, reqHeader) {
				found = true
				break
			}
		}
		if !found {
			g.headers[reqHeader] = defaultValue
		}
	}
}

// GetHeaders returns a copy of all stored gateway headers
func (g *Gateway) GetHeaders() map[string]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.headers == nil {
		return nil
	}

	result := make(map[string]string, len(g.headers))
	for k, v := range g.headers {
		result[k] = v
	}
	return result
}

// GetHeader returns a specific gateway header value
func (g *Gateway) GetHeader(name string) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.headers == nil {
		return ""
	}
	if val, ok := g.headers[name]; ok {
		return val
	}
	if def, ok := requiredGatewayHeaders[name]; ok {
		return def
	}
	return ""
}

// Clear removes all stored gateway headers
func (g *Gateway) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.headers = nil
}
