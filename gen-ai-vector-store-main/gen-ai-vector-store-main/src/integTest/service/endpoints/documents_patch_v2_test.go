// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package endpoints_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC PATCH /v1/{isolationID}/collections/{collectionName}/documents", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with PATCH method to update one document", func() {
		var endpointURI, attrEndpointURI string
		var testExpectations []string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents", baseURI, isolationID, collectionID)
			attrEndpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/attributes", baseURI, isolationID, collectionID)

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
			ExpectServiceReturns404IfIsolationDoesNotExist("PATCH", fmt.Sprintf("%s/%s", endpointURI, "DOC-1"))
		})

		It("test 404: return 404 if collection does not exist", func() {
			ExpectServiceReturns404IfCollectionDoesNotExists("PATCH", fmt.Sprintf("%s/%s", endpointURI, "DOC-1"))
		})

		It("test1 (V2): successfully patch document with updating attributes", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-patch/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("PATCH document with one attribute")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
				ReadTestDataFile("documents-patch/test1/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Validate if attributes were updated correctly")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{\"retrieveAttributes\": [ \"dataSource\" ] }")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-patch/test1/responses/response-1.json", "name")

			By("PATCH document with more than one attribute")
			resp, body, err = HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
				ReadTestDataFile("documents-patch/test1/requests/request-2.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Validate if attributes were updated correctly")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{\"retrieveAttributes\": [ \"version\", \"org\" ] }")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-patch/test1/responses/response-2.json", "name")
		})

		// TODO: uncomment once BUG-858976 resolved
		//It("test2: do not return error when patching attribute that do not exist and relation do not exist", func() {
		//	By("PATCH document with one attribute")
		//	resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
		//		ReadTestDataFile("documents-patch/test2/requests/request-1.json"))
		//	Expect(err).To(BeNil())
		//	Expect(body).NotTo(BeNil())
		//	Expect(resp).NotTo(BeNil())
		//	Expect(resp.StatusCode).To(Equal(http.StatusOK))
		//	Expect(string(body)).To(Equal(""))
		//})

		It("test3 (V2): add attribute when it does not exist", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-patch/test3/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("PATCH document with one attribute")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
				ReadTestDataFile("documents-patch/test3/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Validate if attributes were updated correctly")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{\"retrieveAttributes\": [ \"attributeThatDoNotExist\" ] }")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-patch/test3/responses/response-1.json", "name")
		})

		It("test4 (V2): patch attribute does not affect other documents", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-patch/test4/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("PATCH document with one attribute")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-3"), nil,
				ReadTestDataFile("documents-patch/test4/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Validate if attributes were updated correctly")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{\"retrieveAttributes\": [ \"dataSource\" ] }")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-patch/test4/responses/response-1.json", "name")

			By("Validate if any other attributes were affected")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectSortedJSONEqualsFromFile(string(body), "documents-patch/test4/responses/response-2.json", "name")
		})

		It("test5: should return correct headers", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-patch/test5/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("PATCH document with one attribute")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-3"), nil,
				ReadTestDataFile("documents-patch/test5/requests/request-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
		})

		It("test6: successfully update document status to COMPLETED", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-patch/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Update document status to ERROR")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
				`{"status": "ERROR"}`)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Verify status was updated in database")
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-1", "ERROR")

			By("Update document status back to COMPLETED")
			resp, body, err = HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
				`{"status": "COMPLETED"}`)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Verify status was updated in database")
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-1", "COMPLETED")
		})

		It("test7: return 400 for invalid status value", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-patch/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Try to update document status with invalid value")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
				`{"status": "INVALID_STATUS"}`)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(string(body)).To(ContainSubstring("Invalid status value"))
			Expect(string(body)).To(ContainSubstring("INVALID_STATUS"))
		})

		It("test8: return 404 when trying to set status of non-existent document", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert one document to create collection")
			docIDs := UpsertDocumentsFromDir("documents-patch/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Try to update status of non-existent document - should return 404")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "NON-EXISTENT-DOC"), nil,
				`{"status": "ERROR"}`)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			Expect(string(body)).To(ContainSubstring("not found"))
		})

		It("test9: successfully update both status and attributes", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-patch/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Update both status and attributes")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
				`{"status": "ERROR", "attributes": [{"name": "version", "value": ["9.0"], "type": "string"}]}`)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Verify status was updated")
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-1", "ERROR")

			By("Verify attributes were updated")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{\"retrieveAttributes\": [ \"version\" ] }")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(ContainSubstring("9.0"))
		})

		It("test10: return 400 when both status and attributes are empty", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert one document to create collection")
			docIDs := UpsertDocumentsFromDir("documents-patch/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Try to patch with empty body")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil, `{}`)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(string(body)).To(ContainSubstring("At least one of"))
		})

		It("test12: successfully update status with errorMessage field", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-patch/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Update document status to ERROR with errorMessage")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "DOC-1"), nil,
				`{"status": "ERROR", "errorMessage": "chunking failed: unsupported file format"}`)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(body)).To(Equal(""))

			By("Verify status was updated in database")
			WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, "DOC-1", "ERROR")
		})

		It("test11: return 404 when trying to update attributes of non-existent document", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert one document to create collection")
			docIDs := UpsertDocumentsFromDir("documents-patch/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Try to update attributes of non-existent document")
			resp, body, err := HttpCall("PATCH", getDocumentsPatchEndpoint(endpointURI, "NON-EXISTENT-DOC"), nil,
				`{"attributes": [{"name": "version", "value": ["1.0"], "type": "string"}]}`)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			Expect(string(body)).To(ContainSubstring("document"))
			Expect(string(body)).To(ContainSubstring("not found"))
		})
	})
})

func getDocumentsPatchEndpoint(endpointURI, docID string) string {
	u, err := url.Parse(endpointURI)
	if err != nil {
		panic(err)
	}
	u.Path = fmt.Sprintf("%s/%s", u.Path, docID)
	return u.String()
}
