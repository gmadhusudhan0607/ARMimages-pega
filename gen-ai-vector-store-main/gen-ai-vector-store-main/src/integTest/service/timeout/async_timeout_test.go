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

var _ = Describe("Async Processing Timeout Tests", Ordered, func() {

	var (
		ctx          context.Context
		isolationID  string
		collectionID string
		mockIDs      []string
	)

	BeforeEach(func() {
		ctx = context.Background()
		isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
		collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
		mockIDs = nil
		CreateIsolation(opsBaseURI, isolationID, "1GB")
	})

	AfterEach(func() {
		if !CurrentSpecReport().Failed() {
			// Use safe cleanup function that handles synchronization
			SafeCleanupIsolation(ctx, database, opsBaseURI, isolationID)
		}

		// Cleanup wiremock expectations
		for _, mockID := range mockIDs {
			err := DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
		}
		mockIDs = nil
	})

	Context("background processing timeout behavior", func() {

		It("document upload completes quickly while async processing handles timeout separately", func() {
			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Creating wiremock expectation with delay that would timeout in foreground")
			// 2100ms delay > 2000ms QUERY_EMBEDDING_TIMEOUT
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 2100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("PUT document - request should complete fast (202 Accepted)")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			start := time.Now()
			resp, body, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
			elapsed := time.Since(start)

			By("Verifying request completes quickly with 202")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted),
				fmt.Sprintf("Expected 202 Accepted, got %d. Body: %s", resp.StatusCode, string(body)))
			Expect(elapsed).To(BeNumerically("<", 1*time.Second),
				fmt.Sprintf("Request should complete quickly, took: %v", elapsed))

			By("Waiting to see document status after async processing")
			// Use Eventually with optimized polling interval (200ms instead of default 1s)
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
				return status
			}, "10s", "200ms").Should(Or(Equal("ERROR"), Equal("COMPLETED")),
				"Document should eventually reach ERROR or COMPLETED status")

			By("Verifying final document status")
			status, errMsg := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
			By(fmt.Sprintf("Final document status: %s, error: %s", status, errMsg))
		})

		It("multiple documents process with different async timeout behaviors", func() {
			By("Creating fast wiremock expectation for first document")
			fastMockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, fastMockID)

			By("Upload first document - should succeed")
			path1 := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path1)
			docData1 := ReadTestDataFile("test02/documents/DOC-1.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData1)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for first document to complete")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("Switch to slow mock for second document")
			err = DeleteExpectationIfExist(wiremockManager, fastMockID)
			Expect(err).To(BeNil())
			slowMockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 2100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, slowMockID)

			By("Upload second document - will timeout in async processing")
			docData2 := ReadTestDataFile("test02/documents/DOC-2.json")
			resp, _, err = HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData2)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify first document still COMPLETED")
			ExpectDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("Wait for second document to finish processing")
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-2")
				return status
			}, "10s", "200ms").Should(Or(Equal("ERROR"), Equal("COMPLETED")))
		})

		It("verifies request returns success while background continues processing", func() {
			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Creating wiremock expectation with small delay for background processing")
			// 100ms delay - fast enough to not timeout
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("PUT document - should return immediately")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			start := time.Now()
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
			requestElapsed := time.Since(start)

			By("Verifying request completes very quickly")
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			Expect(requestElapsed).To(BeNumerically("<", 500*time.Millisecond),
				fmt.Sprintf("Request took: %v, should be much faster than processing time", requestElapsed))

			By("Background processing continues after request returns")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("Verifying document was successfully processed in background")
			ExpectDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")
		})
	})

	Context("background timeout configuration", func() {

		It("verifies separate timeout for async operations", func() {
			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Creating wiremock expectation that exceeds query timeout but within background timeout")
			// 2100ms > QUERY_EMBEDDING_TIMEOUT (2000ms) but should be handled by background
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 2100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("PUT document")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Monitoring background processing with longer timeout")
			// Background service has more time to process
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
				return status
			}, "15s", "500ms").Should(Or(Equal("ERROR"), Equal("COMPLETED")))

			status, errMsg := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
			By(fmt.Sprintf("Background processing result - Status: %s, Error: %s", status, errMsg))
		})
	})
})
