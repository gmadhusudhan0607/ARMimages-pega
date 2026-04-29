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

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("PUT /v1/{isolationID}/collections/{collectionName}/documents - Timeout Handling", Ordered, func() {

	var (
		ctx          context.Context
		isolationID  string
		collectionID string
		mockIDs      []string
	)

	_ = Context("calling service with strong consistency", func() {

		BeforeEach(func() {
			ctx = context.Background()
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			mockIDs = nil
			CreateIsolation(opsBaseURI, isolationID, "1GB")
		})

		AfterEach(func() {
			if !CurrentSpecReport().Failed() {
				SafeCleanupIsolation(ctx, database, opsBaseURI, isolationID)
			}

			// Cleanup wiremock expectations
			for _, mockID := range mockIDs {
				err := DeleteExpectationIfExist(wiremockManager, mockID)
				Expect(err).To(BeNil())
			}
			mockIDs = nil
		})

		_ = Context("synchronous timeout handling", func() {

			It("returns document created without retry when service responds quickly", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/documents?consistencyLevel=strong", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating wiremock expectation that responds quickly")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, mockID)

				By("PUT document with strong consistency - should succeed without retry")
				docData := ReadTestDataFile("test02/documents/DOC-1.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))

				By("Verify mock was called exactly 2 times (one per chunk, no retry)")
				callCount, err := GetCallCountByMockID(wiremockManager, mockID)
				Expect(err).To(BeNil())
				Expect(callCount).To(Equal(2),
					fmt.Sprintf("Expected exactly 2 calls to fast mock (ID: %s) for 2 chunks, but got %d", mockID, callCount))
			})

			It("exhausts retries and returns gateway timeout when service is slow", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/documents?consistencyLevel=strong", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating wiremock expectation with delay to simulate timeout")
				// 3000ms delay > 2000ms QUERY_EMBEDDING_TIMEOUT
				delayMockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, delayMockID)

				By("PUT document with strong consistency - should timeout and retry")
				docData := ReadTestDataFile("test02/documents/DOC-1.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusGatewayTimeout))

				By("Verify delay mock was called exactly 4 times (2 chunks × 2 attempts each)")
				callCount, err := GetCallCountByMockID(wiremockManager, delayMockID)
				Expect(err).To(BeNil())
				Expect(callCount).To(Equal(4),
					fmt.Sprintf("Expected exactly 4 calls to delay mock (ID: %s) for 2 chunks with retry, but got %d", delayMockID, callCount))
			})

			It("succeeds after retry when initial request times out", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/documents?consistencyLevel=strong", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating scenario-based wiremock expectation")
				// First call with delay (timeout), second call without delay (success)
				scenarioMockIDs, err := CreateExpectationEmbeddingAdaWithTimeoutRetryScenario(wiremockManager, isolationID, 3000)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, scenarioMockIDs...)

				By("PUT document with strong consistency - first call will timeout, retry will succeed")
				docData := ReadTestDataFile("test02/documents/DOC-1.json")
				resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated))

				By("Verify retry mechanism worked: at least 1 timeout occurred and total calls in range [3,4]")
				// Due to concurrent chunk processing, the WireMock scenario state (global) transitions after the
				// first chunk hits the timeout stub, so exact per-mock counts are non-deterministic:
				// - 1 chunk times out:  1 timeout + 2 success = 3 total calls (common case)
				// - 2 chunks time out:  2 timeout + 2 success = 4 total calls (rare race)
				// We verify that: 1) at least 1 timeout occurred, 2) total is in [3, 4]
				timeoutCallCount, err := GetCallCountByMockID(wiremockManager, scenarioMockIDs[0])
				Expect(err).To(BeNil())
				Expect(timeoutCallCount).To(BeNumerically(">=", 1),
					fmt.Sprintf("Expected at least 1 call to timeout mock (ID: %s) to verify retry was triggered, but got %d", scenarioMockIDs[0], timeoutCallCount))

				successCallCount, err := GetCallCountByMockID(wiremockManager, scenarioMockIDs[1])
				Expect(err).To(BeNil())

				totalCallCount := timeoutCallCount + successCallCount
				Expect(totalCallCount).To(BeNumerically(">=", 3),
					fmt.Sprintf("Expected at least 3 total calls (timeout: %d + success: %d) for 2 chunks with retry, but got %d",
						timeoutCallCount, successCallCount, totalCallCount))
				Expect(totalCallCount).To(BeNumerically("<=", 4),
					fmt.Sprintf("Expected at most 4 total calls (timeout: %d + success: %d) for 2 chunks with retry, but got %d",
						timeoutCallCount, successCallCount, totalCallCount))
			})

			It("completes fast document upload successfully within timeout", func() {
				path := fmt.Sprintf("/v1/%s/collections/%s/documents?consistencyLevel=strong", isolationID, collectionID)
				endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

				By("Creating normal wiremock expectation")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, mockID)

				By("Making fast PUT request")
				start := time.Now()
				docData := ReadTestDataFile("test02/documents/DOC-1.json")
				resp, body, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
				elapsed := time.Since(start)

				By("Verifying successful response")
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusCreated),
					fmt.Sprintf("Expected 201 Created, got %d. Body: %s", resp.StatusCode, string(body)))
				Expect(elapsed).To(BeNumerically("<", 5*time.Second),
					fmt.Sprintf("Fast PUT took too long: %v", elapsed))
			})
		})
	})
})
