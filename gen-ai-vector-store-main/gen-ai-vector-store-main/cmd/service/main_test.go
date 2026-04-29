/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestEndpoints(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		uri              string
		code             int
		errMsgText       string
		isHealthEndpoint bool
		req              string
	}{
		{
			name:             "GET health readiness",
			method:           http.MethodGet,
			uri:              "/health/readiness",
			code:             http.StatusOK,
			isHealthEndpoint: true,
		},
		{
			name:             "GET health liveness",
			method:           http.MethodGet,
			uri:              "/health/liveness",
			code:             http.StatusOK,
			errMsgText:       "",
			isHealthEndpoint: true,
		},
		{
			name:   "GET swagger",
			method: http.MethodGet,
			uri:    "/",
			code:   http.StatusOK,
		},
		//{
		//	name:   "PUT documents with bad request",
		//	method: http.MethodPut,
		//	uri:    "/v1/some-isolation-id/collections/some-collection-name/documents",
		//	code:   http.StatusBadRequest,
		//	req:    "unformatted request",
		//},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api-version") == "invalid" {
			w.WriteHeader(http.StatusNotFound)
		}
		fmt.Fprintln(w, "Response")
	}))
	defer ts.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			_, eh := gin.CreateTestContext(recorder)
			_, es := gin.CreateTestContext(recorder)
			es.Use(
				middleware.DatabaseHandler(middleware.DatabasesConfig{}),
				middleware.ErrorHandler(),
			)
			setupEngine(eh, es)

			req, err := http.NewRequest(tt.method, tt.uri, bytes.NewBuffer([]byte(tt.req)))
			assert.NoError(t, err)

			if !tt.isHealthEndpoint {
				es.ServeHTTP(recorder, req)
			} else {
				eh.ServeHTTP(recorder, req)
			}

			assert.Equal(t, tt.code, recorder.Result().StatusCode)

			if tt.code != http.StatusOK {
				responseBody, _ := io.ReadAll(recorder.Result().Body)
				assert.Contains(t, string(responseBody), tt.errMsgText)
			}
		})
	}
}
