/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package client

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDoConverseForIntegrationTest_HappyPath(t *testing.T) {
	// Setup
	ctx := cntx.ServiceContext("integration-test-client-test")
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

	modelCall := &ConverseModelInference{
		Ctx: ctx,
		InfraModel: infra.ModelConfig{
			Endpoint: "http://localhost:8080",
			Path:     "/api/converse",
		},
		RawInput:   []byte(`{"message":"test"}`),
		GinContext: ginCtx,
	}

	// Mock AWS provider
	mockProvider := new(MockAwsProvider)

	// Setup expected response
	expectedBody := `{"response":"success"}`
	responseHeaders := http.Header{}
	responseHeaders.Set("Content-Type", "application/json")
	responseHeaders.Set("X-Custom-Header", "custom-value")

	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(expectedBody))),
		Header:     responseHeaders,
	}

	mockProvider.On("GetAwsClient", "http://localhost:8080/api/converse").Return(mockProvider)
	mockProvider.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Method == http.MethodPost &&
			req.URL.String() == "http://localhost:8080/api/converse" &&
			req.Header.Get("Content-Type") == "application/json"
	})).Return(mockResponse, nil)

	// Execute
	err := doConverseForIntegrationTest(modelCall, mockProvider)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, expectedBody, recorder.Body.String())
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "custom-value", recorder.Header().Get("X-Custom-Header"))
	mockProvider.AssertExpectations(t)
}

func TestDoConverseForIntegrationTest_WithHeaders(t *testing.T) {
	// Setup
	ctx := cntx.ServiceContext("integration-test-client-test")
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)

	// Set request headers
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
	ginCtx.Request.Header.Set("X-Genai-Gateway-Isolation-Id", "isolation-123")
	ginCtx.Request.Header.Set("Authorization", "Bearer token123")

	modelCall := &ConverseModelInference{
		Ctx: ctx,
		InfraModel: infra.ModelConfig{
			Endpoint: "http://localhost:8080",
			Path:     "/api/converse",
		},
		RawInput:   []byte(`{"message":"test"}`),
		GinContext: ginCtx,
	}

	// Mock AWS provider
	mockProvider := new(MockAwsProvider)

	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		Header:     http.Header{},
	}

	mockProvider.On("GetAwsClient", "http://localhost:8080/api/converse").Return(mockProvider)
	mockProvider.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Header.Get("X-Genai-Gateway-Isolation-ID") == "isolation-123" &&
			req.Header.Get("Authorization") == "Bearer token123" &&
			req.Header.Get("Content-Type") == "application/json"
	})).Return(mockResponse, nil)

	// Execute
	err := doConverseForIntegrationTest(modelCall, mockProvider)

	// Assert
	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

func TestDoConverseForIntegrationTest_WithoutOptionalHeaders(t *testing.T) {
	// Setup
	ctx := cntx.ServiceContext("integration-test-client-test")
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

	modelCall := &ConverseModelInference{
		Ctx: ctx,
		InfraModel: infra.ModelConfig{
			Endpoint: "http://localhost:8080",
			Path:     "/api/converse",
		},
		RawInput:   []byte(`{"message":"test"}`),
		GinContext: ginCtx,
	}

	// Mock AWS provider
	mockProvider := new(MockAwsProvider)

	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		Header:     http.Header{},
	}

	mockProvider.On("GetAwsClient", "http://localhost:8080/api/converse").Return(mockProvider)
	mockProvider.On("Do", mock.MatchedBy(func(req *http.Request) bool {
		return req.Header.Get("Content-Type") == "application/json" &&
			req.Header.Get("X-Genai-Gateway-Isolation-ID") == "" &&
			req.Header.Get("Authorization") == ""
	})).Return(mockResponse, nil)

	// Execute
	err := doConverseForIntegrationTest(modelCall, mockProvider)

	// Assert
	assert.NoError(t, err)
	mockProvider.AssertExpectations(t)
}

func TestDoConverseForIntegrationTest_ErrorExecutingRequest(t *testing.T) {
	// Setup
	ctx := cntx.ServiceContext("integration-test-client-test")
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

	modelCall := &ConverseModelInference{
		Ctx: ctx,
		InfraModel: infra.ModelConfig{
			Endpoint: "http://localhost:8080",
			Path:     "/api/converse",
		},
		RawInput:   []byte(`{"message":"test"}`),
		GinContext: ginCtx,
	}

	// Mock AWS provider with error
	mockProvider := new(MockAwsProvider)
	expectedError := errors.New("connection refused")

	mockProvider.On("GetAwsClient", "http://localhost:8080/api/converse").Return(mockProvider)
	mockProvider.On("Do", mock.Anything).Return(nil, expectedError)

	// Execute
	err := doConverseForIntegrationTest(modelCall, mockProvider)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute request")
	assert.Contains(t, err.Error(), "connection refused")
	mockProvider.AssertExpectations(t)
}

func TestDoConverseForIntegrationTest_DifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedStatus int
	}{
		{
			name:           "Success_200",
			statusCode:     200,
			responseBody:   `{"status":"ok"}`,
			expectedStatus: 200,
		},
		{
			name:           "BadRequest_400",
			statusCode:     400,
			responseBody:   `{"error":"bad request"}`,
			expectedStatus: 400,
		},
		{
			name:           "Unauthorized_401",
			statusCode:     401,
			responseBody:   `{"error":"unauthorized"}`,
			expectedStatus: 401,
		},
		{
			name:           "InternalError_500",
			statusCode:     500,
			responseBody:   `{"error":"internal error"}`,
			expectedStatus: 500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			ctx := cntx.ServiceContext("integration-test-client-test")
			recorder := httptest.NewRecorder()
			ginCtx, _ := gin.CreateTestContext(recorder)
			ginCtx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

			modelCall := &ConverseModelInference{
				Ctx: ctx,
				InfraModel: infra.ModelConfig{
					Endpoint: "http://localhost:8080",
					Path:     "/api/converse",
				},
				RawInput:   []byte(`{"message":"test"}`),
				GinContext: ginCtx,
			}

			// Mock AWS provider
			mockProvider := new(MockAwsProvider)

			mockResponse := &http.Response{
				StatusCode: tc.statusCode,
				Body:       io.NopCloser(bytes.NewReader([]byte(tc.responseBody))),
				Header:     http.Header{},
			}

			mockProvider.On("GetAwsClient", "http://localhost:8080/api/converse").Return(mockProvider)
			mockProvider.On("Do", mock.Anything).Return(mockResponse, nil)

			// Execute
			err := doConverseForIntegrationTest(modelCall, mockProvider)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, recorder.Code)
			assert.Equal(t, tc.responseBody, recorder.Body.String())
			mockProvider.AssertExpectations(t)
		})
	}
}

func TestDoConverseForIntegrationTest_MultipleResponseHeaders(t *testing.T) {
	// Setup
	ctx := cntx.ServiceContext("integration-test-client-test")
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

	modelCall := &ConverseModelInference{
		Ctx: ctx,
		InfraModel: infra.ModelConfig{
			Endpoint: "http://localhost:8080",
			Path:     "/api/converse",
		},
		RawInput:   []byte(`{"message":"test"}`),
		GinContext: ginCtx,
	}

	// Mock AWS provider with multiple headers
	mockProvider := new(MockAwsProvider)

	responseHeaders := http.Header{}
	responseHeaders.Set("Content-Type", "application/json")
	responseHeaders.Add("X-Custom-Header", "value1")
	responseHeaders.Add("X-Custom-Header", "value2")
	responseHeaders.Set("X-Request-Id", "req-123")

	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
		Header:     responseHeaders,
	}

	mockProvider.On("GetAwsClient", "http://localhost:8080/api/converse").Return(mockProvider)
	mockProvider.On("Do", mock.Anything).Return(mockResponse, nil)

	// Execute
	err := doConverseForIntegrationTest(modelCall, mockProvider)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "req-123", recorder.Header().Get("X-Request-Id"))
	// Note: gin.Context.Header() uses Set() internally, so only the last value is kept for duplicate header names
	assert.Equal(t, "value2", recorder.Header().Get("X-Custom-Header"))
	mockProvider.AssertExpectations(t)
}

func TestDoConverseForIntegrationTest_EmptyResponseBody(t *testing.T) {
	// Setup
	ctx := cntx.ServiceContext("integration-test-client-test")
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

	modelCall := &ConverseModelInference{
		Ctx: ctx,
		InfraModel: infra.ModelConfig{
			Endpoint: "http://localhost:8080",
			Path:     "/api/converse",
		},
		RawInput:   []byte(`{"message":"test"}`),
		GinContext: ginCtx,
	}

	// Mock AWS provider with empty response
	mockProvider := new(MockAwsProvider)

	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader([]byte(""))),
		Header:     http.Header{},
	}

	mockProvider.On("GetAwsClient", "http://localhost:8080/api/converse").Return(mockProvider)
	mockProvider.On("Do", mock.Anything).Return(mockResponse, nil)

	// Execute
	err := doConverseForIntegrationTest(modelCall, mockProvider)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, recorder.Code)
	assert.Empty(t, recorder.Body.String())
	mockProvider.AssertExpectations(t)
}

func TestDoConverseForIntegrationTest_LargePayload(t *testing.T) {
	// Setup
	ctx := cntx.ServiceContext("integration-test-client-test")
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodPost, "/test", nil)

	// Create large payload (1MB)
	largePayload := bytes.Repeat([]byte("x"), 1024*1024)

	modelCall := &ConverseModelInference{
		Ctx: ctx,
		InfraModel: infra.ModelConfig{
			Endpoint: "http://localhost:8080",
			Path:     "/api/converse",
		},
		RawInput:   largePayload,
		GinContext: ginCtx,
	}

	// Mock AWS provider
	mockProvider := new(MockAwsProvider)

	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(largePayload)),
		Header:     http.Header{},
	}

	mockProvider.On("GetAwsClient", "http://localhost:8080/api/converse").Return(mockProvider)
	mockProvider.On("Do", mock.Anything).Return(mockResponse, nil)

	// Execute
	err := doConverseForIntegrationTest(modelCall, mockProvider)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, len(largePayload), recorder.Body.Len())
	mockProvider.AssertExpectations(t)
}
