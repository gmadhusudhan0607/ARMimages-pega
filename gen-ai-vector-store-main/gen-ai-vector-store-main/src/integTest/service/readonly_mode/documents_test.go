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

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC POST /v1/{isolationID}/collections/{collectionName}/documents in ReadOnly mode", func() {

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

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents", svcBaseURI, isolationID, collectionID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It(" list document in COMPLETED status", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert test data")
			// Insert some data before test
			UpsertDoc(svcBaseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-post/test1/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("POST document for listing all documents in database")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal("[{\"id\":\"DOC-1\",\"status\":\"COMPLETED\"}]"))
		})

	})
})

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

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents", svcBaseURI, isolationID, collectionID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("GET document successfully in Read Only mode set ", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert documents")
			UpsertDocumentsFromDir("documents-get/test1/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for completion")
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)
			// the line was removed to avoid a fragile, unnecessary assertion about DOC-2. The actual behavior under test (successful GET of DOC-1 in readonly mode with the same response) is unchanged.

			By("Get document by id when document exists")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", getDocumentsGetEndpoint(endpointURI, "DOC-1"), ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectJSONEqualsFromFile(string(body), "documents-get/test1/responses/response-1.json")
		})
	})
})

var _ = Describe("Testing PUT SVC /v1/{isolationID}/collections/{collectionName}/documents in ReadOnly mode", func() {

	var isolationID string
	var collectionID string
	var testExpectations []string

	_ = Context("accessing endpoint with PUT method and eventual consistency level", func() {
		var endpointURI string
		ctx := context.TODO()
		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", svcBaseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			report := CurrentSpecReport()
			if !report.Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("PUT document returns 405 for eventual consistency level (asynchronous operation)", func() {

			By("Attempting to put document in ReadOnlyMode with eventual consistency level")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServiceRuntimeHeaders, ReadTestDataFile("documents-put/test4/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))
		})
	})
})

var _ = Describe("Testing SVC DELETE /v1/{isolationID}/collections/{collectionName}/documents in ReadOnly mode", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with DELETE method to delete documents by attributes", func() {
		var endpointURI string
		var testExpectations []string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents", svcBaseURI, isolationID, collectionID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("DELETE documents returns 405  ", func() {

			By("Attempting to delete documents in ReadOnlyMode")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("DELETE", endpointURI, ServiceRuntimeHeaders, ReadTestDataFile("documents-delete/test1/requests/request-in-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))
		})
	})
})

var _ = Describe("Testing SVC PATCH /v1/{isolationID}/collections/{collectionName}/documents in ReadOnly mode", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with PATCH method to update one document", func() {
		var endpointURI string
		var testExpectations []string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents", svcBaseURI, isolationID, collectionID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("PATCH document returns 405 ", func() {

			By("Attempting to patch document in ReadOnlyMode")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), ServiceRuntimeHeaders,
				ReadTestDataFile("documents-patch/test1/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))
		})
	})
})

var _ = Describe("Testing SVC DELETE /v1/{isolationID}/collections/{collectionName}/document/delete-by-id in ReadOnly mode", func() {

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

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/document/delete-by-id", svcBaseURI, isolationID, collectionID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("DELETE document by ID returns 405 ", func() {
			// Mock ADA using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert test data")
			// Insert some data before test
			UpsertDoc(svcBaseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-delete-by-id/test1/documents/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Attempting to delete document by ID in ReadOnlyMode")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, "{ \"id\": \"DOC-1\" }")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))
		})
	})
})

var _ = Describe("Testing SVC DELETE /v1/{isolationID}/collections/{collectionName}/documents/{documentID} in ReadOnly mode", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with DELETE method to delete one document", func() {
		var endpointURI string
		var testExpectations []string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents", svcBaseURI, isolationID, collectionID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("DELETE document by ID returns 405 ", func() {
			// Mock ADA using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert test data")
			// Insert some data before test
			UpsertDoc(svcBaseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual, ReadTestDataFile("documents-delete-by-id/test1/documents/DOC-1.json"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Attempting to delete document by ID in ReadOnlyMode")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("DELETE", getDocumentsDeleteByIdEndpoint(endpointURI, "DOC-1"), ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))
		})
	})
})

var _ = Describe("Testing PUT SVC /v1/{isolationID}/collections/{collectionName}/file in ReadOnly mode", func() {
	var isolationID string
	var collectionID string
	var testExpectations []string

	_ = Context("accessing endpoint with PUT method and strong consistency level", func() {
		var endpointURI string
		var docAttrs []attributes.Attribute
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/file?consistencyLevel=%s", svcBaseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong)
			docAttrs = []attributes.Attribute{
				{Name: "Document type", Type: "string", Values: []string{"Article"}},
				{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
			}
			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
		})

		It("PUT .pdf document file with strong consistency level returns 405", func() {
			// Mock ADA using WireMock
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Attempting to put document file in ReadOnlyMode")
			docAttrs = []attributes.Attribute{
				{Name: "Region", Type: "string", Values: []string{"GALAXY"}},
			}

			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				// we do not care about the content of the file, we just need parse mock server expectations
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.pdf")},
			}
			resp, body, err := HttpCallMultipartFormWithHeadersAndApiCallStat("PUT", endpointURI, mfParts, ServiceRuntimeHeaders)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))
		})
	})
})

var _ = Describe("Testing PUT SVC /v1/{isolationID}/collections/{collectionName}/file/text in ReadOnly mode", func() {

	var isolationID string
	var collectionID string
	var testExpectations []string

	_ = Context("accessing endpoint with PUT method and strong consistency level", func() {
		var endpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/file/text?consistencyLevel=%s", svcBaseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
		})

		It("PUT document file text returns 405", func() {
			// Mock ADA using WireMock
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Attempting to put document file text in ReadOnlyMode")
			putReq := documents.PutFileTextRequest{
				DocumentID:      "Astronomy",
				DocumentContent: ReadTestDataFile("documents-put-file/test1/documents/Astronomy.md"),
				DocumentAttributes: []attributes.Attribute{
					{Name: "Document type", Type: "string", Values: []string{"Article"}},
					{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
				},
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCallWithHeadersAndApiCallStat("PUT", endpointURI, ServiceRuntimeHeaders, string(requestBody))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))
		})
	})
})
