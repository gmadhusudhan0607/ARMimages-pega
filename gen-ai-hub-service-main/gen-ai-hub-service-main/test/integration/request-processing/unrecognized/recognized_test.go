//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package unrecognized_test

import (
	"fmt"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions/max_tokens"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tests SVC:", Ordered, func() {

	var err error
	var testID string
	var testWireMockExpectations []string // Track WireMock mappings for cleanup

	BeforeAll(func() {
		Expect(err).To(BeNil())
	})

	AfterAll(func() {
		// Cleanup WireMock mappings - ignore 404 errors as mappings may have been cleared by reset
		for _, mappingId := range testWireMockExpectations {
			err := DeleteWireMockExpectation(mockServerURL, mappingId)
			Expect(err).To(BeNil())
		}
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
		// Reset WireMock to clear any previous requests/mappings
		err := ResetWireMockServer(mockServerURL)
		Expect(err).To(BeNil())

		// Recreate monitoring endpoint expectation after reset
		err = CreateMonitoringEndpointExpectation(mockServerURL, monitoringEventsPath)
		Expect(err).To(BeNil())

		// Recreate mapping and defaults endpoint expectations after reset
		err = CreateMappingEndpointExpectation(mockServerURL, mappingsEndpointPath, mappingEndpointFile)
		Expect(err).To(BeNil())
		err = CreateDefaultsEndpointExpectation(mockServerURL, defaultsEndpointPath, defaultsEndpointFile)
		Expect(err).To(BeNil())
	})

	_ = Context("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED", func() {

		_ = Context("Recognized chat models", func() {

			It("when calling model gpt-35-turbo, max_token must be set to fixed value if max_token was NOT provided in original request", func() {
				// Create WireMock expectation that validates max_tokens=1022 is added to the request
				expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"
				expectedModelName := "gpt-35-turbo"
				expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, 1022, 50)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

				// Call gpt-35-turbo model with simple request but without max_tokens in request
				requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
				requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
				max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

				// Get the expected maximum output tokens from model specifications
				expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")

				// Check all required metrics after calling service:
				// - genai_request_duration_ms (must be > 0)
				// - genai_gateway_output_tokens_requested (must be 0/not set since no max_tokens in original request)
				// - genai_gateway_output_tokens_adjusted (must be 1022)
				// - genai_gateway_output_tokens_maximum (must equal model's max from specs)
				// - genai_gateway_output_tokens_used (must be > 0)
				max_tokens.CheckMetricsFixed(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", -1, expectedMaxOutputTokens, 1022)

				// Verify that the WireMock expectation was matched
				err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
				Expect(err).To(BeNil())
			})

			It("when calling model gpt-35-turbo, max_token must be not changed if max_token was provided in original request", func() {
				// Define the original max_tokens value that will be sent in the request
				originalMaxTokens := 500
				expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"
				expectedModelName := "gpt-35-turbo"

				// Create WireMock expectation that validates max_tokens remains at original value (500)
				expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, originalMaxTokens, 50)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

				// Call gpt-35-turbo model with request that includes max_tokens
				requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
				requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(originalMaxTokens).Build()
				max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

				// Get the expected maximum output tokens from model specifications
				expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")

				// Check all required metrics after calling service:
				// - genai_request_duration_ms (must be > 0)
				// - genai_gateway_output_tokens_requested (must be 500 since max_tokens was in original request)
				// - genai_gateway_output_tokens_adjusted (must be 500 - no adjustment)
				// - genai_gateway_output_tokens_maximum (must equal model's max from specs)
				// - genai_gateway_output_tokens_used (must be > 0)
				max_tokens.CheckMetricsFixed(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", originalMaxTokens, expectedMaxOutputTokens, float64(originalMaxTokens))

				// Verify that the WireMock expectation was matched
				err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
				Expect(err).To(BeNil())
			})

			XIt("when calling model gpt-4o, max_token must be set to fixed value if max_token was NOT provided in original request", func() {
				// Create WireMock expectation that validates max_tokens=1022 is added to the request
				expectedPath := "/openai/deployments/gpt-4o/chat/completions"
				expectedModelName := "gpt-4o"

				// FIXED strategy uses a hardcoded value of 1022 for all models
				expectedFixedMaxTokens := 1022

				expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, expectedFixedMaxTokens, 50)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

				// Call gpt-4o model with simple request but without max_tokens in request
				requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-01", svcBaseURL)
				requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-4o", "2024-05-13", "2024-10-21").WithoutMaxTokens().Build()
				max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

				// Get the expected maximum output tokens from model specifications
				expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-4o")

				// Check all required metrics
				max_tokens.CheckMetricsFixed(metricsUrl, testID, "gpt-4o", "gpt-4o", -1, expectedMaxOutputTokens, float64(expectedFixedMaxTokens))

				// Verify that the WireMock expectation was matched
				err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
				Expect(err).To(BeNil())
			})

			XIt("when calling model gpt-4o, max_token must be not changed if max_token was provided in original request", func() {
				// Define the original max_tokens value that will be sent in the request
				originalMaxTokens := 750
				expectedPath := "/openai/deployments/gpt-4o/chat/completions"
				expectedModelName := "gpt-4o"

				// Create WireMock expectation that validates max_tokens remains at original value
				expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, originalMaxTokens, 50)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

				// Call gpt-4o model with request that includes max_tokens
				requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-4o/chat/completions?api-version=2024-02-01", svcBaseURL)
				requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-4o", "2024-05-13", "2024-10-21").WithMaxTokens(originalMaxTokens).Build()
				max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

				// Get the expected maximum output tokens from model specifications
				expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-4o")

				// Check all required metrics
				max_tokens.CheckMetricsFixed(metricsUrl, testID, "gpt-4o", "gpt-4o", originalMaxTokens, expectedMaxOutputTokens, float64(originalMaxTokens))

				// Verify that the WireMock expectation was matched
				err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
				Expect(err).To(BeNil())
			})

		})

		_ = Context("Embedding models", func() {

			It("when calling text-embedding-ada-002, service correctly does NOT add max_tokens to embedding requests", func() {
				isolationID := testID
				modelName := "text-embedding-ada-002"
				expectedPath := "/openai/deployments/text-embedding-ada-002/embeddings"

				// Create an embedding request body without max_tokens
				originalRequestBody := CreateEmbeddingRequestBody("This is a test text to embed", nil)

				// Expected response from backend
				expectedResponse := `{"object": "list", "data": [{"object": "embedding", "index": 0, "embedding": [0.1, 0.2, 0.3]}], "model": "text-embedding-ada-002", "usage": {"prompt_tokens": 7, "total_tokens": 7}}`

				// Set up WireMock expectation for the backend call with the original body (unchanged)
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, originalRequestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service with original request (without max_tokens)
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, originalRequestBody, expectedResponse)

				// Verify request body was NOT modified (max_tokens was NOT added)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, originalRequestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Note: Embedding models may not generate model recognition metrics like chat completion models
			})

			It("when calling text-embedding-ada-002 with max_tokens, service forwards request unchanged and returns response unchanged", func() {
				isolationID := testID
				modelName := "text-embedding-ada-002"
				expectedPath := "/openai/deployments/text-embedding-ada-002/embeddings"
				originalMaxTokens := 100

				// Create an embedding request body with max_tokens
				requestBody := CreateEmbeddingRequestBody("Another test text for embedding", &originalMaxTokens)

				// Expected response from backend
				expectedResponse := `{"object": "list", "data": [{"object": "embedding", "index": 0, "embedding": [0.4, 0.5, 0.6]}], "model": "text-embedding-ada-002", "usage": {"prompt_tokens": 6, "total_tokens": 6}}`

				// Set up WireMock expectation for the backend call
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, requestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, requestBody, expectedResponse)

				// Verify request body was forwarded unchanged (including max_tokens if present)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, requestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Note: Embedding models may not generate model recognition metrics like chat completion models
			})

		})

	})

})
