//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	"github.com/onsi/gomega"
)

// TODO: look into the wiremock/go-wiremock as a client for wiremock server

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

type WireMockMappingParameters struct {
	WiremockURL       string
	IsolationID       string
	UrlPath           string
	UrlPattern        string
	ModelName         string
	ExpectedMaxTokens int
	CompletionTokens  int
}

// ############ Create Expectation Functions ############//

func (w *WireMockMappingParameters) CreateExpectation() (*functions.WireMockExpectation, error) {

	gomega.Expect(w.ModelName).NotTo(gomega.BeEmpty(), "modelName parameter must not be empty")

	// Default completionTokens to 50 if not specified (for backward compatibility)
	if w.CompletionTokens == 0 {
		w.CompletionTokens = 50
	}

	urlMatchingTypeKey := "urlPath"
	urlMatchingTypeValue := w.UrlPath
	if len(w.UrlPattern) > 0 {
		urlMatchingTypeKey = "urlPattern"
		urlMatchingTypeValue = w.UrlPattern
	}
	gomega.Expect(urlMatchingTypeValue).NotTo(gomega.BeEmpty(), "urlPath or urlPattern parameter must not be empty")

	// Build body patterns based on expectedMaxTokens
	var bodyPatterns []map[string]interface{}
	if w.ExpectedMaxTokens <= 0 {
		// Require max_tokens to be absent
		bodyPatterns = []map[string]interface{}{
			{
				"absent": "$.max_tokens",
			},
		}
	} else {
		// Require max_tokens to match expected value
		bodyPatterns = []map[string]interface{}{
			{
				"equalToJson":         fmt.Sprintf(`{"max_tokens":%d}`, w.ExpectedMaxTokens),
				"ignoreExtraElements": true,
			},
		}
	}

	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":           "POST",
			urlMatchingTypeKey: urlMatchingTypeValue,
			"headers": map[string]interface{}{
				"X-Genai-Gateway-Isolation-ID": map[string]string{
					"equalTo": w.IsolationID,
				},
			},
			"bodyPatterns": bodyPatterns,
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"jsonBody": map[string]interface{}{
				"id":      "chatcmpl-test",
				"object":  "chat.completion",
				"created": 1234567890,
				"model":   w.ModelName,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]string{
							"role":    "assistant",
							"content": "Test response",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]int{
					"prompt_tokens":     10,
					"completion_tokens": w.CompletionTokens,
					"total_tokens":      10 + w.CompletionTokens,
				},
			},
		},
	}

	return functions.CreateWireMockExpectation(w.WiremockURL, mapping)
}

// CreateWireMockMaxTokensExpectation creates expectation for max_tokens validation
// If expectedMaxTokens <= 0, max_tokens should be absent from the request
// completionTokens specifies the completion_tokens in the response (defaults to 50 if 0)
func CreateWireMockMaxTokensExpectation(wiremockURL, isolationID, urlPath, modelName string, expectedMaxTokens int, completionTokens int) (*functions.WireMockExpectation, error) {

	p := WireMockMappingParameters{
		WiremockURL:       wiremockURL,
		IsolationID:       isolationID,
		UrlPath:           urlPath,
		ModelName:         modelName,
		ExpectedMaxTokens: expectedMaxTokens,
		CompletionTokens:  completionTokens,
	}

	return p.CreateExpectation()

}

// CreateWireMockMaxTokensStreamingExpectation creates streaming SSE response expectation
// If expectedMaxTokens <= 0, max_tokens should be absent from the request
func CreateWireMockMaxTokensStreamingExpectation(wiremockURL, isolationID, urlPath, modelName string, expectedMaxTokens int) (*functions.WireMockExpectation, error) {
	gomega.Expect(urlPath).NotTo(gomega.BeEmpty(), "urlPath parameter must not be empty")
	gomega.Expect(modelName).NotTo(gomega.BeEmpty(), "modelName parameter must not be empty")

	streamingBody := fmt.Sprintf(`data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"%s","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"%s","choices":[{"index":0,"delta":{"content":"Test"},"finish_reason":null}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"%s","choices":[{"index":0,"delta":{"content":" response"},"finish_reason":null}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"%s","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`, modelName, modelName, modelName, modelName)

	// Build body patterns based on expectedMaxTokens
	var bodyPatterns []map[string]interface{}
	if expectedMaxTokens <= 0 {
		// Require max_tokens to be absent
		bodyPatterns = []map[string]interface{}{
			{
				"matchesJsonPath": "$.stream[?(@ == true)]",
			},
			{
				"absent": "$.max_tokens",
			},
		}
	} else {
		// Require max_tokens to match expected value
		bodyPatterns = []map[string]interface{}{
			{
				"matchesJsonPath": fmt.Sprintf("$.max_tokens[?(@ == %d)]", expectedMaxTokens),
			},
			{
				"matchesJsonPath": "$.stream[?(@ == true)]",
			},
		}
	}

	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":  "POST",
			"urlPath": urlPath,
			"headers": map[string]interface{}{
				"X-Genai-Gateway-Isolation-ID": map[string]string{
					"equalTo": isolationID,
				},
			},
			"bodyPatterns": bodyPatterns,
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type":  "text/event-stream",
				"Cache-Control": "no-cache",
				"Connection":    "keep-alive",
			},
			"body": streamingBody,
		},
	}

	return functions.CreateWireMockExpectation(wiremockURL, mapping)
}

//  #### Truncation and Retry Specific Expectations #### //

// CreateWireMockMaxTokensTruncatedExpectation creates expectation for truncated response that should trigger retry
func CreateWireMockMaxTokensTruncatedExpectation(wiremockURL, isolationID string, maxTokens int) (*functions.WireMockExpectation, error) {
	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":     "POST",
			"urlPattern": "/openai/deployments/gpt-35-turbo-1106/chat/completions.*",
			"headers": map[string]interface{}{
				"X-Genai-Gateway-Isolation-ID": map[string]string{
					"equalTo": isolationID,
				},
			},
			"bodyPatterns": []map[string]interface{}{
				{
					"equalToJson":         fmt.Sprintf(`{"max_tokens":%d}`, maxTokens),
					"ignoreExtraElements": true,
				},
			},
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"jsonBody": map[string]interface{}{
				"id":      "chatcmpl-test-truncated",
				"object":  "chat.completion",
				"created": 1234567890,
				"model":   "gpt-35-turbo",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]string{
							"role":    "assistant",
							"content": "This is a truncated response that was cut off due to max_tokens limit",
						},
						"finish_reason": "length", // This indicates truncation due to max_tokens
					},
				},
				"usage": map[string]int{
					"prompt_tokens":     10,
					"completion_tokens": maxTokens, // Used all available tokens
					"total_tokens":      10 + maxTokens,
				},
			},
		},
	}

	return functions.CreateWireMockExpectation(wiremockURL, mapping)
}

// CreateWireMockMaxTokensTruncatedStreamingExpectation creates expectation for truncated streaming response
func CreateWireMockMaxTokensTruncatedStreamingExpectation(wiremockURL, isolationID string, maxTokens int) (*functions.WireMockExpectation, error) {
	// Create a truncated streaming response that ends with "length" finish reason
	streamingBody := `data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"gpt-35-turbo","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"gpt-35-turbo","choices":[{"index":0,"delta":{"content":"This"},"finish_reason":null}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"gpt-35-turbo","choices":[{"index":0,"delta":{"content":" is"},"finish_reason":null}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"gpt-35-turbo","choices":[{"index":0,"delta":{"content":" truncated"},"finish_reason":null}]}

data: {"id":"1","object":"chat.completion.chunk","created":1234567890,"model":"gpt-35-turbo","choices":[{"index":0,"delta":{},"finish_reason":"length"}]}

data: [DONE]

`

	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":     "POST",
			"urlPattern": "/openai/deployments/gpt-35-turbo-1106/chat/completions.*",
			"headers": map[string]interface{}{
				"X-Genai-Gateway-Isolation-ID": map[string]string{
					"equalTo": isolationID,
				},
			},
			"bodyPatterns": []map[string]interface{}{
				{
					"equalToJson":         fmt.Sprintf(`{"max_tokens":%d}`, maxTokens),
					"ignoreExtraElements": true,
				},
				{
					"equalToJson":         `{"stream":true}`,
					"ignoreExtraElements": true,
				},
			},
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type":  "text/event-stream",
				"Cache-Control": "no-cache",
				"Connection":    "keep-alive",
			},
			"body": streamingBody,
		},
	}

	return functions.CreateWireMockExpectation(wiremockURL, mapping)
}

// CreateWireMockRetryExpectation creates expectation for retry request
// If expectedMaxTokens <= 0, max_tokens should be absent from the request
// If expectedMaxTokens > 0, max_tokens should match the expected value
func CreateWireMockRetryExpectation(wiremockURL, isolationID string, expectedMaxTokens int) (*functions.WireMockExpectation, error) {
	// Build body patterns based on expectedMaxTokens
	var bodyPatterns []map[string]interface{}
	if expectedMaxTokens <= 0 {
		// Require max_tokens to be absent - exact body match for retry scenario
		bodyPatterns = []map[string]interface{}{
			{
				"equalToJson": `{
					"messages": [
						{"role": "user", "content": "Hello, how are you?"}
					],
					"temperature": 0.7
				}`,
				"ignoreExtraElements": false,
			},
		}
	} else {
		// Require max_tokens to match expected value - for retry scenarios with max_tokens
		bodyPatterns = []map[string]interface{}{
			{
				"equalToJson": fmt.Sprintf(`{
					"messages": [
						{"role": "user", "content": "Hello, how are you?"}
					],
					"temperature": 0.7,
					"max_tokens": %d
				}`, expectedMaxTokens),
				"ignoreExtraElements": false,
			},
		}
	}

	mapping := map[string]interface{}{
		"request": map[string]interface{}{
			"method":     "POST",
			"urlPattern": "/openai/deployments/gpt-35-turbo-1106/chat/completions.*",
			"headers": map[string]interface{}{
				"X-Genai-Gateway-Isolation-ID": map[string]string{
					"equalTo": isolationID,
				},
			},
			"bodyPatterns": bodyPatterns,
		},
		"response": map[string]interface{}{
			"status": 200,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
			"jsonBody": map[string]interface{}{
				"id":      "chatcmpl-test-retry",
				"object":  "chat.completion",
				"created": 1234567890,
				"model":   "gpt-35-turbo",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]string{
							"role":    "assistant",
							"content": "This is the complete response after retry without max_tokens limit",
						},
						"finish_reason": "stop", // Completed successfully
					},
				},
				"usage": map[string]int{
					"prompt_tokens":     10,
					"completion_tokens": 75, // Less than the original max_tokens limit
					"total_tokens":      85,
				},
			},
		},
	}

	return functions.CreateWireMockExpectation(wiremockURL, mapping)
}

//############ Verify Functions ############//

// VerifyWireMockTruncatedExpectation verifies the truncated response expectation was matched
func VerifyWireMockTruncatedExpectation(wiremockURL, isolationID, urlPattern string, expectedCount int) error {
	gomega.Expect(urlPattern).NotTo(gomega.BeEmpty(), "urlPattern parameter must not be empty")

	// Use specific criteria that match only requests WITH max_tokens
	criteria := map[string]interface{}{
		"method":     "POST",
		"urlPattern": urlPattern,
		"headers": map[string]interface{}{
			"X-Genai-Gateway-Isolation-ID": map[string]string{
				"equalTo": isolationID,
			},
		},
		"bodyPatterns": []map[string]interface{}{
			{
				"matchesJsonPath": "$.max_tokens", // Only count requests WITH max_tokens
			},
		},
	}
	return functions.VerifyWireMockRequest(wiremockURL, criteria, expectedCount)
}

// VerifyWireMockRetryExpectation verifies the retry expectation was matched
func VerifyWireMockRetryExpectation(wiremockURL, isolationID, urlPattern string, expectedCount int) error {
	gomega.Expect(urlPattern).NotTo(gomega.BeEmpty(), "urlPattern parameter must not be empty")

	// Use specific criteria that match only retry requests (exact body match without max_tokens)
	criteria := map[string]interface{}{
		"method":     "POST",
		"urlPattern": urlPattern,
		"headers": map[string]interface{}{
			"X-Genai-Gateway-Isolation-ID": map[string]string{
				"equalTo": isolationID,
			},
		},
		"bodyPatterns": []map[string]interface{}{
			{
				"equalToJson": `{
					"messages": [
						{"role": "user", "content": "Hello, how are you?"}
					],
					"temperature": 0.7
				}`,
				"ignoreExtraElements": false,
			},
		},
	}
	return functions.VerifyWireMockRequest(wiremockURL, criteria, expectedCount)
}

// VerifyWireMockExpectation verifies the expectation was matched
func VerifyWireMockExpectation(wiremockURL, isolationID, urlPath string, expectedCount int) error {
	gomega.Expect(urlPath).NotTo(gomega.BeEmpty(), "urlPath parameter must not be empty")

	criteria := map[string]interface{}{
		"method": "POST",
		"headers": map[string]interface{}{
			"X-Genai-Gateway-Isolation-ID": map[string]string{
				"equalTo": isolationID,
			},
		},
		"urlPath": urlPath,
	}

	return functions.VerifyWireMockRequest(wiremockURL, criteria, expectedCount)
}

func VerifyWireMockExpectationId(wiremockURL, expecationId string, expectedCount int) error {
	return functions.VerifyWireMockRequestCountForExpecation(wiremockURL, expecationId, expectedCount)
}

// GetWireMockRequests retrieves all requests made to WireMock
func GetWireMockRequests(mockServerURL string) ([]WireMockRequest, error) {
	requestsURL := fmt.Sprintf("%s/__admin/requests", mockServerURL)
	resp, body, err := functions.ExpectHttpCall("GET", requestsURL, nil, "")
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

// VerifyRequestBodyUnchanged compares the original request body with what WireMock received
func VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, originalBody string) {
	requests, err := GetWireMockRequests(mockServerURL)
	gomega.Expect(err).To(gomega.BeNil())

	// Find the request that matches our criteria
	var matchingRequest *WireMockRequest
	for _, req := range requests {
		if req.Request.Method == "POST" && strings.Contains(req.Request.URL, expectedPath) {
			// For embeddings and buddy requests, don't require isolation header
			if strings.Contains(expectedPath, "/embeddings") || strings.Contains(expectedPath, "/question") {
				matchingRequest = &req
				break
			}
			// For other requests, require the isolation header
			if isolationHeader, exists := req.Request.Headers["X-Genai-Gateway-Isolation-Id"]; exists && isolationHeader == isolationID {
				matchingRequest = &req
				break
			}
		}
	}

	gomega.Expect(matchingRequest).NotTo(gomega.BeNil(), fmt.Sprintf("No matching request found for path %s with isolation ID %s", expectedPath, isolationID))

	// Compare the bodies exactly
	gomega.Expect(matchingRequest.Request.Body).To(gomega.Equal(originalBody),
		fmt.Sprintf("Request body was modified. Expected: %s, Got: %s", originalBody, matchingRequest.Request.Body))
}
