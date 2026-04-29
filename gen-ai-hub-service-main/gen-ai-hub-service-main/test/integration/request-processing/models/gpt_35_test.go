//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package models_test

import (
	"fmt"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions/max_tokens"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tests SVC (REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED):", Ordered, func() {

	var err error
	var testID string
	var testWireMockExpectations []string // Track WireMock mappings for cleanup

	BeforeAll(func() {
		Expect(err).To(BeNil())
	})

	AfterAll(func() {
		// Cleanup WireMock mappings
		for _, mappingId := range testWireMockExpectations {
			err := DeleteWireMockExpectation(mockServerURL, mappingId)
			Expect(err).To(BeNil())

			// Recreate mapping and defaults endpoint expectations after reset
			err = CreateMappingEndpointExpectation(mockServerURL, mappingsEndpointPath, mappingEndpointFile)
			Expect(err).To(BeNil())
			err = CreateDefaultsEndpointExpectation(mockServerURL, defaultsEndpointPath, defaultsEndpointFile)
			Expect(err).To(BeNil())
		}
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
		// Reset WireMock to clear any previous requests/mappings
		err := ResetWireMockServer(mockServerURL)
		Expect(err).To(BeNil())
	})

	_ = Context("gpt-35-turbo model tests", func() {

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

			expectedMaxOutputTokens := 4096

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
			originalMaxTokens := 512
			expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"
			expectedModelName := "gpt-35-turbo"
			// Create WireMock expectation that validates max_tokens=512 remains unchanged
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, originalMaxTokens, 50)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call gpt-35-turbo model with simple request but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(originalMaxTokens).Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 4096

			// Check all required metrics after calling service:
			// - genai_request_duration_ms (must be > 0)
			// - genai_gateway_output_tokens_requested (must be present with original value since max_tokens was in original request)
			// - genai_gateway_output_tokens_adjusted (must be present - in FIXED strategy, when max_tokens provided, it should remain unchanged)
			// - genai_gateway_output_tokens_maximum (must equal model's max from specs)
			// - genai_gateway_output_tokens_used (must be > 0)
			// - genai_gateway_output_tokens_adjusted_wasted_total (should be > 0 if original max_tokens > actual usage)
			// - genai_gateway_output_tokens_requested_wasted_total (should be > 0 if original max_tokens > actual usage)
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", originalMaxTokens, expectedMaxOutputTokens, float64(originalMaxTokens))

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, "/openai/deployments/gpt-35-turbo-1106/chat/completions", 1)
			Expect(err).To(BeNil())
		})

		It("when calling model gpt-35-turbo with streaming, max_token must NOT be changed if max_token was NOT provided in original request (OutputTokensAdjustmentStreams=false)", func() {
			// Create WireMock expectation that validates max_tokens is NOT added to the streaming request
			expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"
			expectedModelName := "gpt-35-turbo"
			mapping, err := max_tokens.CreateWireMockMaxTokensStreamingExpectation(mockServerURL, testID, expectedPath, expectedModelName, 0)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping.Id)

			// Call gpt-35-turbo model with streaming enabled but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().WithStreaming(true).Build()
			headers := map[string]string{"Accept": "text/event-stream"}
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, headers)

			expectedMaxOutputTokens := 4096

			// Check all required metrics for streaming after calling service:
			// - genai_request_duration_ms (must be > 0)
			// - genai_gateway_output_tokens_requested (flexible for streaming - may or may not be present)
			// - genai_gateway_output_tokens_adjusted (should be 1022 since default adjustment for streaming)
			// - genai_gateway_output_tokens_maximum (must equal model's max from specs)
			// - genai_gateway_output_tokens_used (must be > 0)
			max_tokens.CheckMetricsStreaming(metricsUrl, testID, "gpt-35-turbo", expectedModelName, 0, expectedMaxOutputTokens, 1022)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when calling model gpt-35-turbo with streaming, max_token must NOT be changed if max_token was provided in original request (OutputTokensAdjustmentStreams=false)", func() {
			originalMaxTokens := 768
			expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"
			// Create WireMock expectation that validates max_tokens=768 remains unchanged in streaming request
			// Since OutputTokensAdjustmentStreams=false, the original max_tokens should pass through unchanged
			mapping, err := max_tokens.CreateWireMockMaxTokensStreamingExpectation(mockServerURL, testID, expectedPath, "gpt-35-turbo", originalMaxTokens)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping.Id)

			// Call gpt-35-turbo model with streaming enabled and max_tokens=768 in request
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(originalMaxTokens).WithStreaming(true).Build()
			headers := map[string]string{"Accept": "text/event-stream"}
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, headers)

			expectedMaxOutputTokens := 4096

			// Check all required metrics after calling service:
			// - genai_request_duration_ms (must be > 0)
			// - genai_gateway_output_tokens_requested (must be present with original value since max_tokens was in original request)
			// - genai_gateway_output_tokens_adjusted (flexible for streaming with original max_tokens)
			// - genai_gateway_output_tokens_maximum (must equal model's max from specs)
			// - genai_gateway_output_tokens_used (must be > 0)
			// Note: For streaming with OutputTokensAdjustmentStreams=false, original max_tokens should pass through unchanged
			max_tokens.CheckMetricsStreaming(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", originalMaxTokens, expectedMaxOutputTokens, 0)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})
	})

})
