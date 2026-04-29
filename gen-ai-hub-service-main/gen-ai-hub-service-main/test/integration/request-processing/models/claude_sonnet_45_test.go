//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package models_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions/max_tokens"
)

var _ = Describe("Tests SVC Claude Sonnet 4.5 Model Recognition (REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED):", Ordered, func() {

	var err error
	var testID string
	var testWireMockExpectations []string // Track WireMock mappings for cleanup

	BeforeAll(func() {
		Expect(err).To(BeNil())
	})

	AfterAll(func() {
		// Cleanup WireMock mappings
		for _, mappingId := range testWireMockExpectations {
			err := functions.DeleteWireMockExpectation(mockServerURL, mappingId)
			Expect(err).To(BeNil())
		}
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", functions.RandStringRunes(10)))
		// Reset WireMock to clear any previous requests/mappings
		err := functions.ResetWireMockServer(mockServerURL)
		Expect(err).To(BeNil())

		// Recreate mapping and defaults endpoint expectations after reset
		err = functions.CreateMappingEndpointExpectation(mockServerURL, mappingsEndpointPath, mappingEndpointFile)
		Expect(err).To(BeNil())
		err = functions.CreateDefaultsEndpointExpectation(mockServerURL, defaultsEndpointPath, defaultsEndpointFile)
		Expect(err).To(BeNil())
	})

	_ = Context("claude-sonnet-4-5 model tests", func() {

		It("when calling model claude-sonnet-4-5, model should be RECOGNIZED (not unrecognized) in metrics", func() {
			// This is the PRIMARY test to verify model recognition
			// Expected: genai_gateway_model_recognition_total{status="recognized"}

			// Create WireMock expectation for Anthropic endpoint
			expectedPath := "/anthropic/deployments/claude-sonnet-4-5/chat/completions"
			expectedModelName := "claude-sonnet-4-5"
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, 1022, 256)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call claude-sonnet-4-5 model with simple request without max_tokens
			requestUrl := fmt.Sprintf("%s/anthropic/deployments/claude-sonnet-4-5/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("anthropic/claude-sonnet-4-5", "1.0", "").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 64000 // From claude.yaml specs

			// Check all required metrics after calling service:
			// PRIMARY CHECK: Model should be RECOGNIZED, not unrecognized
			// - genai_gateway_model_recognition_total{status="recognized"} should increment
			// - genai_gateway_output_tokens_maximum (must be 64000 from model specs)
			// - genai_gateway_output_tokens_adjusted (must be 1022)
			// - genai_gateway_output_tokens_used (must be > 0)
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "claude-sonnet-4-5", "claude-sonnet-4-5", -1, expectedMaxOutputTokens, 1022)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when calling model claude-sonnet-4-5, max_token must be set to fixed value if max_token was NOT provided in original request", func() {
			// Create WireMock expectation that validates max_tokens=1022 is added to the request
			expectedPath := "/anthropic/deployments/claude-sonnet-4-5/chat/completions"
			expectedModelName := "claude-sonnet-4-5"
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, 1022, 256)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call claude-sonnet-4-5 model with simple request but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/anthropic/deployments/claude-sonnet-4-5/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("anthropic/claude-sonnet-4-5", "1.0", "").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 64000

			// Check all required metrics after calling service
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "claude-sonnet-4-5", "claude-sonnet-4-5", -1, expectedMaxOutputTokens, 1022)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when calling model claude-sonnet-4-5, max_token must not be changed if max_token was provided in original request", func() {
			originalMaxTokens := 512
			expectedPath := "/anthropic/deployments/claude-sonnet-4-5/chat/completions"
			expectedModelName := "claude-sonnet-4-5"
			// Create WireMock expectation that validates max_tokens=512 remains unchanged
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, originalMaxTokens, 256)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call claude-sonnet-4-5 model with max_tokens in request
			requestUrl := fmt.Sprintf("%s/anthropic/deployments/claude-sonnet-4-5/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("anthropic/claude-sonnet-4-5", "1.0", "").WithMaxTokens(originalMaxTokens).Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 64000

			// Check all required metrics after calling service
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "claude-sonnet-4-5", "claude-sonnet-4-5", originalMaxTokens, expectedMaxOutputTokens, float64(originalMaxTokens))

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when calling model claude-sonnet-4-5 with streaming, max_token must NOT be changed if max_token was NOT provided in original request (OutputTokensAdjustmentStreams=false)", func() {
			// Create WireMock expectation that validates max_tokens is NOT added to the streaming request
			expectedPath := "/anthropic/deployments/claude-sonnet-4-5/chat/completions"
			expectedModelName := "claude-sonnet-4-5"
			mapping, err := max_tokens.CreateWireMockMaxTokensStreamingExpectation(mockServerURL, testID, expectedPath, expectedModelName, 0)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping.Id)

			// Call claude-sonnet-4-5 model with streaming enabled but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/anthropic/deployments/claude-sonnet-4-5/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("anthropic/claude-sonnet-4-5", "1.0", "").WithoutMaxTokens().WithStreaming(true).Build()
			headers := map[string]string{"Accept": "text/event-stream"}
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, headers)

			expectedMaxOutputTokens := 64000

			// Check all required metrics for streaming after calling service
			max_tokens.CheckMetricsStreaming(metricsUrl, testID, "claude-sonnet-4-5", expectedModelName, 0, expectedMaxOutputTokens, 1022)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when calling model claude-sonnet-4-5 with streaming, max_token must NOT be changed if max_token was provided in original request (OutputTokensAdjustmentStreams=false)", func() {
			originalMaxTokens := 768
			expectedPath := "/anthropic/deployments/claude-sonnet-4-5/chat/completions"
			expectedModelName := "claude-sonnet-4-5"
			// Create WireMock expectation that validates max_tokens=768 remains unchanged in streaming request
			// Since OutputTokensAdjustmentStreams=false, the original max_tokens should pass through unchanged
			mapping, err := max_tokens.CreateWireMockMaxTokensStreamingExpectation(mockServerURL, testID, expectedPath, expectedModelName, originalMaxTokens)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping.Id)

			// Call claude-sonnet-4-5 model with streaming enabled and max_tokens=768 in request
			requestUrl := fmt.Sprintf("%s/anthropic/deployments/claude-sonnet-4-5/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("anthropic/claude-sonnet-4-5", "1.0", "").WithMaxTokens(originalMaxTokens).WithStreaming(true).Build()
			headers := map[string]string{"Accept": "text/event-stream"}
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, headers)

			expectedMaxOutputTokens := 64000

			// Check all required metrics after calling service
			max_tokens.CheckMetricsStreaming(metricsUrl, testID, "claude-sonnet-4-5", "claude-sonnet-4-5", originalMaxTokens, expectedMaxOutputTokens, 0)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})
	})
})
