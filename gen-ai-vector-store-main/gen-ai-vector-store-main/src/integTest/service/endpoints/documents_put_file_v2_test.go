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

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/embedings"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing PUT SVC /v1/{isolationID}/collections/{collectionName}/file", func() {

	var isolationID string
	var collectionID string
	var testExpectations []string

	_ = Context("accessing endpoint with PUT method (async SC job submission)", func() {
		var endpointURI string
		var docAttrs []attributes.Attribute

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/file", baseURI, isolationID, collectionID)

			docAttrs = []attributes.Attribute{
				{Name: "Document type", Type: "string", Values: []string{"Article"}},
				{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
			}

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(context.TODO(), database, isolationID, "1GB")
		})

		AfterEach(func() {
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
			if !CurrentSpecReport().Failed() {
				RemovedIsolationFromEmbeddingQueue(context.TODO(), database, isolationID)
				DeleteIsolation(opsURI, isolationID)
			}
		})

		It("test 404: return 404 if isolation does not exist", func() {
			badURI := fmt.Sprintf("%s/v1/%s/collections/%s/file", baseURI, "non-existent-iso", collectionID)
			ExpectServiceReturns404IfIsolationDoesNotExist("PUT", badURI)
		})

		It("test0: Put file fails if documentID not provided", func() {
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test0/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test0: Put file fails if documentAttributes not provided", func() {
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test0/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test0: Put file fails if documentAttributes is empty", func() {
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: []attributes.Attribute{}},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test0/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test0: Put file fails if documentFile not provided", func() {
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test0: Put file fails if documentFile contains no data (0 length file)", func() {
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test0/documents/Astronomy-empty.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test0: Put file fails if documentFile contains no data (empty txt file)", func() {
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test0/documents/Astronomy-empty1.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test1: Put .txt file returns 202 with documentID", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Put .txt file")
			metadata := map[string]interface{}{
				"extraAttributesKinds": []string{"auto-resolved"},
			}
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: metadata},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify 202 response body")
			var respBody map[string]interface{}
			err = json.Unmarshal([]byte(body), &respBody)
			Expect(err).To(BeNil())
			Expect(respBody).To(HaveKey("documentID"))
			Expect(respBody["documentID"]).To(ContainSubstring("Astronomy"))
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))
		})

		It("test1: Put .pdf file returns 202 with documentID", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Put .pdf file")
			docAttrs = []attributes.Attribute{
				{Name: "Region", Type: "string", Values: []string{"GALAXY"}},
			}
			metadata := map[string]interface{}{
				"extraAttributesKinds": []string{"auto-resolved"},
			}
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: metadata},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test3/documents/visa-partial.pdf")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify 202 response body")
			var respBody map[string]interface{}
			err = json.Unmarshal([]byte(body), &respBody)
			Expect(err).To(BeNil())
			Expect(respBody).To(HaveKey("documentID"))
			Expect(respBody["documentID"]).To(ContainSubstring("Astronomy"))
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))
		})

		It("test-ocr: Put .pdf file with enableOCR in metadata returns 202", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Put .pdf file with OCR enabled via metadata")
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: map[string]interface{}{
					"enableOCR": true,
				}},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test3/documents/visa-partial.pdf")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify 202 response body")
			var respBody map[string]interface{}
			err = json.Unmarshal([]byte(body), &respBody)
			Expect(err).To(BeNil())
			Expect(respBody).NotTo(HaveKey("operationID"))
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))
		})

		It("test-metadata: Put .pdf file with documentMetadata returns 202", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Put .pdf file with metadata")
			docAttrs = []attributes.Attribute{
				{Name: "Region", Type: "string", Values: []string{"GALAXY"}},
			}
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: map[string]interface{}{
					"extraAttributesKinds": []string{"auto-resolved", "index", "static"},
				}},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.pdf")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify 202 response body")
			var respBody map[string]interface{}
			err = json.Unmarshal([]byte(body), &respBody)
			Expect(err).To(BeNil())
			Expect(respBody).To(HaveKey("documentID"))
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))
		})

		It("test-smart-attr: Put .txt file with enableSmartAttribution and embedSmartAttributes returns 202", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Put .txt file with both smart attribution flags")
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: map[string]interface{}{
					"enableSmartAttribution": true,
					"embedSmartAttributes":   true,
					"embeddingAttributes":    []string{"department"},
				}},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify 202 response body")
			var respBody map[string]interface{}
			err = json.Unmarshal([]byte(body), &respBody)
			Expect(err).To(BeNil())
			Expect(respBody).To(HaveKey("documentID"))
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))
		})

		It("test-smart-attr-only: Put .txt file with enableSmartAttribution only (no embed) returns 202", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Put .txt file with enableSmartAttribution only")
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: map[string]interface{}{
					"enableSmartAttribution": true,
					"extraAttributesKinds":   []string{"auto-resolved"},
				}},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify 202 response body")
			var respBody map[string]interface{}
			err = json.Unmarshal([]byte(body), &respBody)
			Expect(err).To(BeNil())
			Expect(respBody).To(HaveKey("documentID"))
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))
		})

		It("test-sc-down: returns 502 when SC is unavailable", func() {
			By("Creating WireMock expectation for SC returning 500")
			scExpID, err := CreateExpectationSmartChunkingJobError(wiremockManager, isolationID, 500)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Put .txt file")
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "Astronomy"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadGateway))
		})

		It("test-sc-callback-ar-attrs: SC callback with auto-resolved attrs stores them when extraAttributesKinds opts in", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Creating generic ADA embedding mock")
			adaMockID, err := CreateExpectationEmbeddingAdaWithoutIsolationValidation(wiremockManager)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, adaMockID)

			By("PUT /file with extraAttributesKinds=[auto-resolved] — VS submits job to SC and returns 202")
			const docID = "doc-sc-ar"
			metadata := map[string]interface{}{
				"extraAttributesKinds": []string{"auto-resolved"},
			}
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: docID},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: metadata},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			var respBody map[string]interface{}
			err = json.Unmarshal([]byte(body), &respBody)
			Expect(err).To(BeNil())
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))

			By("Simulate SC callback: PUT /documents with auto-resolved attribute and extraAttributesKinds=[auto-resolved]")
			callbackURI := fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=strong",
				baseURI, isolationID, collectionID)
			callbackBody := fmt.Sprintf(`{
				"id": %q,
				"chunks": [{
					"content": "extracted text",
					"attributes": [{"name":"region","type":"string","kind":"auto-resolved","value":["EU"]}]
				}],
				"metadata": {"extraAttributesKinds": ["auto-resolved"]}
			}`, docID)

			callbackResp, callbackRespBody, err := HttpCall("PUT", callbackURI, nil, callbackBody)
			Expect(err).To(BeNil())
			Expect(callbackRespBody).NotTo(BeNil())
			Expect(callbackResp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for document COMPLETED")
			WaitForDocumentStatusInDB(context.TODO(), database, isolationID, collectionID, docID, "COMPLETED")

			By("Verify auto-resolved attribute is stored via attr_ids2")
			ExpectChunksInDatabase(context.TODO(), database, isolationID, collectionID, docID, []embedings.Chunk{
				{
					ID:      docID + "-EMB-0",
					Content: "extracted text",
					Attributes: attributes.Attributes{
						{Name: "region", Type: "string"},
					},
				},
			})
		})

		It("test-sc-callback-ar-attrs-dropped: auto-resolved attrs are dropped when extraAttributesKinds is absent from SC callback", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, scExpID)

			By("Creating generic ADA embedding mock")
			adaMockID, err := CreateExpectationEmbeddingAdaWithoutIsolationValidation(wiremockManager)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, adaMockID)

			By("PUT /file — VS submits job to SC and returns 202")
			const docID = "doc-sc-ar-dropped"
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: docID},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.txt")},
			}

			resp, _, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Simulate SC callback WITHOUT extraAttributesKinds — attribute should be dropped")
			callbackURI := fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=strong",
				baseURI, isolationID, collectionID)
			callbackBody := fmt.Sprintf(`{
				"id": %q,
				"chunks": [{
					"content": "extracted text",
					"attributes": [{"name":"region","type":"string","kind":"auto-resolved","value":["EU"]}]
				}]
			}`, docID)

			callbackResp, callbackRespBody, err := HttpCall("PUT", callbackURI, nil, callbackBody)
			Expect(err).To(BeNil())
			Expect(callbackRespBody).NotTo(BeNil())
			Expect(callbackResp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for document COMPLETED")
			WaitForDocumentStatusInDB(context.TODO(), database, isolationID, collectionID, docID, "COMPLETED")

			By("Verify no attributes are linked via attr_ids2 (auto-resolved was dropped)")
			ExpectChunkHasNoAttributesLinked(context.TODO(), database, isolationID, collectionID, docID+"-EMB-0")
		})

		It("test-invalid-metadata-json-returns-400: PUT /file returns 400 when documentMetadata is invalid JSON", func() {
			mfParts := []MultiformPart{
				{Type: "field", Name: "documentID", Value: "doc-1"},
				{Type: "field", Name: "documentAttributes", Value: docAttrs},
				{Type: "field", Name: "documentMetadata", Value: "{invalid-json"},
				{Type: "file", Name: "documentFile", Value: GetAbsPath("documents-put-file/test1/documents/Astronomy.txt")},
			}

			resp, body, err := HttpCallMultipartForm("PUT", endpointURI, mfParts)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
})
