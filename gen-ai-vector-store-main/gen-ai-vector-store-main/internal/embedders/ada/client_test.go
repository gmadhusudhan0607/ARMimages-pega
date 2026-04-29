/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package ada

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/stretchr/testify/assert"
)

//go:embed test-data/chunk.txt
var testChunk string

//go:embed test-data/forbidden.json
var forbiddenResponse string

//go:embed test-data/post-embedding-response.json
var testPostEmbeddingResponse string

func TestAdaClient_GetEmbedding(t *testing.T) {
	type args struct {
		ctx   context.Context
		chunk string
	}
	tests := []struct {
		name        string
		args        args
		response    http.Response
		expected    []float32
		expectedErr error
	}{
		{
			name: "Successfully get embedding",
			args: args{
				ctx:   context.Background(),
				chunk: testChunk,
			},
			response: http.Response{
				StatusCode: http.StatusAccepted,
				Body:       BodyFromString(testPostEmbeddingResponse),
			},
			expected:    testEmbedding,
			expectedErr: nil,
		},
		{
			name: "403 response on get embedding",
			args: args{
				ctx:   context.Background(),
				chunk: testChunk,
			},
			response: http.Response{
				StatusCode: http.StatusForbidden,
				Body:       BodyFromString(forbiddenResponse),
			},
			expected:    nil,
			expectedErr: embedders.ConstructModelForbiddenError(io.NopCloser(strings.NewReader(forbiddenResponse))),
		},
		{
			name: "404 response on get embedding",
			args: args{
				ctx:   context.Background(),
				chunk: testChunk,
			},
			response: http.Response{
				StatusCode: http.StatusNotFound,
			},
			expected:    nil,
			expectedErr: embedders.ConstructModelNotFoundError(nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SAX_CLIENT_DISABLED", "true")
			defer os.Unsetenv("SAX_CLIENT_DISABLED")

			srv := MockAPI(t, "POST", tt.response)
			a, err := NewTestAdaClient(srv)
			assert.NoError(t, err)

			got, _, err := a.GetEmbedding(tt.args.ctx, tt.args.chunk)
			if tt.expectedErr == nil {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.expected, got)
			} else {
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
			}
		})
	}
}

func NewTestAdaClient(srv *httptest.Server) (embedders.Embedder, error) {
	a, err := NewAdaEmbedder(srv.URL, nil, http_client.GetDefaultHTTPClientConfig(), log.GetNamedLogger("genai-vector-store"))
	if err != nil {
		return nil, fmt.Errorf("failed to init ADA client: %w", err)
	}
	// Note: We can't easily mock the HTTP client in the new architecture
	// The test server will handle the HTTP requests directly
	return a, nil
}

func MockAPI(t *testing.T, method string, response http.Response) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, method)
		contentType := response.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json"
		}
		rw.Header().Add("Content-Type", contentType)
		rw.WriteHeader(response.StatusCode)
		if response.Body != nil {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			rw.Write(body)
		}
	}))
	return srv
}

func BodyFromString(content string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(content))
}
