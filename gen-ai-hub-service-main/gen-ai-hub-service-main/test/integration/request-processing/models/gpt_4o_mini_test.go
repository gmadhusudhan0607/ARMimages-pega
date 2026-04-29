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

var _ = Describe("Tests SVC GPT-4o Mini Model Recognition (REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED):", Ordered, func() {

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

	_ = Context("gpt-4o-mini model tests", func() {

		It("when calling model gpt-4o-mini, max_token must be set to fixed value if max_token was NOT provided in original request", func() {
			expectedPath := "/openai/deployments/gpt-4o-mini/chat/completions"
			expectedModelName := "gpt-4o-mini"
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, 1022, 256)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			requestUrl := fmt.Sprintf("%s/openai/deployments/gpt-4o-mini/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("openai/gpt-4o-mini", "2024-07-18", "").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 16384

			max_tokens.CheckMetricsFixed(metricsUrl, testID, "gpt-4o-mini", "gpt-4o-mini", -1, expectedMaxOutputTokens, 1022)

			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})
	})
})
