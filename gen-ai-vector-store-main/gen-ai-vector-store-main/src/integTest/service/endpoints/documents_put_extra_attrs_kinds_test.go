/*
 * Copyright (c) 2026 Pegasystems Inc.
 * All rights reserved.
 */

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

var _ = Describe("Testing extraAttributesKinds kind filtering in PUT SVC /v1/{isolationID}/collections/{collectionName}/documents", func() {

	var isolationID string
	var collectionID string
	var testExpectations []string

	_ = Context("kind filtering controls which chunk attributes are stored in attr_ids2", func() {
		var endpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s",
				baseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong)
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

		It("test-ar-stored: auto-resolved attribute is stored in attr_ids2 when extraAttributesKinds includes it", func() {
			By("Creating generic ADA embedding mock")
			mockID, err := CreateExpectationEmbeddingAdaWithoutIsolationValidation(wiremockManager)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Building PUT /documents request: DOC-AR + extraAttributesKinds=[auto-resolved]")
			reqBody := putDocumentWithMetadata(
				ReadFromTesDatatDir("documents-put-extra-kinds/test-auto-resolved", "DOC-AR.json"),
				&documents.DocumentMetadata{ExtraAttributesKinds: []string{"auto-resolved"}},
			)

			resp, body, err := HttpCall("PUT", endpointURI, nil, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for document COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-AR", "COMPLETED")

			By("Verify auto-resolved attribute is linked via attr_ids2")
			ExpectChunksInDatabase(ctx, database, isolationID, collectionID, "DOC-AR", []embedings.Chunk{
				{
					ID:      "DOC-AR-EMB-0",
					Content: "some text",
					Attributes: attributes.Attributes{
						{Name: "region", Type: "string"},
					},
				},
			})
		})

		It("test-ar-dropped: auto-resolved attribute is NOT stored when extraAttributesKinds is absent", func() {
			By("Creating generic ADA embedding mock")
			mockID, err := CreateExpectationEmbeddingAdaWithoutIsolationValidation(wiremockManager)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Building PUT /documents request: DOC-AR without metadata")
			docJSON := ReadFromTesDatatDir("documents-put-extra-kinds/test-auto-resolved", "DOC-AR.json")

			resp, body, err := HttpCall("PUT", endpointURI, nil, docJSON)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for document COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-AR", "COMPLETED")

			By("Verify no attributes are linked via attr_ids2")
			ExpectChunkHasNoAttributesLinked(ctx, database, isolationID, collectionID, "DOC-AR-EMB-0")
		})

		It("test-index-stored: index attribute is stored in attr_ids2 when extraAttributesKinds includes it", func() {
			By("Creating generic ADA embedding mock")
			mockID, err := CreateExpectationEmbeddingAdaWithoutIsolationValidation(wiremockManager)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Building PUT /documents request: DOC-IDX + extraAttributesKinds=[index]")
			reqBody := putDocumentWithMetadata(
				ReadFromTesDatatDir("documents-put-extra-kinds/test-index", "DOC-IDX.json"),
				&documents.DocumentMetadata{ExtraAttributesKinds: []string{"index"}},
			)

			resp, body, err := HttpCall("PUT", endpointURI, nil, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for document COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-IDX", "COMPLETED")

			By("Verify index attribute is linked via attr_ids2")
			ExpectChunksInDatabase(ctx, database, isolationID, collectionID, "DOC-IDX", []embedings.Chunk{
				{
					ID:      "DOC-IDX-EMB-0",
					Content: "intro content",
					Attributes: attributes.Attributes{
						{Name: "section", Type: "string"},
					},
				},
			})
		})

		It("test-index-dropped: index attribute is NOT stored when extraAttributesKinds is absent", func() {
			By("Creating generic ADA embedding mock")
			mockID, err := CreateExpectationEmbeddingAdaWithoutIsolationValidation(wiremockManager)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Building PUT /documents request: DOC-IDX without metadata")
			docJSON := ReadFromTesDatatDir("documents-put-extra-kinds/test-index", "DOC-IDX.json")

			resp, body, err := HttpCall("PUT", endpointURI, nil, docJSON)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for document COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-IDX", "COMPLETED")

			By("Verify no attributes are linked via attr_ids2")
			ExpectChunkHasNoAttributesLinked(ctx, database, isolationID, collectionID, "DOC-IDX-EMB-0")
		})

		It("test-ar-embed-injected: auto-resolved attribute is injected into embedding input when extraAttributesKinds includes it", func() {
			By("Creating ADA embedding mock that expects region attribute in input")
			embeddingInput := "region: EU | Content: some text"
			adaRequest := map[string]interface{}{
				"method":  "POST",
				"urlPath": "/openai/deployments/text-embedding-ada-002/embeddings",
				"bodyPatterns": []map[string]interface{}{
					{"contains": embeddingInput},
				},
			}
			mockID, err := wiremockManager.CreateStub(adaRequest, CreateAdaStubResponse(CreateAdaEmbeddingVector(), 200, nil))
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Building PUT /documents request: DOC-AR-embed + extraAttributesKinds=[auto-resolved]")
			reqBody := putDocumentWithMetadata(
				ReadFromTesDatatDir("documents-put-extra-kinds/test-auto-resolved", "DOC-AR-embed.json"),
				&documents.DocumentMetadata{ExtraAttributesKinds: []string{"auto-resolved"}},
			)

			resp, body, err := HttpCall("PUT", endpointURI, nil, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for document COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-AR", "COMPLETED")

			By("Verify the ADA mock with region-prefixed input was called")
			count, err := GetCallCountByMockID(wiremockManager, mockID)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(1),
				fmt.Sprintf("Expected ADA to be called once with input '%s'", embeddingInput))
		})

		It("test-ar-embed-not-injected: auto-resolved attribute is NOT injected into embedding when extraAttributesKinds is absent", func() {
			By("Creating ADA embedding mock that expects content-only input (no attribute prefix)")
			embeddingInput := "Content: some text"
			adaRequest := map[string]interface{}{
				"method":  "POST",
				"urlPath": "/openai/deployments/text-embedding-ada-002/embeddings",
				"bodyPatterns": []map[string]interface{}{
					{"equalTo": fmt.Sprintf(`{"input":"%s"}`, embeddingInput)},
				},
			}
			mockID, err := wiremockManager.CreateStub(adaRequest, CreateAdaStubResponse(CreateAdaEmbeddingVector(), 200, nil))
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Building PUT /documents request: DOC-AR-embed without extraAttributesKinds")
			docJSON := ReadFromTesDatatDir("documents-put-extra-kinds/test-auto-resolved", "DOC-AR-embed.json")

			resp, body, err := HttpCall("PUT", endpointURI, nil, docJSON)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for document COMPLETED")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID, "DOC-AR", "COMPLETED")

			By("Verify the ADA mock with content-only input was called (attribute was dropped)")
			count, err := GetCallCountByMockID(wiremockManager, mockID)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(1),
				fmt.Sprintf("Expected ADA to be called once with input '%s'", embeddingInput))
		})

		It("test-invalid-kind-returns-400: PUT returns 400 when extraAttributesKinds contains an unknown kind", func() {
			By("Building PUT /documents request with invalid kind")
			reqBody := putDocumentWithMetadata(
				ReadFromTesDatatDir("documents-put-extra-kinds/test-auto-resolved", "DOC-AR.json"),
				&documents.DocumentMetadata{ExtraAttributesKinds: []string{"unknown-kind"}},
			)

			resp, body, err := HttpCall("PUT", endpointURI, nil, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
})

// putDocumentWithMetadata merges a metadata object into a raw PUT /documents JSON body.
// The document JSON is unmarshalled into PutDocumentRequest, the metadata field is replaced,
// and the result is re-marshalled to JSON.
func putDocumentWithMetadata(docJSON string, metadata *documents.DocumentMetadata) string {
	GinkgoHelper()
	var req documents.PutDocumentRequest
	Expect(json.Unmarshal([]byte(docJSON), &req)).To(Succeed())
	req.Metadata = metadata
	out, err := json.Marshal(req)
	Expect(err).To(BeNil())
	return string(out)
}
