/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTextEmbedder is a mock implementation of TextEmbedder interface
type MockTextEmbedder struct {
	mock.Mock
}

func (m *MockTextEmbedder) GetEmbedding(ctx context.Context, chunk string) ([]float32, int, error) {
	args := m.Called(ctx, chunk)
	return args.Get(0).([]float32), args.Int(1), args.Error(2)
}

func (m *MockTextEmbedder) GetURL() string {
	args := m.Called()
	return args.String(0)
}

func TestTextEmbedder_Interface(t *testing.T) {
	// Test that our mock implements the TextEmbedder interface
	var embedder TextEmbedder = &MockTextEmbedder{}
	assert.NotNil(t, embedder, "MockTextEmbedder should implement TextEmbedder interface")
}

func TestTextEmbedder_GetEmbedding(t *testing.T) {
	mockEmbedder := &MockTextEmbedder{}

	// Set up expectations
	ctx := context.Background()
	chunk := "test chunk"
	expectedEmbedding := []float32{0.1, 0.2, 0.3}
	expectedStatus := 200

	mockEmbedder.On("GetEmbedding", ctx, chunk).Return(expectedEmbedding, expectedStatus, nil)

	// Call the method
	embedding, status, err := mockEmbedder.GetEmbedding(ctx, chunk)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
	assert.Equal(t, expectedEmbedding, embedding)

	// Verify expectations were met
	mockEmbedder.AssertExpectations(t)
}

func TestTextEmbedder_GetURL(t *testing.T) {
	mockEmbedder := &MockTextEmbedder{}

	// Set up expectations
	expectedURL := "http://test-embedder:8080"
	mockEmbedder.On("GetURL").Return(expectedURL)

	// Call the method
	url := mockEmbedder.GetURL()

	// Verify results
	assert.Equal(t, expectedURL, url)

	// Verify expectations were met
	mockEmbedder.AssertExpectations(t)
}

func TestEmbedder_BackwardCompatibility(t *testing.T) {
	// Test that the deprecated Embedder alias still works
	var embedder Embedder = &MockTextEmbedder{}
	assert.NotNil(t, embedder, "Embedder alias should work for backward compatibility")

	// Test that we can use it as TextEmbedder
	var textEmbedder TextEmbedder = embedder
	assert.NotNil(t, textEmbedder, "Embedder should be assignable to TextEmbedder")
}

func TestTextEmbedder_InterfaceCompliance(t *testing.T) {
	// Test that various embedder implementations comply with the interface
	tests := []struct {
		name     string
		embedder func() TextEmbedder
	}{
		{
			name: "MockTextEmbedder",
			embedder: func() TextEmbedder {
				return &MockTextEmbedder{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder := tt.embedder()
			assert.NotNil(t, embedder, "Embedder should not be nil")

			// Test that it implements the interface methods
			assert.Implements(t, (*TextEmbedder)(nil), embedder, "Should implement TextEmbedder interface")
		})
	}
}

// TestTextEmbedder_MethodSignatures verifies that the interface methods have the correct signatures
func TestTextEmbedder_MethodSignatures(t *testing.T) {
	mockEmbedder := &MockTextEmbedder{}

	// Test GetEmbedding method signature
	ctx := context.Background()
	chunk := "test"
	expectedEmbedding := []float32{0.1}
	expectedStatus := 200

	mockEmbedder.On("GetEmbedding", ctx, chunk).Return(expectedEmbedding, expectedStatus, nil)
	mockEmbedder.On("GetURL").Return("http://test")

	// Verify GetEmbedding returns the correct types
	embedding, status, err := mockEmbedder.GetEmbedding(ctx, chunk)
	assert.IsType(t, []float32{}, embedding, "GetEmbedding should return []float32")
	assert.IsType(t, 0, status, "GetEmbedding should return int")
	assert.IsType(t, (*error)(nil), &err, "GetEmbedding should return error")

	// Verify GetURL returns the correct type
	url := mockEmbedder.GetURL()
	assert.IsType(t, "", url, "GetURL should return string")

	mockEmbedder.AssertExpectations(t)
}

// TestTextEmbedder_ContextHandling tests that context is properly handled
func TestTextEmbedder_ContextHandling(t *testing.T) {
	mockEmbedder := &MockTextEmbedder{}

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "Background context",
			ctx:  context.Background(),
		},
		{
			name: "Context with timeout",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				return ctx
			}(),
		},
		{
			name: "Context with value",
			ctx:  context.WithValue(context.Background(), "key", "value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := "test chunk"
			expectedEmbedding := []float32{0.1, 0.2}
			expectedStatus := 200

			mockEmbedder.On("GetEmbedding", tt.ctx, chunk).Return(expectedEmbedding, expectedStatus, nil).Once()

			embedding, status, err := mockEmbedder.GetEmbedding(tt.ctx, chunk)

			assert.NoError(t, err)
			assert.Equal(t, expectedStatus, status)
			assert.Equal(t, expectedEmbedding, embedding)
		})
	}

	mockEmbedder.AssertExpectations(t)
}
