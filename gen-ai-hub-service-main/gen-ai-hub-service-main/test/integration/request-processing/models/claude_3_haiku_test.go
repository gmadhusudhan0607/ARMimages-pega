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

var _ = Describe("Tests SVC Claude 3 Haiku Model Recognition (REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED):", Ordered, func() {

	var err error
	var testID string
	var testWireMockExpectations []string

	BeforeAll(func() {
		Expect(err).To(BeNil())
	})

	AfterAll(func() {
		for _, mappingId := range testWireMockExpectations {
			err := functions.DeleteWireMockExpectation(mockServerURL, mappingId)
			Expect(err).To(BeNil())
		}
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", functions.RandStringRunes(10)))
		err := functions.ResetWireMockServer(mockServerURL)
		Expect(err).To(BeNil())
		err = functions.CreateMappingEndpointExpectation(mockServerURL, mappingsEndpointPath, mappingEndpointFile)
		Expect(err).To(BeNil())
		err = functions.CreateDefaultsEndpointExpectation(mockServerURL, defaultsEndpointPath, defaultsEndpointFile)
		Expect(err).To(BeNil())
	})

	_ = Context("claude-3-haiku model tests", func() {

		XIt("when calling model claude-3-haiku, max_token must be set to fixed value if max_token was NOT provided in original request", func() { // temporary skipped
			expectedPath := "/anthropic/deployments/claude-3-haiku/chat/completions"
			expectedModelName := "claude-3-haiku"
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, 1022, 256)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			requestUrl := fmt.Sprintf("%s/anthropic/deployments/claude-3-haiku/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("anthropic/claude-3-haiku", "v1", "").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 4096

			max_tokens.CheckMetricsFixed(metricsUrl, testID, "claude-3-haiku", "claude-3-haiku", -1, expectedMaxOutputTokens, 1022)

			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})
	})
})
