/*
* Copyright (c) 2024 Pegasystems Inc.
* All rights reserved.
 */

package main

import (
	"bytes"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"

	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNonServiceEndpoints(t *testing.T) {
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
			uri:    "/swagger/ops.yaml",
			code:   http.StatusOK,
		},
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

// func TestIsolationEndpoints(t *testing.T) {
// 	tests := []struct {
// 		name             string
// 		method           string
// 		uri              string
// 		code             int
// 		errMsgText       string
// 		req              string
// 	}{
// 		{
// 			name:   "GET isolation with valid isolationID",
// 			method: http.MethodGet,
// 			uri:    "/v1/isolations/testIsolationID",
// 			code:   http.StatusOK,
// 		},
// 		{
// 			name:   "GET isolation with missing isolationID",
// 			method: http.MethodGet,
// 			uri:    "/v1/isolations/",
// 			code:   http.StatusBadRequest,
// 			errMsgText: "isolationID param is required",
// 		},
// 		{
// 			name:   "DELETE isolation with success",
// 			method: http.MethodDelete,
// 			uri:    "/v1/isolations/testIsolationID",
// 			code:   http.StatusOK,
// 			errMsgText:    "",
// 			req: "",
// 		},
// 		{
// 			name:   "DELETE isolation with missing isolationID",
// 			method: http.MethodDelete,
// 			uri:    "/v1/isolations/",
// 			code:   http.StatusBadRequest,
// 			errMsgText:    "isolationID param is required",
// 			req: "",
// 		},
// 		{
// 			name:   "DELETE isolation with wrong isolationID",
// 			method: http.MethodDelete,
// 			uri:    "/v1/isolations/badIsolationID",
// 			code:   http.StatusInternalServerError,
// 			errMsgText:    "isolation badIsolationID does not exist",
// 			req: "",
// 		},
// 		{
// 			name:   "DELETE isolation with error",
// 			method: http.MethodDelete,
// 			uri:    "/v1/isolations/testIsolationID",
// 			code:   http.StatusBadRequest,
// 			errMsgText:    "isolation testIsolationID does not exist",
// 			req: "",
// 		},
// 		{
// 			name:   "Create isolation",
// 			method: http.MethodPost,
// 			uri:    "/v1/isolations",
// 			code:   http.StatusOK,
// 			errMsgText:    "",
// 			req: "Details{ID: "testIsolationID", MaxStorageSize: 100, CreatedAt: someTime, ModifiedAt: someTime}",
// 		},
// 		{
// 			name:   "Create isolation",
// 			method: http.MethodPost,
// 			uri:    "/v1/isolations",
// 			code:   http.StatusBadRequest,
// 			errMsgText:    "",
// 			req: "Details{ID: "", MaxStorageSize: 0, CreatedAt: someTime, ModifiedAt: someTime}",
// 		},
// 	}

// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.URL.Query().Get("api-version") == "invalid" {
// 			w.WriteHeader(http.StatusNotFound)
// 		}
// 		fmt.Fprintln(w, "Response")
// 	}))
// 	defer ts.Close()
// 	var someTime = time.Date(2014, 07, 14, 23, 25, 0, 0, time.UTC)
// 	mockIso := new(mocks.MockIsolation)
// 	mockIso.On("GetIsolationDetails", mock.Anything).Return(isolation.Details{ID: "testIsolationID", MaxStorageSize: 100, CreatedAt: someTime, ModifiedAt: someTime})
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			recorder := httptest.NewRecorder()
// 			_, eh := gin.CreateTestContext(recorder)
// 			c, es := gin.CreateTestContext(recorder)
// 			mockDB := dbmock.NewMockDb()
// 			es.Use(dbMiddleware(mockDB))
// 			setupEngine(eh, es)

// 			req, err := http.NewRequest(tt.method, tt.uri, bytes.NewBuffer([]byte(tt.req)))
// 			assert.NoError(t, err)
// 			es.ServeHTTP(recorder, req)
// 			c.Request = req

// 			assert.Equal(t, tt.code, recorder.Result().StatusCode)

// 			if tt.code != http.StatusOK {
// 				responseBody, _ := io.ReadAll(recorder.Result().Body)
// 				assert.Contains(t, string(responseBody), tt.errMsgText)
// 			}
// 		})
// 	}
// }
