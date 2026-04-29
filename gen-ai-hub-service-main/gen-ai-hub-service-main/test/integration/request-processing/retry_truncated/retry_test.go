//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package retry_truncated_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	max_tokens "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions/max_tokens"
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

		// Recreate mapping and defaults endpoint expectations after reset
		err = CreateMappingEndpointExpectation(mockServerURL, mappingsEndpointPath, mappingEndpointFile)
		Expect(err).To(BeNil())
		err = CreateDefaultsEndpointExpectation(mockServerURL, defaultsEndpointPath, defaultsEndpointFile)
		Expect(err).To(BeNil())
		Expect(err).To(BeNil())
	})

	_ = Context("Retry on Truncation", func() {

		It("when calling gpt-35-turbo with max_tokens that causes truncation, should retry without max_tokens", func() {
			// Define test parameters
			maxTokens := 50 // Small value that will cause truncation
			expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"
			expectedModelName := "gpt-35-turbo"

			// Create WireMock expectation for initial request that validates max_tokens=50 and returns truncated response
			truncatedMapping, err := max_tokens.CreateWireMockMaxTokensTruncatedExpectation(mockServerURL, testID, maxTokens)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, truncatedMapping.Id)

			// Create WireMock expectation for retry request that validates no max_tokens and returns complete response
			retryMapping, err := max_tokens.CreateWireMockRetryExpectation(mockServerURL, testID, 0)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, retryMapping.Id)

			// Call gpt-35-turbo model with max_tokens that will trigger truncation
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder(expectedModelName, "1106", "2024-10-21").WithMaxTokens(maxTokens).Build()
			h := map[string]string{
				"Accept-Encoding": "identity",
			}
			httpResponse, respBody := max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, h)
			Expect(httpResponse.StatusCode).To(Equal(http.StatusOK))
			assertIsStopFinishReason(respBody)

			// Verify both requests were made as expected:
			// 1. Initial request with max_tokens should be matched once (returns truncated response)
			err = max_tokens.VerifyWireMockTruncatedExpectation(mockServerURL, testID, expectedPath+".*", 1)
			Expect(err).To(BeNil())

			// 2. Retry request without max_tokens should be matched once (returns complete response)
			err = max_tokens.VerifyWireMockRetryExpectation(mockServerURL, testID, expectedPath+".*", 1)
			Expect(err).To(BeNil())

			// Check that retry metrics are correctly updated:
			// - genai_gateway_retries_total (must be 1)
			// - genai_gateway_retry_reason (must be "max_tokens_exceeded")
			CheckRetryMetrics(metricsUrl, expectedModelName, testID, 1, "max_tokens_exceeded")
		})

		It("when calling gpt-35-turbo with streaming and max_tokens that causes truncation, should NOT retry because streaming retry is disabled by default", func() {
			// Define test parameters
			maxTokens := 50 // Small value that will cause truncation
			expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"
			expectedModelName := "gpt-35-turbo"

			// Create WireMock expectation for streaming request that validates max_tokens=50 and returns truncated streaming response
			truncatedStreamingMapping, err := max_tokens.CreateWireMockMaxTokensTruncatedStreamingExpectation(mockServerURL, testID, maxTokens)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, truncatedStreamingMapping.Id)

			// Create WireMock expectation for potential retry (this should NOT be matched since streaming retry is disabled)
			retryStreamingMapping, err := max_tokens.CreateWireMockMaxTokensStreamingExpectation(mockServerURL, testID, "/openai/deployments/gpt-35-turbo-1106/chat/completions", "gpt-35-turbo", 0)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, retryStreamingMapping.Id)

			// Call gpt-35-turbo model with streaming enabled and max_tokens that will trigger truncation
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder(expectedModelName, "1106", "2024-10-21").WithMaxTokens(maxTokens).WithStreaming(true).Build()
			headers := map[string]string{"Accept": "text/event-stream"}
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, headers)

			// Verify only the initial truncated request was made (no retry for streaming):
			// 1. Initial streaming request with max_tokens should be matched once (returns truncated response)
			err = max_tokens.VerifyWireMockTruncatedExpectation(mockServerURL, testID, expectedPath+".*", 1)
			Expect(err).To(BeNil())

			// 2. Retry streaming request should NOT be matched (0 times) because streaming retry is disabled by default
			err = max_tokens.VerifyWireMockRetryExpectation(mockServerURL, testID, expectedPath+".*", 0)
			Expect(err).To(BeNil())

			// Check that NO retry metrics are updated for streaming requests:
			// - genai_gateway_retries_total (must be 0 or not present)
			CheckNoRetryMetrics(metricsUrl, expectedModelName, testID)
		})

		It("when calling gpt-35-turbo with max_tokens that does NOT cause truncation, should NOT retry", func() {
			// Define test parameters
			maxTokens := 200 // Large enough value that won't cause truncation
			expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions.*"
			expectedModelName := "gpt-35-turbo"

			// Create WireMock expectation for request that validates max_tokens=200 and returns complete response
			p := max_tokens.WireMockMappingParameters{
				WiremockURL:       mockServerURL,
				IsolationID:       testID,
				UrlPattern:        expectedPath,
				ModelName:         expectedModelName,
				ExpectedMaxTokens: maxTokens,
				CompletionTokens:  50,
			}
			//completeMapping, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, maxTokens, 50)
			completeMapping, err := p.CreateExpectation()
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, completeMapping.Id)

			// Create WireMock expectation for potential retry (this should NOT be matched since no truncation occurs)
			retryMapping, err := max_tokens.CreateWireMockRetryExpectation(mockServerURL, testID, 0)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, retryMapping.Id)

			// Call gpt-35-turbo model with max_tokens that will NOT trigger truncation
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder(expectedModelName, "1106", "2024-02-01").WithMaxTokens(maxTokens).Build()
			h := map[string]string{
				"Accept-Encoding": "identity",
			}
			httpResponse, respBody := max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, h)
			Expect(httpResponse.StatusCode).To(Equal(http.StatusOK))

			assertIsStopFinishReason(respBody)

			// Verify only the initial request was made (no retry needed):
			// 1. Initial request with max_tokens should be matched once (returns complete response)
			err = max_tokens.VerifyWireMockExpectationId(mockServerURL, completeMapping.Id, 1)
			Expect(err).To(BeNil())

			// 2. Retry request should NOT be matched (0 times) because there was no truncation
			err = max_tokens.VerifyWireMockExpectationId(mockServerURL, retryMapping.Id, 0)
			Expect(err).To(BeNil())

			// Check that NO retry metrics are updated since no retry was needed:
			// - genai_gateway_retries_total (must be 0 or not present)
			CheckNoRetryMetrics(metricsUrl, expectedModelName, testID)
		})
	})
})

func assertIsStopFinishReason(respBody []byte) {
	var payload ChatCompletionsResponseType
	err := json.Unmarshal(respBody, &payload)
	Expect(err).To(BeNil())
	Expect(payload.Choices).To(HaveLen(1))
	Expect(payload.Choices[0].FinishReason).To(Equal("stop"))
}
