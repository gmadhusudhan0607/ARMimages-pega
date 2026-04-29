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

		_ = Context("/v1/{isolationID}/collections/{collectionName}/documents", func() {

			It("PUT v1 collections/{collectionName}/documents returns expected headers when response is 202 for ASYNC call", func() {
				// Setup
				path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Prepare document payload
				docPayload := `{"id": "DOC-1", "chunks": [{"content": "some text here"}], "attributes": []}`

				// Call PUT document endpoint
				putResp, _, putErr := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docPayload)
				Expect(putErr).To(BeNil())
				Expect(putResp).NotTo(BeNil())
				Expect(putResp.StatusCode).To(Equal(http.StatusAccepted))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "PUT", http.StatusAccepted)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					// Asynchronous operation Does not return next headers ModelId, ModelVersion, EmbeddingTimeMs, EmbeddingCallsCount

					// DB - ASYNC: Document may not be processed yet
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 1}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 2}},
				}

				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(putResp, checks)
			})

			It("PUT v1 collections/{collectionName}/documents returns expected headers when response is 201 for SYNC call", func() {

				// Setup
				path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s?consistencyLevel=strong", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Prepare document payload
				docPayload := `{"id": "DOC-1",
			                   "chunks": [
			                        {"content": "some text here"},
									{"content": "another text here"}],
			                    "attributes": []}`

				// Call PUT document endpoint with SYNC (strong consistency)
				putResp, _, putErr := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docPayload)
				Expect(putErr).To(BeNil())
				Expect(putResp).NotTo(BeNil())
				Expect(putResp.StatusCode).To(Equal(http.StatusCreated))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "PUT", http.StatusCreated)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ModelId, Type: HeaderEquals, Expected: "text-embedding-ada-002"},
					{Name: headers.ModelVersion, Type: HeaderEquals, Expected: "2"},
					{Name: headers.EmbeddingTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingCallsCount, Type: HeaderEquals, Expected: 2},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 1}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 2}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(putResp, checks)
			})

			It("POST v1 collections/{collectionName}/documents returns expected headers when response is 200", func() {
				// Setup endpoint
				path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert a document in the DB so POST returns a result
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Call POST endpoint to list document statuses
				postResp, _, postErr := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, "")
				Expect(postErr).To(BeNil())
				Expect(postResp).NotTo(BeNil())
				Expect(postResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", http.StatusOK)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderGreaterOrEqual, Expected: 1},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(postResp, checks)
			})

			It("DELETE v1 collections/{collectionName}/documents returns expected headers when response is 200", func() {
				// Setup endpoint
				path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert documents so DELETE has something to remove
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Call DELETE endpoint to remove all documents
				reqBody := ReadTestDataFile("test01/requests/delete-document.json")
				deleteResp, _, deleteErr := HttpCallWithHeadersAndApiCallStat("DELETE", endpointURI, ServerConfigurationHeaders, reqBody)
				Expect(deleteErr).To(BeNil())
				Expect(deleteResp).NotTo(BeNil())
				if deleteResp.StatusCode != http.StatusOK {
					body, _ := ReadResponseBody(deleteResp)
					fmt.Printf("DELETE response body: %s\n", body)
				}
				Expect(deleteResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "DELETE", http.StatusOK)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 2}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 7}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(deleteResp, checks)
			})

			It("GET v1 collections/{collectionName}/documents/{documentID} returns expected headers when response is 200", func() {
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert a document so GET has something to retrieve
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Use the first document ID for GET
				documentID := "DOC-1"
				path := fmt.Sprintf("/v1/%s/collections/%s/documents/%s", isolationID, collectionID, documentID)
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
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 1},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(getResp, checks)
			})

			It("DELETE v1 collections/{collectionName}/documents/{documentID} returns expected headers when response is 200", func() {
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert a document so DELETE has something to remove
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Use the first document ID for DELETE
				documentID := "DOC-1"
				path := fmt.Sprintf("/v1/%s/collections/%s/documents/%s", isolationID, collectionID, documentID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By(fmt.Sprintf("Calling DELETE endpoint %s", endpointURI))
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

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 2}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 7}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(deleteResp, checks)
			})

			It("PATCH v1 collections/{collectionName}/documents/{documentID} returns expected headers when response is 200", func() {
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert a document so PATCH has something to update
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Use the first document ID for PATCH
				documentID := "DOC-1"
				path := fmt.Sprintf("/v1/%s/collections/%s/documents/%s", isolationID, collectionID, documentID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				// Prepare patch payload (update attributes)
				reqBody := ReadTestDataFile("test01/requests/patch-document.json")

				By(fmt.Sprintf("Calling PATCH endpoint %s", endpointURI))
				patchResp, _, patchErr := HttpCallWithHeadersAndApiCallStat("PATCH", endpointURI, ServerConfigurationHeaders, reqBody)
				Expect(patchErr).To(BeNil())
				Expect(patchResp).NotTo(BeNil())
				Expect(patchResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "PATCH", http.StatusOK)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(patchResp, checks)
			})

			It("POST v1 collections/{collectionName}/document/delete-by-id returns expected headers when response is 200", func() {
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert a document so POST delete-by-id has something to delete
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Use the first document ID for delete-by-id
				documentID := "DOC-1"
				path := fmt.Sprintf("/v1/%s/collections/%s/document/delete-by-id", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				// Prepare request body for delete-by-id
				deleteByIdPayload := fmt.Sprintf(`{"id": "%s"}`, documentID)

				By(fmt.Sprintf("Calling POST endpoint %s", endpointURI))
				postResp, _, postErr := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, deleteByIdPayload)
				Expect(postErr).To(BeNil())
				Expect(postResp).NotTo(BeNil())
				Expect(postResp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", http.StatusOK)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 2}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 7}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(postResp, checks)
			})

			It("GET v2 collections/{collectionID}/documents/{documentID}/chunks returns expected headers when response is 200", func() {
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert a document so GET has something to retrieve
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Use the "DOC-3" for test
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

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(getResp, checks)
			})
		})

	})

})
