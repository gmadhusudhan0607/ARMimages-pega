// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package timeout_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("Concurrent Processing and Context Cancellation Tests", Ordered, func() {

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
		CreateCollection(svcBaseURI, isolationID, collectionID)
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

	Context("concurrent document processing with semaphore limits", func() {

		It("processes multiple documents concurrently within semaphore limits", func() {
			By("Creating fast wiremock expectation")
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Uploading multiple documents that will be processed concurrently")
			numDocs := 15
			var wg sync.WaitGroup

			for i := 1; i <= numDocs; i++ {
				wg.Add(1)
				go func(index int) {
					defer GinkgoRecover()
					defer wg.Done()

					docID := fmt.Sprintf("DOC-%d", index)
					docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Concurrent test %d"}]}`, docID, index)
					resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
					Expect(err).To(BeNil())
					Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
				}(i)
			}

			wg.Wait()

			By("Verifying all documents eventually complete")
			Eventually(func() int {
				completedCount := 0
				for i := 1; i <= numDocs; i++ {
					docID := fmt.Sprintf("DOC-%d", i)
					status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, docID)
					if status == "COMPLETED" {
						completedCount++
					}
				}
				return completedCount
			}, "30s", "500ms").Should(Equal(numDocs),
				fmt.Sprintf("All %d documents should complete processing", numDocs))
		})

		It("verifies no deadlocks occur with concurrent processing", func() {
			By("Creating wiremock expectation with variable delay")
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 200)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Uploading documents in waves to test semaphore management")
			numWaves := 3
			docsPerWave := 5

			for wave := 1; wave <= numWaves; wave++ {
				By(fmt.Sprintf("Wave %d: uploading %d documents", wave, docsPerWave))
				var wg sync.WaitGroup

				for i := 1; i <= docsPerWave; i++ {
					wg.Add(1)
					go func(waveNum, index int) {
						defer GinkgoRecover()
						defer wg.Done()

						docID := fmt.Sprintf("DOC-W%d-D%d", waveNum, index)
						docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Wave %d doc %d"}]}`, docID, waveNum, index)
						resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
						Expect(err).To(BeNil())
						Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
					}(wave, i)
				}

				wg.Wait()
				By(fmt.Sprintf("Wave %d completed without deadlock", wave))
			}

			By("Verifying all documents are processed")
			totalDocs := numWaves * docsPerWave
			Eventually(func() int {
				completedCount := 0
				for wave := 1; wave <= numWaves; wave++ {
					for i := 1; i <= docsPerWave; i++ {
						docID := fmt.Sprintf("DOC-W%d-D%d", wave, i)
						status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, docID)
						if status == "COMPLETED" || status == "ERROR" {
							completedCount++
						}
					}
				}
				return completedCount
			}, "40s", "500ms").Should(Equal(totalDocs),
				"All documents should finish processing without deadlock")
		})
	})

	Context("error propagation during concurrent processing", func() {

		It("handles errors gracefully without affecting other documents", func() {
			By("Creating scenario with mixed success and failure")
			// Start with success mock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Upload documents that will succeed")
			for i := 1; i <= 3; i++ {
				docID := fmt.Sprintf("SUCCESS-%d", i)
				docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Success doc %d"}]}`, docID, i)
				resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			}

			By("Wait for success documents to complete before switching mock")
			for i := 1; i <= 3; i++ {
				docID := fmt.Sprintf("SUCCESS-%d", i)
				Eventually(func() string {
					status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, docID)
					return status
				}, "10s", "200ms").Should(Equal("COMPLETED"))
			}

			By("Switch to error mock")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			errorMockID, err := CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, 500)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, errorMockID)

			By("Upload documents that will fail")
			for i := 1; i <= 3; i++ {
				docID := fmt.Sprintf("FAILURE-%d", i)
				docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Failure doc %d"}]}`, docID, i)
				resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
				Expect(err).To(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
			}

			By("Verify success documents completed")
			for i := 1; i <= 3; i++ {
				docID := fmt.Sprintf("SUCCESS-%d", i)
				Eventually(func() string {
					status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, docID)
					return status
				}, "30s", "200ms").Should(Equal("COMPLETED"))
			}

			By("Verify failure documents have error status")
			for i := 1; i <= 3; i++ {
				docID := fmt.Sprintf("FAILURE-%d", i)
				Eventually(func() string {
					status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, docID)
					return status
				}, "30s", "200ms").Should(Equal("ERROR"))
			}
		})

		It("verifies proper error messages are propagated for failed documents", func() {
			By("Creating error wiremock expectation")
			mockID, err := CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, 503)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Upload document that will fail")
			docData := ReadTestDataFile("test02/documents/DOC-1.json")
			resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for ERROR status")
			Eventually(func() string {
				status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
				return status
			}, "10s", "200ms").Should(Equal("ERROR"))

			By("Verify error message is recorded")
			_, errMsg := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
			Expect(errMsg).NotTo(BeEmpty(), "Error message should be recorded for failed document")
			By(fmt.Sprintf("Error message recorded: %s", errMsg))
		})
	})

	Context("database transaction handling under concurrent load", func() {

		It("verifies no zombie transactions after concurrent operations", func() {
			By("Creating fast wiremock expectation")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Perform concurrent operations")
			numOps := 10
			var wg sync.WaitGroup

			for i := 1; i <= numOps; i++ {
				wg.Add(1)
				go func(index int) {
					defer GinkgoRecover()
					defer wg.Done()

					docID := fmt.Sprintf("DOC-%d", index)
					docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Transaction test %d"}]}`, docID, index)
					resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
					Expect(err).To(BeNil())
					Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
				}(i)
			}

			wg.Wait()

			By("Wait for all operations to complete")
			Eventually(func() int {
				completedCount := 0
				for i := 1; i <= numOps; i++ {
					docID := fmt.Sprintf("DOC-%d", i)
					status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, docID)
					if status == "COMPLETED" || status == "ERROR" {
						completedCount++
					}
				}
				return completedCount
			}, "20s", "500ms").Should(Equal(numOps))

			By("Verify no idle transactions remain")
			ExpectNoIdleTransactionsLeft(ctx, database, "testuser")
		})
	})
})
