//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package unrecognized_test

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

		// Recreate monitoring endpoint expectation after reset
		err = CreateMonitoringEndpointExpectation(mockServerURL, monitoringEventsPath)
		Expect(err).To(BeNil())

		// Recreate mapping and defaults endpoint expectations after reset
		err = CreateMappingEndpointExpectation(mockServerURL, mappingsEndpointPath, mappingEndpointFile)
		Expect(err).To(BeNil())
		err = CreateDefaultsEndpointExpectation(mockServerURL, defaultsEndpointPath, defaultsEndpointFile)
		Expect(err).To(BeNil())
	})

	_ = Context("REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED", func() {

		_ = Context("Buddies (selfstudybuddy)", func() {

			It("when calling selfstudybuddy without max_tokens, service forwards request unchanged and returns response unchanged", func() {
				isolationID := testID
				buddyId := "selfstudybuddy"
				expectedPath := fmt.Sprintf("/v1/%s/buddies/%s/question", isolationID, buddyId)
				backendPath := "/question"

				// Create a complex request body without max_tokens
				requestBody := CreateBuddyRequestBody("Hello, I need help with my studies!", nil)

				// Expected response from backend
				expectedResponse := `{"response": "I'd be happy to help you with your studies! What specific topic would you like to work on?"}`

				// Set up WireMock expectation for the backend call
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, backendPath, requestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service
				requestUrl := fmt.Sprintf("%s%s", svcBaseURL, expectedPath)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, requestBody, expectedResponse)

				// Verify request body was forwarded unchanged
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, backendPath, requestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, backendPath, 1)
				Expect(err).To(BeNil())
			})

			It("when calling selfstudybuddy with max_tokens, service forwards request unchanged and returns response unchanged", func() {
				isolationID := testID
				buddyId := "selfstudybuddy"
				expectedPath := fmt.Sprintf("/v1/%s/buddies/%s/question", isolationID, buddyId)
				backendPath := "/question"
				originalMaxTokens := 150

				// Create a complex request body with max_tokens
				requestBody := CreateBuddyRequestBody("Explain photosynthesis in simple terms", &originalMaxTokens)

				// Expected response from backend
				expectedResponse := `{"response": "Photosynthesis is how plants make food using sunlight, water, and air!", "max_tokens_used": 150}`

				// Set up WireMock expectation for the backend call
				mappingId, err := CreateWireMockExpectationWithExactBody(mockServerURL, isolationID, backendPath, requestBody, expectedResponse)
				Expect(err).To(BeNil())
				testWireMockExpectations = append(testWireMockExpectations, mappingId)

				// Call the service
				requestUrl := fmt.Sprintf("%s%s", svcBaseURL, expectedPath)
				actualResponse := max_tokens.ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, requestBody, expectedResponse)

				// Verify request body was forwarded unchanged (including max_tokens)
				max_tokens.VerifyRequestBodyUnchanged(mockServerURL, isolationID, backendPath, requestBody)

				// Verify response was returned unchanged
				Expect(actualResponse).To(Equal(expectedResponse))

				// Verify the backend was called exactly once
				err = VerifyWireMockExpectation(mockServerURL, isolationID, backendPath, 1)
				Expect(err).To(BeNil())
			})

		})

	})

})
