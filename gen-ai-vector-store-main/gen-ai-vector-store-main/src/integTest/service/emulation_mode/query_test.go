//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package emulation_mode_test

import (
	"fmt"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Query API in Emulation Mode", func() {
	var testCtx *TestContext
	var httpHelper *HTTPTestHelper
	var validator *ResponseValidator

	BeforeEach(func() {
		testCtx = NewTestContext(svcBaseURI)
		httpHelper = NewHTTPTestHelper(testCtx)
		validator = NewResponseValidator()
	})

	Context("Query API - Query Operations on /v1/{isolationID}/collections/{collectionID}/query", func() {
		It("POST /v1/{isolationID}/collections/{collectionID}/query/chunks - Query chunks with limit parameter", func() {
			By("Querying chunks with emulated result")
			uri := testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+QueryChunksPath)
			jsonData := ReadTestDataFile(QueryChunksRequestFile)

			_, body := httpHelper.MakeAPICallWithValidation("POST", uri, jsonData, StatusOK)
			By(fmt.Sprintf(" -> Response: %s", body))

			By("Validating response schema matches service.yaml specification")
			queryChunks := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateQueryChunksResponse(b) },
				"query chunks response").([]QueryChunksResponse)
			Expect(queryChunks).NotTo(BeNil())
		})

		It("POST /v1/{isolationID}/collections/{collectionID}/query/documents - Query documents with limit parameter", func() {
			By("Querying documents with emulated result")
			uri := testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+QueryDocumentsPath)
			jsonData := ReadTestDataFile(QueryDocumentsRequestFile)

			_, body := httpHelper.MakeAPICallWithValidation("POST", uri, jsonData, StatusAccepted)
			By(fmt.Sprintf(" -> Response: %s", body))

			By("Validating response schema matches service.yaml specification")
			queryDocs := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateQueryDocumentsResponse(b) },
				"query documents response").([]QueryDocumentsResponse)
			Expect(queryDocs).NotTo(BeNil())
		})
	})

	Context("Query API - Attributes Operations on /v1/{isolationID}/collections/{collectionID}/attributes", func() {
		It("POST /v1/{isolationID}/collections/{collectionID}/attributes - List attributes using file-based request", func() {
			By("Listing attributes in EmulationMode using test data file")
			uri := testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+AttributesPath)
			jsonData := ReadTestDataFile(QueryAttributesRequestFile)

			_, body := httpHelper.MakeAPICallWithValidation("POST", uri, jsonData, StatusOK)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			attributes := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateAttributesResponse(b) },
				"attributes response").([]Attribute)
			Expect(attributes).NotTo(BeNil())
		})

		It("POST /v1/{isolationID}/collections/{collectionID}/attributes - List attributes using inline request", func() {
			By("Listing attributes in EmulationMode using inline payload")
			uri := testCtx.BuildEndpointURI(APIv1Path, CollectionsPath+"/"+testCtx.CollectionID+AttributesPath)

			_, body := httpHelper.MakeAPICallWithValidation("POST", uri, AttributesPayload, StatusOK)
			By(fmt.Sprintf(" -> Response body: %s", body))

			By("Validating response schema matches service.yaml specification")
			attributes := validator.ValidateWithSchema(body,
				func(b []byte) (interface{}, error) { return ValidateAttributesResponse(b) },
				"attributes response").([]Attribute)
			Expect(attributes).NotTo(BeNil())
		})
	})
})
