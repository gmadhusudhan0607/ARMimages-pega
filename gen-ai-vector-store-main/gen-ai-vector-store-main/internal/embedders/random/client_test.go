/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package random

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/stretchr/testify/assert"
)

func TestRandomClient_GetEmbedding(t *testing.T) {
	type args struct {
		ctx   context.Context
		chunk string
	}
	tests := []struct {
		name           string
		args           args
		vectorLen      int
		expectedStatus int
		delay          string
	}{
		{
			name: "Successfully get random embedding",
			args: args{
				ctx:   context.Background(),
				chunk: "test chunk",
			},
			vectorLen:      1536,
			expectedStatus: 200,
		},
		{
			name: "Successfully get random embedding with custom vector length",
			args: args{
				ctx:   context.Background(),
				chunk: "test chunk",
			},
			vectorLen:      512,
			expectedStatus: 200,
		},
		{
			name: "Successfully get random embedding with delay",
			args: args{
				ctx:   context.Background(),
				chunk: "test chunk",
			},
			vectorLen:      1024,
			expectedStatus: 200,
			delay:          "0", // Set to 0 to avoid long test execution
		},
		{
			name: "Successfully get random embedding with float delay",
			args: args{
				ctx:   context.Background(),
				chunk: "test chunk",
			},
			vectorLen:      768,
			expectedStatus: 200,
			delay:          "0.1", // Float delay value
		},
		{
			name: "Successfully get random embedding with no delay (default)",
			args: args{
				ctx:   context.Background(),
				chunk: "test chunk",
			},
			vectorLen:      512,
			expectedStatus: 200,
			delay:          "", // No delay environment variable set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set delay environment variable if specified
			if tt.delay != "" {
				err := os.Setenv("RANDOM_EMBEDDER_DELAY", tt.delay)
				assert.NoError(t, err)
				defer func() {
					err := os.Unsetenv("RANDOM_EMBEDDER_DELAY")
					assert.NoError(t, err)
				}()
			}

			client, err := NewTestRandomClient("http://localhost:8080", tt.vectorLen, nil)
			assert.NoError(t, err)
			assert.NotNil(t, client)

			start := time.Now()
			got, status, err := client.GetEmbedding(tt.args.ctx, tt.args.chunk)
			duration := time.Since(start)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, status)
			assert.NotNil(t, got)
			assert.Equal(t, tt.vectorLen, len(got))

			// Verify all values are between -1 and 1
			for i, val := range got {
				assert.GreaterOrEqual(t, val, float32(-1.0), "Value at index %d should be >= -1", i)
				assert.LessOrEqual(t, val, float32(1.0), "Value at index %d should be <= 1", i)
			}

			// If delay was set to 0, execution should be fast
			if tt.delay == "0" {
				assert.Less(t, duration, 500*time.Millisecond, "Execution should be fast with no delay")
			}
		})
	}
}

func TestRandomClient_GetURL(t *testing.T) {
	expectedURL := "http://test-url:8080"
	client, err := NewTestRandomClient(expectedURL, 1536, nil)
	assert.NoError(t, err)

	actualURL := client.GetURL()
	assert.Equal(t, expectedURL, actualURL)
}

func TestNewRandomEmbedder(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		vectorLen   int
		httpHeaders map[string]string
		wantErr     bool
	}{
		{
			name:        "Successfully create random embedder",
			uri:         "http://localhost:8080",
			vectorLen:   1536,
			httpHeaders: nil,
			wantErr:     false,
		},
		{
			name:        "Successfully create random embedder with headers",
			uri:         "http://localhost:8080",
			vectorLen:   512,
			httpHeaders: map[string]string{"Authorization": "Bearer token"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRandomEmbedder(tt.uri, tt.vectorLen, tt.httpHeaders)
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

func TestNewRandomClient_BackwardCompatibility(t *testing.T) {
	// Test the deprecated NewRandomClient function for backward compatibility
	uri := "http://localhost:8080"
	vectorLen := 1536

	client, err := NewRandomClient(uri, vectorLen, nil)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, uri, client.GetURL())
}

func NewTestRandomClient(uri string, vectorLen int, httpHeaders map[string]string) (embedders.TextEmbedder, error) {
	return NewRandomEmbedder(uri, vectorLen, httpHeaders)
}
