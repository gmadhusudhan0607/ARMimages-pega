//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package endpoints_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/service/apiV2"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC /v2/{isolationID}/collections/{collectionID}/documents/{documentID}/chunks", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()
	var testExpectations []string

	_ = Context("calling service", func() {
		var endpointURI string

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

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

		It("test1 a: must return document  without pagination", func() {
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			UpsertDocumentsFromDir("document-get-chunks/test1/documents", baseURI, isolationID, collectionID)

			By("Waiting for completion")
			docID := "DOC #1"
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")

			var docChunksResp apiV2.GetDocumentChunksResponse

			By("Get document chunks without pagination")
			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/documents/%s/chunks", baseURI, isolationID, collectionID, url.PathEscape(docID))
			resp, body, err := HttpCall("GET", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &docChunksResp)
			Expect(err).To(BeNil())
			Expect(len(docChunksResp.Chunks)).To(Equal(30))
			ExpectResponseMatchFromFile(string(body), "document-get-chunks/test1/response-wo-pagination.json")
		})

		It("test1 b: must return document chunks paginated", func() {
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			UpsertDocumentsFromDir("document-get-chunks/test1/documents", baseURI, isolationID, collectionID)

			By("Waiting for completion")
			docID := "DOC #1"
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")

			var docChunksResp apiV2.GetDocumentChunksResponse

			By("Get document chunks with pagination with limit 8. Page 1")
			docChunksResp = apiV2.GetDocumentChunksResponse{} // Reset before each call
			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/documents/%s/chunks?cursor=%s&limit=%d",
				baseURI, isolationID, collectionID, url.PathEscape(docID), "", 8)
			resp, body, err := HttpCall("GET", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &docChunksResp)
			Expect(err).To(BeNil())
			Expect(len(docChunksResp.Chunks)).To(Equal(8))
			Expect(docChunksResp.Pagination.Limit).To(Equal(8))
			Expect(docChunksResp.Pagination.NextCursor).To(Equal("DOC%20%231-EMB-7"))
			ExpectResponseMatchFromFile(string(body), "document-get-chunks/test1/response-w-pagination-0-7.json")

			By("Get document chunks with pagination with limit 8. Page 1")
			docChunksResp = apiV2.GetDocumentChunksResponse{} // Reset before each call
			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/documents/%s/chunks?cursor=%s&limit=%d",
				baseURI, isolationID, collectionID, url.PathEscape(docID), "DOC%20%231-EMB-7", 8)
			resp, body, err = HttpCall("GET", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &docChunksResp)
			Expect(err).To(BeNil())
			Expect(len(docChunksResp.Chunks)).To(Equal(8))
			Expect(docChunksResp.Pagination.Limit).To(Equal(8))
			Expect(docChunksResp.Pagination.NextCursor).To(Equal("DOC%20%231-EMB-15"))
			ExpectResponseMatchFromFile(string(body), "document-get-chunks/test1/response-w-pagination-8-15.json")

			By("Get document chunks with pagination with limit 8. Page 3")
			docChunksResp = apiV2.GetDocumentChunksResponse{} // Reset before each call
			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/documents/%s/chunks?cursor=%s&limit=%d",
				baseURI, isolationID, collectionID, url.PathEscape(docID), "DOC%20%231-EMB-15", 8)
			resp, body, err = HttpCall("GET", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &docChunksResp)
			Expect(err).To(BeNil())
			Expect(len(docChunksResp.Chunks)).To(Equal(8))
			Expect(docChunksResp.Pagination.Limit).To(Equal(8))
			Expect(docChunksResp.Pagination.NextCursor).To(Equal("DOC%20%231-EMB-23"))
			ExpectResponseMatchFromFile(string(body), "document-get-chunks/test1/response-w-pagination-16-23.json")

			By("Get document chunks with pagination with limit 8. Page 4")
			docChunksResp = apiV2.GetDocumentChunksResponse{} // Reset before each call
			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/documents/%s/chunks?cursor=%s&limit=%d",
				baseURI, isolationID, collectionID, url.PathEscape(docID), "DOC%20%231-EMB-23", 8)
			resp, body, err = HttpCall("GET", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &docChunksResp)
			Expect(err).To(BeNil())
			Expect(len(docChunksResp.Chunks)).To(Equal(6))
			Expect(docChunksResp.Pagination.Limit).To(Equal(8))
			Expect(docChunksResp.Pagination.NextCursor).To(Equal(""))
			ExpectResponseMatchFromFile(string(body), "document-get-chunks/test1/response-w-pagination-24-29.json")

		})

		It("test2: should return correct headers", func() {
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			UpsertDocumentsFromDir("document-get-chunks/test2/documents", baseURI, isolationID, collectionID)

			By("Waiting for completion")
			docID := "DOC #1"
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")

			By("Get document chunks")
			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/documents/%s/chunks",
				baseURI, isolationID, collectionID, url.PathEscape(docID))
			resp, body, err := HttpCall("GET", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 30)
		})
	})
})
