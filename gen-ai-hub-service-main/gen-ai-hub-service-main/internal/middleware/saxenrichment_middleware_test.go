/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"encoding/base64"
	"errors"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/testutils"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient/saxtypes"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSaxRequestEnrichment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey := base64.StdEncoding.EncodeToString(testutils.GeneratePrivateKey())

	tests := []struct {
		name           string
		useSax         string
		pk             string
		mockCallFunc   func(url, method string, header http.Header, body io.ReadCloser) (*http.Request, *http.Response, error)
		expectedStatus int
		expectedHeader string
	}{
		{
			name:   "SAX token added successfully",
			useSax: "true",
			mockCallFunc: func(url, method string, header http.Header, body io.ReadCloser) (*http.Request, *http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"access_token": "test-token"}`)),
					Header:     header,
				}
				return nil, resp, nil
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "Bearer test-token",
		},
		{
			name:           "Cannot decode Private key",
			useSax:         "true",
			pk:             "bad",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "SAX token generation failed",
			useSax: "true",
			mockCallFunc: func(url, method string, header http.Header, body io.ReadCloser) (*http.Request, *http.Response, error) {
				return nil, nil, errors.New("token generation failed")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "SAX not used",
			useSax:         "false",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new context and set up the mock functions
			os.Setenv("USE_SAX", tt.useSax)
			helpers.HelperSuite.HttpCaller = tt.mockCallFunc
			ctx := cntx.ServiceContext("saxenrichment_middleware_test")

			pk := privateKey
			if tt.pk != "" {
				pk = tt.pk
			}
			ctx = cntx.ContextWithSaxClientConfig(ctx, &saxtypes.SaxAuthClientConfig{
				PrivateKey: pk,
			})

			// Create a new Gin engine and add the middleware
			r := gin.New()
			r.Use(SaxRequestEnrichment(ctx))
			r.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Create a new HTTP request and recorder
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Perform the request
			r.ServeHTTP(w, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check the Authorization header if expected
			if tt.expectedHeader != "" {
				assert.Equal(t, tt.expectedHeader, req.Header.Get("Authorization"))
			}

			os.Unsetenv("USE_SAX")
			helpers.HelperSuite.Reset()
		})
	}
}
