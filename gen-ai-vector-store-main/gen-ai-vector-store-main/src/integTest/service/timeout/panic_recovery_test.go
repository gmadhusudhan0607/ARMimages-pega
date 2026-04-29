// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package timeout_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("Panic Recovery in Async Processing Tests", Ordered, func() {

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

	Context("error handling during async processing", func() {

		It("handles embedding generation errors gracefully", func() {
			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Creating wiremock expectation that returns error")
			// Return 500 error to trigger error handling
			mockID, err := CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, 500)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("PUT document - should accept request")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Waiting for document to reach ERROR status")
			// Use Eventually with optimized polling
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
				return status
			}, "10s", "200ms").Should(Equal("ERROR"),
				"Document should reach ERROR status after embedding failure")

			By("Verifying document has error details")
			status, errMsg := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
			Expect(status).To(Equal("ERROR"))
			Expect(errMsg).NotTo(BeEmpty(), "Error message should be recorded")
			By(fmt.Sprintf("Error message: %s", errMsg))
		})

		It("verifies other documents continue processing after one fails", func() {
			By("Creating normal wiremock expectation for successful documents")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Upload first document - should succeed")
			docData1 := ReadTestDataFile("test02/documents/DOC-1.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData1)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for first document to complete before switching mocks")
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
				return status
			}, "10s", "200ms").Should(Equal("COMPLETED"))

			By("Switch to error mock for second document")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			errorMockID, err := CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, 500)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, errorMockID)

			By("Upload second document - will fail")
			docData2 := ReadTestDataFile("test02/documents/DOC-2.json")
			resp, _, err = HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData2)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for second document to reach ERROR status before switching mocks")
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-2")
				return status
			}, "10s", "200ms").Should(Equal("ERROR"))

			By("Switch back to normal mock for third document")
			err = DeleteExpectationIfExist(wiremockManager, errorMockID)
			Expect(err).To(BeNil())
			normalMockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, normalMockID)

			By("Upload third document - should succeed")
			docData3 := ReadTestDataFile("test02/documents/DOC-3.json")
			resp, _, err = HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData3)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify first document completed successfully")
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
				return status
			}, "10s", "200ms").Should(Equal("COMPLETED"))

			By("Verify second document still has ERROR status")
			status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-2")
			Expect(status).To(Equal("ERROR"), "DOC-2 should remain in ERROR status")

			By("Verify third document completed successfully despite second failing")
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-3")
				return status
			}, "10s", "200ms").Should(Equal("COMPLETED"))
		})

		It("handles multiple consecutive errors without system failure", func() {
			By("Creating error wiremock expectation")
			mockID, err := CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, 503)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Upload multiple documents that will all fail")
			for i := 1; i <= 3; i++ {
				docID := fmt.Sprintf("DOC-%d", i)
				docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Test content %d"}]}`, docID, i)
				resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			}

			By("Verify all documents reach ERROR status")
			for i := 1; i <= 3; i++ {
				docID := fmt.Sprintf("DOC-%d", i)
				Eventually(func() string {
					status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, docID)
					return status
				}, "10s", "200ms").Should(Equal("ERROR"),
					fmt.Sprintf("Document %s should reach ERROR status", docID))
			}

			By("Verify system remains operational after multiple errors")
			// Switch to normal mock and upload a new document
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			normalMockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, normalMockID)

			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
				return status
			}, "10s", "200ms").Should(Equal("COMPLETED"),
				"System should recover and process new documents successfully")
		})
	})

	Context("retry behavior with errors", func() {

		It("handles error retry scenarios appropriately", func() {
			By("Creating retry scenario: 2 errors then success")
			var err error
			mockIDs, err = CreateExpectationEmbeddingAdaWithRetryScenario(wiremockManager, isolationID, []int{503, 503})
			Expect(err).To(BeNil())

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Upload document - should retry and eventually succeed")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify document eventually completes after retries")
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
				return status
			}, "45s", "200ms").Should(Equal("COMPLETED"),
				"Document should complete after successful retry")
		})
	})
})
