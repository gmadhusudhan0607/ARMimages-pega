// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
package endpoints_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC POST /v2/{isolationID}/collections/{collectionID}/find-documents", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with POST method to find documents", func() {
		var endpointURI string
		var testExpectations []string
		ctx := context.TODO()

		BeforeEach(func() {
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/find-documents", baseURI, isolationID, collectionID)

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsURI, isolationID)
			}
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("test 404: return 404 if isolation does not exist", func() {
			ExpectServiceReturns404IfIsolationDoesNotExist("POST", endpointURI)
		})

		It("test 404: return 404 if collection does not exist", func() {
			ExpectServiceReturns404IfCollectionDoesNotExists("POST", endpointURI)
		})

		It("test1: find documents with COMPLETED status", func() {
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data with COMPLETED status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test1/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)

			By("Switch embedder to error for second document")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusInternalServerError)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert document with ERROR status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test1/DOC-2.json"))
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-2", "ERROR")

			By("POST request to find documents with COMPLETED status")
			resp, body, err := HttpCall("POST", endpointURI, nil, ReadTestDataFile("find-documents/test1/request.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFileIgnoringFields(string(body), "find-documents/test1/response.json", "ingestionTime", "updateTime")
		})

		It("test2: find documents with ERROR status", func() {
			// First create an error stub so DOC-1 fails
			mockID, err := CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusInternalServerError)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert document with ERROR status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test2/DOC-1.json"))

			By("Wait for ERROR status")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "ERROR")

			// Then switch to a success stub so DOC-2 completes
			By("Switch embedder to success for second document")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert document with COMPLETED status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test2/DOC-2.json"))
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-2", "COMPLETED")

			By("POST request to find documents with ERROR status")
			resp, body, err := HttpCall("POST", endpointURI, nil, ReadTestDataFile("find-documents/test2/request.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectFindDocumentsArrayItemsHaveFieldsFromFile(string(body), "find-documents/test2/response.json")
		})

		It("test3: find documents with IN_PROGRESS status", func() {
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert document with COMPLETED status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test3/DOC-2.json"))
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-2", "COMPLETED")

			By("Insert document with IN_PROGRESS status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test3/DOC-1.json"))

			By("Wait for IN_PROGRESS status")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "IN_PROGRESS")

			By("POST request to find documents with IN_PROGRESS status")
			resp, body, err := HttpCall("POST", endpointURI, nil, ReadTestDataFile("find-documents/test3/request.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectFindDocumentsArrayItemsHaveFieldsFromFile(string(body), "find-documents/test3/response.json")
		})

		It("test4: find documents with specific fields requested", func() {
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test4/DOC-1.json"))

			By("Wait for COMPLETED status")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("POST request to find documents with specific fields")
			resp, body, err := HttpCall("POST", endpointURI, nil, ReadTestDataFile("find-documents/test4/request.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFileIgnoringFields(string(body), "find-documents/test4/response.json", "ingestionTime")
		})

		It("test5: filter documents by attributes", func() {
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert document with finance department")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test5/DOC-1.json"))

			By("Insert document with hr department")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test5/DOC-2.json"))

			By("Wait for documents to complete processing")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-2", "COMPLETED")

			By("POST request to find documents with finance department attribute")
			resp, body, err := HttpCall("POST", endpointURI, nil, ReadTestDataFile("find-documents/test5/request.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Verify only the finance document is returned
			ExpectResponseMatchFromFileIgnoringFields(string(body), "find-documents/test5/response.json", "ingestionTime", "updateTime")
		})

		It("test6: should return correct headers", func() {
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data with COMPLETED status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test6/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)

			By("Insert document with ERROR status")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusInternalServerError)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("find-documents/test6/DOC-2.json"))
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-2", "ERROR")

			By("POST request to find documents with COMPLETED status")
			resp, body, err := HttpCall("POST", endpointURI, nil, ReadTestDataFile("find-documents/test6/request.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 1)
		})
	})
})
