/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package sax

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/go-sax"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestIsolationValidator_Integration(t *testing.T) {
	t.Setenv("ISOLATION_ID_VERIFICATION_DISABLED", "false")
	t.Setenv("SAX_DISABLED", "false")
	// Helper to create dummy token header
	makeHeader := func(claims map[string]interface{}) string {
		b, _ := json.Marshal(claims)
		return "Bearer h." + base64.RawURLEncoding.EncodeToString(b) + ".s"
	}

	tests := []struct {
		name             string
		claims           sax.Claims
		header           string
		pathParam        string
		expectedStatus   int
		expectAttributes []attribute.KeyValue
	}{
		{
			name:           "Authorized via GUID",
			pathParam:      "user-1",
			claims:         sax.Claims{GUID: "user-1"},
			expectedStatus: http.StatusOK,
			expectAttributes: []attribute.KeyValue{
				attrValidationGUID.String("user-1"),
				attrIsolationID.String("user-1"),
			},
		},
		{
			name:           "Authorized via IsolationID (Header)",
			pathParam:      "iso-1",
			claims:         sax.Claims{GUID: "mismatch"},
			header:         makeHeader(map[string]interface{}{"isolationId": "iso-1"}),
			expectedStatus: http.StatusOK,
			expectAttributes: []attribute.KeyValue{
				attrValidationIsolationID.String("iso-1"),
				attrIsolationID.String("iso-1"),
			},
		},
		{
			name:           "Forbidden: Mismatch Both",
			pathParam:      "target",
			claims:         sax.Claims{GUID: "guid-x"},
			header:         makeHeader(map[string]interface{}{"isolationId": "iso-y"}),
			expectedStatus: http.StatusForbidden, // 403
			expectAttributes: []attribute.KeyValue{
				attrIsolationID.String("target"),
			},
		},
		{
			name:           "Bad Request: Missing Path Param",
			pathParam:      "", // Gin treats this as root, but we simulate empty param extraction
			claims:         sax.Claims{GUID: "any"},
			expectedStatus: http.StatusBadRequest, // 400
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer, recorder := setupTestTracer(t)
			isoVal := NewIsolationValidator()
			isoVal.tracer = tracer

			r := gin.New()

			// Middleware Mock
			r.Use(func(c *gin.Context) {
				c.Set(contextKeyClaims, tt.claims)
				c.Next()
			})

			// Handler
			path := "/:isolationID"
			if tt.pathParam == "" {
				path = "/" // Simulate case where param is missing or routing matches root
			}
			r.GET(path, isoVal.Validate(), func(c *gin.Context) {
				c.Status(200)
			})

			// Request
			url := "/" + tt.pathParam
			req := httptest.NewRequest("GET", url, nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Assert Attributes
			if len(tt.expectAttributes) > 0 {
				spans := recorder.Ended()
				require.NotEmpty(t, spans)
				lastSpan := spans[len(spans)-1]
				for _, ea := range tt.expectAttributes {
					found := false
					for _, a := range lastSpan.Attributes() {
						if a.Key == ea.Key && a.Value == ea.Value {
							found = true
						}
					}
					assert.True(t, found, "Missing attribute: %v", ea)
				}
			}
		})
	}
}

func TestIsolationValidator_DisableSwitch(t *testing.T) {
	t.Setenv("ISOLATION_ID_VERIFICATION_DISABLED", "true")

	isoVal := NewIsolationValidator()
	assert.False(t, isoVal.enabled)

	r := gin.New()
	r.GET("/", isoVal.Validate(), func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	assert.Equal(t, 200, w.Code)
}
