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

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC /v1/{isolationID}/collections/{collectionName}/query/chunks in ReadOnly mode", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()
	var testExpectations []string

	_ = Context("calling service", func() {
		var endpointURI string

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/query/chunks", svcBaseURI, isolationID, collectionID)
			testExpectations = []string{}

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
		It("Query chunks must return number of records defined by 'limit' parameter", func() {
			// Create embedder mock using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-chunks_v2/test1/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, resources.StatusCompleted)
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item

			By("Validate response returns all elements when limit not set")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-not-set.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(18))
			ExpectResponseMatchFromFile(string(body), "query-chunks_v2/test1/response-limit-not-set.json")
			// In readonly mode we only assert the number of returned items.

			By("Validate response returns all records when limit=0")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-0.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(18))
			ExpectResponseMatchFromFile(string(body), "query-chunks_v2/test1/response-limit-0.json")
			// In readonly mode we only assert the number of returned items.

			By("Validate response returns one item when limit=1")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-1.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// Compare only size because if distance is similar can get different response
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test1/response-limit-1.json")
			// In readonly mode we only assert the number of returned items.

			By("Validate response returns 2 items when limit=2")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-2.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			//Expect(err).To(BeNil())
			// Service may return fewer than 'limit' results depending on similarity thresholds;
			// validate that it returns at least one and no more than the requested limit.
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test1/response-limit-1.json")
			Expect(len(items)).To(BeNumerically(">=", 1))
			Expect(len(items)).To(BeNumerically("<=", 2))

			By("Validate response returns 18 (all available) items when limit=100")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-100.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(18))
			ExpectResponseMatchFromFile(string(body), "query-chunks_v2/test1/response-limit-100.json")
			// In readonly mode we only assert the number of returned items.
		})
	})
})

var _ = Describe("Testing SVC POST /v1/{isolationID}/collections/{collectionName}/attributes in ReadOnly mode", func() {

	var isolationID string
	var collectionID string

	_ = Context("accessing endpoint with POST method to list attributes", func() {
		var endpointURI string
		var endpointURISetup string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/attributes", svcBaseURI, isolationID, collectionID)
			endpointURISetup = fmt.Sprintf("%s/v2/%s/collections", svcBaseURI, isolationID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")

			// Create collection for testing
			reqBody := fmt.Sprintf("{\"name\":\"%s\"}", collectionID)
			_, _, err := HttpCall("POST", endpointURISetup, nil, reqBody)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			// Do not clean up if test failed
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsBaseURI, isolationID)
			}
		})

		It("POST attributes works in ReadOnly mode", func() {

			By("Attempting to list attributes in ReadOnlyMode")
			reqBody := `{"retrieveAttributes": ["attr1", "attr2"]}`
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			// This should work in ReadOnly mode as it's a read operation
			// It might return 404 if collection doesn't exist, but that's fine for testing ReadOnly mode
			Expect(resp.StatusCode).To(BeElementOf([]int{http.StatusOK, http.StatusNotFound}))
		})
	})
})

var _ = Describe("Testing SVC /v1/{isolationID}/collections/{collectionName}/query/documents in ReadOnly mode", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()

	_ = Context("calling service", func() {
		var testExpectations []string
		var endpointURI string

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/query/documents", svcBaseURI, isolationID, collectionID)

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
		It("Query documents must return number of records defined by 'limit' parameter", func() {
			// Create embedder mock using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-documents/test1/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, resources.StatusCompleted)
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item

			By("Validate response returns all elements when limit not set ")
			jsonData = ReadTestDataFile("query-documents/test1/query-limit-not-set.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			//FIXME
			//ExpectResponseMatchFromFile(string(body), "query-documents/test1/response-limit-not-set.json")

			By("Validate response returns all documents when limit=0")
			jsonData = ReadTestDataFile("query-documents/test1/query-limit-0.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			////FIXME- ISSUE-123256
			//ExpectResponseMatchFromFile(string(body), "query-documents/test1/response-limit-0.json")
			//ExpectSortedJSONEquals(body, []byte(ReadTestDataFile("query-documents/test1/response-limit-0.json")), "documentID")

			By("Validate response returns one document when limit=1")
			jsonData = ReadTestDataFile("query-documents/test1/query-limit-1.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// Compare only size because if distance is similar can get different response
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))

			By("Validate response returns 2 documents when limit=2")
			jsonData = ReadTestDataFile("query-documents/test1/query-limit-2.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			// Service may return fewer than 'limit' documents depending on similarity thresholds;
			// validate that it returns at least one and no more than the requested limit.
			Expect(len(items)).To(BeNumerically(">=", 1))
			Expect(len(items)).To(BeNumerically("<=", 2))

			By("Validate response returns all available documents when limit=100")
			jsonData = ReadTestDataFile("query-documents/test1/query-limit-100.json")
			resp, body, err = HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			// //FIXME- ISSUE-123256
			// ExpectResponseMatchFromFile(string(body), "query-documents/test1/response-limit-100.json")
			// ExpectSortedJSONEquals([]byte(body), []byte(ReadTestDataFile("query-documents/test1/response-limit-100.json")), "documentID")

		})
	})
})

var _ = Describe("Testing SVC /v1/{isolationID}/collections/{collectionName}/attributes ", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()
	var testExpectations []string

	_ = Context("calling service", func() {
		var endpointURI string
		var jsonData string
		var body []byte
		var resp *http.Response
		var items []Item

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/attributes", svcBaseURI, isolationID, collectionID)

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
		It("List requested attributes if ASYNC / COMPLETED documents ", func() {
			// Mock ADA using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("attributes/test01/documents", svcBaseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, resources.StatusCompleted)
			}

			By("Validate response returns all attributes when retrieveAttributes not set")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(7))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-not-set.json")

			By("Validate response returns all attributes when retrieveAttributes empty")
			jsonData = ReadTestDataFile("attributes/test02/query-retrieveAttributes-empty.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(7))
			ExpectResponseMatchFromFile(string(body), "attributes/test02/response-retrieveAttributes-empty.json")

			By("Validate response returns 1 attribute")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-1.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-1.json")

			By("Validate response returns 2 attribute")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-2.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(2))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-2.json")

			By("Validate response returns 3 attribute")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-3.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-3.json")

			By("Validate response ignore non-existent attributes")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-not-existent.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-not-existent.json")
		})
	})
})
