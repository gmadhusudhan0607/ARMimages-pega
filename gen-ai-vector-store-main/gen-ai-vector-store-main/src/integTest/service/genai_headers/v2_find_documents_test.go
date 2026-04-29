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
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC Response headers", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()

	_ = Context("calling service", func() {
		var testExpectations []string

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			CreateIsolation(opsBaseURI, isolationID, "1GB")
			CreateCollection(svcBaseURI, isolationID, collectionID)
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

		_ = Context("/v2/{isolationID}/collections/{collectionName}/documents", func() {
			It("GET v2 collections/{collectionID}/documents/{documentID}/chunks returns expected headers when response is 200", func() {
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert a document so GET has something to retrieve
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Use "DOC-3" for testing, as it is present in the test01/documents directory
				documentID := "DOC-3"
				path := fmt.Sprintf("/v2/%s/collections/%s/documents/%s/chunks", isolationID, collectionID, documentID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

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
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 5},

					// DB metrics: allow 0 when metrics are not yet available
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(getResp, checks)
			})

			It("POST v2 collections/{collectionID}/find-documents returns expected headers when response is 200", func() {
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert documents so find-documents has something to retrieve
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Prepare FindDocumentsRequest payload to filter for inserted documents
				findDocumentsPayload := `{
					"fields": ["documentID"],
					"filter": {
						"attributes": {
							"operator": "AND",
							"items": [{
								"operator": "or",
								"name": "version",
								"values": ["v1.1", "v1.2", "v1.3"],
								"type": "string"
							}]
						}
					}
				}`
				path := fmt.Sprintf("/v2/%s/collections/%s/find-documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By(fmt.Sprintf("Calling POST endpoint %s", endpointURI))

				postResp, _, postErr := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, findDocumentsPayload)
				Expect(postErr).To(BeNil())
				Expect(postResp).NotTo(BeNil())
				Expect(postResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", http.StatusOK)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 2},

					// DB metrics: allow 0 when metrics are not yet available
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(postResp, checks)
			})
		})

	})

})
