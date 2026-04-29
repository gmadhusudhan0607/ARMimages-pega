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

		_ = Context("/v1/{isolationID}/collections/{collectionName}/query/*", func() {

			It("POST v1 collections/{collectionName}/query/chunks returns expected headers when response is 200", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/query/chunks", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				By("Insert documents")
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Waiting for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				By("Query chunks with test data")
				queryData := ReadTestDataFile("test01/requests/query-chunks.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", resp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					// Check headers for vector store service
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ModelId, Type: HeaderEquals, Expected: "text-embedding-ada-002"},
					{Name: headers.ModelVersion, Type: HeaderEquals, Expected: "2"},
					{Name: headers.EmbeddingTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingCallsCount, Type: HeaderEquals, Expected: 1},
					{Name: headers.EmbeddingRetryCount, Type: HeaderEquals, Expected: 0},        // No retry
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 9}, // unique attributes in test01/documents/*

					// Processing overhead headers
					{Name: headers.ProcessingDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.OverheadMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingNetOverheadMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},

					// Added checks for gateway headers
					{Name: headers.GatewayResponseTimeMs, Type: HeaderEquals, Expected: 111},
					{Name: headers.GatewayInputTokens, Type: HeaderEquals, Expected: 222},
					{Name: headers.GatewayOutputTokens, Type: HeaderEquals, Expected: 333},
					{Name: headers.GatewayTokensPerSecond, Type: HeaderEquals, Expected: 444},
					{Name: headers.GatewayModelId, Type: HeaderEquals, Expected: "text-embedding-ada-002"},
					{Name: headers.GatewayRegion, Type: HeaderEquals, Expected: "us-east-1"},
					{Name: headers.GatewayRetryCount, Type: HeaderEquals, Expected: 2},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(resp, checks)
			})

			It("POST v1 collections/{collectionName}/query/chunks returns expected headers when response is 200 and GW headers missed", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/query/chunks", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings without gateway headers")
				mockID, err := CreateExpectationEmbeddingAdaWithoutGatewayHeaders(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				By("Insert documents")
				docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
				By("Waiting for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				By("Query chunks with test data")
				queryData := ReadTestDataFile("test02/requests/query-chunks.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", resp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					// Check headers for vector store service
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ModelId, Type: HeaderEquals, Expected: "text-embedding-ada-002"},
					{Name: headers.ModelVersion, Type: HeaderEquals, Expected: "2"},
					{Name: headers.EmbeddingTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingCallsCount, Type: HeaderEquals, Expected: 1},
					{Name: headers.EmbeddingRetryCount, Type: HeaderEquals, Expected: 0},        // No retry
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 2}, // unique attributes in test01/documents/*

					// Processing overhead headers
					{Name: headers.ProcessingDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.OverheadMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingNetOverheadMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},

					// Added checks for gateway headers
					{Name: headers.GatewayResponseTimeMs, Type: HeaderEquals, Expected: -1},
					{Name: headers.GatewayInputTokens, Type: HeaderEquals, Expected: -1},
					{Name: headers.GatewayOutputTokens, Type: HeaderEquals, Expected: -1},
					{Name: headers.GatewayTokensPerSecond, Type: HeaderEquals, Expected: -1},
					{Name: headers.GatewayModelId, Type: HeaderEquals, Expected: "not-set"},
					{Name: headers.GatewayRegion, Type: HeaderEquals, Expected: "not-set"},
					{Name: headers.GatewayRetryCount, Type: HeaderEquals, Expected: -1},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 1}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 2}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(resp, checks)
			})

			It("POST v1/{isolationID}/collections/{collectionName}/query/documents returns expected headers when response is 200", func() {
				// Setup endpoint
				path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert documents for query
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				By("Query documents with test data")
				queryData := ReadTestDataFile("test01/requests/query-documents.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", resp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ModelId, Type: HeaderEquals, Expected: "text-embedding-ada-002"},
					{Name: headers.ModelVersion, Type: HeaderEquals, Expected: "2"},
					{Name: headers.EmbeddingTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingCallsCount, Type: HeaderEquals, Expected: 1},
					{Name: headers.EmbeddingRetryCount, Type: HeaderEquals, Expected: 0}, // No retry
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 3},

					// Processing overhead headers
					{Name: headers.ProcessingDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.OverheadMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingNetOverheadMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},

					// Added checks for gateway headers
					{Name: headers.GatewayResponseTimeMs, Type: HeaderEquals, Expected: 111},
					{Name: headers.GatewayInputTokens, Type: HeaderEquals, Expected: 222},
					{Name: headers.GatewayOutputTokens, Type: HeaderEquals, Expected: 333},
					{Name: headers.GatewayTokensPerSecond, Type: HeaderEquals, Expected: 444},
					{Name: headers.GatewayModelId, Type: HeaderEquals, Expected: "text-embedding-ada-002"},
					{Name: headers.GatewayRegion, Type: HeaderEquals, Expected: "us-east-1"},
					{Name: headers.GatewayRetryCount, Type: HeaderEquals, Expected: 2},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(resp, checks)
			})

			It("POST v1/{isolationID}/collections/{collectionName}/query/documents returns expected headers when response is 200 and GW headers missed", func() {
				// Setup endpoint
				path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings without gateway headers")
				mockID, err := CreateExpectationEmbeddingAdaWithoutGatewayHeaders(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert documents for query
				docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				By("Query documents with test data")
				queryData := ReadTestDataFile("test02/requests/query-documents.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", resp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ModelId, Type: HeaderEquals, Expected: "text-embedding-ada-002"},
					{Name: headers.ModelVersion, Type: HeaderEquals, Expected: "2"},
					{Name: headers.EmbeddingTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingCallsCount, Type: HeaderEquals, Expected: 1},
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 1},
					{Name: headers.EmbeddingRetryCount, Type: HeaderEquals, Expected: 0},

					// Processing overhead headers
					{Name: headers.ProcessingDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.OverheadMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.EmbeddingNetOverheadMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},

					// Added checks for gateway headers
					{Name: headers.GatewayResponseTimeMs, Type: HeaderEquals, Expected: -1},
					{Name: headers.GatewayInputTokens, Type: HeaderEquals, Expected: -1},
					{Name: headers.GatewayOutputTokens, Type: HeaderEquals, Expected: -1},
					{Name: headers.GatewayTokensPerSecond, Type: HeaderEquals, Expected: -1},
					{Name: headers.GatewayModelId, Type: HeaderEquals, Expected: "not-set"},
					{Name: headers.GatewayRegion, Type: HeaderEquals, Expected: "not-set"},
					{Name: headers.GatewayRetryCount, Type: HeaderEquals, Expected: -1},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 1}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 2}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(resp, checks)
			})

			It("POST v1/{isolationID}/collections/{collectionName}/attributes returns expected headers when response is 200", func() {
				// Setup endpoint
				path := fmt.Sprintf("/v1/%s/collections/%s/attributes", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = []string{mockID}

				// Insert documents for query
				docIDs := UpsertDocumentsFromDir("test01/documents", svcBaseURI, isolationID, collectionID)
				By("Wait for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				// Call endpoint
				By("Call endpoint attributes")
				attrData := ReadTestDataFile("test01/requests/query-attributes.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, attrData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				expectedHeaderNames, err := GetExpectedResponseHeadersForEndpoint(svcBaseURI, path, "POST", resp.StatusCode)
				Expect(err).To(BeNil())
				Expect(expectedHeaderNames).NotTo(BeNil())

				checks := []HeaderCheck{
					{Name: headers.RequestDurationMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.DbQueryTimeMs, Type: HeaderBetween, Expected: [2]int{0, 10000}},
					{Name: headers.ResponseReturnedItemsCount, Type: HeaderEquals, Expected: 3},

					// DB metrics
					{Name: headers.DocumentsCount, Type: HeaderBetween, Expected: [2]int{0, 3}},
					{Name: headers.VectorsCount, Type: HeaderBetween, Expected: [2]int{0, 9}},
				}
				ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames, checks)
				ExpectHeadersFlexible(resp, checks)
			})

		})

	})

})
