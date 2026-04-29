/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedders

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmbeddingProcessor is a mock implementation of EmbeddingProcessor
type MockEmbeddingProcessor struct {
	mock.Mock
}

func (m *MockEmbeddingProcessor) CreateRequest(ctx context.Context, chunk string) (*http.Request, error) {
	args := m.Called(ctx, chunk)
	return args.Get(0).(*http.Request), args.Error(1)
}

func (m *MockEmbeddingProcessor) ProcessResponse(resp *http.Response) ([]float32, error) {
	args := m.Called(resp)
	return args.Get(0).([]float32), args.Error(1)
}

func TestNewEmbedderBase(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("SAX_CLIENT_SECRET", "test-secret")
	t.Setenv("SAX_CLIENT_DISABLED", "true")

	tests := []struct {
		name         string
		uri          string
		httpHeaders  map[string]string
		modelName    string
		modelVersion string
		logPrefix    string
		wantErr      bool
	}{
		{
			name:         "Successfully create embedder base",
			uri:          "http://localhost:8080",
			httpHeaders:  map[string]string{"Authorization": "Bearer token"},
			modelName:    "test-model",
			modelVersion: "1.0",
			logPrefix:    "TEST",
			wantErr:      false,
		},
		{
			name:         "Successfully create embedder base with nil headers",
			uri:          "http://localhost:8080",
			httpHeaders:  nil,
			modelName:    "test-model",
			modelVersion: "1.0",
			logPrefix:    "TEST",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := http_client.GetDefaultHTTPClientConfig()
			logger := log.GetNamedLogger("genai-vector-store")
			got, err := NewEmbedderBase(tt.uri, tt.httpHeaders, cfg, tt.modelName, tt.modelVersion, tt.logPrefix, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.uri, got.GetURL())
			}
		})
	}
}

func TestEmbedderBase_GetURL(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("SAX_CLIENT_SECRET", "test-secret")
	t.Setenv("SAX_CLIENT_DISABLED", "true")

	expectedURL := "http://test-url:8080"
	cfg := http_client.GetDefaultHTTPClientConfig()

	logger := log.GetNamedLogger("genai-vector-store")

	embedder, err := NewEmbedderBase(expectedURL, nil, cfg, "test-model", "1.0", "TEST", logger)
	assert.NoError(t, err)

	actualURL := embedder.GetURL()
	assert.Equal(t, expectedURL, actualURL)
}

func TestEmbedderBase_CreateJSONRequest(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("SAX_CLIENT_SECRET", "test-secret")
	t.Setenv("SAX_CLIENT_DISABLED", "true")

	cfg := http_client.GetDefaultHTTPClientConfig()
	logger := log.GetNamedLogger("genai-vector-store")
	embedder, err := NewEmbedderBase("http://localhost:8080", nil, cfg, "test-model", "1.0", "TEST", logger)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		method   string
		body     interface{}
		wantErr  bool
		wantType string
	}{
		{
			name:     "Successfully create POST request with JSON body",
			method:   http.MethodPost,
			body:     map[string]string{"text": "test"},
			wantErr:  false,
			wantType: "application/json",
		},
		{
			name:     "Successfully create GET request with nil body",
			method:   http.MethodGet,
			body:     nil,
			wantErr:  false,
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req, err := embedder.CreateJSONRequest(ctx, tt.method, tt.body)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, req)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
				assert.Equal(t, tt.method, req.Method)
				assert.Equal(t, "http://localhost:8080", req.URL.String())

				if tt.wantType != "" {
					assert.Equal(t, tt.wantType, req.Header.Get("Content-Type"))
				}
			}
		})
	}
}

func TestEmbedderBase_UnmarshalJSONResponse(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("SAX_CLIENT_SECRET", "test-secret")
	t.Setenv("SAX_CLIENT_DISABLED", "true")

	cfg := http_client.GetDefaultHTTPClientConfig()
	logger := log.GetNamedLogger("genai-vector-store")
	embedder, err := NewEmbedderBase("http://localhost:8080", nil, cfg, "test-model", "1.0", "TEST", logger)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		contentType string
		body        string
		target      interface{}
		wantErr     bool
	}{
		{
			name:        "Successfully unmarshal JSON response",
			contentType: "application/json",
			body:        `{"message": "test"}`,
			target:      &map[string]string{},
			wantErr:     false,
		},
		{
			name:        "Fail with unsupported content type",
			contentType: "text/plain",
			body:        "test",
			target:      &map[string]string{},
			wantErr:     true,
		},
		{
			name:        "Fail with invalid JSON",
			contentType: "application/json",
			body:        `{"invalid": json}`,
			target:      &map[string]string{},
			wantErr:     true,
		},
		{
			name:        "Successfully handle nil target",
			contentType: "application/json",
			body:        `{"message": "test"}`,
			target:      nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: make(http.Header),
				Body:   io.NopCloser(strings.NewReader(tt.body)),
			}
			resp.Header.Set("Content-Type", tt.contentType)

			err := embedder.UnmarshalJSONResponse(resp, tt.target)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmbedderBase_ValidateEmbeddingResponse(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("SAX_CLIENT_SECRET", "test-secret")
	t.Setenv("SAX_CLIENT_DISABLED", "true")

	cfg := http_client.GetDefaultHTTPClientConfig()
	logger := log.GetNamedLogger("genai-vector-store")
	embedder, err := NewEmbedderBase("http://localhost:8080", nil, cfg, "test-model", "1.0", "TEST", logger)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		embeddings [][]float32
		statusCode int
		wantErr    bool
	}{
		{
			name:       "Successfully validate single embedding",
			embeddings: [][]float32{{0.1, 0.2, 0.3}},
			statusCode: 200,
			wantErr:    false,
		},
		{
			name:       "Fail with no embeddings",
			embeddings: [][]float32{},
			statusCode: 200,
			wantErr:    true,
		},
		{
			name:       "Fail with multiple embeddings",
			embeddings: [][]float32{{0.1, 0.2}, {0.3, 0.4}},
			statusCode: 200,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := embedder.ValidateEmbeddingResponse(tt.embeddings, tt.statusCode)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSimpleEmbeddingProcessor(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("SAX_CLIENT_SECRET", "test-secret")
	t.Setenv("SAX_CLIENT_DISABLED", "true")

	cfg := http_client.GetDefaultHTTPClientConfig()
	logger := log.GetNamedLogger("genai-vector-store")
	embedder, err := NewEmbedderBase("http://localhost:8080", nil, cfg, "test-model", "1.0", "TEST", logger)
	assert.NoError(t, err)

	createBody := func(chunk string) interface{} {
		return map[string]string{"text": chunk}
	}

	extractEmbedding := func(resp interface{}) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	processor := NewSimpleEmbeddingProcessor(embedder, createBody, extractEmbedding)
	assert.NotNil(t, processor)

	t.Run("CreateRequest", func(t *testing.T) {
		ctx := context.Background()
		req, err := processor.CreateRequest(ctx, "test chunk")
		assert.NoError(t, err)
		assert.NotNil(t, req)
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	})

	t.Run("ProcessResponse", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
			Body:   io.NopCloser(strings.NewReader(`{"data": "test"}`)),
		}
		resp.Header.Set("Content-Type", "application/json")

		embedding, err := processor.ProcessResponse(resp)
		assert.NoError(t, err)
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, embedding)
	})
}

func TestEmbedderBase_GetEmbeddingWithProcessor_Success(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("SAX_CLIENT_SECRET", "test-secret")
	t.Setenv("SAX_CLIENT_DISABLED", "true")

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"embedding": [0.1, 0.2, 0.3]}`)
	}))
	defer server.Close()

	cfg := http_client.GetDefaultHTTPClientConfig()
	logger := log.GetNamedLogger("genai-vector-store")
	embedder, err := NewEmbedderBase(server.URL, nil, cfg, "test-model", "1.0", "TEST", logger)
	assert.NoError(t, err)

	// Create a mock processor
	mockProcessor := &MockEmbeddingProcessor{}

	// Set up expectations
	req, _ := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader([]byte(`{"text":"test"}`)))
	req.Header.Set("Content-Type", "application/json")

	mockProcessor.On("CreateRequest", mock.Anything, "test chunk").Return(req, nil)
	mockProcessor.On("ProcessResponse", mock.Anything).Return([]float32{0.1, 0.2, 0.3}, nil)

	ctx := context.Background()
	embedding, status, err := embedder.GetEmbeddingWithProcessor(ctx, "test chunk", mockProcessor)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, embedding)

	mockProcessor.AssertExpectations(t)
}

func TestEmbedderBase_GetEmbeddingWithProcessor_HTTPErrors(t *testing.T) {
	// Set required environment variables for testing
	t.Setenv("SAX_CLIENT_SECRET", "test-secret")
	t.Setenv("SAX_CLIENT_DISABLED", "true")

	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Handle 403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"error": "forbidden"}`,
			expectedStatus: http.StatusForbidden,
			expectError:    true,
		},
		{
			name:           "Handle 404 Not Found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "not found"}`,
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "Handle 500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "internal error"}`,
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			cfg := http_client.GetDefaultHTTPClientConfig()
			logger := log.GetNamedLogger("genai-vector-store")
			embedder, err := NewEmbedderBase(server.URL, nil, cfg, "test-model", "1.0", "TEST", logger)
			assert.NoError(t, err)

			// Create a mock processor
			mockProcessor := &MockEmbeddingProcessor{}

			// Set up expectations
			req, _ := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader([]byte(`{"text":"test"}`)))
			req.Header.Set("Content-Type", "application/json")

			mockProcessor.On("CreateRequest", mock.Anything, "test chunk").Return(req, nil)

			ctx := context.Background()
			embedding, status, err := embedder.GetEmbeddingWithProcessor(ctx, "test chunk", mockProcessor)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedStatus, status)
				assert.Nil(t, embedding)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
				assert.NotNil(t, embedding)
			}

			mockProcessor.AssertExpectations(t)
		})
	}
}
