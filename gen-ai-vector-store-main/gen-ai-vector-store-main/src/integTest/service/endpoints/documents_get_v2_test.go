//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package endpoints_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC GET /v1/{isolationID}/collections/{collectionName}/documents/{documentID}", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with GET method to retrieve one document", func() {
		var endpointURI string
		var testExpectations []string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents", baseURI, isolationID, collectionID)

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
			ExpectServiceReturns404IfIsolationDoesNotExist("GET", fmt.Sprintf("%s/%s", endpointURI, "DOC-1"))
		})

		It("test 404: return 404 if collection does not exist", func() {
			ExpectServiceReturns404IfCollectionDoesNotExists("GET", fmt.Sprintf("%s/%s", endpointURI, "DOC-1"))
		})

		It("test1: get document", func() {
			// First, use a success stub so DOC-1 completes
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert document with COMPLETED status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-get/test1/documents/DOC-1.json"))
			By("Waiting for DOC-1 completion")
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-1", "COMPLETED")

			// Then switch to an error stub so DOC-2 ends in ERROR
			By("Switch embedder to error for DOC-2")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusInternalServerError)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert document with ERROR status")
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-get/test1/documents/DOC-2.json"))
			By("Waiting for DOC-2 error")
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-2", "ERROR")

			By("Get document by id when document exists")
			resp, body, err := HttpCall("GET", getDocumentsGetEndpoint(endpointURI, "DOC-1"), nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectJSONEqualsFromFile(string(body), "documents-get/test1/responses/response-1.json")

			By("Get document by id when document is in ERROR status")
			resp, body, err = HttpCall("GET", getDocumentsGetEndpoint(endpointURI, "DOC-2"), nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectJSONEqualsFromOneOfFiles(string(body),
				"documents-get/test1/responses/response-2.json",
				"documents-get/test1/responses/response-2-alt.json",
				"documents-get/test1/responses/response-2-alt2.json")

			By("Get document by id when document does not exist should return 404")
			resp, body, err = HttpCall("GET", getDocumentsGetEndpoint(endpointURI, "DOC-3"), nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("test2: should return correct headers", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			UpsertDocumentsFromDir("documents-get/test2/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("Get document by id when document exists")
			resp, body, err := HttpCall("GET", getDocumentsGetEndpoint(endpointURI, "DOC-1"), nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectJSONEqualsFromFile(string(body), "documents-get/test2/responses/response-1.json")

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 1)
		})
	})
})

func getDocumentsGetEndpoint(endpointURI, docID string) string {
	u, err := url.Parse(endpointURI)
	if err != nil {
		panic(err)
	}
	u.Path = fmt.Sprintf("%s/%s", u.Path, docID)
	return u.String()
}
