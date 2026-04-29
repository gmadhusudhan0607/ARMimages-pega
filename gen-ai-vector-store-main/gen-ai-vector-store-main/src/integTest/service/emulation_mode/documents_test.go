//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package emulation_mode_test

import (
	"encoding/json"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/documents"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Documents API in Emulation Mode", func() {
	var testCtx *TestContext
	var httpHelper *HTTPTestHelper
	var validator *ResponseValidator
	var docBuilder *DocumentEndpointBuilder

	BeforeEach(func() {
		testCtx = NewTestContext(svcBaseURI)
		httpHelper = NewHTTPTestHelper(testCtx)
		validator = NewResponseValidator()
		baseURI := testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+DocumentsPath)
		docBuilder = NewDocumentEndpointBuilder(baseURI)
	})

	Context("Documents API - CRUD Operations on /v1/{isolationID}/collections/{collectionID}/documents", func() {
		It("POST /v1/{isolationID}/collections/{collectionID}/documents - List documents in COMPLETED status", func() {
			By("Listing all documents in database")
			uri := testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+DocumentsPath)

			_, body := httpHelper.MakeAPICallWithValidation("POST", uri, "", StatusOK)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			docStatuses := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDocumentStatusArrayResponse(b) },
				"document status array response").([]DocumentStatus)
			Expect(docStatuses).NotTo(BeNil())
		})

		It("GET /v1/{isolationID}/collections/{collectionID}/documents/{documentID} - Get document by ID successfully", func() {
			By("Getting document by ID in emulation mode")
			uri := docBuilder.GetEndpoint(TestDocumentID)

			_, body := httpHelper.MakeAPICallWithValidation("GET", uri, "", StatusOK)

			By("Validating response schema matches service.yaml specification")
			docStatus := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDocumentStatusResponse(b) },
				"document status response").(*DocumentStatus)
			Expect(docStatus.ID).NotTo(BeEmpty())
			Expect(docStatus.Status).NotTo(BeEmpty())
		})

		It("PUT /v1/{isolationID}/collections/{collectionID}/documents - Put document with eventual consistency level", func() {
			By("Putting document in EmulationMode with eventual consistency level")
			uri := BuildQueryParams(
				testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+DocumentsPath),
				map[string]string{"consistencyLevel": indexer.ConsistencyLevelEventual},
			)

			_, body := httpHelper.MakeAPICallWithValidation("PUT", uri, ReadTestDataFile(DocumentsPutRequestFile), StatusAccepted)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			docStatus := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDocumentStatusResponse(b) },
				"document status response").(*DocumentStatus)
			Expect(docStatus.ID).NotTo(BeEmpty())
			Expect(docStatus.Status).NotTo(BeEmpty())
		})

		It("PATCH /v1/{isolationID}/collections/{collectionID}/documents/{documentID} - Patch document successfully", func() {
			By("Patching document in EmulationMode")
			uri := docBuilder.PatchEndpoint(TestDocumentID)

			_, body := httpHelper.MakeAPICallWithValidation("PATCH", uri, ReadTestDataFile(DocumentsPatchRequestFile), StatusOK)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			docStatus := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDocumentStatusResponse(b) },
				"document status response").(*DocumentStatus)
			Expect(docStatus.ID).NotTo(BeEmpty())
			Expect(docStatus.Status).NotTo(BeEmpty())
		})

		It("DELETE /v1/{isolationID}/collections/{collectionID}/documents - Delete documents by attributes", func() {
			By("Deleting documents by attributes in EmulationMode")
			uri := testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+DocumentsPath)

			_, body := httpHelper.MakeAPICallWithValidation("DELETE", uri, ReadTestDataFile(DocumentsDeleteRequestFile), StatusOK)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			deletedDocs := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDeletedDocumentsResponse(b) },
				"deleted documents response").(*DeletedDocuments)
			Expect(deletedDocs).NotTo(BeNil())
		})

		It("POST /v1/{isolationID}/collections/{collectionID}/document/delete-by-id - Delete document by ID using POST method", func() {
			By("Deleting document by ID using POST method")
			uri := testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+DeleteByIDPath)

			_, body := httpHelper.MakeAPICallWithValidation("POST", uri, DeleteByIDPayload, StatusOK)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			deletedDocs := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDeletedDocumentByIdResponse(b) },
				"deleted document by ID response").(*DeletedDocumentByIdRequest)
			Expect(deletedDocs).NotTo(BeNil())
			Expect(deletedDocs.Id).NotTo(BeNil())
		})

		It("DELETE /v1/{isolationID}/collections/{collectionID}/documents/{documentID} - Delete document by ID using DELETE method", func() {
			By("Deleting document by ID using DELETE method")
			uri := docBuilder.DeleteEndpoint(TestDocumentID)

			_, body := httpHelper.MakeAPICallWithValidation("DELETE", uri, "", StatusOK)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			deletedDocs := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDeletedDocumentsResponse(b) },
				"deleted documents response").(*DeletedDocuments)
			Expect(deletedDocs).NotTo(BeNil())
		})
	})

	Context("Documents API - File Upload Operations on /v1/{isolationID}/collections/{collectionID}/file", func() {
		It("PUT /v1/{isolationID}/collections/{collectionID}/file - Upload PDF document file with strong consistency level", func() {
			By("Uploading PDF document file in EmulationMode")
			uri := BuildQueryParams(
				testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+FileUploadPath),
				map[string]string{"consistencyLevel": indexer.ConsistencyLevelStrong},
			)

			docAttrs := []attributes.Attribute{
				{Name: "Region", Type: "string", Values: []string{"GALAXY"}},
			}

			formHelper := NewMultipartFormHelper()
			mfParts := formHelper.BuildDocumentUploadParts("Astronomy", AstronomyPDFFile, docAttrs)

			resp, body, err := HttpCallMultipartFormWithHeadersAndApiCallStat("PUT", uri, mfParts, ServiceRuntimeHeaders)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(StatusCreated))
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			docStatus := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDocumentStatusResponse(b) },
				"document status response").(*DocumentStatus)
			Expect(docStatus.ID).NotTo(BeEmpty())
			Expect(docStatus.Status).NotTo(BeEmpty())
		})

		It("PUT /v1/{isolationID}/collections/{collectionID}/file/text - Upload text document file with strong consistency level", func() {
			By("Uploading text document file in EmulationMode")
			uri := BuildQueryParams(
				testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+FileTextUploadPath),
				map[string]string{"consistencyLevel": indexer.ConsistencyLevelStrong},
			)

			putReq := documents.PutFileTextRequest{
				DocumentID:      "Astronomy",
				DocumentContent: ReadTestDataFile(AstronomyMarkdownFile),
				DocumentAttributes: []attributes.Attribute{
					{Name: "Document type", Type: "string", Values: []string{"Article"}},
					{Name: "Category", Type: "string", Values: []string{"Astronomy", "Science", "Physics"}},
				},
			}
			requestBody, err := json.Marshal(putReq)
			Expect(err).To(BeNil())

			_, body := httpHelper.MakeAPICallWithValidation("PUT", uri, string(requestBody), StatusCreated)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			docStatus := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateDocumentStatusResponse(b) },
				"document status response").(*DocumentStatus)
			Expect(docStatus.ID).NotTo(BeEmpty())
			Expect(docStatus.Status).NotTo(BeEmpty())
		})
	})
})
