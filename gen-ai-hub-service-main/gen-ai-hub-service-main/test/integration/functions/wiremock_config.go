//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package functions

import (
	"encoding/json"
	"fmt"
	"os"
)

// CreateMappingEndpointExpectation creates WireMock expectation for MAPPING_ENDPOINT
// Returns AWS Bedrock model configurations loaded from the specified file
func CreateMappingEndpointExpectation(mockServerURL, urlPath, mappingFilePath string) error {
	// Read the mapping file
	fileData, err := os.ReadFile(mappingFilePath)
	if err != nil {
		return fmt.Errorf("failed to read mapping file %s: %w", mappingFilePath, err)
	}

	// Parse the JSON data
	var mappingResponse []map[string]interface{}
	if err := json.Unmarshal(fileData, &mappingResponse); err != nil {
		return fmt.Errorf("failed to parse mapping file %s: %w", mappingFilePath, err)
	}

	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":  "GET",
			"urlPath": urlPath,
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"jsonBody": mappingResponse,
		},
	}

	// Use retry logic to handle connection issues
	var expectation *WireMockExpectation
	retryConfig := DefaultRetryConfig()
	err = RetryWithBackoff(func() error {
		var createErr error
		expectation, createErr = CreateWireMockExpectation(mockServerURL, mapping)
		return createErr
	}, retryConfig)

	if err != nil {
		return fmt.Errorf("failed to create mapping endpoint expectation: %w", err)
	}

	fmt.Printf("Created mapping endpoint expectation with ID: %s\n", expectation.Id)
	return nil
}

// CreateDefaultsEndpointExpectation creates WireMock expectation for MODELS_DEFAULTS_ENDPOINT
// Returns default fast/smart model configurations loaded from the specified file
func CreateDefaultsEndpointExpectation(mockServerURL, urlPath, defaultsFilePath string) error {
	// Read the defaults file
	fileData, err := os.ReadFile(defaultsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read defaults file %s: %w", defaultsFilePath, err)
	}

	// Parse the JSON data
	var defaultsResponse map[string]interface{}
	if err := json.Unmarshal(fileData, &defaultsResponse); err != nil {
		return fmt.Errorf("failed to parse defaults file %s: %w", defaultsFilePath, err)
	}

	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":  "GET",
			"urlPath": urlPath,
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"jsonBody": defaultsResponse,
		},
	}

	// Use retry logic to handle connection issues
	var expectation *WireMockExpectation
	retryConfig := DefaultRetryConfig()
	err = RetryWithBackoff(func() error {
		var createErr error
		expectation, createErr = CreateWireMockExpectation(mockServerURL, mapping)
		return createErr
	}, retryConfig)

	if err != nil {
		return fmt.Errorf("failed to create defaults endpoint expectation: %w", err)
	}

	fmt.Printf("Created defaults endpoint expectation with ID: %s\n", expectation.Id)
	return nil
}

// CreateMonitoringEndpointExpectation creates WireMock expectation for MONITORING_ENDPOINT
// Returns HTTP 200 OK for any POST request to accept monitoring events
func CreateMonitoringEndpointExpectation(mockServerURL, urlPath string) error {
	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":  "POST",
			"urlPath": urlPath,
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"jsonBody": map[string]interface{}{
				"status": "ok",
			},
		},
	}

	// Use retry logic to handle connection issues
	var expectation *WireMockExpectation
	retryConfig := DefaultRetryConfig()
	err := RetryWithBackoff(func() error {
		var createErr error
		expectation, createErr = CreateWireMockExpectation(mockServerURL, mapping)
		return createErr
	}, retryConfig)

	if err != nil {
		return fmt.Errorf("failed to create monitoring endpoint expectation: %w", err)
	}

	fmt.Printf("Created monitoring endpoint expectation with ID: %s\n", expectation.Id)
	return nil
}

// CreatePrivateModelFiles creates test private model configuration files
func CreatePrivateModelFiles(privateModelsDir string) error {
	privateModelYAML := `models:
  - name: gpt-4-private-test
    modelId: ""
    modelUrl: "http://localhost:20090/openai/deployments/gpt-4-private-test"
    redirectUrl: "http://localhost:11818/openai/deployments/gpt-4-private-test"
    provider: openai
    creator: openai
    targetAPI: "/chat/completions"
    path: "/openai/deployments/gpt-4-private-test/chat/completions"
    infrastructure: azure
    active: true
    capabilities:
      completions: true
      embeddings: false
`

	filePath := fmt.Sprintf("%s/private-model-test-azure.yaml", privateModelsDir)
	if err := os.WriteFile(filePath, []byte(privateModelYAML), 0644); err != nil {
		return fmt.Errorf("failed to create private model file: %w", err)
	}

	fmt.Printf("Created private model file: %s\n", filePath)
	return nil
}
