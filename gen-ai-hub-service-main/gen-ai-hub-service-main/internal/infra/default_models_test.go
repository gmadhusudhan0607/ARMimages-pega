/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package infra

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/stretchr/testify/assert"
)

func TestGetDefaultModelsForContext(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse interface{}
		serverStatus   int
		serverError    bool
		envVarSet      bool
		expectError    bool
		expected       DefaultModelConfig
	}{
		{
			name: "successful response with Pro field",
			serverResponse: DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
			serverStatus: http.StatusOK,
			envVarSet:    true,
			expectError:  false,
			expected: DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "pro-model",
			},
		},
		{
			name: "successful response without Pro field",
			serverResponse: map[string]string{
				"fast":  "fast-model",
				"smart": "smart-model",
			},
			serverStatus: http.StatusOK,
			envVarSet:    true,
			expectError:  false,
			expected: DefaultModelConfig{
				Fast:  "fast-model",
				Smart: "smart-model",
				Pro:   "", // Pro will be empty when not present
			},
		},
		{
			name:         "non-200 response",
			serverStatus: http.StatusInternalServerError,
			envVarSet:    true,
			expectError:  true,
		},
		{
			name:        "missing environment variable",
			envVarSet:   false,
			expectError: true,
		},
		{
			name:        "server error",
			serverError: true,
			envVarSet:   true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment to restore later
			originalEnv := os.Getenv("MODELS_DEFAULTS_ENDPOINT")
			defer os.Setenv("MODELS_DEFAULTS_ENDPOINT", originalEnv)

			var server *httptest.Server
			if !tt.serverError {
				// Create test server
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.serverStatus)
					if tt.serverResponse != nil {
						var respBody []byte
						var err error
						if str, ok := tt.serverResponse.(string); ok {
							respBody = []byte(str)
						} else {
							respBody, err = json.Marshal(tt.serverResponse)
							assert.NoError(t, err)
						}
						_, err = w.Write(respBody)
						assert.NoError(t, err)
					}
				}))
				defer server.Close()

				if tt.envVarSet {
					os.Setenv("MODELS_DEFAULTS_ENDPOINT", server.URL)
				} else {
					os.Unsetenv("MODELS_DEFAULTS_ENDPOINT")
				}
			} else {
				// Create a non-existent URL to simulate server error
				os.Setenv("MODELS_DEFAULTS_ENDPOINT", "http://non-existent-server:12345")
			}

			// Create context with logger
			ctx := cntx.ServiceContext("test")

			// Call the function
			result, err := GetDefaultModelsForContext(ctx)

			// Verify expectations
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDefaultModelConfigToResponse(t *testing.T) {
	config := DefaultModelConfig{
		Fast:  "fast-model",
		Smart: "smart-model",
		Pro:   "pro-model",
	}

	t.Run("with Pro field enabled", func(t *testing.T) {
		response := config.ToResponse(true)

		// Check the typed response directly
		assert.Equal(t, "fast-model", response.Fast)
		assert.Equal(t, "smart-model", response.Smart)
		assert.Equal(t, "pro-model", response.Pro)

		// Verify JSON marshaling includes Pro
		jsonBytes, err := json.Marshal(response)
		assert.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(jsonBytes, &result)
		assert.NoError(t, err)

		assert.Equal(t, "fast-model", result["fast"])
		assert.Equal(t, "smart-model", result["smart"])
		assert.Equal(t, "pro-model", result["pro"])
	})

	t.Run("with Pro field disabled", func(t *testing.T) {
		response := config.ToResponse(false)

		// Check the typed response
		assert.Equal(t, "fast-model", response.Fast)
		assert.Equal(t, "smart-model", response.Smart)
		assert.Equal(t, "", response.Pro) // Pro should be empty string when disabled

		// Should return struct without Pro field in JSON (due to omitempty)
		jsonBytes, err := json.Marshal(response)
		assert.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(jsonBytes, &result)
		assert.NoError(t, err)

		// Check that only Fast and Smart are present
		assert.Equal(t, "fast-model", result["fast"])
		assert.Equal(t, "smart-model", result["smart"])
		_, hasProField := result["pro"]
		assert.False(t, hasProField, "Pro field should not be present when disabled")
	})

	t.Run("with empty Pro value and flag enabled", func(t *testing.T) {
		emptyConfig := DefaultModelConfig{
			Fast:  "fast-model",
			Smart: "smart-model",
			Pro:   "",
		}
		response := emptyConfig.ToResponse(true)

		// Check the typed response - Pro will be empty string
		assert.Equal(t, "fast-model", response.Fast)
		assert.Equal(t, "smart-model", response.Smart)
		assert.Equal(t, "", response.Pro)

		// Verify JSON marshaling omits empty Pro (due to omitempty)
		jsonBytes, err := json.Marshal(response)
		assert.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(jsonBytes, &result)
		assert.NoError(t, err)

		assert.Equal(t, "fast-model", result["fast"])
		assert.Equal(t, "smart-model", result["smart"])
		_, hasProField := result["pro"]
		assert.False(t, hasProField, "Pro field should not be present when empty (omitempty)")
	})
}
