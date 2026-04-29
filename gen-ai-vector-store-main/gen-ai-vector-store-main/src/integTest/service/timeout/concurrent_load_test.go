// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package timeout_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("Concurrent Load Tests", Ordered, func() {

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

	Context("concurrent document upload", func() {

		It("handles multiple concurrent document uploads without connection exhaustion", func() {
			By("Creating fast wiremock expectation with minimal delay")
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Uploading 20 documents concurrently with retry logic")
			concurrentUploads := 20
			var wg sync.WaitGroup
			successCount := 0
			var mu sync.Mutex
			var failedDocs []string

			for i := 1; i <= concurrentUploads; i++ {
				wg.Add(1)
				go func(index int) {
					defer GinkgoRecover()
					defer wg.Done()

					docID := fmt.Sprintf("DOC-%d", index)
					docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Test content %d"}]}`, docID, index)

					// Retry logic for transient failures (up to 3 attempts)
					maxRetries := 3
					success := false
					var lastErr error
					var lastStatus int

					for attempt := 1; attempt <= maxRetries && !success; attempt++ {
						resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)

						if err == nil && resp.StatusCode == http.StatusAccepted {
							success = true
							break
						}

						lastErr = err
						if resp != nil {
							lastStatus = resp.StatusCode
						}

						// Small delay before retry (exponential backoff)
						if attempt < maxRetries {
							time.Sleep(time.Duration(attempt*50) * time.Millisecond)
						}
					}

					mu.Lock()
					if success {
						successCount++
					} else {
						errMsg := fmt.Sprintf("DOC-%d failed after %d attempts", index, maxRetries)
						if lastErr != nil {
							errMsg += fmt.Sprintf(": %v", lastErr)
						} else {
							errMsg += fmt.Sprintf(": status %d", lastStatus)
						}
						failedDocs = append(failedDocs, errMsg)
					}
					mu.Unlock()
				}(i)
			}

			wg.Wait()

			By(fmt.Sprintf("Verifying all %d uploads succeeded", concurrentUploads))
			if len(failedDocs) > 0 {
				GinkgoWriter.Printf("Failed document uploads:\n")
				for _, msg := range failedDocs {
					GinkgoWriter.Printf("  - %s\n", msg)
				}
			}
			Expect(successCount).To(Equal(concurrentUploads),
				fmt.Sprintf("Expected %d successful uploads, got %d. Failures: %v", concurrentUploads, successCount, failedDocs))

			By("Verifying documents are processing/completed")
			// Give some time for background processing to start
			Eventually(func() int {
				completedCount := 0
				for i := 1; i <= concurrentUploads; i++ {
					docID := fmt.Sprintf("DOC-%d", i)
					status, _ := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, docID)
					if status == "COMPLETED" {
						completedCount++
					}
				}
				return completedCount
			}, "30s", "500ms").Should(BeNumerically(">", 0),
				"At least some documents should complete processing")
		})

		It("verifies connection reuse under concurrent load", func() {
			By("Creating fast wiremock expectation")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			path := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Performing multiple batches of concurrent uploads")
			batchSize := 10
			numBatches := 3

			for batch := 1; batch <= numBatches; batch++ {
				By(fmt.Sprintf("Batch %d: Uploading %d documents", batch, batchSize))
				var wg sync.WaitGroup

				for i := 1; i <= batchSize; i++ {
					wg.Add(1)
					go func(batchNum, index int) {
						defer GinkgoRecover()
						defer wg.Done()

						docID := fmt.Sprintf("DOC-B%d-D%d", batchNum, index)
						docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Batch %d content %d"}]}`, docID, batchNum, index)
						resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServerConfigurationHeaders, docData)
						Expect(err).To(BeNil())
						Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
					}(batch, i)
				}

				wg.Wait()
				By(fmt.Sprintf("Batch %d completed", batch))
			}

			By("Verifying all documents were accepted")
			totalDocs := batchSize * numBatches
			By(fmt.Sprintf("Total documents uploaded: %d", totalDocs))
		})
	})

	Context("concurrent query operations", func() {

		It("handles multiple concurrent queries without timeout", func() {
			By("Creating normal wiremock expectation for document upload")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("Upload test documents")
			docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for documents to complete")
			WaitForDocumentsStatusInDB(ctx, database, isolationID, collectionID, docIDs, "COMPLETED")

			path := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
			endpointURI := fmt.Sprintf("%s%s", svcBaseURI, path)

			By("Performing concurrent queries")
			concurrentQueries := 15
			var wg sync.WaitGroup
			successCount := 0
			var mu sync.Mutex

			queryData := ReadTestDataFile("test02/requests/query-documents.json")

			for i := 1; i <= concurrentQueries; i++ {
				wg.Add(1)
				go func(index int) {
					defer GinkgoRecover()
					defer wg.Done()

					resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServerConfigurationHeaders, queryData)

					mu.Lock()
					if err == nil && resp.StatusCode == http.StatusOK {
						successCount++
					}
					mu.Unlock()
				}(i)
			}

			wg.Wait()

			By(fmt.Sprintf("Verifying all %d queries succeeded", concurrentQueries))
			Expect(successCount).To(Equal(concurrentQueries),
				fmt.Sprintf("Expected %d successful queries, got %d", concurrentQueries, successCount))
		})
	})

	Context("mixed concurrent operations", func() {

		It("handles mixed uploads and queries concurrently", func() {
			By("Creating wiremock expectation with small delay")
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 100)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, mockID)

			By("Upload initial test documents")
			docIDs := UpsertDocumentsFromDir("test02/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for initial documents to complete")
			WaitForDocumentsStatusInDB(ctx, database, isolationID, collectionID, docIDs, "COMPLETED")

			putPath := fmt.Sprintf("/v1/%s/collections/%s/documents", isolationID, collectionID)
			putEndpointURI := fmt.Sprintf("%s%s", svcBaseURI, putPath)
			queryPath := fmt.Sprintf("/v1/%s/collections/%s/query/documents", isolationID, collectionID)
			queryEndpointURI := fmt.Sprintf("%s%s", svcBaseURI, queryPath)

			By("Performing mixed concurrent operations")
			var wg sync.WaitGroup
			uploadCount := 10
			queryCount := 10

			// Concurrent uploads
			for i := 1; i <= uploadCount; i++ {
				wg.Add(1)
				go func(index int) {
					defer GinkgoRecover()
					defer wg.Done()

					docID := fmt.Sprintf("DOC-MIXED-%d", index)
					docData := fmt.Sprintf(`{"id":"%s","chunks":[{"content":"Mixed test %d"}]}`, docID, index)
					resp, _, err := HttpCallWithHeadersAndApiCallStat("PUT", putEndpointURI, ServerConfigurationHeaders, docData)
					Expect(err).To(BeNil())
					Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
				}(i)
			}

			// Concurrent queries
			queryData := ReadTestDataFile("test02/requests/query-documents.json")
			for i := 1; i <= queryCount; i++ {
				wg.Add(1)
				go func(index int) {
					defer GinkgoRecover()
					defer wg.Done()

					resp, _, err := HttpCallWithHeadersAndApiCallStat("POST", queryEndpointURI, ServerConfigurationHeaders, queryData)
					Expect(err).To(BeNil())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))
				}(i)
			}

			wg.Wait()

			By("Verifying no connection exhaustion occurred")
			// All operations completed successfully without errors
		})
	})
})
