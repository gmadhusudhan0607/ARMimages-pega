//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

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

var _ = Describe("Testing SVC PUST /v1/{isolationID}/collections/{collectionName}/document/delete-by-id", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with POST method to delete one document", func() {
		var endpointURI string
		var testExpectations []string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/document/delete-by-id", baseURI, isolationID, collectionID)

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("test 404: return 404 if isolation does not exist", func() {
			// v.POST("/document/delete-by-id", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.DeleteDocumentById)
			ExpectServiceReturns404IfIsolationDoesNotExist("POST", endpointURI)
		})

		It("test 404: return 404 if collection does not exist", func() {
			// v.POST("/document/delete-by-id", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.DeleteDocumentById)
			ExpectServiceReturns404IfCollectionDoesNotExists("POST", endpointURI)
		})

		It("test1: successfully delete document by ID", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-delete-by-id/test1/documents/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("DELETE document by ID")
			resp, body, err := HttpCall("POST", endpointURI, nil, "{ \"id\": \"DOC-1\" }")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("{\"deletedDocuments\":1}"))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
		})

		It("test2: delete document that does not exist should not fail", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test, it's needed for relation to be created
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-delete-by-id/test2/documents/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("DELETE document that does not exist")
			resp, body, err := HttpCall("POST", endpointURI, nil, "{ \"id\": \"DOC-2\" }")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("{\"deletedDocuments\":0}"))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
		})

		It("test3: delete document without document ID should return 400", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test, it's needed for relation to be created
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-delete-by-id/test3/documents/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("DELETE document without document ID")
			resp, body, err := HttpCall("POST", endpointURI, nil, "{ \"id\": \"\" }")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test3: delete document without content should return 400", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test, it's needed for relation to be created
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-delete-by-id/test4/documents/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("DELETE document without document ID")
			resp, body, err := HttpCall("POST", endpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test5: successfully delete document by ID where ID is a url ", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			docID := "https://en.wikipedia.org/wiki/WorldCup"

			By("Insert test data")
			// Insert some data before test
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-delete-by-id/test5/documents/DOC-with-url-id.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, docID, "COMPLETED")

			By("DELETE document by ID")
			resp, body, err := HttpCall("POST", endpointURI, nil, fmt.Sprintf("{ \"id\": \"%s\" }", docID))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("{\"deletedDocuments\":1}"))
		})
	})
})
