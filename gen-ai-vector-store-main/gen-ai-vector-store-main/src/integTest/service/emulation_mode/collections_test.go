//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package emulation_mode_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Collections API in Emulation Mode", func() {
	var testCtx *TestContext
	var httpHelper *HTTPTestHelper
	var validator *ResponseValidator

	BeforeEach(func() {
		testCtx = NewTestContext(svcBaseURI)
		httpHelper = NewHTTPTestHelper(testCtx)
		validator = NewResponseValidator()
	})

	Context("Collections API - CRUD Operations on /v2/{isolationID}/collections", func() {
		It("POST /v2/{isolationID}/collections - Create collection successfully", func() {
			By("Creating collection in EmulationMode")
			uri := testCtx.BuildEndpointURI(APIv2Path, CollectionsPath)
			reqBody := CollectionRequestBody(testCtx.CollectionID)

			_, body := httpHelper.MakeAPICallWithValidation("POST", uri, reqBody, StatusAccepted)
			By(fmt.Sprintf(" -> Response: %s", body))

			By("Validating response schema matches service.yaml specification")
			collection := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateCollectionResponse(b) },
				"collection creation response").(*Collection)
			Expect(collection.CollectionID).NotTo(BeEmpty())
		})

		It("DELETE /v2/{isolationID}/collections/{collectionID} - Delete collection by URL-encoded ID", func() {
			By("Deleting collection in EmulationMode")
			uri := testCtx.BuildEndpointURI(APIv2Path, CollectionsPath, testCtx.CollectionID)

			_, body := httpHelper.MakeAPICallWithValidation("DELETE", uri, "", StatusOK)
			By(fmt.Sprintf(" -> Response: %s", body))

			By("Validating empty response")
			err := ValidateEmptyResponse(body)
			Expect(err).To(BeNil())
		})

		It("GET /v2/{isolationID}/collections - List all collections successfully", func() {
			By("Getting collections list in EmulationMode")
			uri := testCtx.BuildEndpointURI(APIv2Path, CollectionsPath)

			_, body := httpHelper.MakeAPICallWithValidation("GET", uri, "", StatusOK)

			By("Validating response schema matches service.yaml specification")
			collections := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateCollectionsListResponse(b) },
				"collections list response").(*Collections)
			Expect(collections.Collections).NotTo(BeNil())
		})

		It("GET /v2/{isolationID}/collections/{collectionID} - Get single collection successfully", func() {
			By("Getting single collection in EmulationMode")
			uri := testCtx.BuildEndpointURI(APIv2Path, CollectionsPath, testCtx.CollectionID)

			_, body := httpHelper.MakeAPICallWithValidation("GET", uri, "", StatusOK)

			By("Validating response schema matches service.yaml specification")
			collection := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateCollectionResponse(b) },
				"single collection response").(*Collection)
			Expect(collection.CollectionID).NotTo(BeEmpty())
		})
	})

	Context("Collections API - Document Operations on /v2/{isolationID}/collections/{collectionID}", func() {
		It("POST /v2/{isolationID}/collections/{collectionID}/find-documents - Find documents successfully (ASYNC)", func() {
			By("Finding documents in EmulationMode")
			uri := testCtx.BuildEndpointURI(APIv2Path, CollectionsPath+"/"+testCtx.CollectionID+FindDocumentsPath)

			_, body := httpHelper.MakeAPICallWithValidation("POST", uri, FindDocumentsPayload, StatusOK)
			By(fmt.Sprintf(" -> Response: %s", body))

			By("Validating response schema matches service.yaml specification")
			findDocsResp := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateFindDocumentsResponse(b) },
				"find documents response").(*FindDocumentsResponse)
			Expect(findDocsResp.Documents).NotTo(BeNil())
		})

		It("GET /v2/{isolationID}/collections/{collectionID}/documents/{documentID}/chunks - Get document chunks successfully", func() {
			By("Getting document chunks in EmulationMode")
			uri := testCtx.BuildEndpointURI(APIv2Path, CollectionsPath+"/"+testCtx.CollectionID+DocumentsPath, TestChunkID, "chunks")

			_, body := httpHelper.MakeAPICallWithValidation("GET", uri, "", StatusOK)
			By(fmt.Sprintf(" -> Response: %s", body))

			By("Validating response schema matches service.yaml specification")
			docChunks := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDocumentChunksResponse(b) },
				"document chunks response").(*DocumentChunks)
			Expect(docChunks.DocumentID).NotTo(BeEmpty())
			Expect(docChunks.Chunks).NotTo(BeEmpty())
			Expect(docChunks.Chunks[0].ID).NotTo(BeEmpty())
			Expect(docChunks.Chunks[0].Content).NotTo(BeEmpty())
			Expect(docChunks.Chunks[0].Attributes).NotTo(BeEmpty())
		})
	})
})
