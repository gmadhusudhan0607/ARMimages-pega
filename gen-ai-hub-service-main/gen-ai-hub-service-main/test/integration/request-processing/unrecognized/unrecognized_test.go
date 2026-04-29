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

		_ = Context("Unrecognized models", func() {

			It("when calling fake-model-test without max_tokens, service forwards request unchanged and marks model as unrecognized", func() {
				isolationID := testID
				modelName := "fake-model-test"
				expectedPath := "/openai/deployments/fake-model-test/chat/completions"

				// Create a custom request body without max_tokens
				originalRequestBody := CreateFakeModelRequestBody("Hello, test message!", nil)

				// Expected response from backend
				expectedResponse := `{"id": "chatcmpl-test", "object": "chat.completion", "created": 1700000000, "model": "fake-model-test", "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello! This is a test response."}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 5, "completion_tokens": 8, "total_tokens": 13}}`

				// Set up WireMock expectation for the backend call with the original body (unchanged)
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, originalRequestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service with original request (without max_tokens)
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, originalRequestBody, expectedResponse)

				// Verify request body was NOT modified (max_tokens was NOT added)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, originalRequestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Verify model recognition metric shows "unrecognized"
				max_tokens.CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName)

				// Verify that NO max_tokens metrics are collected
				max_tokens.VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName)
			})

			It("when calling fake-model-test with max_tokens, service forwards request unchanged and marks model as unrecognized", func() {
				isolationID := testID
				modelName := "fake-model-test"
				expectedPath := "/openai/deployments/fake-model-test/chat/completions"
				originalMaxTokens := 100

				// Create a custom request body with max_tokens
				requestBody := CreateFakeModelRequestBody("Another test message with max_tokens!", &originalMaxTokens)

				// Expected response from backend
				expectedResponse := `{"id": "chatcmpl-test2", "object": "chat.completion", "created": 1700000001, "model": "fake-model-test", "choices": [{"index": 0, "message": {"role": "assistant", "content": "This is another test response with max_tokens!"}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 8, "completion_tokens": 10, "total_tokens": 18}}`

				// Set up WireMock expectation for the backend call
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, requestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, requestBody, expectedResponse)

				// Verify request body was forwarded unchanged (including max_tokens)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, requestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Verify model recognition metric shows "unrecognized"
				max_tokens.CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName)

				// Verify that NO max_tokens metrics are collected
				max_tokens.VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName)
			})

			It("when calling unrecognized-llm without max_tokens, service forwards request unchanged and marks model as unrecognized", func() {
				isolationID := testID
				modelName := "unrecognized-llm"
				expectedPath := "/openai/deployments/unrecognized-llm/chat/completions"

				// Create a custom request body without max_tokens
				originalRequestBody := CreateUnrecognizedLLMRequestBody("Hello from unrecognized-llm!", nil)

				// Expected response from backend
				expectedResponse := `{"id": "chatcmpl-unrecognized", "object": "chat.completion", "created": 1700000010, "model": "unrecognized-llm", "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello! This is a response from unrecognized-llm."}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 6, "completion_tokens": 9, "total_tokens": 15}}`

				// Set up WireMock expectation for the backend call with the original body (unchanged)
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, originalRequestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service with original request (without max_tokens)
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, originalRequestBody, expectedResponse)

				// Verify request body was NOT modified (max_tokens was NOT added)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, originalRequestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Verify model recognition metric shows "unrecognized"
				max_tokens.CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName)

				// Verify that NO max_tokens metrics are collected
				max_tokens.VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName)
			})

			It("when calling unrecognized-llm with max_tokens, service forwards request unchanged and marks model as unrecognized", func() {
				isolationID := testID
				modelName := "unrecognized-llm"
				expectedPath := "/openai/deployments/unrecognized-llm/chat/completions"
				originalMaxTokens := 120

				// Create a custom request body with max_tokens
				requestBody := CreateUnrecognizedLLMRequestBody("Another message for unrecognized-llm with max_tokens!", &originalMaxTokens)

				// Expected response from backend
				expectedResponse := `{"id": "chatcmpl-unrecognized2", "object": "chat.completion", "created": 1700000020, "model": "unrecognized-llm", "choices": [{"index": 0, "message": {"role": "assistant", "content": "This is another response from unrecognized-llm with max_tokens!"}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 9, "completion_tokens": 12, "total_tokens": 21}}`

				// Set up WireMock expectation for the backend call
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, requestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, requestBody, expectedResponse)

				// Verify request body was forwarded unchanged (including max_tokens)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, requestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Verify model recognition metric shows "unrecognized"
				max_tokens.CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName)

				// Verify that NO max_tokens metrics are collected
				max_tokens.VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName)
			})

			It("when calling fake-gpt-ultra without max_tokens, service forwards request unchanged and marks model as unrecognized", func() {
				isolationID := testID
				modelName := "fake-gpt-ultra"
				expectedPath := "/openai/deployments/fake-gpt-ultra/chat/completions"

				// Create a custom request body without max_tokens
				originalRequestBody := CreateFakeGptUltraRequestBody("Hello from fake-gpt-ultra!", nil)

				// Expected response from backend
				expectedResponse := `{"id": "chatcmpl-ultra", "object": "chat.completion", "created": 1700000030, "model": "fake-gpt-ultra", "choices": [{"index": 0, "message": {"role": "assistant", "content": "Hello! This is a response from fake-gpt-ultra."}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 5, "completion_tokens": 10, "total_tokens": 15}}`

				// Set up WireMock expectation for the backend call with the original body (unchanged)
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, originalRequestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service with original request (without max_tokens)
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, originalRequestBody, expectedResponse)

				// Verify request body was NOT modified (max_tokens was NOT added)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, originalRequestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Verify model recognition metric shows "unrecognized"
				max_tokens.CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName)

				// Verify that NO max_tokens metrics are collected
				max_tokens.VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName)
			})

			It("when calling fake-gpt-ultra with max_tokens, service forwards request unchanged and marks model as unrecognized", func() {
				isolationID := testID
				modelName := "fake-gpt-ultra"
				expectedPath := "/openai/deployments/fake-gpt-ultra/chat/completions"
				originalMaxTokens := 200

				// Create a custom request body with max_tokens
				requestBody := CreateFakeGptUltraRequestBody("Another message for fake-gpt-ultra with max_tokens!", &originalMaxTokens)

				// Expected response from backend
				expectedResponse := `{"id": "chatcmpl-ultra2", "object": "chat.completion", "created": 1700000040, "model": "fake-gpt-ultra", "choices": [{"index": 0, "message": {"role": "assistant", "content": "This is another response from fake-gpt-ultra with max_tokens!"}, "finish_reason": "stop"}], "usage": {"prompt_tokens": 8, "completion_tokens": 11, "total_tokens": 19}}`

				// Set up WireMock expectation for the backend call
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, requestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, requestBody, expectedResponse)

				// Verify request body was forwarded unchanged (including max_tokens)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, requestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Verify model recognition metric shows "unrecognized"
				max_tokens.CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName)

				// Verify that NO max_tokens metrics are collected
				max_tokens.VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName)
			})

		})

		_ = Context("Unrecognized embedding models", func() {

			It("when calling non-existent-embedding without max_tokens, service forwards request unchanged and marks model as unrecognized", func() {
				isolationID := testID
				modelName := "non-existent-embedding"
				expectedPath := "/openai/deployments/non-existent-embedding/embeddings"

				// Create an embedding request body without max_tokens
				originalRequestBody := CreateNonExistentEmbeddingRequestBody("This is text for non-existent-embedding", nil)

				// Expected response from backend
				expectedResponse := `{"object": "list", "data": [{"object": "embedding", "index": 0, "embedding": [0.7, 0.8, 0.9]}], "model": "non-existent-embedding", "usage": {"prompt_tokens": 7, "total_tokens": 7}}`

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

				// Verify model recognition metric shows "unrecognized"
				max_tokens.CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName)

				// Verify that NO max_tokens metrics are collected
				max_tokens.VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName)
			})

			It("when calling non-existent-embedding with max_tokens, service forwards request unchanged and marks model as unrecognized", func() {
				isolationID := testID
				modelName := "non-existent-embedding"
				expectedPath := "/openai/deployments/non-existent-embedding/embeddings"
				originalMaxTokens := 50

				// Create an embedding request body with max_tokens
				requestBody := CreateNonExistentEmbeddingRequestBody("Another text for non-existent-embedding with max_tokens", &originalMaxTokens)

				// Expected response from backend
				expectedResponse := `{"object": "list", "data": [{"object": "embedding", "index": 0, "embedding": [0.1, 0.4, 0.7]}], "model": "non-existent-embedding", "usage": {"prompt_tokens": 9, "total_tokens": 9}}`

				// Set up WireMock expectation for the backend call
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, expectedPath, requestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service
				requestUrl := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=2024-02-01", svcBaseURL, modelName)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, requestBody, expectedResponse)

				// Verify request body was forwarded unchanged (including max_tokens)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, expectedPath, requestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 1)
				Expect(err).To(BeNil())

				// Verify model recognition metric shows "unrecognized"
				max_tokens.CheckUnrecognizedModelMetric(metricsUrl, isolationID, modelName)

				// Verify that NO max_tokens metrics are collected
				max_tokens.VerifyNoMaxTokensMetrics(metricsUrl, isolationID, modelName)
			})

		})

	})

})
