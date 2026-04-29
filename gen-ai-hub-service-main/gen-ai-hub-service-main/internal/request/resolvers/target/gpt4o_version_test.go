/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package target

import (
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
)

func TestFindGPT4oVersion(t *testing.T) {
	tests := []struct {
		name     string
		models   []api.AzureModelResponse
		expected string
	}{
		{
			name: "finds gpt-4o model when multiple models present",
			models: []api.AzureModelResponse{
				{
					DeploymentID: "gpt-4o",
					ModelName:    "gpt-4o",
					ModelVersion: "2024-05-13",
				},
				{
					DeploymentID: "gpt-4o-08",
					ModelName:    "gpt-4o",
					ModelVersion: "2024-08-06",
				},
			},
			expected: "2024-05-13",
		},
		{
			name: "finds correct gpt-4o when both have same model-name but different deployment-id",
			models: []api.AzureModelResponse{
				{
					DeploymentID: "gpt-4o-08",
					ModelName:    "gpt-4o",
					ModelVersion: "2024-08-06",
				},
				{
					DeploymentID: "gpt-4o",
					ModelName:    "gpt-4o",
					ModelVersion: "2024-05-13",
				},
			},
			expected: "2024-05-13",
		},
		{
			name: "no matching model when only deployment-id matches",
			models: []api.AzureModelResponse{
				{
					DeploymentID: "gpt-4o-08",
					ModelName:    "gpt-4o",
					ModelVersion: "2024-08-06",
				},
			},
			expected: "",
		},
		{
			name:     "empty models list",
			models:   []api.AzureModelResponse{},
			expected: "",
		},
		{
			name: "matching deployment-id but wrong model-name",
			models: []api.AzureModelResponse{
				{
					DeploymentID: "gpt-4o",
					ModelName:    "gpt-4o-mini",
					ModelVersion: "2024-07-18",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findGPT4oVersion(tt.models)
			if result != tt.expected {
				t.Errorf("findGPT4oVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestInitGPT4oVersion(t *testing.T) {
	tests := []struct {
		name           string
		genaiURL       string
		responseStatus int
		responseBody   string
		expectVersion  string
		expectError    bool
	}{
		{
			name:           "successful fetch with single model",
			genaiURL:       "will-be-replaced",
			responseStatus: http.StatusOK,
			responseBody: `{
				"models": [
					{
						"deployment-id": "gpt-4o",
						"model-name": "gpt-4o",
						"model-version": "2024-05-13",
						"deployment-type": "DataZoneStandard",
						"type": "chat-completion",
						"endpoint": "/deployments/gpt-4o/chat/completions"
					}
				]
			}`,
			expectVersion: "2024-05-13",
			expectError:   false,
		},
		{
			name:           "successful fetch with two gpt-4o models - picks correct one",
			genaiURL:       "will-be-replaced",
			responseStatus: http.StatusOK,
			responseBody: `{
				"models": [
					{
						"deployment-id": "gpt-4o",
						"model-name": "gpt-4o",
						"model-version": "2024-05-13",
						"deployment-type": "DataZoneStandard",
						"type": "chat-completion",
						"endpoint": "/deployments/gpt-4o/chat/completions"
					},
					{
						"deployment-id": "gpt-4o-08",
						"model-name": "gpt-4o",
						"model-version": "2024-08-06",
						"deployment-type": "DataZoneStandard",
						"type": "chat-completion",
						"endpoint": "/deployments/gpt-4o-08/chat/completions"
					}
				]
			}`,
			expectVersion: "2024-05-13",
			expectError:   false,
		},
		{
			name:           "empty URL",
			genaiURL:       "",
			responseStatus: 0,
			responseBody:   "",
			expectVersion:  "",
			expectError:    false,
		},
		{
			name:           "API returns 401",
			genaiURL:       "will-be-replaced",
			responseStatus: http.StatusUnauthorized,
			responseBody:   `{"error": "unauthorized"}`,
			expectVersion:  "",
			expectError:    true,
		},
		{
			name:           "model not found in response",
			genaiURL:       "will-be-replaced",
			responseStatus: http.StatusOK,
			responseBody: `{
				"models": [
					{
						"deployment-id": "gpt-4o-mini",
						"model-name": "gpt-4o-mini",
						"model-version": "2024-07-18"
					}
				]
			}`,
			expectVersion: "",
			expectError:   true,
		},
		{
			name:           "invalid JSON response",
			genaiURL:       "will-be-replaced",
			responseStatus: http.StatusOK,
			responseBody:   `{invalid json}`,
			expectVersion:  "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the cached version before each test
			gpt4oVersionMutex.Lock()
			gpt4oModelVersion = ""
			gpt4oVersionMutex.Unlock()

			ctx := cntx.ServiceContext("test-service")

			var server *httptest.Server
			if tt.genaiURL != "" && tt.genaiURL != "will-be-replaced" {
				// Use provided URL
			} else if tt.responseStatus > 0 {
				// Create test server
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify the request path
					if r.URL.Path != "/openai/models" {
						t.Errorf("Expected path '/openai/models', got '%s'", r.URL.Path)
					}

					w.WriteHeader(tt.responseStatus)
					_, _ = w.Write([]byte(tt.responseBody))
				}))
				defer server.Close()
				tt.genaiURL = server.URL
			}

			err := InitGPT4oVersion(ctx, tt.genaiURL, http.Header{})

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			version := GetGPT4oVersion()
			if version != tt.expectVersion {
				t.Errorf("Expected version '%s', got '%s'", tt.expectVersion, version)
			}
		})
	}
}

func TestGetGPT4oVersion(t *testing.T) {
	// Test getting version when it's set
	gpt4oVersionMutex.Lock()
	gpt4oModelVersion = "2024-05-13"
	gpt4oVersionMutex.Unlock()

	version := GetGPT4oVersion()
	if version != "2024-05-13" {
		t.Errorf("Expected version '2024-05-13', got '%s'", version)
	}

	// Test getting version when it's empty
	gpt4oVersionMutex.Lock()
	gpt4oModelVersion = ""
	gpt4oVersionMutex.Unlock()

	version = GetGPT4oVersion()
	if version != "" {
		t.Errorf("Expected empty version, got '%s'", version)
	}
}
