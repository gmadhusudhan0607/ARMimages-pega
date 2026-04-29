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

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC DELETE /v1/{isolationID}/collections/{collectionName}/documents", func() {

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
			// v.DELETE("/documents", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.DeleteDocuments)
			ExpectServiceReturns404IfIsolationDoesNotExist("DELETE", endpointURI)
		})

		It("test 404: return 404 if collection does not exist", func() {
			// v.DELETE("/documents", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.DeleteDocuments)
			ExpectServiceReturns404IfCollectionDoesNotExists("DELETE", endpointURI)
		})

		It("test1: successfully delete documents by one attribute for 'IN' operator", func() {
			// Mock ADA
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("documents-delete/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("DELETE document with 'IN' operator for one attribute")
			resp, body, err := HttpCall("DELETE", endpointURI, nil, ReadTestDataFile("documents-delete/test1/requests/request-in-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// DOC-1 and DOC-4 should be deleted
			Expect(string(body)).To(Equal("{\"deletedDocuments\":2}"))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
		})

		//// TODO: uncomment when BUG-859130 is resolved
		////It("test2: successfully delete documents by one attribute for 'OR' operator", func() {
		////	// Mock ADA
		////	testExpectations = CreateMockServerExpectationsFromDir("documents-delete/test2/expectations", isolationID, collectionID)
		////
		////	By("Insert test data")
		////	// Insert some data before test
		////  By("Insert documents")
		////  docIDs := UpsertDocumentsFromDir("documents-delete/test2/documents", baseURI, isolationID, collectionID)
		////  By("Waiting for completion")
		////  for _, docID := range docIDs {
		////	  WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
		////  }
		////
		////	By("DELETE document with 'OR' operator for one attribute")
		////	resp, body, err := HttpCall("DELETE", endpointURI, nil, ReadTestDataFile("documents-delete/test2/requests/request-or-1.json"))
		////	Expect(err).To(BeNil())
		////	Expect(body).NotTo(BeNil())
		////	Expect(resp).NotTo(BeNil())
		////	Expect(resp.StatusCode).To(Equal(http.StatusOK))
		////	// DOC-1 and DOC-4 should be deleted
		////	Expect(string(body)).To(Equal("{\"deletedDocuments\":2}"))
		////})
		//
		//It("test3: successfully delete documents by two attributes for 'IN' operator", func() {
		//	// Mock ADA
		//	testExpectations = CreateMockServerExpectationsFromDir("documents-delete/test3/expectations", isolationID, collectionID)
		//
		//	By("Insert documents")
		//	docIDs := UpsertDocumentsFromDir("documents-delete/test3/documents", baseURI, isolationID, collectionID)
		//	By("Waiting for completion")
		//	for _, docID := range docIDs {
		//		WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
		//	}
		//
		//	By("DELETE document with 'IN' operator for two attributes")
		//	resp, body, err := HttpCall("DELETE", endpointURI, nil, ReadTestDataFile("documents-delete/test3/requests/request-in-2.json"))
		//	Expect(err).To(BeNil())
		//	Expect(body).NotTo(BeNil())
		//	Expect(resp).NotTo(BeNil())
		//	Expect(resp.StatusCode).To(Equal(http.StatusOK))
		//	// DOC-1, DOC-2 and DOC-4 should be deleted
		//	Expect(string(body)).To(Equal("{\"deletedDocuments\":3}"))
		//})
		//
		//It("test4: successfully delete documents by one attribute for 'EQ' operator", func() {
		//	// Mock ADA
		//	testExpectations = CreateMockServerExpectationsFromDir("documents-delete/test4/expectations", isolationID, collectionID)
		//
		//	By("Insert documents")
		//	docIDs := UpsertDocumentsFromDir("documents-delete/test4/documents", baseURI, isolationID, collectionID)
		//	By("Waiting for completion")
		//	for _, docID := range docIDs {
		//		WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
		//	}
		//
		//	By("DELETE document with 'EQ' operator for one attribute")
		//	resp, body, err := HttpCall("DELETE", endpointURI, nil, ReadTestDataFile("documents-delete/test4/requests/request-eq-1.json"))
		//	Expect(err).To(BeNil())
		//	Expect(body).NotTo(BeNil())
		//	Expect(resp).NotTo(BeNil())
		//	Expect(resp.StatusCode).To(Equal(http.StatusOK))
		//	Expect(string(body)).To(Equal("{\"deletedDocuments\":1}"))
		//})
		//
		//It("test5: successfully delete documents by two attributes for 'EQ' operator", func() {
		//	// Mock ADA
		//	testExpectations = CreateMockServerExpectationsFromDir("documents-delete/test5/expectations", isolationID, collectionID)
		//
		//	By("Insert documents")
		//	docIDs := UpsertDocumentsFromDir("documents-delete/test5/documents", baseURI, isolationID, collectionID)
		//	By("Waiting for completion")
		//	for _, docID := range docIDs {
		//		WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
		//	}
		//
		//	By("DELETE document with 'EQ' operator for two attributes")
		//	resp, body, err := HttpCall("DELETE", endpointURI, nil, ReadTestDataFile("documents-delete/test5/requests/request-eq-2.json"))
		//	Expect(err).To(BeNil())
		//	Expect(body).NotTo(BeNil())
		//	Expect(resp).NotTo(BeNil())
		//	Expect(resp.StatusCode).To(Equal(http.StatusOK))
		//	Expect(string(body)).To(Equal("{\"deletedDocuments\":1}"))
		//})
		//
		//It("test6: should not delete any documents when attribute is available only on chunk level", func() {
		//	// Mock ADA
		//	testExpectations = CreateMockServerExpectationsFromDir("documents-delete/test6/expectations", isolationID, collectionID)
		//
		//	By("Insert documents")
		//	docIDs := UpsertDocumentsFromDir("documents-delete/test6/documents", baseURI, isolationID, collectionID)
		//	By("Waiting for completion")
		//	for _, docID := range docIDs {
		//		WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
		//	}
		//
		//	By("DELETE document with 'IN' operator for one attribute")
		//	resp, body, err := HttpCall("DELETE", endpointURI, nil, ReadTestDataFile("documents-delete/test6/requests/request-in-1.json"))
		//	Expect(err).To(BeNil())
		//	Expect(body).NotTo(BeNil())
		//	Expect(resp).NotTo(BeNil())
		//	Expect(resp.StatusCode).To(Equal(http.StatusOK))
		//	Expect(string(body)).To(Equal("{\"deletedDocuments\":0}"))
		//})
		//
		//// TODO: uncomment once BUG-858976 is resolved
		////It("test7: should not return an error when no documents are stored in database", func() {
		////	By("DELETE document with 'IN' operator for one attribute")
		////	resp, body, err := HttpCall("DELETE", endpointURI, nil, ReadTestDataFile("documents-delete/test7/request-in-1.json"))
		////	Expect(err).To(BeNil())
		////	Expect(body).NotTo(BeNil())
		////	Expect(resp).NotTo(BeNil())
		////	Expect(resp.StatusCode).To(Equal(http.StatusOK))
		////	Expect(string(body)).To(Equal("{\"deletedDocuments\":0}"))
		////})
		//
		//It("test8: should not delete any documents when no body in request passed", func() {
		//	// Mock ADA
		//	testExpectations = CreateMockServerExpectationsFromDir("documents-delete/test8/expectations", isolationID, collectionID)
		//
		//	By("Insert documents")
		//	docIDs := UpsertDocumentsFromDir("documents-delete/test8/documents", baseURI, isolationID, collectionID)
		//	By("Waiting for completion")
		//	for _, docID := range docIDs {
		//		WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
		//	}
		//
		//	By("DELETE document with 'IN' operator for one attribute")
		//	resp, body, err := HttpCall("DELETE", endpointURI, nil, "[]")
		//	Expect(err).To(BeNil())
		//	Expect(body).NotTo(BeNil())
		//	Expect(resp).NotTo(BeNil())
		//	Expect(resp.StatusCode).To(Equal(http.StatusOK))
		//	Expect(string(body)).To(Equal("{\"deletedDocuments\":0}"))
		//})
	})
})
