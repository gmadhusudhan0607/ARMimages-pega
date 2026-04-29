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

var _ = Describe("Testing SVC POST /v1/{isolationID}/collections/{collectionName}/documents", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with POST method to retrieve all documents", func() {
		var endpointURI string
		var testExpectations []string
		ctx := context.TODO()

		BeforeEach(func() {
			RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
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
			ExpectServiceReturns404IfIsolationDoesNotExist("POST", endpointURI)
		})

		It("test 404: return 404 if collection does not exist", func() {
			ExpectServiceReturns404IfCollectionDoesNotExists("POST", endpointURI)
		})

		It("test1: list document in COMPLETED status", func() {
			// Mock ADA - successful embedding for COMPLETED status
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-post/test1/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("POST document for listing all documents in database")
			resp, body, err := HttpCall("POST", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("[{\"id\":\"DOC-1\",\"status\":\"COMPLETED\"}]"))
		})

		It("test2: list document in ERROR status", func() {
			// Mock ADA - error response to drive ERROR status
			mockID, err := CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusUnauthorized)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-post/test2/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "ERROR")

			By("POST document for listing all documents in database")
			resp, body, err := HttpCall("POST", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(ContainSubstring("embedding returned status code 401 without an error"))
		})

		It("test3: list document in IN_PROGRESS status", func() {
			// Mock ADA - delayed success to keep document IN_PROGRESS during listing
			mockID, err := CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-post/test3/DOC-1.json"))

			By("Wait for IN_PROGRESS status")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "IN_PROGRESS")

			By("POST document for listing all documents in database")
			resp, body, err := HttpCall("POST", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("[{\"id\":\"DOC-1\",\"status\":\"IN_PROGRESS\"}]"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")
		})

		It("test4: filter documents by status", func() {
			// Mock ADA
			// 1) COMPLETED documents: use success stub
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			// Status COMPLETED
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-post/test4/documents/Albert Einstein.json"))
			By("Wait for COMPLETED status for Albert Einstein")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein", "COMPLETED")

			// 2) ERROR documents: swap to error stub
			By("Switch embedder to error for Vanity URL")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusUnauthorized)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-post/test4/documents/Vanity URL.json"))
			By("Wait for ERROR status for Vanity URL")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL", "ERROR")

			// 3) IN_PROGRESS documents: swap to delayed stub so they stay IN_PROGRESS during listing
			By("Switch embedder to delayed success for Lab API")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-post/test4/documents/Lab API.json"))

			By("Wait for IN_PROGRESS status for Lab API")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API", "IN_PROGRESS")

			By("POST document for listing all documents in database")
			resp, body, err := HttpCall("POST", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromOneOfFiles(string(body), "id",
				"documents-post/test4/responses/response-1.json",
				"documents-post/test4/responses/response-1-alt.json",
				"documents-post/test4/responses/response-1-alt2.json")

			By("POST document for listing documents in status IN_PROGRESS")
			resp, body, err = HttpCall("POST", getDocumentEndpointWithStatus(endpointURI, "IN_PROGRESS"), nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-post/test4/responses/response-2.json", "id")

			By("POST document for listing documents in status COMPLETED")
			resp, body, err = HttpCall("POST", getDocumentEndpointWithStatus(endpointURI, "COMPLETED"), nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-post/test4/responses/response-3.json", "id")

			By("POST document for listing documents in status ERROR")
			resp, body, err = HttpCall("POST", getDocumentEndpointWithStatus(endpointURI, "ERROR"), nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromOneOfFiles(string(body), "id",
				"documents-post/test4/responses/response-4.json",
				"documents-post/test4/responses/response-4-alt.json",
				"documents-post/test4/responses/response-4-alt2.json")
		})

		It("test5: filter documents by status with attributes filter on document level", func() {
			// Mock ADA
			// 1) COMPLETED documents: use success stub
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			// Status COMPLETED
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test5/documents/Albert Einstein.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test5/documents/Albert Einstein 2.json"))
			By("Wait for COMPLETED status for Albert Einstein documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein", "COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein 2", "COMPLETED")

			// 2) ERROR documents: swap to error stub
			By("Switch embedder to error for Vanity URL documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusUnauthorized)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test5/documents/Vanity URL.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test5/documents/Vanity URL 2.json"))
			By("Wait for ERROR status for Vanity URL documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL", "ERROR")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL 2", "ERROR")

			// 3) IN_PROGRESS documents: swap to delayed stub
			By("Switch embedder to delayed success for Lab API documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test5/documents/Lab API.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test5/documents/Lab API 2.json"))

			By("Wait for IN_PROGRESS status for Lab API documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API", "IN_PROGRESS")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API 2", "IN_PROGRESS")

			By("POST document for listing documents in status IN_PROGRESS with dataSource filter applied")
			resp, body, err := HttpCall("POST", getDocumentEndpointWithStatus(endpointURI, "IN_PROGRESS"), nil,
				ReadTestDataFile("documents-post/test5/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-post/test5/responses/response-1.json", "id")

			By("POST document for listing documents in status COMPLETED")
			resp, body, err = HttpCall("POST", getDocumentEndpointWithStatus(endpointURI, "COMPLETED"), nil,
				ReadTestDataFile("documents-post/test5/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-post/test5/responses/response-2.json", "id")

			By("POST document for listing documents in status ERROR")
			resp, body, err = HttpCall("POST", getDocumentEndpointWithStatus(endpointURI, "ERROR"), nil,
				ReadTestDataFile("documents-post/test5/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromOneOfFiles(string(body), "id",
				"documents-post/test5/responses/response-3.json",
				"documents-post/test5/responses/response-3-alt.json",
				"documents-post/test5/responses/response-3-alt2.json")

		})

		It("test6: filter documents by attributes filter on document level with or operator", func() {
			// Mock ADA
			// 1) COMPLETED documents: use success stub
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			// Status COMPLETED
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test6/documents/Albert Einstein.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test6/documents/Albert Einstein 2.json"))
			By("Wait for COMPLETED status for Albert Einstein documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein", "COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein 2", "COMPLETED")

			// 2) ERROR documents: swap to error stub
			By("Switch embedder to error for Vanity URL documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusUnauthorized)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test6/documents/Vanity URL.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test6/documents/Vanity URL 2.json"))
			By("Wait for ERROR status for Vanity URL documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL", "ERROR")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL 2", "ERROR")

			// 3) IN_PROGRESS documents: swap to delayed stub
			By("Switch embedder to delayed success for Lab API documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test6/documents/Lab API.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test6/documents/Lab API 2.json"))

			By("Wait for IN_PROGRESS status for Lab API documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API", "IN_PROGRESS")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API 2", "IN_PROGRESS")

			By("POST document for listing documents in with dataSource filter applied")
			resp, body, err := HttpCall("POST", endpointURI, nil,
				ReadTestDataFile("documents-post/test6/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromOneOfFiles(string(body), "id",
				"documents-post/test6/responses/response-1.json",
				"documents-post/test6/responses/response-1-alt.json",
				"documents-post/test6/responses/response-1-alt2.json")
		})

		It("test7: filter documents by attributes with and operator", func() {
			// Mock ADA
			// 1) COMPLETED documents: use success stub
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			// Status COMPLETED
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test7/documents/Albert Einstein.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test7/documents/Albert Einstein 2.json"))
			By("Wait for COMPLETED status for Albert Einstein documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein", "COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein 2", "COMPLETED")

			// 2) ERROR documents: swap to error stub
			By("Switch embedder to error for Vanity URL documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusUnauthorized)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test7/documents/Vanity URL.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test7/documents/Vanity URL 2.json"))
			By("Wait for ERROR status for Vanity URL documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL", "ERROR")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL 2", "ERROR")

			// 3) IN_PROGRESS documents: swap to delayed stub
			By("Switch embedder to delayed success for Lab API documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test7/documents/Lab API.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test7/documents/Lab API 2.json"))

			By("Wait for IN_PROGRESS status for Lab API documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API", "IN_PROGRESS")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API 2", "IN_PROGRESS")

			By("POST document for listing documents in with dataSource filter applied")
			resp, body, err := HttpCall("POST", endpointURI, nil,
				ReadTestDataFile("documents-post/test7/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "documents-post/test7/responses/response-1.json")
		})

		It("test8: should return correct headers", func() {
			// Mock ADA - simple success for COMPLETED documents
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			// Status COMPLETED
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test7/documents/Albert Einstein.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test7/documents/Albert Einstein 2.json"))

			By("Wait for status")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein", "COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein 2", "COMPLETED")

			By("POST document for listing documents in with dataSource filter applied")
			resp, body, err := HttpCall("POST", endpointURI, nil,
				ReadTestDataFile("documents-post/test8/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-post/test8/responses/response-1.json", "id")

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 2)
		})

		It("test9: filter documents by attributes filter on chunk level with or operator", func() {
			// Mock ADA
			// 1) COMPLETED documents: use success stub
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			// Status COMPLETED
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test9/documents/Albert Einstein.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test9/documents/Albert Einstein 2.json"))
			By("Wait for COMPLETED status for Albert Einstein documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein", "COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein 2", "COMPLETED")

			// 2) ERROR documents: swap to error stub
			By("Switch embedder to error for Vanity URL documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusUnauthorized)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test9/documents/Vanity URL.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test9/documents/Vanity URL 2.json"))
			By("Wait for ERROR status for Vanity URL documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL", "ERROR")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL 2", "ERROR")

			// 3) IN_PROGRESS documents: swap to delayed stub
			By("Switch embedder to delayed success for Lab API documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test9/documents/Lab API.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test9/documents/Lab API 2.json"))

			By("Wait for IN_PROGRESS status for Lab API documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API", "IN_PROGRESS")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API 2", "IN_PROGRESS")

			By("POST document for listing documents in with dataSource filter applied")
			resp, body, err := HttpCall("POST", endpointURI, nil,
				ReadTestDataFile("documents-post/test9/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "documents-post/test9/responses/response-1.json")
		})

		It("test10: filter documents by attributes on chunk level with and operator", func() {
			// Mock ADA
			// 1) COMPLETED documents: use success stub
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert test data")
			// Insert some data before test
			// Status COMPLETED
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test10/documents/Albert Einstein.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test10/documents/Albert Einstein 2.json"))
			By("Wait for COMPLETED status for Albert Einstein documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein", "COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Albert Einstein 2", "COMPLETED")

			// 2) ERROR documents: swap to error stub
			By("Switch embedder to error for Vanity URL documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithError(wiremockManager, isolationID, http.StatusUnauthorized)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test10/documents/Vanity URL.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test10/documents/Vanity URL 2.json"))
			By("Wait for ERROR status for Vanity URL documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL", "ERROR")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Vanity URL 2", "ERROR")

			// 3) IN_PROGRESS documents: swap to delayed stub
			By("Switch embedder to delayed success for Lab API documents")
			err = DeleteExpectationIfExist(wiremockManager, mockID)
			Expect(err).To(BeNil())
			mockID, err = CreateExpectationEmbeddingAdaWithDelay(wiremockManager, isolationID, 3000)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test10/documents/Lab API.json"))
			UpsertDoc(baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual,
				ReadTestDataFile("documents-post/test10/documents/Lab API 2.json"))

			By("Wait for IN_PROGRESS status for Lab API documents")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API", "IN_PROGRESS")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "Lab API 2", "IN_PROGRESS")

			By("POST document for listing documents in with dataSource filter applied")
			resp, body, err := HttpCall("POST", endpointURI, nil,
				ReadTestDataFile("documents-post/test10/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromOneOfFiles(string(body), "id",
				"documents-post/test10/responses/response-1.json",
				"documents-post/test10/responses/response-1-alt.json",
				"documents-post/test10/responses/response-1-alt2.json")
		})
	})
})

func getDocumentEndpointWithStatus(endpointURI, status string) string {
	if strings.Contains(endpointURI, "?") {
		return fmt.Sprintf("%s&status=%s", endpointURI, status)
	}
	return fmt.Sprintf("%s?status=%s", endpointURI, status)
}
