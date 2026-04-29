//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package cross_functional_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC Response headers", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()

	_ = Context("calling service", func() {

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			CreateIsolation(opsBaseURI, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
		})

		_ = Context("/v2/{isolationID}/collections", func() {

			It("POST v2 collections return expected headers when response is 200", func() {
				path := fmt.Sprintf("/v2/%s/collections", isolationID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)
				By(fmt.Sprintf("Calling endpoint %s", endpointURI))
				reqBody := fmt.Sprintf("{\"collectionID\": \"%s\"}", collectionID)

				response, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, reqBody)
				Expect(err).To(BeNil())
				Expect(response).NotTo(BeNil())
				Expect(response.StatusCode).To(Equal(http.StatusCreated))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", http.StatusCreated)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				// Header checks
				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
				}
				// Ensure test checks cover all expected headers
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				// validate response headers have expected values
				ExpectHeadersFlexible(response, checks)

			})

			It("GET v2 collections return expected headers when response is 200", func() {
				path := fmt.Sprintf("/v2/%s/collections", isolationID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)
				By(fmt.Sprintf("Ensuring collection exists via POST %s", endpointURI))
				// Ensure collection exists
				reqBody := fmt.Sprintf("{\"collectionID\": \"%s\"}", collectionID)

				postResp, _, postErr := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, reqBody)
				Expect(postErr).To(BeNil())
				Expect(postResp).NotTo(BeNil())
				Expect(postResp.StatusCode).To(Equal(http.StatusCreated))

				By(fmt.Sprintf("Calling GET endpoint %s", endpointURI))
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
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 1},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(getResp, checks)
			})

			It("GET v2 collections/{collectionID} return expected headers when response is 200", func() {
				// Ensure collection exists
				collectionPath := fmt.Sprintf("/v2/%s/collections", isolationID)
				collectionEndpoint := fmt.Sprintf("%s%s", svcBaseURI, collectionPath)
				reqBody := fmt.Sprintf("{\"collectionID\": \"%s\"}", collectionID)
				requestHeaders := map[string]string{
					headers.ForceFreshDbMetrics: "true",
				}
				postResp, _, postErr := HttpCallWithHeadersAndApiCallStat("POST", collectionEndpoint, requestHeaders, reqBody)
				Expect(postErr).To(BeNil())
				Expect(postResp).NotTo(BeNil())
				Expect(postResp.StatusCode).To(Equal(http.StatusCreated))

				// Call GET for specific collection
				path := fmt.Sprintf("/v2/%s/collections/%s", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)
				By(fmt.Sprintf("Calling GET endpoint %s", endpointURI))
				requestHeaders = map[string]string{
					headers.ForceFreshDbMetrics: "true",
				}
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

			It("DELETE v2 collections/{collectionID} return expected headers when response is 200", func() {
				// Ensure collection exists
				collectionPath := fmt.Sprintf("/v2/%s/collections", isolationID)
				collectionEndpoint := fmt.Sprintf("%s%s", svcBaseURI, collectionPath)
				reqBody := fmt.Sprintf("{\"collectionID\": \"%s\"}", collectionID)
				requestHeaders := map[string]string{
					headers.ForceFreshDbMetrics: "true",
				}
				postResp, _, postErr := HttpCallWithHeadersAndApiCallStat("POST", collectionEndpoint, requestHeaders, reqBody)
				Expect(postErr).To(BeNil())
				Expect(postResp).NotTo(BeNil())
				Expect(postResp.StatusCode).To(Equal(http.StatusCreated))

				// Call DELETE for specific collection
				path := fmt.Sprintf("/v2/%s/collections/%s", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)
				By(fmt.Sprintf("Calling DELETE endpoint %s", endpointURI))
				requestHeaders = map[string]string{
					headers.ForceFreshDbMetrics: "true",
				}
				deleteResp, _, deleteErr := HttpCallWithHeadersAndApiCallStat("DELETE", endpointURI, ServerConfigurationHeaders, "")
				Expect(deleteErr).To(BeNil())
				Expect(deleteResp).NotTo(BeNil())
				Expect(deleteResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "DELETE", http.StatusOK)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(deleteResp, checks)
			})
		})

	})

})
