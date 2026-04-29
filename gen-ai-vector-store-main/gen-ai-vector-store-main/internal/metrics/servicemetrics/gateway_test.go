/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package servicemetrics

import (
	"net/http"
	"testing"
)

func TestGateway_SetHeadersFromResponse(t *testing.T) {
	tests := []struct {
		name            string
		responseHeaders map[string][]string
		expectedHeaders map[string]string
	}{
		{
			name: "extracts all X-Genai headers",
			responseHeaders: map[string][]string{
				"X-Genai-Gateway-Response-Time-Ms":          {"150"},
				"X-Genai-Gateway-Input-Tokens":              {"100"},
				"X-Genai-Gateway-Model-Id":                  {"gpt-4"},
				"X-Genai-Gateway-Region":                    {"us-east-1"},
				"X-Genai-Gateway-Output-Tokens":             {"200"},
				"X-Genai-Gateway-Tokens-Per-Second":         {"50"},
				"X-Genai-Gateway-Retry-Count":               {"1"},
				"X-Genai-Vectorstore-Embedding-Retry-Count": {"0"},
				"Content-Type":                              {"application/json"}, // Should be ignored
				"X-Other-Header":                            {"value"},            // Should be ignored
			},
			expectedHeaders: map[string]string{
				"X-Genai-Gateway-Response-Time-Ms":          "150",
				"X-Genai-Gateway-Input-Tokens":              "100",
				"X-Genai-Gateway-Model-Id":                  "gpt-4",
				"X-Genai-Gateway-Region":                    "us-east-1",
				"X-Genai-Gateway-Output-Tokens":             "200",
				"X-Genai-Gateway-Tokens-Per-Second":         "50",
				"X-Genai-Gateway-Retry-Count":               "1",
				"X-Genai-Vectorstore-Embedding-Retry-Count": "0",
			},
		},
		{
			name: "handles case insensitive headers",
			responseHeaders: map[string][]string{
				"x-genai-gateway-response-time-ms":          {"250"},
				"X-GENAI-GATEWAY-INPUT-TOKENS":              {"300"},
				"X-Genai-Gateway-Model-Id":                  {"gpt-4"},
				"X-Genai-Gateway-Region":                    {"us-east-1"},
				"X-Genai-Gateway-Output-Tokens":             {"200"},
				"X-Genai-Gateway-Tokens-Per-Second":         {"50"},
				"X-Genai-Gateway-Retry-Count":               {"1"},
				"X-Genai-Vectorstore-Embedding-Retry-Count": {"0"},
			},
			expectedHeaders: map[string]string{
				"x-genai-gateway-response-time-ms":          "250",
				"X-GENAI-GATEWAY-INPUT-TOKENS":              "300",
				"X-Genai-Gateway-Model-Id":                  "gpt-4",
				"X-Genai-Gateway-Region":                    "us-east-1",
				"X-Genai-Gateway-Output-Tokens":             "200",
				"X-Genai-Gateway-Tokens-Per-Second":         "50",
				"X-Genai-Gateway-Retry-Count":               "1",
				"X-Genai-Vectorstore-Embedding-Retry-Count": "0",
			},
		},
		{
			name: "handles multiple values - takes first",
			responseHeaders: map[string][]string{
				"X-Genai-Gateway-Model-Id": {"gpt-4", "gpt-3.5"},
			},
			expectedHeaders: map[string]string{
				"X-Genai-Gateway-Model-Id":          "gpt-4",
				"X-Genai-Gateway-Response-Time-Ms":  "-1",
				"X-Genai-Gateway-Input-Tokens":      "-1",
				"X-Genai-Gateway-Region":            "not-set",
				"X-Genai-Gateway-Output-Tokens":     "-1",
				"X-Genai-Gateway-Tokens-Per-Second": "-1",
				"X-Genai-Gateway-Retry-Count":       "-1",
			},
		},
		{
			name:            "handles empty headers",
			responseHeaders: map[string][]string{},
			expectedHeaders: map[string]string{
				"X-Genai-Gateway-Model-Id":          "not-set",
				"X-Genai-Gateway-Response-Time-Ms":  "-1",
				"X-Genai-Gateway-Input-Tokens":      "-1",
				"X-Genai-Gateway-Region":            "not-set",
				"X-Genai-Gateway-Output-Tokens":     "-1",
				"X-Genai-Gateway-Tokens-Per-Second": "-1",
				"X-Genai-Gateway-Retry-Count":       "-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := &Gateway{}

			// Create mock response
			resp := &http.Response{
				Header: tt.responseHeaders,
			}

			// Extract headers
			gateway.SetGenaiHeadersFromResponse(resp)

			// Get stored headers
			storedHeaders := gateway.GetHeaders()

			// Verify results
			if tt.expectedHeaders == nil {
				if storedHeaders != nil {
					t.Errorf("Expected nil headers, got %v", storedHeaders)
				}
				return
			}

			if len(storedHeaders) != len(tt.expectedHeaders) {
				t.Errorf("Expected %d headers, got %d, %s %s %s %s", len(tt.expectedHeaders), len(storedHeaders), "expectedHeaders", tt.expectedHeaders, "storedHeaders", storedHeaders)
			}

			for expectedKey, expectedValue := range tt.expectedHeaders {
				if storedValue, exists := storedHeaders[expectedKey]; !exists {
					t.Errorf("Expected header %s not found", expectedKey)
				} else if storedValue != expectedValue {
					t.Errorf("Expected header %s=%s, got %s", expectedKey, expectedValue, storedValue)
				}
			}
		})
	}
}

func TestGateway_GetHeader(t *testing.T) {
	gateway := &Gateway{}

	// Set up test response
	resp := &http.Response{
		Header: map[string][]string{
			"X-Genai-Gateway-Model-Id": {"gpt-4"},
			"X-Genai-Gateway-Region":   {"us-west-2"},
		},
	}

	gateway.SetGenaiHeadersFromResponse(resp)

	// Test getting existing header
	value := gateway.GetHeader("X-Genai-Gateway-Model-Id")
	if value != "gpt-4" {
		t.Errorf("Expected 'gpt-4', got '%s'", value)
	}

	// Test getting non-existing header
	value = gateway.GetHeader("X-Genai-Gateway-NonExistent")
	if value != "" {
		t.Errorf("Expected empty string, got '%s'", value)
	}
}

func TestGateway_Clear(t *testing.T) {
	gateway := &Gateway{}

	// Set up test response
	resp := &http.Response{
		Header: map[string][]string{
			"X-Genai-Gateway-Model-Id": {"gpt-4"},
		},
	}

	gateway.SetGenaiHeadersFromResponse(resp)

	// Verify header exists
	headers := gateway.GetHeaders()
	if len(headers) == 0 {
		t.Error("Expected headers to be present before clear")
	}

	// Clear headers
	gateway.Clear()

	// Verify headers are cleared
	headers = gateway.GetHeaders()
	if headers != nil {
		t.Errorf("Expected nil headers after clear, got %v", headers)
	}
}
