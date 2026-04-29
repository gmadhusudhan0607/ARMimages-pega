//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens_p95_test

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
	})

	_ = Context("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=P95", func() {

		It("when calling model gpt-35-turbo without max_tokens, first call uses config value, subsequent calls use P95 percentile", func() {
			// Test scenario: Progressive P95 calculation within same isolation ID
			// - First call without max_tokens uses config value (1022), returns 1100 tokens
			// - Subsequent calls use P95 percentile of cached completion token values
			// - Each response updates cache with max(actual_tokens, config_value)
			// - P95 percentile increases as more samples are collected: 1100 -> 1300 -> 1500

			// Use the same isolation ID for all calls to test progressive P95 calculation within the same isolation
			isolationID := testID + "-shared"

			// Create WireMock expectation that validates max_tokens=1022 is added to the request
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1022, 1100)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			// Call gpt-35-turbo model with simple request but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that P95 current metric shows 1100 (P95 of single sample [1100])
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1100)

			// Second call - expect max_tokens=1100 (P95 of [1100]), mock returns 1300 tokens
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1100, 1300)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that P95 current metric shows 1300 (P95 of [1100, 1300])
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1300)

			// Third call - expect max_tokens=1300 (P95 of [1100, 1300]), mock returns 1500 tokens
			expectation3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1300, 1500)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation3.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that P95 current metric shows 1500 (P95 of [1100, 1300, 1500])
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Fourth call - expect max_tokens=1500 (P95 of [1100, 1300, 1500]), mock returns 1200 tokens
			expectation4, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1500, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation4.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that P95 current metric shows 1500 (P95 of [1100, 1300, 1500, 1200])
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Note: We don't need to verify comprehensive metrics here since we're already checking
			// the auto_increasing_current value throughout the test, and the adjusted_sum would be
			// the sum of all adjusted values (1022 + 1100 + 1300 + 1500 = 4922) which differs
			// from the current P95 value (1500)
		})

		It("when calling model gpt-35-turbo with max_tokens and ForceAdjustment=false, max_tokens should not be changed", func() {
			// Test scenario: max_tokens passthrough when provided in original request
			// - Call with max_tokens=500 should pass through unchanged
			// - Cache should not be updated since original request had max_tokens
			// - P95 current metric should NOT be updated

			// Create WireMock expectation that validates max_tokens remains at original value (500)
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, config.URLPath, "gpt-35-turbo", 500, 400)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call gpt-35-turbo model with request that includes max_tokens
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(500).Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			// Verify that P95 current metric is NOT present since no P95 adjustment occurred
			// But requested metrics SHOULD be present since original request had max_tokens=500
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")
			max_tokens.CheckMetricsFixed(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", 500, expectedMaxOutputTokens, 500)
		})

		It("when using different isolation IDs, P95 values should be unique per isolation", func() {
			// Test scenario: P95 cache isolation per isolation ID
			// - isolationID-A: uses config value (1022), returns 1500 tokens -> P95 becomes 1500
			// - isolationID-B: uses config value (1022), returns 800 tokens -> P95 remains 1022
			// - Subsequent calls verify each isolation maintains its own P95 values

			isolationA := testID + "-isolation-a"
			isolationB := testID + "-isolation-b"

			// Create WireMock expectation that validates max_tokens=1022 is added to the request
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1022, 1500)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			// Call gpt-35-turbo model with simple request but without max_tokens in request
			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, requestUrl, requestBody, map[string]string{})

			// Check that P95 current metric for isolation A shows 1500
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// First call with isolation B - expect max_tokens=1022, mock returns 800 tokens
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, "gpt-35-turbo", 1022, 800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationB, requestUrl, requestBody, map[string]string{})

			// Check that P95 current metric for isolation B shows 1022 (max of 800 used and 1022 config)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Second call with isolation A - expect max_tokens=1500 (P95 of [1500])
			expectation3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1500, 900)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation3.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, requestUrl, requestBody, map[string]string{})

			// Check that P95 current metric for isolation A remains 1500 (P95 of [1500, 1022])
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Second call with isolation B - expect max_tokens=1022 (P95 of [1022])
			// Note: This will reuse expectation2 since both calls use the same max_tokens value
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationB, requestUrl, requestBody, map[string]string{})

			// Check that P95 current metric for isolation B remains 1022
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Note: We don't need to verify comprehensive metrics here since we're already checking
			// the auto_increasing_current value throughout the test for both isolations
		})

	})
})
