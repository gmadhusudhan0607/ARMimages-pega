/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package factory

import (
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbmock "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db/mocks"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var defaultHttpClientCfg = http_client.GetDefaultHTTPClientConfig()

func TestPrepareAdaClient(t *testing.T) {
	embedder, err := prepareAdaEmbedder("http://example.com/openai/deployments/text-embedding-ada-002/embeddings?api-version=2023-05-15", map[string]string{"Authorization": "Bearer token"}, defaultHttpClientCfg, zap.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, embedder)
}

func TestPrepareTitanClient(t *testing.T) {
	dbMock := dbmock.NewMockDb()
	profileID := "amazon-titan-embed-text"

	dbMock.Mock.ExpectQuery("SELECT ").
		WithArgs(profileID).
		WillReturnRows(sqlmock.NewRows([]string{"vector_len"}).AddRow(1024))

	embedder, err := prepareTitanEmbedder(dbMock, "iso-1", profileID, "http://example.com/amazon/deployments/titan-embed-text/embeddings", map[string]string{"Authorization": "Bearer token"}, http_client.GetDefaultHTTPClientConfig(), zap.NewNop())
	assert.NoError(t, err)
	assert.NotNil(t, embedder)
}

func TestGetModelURL(t *testing.T) {
	// Save original environment to restore later
	oldGenAIGateway := os.Getenv("GENAI_GATEWAY_SERVICE_URL")
	oldGenAICustomConfig := os.Getenv("GENAI_GATEWAY_CUSTOM_CONFIG")

	// Cleanup after test
	defer func() {
		os.Setenv("GENAI_GATEWAY_SERVICE_URL", oldGenAIGateway)
		os.Setenv("GENAI_GATEWAY_CUSTOM_CONFIG", oldGenAICustomConfig)
	}()

	// Test case 1: Standard configuration
	t.Run("Standard configuration", func(t *testing.T) {
		os.Setenv("GENAI_GATEWAY_SERVICE_URL", "http://gateway-service")
		os.Setenv("GENAI_GATEWAY_CUSTOM_CONFIG", "")

		url, err := getModelURL("openai-text-embedding-ada-002")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := "http://gateway-service/openai/deployments/text-embedding-ada-002/embeddings?api-version=2023-05-15"
		if url != expected {
			t.Errorf("Expected URL %s, got %s", expected, url)
		}
	})

	// Test case 2: Custom configuration
	t.Run("Custom configuration", func(t *testing.T) {
		os.Setenv("GENAI_GATEWAY_CUSTOM_CONFIG", `{"openai-text-embedding-ada-002":"https://custom-ada-endpoint.com"}`)

		url, err := getModelURL("openai-text-embedding-ada-002")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := "https://custom-ada-endpoint.com"
		if url != expected {
			t.Errorf("Expected URL %s, got %s", expected, url)
		}
	})

	// Test case 3: Invalid profile
	t.Run("Invalid profile", func(t *testing.T) {
		os.Setenv("GENAI_GATEWAY_CUSTOM_CONFIG", "")
		os.Setenv("GENAI_GATEWAY_SERVICE_URL", "http://gateway-service")

		_, err := getModelURL("non-existent-profile")
		if err == nil {
			t.Error("Expected error for invalid profile, got nil")
		}
	})

	// Test case 4: Invalid custom configuration
	t.Run("Invalid custom configuration", func(t *testing.T) {
		os.Setenv("GENAI_GATEWAY_CUSTOM_CONFIG", `{"invalid-profile":"https://endpoint.com"}`)

		_, err := getModelURL("openai-text-embedding-ada-002")
		if err == nil {
			t.Error("Expected error for invalid custom config, got nil")
		}
	})

	// Test case 5: Malformed JSON in custom configuration
	t.Run("Malformed custom configuration", func(t *testing.T) {
		os.Setenv("GENAI_GATEWAY_CUSTOM_CONFIG", `{invalid-json}`)

		_, err := getModelURL("openai-text-embedding-ada-002")
		if err == nil {
			t.Error("Expected error for malformed JSON, got nil")
		}
	})

	// Test case 5: Custom configuration but different profile
	t.Run("Standard configuration", func(t *testing.T) {
		os.Setenv("GENAI_GATEWAY_SERVICE_URL", "http://gateway-service")
		os.Setenv("GENAI_GATEWAY_CUSTOM_CONFIG", `{"amazon-titan-embed-text":"https://custom-endpoint.com"}`)

		// Test for a profile not overridden by custom config
		url, err := getModelURL("openai-text-embedding-ada-002")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expected := "http://gateway-service/openai/deployments/text-embedding-ada-002/embeddings?api-version=2023-05-15"
		if url != expected {
			t.Errorf("Expected URL %s, got %s", expected, url)
		}
	})

}
