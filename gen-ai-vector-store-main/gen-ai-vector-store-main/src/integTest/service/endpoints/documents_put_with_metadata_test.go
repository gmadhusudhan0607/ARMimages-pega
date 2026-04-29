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
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing metadata PUT SVC /v1/{isolationID}/collections/{collectionName}/documents", func() {

	var isolationID string
	var collectionID string
	var testExpectations []string

	_ = Context("accessing endpoint with PUT method SYNC", func() {
		var endpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong)
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

		It("test1s: Put document with embeddingAttributes on chunk level SYNC", func() {
			// Mock ADA
			testDataDir := "documents-put-with-metadata/test1"
			expTpl := ReadFromTesDatatDir(testDataDir, "mock-expectation-tpl.json")

			// Expect this content to be sent to ADA for chunk 0
			jsonData := fmt.Sprintf(expTpl, isolationID, collectionID, "region: EU, US | Content: text of chunk 1")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			// Expect this content to be sent to ADA for chunk 1
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "org: dev | version: v1, v1.1 | Content: text of chunk 2")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadFromTesDatatDir(testDataDir, "DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("Validate if data was inserted to database for the first call")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "text of chunk 1",
					Embedding: make([]float32, 1536),
				},
			})
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-1",
					Content:   "text of chunk 2",
					Embedding: make([]float32, 1536),
				},
			})

			ExpectEmbeddingMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1-EMB-0", embedings.MetadataKeyStaticEmbeddingAttributes, "region,not-existing-chunk-attribute")
			ExpectEmbeddingMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1-EMB-1", embedings.MetadataKeyStaticEmbeddingAttributes, "region,org,version")

			// ------------ Update document and check if the metadata is updated --------------------

			// Cleanup expectations created before
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
			// Create new expectations
			// Expect this content to be sent to ADA for chunk 0
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "region: EU, US | Content: text of chunk 1")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			// Expect this content to be sent to ADA for chunk 1
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "org: dev | Content: text of chunk 2")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			resp, body, err = HttpCall("PUT", endpointURI, nil, ReadFromTesDatatDir(testDataDir, "DOC-1u.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			ExpectEmbeddingMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1-EMB-0", embedings.MetadataKeyStaticEmbeddingAttributes, "region")
			ExpectEmbeddingMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1-EMB-1", embedings.MetadataKeyStaticEmbeddingAttributes, "org")

		})

		It("test2s: Put document with embeddingAttributes on document level SYNC", func() {
			// Mock ADA
			testDataDir := "documents-put-with-metadata/test2"
			expTpl := ReadFromTesDatatDir(testDataDir, "mock-expectation-tpl.json")

			// Expect this content to be sent to ADA for chunk 0
			jsonData := fmt.Sprintf(expTpl, isolationID, collectionID, "access: private | region: EU, UA, US | version: v0, v0.0 | Content: text of chunk 1")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			// Expect this content to be sent to ADA for chunk 1
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "org: DEV | region: EU, US | version: v0, v0.1 | Content: text of chunk 2")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadFromTesDatatDir(testDataDir, "DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("Validate if data was inserted to database for the first call")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "text of chunk 1",
					Embedding: make([]float32, 1536),
				},
			})
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-1",
					Content:   "text of chunk 2",
					Embedding: make([]float32, 1536),
				},
			})

			ExpectDocumentMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1", documents.MetadataKeyStaticEmbeddingAttributes, "version,region,not-existing-document-attribute")

			// ------------ Update document and check if the metadata is updated --------------------

			// Cleanup expectations created before
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}

			// Create new expectations
			// Expect this content to be sent to ADA for chunk 0
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "access: private | name: test 0.0 | version: v0, v0.0 | Content: text of chunk 1")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			// Expect this content to be sent to ADA for chunk 1
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "name: test 0.1 | org: DEV | version: v0, v0.1 | Content: text of chunk 2")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			resp, body, err = HttpCall("PUT", endpointURI, nil, ReadFromTesDatatDir(testDataDir, "DOC-1u.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			ExpectDocumentMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1", documents.MetadataKeyStaticEmbeddingAttributes, "version,name")

		})
	})

	_ = Context("accessing endpoint with PUT method ASYNC", func() {
		var endpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s",
				baseURI, isolationID, collectionID, indexer.ConsistencyLevelEventual)
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

		It("test1a: Put document with embeddingAttributes on chunk level ASYNC", func() {
			// Mock ADA
			testDataDir := "documents-put-with-metadata/test1"
			expTpl := ReadFromTesDatatDir(testDataDir, "mock-expectation-tpl.json")

			// Expect this content to be sent to ADA for chunk 0
			jsonData := fmt.Sprintf(expTpl, isolationID, collectionID, "region: EU, US | Content: text of chunk 1")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			// Expect this content to be sent to ADA for chunk 1
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "org: dev | version: v1, v1.1 | Content: text of chunk 2")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadFromTesDatatDir(testDataDir, "DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("Validate if data was inserted to database for the first call")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "text of chunk 1",
					Embedding: make([]float32, 1536),
				},
			})
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-1",
					Content:   "text of chunk 2",
					Embedding: make([]float32, 1536),
				},
			})

			ExpectEmbeddingMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1-EMB-0", embedings.MetadataKeyStaticEmbeddingAttributes, "region,not-existing-chunk-attribute")
			ExpectEmbeddingMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1-EMB-1", embedings.MetadataKeyStaticEmbeddingAttributes, "region,org,version")

			// ------------ Update document and check if the metadata is updated --------------------

			// Cleanup expectations created before
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
			// Create new expectations
			// Expect this content to be sent to ADA for chunk 0
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "region: EU, US | Content: text of chunk 1")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			// Expect this content to be sent to ADA for chunk 1
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "org: dev | Content: text of chunk 2")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			resp, body, err = HttpCall("PUT", endpointURI, nil, ReadFromTesDatatDir(testDataDir, "DOC-1u.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			ExpectEmbeddingMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1-EMB-0", embedings.MetadataKeyStaticEmbeddingAttributes, "region")
			ExpectEmbeddingMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1-EMB-1", embedings.MetadataKeyStaticEmbeddingAttributes, "org")

		})

		It("test2a: Put document with embeddingAttributes on document level ASYNC", func() {
			// Mock ADA
			testDataDir := "documents-put-with-metadata/test2"
			expTpl := ReadFromTesDatatDir(testDataDir, "mock-expectation-tpl.json")

			// Expect this content to be sent to ADA for chunk 0
			jsonData := fmt.Sprintf(expTpl, isolationID, collectionID, "access: private | region: EU, UA, US | version: v0, v0.0 | Content: text of chunk 1")
			mockID, err := CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			// Expect this content to be sent to ADA for chunk 1
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "org: DEV | region: EU, US | version: v0, v0.1 | Content: text of chunk 2")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			resp, body, err := HttpCall("PUT", endpointURI, nil, ReadFromTesDatatDir(testDataDir, "DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			By("Validate if data was inserted to database for the first call")
			ExpectChunksCountInDatabase(ctx, database, isolationID, collectionID, "DOC-1", 2)
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-0",
					Content:   "text of chunk 1",
					Embedding: make([]float32, 1536),
				},
			})
			ExpectEmbeddingsInDatabase(ctx, database, isolationID, collectionID, "DOC-1", []embedings.Embedding{
				{
					ID:        "DOC-1-EMB-1",
					Content:   "text of chunk 2",
					Embedding: make([]float32, 1536),
				},
			})

			ExpectDocumentMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1", documents.MetadataKeyStaticEmbeddingAttributes, "version,region,not-existing-document-attribute")

			// ------------ Update document and check if the metadata is updated --------------------

			// Cleanup expectations created before
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}

			// Expect this content to be sent to ADA for chunk 0
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "access: private | name: test 0.0 | version: v0, v0.0 | Content: text of chunk 1")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			// Expect this content to be sent to ADA for chunk 1
			jsonData = fmt.Sprintf(expTpl, isolationID, collectionID, "name: test 0.1 | org: DEV | version: v0, v0.1 | Content: text of chunk 2")
			mockID, err = CreateAdaExpectationFromTpl(wiremockManager, jsonData)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			resp, body, err = HttpCall("PUT", endpointURI, nil, ReadFromTesDatatDir(testDataDir, "DOC-1u.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-1", "COMPLETED")

			ExpectDocumentMetadataInDatabase(ctx, database, isolationID, collectionID,
				"DOC-1", documents.MetadataKeyStaticEmbeddingAttributes, "version,name")

		})
	})

	_ = Context("accessing endpoint /file with PUT method (async SC job submission)", func() {
		var endpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/file",
				baseURI, isolationID, collectionID)
			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(ctx, database, isolationID)
				DeleteIsolation(opsURI, isolationID)
			}
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("test3s : Put .pdf file document with metadata returns 202", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			docAttrs := []attributes.Attribute{
				{Name: "Region", Type: "string", Values: []string{"GLOBAL"}},
			}
			docMetadata := documents.DocumentMetadata{
				StaticEmbeddingAttributes: []string{"Region"},
				ExtraAttributesKinds:      []string{"auto-resolved"},
			}
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: docMetadata},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-with-metadata/test3/Astronomy.pdf")},
			}

			By("Put document")
			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)

			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify 202 response body")
			var respBody map[string]interface{}
			err = json.Unmarshal(body, &respBody)
			Expect(err).To(BeNil())
			Expect(respBody).To(HaveKey("documentID"))
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))
		})
	})

})
