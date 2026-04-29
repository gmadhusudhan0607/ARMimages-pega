//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package unrecognized_test

import (
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	. "github.com/onsi/gomega"
)

// CreateEmbeddingRequestBody creates a request body for embedding API calls
func CreateEmbeddingRequestBody(input string, maxTokens *int) string {
	requestObj := map[string]interface{}{
		"input": input,
		"model": "text-embedding-ada-002",
	}

	if maxTokens != nil {
		requestObj["max_tokens"] = *maxTokens
	}

	jsonBytes, err := json.Marshal(requestObj)
	Expect(err).To(BeNil())

	return string(jsonBytes)
}

// CreateBuddyRequestBody creates a request body for buddy API calls
func CreateBuddyRequestBody(question string, maxTokens *int) string {
	requestObj := map[string]interface{}{
		"question":    question,
		"temperature": 0.7,
		"context": map[string]interface{}{
			"subject":    "general",
			"difficulty": "beginner",
			"metadata": map[string]interface{}{
				"session_id": "study-session-123",
				"user_id":    "student-456",
			},
		},
	}

	if maxTokens != nil {
		requestObj["max_tokens"] = *maxTokens
	}

	jsonBytes, err := json.Marshal(requestObj)
	Expect(err).To(BeNil())

	return string(jsonBytes)
}

// CreateFakeModelRequestBody creates a request body for fake-model-test calls
func CreateFakeModelRequestBody(prompt string, maxTokens *int) string {
	requestObj := map[string]interface{}{
		"model": "fake-model-test",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
	}

	if maxTokens != nil {
		requestObj["max_tokens"] = *maxTokens
	}

	jsonBytes, err := json.Marshal(requestObj)
	Expect(err).To(BeNil())

	return string(jsonBytes)
}

// VerifyWireMockExpectation verifies that a WireMock expectation was called the expected number of times
func VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath string, expectedCount int) error {
	// For embeddings and buddy paths, don't require isolation header as the service doesn't send it
	if strings.Contains(expectedPath, "/embeddings") || strings.Contains(expectedPath, "/question") {
		criteria := map[string]interface{}{
			"method":  "POST",
			"urlPath": expectedPath,
		}
		return VerifyWireMockRequest(mockServerURL, criteria, expectedCount)
	}

	// For other paths, require the isolation header
	criteria := map[string]interface{}{
		"method":  "POST",
		"urlPath": expectedPath,
		"headers": map[string]interface{}{
			"X-Genai-Gateway-Isolation-Id": map[string]interface{}{
				"equalTo": isolationID,
			},
		},
	}

	return VerifyWireMockRequest(mockServerURL, criteria, expectedCount)
}

// WireMockRequest represents a request captured by WireMock
type WireMockRequest struct {
	ID      string `json:"id"`
	Request struct {
		URL         string            `json:"url"`
		AbsoluteURL string            `json:"absoluteUrl"`
		Method      string            `json:"method"`
		Headers     map[string]string `json:"headers"`
		Body        string            `json:"body"`
		LoggedDate  int64             `json:"loggedDate"`
	} `json:"request"`
}

// WireMockRequestsResponse represents the response from WireMock's requests endpoint
type WireMockRequestsResponse struct {
	Requests []WireMockRequest `json:"requests"`
}

// GetWireMockRequests retrieves all requests made to WireMock
func GetWireMockRequests(mockServerURL string) ([]WireMockRequest, error) {
	requestsURL := fmt.Sprintf("%s/__admin/requests", mockServerURL)
	resp, body, err := ExpectHttpCall("GET", requestsURL, nil, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get WireMock requests: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get WireMock requests, status: %d, body: %s", resp.StatusCode, string(body))
	}

	var response WireMockRequestsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal WireMock requests response: %w", err)
	}

	return response.Requests, nil
}

// CreateWireMockExpectationWithExactBody creates a WireMock expectation that matches exact request body
func CreateWireMockExpectationWithExactBody(wiremockURL, isolationID, expectedPath, expectedBody, responseBody string) (string, error) {
	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":  "POST",
			"urlPath": expectedPath,
			"bodyPatterns": []map[string]interface{}{
				{
					"equalTo": expectedBody,
				},
			},
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]interface{}{
				"Content-Type": "application/json",
			},
			"body": responseBody,
		},
	}

	expectation, err := CreateWireMockExpectation(wiremockURL, mapping)
	if err != nil {
		return "", err
	}

	return expectation.Id, nil
}

// CreateUnrecognizedLLMRequestBody creates a request body for unrecognized-llm calls
func CreateUnrecognizedLLMRequestBody(prompt string, maxTokens *int) string {
	requestObj := map[string]interface{}{
		"model": "unrecognized-llm",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
	}

	if maxTokens != nil {
		requestObj["max_tokens"] = *maxTokens
	}

	jsonBytes, err := json.Marshal(requestObj)
	Expect(err).To(BeNil())

	return string(jsonBytes)
}

// CreateFakeGptUltraRequestBody creates a request body for fake-gpt-ultra calls
func CreateFakeGptUltraRequestBody(prompt string, maxTokens *int) string {
	requestObj := map[string]interface{}{
		"model": "fake-gpt-ultra",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
	}

	if maxTokens != nil {
		requestObj["max_tokens"] = *maxTokens
	}

	jsonBytes, err := json.Marshal(requestObj)
	Expect(err).To(BeNil())

	return string(jsonBytes)
}

// CreateNonExistentEmbeddingRequestBody creates a request body for non-existent-embedding calls
func CreateNonExistentEmbeddingRequestBody(input string, maxTokens *int) string {
	requestObj := map[string]interface{}{
		"input": input,
		"model": "non-existent-embedding",
	}

	if maxTokens != nil {
		requestObj["max_tokens"] = *maxTokens
	}

	jsonBytes, err := json.Marshal(requestObj)
	Expect(err).To(BeNil())

	return string(jsonBytes)
}
