//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collection metrics retrieval in readonly mode", func() {
	var opsEndpointURL string
	ctx := context.TODO()
	var isolationID string
	collectionID1 := strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
	collectionID2 := strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

	_ = Context("when requesting detailed collection metrics", func() {
		var mockIDs []string

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			opsEndpointURL = fmt.Sprintf("%s/v1/ops/%s/documentsDetails", baseOpsURL, isolationID)

			CreateIsolation(baseOpsURL, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			report := CurrentSpecReport()
			if !report.Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(baseOpsURL, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, mockID := range mockIDs {
				err := DeleteExpectationIfExist(wiremockManager, mockID)
				Expect(err).To(BeNil())
			}
			mockIDs = nil
		})

		It("should return comprehensive collection metrics", func() {
			By("Creating WireMock expectations for embeddings")
			embeddingMockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, embeddingMockID)

			By("Insert test data")
			// Insert some data before test
			docIDs := UpsertDocumentsFromDir("documents_metrics_details/test1/documents/coll1", baseSvcURI, isolationID, collectionID1)
			docIDs2 := UpsertDocumentsFromDir("documents_metrics_details/test1/documents/coll2", baseSvcURI, isolationID, collectionID2)

			By("Wait for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID1, docID, resources.StatusCompleted)
			}
			for _, docID2 := range docIDs2 {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID2, docID2, resources.StatusCompleted)
			}

			By("Validate POST documentsDetails retrieve all metrics for collection when all parameters are included in request")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", opsEndpointURL, ServiceRuntimeHeaders, ReadTestDataFile("documents_metrics_details/test1/request-all-params.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var collMetrResp []DocumentMetricForCollectionResponse
			err = json.Unmarshal(body, &collMetrResp)
			Expect(err).To(BeNil())
			Expect(len(collMetrResp)).To(Equal(2))

			coll1 := getCollectionMetricsByName(collMetrResp, collectionID1)
			Expect(coll1).NotTo(BeNil())
			Expect(coll1.ID).To(Equal(collectionID1))
			Expect(coll1.DocumentsMetrics.DocumentsCount).To(Equal(int64(3)))
			Expect(coll1.DocumentsMetrics.DiskUsage).To(BeNumerically(">", 0))
			Expect(coll1.DocumentsMetrics.DocumentsModification).NotTo(BeNil())

			coll2 := getCollectionMetricsByName(collMetrResp, collectionID2)
			Expect(coll2).NotTo(BeNil())
			Expect(coll2.ID).To(Equal(collectionID2))
			Expect(coll2.DocumentsMetrics.DocumentsCount).To(Equal(int64(1)))
			Expect(coll1.DocumentsMetrics.DiskUsage).To(BeNumerically(">", 0))
			Expect(coll2.DocumentsMetrics.DocumentsModification).NotTo(BeNil())

			By("Validate POST documentsDetails retrieve specified metrics for isolation when not all parameters are included in request")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", opsEndpointURL, ServiceRuntimeHeaders, ReadTestDataFile("documents_metrics_details/test1/request-some-params.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			err = json.Unmarshal(body, &collMetrResp)
			Expect(err).To(BeNil())
			Expect(len(collMetrResp)).To(Equal(2))

			coll1 = getCollectionMetricsByName(collMetrResp, collectionID1)
			Expect(coll1).NotTo(BeNil())
			Expect(coll1.ID).To(Equal(collectionID1))
			Expect(coll1.DocumentsMetrics.DocumentsCount).To(Equal(int64(3)))
			Expect(coll1.DocumentsMetrics.DocumentsModification).NotTo(BeNil())

			coll2 = getCollectionMetricsByName(collMetrResp, collectionID2)
			Expect(coll2).NotTo(BeNil())
			Expect(coll2.ID).To(Equal(collectionID2))
			Expect(coll2.DocumentsMetrics.DocumentsCount).To(Equal(int64(1)))
			Expect(coll2.DocumentsMetrics.DocumentsModification).NotTo(BeNil())

			By("POST documents for retrieving all metrics for collections when no parameters are included in request")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", opsEndpointURL, ServiceRuntimeHeaders, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			err = json.Unmarshal(body, &collMetrResp)
			Expect(err).To(BeNil())
			Expect(len(collMetrResp)).To(Equal(2))

			coll1 = getCollectionMetricsByName(collMetrResp, collectionID1)
			Expect(coll1).NotTo(BeNil())
			Expect(coll1.ID).To(Equal(collectionID1))
			Expect(coll1.DocumentsMetrics.DocumentsCount).To(Equal(int64(3)))
			Expect(coll1.DocumentsMetrics.DiskUsage).To(BeNumerically(">", 0))
			Expect(coll1.DocumentsMetrics.DocumentsModification).NotTo(BeNil())

			coll2 = getCollectionMetricsByName(collMetrResp, collectionID2)
			Expect(coll2).NotTo(BeNil())
			Expect(coll2.ID).To(Equal(collectionID2))
			Expect(coll2.DocumentsMetrics.DocumentsCount).To(Equal(int64(1)))
			Expect(coll1.DocumentsMetrics.DiskUsage).To(BeNumerically(">", 0))
			Expect(coll2.DocumentsMetrics.DocumentsModification).NotTo(BeNil())
		})
	})
})

var _ = Describe("Isolation metrics retrieval in readonly mode", func() {
	var endpointURI string
	ctx := context.TODO()
	var isolationID string
	var collectionID string

	_ = Context("when requesting isolation-level metrics", func() {
		var mockIDs []string

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/ops/%s/documents", baseOpsURL, isolationID)

			CreateIsolation(baseOpsURL, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			report := CurrentSpecReport()
			if !report.Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(baseOpsURL, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, mockID := range mockIDs {
				err := DeleteExpectationIfExist(wiremockManager, mockID)
				Expect(err).To(BeNil())
			}
			mockIDs = nil
		})

		It("should return aggregated isolation metrics", func() {
			By("Creating WireMock expectations for embeddings")
			embeddingMockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, embeddingMockID)

			By("Insert test data")
			// Insert some data before test
			docIDs := UpsertDocumentsFromDir("documents_metrics/test1/documents", baseSvcURI, isolationID, collectionID)

			By("Wait for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, resources.StatusCompleted)
			}
			var isoMetrResp DocumentMetricForIsolationResponse

			By("POST documents for retrieving all metrics for isolation when all parameters are included in request")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, ReadTestDataFile("documents_metrics/test1/request-all-params.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			err = json.Unmarshal(body, &isoMetrResp)
			Expect(err).To(BeNil())
			Expect(isoMetrResp.DocumentsCount).To(Equal(int64(3)))
			Expect(isoMetrResp.DiskUsage).To(BeNumerically(">", 0))
			Expect(isoMetrResp.DocumentsModification).NotTo(BeNil())

			By("POST documents for retrieving specified metrics for isolation when not all parameters are included in request")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, ReadTestDataFile("documents_metrics/test1/request-some-params.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			err = json.Unmarshal(body, &isoMetrResp)
			Expect(err).To(BeNil())
			Expect(isoMetrResp.DocumentsCount).To(Equal(int64(3)))
			Expect(isoMetrResp.DocumentsModification).NotTo(BeNil())

			By("POST documents for retrieving all metrics for isolation when no parameters are included in request")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			err = json.Unmarshal(body, &isoMetrResp)
			Expect(err).To(BeNil())
			Expect(isoMetrResp.DocumentsCount).To(Equal(int64(3)))
			Expect(isoMetrResp.DiskUsage).To(BeNumerically(">", 0))
			Expect(isoMetrResp.DocumentsModification).NotTo(BeNil())

			By("POST documents for retrieving all metrics for isolation when empty JSON included in request")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			err = json.Unmarshal(body, &isoMetrResp)
			Expect(err).To(BeNil())
			Expect(isoMetrResp.DocumentsCount).To(Equal(int64(3)))
			Expect(isoMetrResp.DiskUsage).To(BeNumerically(">", 0))
			Expect(isoMetrResp.DocumentsModification).NotTo(BeNil())
		})
	})
})
