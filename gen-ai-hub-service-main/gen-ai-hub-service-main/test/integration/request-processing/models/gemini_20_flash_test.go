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

var _ = Describe("Tests SVC Gemini 2.0 Flash Model Recognition (REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED):", Ordered, func() {

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
	})

	_ = Context("gemini-2.0-flash model tests", func() {

		It("when calling model gemini-2.0-flash, model should be RECOGNIZED (not unrecognized) in metrics", func() {
			// This is the PRIMARY test to reproduce the production issue
			// Production shows: genai_gateway_model_recognition_total{status="unrecognized"}
			// Expected after fix: genai_gateway_model_recognition_total{status="recognized"}

			// Create WireMock expectation for Gemini endpoint
			expectedPath := "/google/deployments/gemini-2.0-flash/chat/completions"
			expectedModelName := "gemini-2.0-flash"
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, 1022, 128)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call gemini-2.0-flash model with simple request without max_tokens
			requestUrl := fmt.Sprintf("%s/google/deployments/gemini-2.0-flash/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("google/gemini-2.0-flash", "001", "").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 8192 // From gemini-2.0.yaml specs

			// Check all required metrics after calling service:
			// PRIMARY CHECK: Model should be RECOGNIZED, not unrecognized
			// - genai_gateway_model_recognition_total{status="recognized"} should increment
			// - genai_gateway_output_tokens_maximum (must be 8192 from model specs)
			// - genai_gateway_output_tokens_adjusted (must be 1022)
			// - genai_gateway_output_tokens_used (must be > 0)
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "gemini-2.0-flash", "gemini-2.0-flash", -1, expectedMaxOutputTokens, 1022)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when calling model gemini-2.0-flash, max_token must be set to fixed value if max_token was NOT provided in original request", func() {
			// Create WireMock expectation that validates max_tokens=1022 is added to the request
			expectedPath := "/google/deployments/gemini-2.0-flash/chat/completions"
			expectedModelName := "gemini-2.0-flash"
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, 1022, 128)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call gemini-2.0-flash model with simple request but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/google/deployments/gemini-2.0-flash/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("google/gemini-2.0-flash", "001", "").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 8192

			// Check all required metrics after calling service
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "gemini-2.0-flash", "gemini-2.0-flash", -1, expectedMaxOutputTokens, 1022)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when calling model gemini-2.0-flash, max_token must not be changed if max_token was provided in original request", func() {
			originalMaxTokens := 512
			expectedPath := "/google/deployments/gemini-2.0-flash/chat/completions"
			expectedModelName := "gemini-2.0-flash"
			// Create WireMock expectation that validates max_tokens=512 remains unchanged
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, originalMaxTokens, 128)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call gemini-2.0-flash model with max_tokens in request
			requestUrl := fmt.Sprintf("%s/google/deployments/gemini-2.0-flash/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("google/gemini-2.0-flash", "001", "").WithMaxTokens(originalMaxTokens).Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 8192

			// Check all required metrics after calling service
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "gemini-2.0-flash", "gemini-2.0-flash", originalMaxTokens, expectedMaxOutputTokens, float64(originalMaxTokens))

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when calling model gemini-2.0-flash with streaming, max_token must NOT be changed if max_token was NOT provided in original request (OutputTokensAdjustmentStreams=false)", func() {
			// Create WireMock expectation that validates max_tokens is NOT added to the streaming request
			expectedPath := "/google/deployments/gemini-2.0-flash/chat/completions"
			expectedModelName := "gemini-2.0-flash"
			mapping, err := max_tokens.CreateWireMockMaxTokensStreamingExpectation(mockServerURL, testID, expectedPath, expectedModelName, 0)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping.Id)

			// Call gemini-2.0-flash model with streaming enabled but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/google/deployments/gemini-2.0-flash/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("google/gemini-2.0-flash", "001", "").WithoutMaxTokens().WithStreaming(true).Build()
			headers := map[string]string{"Accept": "text/event-stream"}
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, headers)

			expectedMaxOutputTokens := 8192

			// Check all required metrics for streaming after calling service
			max_tokens.CheckMetricsStreaming(metricsUrl, testID, "gemini-2.0-flash", expectedModelName, 0, expectedMaxOutputTokens, 1022)

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})
	})

})
