//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens_p99_forced_test

import (
	"fmt"
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

	_ = Context("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=P99 with FORCED=true", func() {

		It("when calling model gpt-35-turbo with max_tokens and ForceAdjustment=true, max_tokens should be replaced with P99 value", func() {
			// Test scenario with new forcing behavior:
			// 1. Build up some P99 samples first by calling without max_tokens
			// 2. Call with max_tokens=2000 and ForceAdjustment=true -> should force to P99 since P99 < 2000
			// 3. Call with max_tokens=1400 and ForceAdjustment=true -> should keep 1400 since P99 >= 1400

			isolationID := testID + "-forced"

			// Load mapping and construct full URL for gpt-35-turbo model
			mappings, err := LoadMappingFromFile(mappingFile)
			Expect(err).To(BeNil())
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			Expect(model).NotTo(BeNil())
			fullURL := fmt.Sprintf("%s%s", model.ModelUrl, "/chat/completions?api-version=2024-02-01")

			// Build up P99 baseline with multiple calls to establish cache samples
			// Add samples: [1100, 1300, 1500] -> P99 should be around 1500
			sampleValues := []int{1100, 1300, 1500}
			// First call: no cache, uses config value (1022)
			// Second call: P99 of [1100] = 1100
			// Third call: P99 of [1100, 1300] = 1300 (P99 with 2 samples returns the higher value)
			expectedMaxTokens := []int{1022, 1100, 1300}

			// Execute the baseline setup calls to build up P99 cache
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			for i, returnTokens := range sampleValues {
				expectedTokens := expectedMaxTokens[i]

				// Create WireMock expectation that validates the expected max_tokens value is sent
				expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", expectedTokens, returnTokens)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

				// Call gpt-35-turbo model without max_tokens to build up P99 samples
				requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
				max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})
			}

			// Verify that P99 current metric shows 1500 (P99 of [1100, 1300, 1500])
			// This P99 value will be used for forced adjustment calculations
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Test case 1: Call with max_tokens=2000 and ForceAdjustment=true
			// Since P99=1500 < original=2000, should force to P99 value (1500)
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1500, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			// Call gpt-35-turbo model with max_tokens=2000 - should be forced to P99 value
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(2000).Build()
			max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})

			// Test case 2: Call with max_tokens=1400 and ForceAdjustment=true
			// When forced=true, the logic is:
			// - If P99 < original, force to P99 (reduce tokens)
			// - If P99 >= original, keep original (don't increase tokens)
			// Since P99=1500 >= original=1400, we should keep the original value (1400)
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1400, 400)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			// Call gpt-35-turbo model with max_tokens=1400 - should keep original value
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(1400).Build()
			max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric remains 1500 since we didn't update cache when keeping original
			// Cache updates only occur when we adjust the max_tokens value downward
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// For this complex test scenario with mixed requests (some with max_tokens, some without),
			// we skip the comprehensive metrics validation and rely on the specific checks we've already done.
			// The key validation is that the current adjusted value is correct (1500), which we validated above.
		})

		It("when using different isolation IDs with ForceAdjustment=true, P99 values should be unique per isolation", func() {
			// Test scenario:
			// 1. Build up different P99 baselines for two isolations
			// 2. Call both with explicit max_tokens and ForceAdjustment=true
			// 3. Verify each uses its own P99 value for forced adjustment

			isolationA := testID + "-forced-a"
			isolationB := testID + "-forced-b"

			// Load mapping and construct full URL for gpt-35-turbo model
			mappings, err := LoadMappingFromFile(mappingFile)
			Expect(err).To(BeNil())
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			Expect(model).NotTo(BeNil())
			fullURL := fmt.Sprintf("%s%s", model.ModelUrl, "/chat/completions?api-version=2024-02-01")

			// Create WireMock expectation for isolation A first call - uses config value (1022)
			// Mock returns 1200 tokens which will be cached for this isolation
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1022, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			// Call gpt-35-turbo model for isolation A without max_tokens in request
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, fullURL, requestBody, map[string]string{})

			// Create WireMock expectation for isolation A second call - now uses P99 value (1200)
			// Mock returns 1400 tokens which will update the cache
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1200, 1400)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			// Call gpt-35-turbo model for isolation A again without max_tokens - should use P99 value
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, fullURL, requestBody, map[string]string{})

			// Create WireMock expectation for isolation B first call - uses config value (1022)
			// Mock returns 800 tokens which will result in caching max(800, 1022) = 1022
			expectation3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, "gpt-35-turbo", 1022, 800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation3.Id)

			// Call gpt-35-turbo model for isolation B without max_tokens in request
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationB, fullURL, requestBody, map[string]string{})

			// Verify that each isolation has established its own P99 baseline
			// Isolation A should show P99=1400 from samples [1200, 1400]
			// Isolation B should show P99=1022 from single sample [1022] (max of returned 800 and config 1022)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1400)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Test forced adjustment behavior with both isolations using the same original max_tokens=600
			// Forced behavior logic:
			// - If P99 < original, force to P99 (reduce tokens)
			// - If P99 >= original, keep original (don't increase tokens)
			// Isolation A: P99=1400 >= original=600, so keep 600
			// Isolation B: P99=1022 >= original=600, so keep 600

			// Create WireMock expectation for isolation A with forced adjustment
			// Since P99=1400 >= original=600, should keep original value (600)
			expectation4, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 600, 500)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation4.Id)

			// Call gpt-35-turbo model for isolation A with max_tokens=600 - should keep original
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(600).Build()
			max_tokens.ExpectLLMCall(isolationA, fullURL, requestBody, map[string]string{})

			// Create WireMock expectation for isolation B with forced adjustment
			// Since P99=1022 >= original=600, should keep original value (600)
			expectation5, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, "gpt-35-turbo", 600, 300)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation5.Id)

			// Call gpt-35-turbo model for isolation B with max_tokens=600 - should keep original
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(600).Build()
			max_tokens.ExpectLLMCall(isolationB, fullURL, requestBody, map[string]string{})

			// Verify that both isolations maintain their separate P99 values after forced calls
			// Since both calls kept their original values, P99 cache should remain unchanged
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1400)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// For this complex test scenario with mixed requests (some with max_tokens, some without),
			// we skip the comprehensive metrics validation and rely on the specific checks we've already done.
			// The key validation is that the current adjusted values are correct for each isolation, which we validated above.
		})

	})
})
