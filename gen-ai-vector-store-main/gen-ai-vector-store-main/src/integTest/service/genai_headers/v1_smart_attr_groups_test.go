//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package cross_functional_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	attributesgroup "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes_group"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC Response headers", func() {

	var isolationID string
	var ctx = context.TODO()

	_ = Context("calling service", func() {
		var testExpectations []string
		Expect(true).To(Equal(true))

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			CreateIsolation(opsBaseURI, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		_ = Context("/v1/{isolationID}/smart-attributes-group", func() {

			It("POST v1/{isolationID}/smart-attributes-group returns expected headers when response is 200", func() {
				path := fmt.Sprintf("/v1/%s/smart-attributes-group", isolationID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Prepare request body
				description := fmt.Sprintf("Test group %s", RandStringRunes(6))
				attributes := []string{RandStringRunes(5), RandStringRunes(5)}
				reqBody := fmt.Sprintf(`{"description": "%s", "attributes": ["%s", "%s"]}`,
					description, attributes[0], attributes[1])

				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, reqBody)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", resp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(resp, checks)
			})

			It("GET v1/{isolationID}/smart-attributes-group returns expected headers when response is 200", func() {
				path := fmt.Sprintf("/v1/%s/smart-attributes-group", isolationID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Ensure at least one attribute group exists for isolation
				description := fmt.Sprintf("Test group %s", RandStringRunes(6))
				attributes := []string{RandStringRunes(5), RandStringRunes(5)}
				postReqBody := fmt.Sprintf(`{"description": "%s", "attributes": ["%s", "%s"]}`,
					description, attributes[0], attributes[1])
				postResp, _, postErr := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, postReqBody)
				Expect(postErr).To(BeNil())
				Expect(postResp).NotTo(BeNil())
				Expect(postResp.StatusCode).To(Equal(http.StatusOK))

				// Now test GET
				getResp, getBody, getErr := HttpCallWithHeadersAndApiCallStat("GET", endpointURI, ServerConfigurationHeaders, "")
				Expect(getErr).To(BeNil())
				Expect(getResp).NotTo(BeNil())
				Expect(getResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "GET", getResp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 1},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(getResp, checks)

				// Optionally, check that the response body contains at least one group
				Expect(getBody).NotTo(BeEmpty())
			})

			It("GET v1 smart-attributes-group/{groupID} returns expected headers when response is 200", func() {

				By("Create a smart attributes group to test GET")
				// Create Attributes Group
				path := fmt.Sprintf("/v1/%s/smart-attributes-group", isolationID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)
				description := fmt.Sprintf("Test group %s", RandStringRunes(6))
				attributes := []string{RandStringRunes(5), RandStringRunes(5)}
				reqBody := fmt.Sprintf(`{"description": "%s", "attributes": ["%s", "%s"]}`,
					description, attributes[0], attributes[1])

				resp, respBody, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, reqBody)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				newAG := attributesgroup.AttributesGroup{}
				err = json.Unmarshal(respBody, &newAG)
				groupID := newAG.GroupID

				By(fmt.Sprintf("Calling GET endpoint %s", endpointURI))
				path = fmt.Sprintf("/v1/%s/smart-attributes-group/%s", isolationID, groupID)
				endpointURI = fmt.Sprintf("%s%s", svcBaseURI, path)

				getResp, _, getErr := HttpCallWithHeadersAndApiCallStat("GET", endpointURI, ServerConfigurationHeaders, "")
				Expect(getErr).To(BeNil())
				Expect(getResp).NotTo(BeNil())
				Expect(getResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "GET", http.StatusOK)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(getResp, checks)
			})

			It("DELETE v1/{isolationID}/smart-attributes-group/{groupID} returns expected headers when response is 200", func() {

				// Create Attributes Group
				path := fmt.Sprintf("/v1/%s/smart-attributes-group", isolationID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)
				description := fmt.Sprintf("Test group %s", RandStringRunes(6))
				attributes := []string{RandStringRunes(5), RandStringRunes(5)}
				reqBody := fmt.Sprintf(`{"description": "%s", "attributes": ["%s", "%s"]}`,
					description, attributes[0], attributes[1])

				resp, respBody, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, reqBody)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				newAG := attributesgroup.AttributesGroup{}
				err = json.Unmarshal(respBody, &newAG)
				groupID := newAG.GroupID

				By(fmt.Sprintf("Calling DELETE endpoint %s", endpointURI))
				path = fmt.Sprintf("/v1/%s/smart-attributes-group/%s", isolationID, groupID)
				endpointURI = fmt.Sprintf("%s%s", svcBaseURI, path)

				delResp, _, delErr := HttpCallWithHeadersAndApiCallStat("DELETE", endpointURI, ServerConfigurationHeaders, "")
				Expect(delErr).To(BeNil())
				Expect(delResp).NotTo(BeNil())
				Expect(delResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "DELETE", http.StatusOK)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(delResp, checks)
			})

		})

	})

})
