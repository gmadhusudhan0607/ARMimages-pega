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
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing PUT SVC /v1/{isolationID}/collections/{collectionName}/file/text", func() {

	var isolationID string
	var collectionID string
	var smartChunkingExpectations []string

	_ = Context("accessing endpoint with PUT method (async SC job submission)", func() {
		var endpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			smartChunkingExpectations = []string{}

			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/file/text", baseURI, isolationID, collectionID)

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
			for _, expID := range smartChunkingExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("test 404: return 404 if isolation does not exist", func() {
			ExpectServiceReturns404IfIsolationDoesNotExist("PUT", endpointURI)
		})

		It("test0: Put file/text fail if documentID not provided", func() {
			By("Put raw file with strong consistency level")
			docAttrs := []attributes.Attribute{
				{Name: "Document type", Type: "string", Values: []string{"Article"}},
				{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
			}
			putReq := documents.PutFileTextRequest{
				DocumentID:         "",
				DocumentAttributes: docAttrs,
				DocumentContent:    ReadTestDataFile("documents-put-file/test0/documents/Astronomy.txt"),
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCall("PUT", endpointURI, nil, string(requestBody))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		})

		It("test0: Put file/text fail if documentAttributes not provided", func() {
			By("Put with strong consistency level")
			putReq := documents.PutFileTextRequest{
				DocumentID:      "DOC-1",
				DocumentContent: ReadTestDataFile("documents-put-file/test0/documents/Astronomy.txt"),
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCall("PUT", endpointURI, nil, string(requestBody))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		})

		It("test0: Put file/text fail if documentAttributes is empty", func() {
			By("Put with strong consistency level")
			putReq := documents.PutFileTextRequest{
				DocumentID:         "DOC-1",
				DocumentAttributes: []attributes.Attribute{},
				DocumentContent:    ReadTestDataFile("documents-put-file/test0/documents/Astronomy.txt"),
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCall("PUT", endpointURI, nil, string(requestBody))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		})

		It("test0: Put file/text fail if DocumentContent not provided", func() {
			docAttrs := []attributes.Attribute{
				{Name: "Document type", Type: "string", Values: []string{"Article"}},
				{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
			}

			By("Put with strong consistency level")
			putReq := documents.PutFileTextRequest{
				DocumentID:         "DOC-1",
				DocumentAttributes: docAttrs,
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCall("PUT", endpointURI, nil, string(requestBody))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		})

		It("test0: Put file/text fail if DocumentContent is empty", func() {
			docAttrs := []attributes.Attribute{
				{Name: "Document type", Type: "string", Values: []string{"Article"}},
				{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
			}

			By("Put with strong consistency level")
			putReq := documents.PutFileTextRequest{
				DocumentID:         "DOC-1",
				DocumentAttributes: docAttrs,
				DocumentContent:    "",
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCall("PUT", endpointURI, nil, string(requestBody))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

		})

		It("test5: Put file/text returns 202 with documentID", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			smartChunkingExpectations = append(smartChunkingExpectations, scExpID)

			By("Put raw file text")
			docAttrs := []attributes.Attribute{
				{Name: "Document type", Type: "string", Values: []string{"Article"}},
				{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
			}
			putReq := documents.PutFileTextRequest{
				DocumentID:         "astronomy-txt",
				DocumentContent:    ReadTestDataFile("documents-put-file/test5/documents/Astronomy.txt"),
				DocumentAttributes: docAttrs,
				DocumentMetadata: &documents.DocumentMetadata{
					ExtraAttributesKinds: []string{"auto-resolved", "index"},
				},
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCall("PUT", endpointURI, nil, string(requestBody))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Verify 202 response body")
			var respBody map[string]interface{}
			err = json.Unmarshal([]byte(body), &respBody)
			Expect(err).To(BeNil())
			Expect(respBody).To(HaveKey("documentID"))
			Expect(respBody["documentID"]).To(ContainSubstring("astronomy-txt"))
			Expect(respBody["status"]).To(Equal("IN_PROGRESS"))
		})

		It("test_extraAttributesKinds: Put file/text with extraAttributesKinds returns 202", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			smartChunkingExpectations = []string{scExpID}

			By("Put raw file text with extraAttributesKinds")
			docAttrs := []attributes.Attribute{
				{Name: "Document type", Type: "string", Values: []string{"Article"}},
				{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
			}
			putReq := documents.PutFileTextRequest{
				DocumentID:         "astronomy-txt",
				DocumentContent:    ReadTestDataFile("documents-put-file/test7/documents/Astronomy.txt"),
				DocumentAttributes: docAttrs,
				DocumentMetadata: &documents.DocumentMetadata{
					ExtraAttributesKinds: []string{"auto-resolved", "index"},
				},
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCall("PUT", endpointURI, nil, string(requestBody))
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

		It("test-invalid-metadata-json-returns-400: PUT /file/text returns 400 when documentMetadata is not a valid JSON object", func() {
			body := `{
				"documentID": "doc-1",
				"documentAttributes": [{"name":"type","type":"string","value":["article"]}],
				"documentContent": "some content",
				"documentMetadata": "this-is-not-an-object"
			}`

			resp, respBody, err := HttpCall("PUT", endpointURI, nil, body)
			Expect(err).To(BeNil())
			Expect(respBody).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("test9: Put file/text returns expected headers on 202 response", func() {
			By("Creating WireMock expectation for SC job submission")
			scExpID, _, err := CreateExpectationSmartChunkingJob(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			smartChunkingExpectations = []string{scExpID}

			By("Put raw file text")
			docAttrs := []attributes.Attribute{
				{Name: "Document type", Type: "string", Values: []string{"Article"}},
				{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
			}
			putReq := documents.PutFileTextRequest{
				DocumentID:         "astronomy-txt",
				DocumentContent:    ReadTestDataFile("documents-put-file/test9/documents/Astronomy.txt"),
				DocumentAttributes: docAttrs,
				DocumentMetadata: &documents.DocumentMetadata{
					ExtraAttributesKinds: []string{"auto-resolved", "index"},
				},
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			resp, body, err := HttpCall("PUT", endpointURI, nil, string(requestBody))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
		})
	})
})
