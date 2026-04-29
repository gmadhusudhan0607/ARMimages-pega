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

var _ = Describe("Tests SVC Llama3 70B Instruct Model Recognition (REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED):", Ordered, func() {

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

	_ = Context("llama3-70b-instruct model tests", func() {

		XIt("when calling model llama3-70b-instruct, max_token must be set to fixed value if max_token was NOT provided in original request", func() { // temporary skipped
			expectedPath := "/meta/deployments/llama3-70b-instruct/chat/completions"
			expectedModelName := "llama3-70b-instruct"
			expectation, err := max_tokens.CreateWireMockMaxTokensExpectation(mockServerURL, testID, expectedPath, expectedModelName, 1022, 256)
			Expect(err).To(BeNil())
			testWireMockExpectations = append(testWireMockExpectations, expectation.Id)

			requestUrl := fmt.Sprintf("%s/meta/deployments/llama3-70b-instruct/chat/completions", svcBaseURL)
			requestBody := max_tokens.NewLLMRequestBodyBuilder("meta/llama3-70b-instruct", "v1", "").WithoutMaxTokens().Build()
			max_tokens.ExpectLLMCall(testID, requestUrl, requestBody, map[string]string{})

			expectedMaxOutputTokens := 8192

			max_tokens.CheckMetricsFixed(metricsUrl, testID, "llama3-70b-instruct", "llama3-70b-instruct", -1, expectedMaxOutputTokens, 1022)

			err = max_tokens.VerifyWireMockExpectation(mockServerURL, testID, expectedPath, 1)
			Expect(err).To(BeNil())
		})
	})
})
