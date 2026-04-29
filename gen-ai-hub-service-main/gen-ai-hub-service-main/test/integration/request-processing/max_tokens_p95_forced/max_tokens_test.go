//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens_p95_forced_test

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

	_ = Context("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=P95 with FORCED=true", func() {

		It("when calling model gpt-35-turbo with max_tokens and ForceAdjustment=true, max_tokens must be adjusted according to P95 forced behavior", func() {
			// Test scenario:
			// - Build P95 baseline with samples [1100, 1300, 1500] resulting in P95=1500
			// - Call with max_tokens=2000 and ForceAdjustment=true -> should force to P95 (1500) since P95 < 2000
			// - Call with max_tokens=1400 and ForceAdjustment=true -> should keep 1400 since P95 >= 1400 (new forcing behavior)

			isolationID := testID + "-forced"

			// Setup: Load mapping and construct full URL for gpt-35-turbo model
			mappings, err := LoadMappingFromFile(mappingFile)
			Expect(err).To(BeNil())
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			Expect(model).NotTo(BeNil())
			fullURL := fmt.Sprintf("%s%s", model.ModelUrl, "/chat/completions?api-version=2024-02-01")

			// Build P95 baseline by calling without max_tokens
			// Sample values: [1100, 1300, 1500] -> P95 = 1500
			// Expected max_tokens progression: [1022, 1100, 1300] (config, P95 of [1100], P95 of [1100,1300])
			sampleValues := []int{1100, 1300, 1500}
			expectedMaxTokens := []int{1022, 1100, 1300}

			config := max_tokens.GetModelConfig("gpt-35-turbo")
			for i, returnTokens := range sampleValues {
				expectedTokens := expectedMaxTokens[i]
				mapping, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", expectedTokens, returnTokens)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mapping.Id)

				requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
				max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})
			}

			// Verify P95 baseline is established at 1500
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)

			// Test Case 1: Call with max_tokens=2000 and ForceAdjustment=true
			// Should force to P95 value (1500) since P95 < original (2000)
			mapping1, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1500, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping1.Id)

			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(2000).Build()
			max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})

			// Test Case 2: Call with max_tokens=1400 and ForceAdjustment=true
			// Should keep original (1400) since P95 >= original (1500 >= 1400)
			mapping2, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationID, config.URLPath, "gpt-35-turbo", 1400, 400)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping2.Id)

			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(1400).Build()
			max_tokens.ExpectLLMCall(isolationID, fullURL, requestBody, map[string]string{})

			// Verify P95 remains at 1500
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationID, "gpt-35-turbo", "gpt-35-turbo", 1500)
		})

		It("when using different isolation IDs with ForceAdjustment=true, each isolation must maintain its own P95 baseline", func() {
			// Test scenario:
			// - Build different P95 baselines for two isolation contexts (A: P95=1400, B: P95=1022)
			// - Call both with same max_tokens=600 and ForceAdjustment=true
			// - Verify each isolation maintains its unique P95 value and applies forced behavior correctly

			isolationA := testID + "-forced-a"
			isolationB := testID + "-forced-b"

			// Setup: Load mapping and construct full URL for gpt-35-turbo model
			mappings, err := LoadMappingFromFile(mappingFile)
			Expect(err).To(BeNil())
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			Expect(model).NotTo(BeNil())
			fullURL := fmt.Sprintf("%s%s", model.ModelUrl, "/chat/completions?api-version=2024-02-01")

			// Build P95 baseline for isolation A: samples [1200, 1400] -> P95 = 1400
			config := max_tokens.GetModelConfig("gpt-35-turbo")
			mapping3, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1022, 1200)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping3.Id)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, fullURL, requestBody, map[string]string{})

			mapping4, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 1200, 1400)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping4.Id)
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationA, fullURL, requestBody, map[string]string{})

			// Build P95 baseline for isolation B: sample [800] -> P95 = 1022 (max of 800 and config)
			mapping5, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, "gpt-35-turbo", 1022, 800)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping5.Id)
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(isolationB, fullURL, requestBody, map[string]string{})

			// Verify P95 baselines are established correctly
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1400)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)

			// Test both isolations with max_tokens=600 and ForceAdjustment=true
			// Isolation A: P95=1400, original=600, since 1400 >= 600, keep 600
			// Isolation B: P95=1022, original=600, since 1022 >= 600, keep 600
			mapping6, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationA, config.URLPath, "gpt-35-turbo", 600, 500)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping6.Id)
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(600).Build()
			max_tokens.ExpectLLMCall(isolationA, fullURL, requestBody, map[string]string{})

			mapping7, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, isolationB, config.URLPath, "gpt-35-turbo", 600, 300)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, mapping7.Id)
			requestBody = max_tokens.NewLLMRequestBodyBuilder("gpt-35-turbo", "1106", "2024-10-21").WithMaxTokens(600).Build()
			max_tokens.ExpectLLMCall(isolationB, fullURL, requestBody, map[string]string{})

			// Verify both isolations maintain their unique P95 values
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationA, "gpt-35-turbo", "gpt-35-turbo", 1400)
			max_tokens.CheckMetricsAdjustedCurrent(metricsUrl, isolationB, "gpt-35-turbo", "gpt-35-turbo", 1022)
		})

	})
})
