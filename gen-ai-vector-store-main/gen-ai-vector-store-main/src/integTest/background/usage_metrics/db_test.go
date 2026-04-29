// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("Testing database metrics collection and upload to PDC", func() {
	var (
		ctx          context.Context
		isolationID  string
		collectionID string
		mockIDs      []string // Track mock IDs for cleanup
	)

	BeforeEach(func() {
		ctx = context.Background()
		isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
		collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
		mockIDs = nil // Reset mock IDs for each test

		By("Creating isolation")
		CreateIsolation(opsBaseURI, isolationID, "1GB")
		ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
	})

	AfterEach(func() {
		// Do not clean up if test failed (so the results can be analyzed)
		if !CurrentSpecReport().Failed() {
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			DeleteIsolation(opsBaseURI, isolationID)
		}

		// Cleanup wiremock expectations created by test
		for _, mockID := range mockIDs {
			err := DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
		}
		mockIDs = nil // Reset the slice for next test
	})
	_ = Context("background service periodically sends usage metrics data to PDC", func() {
		When("PDC url is not specified", func() {
			It("it should not send data to PDC", func() {
				By("Creating WireMock expectations for embeddings")
				embeddingMockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, embeddingMockID)

				By("Creating PDC usage data endpoint expectation (should not be called)")
				pdcMockID, err := CreateExpectationUsageDataEndpoint(wiremockManager)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, pdcMockID)

				By("Creating collection")
				CreateCollection(svcBaseURI, isolationID, collectionID)
				ExpectCollectionExistsInDB(ctx, database, isolationID, collectionID)

				By("Adding a document to the collection")
				endpointURI := fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=eventual",
					svcBaseURI, isolationID, collectionID)
				docData := ReadTestDataFile("db/DOC-1.json")
				resp, _, err := HttpCall("PUT", endpointURI, nil, docData)

				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

				By("Waiting for document to be processed")
				WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

				By("Waiting DB_METRICS_PDC_UPLOAD_INTERVAL_SECONDS and verifying that PDC endpoint is NOT called")
				Consistently(func() (int, error) {
					return GetUsageDataEndpointCallCount(wiremockManager)
				}, 3*time.Second, 250*time.Millisecond).Should(
					Equal(0),
					"PDC endpoint should not be called when PDC URL is not configured",
				)
			})
		})

		When("PDC url is specified", func() {
			It("it should send data to PDC", func() {
				By("Creating WireMock expectations for embeddings")
				embeddingMockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, embeddingMockID)

				By("Creating PDC usage data endpoint expectation")
				pdcMockID, err := CreateExpectationUsageDataEndpoint(wiremockManager)
				Expect(err).To(BeNil())
				mockIDs = append(mockIDs, pdcMockID)

				By("Creating isolation with PDC endpoint URL")
				wiremockURL := wiremockManager.GetBaseURL()
				pdcEndpointURL := fmt.Sprintf("%s/prweb/PRSOAPServlet/test-uuid-%s/SOAP/PegaAES/Events", wiremockURL, RandStringRunes(8))

				DeleteIsolation(opsBaseURI, isolationID) // Clean up the one created in BeforeEach
				CreateIsolationWithPDCEndpoint(opsBaseURI, isolationID, "1GB", pdcEndpointURL)
				ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")

				By("Creating collection")
				CreateCollection(svcBaseURI, isolationID, collectionID)
				ExpectCollectionExistsInDB(ctx, database, isolationID, collectionID)

				By("Adding a document to the collection")
				endpointURI := fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=eventual",
					svcBaseURI, isolationID, collectionID)
				docData := ReadTestDataFile("db/DOC-1.json")
				resp, _, err := HttpCall("PUT", endpointURI, nil, docData)

				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

				By("Waiting for document to be processed")
				WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

				By("Waiting DB_METRICS_PDC_UPLOAD_INTERVAL_SECONDS and verifying that PDC endpoint IS called")
				Eventually(func() (int, error) {
					return GetUsageDataEndpointCallCount(wiremockManager)
				}, 10*time.Second, 250*time.Millisecond).Should(
					BeNumerically(">=", 1),
					"PDC endpoint should be called when PDC URL is configured",
				)

				By("Verifying usage data payload structure")
				usageDataRequests, err := GetUsageDataRequests(wiremockManager)
				Expect(err).To(BeNil())
				Expect(len(usageDataRequests)).To(BeNumerically(">=", 1), "At least one usage data request should be sent")

				// Parse and verify the payload structure
				var payload map[string]interface{}
				err = json.Unmarshal(usageDataRequests[0], &payload)
				Expect(err).To(BeNil())

				// Verify payload has required fields
				Expect(payload).To(HaveKey("data"))
				Expect(payload).To(HaveKey("metadata"))

				metadata, ok := payload["metadata"].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(metadata).To(HaveKey("source"))
				Expect(metadata["source"]).To(Equal("GenAIVectorStore"))

				data, ok := payload["data"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(len(data)).To(BeNumerically(">=", 1))

				// Verify the first metric has expected structure
				metric, ok := data[0].(map[string]interface{})
				Expect(ok).To(BeTrue())
				Expect(metric).To(HaveKey("metricType"))
				Expect(metric["metricType"]).To(Equal("DB"))
				Expect(metric).To(HaveKey("isolationID"))
				Expect(metric["isolationID"]).To(Equal(isolationID))
				Expect(metric).To(HaveKey("diskUsage"))
				Expect(metric["diskUsage"]).To(BeNumerically(">=", 1))
				Expect(metric).To(HaveKey("documentsCount"))
				Expect(metric["documentsCount"]).To(Equal(1.0))
				Expect(metric).To(HaveKey("documentsModification"))
				modTimeStr, ok := metric["documentsModification"].(string)
				Expect(ok).To(BeTrue())
				modTime, err := time.Parse(time.RFC3339, modTimeStr)
				Expect(err).To(BeNil())
				Expect(modTime).To(BeTemporally("<", time.Now()))
			})
		})
	})
})
