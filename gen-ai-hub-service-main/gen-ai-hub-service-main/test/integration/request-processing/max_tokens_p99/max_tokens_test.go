//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens_p99_test

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

	_ = Context("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=P99", func() {

		It("when calling model gpt-35-turbo without max_tokens, first call uses config value, subsequent calls use P99 percentile", func() {
			// Test scenario:
			// 1. First call without max_tokens -> should use REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE (1022)
			// 2. Mock returns 1100 completion tokens -> cache should store max(1100, 1022) = 1100
			// 3. Second call without max_tokens -> should use P99 percentile (1100 since only one sample)
			// 4. Mock returns 1300 completion tokens -> cache should store max(1300, 1022) = 1300
			// 5. Third call without max_tokens -> should use P99 percentile of [1100, 1300] = 1300
			// 6. Mock returns 1500 completion tokens -> cache should store max(1500, 1022) = 1500
			// 7. Fourth call without max_tokens -> should use P99 percentile of [1100, 1300, 1500] = 1500

			// Use the same isolation ID for all calls to test progressive P99 calculation within the same isolation
			isolationID := testID + "-shared"

			// Load mapping and construct full URL for gpt-35-turbo model
			mappings, err := LoadMappingFromFile(mappingFile)
			Expect(err).To(BeNil())
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			Expect(model).NotTo(BeNil())
			fullURL := fmt.Sprintf("%s%s", model.ModelUrl, "/chat/completions?api-version=2024-02-01")

			// Create WireMock expectation that validates max_tokens=1022 is sent to backend
			// This is the configured base value since no cache exists yet
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1022, 1100)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			// Call gpt-35-turbo model with simple request but without max_tokens in request
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric shows 1100 (P99 of single sample [1100])
			// This value comes from the mock response and gets cached for future P99 calculations
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1100)

			// Create WireMock expectation that validates max_tokens=1100 is sent to backend
			// This is now the P99 value from the single cached sample [1100]
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1100, 1300)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			// Call gpt-35-turbo model again without max_tokens - should now use P99 value
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric shows 1300 (P99 of [1100, 1300])
			// Cache now contains two samples, P99 calculation returns the higher value
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1300)

			// Create WireMock expectation that validates max_tokens=1300 is sent to backend
			// This is the P99 value calculated from cached samples [1100, 1300]
			expectation3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1300, 1500)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation3.Id)

			// Call gpt-35-turbo model third time without max_tokens - should use updated P99 value
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric shows 1500 (P99 of [1100, 1300, 1500])
			// Cache now contains three samples, P99 calculation returns the highest value
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Create WireMock expectation that validates max_tokens=1500 is sent to backend
			// This is the P99 value calculated from cached samples [1100, 1300, 1500]
			expectation4, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1500, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation4.Id)

			// Call gpt-35-turbo model fourth time without max_tokens - should continue using P99 value
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric shows 1500 (P99 of [1100, 1300, 1500, 1200])
			// Cache now contains four samples, P99 calculation still returns 1500 as the highest
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Get the expected maximum output tokens from model specifications
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")
			// Verify comprehensive metrics covering all aspects of the P99 auto-increasing strategy
			max_tokens.CheckMetricsAutoIncreasing(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", expectedMaxOutputTokens, 1500)
		})

		It("when calling model gpt-35-turbo with max_tokens and ForceAdjustment=false, max_tokens should not be changed", func() {
			// Test scenario:
			// 1. Call with max_tokens=500 and ForceAdjustment=false
			// 2. Should pass through unchanged (max_tokens=500)
			// 3. Cache should not be updated since original request had max_tokens
			// 4. P99 current metric should NOT be updated with 500

			// Load mapping and construct full URL for gpt-35-turbo model
			mappings, err := LoadMappingFromFile(mappingFile)
			Expect(err).To(BeNil())
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			Expect(model).NotTo(BeNil())
			fullURL := fmt.Sprintf("%s%s", model.ModelUrl, "/chat/completions?api-version=2024-02-01")

			// Create WireMock expectation that validates max_tokens=500 passes through unchanged
			// Since ForceAdjustment=false, the original max_tokens value should be preserved
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, config.URLPath, "gpt-35-turbo", 500, 400)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			// Call gpt-35-turbo model with request that includes max_tokens
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(500).Build()
			max_tokens.ExpectLLMCall(testID, fullURL, requestBody, map[string]string{})

			// Get the expected maximum output tokens from model specifications
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")
			// Verify metrics when original request had max_tokens but no adjustment occurred
			// The cache should not be updated when original request contains max_tokens and ForceAdjustment=false
			max_tokens.CheckMetricsWithOriginalMaxTokens(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", 500, expectedMaxOutputTokens, 500)
		})

		It("when using different isolation IDs, P99 values should be unique per isolation", func() {
			// Test scenario:
			// 1. Call with isolationID-A -> uses config value (1022), returns 1500 tokens -> cache stores 1500
			// 2. Call with isolationID-B -> uses config value (1022), returns 800 tokens -> cache stores 1022
			// 3. Call with isolationID-A again -> should use P99 percentile (1500)
			// 4. Call with isolationID-B again -> should use P99 percentile (1022)
			// This verifies that cache is properly isolated per isolationID

			isolationA := testID + "-isolation-a"
			isolationB := testID + "-isolation-b"

			// Load mapping and construct full URL for gpt-35-turbo model
			mappings, err := LoadMappingFromFile(mappingFile)
			Expect(err).To(BeNil())
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			Expect(model).NotTo(BeNil())
			fullURL := fmt.Sprintf("%s%s", model.ModelUrl, "/chat/completions?api-version=2024-02-01")

			// Create WireMock expectation for isolation A first call - uses config value (1022)
			// Mock returns 1500 tokens which will be cached for this isolation
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1022, 1500)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			// Call gpt-35-turbo model for isolation A without max_tokens in request
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric for isolation A shows 1500
			// This value gets cached and will be used for subsequent P99 calculations for this isolation
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Create WireMock expectation for isolation B first call - uses config value (1022)
			// Mock returns 800 tokens which will result in caching max(800, 1022) = 1022
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, "gpt-35-turbo", 1022, 800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			// Call gpt-35-turbo model for isolation B without max_tokens in request
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationB, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric for isolation B shows 1022 (max of 800 used and 1022 config)
			// Since returned tokens (800) < config (1022), the cache stores the config value
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Create WireMock expectation for isolation A second call - now uses P99 value (1500)
			// Mock returns 900 tokens, but request should still use 1500 from cache
			expectation3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1500, 900)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation3.Id)

			// Call gpt-35-turbo model for isolation A again without max_tokens - should use P99 value
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric for isolation A remains 1500 (P99 of [1500, 1022])
			// The returned value (900) gets stored as max(900, 1022) = 1022, so cache now has [1500, 1022]
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Call gpt-35-turbo model for isolation B again without max_tokens - should use P99 value (1022)
			// Note: This reuses expectation2 since both calls use the same max_tokens value
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationB, fullURL, requestBody, map[string]string{})

			// Verify that P99 current metric for isolation B remains 1022
			// Cache for isolation B contains [1022] so P99 calculation returns 1022
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Get the expected maximum output tokens from model specifications
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")
			// Verify that both isolations maintain separate P99 caches and metrics
			max_tokens.CheckMetricsAutoIncreasing(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", expectedMaxOutputTokens, 1500)
		})

	})
})
