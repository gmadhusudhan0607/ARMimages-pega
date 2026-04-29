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

type QueryChunkItem struct {
	ID         string                 `json:"id" binding:"required"`
	Content    string                 `json:"content" binding:"required"`
	DocumentID string                 `json:"documentID" binding:"required"`
	Distance   float64                `json:"distance" binding:"required"`
	Attributes []attributes.Attribute `json:"attributes"`
}

var _ = Describe("Testing SVC /v1/{isolationID}/collections/{collectionName}/query/chunks", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()
	var testExpectations []string

	_ = Context("calling service", func() {
		var endpointURI string

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/query/chunks", baseURI, isolationID, collectionID)
			testExpectations = []string{}

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
			testExpectations = []string{}
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsURI, isolationID)
			}
		})

		It("test 404: return 404 if isolation does not exist", func() {
			ExpectServiceReturns404IfIsolationDoesNotExist("POST", endpointURI)
		})

		It("test 404: return 404 if collection does not exist", func() {
			ExpectServiceReturns404IfCollectionDoesNotExists("POST", endpointURI)
		})

		//Test limit
		// Return all when limit is set to 0
		// Return all when limit not set
		// Return exact number when limit is set
		It("v2 test1: must return number of records defined by 'limit' parameter", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-chunks_v2/test1/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item

			By("Validate response returns all elements when limit not set")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(18))
			// Order of equally distant chunks can change; compare ignoring order
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test1/response-limit-not-set.json")

			By("Validate response returns all records when limit=0")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-0.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(18))
			// Same here: verify same 18 items, regardless of order
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test1/response-limit-0.json")

			By("Validate response returns one item when limit=1")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-1.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// Compare only size because if distance is similar can get different response
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test1/response-limit-1.json")

			By("Validate response returns 2 items when limit=2")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-2.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(2))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test1/response-limit-2.json")

			By("Validate response returns 18 (all available) items when limit=100")
			jsonData = ReadTestDataFile("query-chunks_v2/test1/query-limit-100.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(18))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test1/response-limit-100.json")

		})

		// Test maxDistance
		//  Return all when maxDistance set not set.
		//  Return exact number when maxDistance is set to 0.0.
		//  Return all when maxDistance set to 1.0.
		//  Return exact number when limit is set.
		//  Return items sorted by maxDistance.
		It("v2 test2: must return number of records defined by 'maxDistance' parameter", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-chunks_v2/test2/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			var jsonData string
			var body []byte
			var resp *http.Response
			var items []QueryChunkItem

			By("Validate response returns all elements when 'maxDistance' is not set")
			jsonData = ReadTestDataFile("query-chunks_v2/test2/query-maxdistance-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(8))
			// We only care that the expected fields are present for each item;
			// item order and concrete values (e.g. distance) may vary with embeddings.
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test2/response-maxdistance-not-set.json")

			By("Validate response returns chunks with a distance of exactly 0 when maxDistance=0")
			jsonData = ReadTestDataFile("query-chunks_v2/test2/query-maxdistance-0.0.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(8))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test2/response-maxdistance-0.0.json")

			By("Validate response returns all items when maxDistance=1.0")
			jsonData = ReadTestDataFile("query-chunks_v2/test2/query-maxdistance-1.0.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(8))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test2/response-maxdistance-1.0.json")

			By("Validate response returns items within the maxDistance threshold (0.29)")
			jsonData = ReadTestDataFile("query-chunks_v2/test2/query-maxdistance-0.29.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(BeNumerically(">", 0))
			for _, item := range items {
				Expect(item.Distance < 0.29).To(BeTrue())
			}
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test2/response-maxdistance-0.29.json")

			By("Validate returned items sorted by maxDistance")
			jsonData = ReadTestDataFile("query-chunks_v2/test2/query-maxdistance-1.0.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			for i := 0; i < len(items)-1; i++ {
				Expect(items[i].Distance <= items[i+1].Distance).To(BeTrue())
			}
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test2/response-maxdistance-1.0.json")

		})

		// Test retrieveAttributes
		//  Return all attributes when retrieveAttributes is not set
		//  Return only chunks (w/o attributes) when retrieveAttributes is empty
		//  Return exact attributes when retrieveAttributes is set
		It("v2 test3: must return records with retrieveAttributes", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-chunks_v2/test3/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			var jsonData string
			var body []byte
			var resp *http.Response
			var items []QueryChunkItem

			By("Validate response returns all attributes when retrieveAttributes not set")
			jsonData = ReadTestDataFile("query-chunks_v2/test3/query-retrieveAttributes-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(7))
			//ExpectSortedJSONEqualsFromFile(string(body), "query-chunks_v2/test3/response-retrieveAttributes-not-set.json") //

			By("Validate response returns only chunks (w/o attributes) when retrieveAttributes is empty")
			jsonData = ReadTestDataFile("query-chunks_v2/test3/query-retrieveAttributes-empty.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(7))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test3/response-retrieveAttributes-empty.json")

			By("Validate response returns only attributes defined in retrieveAttributes")
			jsonData = ReadTestDataFile("query-chunks_v2/test3/query-retrieveAttributes-2.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(7))
			expectedAttributes := []string{"roles", "title"}
			for _, item := range items {
				Expect(len(item.Attributes) <= len(expectedAttributes)).To(BeTrue())
				for _, attr := range item.Attributes {
					Expect(slices.Contains(expectedAttributes, attr.Name)).To(BeTrue())
				}
			}
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test3/response-retrieveAttributes-2.json")

			By("Validate response returns all attributes when retrieveAttributes not set")
			jsonData = ReadTestDataFile("query-chunks_v2/test3/query-retrieveAttributes-extra.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(7))
			expectedAttributes = []string{"roles", "title", "dataSource", "command", "content", "notExistentAttribute"}
			for _, item := range items {
				Expect(len(item.Attributes) <= len(expectedAttributes)).To(BeTrue())
				for _, attr := range item.Attributes {
					Expect(slices.Contains(expectedAttributes, attr.Name)).To(BeTrue())
				}
			}
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test3/response-retrieveAttributes-extra.json")

		})

		// Test filters.attributes
		// Returns all chunks when attributes not set
		// Returns error when attributes.filter is empty
		// Returns exact chunks when attributes are provided
		//  Test filtering by attributes.name
		//  Test filtering by attributes.type
		//  Test filtering by attributes.value
		It("v2 test4: must return records filtered by 'filters.attributes' parameter", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-chunks_v2/test4/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item

			By("Validate response returns all attributes when filters.attributes not set")
			jsonData = ReadTestDataFile("query-chunks_v2/test4/query-filters.attributes-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(9))
			//ExpectSortedJSONEqualsFromFile(string(body), "query-chunks_v2/test4/response-filters.attributes-not-set.json") //

			//// FIXME: BUG-858809 : query chunk must return error when filters.attributes are empty
			//By("Validate response returns error when filters.attributes is empty")
			//jsonData = ReadTestDataFile("query-chunks_v2/test4/query-filters.attributes-empty.json")
			//resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			//Expect(err).To(BeNil())
			//Expect(body).NotTo(BeNil())
			//Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			By("Validate response returns attributes filtered by filters.attributes.name")
			jsonData = ReadTestDataFile("query-chunks_v2/test4/query-filters.attributes-by-value.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(8))
			ExpectArrayItemsHaveFieldsFromFile(string(body), "query-chunks_v2/test4/response-filters.attributes-by-value.json")
		})

		It("v2 test5.1: return all embeddings when filters.attributes is not set", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)
			By("Insert documents")
			UpsertDocumentsFromDirAndWaitForCOMPLETED(database, "query-chunks_v2/test5/documents", baseURI, isolationID, collectionID)
			By("Validate response")
			ValidateResponseMatchedFileData(endpointURI,
				"query-chunks_v2/test5/not-set.request.json",
				"query-chunks_v2/test5/not-set.response.json")
		})

		It("v2 test5.2: return selected embedding when filters.attributes defines one attribute on document level", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)
			By("Insert documents")
			UpsertDocumentsFromDirAndWaitForCOMPLETED(database, "query-chunks_v2/test5/documents", baseURI, isolationID, collectionID)
			By("Validate response")
			ValidateResponseMatchedFileData(endpointURI,
				"query-chunks_v2/test5/one-attr-on-doc-level.request.json",
				"query-chunks_v2/test5/one-attr-on-doc-level.response.json")
		})

		It("v2 test5.3: return selected embedding when filters.attributes defines one attribute on embedding level", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)
			By("Insert documents")
			UpsertDocumentsFromDirAndWaitForCOMPLETED(database, "query-chunks_v2/test5/documents", baseURI, isolationID, collectionID)
			By("Validate response")
			ValidateResponseMatchedFileData(endpointURI,
				"query-chunks_v2/test5/one-attr-on-emb-level.request.json",
				"query-chunks_v2/test5/one-attr-on-emb-level.response.json")
		})

		It("v2 test5.4: return selected embedding when filters.attributes contains 2 attributes on embedding level", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)
			By("Insert documents")
			UpsertDocumentsFromDirAndWaitForCOMPLETED(database, "query-chunks_v2/test5/documents", baseURI, isolationID, collectionID)
			By("Validate response")
			ValidateResponseMatchedFileData(endpointURI,
				"query-chunks_v2/test5/two-attrs-on-emb-level.request.json",
				"query-chunks_v2/test5/two-attrs-on-emb-level.response.json")
		})

		It("v2 test5.5: return selected embedding when filters.attributes contains 2 attributes where one of then is only on document level", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)
			By("Insert documents")
			UpsertDocumentsFromDirAndWaitForCOMPLETED(database, "query-chunks_v2/test5/documents", baseURI, isolationID, collectionID)
			By("Validate response")
			ValidateResponseMatchedFileData(endpointURI,
				"query-chunks_v2/test5/two-attrs-on-doc-level.request.json",
				"query-chunks_v2/test5/two-attrs-on-doc-level.response.json")
		})

		It("test12d (query/chunk): must return chunks by filtered attributes when attributes assigned on document level", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-chunks_v2/test12d/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item

			By("Validate response")
			jsonData = ReadTestDataFile("query-chunks_v2/test12d/query.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(4))
		})

		It("test12e (query/chunk): must return chunks by filtered attributes when attributes assigned on chunk level", func() {
			// Create mocks
			By("Creating WireMock expectations for embeddings")
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("query-chunks_v2/test12e/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}
			var jsonData string
			var body []byte
			var resp *http.Response
			var items []Item

			By("Validate response")
			jsonData = ReadTestDataFile("query-chunks_v2/test12e/query.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(4))
		})

		_ = Context("service response should return expected headers", func() {
			It("test 13: check if query chunks endpoint returns all expected headers", func() {
				// Mock ADA
				By("Creating WireMock expectations for embeddings")
				mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
				Expect(err).To(BeNil())
				testExpectations = append(testExpectations, mockID)

				By("Insert document")
				docIDs := UpsertDocumentsFromDir("query-chunks_v2/test13/documents", baseURI, isolationID, collectionID)
				By("Waiting for completion")
				for _, docID := range docIDs {
					WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
				}

				By("Query chunks with test data")
				var resp *http.Response
				var body []byte
				resp, body, err = HttpCall("POST", endpointURI, nil, ReadTestDataFile("query-chunks_v2/test13/query.json"))
				Expect(err).To(BeNil())
				Expect(body).NotTo(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				By("Check headers in the response")
				ExpectHeadersCommon(resp)
				ExpectHeadersDatabase(resp)
				ExpectHeadersItemsCount(resp, 2)
				ExpectHeadersEmbedding(resp, 0, 1)
				ExpectHeadersProcessingOverhead(resp)
			})
		})

	})
})

func ValidateResponseMatchedFileData(endpointURI, requestDataFile, responseDataFile string) {
	By(fmt.Sprintf("Validate response equals data from %s", responseDataFile))

	var body []byte
	var resp *http.Response
	var items []Item
	var err error

	jsonData := ReadTestDataFile(requestDataFile)
	resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(body).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = json.Unmarshal(body, &items)
	Expect(err).To(BeNil())
	ExpectArrayItemsHaveFieldsFromFile(string(body), responseDataFile)
}
