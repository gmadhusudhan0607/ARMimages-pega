/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package factory

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/http_client"
	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders/ada"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders/google"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders/random"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/embedders/titan"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
)

var DefaultEmbeddingProfileID = helpers.GetEnvOrDefault("DEFAULT_EMBEDDING_PROFILE", "openai-text-embedding-ada-002")

var embeddingProfileToUrlsMap = map[string]string{
	"openai-text-embedding-ada-002":          "/openai/deployments/text-embedding-ada-002/embeddings?api-version=2023-05-15",
	"openai-text-embedding-3-small":          "/openai/deployments/text-embedding-3-small/embeddings?api-version=2023-05-15",
	"openai-text-embedding-3-large":          "/openai/deployments/text-embedding-3-large/embeddings?api-version=2023-05-15",
	"amazon-titan-embed-text":                "/amazon/deployments/titan-embed-text/embeddings",
	"google-text-multilingual-embedding-002": "/google/deployments/text-multilingual-embedding-002/embeddings",
}

// CreateTextEmbedder creates a text embedder based on the provided configuration
func CreateTextEmbedder(database db.Database, isolationID, collectionID, embProfileID string, cfg *http_client.HTTPClientConfig, logger *zap.Logger) (embedders.TextEmbedder, error) {
	httpHeaders := getVsHeaders(isolationID, collectionID)

	// Random embedder is used for testing purposes, it should not be used in production
	if helpers.GetEnvOrDefault("RANDOM_EMBEDDER_ENABLED", "false") == "true" {
		return prepareRandomEmbedder(database, isolationID, embProfileID, httpHeaders)
	}

	url, err := getModelURL(embProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GenAI URL: %w", err)
	}

	switch embProfileID {
	case "openai-text-embedding-ada-002":
		return prepareAdaEmbedder(url, httpHeaders, resolveHttpClientCfg(cfg), logger)
	case "amazon-titan-embed-text":
		return prepareTitanEmbedder(database, isolationID, embProfileID, url, httpHeaders, resolveHttpClientCfg(cfg), logger)
	case "google-text-multilingual-embedding-002":
		return prepareGoogleEmbedder(url, httpHeaders, resolveHttpClientCfg(cfg), logger)
	default:
		return nil, fmt.Errorf("unsupported model (profile) : %s", embProfileID)
	}
}

func prepareRandomEmbedder(database db.Database, isolationID string, embProfileID string, httpHeaders map[string]string) (embedders.TextEmbedder, error) {
	var vectorLen = 1536 // default
	if embProfileID == "amazon-titan-embed-text" {
		var err error
		vectorLen, err = getVectorLength(database, isolationID, embProfileID)
		if err != nil {
			return nil, fmt.Errorf("failed to get vector length for random embedder: %w", err)
		}
	}
	randomEmbedder, err := random.NewRandomEmbedder("random://embedder", vectorLen, httpHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to init RandomEmbedder: %w", err)
	}
	return randomEmbedder, nil
}

func prepareAdaEmbedder(uri string, httpHeaders map[string]string, cfg http_client.HTTPClientConfig, logger *zap.Logger) (embedders.TextEmbedder, error) {
	a, err := ada.NewAdaEmbedder(uri, httpHeaders, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init AdaEmbedder: %w", err)
	}
	return a, nil
}

func prepareTitanEmbedder(database db.Database, isolationID, embProfileID, uri string, httpHeaders map[string]string, cfg http_client.HTTPClientConfig, logger *zap.Logger) (embedders.TextEmbedder, error) {
	// retrieve vector length from database
	vectorLength, err := getVectorLength(database, isolationID, embProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vector length: %w", err)
	}

	a, err := titan.NewTitanEmbedder(uri, vectorLength, httpHeaders, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init TitanEmbedder: %w", err)
	}
	return a, nil
}

func prepareGoogleEmbedder(uri string, httpHeaders map[string]string, cfg http_client.HTTPClientConfig, logger *zap.Logger) (embedders.TextEmbedder, error) {
	c, err := google.NewGoogleEmbedder(uri, httpHeaders, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init GoogleEmbedder: %w", err)
	}
	return c, nil
}

func getModelURL(profile string) (string, error) {

	if err := validateEmbeddingProfileName(profile); err != nil {
		return "", fmt.Errorf("failed to validate embedding profile name: %s", err.Error())
	}

	gwCustomConfig := helpers.GetEnvOrDefault("GENAI_GATEWAY_CUSTOM_CONFIG", "")

	// Custom config set, check if it contains overriding for the profile
	if gwCustomConfig != "" {
		// Unmarshal CustomConfig to map [string]string
		gwConfig := make(map[string]string)
		err := json.Unmarshal([]byte(gwCustomConfig), &gwConfig)
		if err != nil {
			return "", fmt.Errorf("failed to parse GENAI_GATEWAY_CUSTOM_CONFIG: %s", err.Error())
		}

		if err = validateGwCustomConfig(gwConfig); err != nil {
			return "", fmt.Errorf("failed to validate GENAI_GATEWAY_CUSTOM_CONFIG: %s", err.Error())
		}

		// Check if gwConfig contains the profile
		if _, ok := gwConfig[profile]; ok {
			return gwConfig[profile], nil
		}
	}

	// Use default gateway service URL
	gwSvcUrl := helpers.GetEnvOrPanic("GENAI_GATEWAY_SERVICE_URL")
	url := fmt.Sprintf("%s%s", gwSvcUrl, embeddingProfileToUrlsMap[profile])
	return url, nil

}

func validateEmbeddingProfileName(profile string) error {
	// validate profile name
	mapKeys := make([]string, 0, len(embeddingProfileToUrlsMap))
	for k := range embeddingProfileToUrlsMap {
		mapKeys = append(mapKeys, k)
	}
	if _, ok := embeddingProfileToUrlsMap[profile]; !ok {
		return fmt.Errorf("unsupported embedding profile: %s, Supported profiles: [%s]",
			profile, strings.Join(mapKeys, ", "))
	}
	return nil
}

func validateGwCustomConfig(gwConfig map[string]string) error {
	// validate profile name
	mapKeys := make([]string, 0, len(embeddingProfileToUrlsMap))
	for k := range embeddingProfileToUrlsMap {
		mapKeys = append(mapKeys, k)
	}
	for profileName, url := range gwConfig {
		// panic in not in embeddingProfileToUrlsMap
		if !slices.Contains(mapKeys, profileName) {
			return fmt.Errorf("unsupported GENAI_GATEWAY_CUSTOM_CONFIG key: %s, Supported profiles: [%s]",
				profileName, strings.Join(mapKeys, ", "))
		}
		// panic if url is empty
		if url == "" {
			return fmt.Errorf("GENAI_GATEWAY_CUSTOM_CONFIG[%s] is empty", profileName)
		}
		// panic if url is not a valid URL
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return fmt.Errorf("GENAI_GATEWAY_CUSTOM_CONFIG[%s] is not a valid URL", profileName)
		}
	}
	return nil
}

func getVsHeaders(isolationID, collectionID string) map[string]string {
	return map[string]string{
		"vs-isolation-id":  isolationID,
		"vs-collection-id": collectionID,
	}
}

func getVectorLength(database db.Database, isolationID, embProfileID string) (int, error) {
	tableProfiles := db.GetTableEmbeddingProfiles(isolationID)

	query := "SELECT vector_len FROM " + tableProfiles + " WHERE profile_id = $1"
	rows, err := database.GetConn().Query(query, embProfileID)
	if err != nil {
		return 0, fmt.Errorf("failed to query vector length: %w", err)
	}
	defer rows.Close()

	var vectorLen int
	if rows.Next() {
		err = rows.Scan(&vectorLen)
		if err != nil {
			return 0, fmt.Errorf("failed to scan vector length: %w", err)
		}
	} else {
		return 0, fmt.Errorf("embedding profile '%s' not found", embProfileID)
	}

	if vectorLen == 0 {
		return 0, fmt.Errorf("vector length for embedding profile '%s' is 0, which is not allowed", embProfileID)
	}

	return vectorLen, nil
}

func resolveHttpClientCfg(cfg *http_client.HTTPClientConfig) http_client.HTTPClientConfig {
	var adaCfg http_client.HTTPClientConfig
	if cfg != nil {
		adaCfg = *cfg
	} else {
		adaCfg = http_client.GetDefaultHTTPClientConfig()
	}
	return adaCfg
}
