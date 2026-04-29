// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package test_functions

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/tools"
	"github.com/google/uuid"
)

// CreateAdaEmbeddingVector creates a standard 1536-dimensional embedding vector
// with all values set to 0.1 for testing purposes
func CreateAdaEmbeddingVector() []float64 {
	embedding := make([]float64, 1536)
	for i := range embedding {
		embedding[i] = 0.1
	}
	return embedding
}

// CreateGenAIGatewayHeaders creates the standard set of GenAI Gateway response headers
// used in embedding responses
func CreateGenAIGatewayHeaders() map[string]string {
	return map[string]string{
		"Content-Type":                      "application/json",
		"X-Genai-Gateway-Response-Time-Ms":  "111",
		"X-Genai-Gateway-Input-Tokens":      "222",
		"X-Genai-Gateway-Model-Id":          "text-embedding-ada-002",
		"X-Genai-Gateway-Region":            "us-east-1",
		"X-Genai-Gateway-Output-Tokens":     "333",
		"X-Genai-Gateway-Tokens-Per-Second": "444",
		"X-Genai-Gateway-Retry-Count":       "2",
	}
}

// CreateEmbeddingResponseBody creates the standard successful embedding response JSON body
func CreateEmbeddingResponseBody(embedding []float64) map[string]interface{} {
	return map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"object":    "embedding",
				"embedding": embedding,
				"index":     0,
			},
		},
		"model": "/var/azureml-app/azureml-models/text-embedding-ada-002-8k/584175/",
		"usage": map[string]interface{}{
			"prompt_tokens": 3,
			"total_tokens":  3,
		},
	}
}

// CreateAdaRequestPattern creates the standard request matching pattern for Ada embeddings
// with the specified isolation ID using GenAI gateway headers.
func CreateAdaRequestPattern(isolationID string) map[string]interface{} {
	return map[string]interface{}{
		"method":         "POST",
		"urlPathPattern": "/openai/deployments/text-embedding-ada-002/embeddings",
		"headers": map[string]interface{}{
			"X-Genai-Vectorstore-Isolation-Id": map[string]interface{}{
				"equalTo": isolationID,
			},
		},
	}
}

// CreateAdaRequestPatternWithoutIsolationValidation creates a request matching pattern for Ada embeddings
// that does not validate the X-Genai-Vectorstore-Isolation-Id header. It matches only on method and URL path.
// This is useful for scenarios where the service under test does not send the isolation header.
func CreateAdaRequestPatternWithoutIsolationValidation() map[string]interface{} {
	return map[string]interface{}{
		"method":         "POST",
		"urlPathPattern": "/openai/deployments/text-embedding-ada-002/embeddings",
	}
}

// CreateAdaExpectationFromTpl creates a WireMock stub for the Ada embedder endpoint
// based on an existing mock-server expectation template JSON. It preserves the
// response status code, optional delay, headers, and body from the template, but
// intentionally does not validate request headers (it matches only on method and path)
// to avoid coupling tests to specific header shapes.
func CreateAdaExpectationFromTpl(wiremockMgr *tools.WireMockManager, expectationTemplate string) (string, error) {
	if wiremockMgr == nil {
		return "", fmt.Errorf("wiremock manager is nil")
	}

	type httpRequest struct {
		Method string          `json:"method"`
		Path   string          `json:"path"`
		Body   json.RawMessage `json:"body,omitempty"`
	}

	type delay struct {
		TimeUnit string `json:"timeUnit"`
		Value    int    `json:"value"`
	}

	type httpResponse struct {
		StatusCode int               `json:"statusCode"`
		Headers    map[string]string `json:"headers"`
		Body       json.RawMessage   `json:"body"`
		Delay      *delay            `json:"delay,omitempty"`
	}

	type adaTemplate struct {
		HTTPRequest  httpRequest  `json:"httpRequest"`
		HTTPResponse httpResponse `json:"httpResponse"`
	}

	var tpl adaTemplate
	if err := json.Unmarshal([]byte(expectationTemplate), &tpl); err != nil {
		return "", fmt.Errorf("failed to unmarshal Ada expectation template: %w", err)
	}

	request := map[string]interface{}{
		"method":  tpl.HTTPRequest.Method,
		"urlPath": tpl.HTTPRequest.Path,
	}

	// When multiple expectations exist for the same Ada endpoint (e.g., different
	// document chunks and a separate query embedding), we need a way to route each
	// request to the correct stub. Use a simple body matcher based on the
	// `input` field from the original mock-server template to preserve the
	// original semantics without depending on headers.
	if len(tpl.HTTPRequest.Body) > 0 {
		var bodyVal interface{}
		if err := json.Unmarshal(tpl.HTTPRequest.Body, &bodyVal); err == nil {
			switch v := bodyVal.(type) {
			case map[string]interface{}:
				if input, ok := v["input"]; ok {
					switch iv := input.(type) {
					case string:
						request["bodyPatterns"] = []map[string]interface{}{{"contains": iv}}
					case []interface{}:
						if len(iv) > 0 {
							if first, ok := iv[0].(string); ok {
								request["bodyPatterns"] = []map[string]interface{}{{"contains": first}}
							}
						}
					}
				}
			case string:
				request["bodyPatterns"] = []map[string]interface{}{{"contains": v}}
			}
		}
	}

	response := map[string]interface{}{
		"status": tpl.HTTPResponse.StatusCode,
	}

	if len(tpl.HTTPResponse.Headers) > 0 {
		response["headers"] = tpl.HTTPResponse.Headers
	}

	if tpl.HTTPResponse.Delay != nil {
		// Map the mock-server delay into WireMock's fixedDelayMilliseconds
		response["fixedDelayMilliseconds"] = tpl.HTTPResponse.Delay.Value
	}

	if len(tpl.HTTPResponse.Body) > 0 {
		// Body can be either a JSON object or a JSON string. Decode generically
		// and choose jsonBody or body accordingly.
		var body interface{}
		if err := json.Unmarshal(tpl.HTTPResponse.Body, &body); err != nil {
			return "", fmt.Errorf("failed to unmarshal Ada response body: %w", err)
		}

		switch v := body.(type) {
		case string:
			// When the template body is a JSON string, preserve it as a plain body string.
			response["body"] = v
		default:
			// For JSON objects/arrays, use jsonBody so WireMock serializes them as JSON.
			response["jsonBody"] = v
		}
	}

	return wiremockMgr.CreateStub(request, response)
}

// smartChunkingTemplate is used to parse mock-server smart chunking expectation templates
// so we can reuse their response bodies in WireMock stubs.
type smartChunkingTemplate struct {
	HttpResponse struct {
		Body string `json:"body"`
	} `json:"httpResponse"`
}

// loadSmartChunkingResponseBody loads the smart-chunking response body from a mock expectation template file.
func loadSmartChunkingResponseBody(templateFile string) (string, error) {
	raw := ReadTestDataFile(templateFile)
	var tpl smartChunkingTemplate
	if err := json.Unmarshal([]byte(raw), &tpl); err != nil {
		return "", fmt.Errorf("failed to parse smart-chunking template %s: %w", templateFile, err)
	}
	return tpl.HttpResponse.Body, nil
}

// Deprecated: CreateSmartChunkingRequestPattern is used by pre-US-732363 tests.
// SC is now called via /v1/{isolationID}/jobs. Use CreateExpectationSmartChunkingJob instead.
func CreateSmartChunkingRequestPattern(isolationID, collectionID string) map[string]interface{} {
	return map[string]interface{}{
		"method":         "POST",
		"urlPathPattern": "/v1/chunk",
		"headers": map[string]interface{}{
			"vs-isolation-id": map[string]interface{}{
				"equalTo": isolationID,
			},
			"vs-collection-id": map[string]interface{}{
				"equalTo": collectionID,
			},
		},
	}
}

// Deprecated: CreateExpectationSmartChunkingFromTemplate is used by pre-US-732363 tests.
// SC is now called via /v1/{isolationID}/jobs. Use CreateExpectationSmartChunkingJob instead.
func CreateExpectationSmartChunkingFromTemplate(wiremockMgr *tools.WireMockManager, isolationID, collectionID, templateFile string) (string, error) {
	body, err := loadSmartChunkingResponseBody(templateFile)
	if err != nil {
		return "", err
	}

	response := map[string]interface{}{
		"status": 200,
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
		},
		"body": body,
	}

	stub := map[string]interface{}{
		"request":  CreateSmartChunkingRequestPattern(isolationID, collectionID),
		"response": response,
	}

	return wiremockMgr.CreateStub(stub["request"], stub["response"])
}

// CreateExpectationSmartChunkingJob creates a WireMock stub for the SC /v1/{isolationID}/jobs endpoint.
// This replaces the old /v1/chunk mock for the new async job submission flow.
// Returns the stub mapping ID, the generated operationID, and any error.
func CreateExpectationSmartChunkingJob(wiremockMgr *tools.WireMockManager, isolationID string) (string, string, error) {
	if wiremockMgr == nil {
		return "", "", fmt.Errorf("wiremock manager is nil")
	}

	operationID := uuid.New().String()
	request := map[string]interface{}{
		"method":  "POST",
		"urlPath": fmt.Sprintf("/v1/%s/jobs", isolationID),
	}
	response := map[string]interface{}{
		"status":  202,
		"headers": map[string]string{"Content-Type": "application/json"},
		"jsonBody": map[string]interface{}{
			"operationID":        operationID,
			"isolationID":        isolationID,
			"status":             "PENDING",
			"requestedTasks":     []string{"extraction", "chunking", "indexing"},
			"inlineResults":      []string{},
			"message":            fmt.Sprintf("Job accepted. Poll GET /v1/%s/jobs/%s for the result.", isolationID, operationID),
			"callbackRegistered": false,
		},
	}

	stubID, err := wiremockMgr.CreateStub(request, response)
	return stubID, operationID, err
}

// CreateExpectationSmartChunkingJobError creates a WireMock stub for the SC /v1/{isolationID}/jobs endpoint
// that returns an error response. Used to test SC unavailability scenarios.
func CreateExpectationSmartChunkingJobError(wiremockMgr *tools.WireMockManager, isolationID string, statusCode int) (string, error) {
	if wiremockMgr == nil {
		return "", fmt.Errorf("wiremock manager is nil")
	}

	request := map[string]interface{}{
		"method":  "POST",
		"urlPath": fmt.Sprintf("/v1/%s/jobs", isolationID),
	}
	response := map[string]interface{}{
		"status":  statusCode,
		"headers": map[string]string{"Content-Type": "application/json"},
		"jsonBody": map[string]interface{}{
			"error": fmt.Sprintf("simulated SC error (status %d)", statusCode),
		},
	}

	return wiremockMgr.CreateStub(request, response)
}

// CreateAdaRequestPatternWithVsHeaders creates a request matching pattern for Ada embeddings
// that uses legacy vs-* headers (vs-isolation-id, vs-collection-id). This is required for
// endpoints that still send these headers when calling the embedder.
func CreateAdaRequestPatternWithVsHeaders(isolationID, collectionID string) map[string]interface{} {
	return map[string]interface{}{
		"method":         "POST",
		"urlPathPattern": "/openai/deployments/text-embedding-ada-002/embeddings",
		"headers": map[string]interface{}{
			"vs-isolation-id": map[string]interface{}{
				"equalTo": isolationID,
			},
			"vs-collection-id": map[string]interface{}{
				"equalTo": collectionID,
			},
		},
	}
}

// CreateAdaStubResponse creates a stub response configuration with the provided parameters
// If delayMs is nil, no delay is added. If statusCode is not 200, creates an error response.
func CreateAdaStubResponse(embedding []float64, statusCode int, delayMs *int) map[string]interface{} {
	response := map[string]interface{}{
		"status":  statusCode,
		"headers": CreateGenAIGatewayHeaders(),
	}

	// Add delay if specified
	if delayMs != nil {
		response["fixedDelayMilliseconds"] = *delayMs
	}

	// Add appropriate body based on status code
	if statusCode == 200 {
		response["jsonBody"] = CreateEmbeddingResponseBody(embedding)
	} else {
		response["jsonBody"] = map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Simulated error for testing",
				"type":    "server_error",
				"code":    statusCode,
			},
		}
	}

	return response
}

// CreateExpectationEmbeddingAdaWithoutIsolationValidation creates a WireMock stub for the OpenAI
// text-embedding-ada-002 endpoint that does not validate the X-Genai-Vectorstore-Isolation-Id header.
// It matches only on method and URL path and returns a fixed embedding vector. This is intended for
// scenarios where the service under test does not send the isolation header on embedder requests.
// Returns the unique mapping ID which can be used to delete this specific expectation.
func CreateExpectationEmbeddingAdaWithoutIsolationValidation(wiremockMgr *tools.WireMockManager) (string, error) {
	embedding := CreateAdaEmbeddingVector()
	stub := map[string]interface{}{
		"request":  CreateAdaRequestPatternWithoutIsolationValidation(),
		"response": CreateAdaStubResponse(embedding, 200, nil),
	}
	return wiremockMgr.CreateStub(stub["request"], stub["response"])
}

// Deprecated: CreateSmartChunkingExpectationFromTpl is used by pre-US-732363 tests.
// SC is now called via /v1/{isolationID}/jobs. Use CreateExpectationSmartChunkingJob instead.
func CreateSmartChunkingExpectationFromTpl(wiremockMgr *tools.WireMockManager, expectationTemplate string, isolationID, collectionID string) (string, error) {
	if wiremockMgr == nil {
		return "", fmt.Errorf("wiremock manager is nil")
	}

	// Parse only the pieces we need from the mock-server expectation template.
	type httpResponse struct {
		StatusCode int    `json:"statusCode"`
		Body       string `json:"body"`
	}
	type smartChunkingTemplate struct {
		HTTPResponse httpResponse `json:"httpResponse"`
	}

	var tpl smartChunkingTemplate
	if err := json.Unmarshal([]byte(expectationTemplate), &tpl); err != nil {
		return "", fmt.Errorf("failed to unmarshal smart-chunking expectation template: %w", err)
	}

	if tpl.HTTPResponse.StatusCode == 0 {
		// Default to 200 if not explicitly set in template
		tpl.HTTPResponse.StatusCode = 200
	}

	// The body field in the template is a JSON string containing the actual
	// smart-chunking response. Decode it so we can use jsonBody in WireMock.
	var bodyJSON map[string]interface{}
	if err := json.Unmarshal([]byte(tpl.HTTPResponse.Body), &bodyJSON); err != nil {
		return "", fmt.Errorf("failed to unmarshal smart-chunking response body: %w", err)
	}

	request := map[string]interface{}{
		"method":  "POST",
		"urlPath": "/v1/chunk",
		"headers": map[string]interface{}{
			"vs-isolation-id": map[string]interface{}{
				"equalTo": isolationID,
			},
			"vs-collection-id": map[string]interface{}{
				"equalTo": collectionID,
			},
		},
	}

	response := map[string]interface{}{
		"status": tpl.HTTPResponse.StatusCode,
		"headers": map[string]string{
			"Content-Type": "application/json",
		},
		"jsonBody": bodyJSON,
	}

	return wiremockMgr.CreateStub(request, response)
}

// CreateExpectationEmbeddingAda creates a WireMock stub for OpenAI text-embedding-ada-002 endpoint
// that validates the isolation ID header and returns a fixed embedding vector
// Returns the unique mapping ID which can be used to delete this specific expectation
func CreateExpectationEmbeddingAda(wiremockMgr *tools.WireMockManager, isolationID string) (string, error) {
	embedding := CreateAdaEmbeddingVector()
	stub := map[string]interface{}{
		"request":  CreateAdaRequestPattern(isolationID),
		"response": CreateAdaStubResponse(embedding, 200, nil),
	}
	return wiremockMgr.CreateStub(stub["request"], stub["response"])
}

// CreateExpectationEmbeddingAdaWithVsHeaders creates a WireMock stub for the Ada endpoint
// that matches requests using legacy vs-* headers. This is used by endpoints that have not
// yet been migrated to send GenAI gateway headers to the embedder.
func CreateExpectationEmbeddingAdaWithVsHeaders(wiremockMgr *tools.WireMockManager, isolationID, collectionID string) (string, error) {
	embedding := CreateAdaEmbeddingVector()
	stub := map[string]interface{}{
		"request":  CreateAdaRequestPatternWithVsHeaders(isolationID, collectionID),
		"response": CreateAdaStubResponse(embedding, 200, nil),
	}
	return wiremockMgr.CreateStub(stub["request"], stub["response"])
}

// DeleteExpectation deletes a specific expectation by its mapping ID
// Returns an error if deletion fails, including 404 errors
func DeleteExpectation(wiremockMgr *tools.WireMockManager, mappingID string) error {
	return deleteExpectation(wiremockMgr, mappingID, false)
}

// DeleteExpectationIfExist deletes a specific expectation by its mapping ID
// Returns nil if the mock was successfully deleted or if it was already deleted (404 error)
// This is useful for cleanup where it's acceptable if the mock was already deleted
func DeleteExpectationIfExist(wiremockMgr *tools.WireMockManager, mappingID string) error {
	return deleteExpectation(wiremockMgr, mappingID, true)
}

// deleteExpectation is the common implementation for deleting expectations
// ignoreNotFound parameter controls whether 404 errors should be ignored
func deleteExpectation(wiremockMgr *tools.WireMockManager, mappingID string, ignoreNotFound bool) error {
	if wiremockMgr == nil {
		return fmt.Errorf("wiremock manager is nil")
	}
	err := wiremockMgr.DeleteMapping(mappingID)
	if err != nil && ignoreNotFound && isNotFoundError(err) {
		// Ignore 404 errors when requested (for cleanup scenarios)
		return nil
	}
	return err
}

// isNotFoundError checks if an error is a 404 Not Found error from WireMock
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check if error message contains "status 404" which indicates the mapping was not found
	return strings.Contains(err.Error(), "status 404")
}

// VerifyEmbedderCalls verifies that the embedder endpoint was called at least minCalls times
func VerifyEmbedderCalls(wiremockMgr *tools.WireMockManager, minCalls int) error {
	if wiremockMgr == nil {
		return fmt.Errorf("wiremock manager is nil")
	}

	requestPattern := map[string]interface{}{
		"method":         "POST",
		"urlPathPattern": "/openai/deployments/text-embedding-ada-002/embeddings",
	}

	count, err := wiremockMgr.VerifyRequest(requestPattern)
	if err != nil {
		return fmt.Errorf("failed to verify embedder calls: %w", err)
	}

	if count < minCalls {
		return fmt.Errorf("expected at least %d embedder calls, but got %d", minCalls, count)
	}

	return nil
}

// GetAdaEmbedderCallCount returns the current count of Ada embedder requests for a specific isolation ID
// This is useful for tracking requests across different phases of a test
func GetAdaEmbedderCallCount(wiremockMgr *tools.WireMockManager, isolationID string) (int, error) {
	if wiremockMgr == nil {
		return 0, fmt.Errorf("wiremock manager is nil")
	}

	requestPattern := map[string]interface{}{
		"method":         "POST",
		"urlPathPattern": "/openai/deployments/text-embedding-ada-002/embeddings",
		"headers": map[string]interface{}{
			"X-Genai-Vectorstore-Isolation-Id": map[string]interface{}{
				"equalTo": isolationID,
			},
		},
	}

	count, err := wiremockMgr.VerifyRequest(requestPattern)
	if err != nil {
		return 0, fmt.Errorf("failed to get Ada embedder call count: %w", err)
	}

	return count, nil
}

// GetCallCountByMockID returns the count of requests that were actually handled by the specific mock identified by mockID
// This function queries the request journal and filters entries by the stub mapping ID that handled each request
func GetCallCountByMockID(wiremockMgr *tools.WireMockManager, mockID string) (int, error) {
	if wiremockMgr == nil {
		return 0, fmt.Errorf("wiremock manager is nil")
	}

	// Get all request journal entries from WireMock
	requests, err := wiremockMgr.GetRequests()
	if err != nil {
		return 0, fmt.Errorf("failed to get requests: %w", err)
	}

	// Count requests that were handled by this specific mockID
	// Each request journal entry includes a stubMapping field with the ID of the stub that handled it
	count := 0
	for _, request := range requests {
		if stubMapping, ok := request["stubMapping"].(map[string]interface{}); ok {
			if id, ok := stubMapping["id"].(string); ok && id == mockID {
				count++
			}
		}
	}

	return count, nil
}

// CreateExpectationEmbeddingAdaWithError creates a WireMock stub that returns an error response
// This is useful for testing error handling and retry scenarios
// Returns the unique mapping ID which can be used to delete this specific expectation
func CreateExpectationEmbeddingAdaWithError(wiremockMgr *tools.WireMockManager, isolationID string, statusCode int) (string, error) {
	if wiremockMgr == nil {
		return "", fmt.Errorf("wiremock manager is nil")
	}

	embedding := CreateAdaEmbeddingVector()
	stub := map[string]interface{}{
		"request":  CreateAdaRequestPattern(isolationID),
		"response": CreateAdaStubResponse(embedding, statusCode, nil),
	}
	return wiremockMgr.CreateStub(stub["request"], stub["response"])
}

// CreateExpectationEmbeddingAdaWithDelay creates a WireMock stub that returns a successful response
// but with a configurable delay. This is useful for testing timeout and retry scenarios.
// The delay parameter is specified in milliseconds.
// Returns the unique mapping ID which can be used to delete this specific expectation
func CreateExpectationEmbeddingAdaWithDelay(wiremockMgr *tools.WireMockManager, isolationID string, delayMs int) (string, error) {
	if wiremockMgr == nil {
		return "", fmt.Errorf("wiremock manager is nil")
	}

	embedding := CreateAdaEmbeddingVector()
	stub := map[string]interface{}{
		"request":  CreateAdaRequestPattern(isolationID),
		"response": CreateAdaStubResponse(embedding, 200, &delayMs),
	}
	return wiremockMgr.CreateStub(stub["request"], stub["response"])
}

// CreateExpectationEmbeddingAdaWithRetryScenario creates a sequence of WireMock stubs that simulate
// a retry scenario where initial requests fail with specified errors, then eventually succeed.
// This uses WireMock scenarios to create a state machine that transitions through error states.
//
// Parameters:
// - wiremockMgr: The WireMock manager instance
// - isolationID: The isolation ID to match in request headers
// - errorSequence: Slice of error status codes to return in sequence (e.g., []int{503, 503, 429, 429})
//
// Returns:
// - []string: List of mapping IDs for all created stubs (for cleanup)
// - error: Any error that occurred during setup
//
// Example:
//
//	mockIDs, err := CreateExpectationEmbeddingAdaWithRetryScenario(wiremockMgr, "iso-123", []int{503, 503, 429, 429})
//	// First 2 calls return 503, next 2 calls return 429, all subsequent calls return 200
func CreateExpectationEmbeddingAdaWithRetryScenario(wiremockMgr *tools.WireMockManager, isolationID string, errorSequence []int) ([]string, error) {
	if wiremockMgr == nil {
		return nil, fmt.Errorf("wiremock manager is nil")
	}

	if len(errorSequence) == 0 {
		return nil, fmt.Errorf("errorSequence cannot be empty")
	}

	var mockIDs []string
	embedding := CreateAdaEmbeddingVector()
	scenarioName := fmt.Sprintf("retry-scenario-%s", isolationID)
	currentState := "Started"

	// Create stubs for each error in the sequence
	for i, statusCode := range errorSequence {
		nextState := fmt.Sprintf("after-error-%d", i+1)

		stub := map[string]interface{}{
			"scenarioName":          scenarioName,
			"requiredScenarioState": currentState,
			"newScenarioState":      nextState,
			"request":               CreateAdaRequestPattern(isolationID),
			"response":              CreateAdaStubResponse(embedding, statusCode, nil),
		}

		mockID, err := wiremockMgr.CreateMapping(stub)
		if err != nil {
			// Clean up any stubs we've already created
			for _, id := range mockIDs {
				_ = wiremockMgr.DeleteMapping(id)
			}
			return nil, fmt.Errorf("failed to create error stub %d: %w", i, err)
		}
		mockIDs = append(mockIDs, mockID)
		currentState = nextState
	}

	// Create final success stub that matches in the last state
	successStub := map[string]interface{}{
		"scenarioName":          scenarioName,
		"requiredScenarioState": currentState,
		// No newScenarioState - stays in this state for all subsequent requests
		"request":  CreateAdaRequestPattern(isolationID),
		"response": CreateAdaStubResponse(embedding, 200, nil),
	}

	successMockID, err := wiremockMgr.CreateMapping(successStub)
	if err != nil {
		// Clean up any stubs we've already created
		for _, id := range mockIDs {
			_ = wiremockMgr.DeleteMapping(id)
		}
		return nil, fmt.Errorf("failed to create success stub: %w", err)
	}
	mockIDs = append(mockIDs, successMockID)

	return mockIDs, nil
}

// CreateExpectationEmbeddingAdaWithTimeoutRetryScenario creates WireMock stubs using scenarios to simulate
// timeout and retry behavior for testing query endpoint retry scenarios.
// This creates a scenario where the first call times out (with delay > timeout) but subsequent retries succeed.
//
// The scenario works as follows:
// - First call: In "Started" state, responds with specified delay (typically > timeout), transitions to "Success"
// - Second call: In "Success" state, responds immediately without delay
//
// Parameters:
// - wiremockMgr: The WireMock manager instance
// - isolationID: The isolation ID to match in request headers
// - delayMs: Delay in milliseconds for the first call (should exceed the query timeout to simulate timeout)
//
// Returns two mapping IDs: [timeoutMockID, successMockID] which can be used to track and delete expectations
func CreateExpectationEmbeddingAdaWithTimeoutRetryScenario(wiremockMgr *tools.WireMockManager, isolationID string, delayMs int) ([]string, error) {
	if wiremockMgr == nil {
		return nil, fmt.Errorf("wiremock manager is nil")
	}

	embedding := CreateAdaEmbeddingVector()
	scenarioName := fmt.Sprintf("retry-scenario-%s", isolationID)
	mockIDs := make([]string, 0, 2)

	// First stub: Initial state with delay (causes timeout)
	timeoutHeaders := CreateGenAIGatewayHeaders()
	timeoutHeaders["X-Genai-Gateway-Retry-Count"] = "0" // Override retry count for first call

	timeoutStub := map[string]interface{}{
		"scenarioName":          scenarioName,
		"requiredScenarioState": "Started",
		"newScenarioState":      "Success",
		"request":               CreateAdaRequestPattern(isolationID),
		"response": map[string]interface{}{
			"status":                 200,
			"fixedDelayMilliseconds": delayMs,
			"headers":                timeoutHeaders,
			"jsonBody":               CreateEmbeddingResponseBody(embedding),
		},
	}

	timeoutMockID, err := wiremockMgr.CreateMapping(timeoutStub)
	if err != nil {
		return nil, fmt.Errorf("failed to create timeout stub: %w", err)
	}
	mockIDs = append(mockIDs, timeoutMockID)

	// Second stub: Success state without delay
	successStub := map[string]interface{}{
		"scenarioName":          scenarioName,
		"requiredScenarioState": "Success",
		"request":               CreateAdaRequestPattern(isolationID),
		"response": map[string]interface{}{
			"status":   200,
			"headers":  CreateGenAIGatewayHeaders(),
			"jsonBody": CreateEmbeddingResponseBody(embedding),
		},
	}

	successMockID, err := wiremockMgr.CreateMapping(successStub)
	if err != nil {
		// Clean up the first mock if second fails
		_ = DeleteExpectationIfExist(wiremockMgr, timeoutMockID)
		return nil, fmt.Errorf("failed to create success stub: %w", err)
	}
	mockIDs = append(mockIDs, successMockID)

	return mockIDs, nil
}

// CreateExpectationEmbeddingAdaWithoutGatewayHeaders creates a WireMock stub for the OpenAI
// text-embedding-ada-002 endpoint that does NOT include GenAI Gateway tracing headers.
// This is used in tests that verify the service correctly applies default values when
// gateway headers are missing from the embedder response.
// Returns the unique mapping ID which can be used to delete this specific expectation.
func CreateExpectationEmbeddingAdaWithoutGatewayHeaders(wiremockMgr *tools.WireMockManager, isolationID string) (string, error) {
	if wiremockMgr == nil {
		return "", fmt.Errorf("wiremock manager is nil")
	}

	embedding := CreateAdaEmbeddingVector()
	responseHeaders := map[string]interface{}{
		"Content-Type": "application/json",
	}

	stub := map[string]interface{}{
		"request": CreateAdaRequestPattern(isolationID),
		"response": map[string]interface{}{
			"status":   200,
			"headers":  responseHeaders,
			"jsonBody": CreateEmbeddingResponseBody(embedding),
		},
	}

	return wiremockMgr.CreateStub(stub["request"], stub["response"])
}

// CreateExpectationUsageDataEndpoint creates a WireMock stub for PDC usage data endpoint
// that accepts POST requests with usage metrics payload
// Returns the unique mapping ID which can be used to delete this specific expectation
func CreateExpectationUsageDataEndpoint(wiremockMgr *tools.WireMockManager) (string, error) {
	if wiremockMgr == nil {
		return "", fmt.Errorf("wiremock manager is nil")
	}

	stub := map[string]interface{}{
		"request": map[string]interface{}{
			"method":         "POST",
			"urlPathPattern": "/prweb/PRRestService/.*/PegaUVU/v1/UsageDataFile",
			"headers": map[string]interface{}{
				"Content-Type": map[string]interface{}{
					"equalTo": "application/octet-stream",
				},
			},
		},
		"response": map[string]interface{}{
			"status":  200,
			"headers": map[string]string{"Content-Type": "application/json"},
			"body":    "6afc2689-fb90-4340-bcb2-2cae8110046f",
		},
	}

	return wiremockMgr.CreateStub(stub["request"], stub["response"])
}

// GetUsageDataEndpointCallCount returns the count of requests made to the PDC usage data endpoint
func GetUsageDataEndpointCallCount(wiremockMgr *tools.WireMockManager) (int, error) {
	if wiremockMgr == nil {
		return 0, fmt.Errorf("wiremock manager is nil")
	}

	requestPattern := map[string]interface{}{
		"method":         "POST",
		"urlPathPattern": "/prweb/PRRestService/.*/PegaUVU/v1/UsageDataFile",
	}

	count, err := wiremockMgr.VerifyRequest(requestPattern)
	if err != nil {
		return 0, fmt.Errorf("failed to get usage data endpoint call count: %w", err)
	}

	return count, nil
}

// GetUsageDataRequests retrieves all requests made to the PDC usage data endpoint
// Returns the request bodies as a slice of byte arrays
func GetUsageDataRequests(wiremockMgr *tools.WireMockManager) ([][]byte, error) {
	if wiremockMgr == nil {
		return nil, fmt.Errorf("wiremock manager is nil")
	}

	// Get all request journal entries from WireMock
	requests, err := wiremockMgr.GetRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to get requests: %w", err)
	}

	var usageDataRequests [][]byte

	// Filter requests that match the PDC usage data endpoint pattern
	for _, request := range requests {
		if req, ok := request["request"].(map[string]interface{}); ok {
			if url, ok := req["url"].(string); ok {
				// Check if URL matches the pattern for PDC usage data endpoint
				if strings.Contains(url, "/PegaUVU/v1/UsageDataFile") {
					if body, ok := req["body"].(string); ok {
						usageDataRequests = append(usageDataRequests, []byte(body))
					}
				}
			}
		}
	}

	return usageDataRequests, nil
}
