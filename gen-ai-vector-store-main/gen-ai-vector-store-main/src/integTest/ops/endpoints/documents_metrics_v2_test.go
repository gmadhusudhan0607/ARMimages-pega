// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
package endpoints_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing OPS /v1/ops/{isolationID}/documents ", func() {
	var endpointURI string
	ctx := context.TODO()
	var isolationID string
	var collectionID string

	_ = Context("accessing /v1/ops/{isolationID}/documents endpoint with POST method to retrieve documents metrics for isolation", func() {
		var mockIDs []string

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/ops/%s/documents", baseOpsURI, isolationID)

			CreateIsolation(baseOpsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			report := CurrentSpecReport()
			if !report.Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(baseOpsURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, mockID := range mockIDs {
				err := DeleteExpectationIfExist(wiremockManager, mockID)
				Expect(err).To(BeNil())
			}
			mockIDs = nil
		})

		It("test1: retrieve documents metrics for isolation ", func() {
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
			resp, body, err := HttpCall("POST", endpointURI, nil, ReadTestDataFile("documents_metrics/test1/request-all-params.json"))
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
			resp, body, err = HttpCall("POST", endpointURI, nil, ReadTestDataFile("documents_metrics/test1/request-some-params.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			err = json.Unmarshal(body, &isoMetrResp)
			Expect(err).To(BeNil())
			Expect(isoMetrResp.DocumentsCount).To(Equal(int64(3)))
			Expect(isoMetrResp.DocumentsModification).NotTo(BeNil())

			By("POST documents for retrieving all metrics for isolation when no parameters are included in request")
			resp, body, err = HttpCall("POST", endpointURI, nil, "")
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
			resp, body, err = HttpCall("POST", endpointURI, nil, "{}")
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

type DocumentMetricForIsolationResponse struct {
	DiskUsage             int64      `json:"diskUsage,omitempty"`
	DocumentsCount        int64      `json:"documentsCount,omitempty"`
	DocumentsModification *time.Time `json:"documentsModification,omitempty"`
}
