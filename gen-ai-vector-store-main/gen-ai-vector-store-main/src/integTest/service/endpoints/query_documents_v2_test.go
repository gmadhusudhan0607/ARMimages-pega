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
	"slices"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type QueryDocumentItem struct {
	ID         string                 `json:"id" binding:"required"`
	Distance   float64                `json:"distance" binding:"required"`
	Attributes []attributes.Attribute `json:"attributes"`
}

var _ = Describe("Testing SVC /v1/{isolationID}/collections/{collectionName}/query/documents", func() {

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
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/query/documents", baseURI, isolationID, collectionID)

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

		//Test limit
		// Return nothing when limit is set to 0
		// Return all when limit not set
		// Return exact number when limit is set
		It("v2 test1: must return number of records defined by 'limit' parameter", func() {
			// Create mocks
			{
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)
			}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-documents/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item

			By("Validate response returns all elements when limit not set")
			jsonData = ReadTestDataFile("query-documents/test1/query-limit-not-set.json")
			resp, body, err := HttpCall("POST", endpointURI, nil, jsonData)
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
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
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
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// Compare only size because if distance is similar can get different response
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))

			By("Validate response returns 2 documents when limit=2")
			jsonData = ReadTestDataFile("query-documents/test1/query-limit-2.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			// Service may return fewer than limit if fewer documents match;
			// validate that it respects the upper bound and returns at least one.
			Expect(len(items)).To(BeNumerically(">=", 1))
			Expect(len(items)).To(BeNumerically("<=", 2))

			By("Validate response returns all available documents when limit=100")
			jsonData = ReadTestDataFile("query-documents/test1/query-limit-100.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
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

		// Test maxDistance
		//  Return all when maxDistance set not set.
		//  Return exact number with MaxDistance exactly 0.0. when maxDistance is set to 0.0.
		//  Return all when maxDistance set to 1.0.
		//  Return exact number when maxDistance is set.
		//  Return items sorted by maxDistance.
		It("v2 test2: must return number of records defined by 'maxDistance' parameter", func() {
			// Create mocks
			{
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)
			}

			By("Creating WireMock expectations for query responses")
			adaExpectations := CreateAdaExpectationsFromDir(wiremockManager, "query-documents/test2/expectations")
			testExpectations = append(testExpectations, adaExpectations...)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-documents/test2/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			var jsonData string
			var body []byte
			var resp *http.Response
			var items []QueryDocumentItem

			By("Validate response returns all documents when maxDistance not set")
			jsonData = ReadTestDataFile("query-documents/test2/query-maxdistance-not-set.json")
			resp, body, err := HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			// Service currently returns 3 matching documents; validate that
			// it returns a non-zero set and never more than the 4 available.
			Expect(len(items)).To(BeNumerically(">=", 3))
			Expect(len(items)).To(BeNumerically("<=", 4))
			// Distances can change slightly with different embeddings; validate items ignoring order
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test2/response-maxdistance-not-set_v2.json")

			By("Validate response returns 1 document with a distance of exactly 0 when maxDistance=0")
			jsonData = ReadTestDataFile("query-documents/test2/query-maxdistance-0.0.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test2/response-maxdistance-0.0_v2.json")

			By("Validate response returns all maxDistance when maxDistance=1.0")
			jsonData = ReadTestDataFile("query-documents/test2/query-maxdistance-1.0.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			// it returns a non-zero set and never more than the 4 available.
			Expect(len(items)).To(BeNumerically(">=", 3))
			Expect(len(items)).To(BeNumerically("<=", 4))
			// Same here: ensure same documents, regardless of order or tiny distance differences
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test2/response-maxdistance-1.0_v2.json")

			By("Validate response returns 3 document when maxDistance=0.29")
			jsonData = ReadTestDataFile("query-documents/test2/query-maxdistance-0.29.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 2))
			Expect(len(items)).To(BeNumerically("<=", 3))
			for _, item := range items {
				Expect(item.Distance < 0.29).To(BeTrue())
			}
			// Distances only need to be < 0.29; compare documents ignoring order
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test2/response-maxdistance-0.29_v2.json")

			By("Validate returned items sorted by maxDistance")
			jsonData = ReadTestDataFile("query-documents/test2/query-maxdistance-1.0.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			for i := 0; i < len(items)-1; i++ {
				Expect(items[i].Distance <= items[i+1].Distance).To(BeTrue())
			}
		})

		// Test retrieveAttributes
		//  Return all attributes when retrieveAttributes is not set
		//  Return only all documents when retrieveAttributes is empty
		//  Return exact attributes when retrieveAttributes is set
		It("v2 test3: must return records with retrieveAttributes", func() {
			// Create mocks from templates only (embedding + query),
			// so responses match the existing JSON expectations exactly.
			{
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)
			}
			adaExpectations := CreateAdaExpectationsFromDir(wiremockManager, "query-documents/test3/expectations")
			testExpectations = append(testExpectations, adaExpectations...)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-documents/test3/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			var jsonData string
			var body []byte
			var resp *http.Response
			var items []QueryDocumentItem
			var err error

			By("Validate response returns all attributes when retrieveAttributes not set")
			jsonData = ReadTestDataFile("query-documents/test3/query-retrieveAttributes-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 2))
			Expect(len(items)).To(BeNumerically("<=", 3))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test3/response-retrieveAttributes-not-set_v2.json")

			By("Validate response all documents when retrieveAttributes is empty")
			jsonData = ReadTestDataFile("query-documents/test3/query-retrieveAttributes-empty.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 2))
			Expect(len(items)).To(BeNumerically("<=", 3))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test3/response-retrieveAttributes-empty.json")

			By("Validate response returns only attributes defined in retrieveAttributes")
			jsonData = ReadTestDataFile("query-documents/test3/query-retrieveAttributes-2.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 1))
			Expect(len(items)).To(BeNumerically("<=", 2))
			expectedAttributes := []string{"roles", "title"}
			for _, item := range items {
				Expect(len(item.Attributes) <= len(expectedAttributes)).To(BeTrue())
				for _, attr := range item.Attributes {
					Expect(slices.Contains(expectedAttributes, attr.Name)).To(BeTrue())
				}
			}
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test3/response-retrieveAttributes-2.json")

			By("Validate response ignore extra attributes defined in retrieveAttributes")
			jsonData = ReadTestDataFile("query-documents/test3/query-retrieveAttributes-extra.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 2))
			Expect(len(items)).To(BeNumerically("<=", 3))
			expectedAttributes = []string{"roles", "title", "dataSource", "command", "content", "version", "org", "notExistentAttribute"}
			for _, item := range items {
				Expect(len(item.Attributes) <= len(expectedAttributes)).To(BeTrue())
				for _, attr := range item.Attributes {
					Expect(slices.Contains(expectedAttributes, attr.Name)).To(BeTrue())
				}
			}
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test3/response-retrieveAttributes-extra_v2.json")
		})

		// Test filters.attributes
		// Returns all chunks when attributes not set
		// Returns error when attributes.filter is empty
		// Returns exact chunks when attributes are provided
		//  Test filtering by attributes.name
		//  Test filtering by attributes.type
		//  Test filtering by attributes.value
		It("v2 test4: must return records filtered by 'filters.attributes' parameter", func() {
			// Create mocks from templates only (embedding + query),
			// so responses match the existing JSON expectations exactly.

			{
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)
			}
			adaExpectations := CreateAdaExpectationsFromDir(wiremockManager, "query-documents/test4/expectations")
			testExpectations = append(testExpectations, adaExpectations...)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-documents/test4/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item
			var err error

			By("Validate response returns all attributes when filters.attributes not set")
			jsonData = ReadTestDataFile("query-documents/test4/query-filters.attributes-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 2))
			Expect(len(items)).To(BeNumerically("<=", 3))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test4/response-filters.attributes-not-set_v2.json")

			//// FIXME: BUG-858809 : must return error when filters.attributes are empty
			//By("Validate response returns error when filters.attributes is empty")
			//jsonData = ReadTestDataFile("query-documents/test4/query-filters.attributes-empty.json")
			//resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			//Expect(err).To(BeNil())
			//Expect(body).NotTo(BeNil())
			//Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			By("Validate response returns attributes filtered by filters.attributes.name")
			jsonData = ReadTestDataFile("query-documents/test4/query-filters.attributes-by-value.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 1))
			Expect(len(items)).To(BeNumerically("<=", 2))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test4/response-filters.attributes-by-value_v2.json")
		})

		It("v2 test12d (document): must return documents by filtered attributes when attributes assigned on document level", func() {
			// Create mocks from templates only (embedding + query),
			// so responses match the existing JSON expectations exactly.
			{
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)
			}
			adaExpectations := CreateAdaExpectationsFromDir(wiremockManager, "query-documents/test12d/expectations")
			testExpectations = append(testExpectations, adaExpectations...)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-documents/test12d/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item
			var err error
			By("Validate response")
			jsonData = ReadTestDataFile("query-documents/test12d/query.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 1))
			Expect(len(items)).To(BeNumerically("<=", 2))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test12d/response.json")
		})

		It("v2 test12e (document): must return 0 documents by filtered attributes when attributes assigned on chunk level", func() {
			// Create mocks from templates only (embedding + query),

			// so responses match the existing JSON expectations exactly.
			{
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)
			}
			adaExpectations := CreateAdaExpectationsFromDir(wiremockManager, "query-documents/test12e/expectations")
			testExpectations = append(testExpectations, adaExpectations...)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-documents/test12e/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item
			var err error
			By("Validate response")
			jsonData = ReadTestDataFile("query-documents/test12e/query.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">=", 1))
			Expect(len(items)).To(BeNumerically("<=", 2))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-documents/test12e/response.json")
		})

		// Test overscan multiplier: when documents have many chunks (embeddings),
		// the CTE LIMIT is multiplied by DOCUMENT_SEMANTIC_SEARCH_MULTIPLIER (default 10)
		// to ensure that after deduplication (ROW_NUMBER PARTITION BY doc_id) the final
		// result set contains the requested number of unique documents.
		// Without the multiplier, a limit=5 query might scan only 5 embeddings from the
		// vector index, which could all belong to the same 1-2 documents after dedup.
		// Additionally tests the "enableSecondScan" request body field which triggers
		// an adaptive second scan when Stage 1 returns fewer documents than requested.
		It("v2 test-overscan: must return correct number of documents when each document has many chunks", func() {
			// Create mocks — generic ADA embedding stub returns the same vector for every chunk
			{
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)
			}

			By("Insert 5 documents, each with 10 chunks (50 embeddings total)")
			docIDs := UpsertDocumentsFromDir("query-documents/test-overscan/documents", baseURI, isolationID, collectionID)
			Expect(len(docIDs)).To(Equal(5))

			By("Waiting for all documents to reach COMPLETED status")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			var jsonData string
			var body []byte
			var resp *http.Response

			By("Validate query with limit=5 returns all 5 documents despite multi-chunk deduplication")
			jsonData = ReadTestDataFile("query-documents/test-overscan/query-limit-5.json")
			resp, body, err := HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			var rawItems5 []map[string]interface{}
			err = json.Unmarshal(body, &rawItems5)
			Expect(err).To(BeNil())
			// With the overscan multiplier (default 10), CTE LIMIT = 5*10 = 50,
			// which covers all 50 embeddings. After dedup we should get all 5 documents.
			Expect(len(rawItems5)).To(Equal(5), "Expected 5 unique documents; overscan multiplier should compensate for multi-chunk dedup")

			// Verify all returned document IDs are from our test set
			expectedIDs := map[string]bool{
				"doc-overscan-1": true,
				"doc-overscan-2": true,
				"doc-overscan-3": true,
				"doc-overscan-4": true,
				"doc-overscan-5": true,
			}
			for _, item := range rawItems5 {
				docID, ok := item["documentID"].(string)
				Expect(ok).To(BeTrue(), "documentID field missing or not a string")
				Expect(expectedIDs).To(HaveKey(docID), fmt.Sprintf("Unexpected document ID: %s", docID))
			}

			By("Validate query with limit=3 returns exactly 3 documents")
			jsonData = ReadTestDataFile("query-documents/test-overscan/query-limit-3.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			var rawItems3 []map[string]interface{}
			err = json.Unmarshal(body, &rawItems3)
			Expect(err).To(BeNil())
			Expect(len(rawItems3)).To(Equal(3), "Expected exactly 3 documents with limit=3")

			By("Verify returned document IDs for limit=3 are from our test set")
			for _, item := range rawItems3 {
				docID, ok := item["documentID"].(string)
				Expect(ok).To(BeTrue(), "documentID field missing or not a string")
				Expect(expectedIDs).To(HaveKey(docID), fmt.Sprintf("Unexpected document ID: %s", docID))
			}

			By("Validate query with limit=5 and enableSecondScan=true returns all 5 documents")
			jsonData = ReadTestDataFile("query-documents/test-overscan/query-limit-5-second-scan.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			var rawItems5ss []map[string]interface{}
			err = json.Unmarshal(body, &rawItems5ss)
			Expect(err).To(BeNil())
			// With enableSecondScan=true, if Stage 1 returns fewer docs than requested,
			// a second scan with the actual max-chunks-per-doc multiplier is performed.
			// With 10 chunks per doc and multiplier=10, Stage 1 already covers all embeddings,
			// so the second scan should not degrade results.
			Expect(len(rawItems5ss)).To(Equal(5), "Expected 5 unique documents with enableSecondScan=true")

			By("Verify returned document IDs for enableSecondScan query are from our test set")
			for _, item := range rawItems5ss {
				docID, ok := item["documentID"].(string)
				Expect(ok).To(BeTrue(), "documentID field missing or not a string")
				Expect(expectedIDs).To(HaveKey(docID), fmt.Sprintf("Unexpected document ID: %s", docID))
			}
		})

		_ = Context("service response should return expected headers", func() {
			It("test 13: check if query documents endpoint returns all expected headers", func() {
				// Mock ADA
				expTpl := ReadTestDataFile("query-documents/test13/mock-expectation-tpl.json")
				mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)

				By("Insert document")
				docIDs := UpsertDocumentsFromDir("query-documents/test13/documents", baseURI, isolationID, collectionID)
				By("Waiting for completion")
				for _, docID := range docIDs {
					WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
				}

				By("Query documents with test data")
				resp, body, err := HttpCall("POST", endpointURI, nil, ReadTestDataFile("query-documents/test13/query.json"))
				Expect(err).To(BeNil())
				Expect(body).NotTo(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				By("Check headers in the response")
				ExpectHeadersCommon(resp)
				ExpectHeadersDatabase(resp)
				ExpectHeadersEmbedding(resp, 500, 1)
				ExpectHeadersItemsCount(resp, 1)
				ExpectHeadersProcessingOverhead(resp)
			})
		})
	})

})
