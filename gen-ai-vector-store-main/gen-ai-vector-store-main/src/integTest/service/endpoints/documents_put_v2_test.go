//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package endpoints_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing PUT SVC /v1/{isolationID}/collections/{collectionName}/documents", func() {

	var isolationID string
	var collectionID string
	var testExpectations []string

	_ = Context("accessing endpoint with PUT method and strong consistency level", func() {
		var endpointURI, attrEndpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong)
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
			ExpectServiceReturns404IfIsolationDoesNotExist("PUT", endpointURI)
		})

		It("test1: Put document successfully for strong consistency level (synchronous operation)", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test1/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for strong consistency level")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test1/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate if data was inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:         "DOC-1-EMB-0",
					Content:    "some text here 1",
					Embedding:  make([]float32, 1536),
					Attributes: attributes.Attributes{{Name: "version", Type: "string", Values: []string{"c1", "v1"}}},
				},
				{
					ID:         "DOC-1-EMB-1",
					Content:    "some text here 2",
					Embedding:  make([]float32, 1536),
					Attributes: attributes.Attributes{{Name: "version", Type: "string", Values: []string{"c2", "v2"}}},
				},
			})

			By("Validate if attributes were successfully inserted")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test1/attributes-expected-response-1.json")
		})

		It("test1a: Put document successfully for strong consistency (long collection name (254 chars))", func() {
			// Create isolation with long collection name
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(250)))
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong)
			attrEndpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/attributes", baseURI, isolationID, collectionID)

			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test1/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for strong consistency level")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test1/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate if data was inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:         "DOC-1-EMB-0",
					Content:    "some text here 1",
					Embedding:  make([]float32, 1536),
					Attributes: attributes.Attributes{{Name: "version", Type: "string", Values: []string{"c1", "v1"}}},
				},
				{
					ID:         "DOC-1-EMB-1",
					Content:    "some text here 2",
					Embedding:  make([]float32, 1536),
					Attributes: attributes.Attributes{{Name: "version", Type: "string", Values: []string{"c2", "v2"}}},
				},
			})

			By("Validate if attributes were successfully inserted")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test1/attributes-expected-response-1.json")
		})

		It("test2: Put document for strong consistency level when ada returns error", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings error"))
			expTpl := ReadTestDataFile("documents-put/test2/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for strong consistency level")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test2/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			Expect(string(body)).To(ContainSubstring("embedding returned status code 401 without an error"))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusError)

			By("Validate if data was not inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 0)
			ExpectEmbeddingsEmptyInDatabase(ctx, database, isolationID, collectionID, "DOC-1")

			By("Validate if error in db matches error returned by ada API")
			ExpectDocumentErrorInDBOneOf(ctx, database, isolationID, collectionID, "DOC-1",
				"[{\"status\" : \"ERROR\", \"code\" : 401, \"message\" : \"embedding returned status code 401 without an error\", \"count\" : 2}]",
				"[{\"status\" : \"ERROR\", \"code\" : 401, \"message\" : \"embedding returned status code 401 without an error\", \"count\" : 1}, {\"status\" : \"IN_PROGRESS\", \"code\" : 0, \"message\" : \"\", \"count\" : 1}]",
				"[{\"status\" : \"ERROR\", \"code\" : 401, \"message\" : \"embedding returned status code 401 without an error\", \"count\" : 1}, {\"status\" : \"ERROR\", \"code\" : 500, \"message\" : \"embedding returned an error: \", \"count\" : 1}]")
		})

		It("test3: Put document successfully twice and validate if content has changed", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test3/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for eventual consistency level for the first call")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test3/DOC-1a.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion for the first call")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate if data was inserted to database for the first call")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 1)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "some text here 1a",
					Embedding: make([]float32, 1536),
				},
			})

			By("Validate if attributes were successfully inserted for the first call")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test3/attributes-expected-response-1.json")

			By("Put document for eventual consistency level for the second call")
			resp, body, err = HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test3/DOC-1b.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion for the second call")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate if data is inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 1)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "some text here 1b",
					Embedding: make([]float32, 1536),
				},
			})

			By("Validate if attributes were successfully inserted")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test3/attributes-expected-response-2.json")
		})

		It("test11: validate attr_id2 attributes", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test11/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for eventual consistency level for the first call")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test11/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion for the first call")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate embedding attr_id2")

			/// -- Chunk 0
			emb := GetEmbeddingFromDB(ctx, database, isolationID, collectionID, "DOC-1", "DOC-1-EMB-0")
			Expect(emb).NotTo(BeNil())
			// Check if attr_ids contains attribute 'chunkType' (chunk level)
			attrID := GetAttrIdFormDB(ctx, database, isolationID, collectionID, "chunkType", "string", "finance")
			Expect(attrID).NotTo(Equal(0))
			Expect(slices.Contains(emb.AttrIDs, attrID)).To(BeTrue())
			// Check if attr_ids2 contains attribute 'chunkType' (chunk level)
			attrID = GetAttrIdFormDB(ctx, database, isolationID, collectionID, "chunkType", "string", "finance")
			Expect(attrID).NotTo(Equal(0))
			Expect(slices.Contains(emb.AttrIDs2, attrID)).To(BeTrue())
			// Check if attr_ids2 contains attribute 'chunkType' documentType (document level)
			attrID = GetAttrIdFormDB(ctx, database, isolationID, collectionID, "documentType", "string", "contract")
			Expect(attrID).NotTo(Equal(0))
			Expect(slices.Contains(emb.AttrIDs2, attrID)).To(BeTrue())
			// Check if attr_ids2 contains attribute 'org' documentType (document level)
			attrID = GetAttrIdFormDB(ctx, database, isolationID, collectionID, "org", "string", "tso")
			Expect(attrID).NotTo(Equal(0))
			Expect(slices.Contains(emb.AttrIDs2, attrID)).To(BeTrue())

			/// -- Chunk 1
			emb = GetEmbeddingFromDB(ctx, database, isolationID, collectionID, "DOC-1", "DOC-1-EMB-1")
			Expect(emb).NotTo(BeNil())
			// Check if attr_ids contains attribute 'chunkType' (chunk level)
			attrID = GetAttrIdFormDB(ctx, database, isolationID, collectionID, "chunkType", "string", "general")
			Expect(attrID).NotTo(Equal(0))
			Expect(slices.Contains(emb.AttrIDs, attrID)).To(BeTrue())
			// Check if attr_ids2 contains attribute 'chunkType' (chunk level)
			attrID = GetAttrIdFormDB(ctx, database, isolationID, collectionID, "chunkType", "string", "general")
			Expect(attrID).NotTo(Equal(0))
			Expect(slices.Contains(emb.AttrIDs2, attrID)).To(BeTrue())
			// Check if attr_ids2 contains attribute 'documentType' (document level)
			attrID = GetAttrIdFormDB(ctx, database, isolationID, collectionID, "documentType", "string", "contract")
			Expect(attrID).NotTo(Equal(0))
			Expect(slices.Contains(emb.AttrIDs2, attrID)).To(BeTrue())
			// Check if attr_ids2 contains attribute 'org' (document level)
			attrID = GetAttrIdFormDB(ctx, database, isolationID, collectionID, "org", "string", "tso")
			Expect(attrID).NotTo(Equal(0))
			Expect(slices.Contains(emb.AttrIDs2, attrID)).To(BeTrue())

		})
	})

	_ = Context("accessing endpoint with PUT method and eventual consistency level", func() {
		var endpointURI, attrEndpointURI string
		ctx := context.TODO()
		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual)
			attrEndpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/attributes", baseURI, isolationID, collectionID)

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			report := CurrentSpecReport()
			if !report.Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("test4: Put document successfully for eventual consistency level (asynchronous operation)", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test4/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for eventual consistency level")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test4/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate if data was inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "some text here 1",
					Embedding: make([]float32, 1536),
					Attributes: attributes.Attributes{
						{
							Name:   "version",
							Type:   "string",
							Values: []string{"c1", "v1"},
						},
					},
				},
				{
					ID:        "DOC-1-EMB-1",
					Content:   "some text here 2",
					Embedding: make([]float32, 1536),
					Attributes: attributes.Attributes{
						{
							Name:   "version",
							Type:   "string",
							Values: []string{"c2", "v2"},
						},
					},
				},
			})

			By("Validate if attributes were successfully inserted")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test4/attributes-expected-response-1.json")
		})

		It("test5: Put document for eventual consistency level when ada returns error", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings error"))
			expTpl := ReadTestDataFile("documents-put/test5/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for strong consistency level")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test5/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusError)

			By("Validate if data was not inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 0)
			ExpectChunksProcessingCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsProcessingEmptyInDatabase(ctx, database, isolationID, collectionID, "DOC-1")

			By("Validate if error in db matches error returned by ada API")
			ExpectDocumentErrorInDBOneOf(ctx, database, isolationID, collectionID, "DOC-1",
				"[{\"status\" : \"ERROR\", \"code\" : 401, \"message\" : \"embedding returned status code 401 without an error\", \"count\" : 2}]",
				"[{\"status\" : \"ERROR\", \"code\" : 401, \"message\" : \"embedding returned status code 401 without an error\", \"count\" : 1}, {\"status\" : \"IN_PROGRESS\", \"code\" : 0, \"message\" : \"\", \"count\" : 1}]",
				"[{\"status\" : \"ERROR\", \"code\" : 401, \"message\" : \"embedding returned status code 401 without an error\", \"count\" : 1}, {\"status\" : \"ERROR\", \"code\" : 500, \"message\" : \"embedding returned an error: \", \"count\" : 1}]")
		})

		It("test6: Put multiple documents successfully for eventual consistency level", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test6/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put documents for strong consistency level")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test6/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			resp, body, err = HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test6/DOC-2.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			resp, body, err = HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test6/DOC-3.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-2", resources.StatusCompleted)
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-3", resources.StatusCompleted)

			By("Validate if data was inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "some text here 1",
					Embedding: make([]float32, 1536),
				},
				{
					ID:        "DOC-1-EMB-1",
					Content:   "some text here 2",
					Embedding: make([]float32, 1536),
				},
			})

			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-2", 1)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-2", []embedings.Embedding{
				{
					ID:        "DOC-2-EMB-0",
					Content:   "some text here 1",
					Embedding: make([]float32, 1536),
				},
			})

			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-3", 3)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-3", []embedings.Embedding{
				{
					ID:        "DOC-3-EMB-0",
					Content:   "some text here 1",
					Embedding: make([]float32, 1536),
				},
				{
					ID:        "DOC-3-EMB-1",
					Content:   "some text here 2",
					Embedding: make([]float32, 1536),
				},
				{
					ID:        "DOC-3-EMB-2",
					Content:   "some text here 3",
					Embedding: make([]float32, 1536),
				},
			})

			By("Validate if attributes were successfully inserted")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test6/attributes-expected-response-1.json")
		})

		It("test7: Put document successfully twice and validate if content has changed", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test7/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for eventual consistency level for the first call")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test7/DOC-1a.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion for the first call")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate if data was inserted to database for the first call")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 1)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "some text here 1a",
					Embedding: make([]float32, 1536),
				},
			})

			By("Validate if attributes were successfully inserted for the first call")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test7/attributes-expected-response-1.json")

			By("Put document for eventual consistency level for the second call")
			resp, body, err = HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test7/DOC-1b.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion for the second call")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate if data is inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 1)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "some text here 1b",
					Embedding: make([]float32, 1536),
				},
			})

			By("Validate if attributes were successfully inserted")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test7/attributes-expected-response-2.json")
		})

	})

	_ = Context("accessing endpoint with PUT method and default consistency level", func() {
		var endpointURI, attrEndpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents", baseURI, isolationID, collectionID)
			attrEndpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/attributes", baseURI, isolationID, collectionID)

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			report := CurrentSpecReport()
			if !report.Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("test8: Put document successfully for default consistency level (asynchronous operation)", func() {
			// Mock ADA using WireMock
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test8/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for default consistency level")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadTestDataFile("documents-put/test8/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Validate if data was inserted to database")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "some text here 1",
					Embedding: make([]float32, 1536),
					Attributes: attributes.Attributes{
						{
							Name:   "version",
							Type:   "string",
							Values: []string{"c1", "v1"},
						},
					},
				},
				{
					ID:        "DOC-1-EMB-1",
					Content:   "some text here 2",
					Embedding: make([]float32, 1536),
					Attributes: attributes.Attributes{
						{
							Name:   "version",
							Type:   "string",
							Values: []string{"c2", "v2"},
						},
					},
				},
			})

			By("Validate if attributes were successfully inserted")
			resp, body, err = HttpCall("POST", attrEndpointURI, nil, "{}")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			ExpectResponseMatchFromFile(string(body), "documents-put/test8/attributes-expected-response-1.json")
		})
	})

	_ = Context("service response should return expected headers", func() {
		var endpointStrongURI, endpointEventualURI string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

			endpointStrongURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong)
			endpointEventualURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual)

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed or marked to be skipped (So the results could be analyzed)
			report := CurrentSpecReport()
			if !report.Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsURI, isolationID)
			}
			// Cleanup expectations created by test (to avoid conflicts during development)
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("test12a: check if strong consistency level returns all expected headers", func() {
			// Mock ADA using WireMock (including delay for header timing assertions)
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test12/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for strong consistency level")
			resp, body, err := HttpCall("PUT", endpointStrongURI, nil, ReadTestDataFile("documents-put/test12/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersEmbedding(resp, 1000, 2)
		})

		It("test12b: check if eventual consistency level returns expected headers but not embedding headers", func() {
			// Mock ADA using WireMock (not strictly necessary for this test since with eventual
			// consistency the embedding happens asynchronously)
			By(fmt.Sprintf("Creating WireMock expectations for embeddings"))
			expTpl := ReadTestDataFile("documents-put/test12/mock-expectation-tpl.json")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, expTpl)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for eventual consistency level")
			resp, body, err := HttpCall("PUT", endpointEventualURI, nil, ReadTestDataFile("documents-put/test12/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", resources.StatusCompleted)

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
		})
	})
})
