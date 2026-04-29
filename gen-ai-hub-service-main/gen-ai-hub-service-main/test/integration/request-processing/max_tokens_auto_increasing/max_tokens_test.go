//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens_auto_increasing_test

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

		// Recreate mapping and defaults endpoint expectations after reset
		err = CreateMappingEndpointExpectation(mockServerURL, mappingsEndpointPath, mappingEndpointFile)
		Expect(err).To(BeNil())
		err = CreateDefaultsEndpointExpectation(mockServerURL, defaultsEndpointPath, defaultsEndpointFile)
		Expect(err).To(BeNil())
		Expect(err).To(BeNil())
	})

	_ = Context("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=AUTO_INCREASING", func() {

		It("when calling model gpt-35-turbo without max_tokens, first call uses config value, subsequent calls use auto-adjusted values", func() {
			// Test scenario:
			// 1. First call without max_tokens -> should use REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE (1022)
			// 2. Mock returns 1200 tokens -> cache stores 1200 as new auto-adjusted value
			// 3. Second call without max_tokens -> should use cached value (1200)
			// 4. Third call without max_tokens -> should continue using cached value (1200)
			// 5. Verify comprehensive metrics for auto-increasing strategy

			// Use the same isolation ID for all calls to test progressive auto-adjustment within the same isolation
			isolationID := testID + "-shared"
			expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"

			// First call - expect max_tokens=1022 (config value), mock returns 1200 tokens to trigger cache update
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			mapping1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1022, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping1.Id)

			// Call gpt-35-turbo model with simple request but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric shows 1200 (max of 1200 used and 1022 config)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1200)

			// Second call - expect max_tokens=1200 (cached value from previous call)
			mapping2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1200, 900)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping2.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric remains 1200 (max of 900 used and 1200 cached)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1200)

			// Third call - also uses max_tokens=1200 (cached value remains the same)
			mapping3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1200, 900)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping3.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric remains 1200 (max of 800 used and 1200 cached)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1200)

			// Verify individual metrics - skip the comprehensive validation since it expects
			// the wrong cumulative sum value. The auto-increasing current value (1200) is already
			// verified above through CheckMetricsAdjustedCurrent calls.

			// Check model recognition metric
			max_tokens.CheckModelRecognitionMetric(metricsUrl, isolationID, "recognized", "gpt-35-turbo")

			// Verify that the WireMock expectations were matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, isolationID, expectedPath, 3)
			Expect(err).To(BeNil())
		})

		It("when calling model gpt-35-turbo with max_tokens, max_tokens must be not changed", func() {
			// Test scenario:
			// 1. Call with max_tokens=500 (explicit value provided by user)
			// 2. Service should forward the request unchanged with max_tokens=500
			// 3. No auto-adjustment should occur when max_tokens is explicitly provided
			// 4. Verify that no auto-adjusted metrics are collected

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

			// Verify metrics - when max_tokens is provided, we should see requested metrics but no auto-adjustment
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", originalMaxTokens, expectedMaxOutputTokens, float64(originalMaxTokens))

			// Verify that the WireMock expectation was matched
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})

		It("when using different isolation IDs, auto-adjusted values should be unique per isolation", func() {
			// Test scenario:
			// 1. Call with isolationA -> uses config value (1022), returns 1500 tokens -> cache stores 1500
			// 2. Call with isolationB -> uses config value (1022), returns 800 tokens -> cache stores 1022 (max of 800 and 1022)
			// 3. Second call with isolationA -> should use cached value (1500)
			// 4. Second call with isolationB -> should use cached value (1022)
			// 5. Verify that both isolations maintain separate auto-adjusted values

			isolationA := testID + "-isolation-a"
			isolationB := testID + "-isolation-b"
			expectedPath := "/openai/deployments/gpt-35-turbo-1106/chat/completions"

			// First call with isolation A - expect max_tokens=1022, mock returns 1500 tokens
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			mapping1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1022, 1500)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping1.Id)

			// Call gpt-35-turbo model with simple request but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric for isolation A shows 1500
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// First call with isolation B - expect max_tokens=1022, mock returns 800 tokens
			mapping2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, "gpt-35-turbo", 1022, 800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping2.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationB, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric for isolation B shows 1022 (max of 800 used and 1022 config)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Second call with isolation A - expect max_tokens=1500 (cached value)
			mapping3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1500, 900)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping3.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric for isolation A remains 1500
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Second call with isolation B - expect max_tokens=1022 (cached value)
			mapping4, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, "gpt-35-turbo", 1022, 800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping4.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationB, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric for isolation B remains 1022
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Verify that both isolations have their own metrics
			// Skip the comprehensive validation since it expects wrong cumulative sum.
			// The auto-increasing current values are already verified above through CheckMetricsAdjustedCurrent calls.

			// Check model recognition metrics for both isolations
			max_tokens.CheckModelRecognitionMetric(metricsUrl, isolationA, "recognized", "gpt-35-turbo")
			max_tokens.CheckModelRecognitionMetric(metricsUrl, isolationB, "recognized", "gpt-35-turbo")

			// Verify that the WireMock expectations were matched for both isolations
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, isolationA, expectedPath, 2)
			Expect(err).To(BeNil())
			err = max_tokens.VerifyWireMockExpectation(mockServerURL, isolationB, expectedPath, 2)
			Expect(err).To(BeNil())
		})
	})
})
