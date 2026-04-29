// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package reembedding_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
)

var _ = Describe("Testing document PUT with embedding retry on background", func() {

	var (
		ctx          context.Context
		isolationID  string
		collectionID string
		mockIDs      []string // Track mock IDs for cleanup
	)

	_ = Context("eventual consistency with retries", func() {

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

		It("reembedding test1: Put document for eventual consistency level when ada returns error, retry on background", func() {
			endpointURI := fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s",
				svcBaseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual)

			By("Creating WireMock expectations for retry scenario")
			// The test simulates the following sequence using WireMock scenarios:
			// 1. First 2 calls return 503 (Service Unavailable)
			// 2. Next 2 calls return 429 (Too Many Requests)
			// 3. All subsequent calls return 200 (Success)
			errorSequence := []int{503, 503, 429, 429}
			scenarioMockIDs, err := CreateExpectationEmbeddingAdaWithRetryScenario(wiremockManager, isolationID, errorSequence)
			Expect(err).To(BeNil())
			mockIDs = append(mockIDs, scenarioMockIDs...)

			By("Put document for eventual consistency level")
			docData := ReadTestDataFile("test01/DOC-1.json")
			resp, body, err := HttpCall("PUT", endpointURI, nil, docData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for initial processing to start")
			// Give the service a moment to start processing
			time.Sleep(time.Second)

			By("Ensure document is in embedding queue and has status ERROR after initial failures")
			// When embeddings fail initially, the document status is set to ERROR
			// while the embeddings are queued for retry
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusError)
			ExpectDocumentInQueue(ctx, database, isolationID, collectionID, "DOC-1")
			docStatus, errMsg := GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
			Expect(docStatus).To(Equal(resources.StatusError))
			// The error message should indicate embedding failures
			Expect(errMsg).To(ContainSubstring("ERROR"))

			By("Ensure document is completed after background retries")
			// Wait until background process completes the embedding after retries
			// The background service will retry on 503 and 429 errors until it gets 200
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Verify document is no longer in queue")
			ExpectDocumentNotInQueue(ctx, database, isolationID, collectionID, "DOC-1")
			docStatus, _ = GetDocumentStatusAndErrorFromDB(ctx, database, isolationID, collectionID, "DOC-1")
			Expect(docStatus).To(Equal(resources.StatusCompleted))

			By("Verify embeddings were successfully created")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
		})
	})
})
