/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package embedders

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstructModelForbiddenError(t *testing.T) {
	tests := []struct {
		name         string
		respBody     io.ReadCloser
		expectedText string
	}{
		{
			name:     "Error with nil response body",
			respBody: nil,
			expectedText: `Unable to call the Embedding Model. Received HTTP 403 (Access Denied) from LLM/GatewayService.
Verify your permissions and ensure your host/IP is correctly whitelisted to access the LLM Gateway. If the issue persists, contact your administrator`,
		},
		{
			name:     "Error with response body",
			respBody: io.NopCloser(strings.NewReader(`{"error": "access denied", "code": 403}`)),
			expectedText: `Unable to call the Embedding Model. Received HTTP 403 (Access Denied) from LLM/GatewayService.
Verify your permissions and ensure your host/IP is correctly whitelisted to access the LLM Gateway. If the issue persists, contact your administrator
Model response: {"error": "access denied", "code": 403}`,
		},
		{
			name:     "Error with empty response body",
			respBody: io.NopCloser(strings.NewReader("")),
			expectedText: `Unable to call the Embedding Model. Received HTTP 403 (Access Denied) from LLM/GatewayService.
Verify your permissions and ensure your host/IP is correctly whitelisted to access the LLM Gateway. If the issue persists, contact your administrator
Model response: `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ConstructModelForbiddenError(tt.respBody)
			assert.Error(t, err)
			assert.Equal(t, tt.expectedText, err.Error())
		})
	}
}

func TestConstructModelNotFoundError(t *testing.T) {
	tests := []struct {
		name         string
		respBody     io.ReadCloser
		expectedText string
	}{
		{
			name:     "Error with nil response body",
			respBody: nil,
			expectedText: `Unable to call the Embedding Model. Received HTTP 404 (Not Found) from LLM/GatewayService.
Ensure you have correct Gateway URL set and the model is available. If the issue persists, contact your administrator`,
		},
		{
			name:     "Error with response body",
			respBody: io.NopCloser(strings.NewReader(`{"error": "model not found", "code": 404}`)),
			expectedText: `Unable to call the Embedding Model. Received HTTP 404 (Not Found) from LLM/GatewayService.
Ensure you have correct Gateway URL set and the model is available. If the issue persists, contact your administrator
Model response: {"error": "model not found", "code": 404}`,
		},
		{
			name:     "Error with empty response body",
			respBody: io.NopCloser(strings.NewReader("")),
			expectedText: `Unable to call the Embedding Model. Received HTTP 404 (Not Found) from LLM/GatewayService.
Ensure you have correct Gateway URL set and the model is available. If the issue persists, contact your administrator
Model response: `,
		},
		{
			name:     "Error with JSON response body",
			respBody: io.NopCloser(strings.NewReader(`{"message": "The requested model was not found", "details": "Check model name and availability"}`)),
			expectedText: `Unable to call the Embedding Model. Received HTTP 404 (Not Found) from LLM/GatewayService.
Ensure you have correct Gateway URL set and the model is available. If the issue persists, contact your administrator
Model response: {"message": "The requested model was not found", "details": "Check model name and availability"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ConstructModelNotFoundError(tt.respBody)
			assert.Error(t, err)
			assert.Equal(t, tt.expectedText, err.Error())
		})
	}
}

// Test that the response body is properly closed
func TestErrorConstructors_BodyClosure(t *testing.T) {
	t.Run("ConstructModelForbiddenError closes body", func(t *testing.T) {
		body := &mockReadCloser{
			Reader: strings.NewReader("test response"),
			closed: false,
		}

		err := ConstructModelForbiddenError(body)
		assert.Error(t, err)
		assert.True(t, body.closed, "Response body should be closed")
		assert.Contains(t, err.Error(), "test response")
	})

	t.Run("ConstructModelNotFoundError closes body", func(t *testing.T) {
		body := &mockReadCloser{
			Reader: strings.NewReader("test response"),
			closed: false,
		}

		err := ConstructModelNotFoundError(body)
		assert.Error(t, err)
		assert.True(t, body.closed, "Response body should be closed")
		assert.Contains(t, err.Error(), "test response")
	})
}

// mockReadCloser is a helper to test that Close() is called
type mockReadCloser struct {
	io.Reader
	closed bool
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}
