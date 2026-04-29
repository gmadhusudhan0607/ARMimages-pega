//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens_auto_increasing_forced_test

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

	_ = Context("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=AUTO_INCREASING + FORCED", func() {

		It("when calling model gpt-35-turbo without max_tokens, should use auto-adjusted values progressively", func() {
			// Test scenario: First call uses config value (1022), mock returns 1200 tokens -> cache stores 1200.
			// Subsequent calls use cached value (1200) regardless of actual usage.

			// Use the same isolation ID for all calls to test progressive auto-adjustment within the same isolation
			isolationID := testID + "-shared"

			// First call - expect max_tokens=1022 (config value), mock returns 1200 tokens to trigger cache update
			expectedModelName := "gpt-35-turbo"
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, expectedModelName, 1022, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric shows 1200 (max of 1200 used and 1022 config)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1200)

			// Second and third calls - both expect max_tokens=1200 (cached value from previous call)
			// Create a single expectation that will match both calls with 900 and 800 tokens respectively
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, expectedModelName, 1200, 900)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			expectation3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, expectedModelName, 1200, 800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation3.Id)

			// Second call
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric remains 1200 (max of 900 used and 1200 cached)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1200)

			// Third call - also uses max_tokens=1200 (cached value remains the same)
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric remains 1200 (max of 800 used and 1200 cached)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1200)

			// Verify comprehensive metrics manually since CheckMetricsAutoIncreasing expects sum=current
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")

			// Check that all core metrics are present with correct values
			checker := max_tokens.NewMetricsChecker(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo")
			metricsConfig := max_tokens.MetricsConfig{
				ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
				ExpectedAdjustedValue:            3422, // Sum of all adjusted tokens: 1022 + 1200 + 1200
				IsAdjustmentStrategy:             true,
				ShouldCheckAdjustedCurrentMetric: false, // Don't check adjusted current metric here
				ShouldCheckWastedMetrics:         true,
				ShouldCheckRequestedMetrics:      false,
			}
			checker.CheckMetrics(metricsConfig)

			// Separately verify the auto-increasing current metric is 1200 (cached value)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1200)
		})

		It("when calling model gpt-35-turbo with max_tokens lower than auto-adjusted value, max_tokens should be kept", func() {
			// Test scenario: With AdjustmentForced=true, max_tokens=500 should be kept since auto-adjusted (1022) >= original (500).

			// Call with explicit max_tokens=500 - should be kept at 500 since suggested (1022) >= original (500)
			expectedModelName := "gpt-35-turbo"
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, config.URLPath, expectedModelName, 500, 50)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(500).Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			// Verify metrics - should show original request (500) and adjusted (500) since it was kept
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")
			max_tokens.CheckMetricsAutoIncreasingForced(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", 500, 500, expectedMaxOutputTokens)

			// Note: When we keep the original value, the auto-adjusted metric is NOT updated
			// because we didn't actually apply the auto-adjustment strategy
		})

		It("when calling model gpt-35-turbo with max_tokens higher than auto-adjusted value, max_tokens should be forced", func() {
			// Test scenario: With AdjustmentForced=true, max_tokens=2000 should be forced to 1022 since auto-adjusted (1022) < original (2000).

			// Call with explicit max_tokens=2000 - should be forced to 1022 since suggested (1022) < original (2000)
			expectedModelName := "gpt-35-turbo"
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, config.URLPath, expectedModelName, 1022, 800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(2000).Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			// Verify metrics - should show original request (2000) and forced adjustment (1022)
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")
			max_tokens.CheckMetricsAutoIncreasingForced(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", 2000, 1022, expectedMaxOutputTokens)

			// Check that auto-adjusted current metric shows 1022 (max of 800 used and 1022 config)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, testID, "gpt-35-turbo", "gpt-35-turbo", 1022)
		})

		It("when using progressive auto-adjustment with AdjustmentForced=true, should follow new forcing behavior", func() {
			// Test scenario: Progressive calls with AdjustmentForced=true should force high values down but keep low values.

			isolationID := testID + "-progressive"

			// First call with max_tokens=3000 - should be forced to config value (1022) since 1022 < 3000, mock returns 1500 tokens
			expectedModelName := "gpt-35-turbo"
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, expectedModelName, 1022, 1500)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(3000).Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric shows 1500 (max of 1500 used and 1022 config)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Second call with max_tokens=400 - should be kept at 400 since cached value (1500) >= 400, mock returns 300 tokens
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, expectedModelName, 400, 300)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(400).Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric remains 1500 (cache not updated since we kept original)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Third call without max_tokens - should use cached value (1500)
			expectation3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, expectedModelName, 1500, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation3.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationID, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric remains 1500
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Verify comprehensive metrics manually since this test includes explicit max_tokens
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")

			// Check that all core metrics are present with correct values
			checker := max_tokens.NewMetricsChecker(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo")
			metricsConfig := max_tokens.MetricsConfig{
				ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
				ExpectedAdjustedValue:            2922, // Sum of adjusted tokens: 1022 + 400 + 1500
				IsAdjustmentStrategy:             true,
				ShouldCheckAdjustedCurrentMetric: false, // Don't check adjusted current metric here
				ShouldCheckWastedMetrics:         true,
				ShouldCheckRequestedMetrics:      true, // This test has explicit max_tokens requests
			}
			checker.CheckMetrics(metricsConfig)

			// Separately verify the auto-increasing current metric is 1500 (cached value)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)
		})

		It("when using different isolation IDs with AdjustmentForced=true, cache isolation should work properly", func() {
			// Test scenario: Different isolation IDs should maintain separate caches with proper forcing behavior.

			isolationA := testID + "-isolation-a"
			isolationB := testID + "-isolation-b"

			// First call with isolation A and max_tokens=3000 - forced to config value (1022) since 1022 < 3000, mock returns 1800 tokens
			expectedModelName := "gpt-35-turbo"
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			expectation1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, expectedModelName, 1022, 1800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation1.Id)

			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(3000).Build()
			max_tokens.ExpectLLMCall(isolationA, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric for isolation A shows 1800
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1800)

			// First call with isolation B and max_tokens=4000 - forced to config value (1022) since 1022 < 4000, mock returns 900 tokens
			expectation2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, expectedModelName, 1022, 900)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation2.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(4000).Build()
			max_tokens.ExpectLLMCall(isolationB, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric for isolation B shows 1022 (max of 900 used and 1022 config)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Second call with isolation A and max_tokens=250 - kept at 250 since cached value (1800) >= 250
			expectation3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, expectedModelName, 250, 200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation3.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(250).Build()
			max_tokens.ExpectLLMCall(isolationA, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric for isolation A remains 1800 (cache not updated since we kept original)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1800)

			// Second call with isolation B and max_tokens=350 - kept at 350 since cached value (1022) >= 350
			expectation4, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, expectedModelName, 350, 300)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation4.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(350).Build()
			max_tokens.ExpectLLMCall(isolationB, requestUrl, requestBody, map[string]string{})

			// Check that auto-adjusted current metric for isolation B remains 1022 (cache not updated since we kept original)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Verify that both isolations have their own metrics manually since this test includes explicit max_tokens
			expectedMaxOutputTokens := max_tokens.GetMaxOutputTokensFromModelSpecs("gpt-35-turbo")

			// Check that all core metrics are present for isolation A
			checker := max_tokens.NewMetricsChecker(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo")
			metricsConfig := max_tokens.MetricsConfig{
				ExpectedMaxOutputTokens:          expectedMaxOutputTokens,
				ExpectedAdjustedValue:            1272, // Sum of adjusted tokens: 1022 + 250
				IsAdjustmentStrategy:             true,
				ShouldCheckAdjustedCurrentMetric: false, // Don't check adjusted current metric here
				ShouldCheckWastedMetrics:         true,
				ShouldCheckRequestedMetrics:      true, // This test has explicit max_tokens requests
			}
			checker.CheckMetrics(metricsConfig)

			// Separately verify the auto-increasing current metric for isolation A is 1800 (cached value)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1800)
		})
	})
})
