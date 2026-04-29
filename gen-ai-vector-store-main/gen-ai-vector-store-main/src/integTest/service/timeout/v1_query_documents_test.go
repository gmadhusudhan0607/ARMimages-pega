// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package timeout_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("POST /v1/{isolationID}/collections/{collectionName}/query/documents", Ordered, func() {

	var (
		ctx          context.Context
		isolationID  string
		collectionID string
		mockIDs      []string // Track mock IDs for cleanup
	)

	_ = Context("calling service", func() {

		BeforeEach(func() {
			ctx = context.Background()
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			mockIDs = nil // Reset mock IDs for each test
			CreateIsolation(opsBaseURI, isolationID, "1GB")
		})

		AfterEach(func() {
			if !CurrentSpecReport().Failed() {
				// Use safe cleanup function that handles synchronization
				SafeCleanupIsolation(ctx, database, opsBaseURI, isolationID)
			}

			// Cleanup wiremock expectations created by test
			for _, mockID := range mockIDs {
				err := DeleteExpectationIfExist(wiremockManager, mockID)
				Expect(err).To(BeNil())
			}
			mockIDs = nil // Reset the slice for next test
		})

		_ = Context("timeout handling", func() {

			It("returns documents without retry when service responds quickly", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating normal wiremock expectation for document upload")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, mockID)

				By("Insert documents")
				docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
				By("Waiting for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				By("Deleting previous mock and creating expectation that responds quickly")
				// Delete the previous normal expectation
				err = DeleteExpectationIfExist(wiremockManager, mockID)
				Expect(err).To(BeNil())
				// Create new expectation without delay to ensure response within timeout
				fastMockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, fastMockID)

				By("Query documents with test data - service should succeed without retry")
				queryData := ReadTestDataFile("test02/requests/query-documents.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				By("Verify mock was called exactly 1 time (no retry)")
				callCount, err := GetCallCountByMockID(wiremockManager, fastMockID)
				Expect(err).To(BeNil())
				Expect(callCount).To(Equal(1),
					fmt.Sprintf("Expected exactly 1 call to fast mock (ID: %s), but got %d", fastMockID, callCount))

			})

			It("exhausts retries and returns gateway timeout when service is slow", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating normal wiremock expectation for document upload")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, mockID)

				By("Insert documents")
				docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
				By("Waiting for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				By("Deleting previous mock and creating expectation with delay to simulate timeout")
				// Delete the previous normal expectation
				err = DeleteExpectationIfExist(wiremockManager, mockID)
				Expect(err).To(BeNil())
				// Set delay to 3000ms (3 seconds) which is greater than QUERY_EMBEDDING_TIMEOUT (2000 milliseconds)
				delayMockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, delayMockID)

				By("Query documents with test data - service should timeout and retry")
				queryData := ReadTestDataFile("test02/requests/query-documents.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusGatewayTimeout))

				By("Verify delay mock was called exactly 2 times (initial + 1 retry)")
				callCount, err := GetCallCountByMockID(wiremockManager, delayMockID)
				Expect(err).To(BeNil())
				Expect(callCount).To(Equal(2),
					fmt.Sprintf("Expected exactly 2 calls to delay mock (ID: %s), but got %d", delayMockID, callCount))

			})

			It("succeeds after retry when initial request times out", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating normal wiremock expectation for document upload")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, mockID)

				By("Insert documents")
				docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
				By("Waiting for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				By("Deleting previous mock and creating scenario-based expectation")
				// Delete the previous normal expectation
				err = DeleteExpectationIfExist(wiremockManager, mockID)
				Expect(err).To(BeNil())
				// Create scenario: first call with delay (timeout), second call without delay (success)
				// 3000ms delay is greater than QUERY_EMBEDDING_TIMEOUT (2000 milliseconds)
				scenarioMockIDs, err := CreateExpectationEmbeddingAdaWithTimeoutRetryScenario(wiremockManager, isolationID, 3000)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, scenarioMockIDs...)

				By("Query documents with test data - first call will timeout, retry will succeed")
				queryData := ReadTestDataFile("test02/requests/query-documents.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				By("Verify both scenario mocks were called exactly once each (2 total calls)")
				// Verify the timeout mock (first in scenario) was called once
				timeoutCallCount, err := GetCallCountByMockID(wiremockManager, scenarioMockIDs[0])
				Expect(err).To(BeNil())
				Expect(timeoutCallCount).To(Equal(1),
					fmt.Sprintf("Expected exactly 1 call to timeout mock (ID: %s), but got %d", scenarioMockIDs[0], timeoutCallCount))

				// Verify the success mock (second in scenario) was called once
				successCallCount, err := GetCallCountByMockID(wiremockManager, scenarioMockIDs[1])
				Expect(err).To(BeNil())
				Expect(successCallCount).To(Equal(1),
					fmt.Sprintf("Expected exactly 1 call to success mock (ID: %s), but got %d", scenarioMockIDs[1], successCallCount))

			})

			It("completes fast query successfully within timeout", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating normal wiremock expectation for document upload")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, mockID)

				By("Insert test document")
				docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
				By("Waiting for completion")
				WaitForDocumentsStatusInDB(context.Background(), database, isolationID, collectionID, docIDs, resources.StatusCompleted)

				By("Making fast query request")
				start := time.Now()
				queryData := ReadTestDataFile("test02/requests/query-documents.json")
				resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)
				elapsed := time.Since(start)

				By("Verifying successful response")
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK),
					fmt.Sprintf("Expected 200 OK, got %d. Body: %s", resp.StatusCode, string(body)))
				Expect(elapsed).To(BeNumerically("<", 5*time.Second),
					fmt.Sprintf("Fast query took too long: %v", elapsed))
			})

		})
	})
})
