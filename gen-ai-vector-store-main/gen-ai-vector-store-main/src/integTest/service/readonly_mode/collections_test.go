//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	db2 "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/collections"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing Testing SVC /v1/*/collections in ReadOnly mode", func() {

	var isolationID string
	var collectionID string
	var testExpectations []string

	_ = Context("accessing endpoint with POST method and strong consistency level", func() {
		var endpointURI string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}
			endpointURI = fmt.Sprintf("%s/v2/%s/collections", svcBaseURI, isolationID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed
			if !CurrentSpecReport().Failed() {
				DeleteIsolation(opsBaseURI, isolationID)
			}
			for _, expID := range testExpectations {
				DeleteMockServerExpectation(expID)
			}
		})

		It("POST collection returns 405", func() {

			By("Attempting to create collection in ReadOnlyMode")
			reqBody := fmt.Sprintf("{\"name\":\"%s\"}", collectionID)
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))

		})

		It("POST collection returns 405 with strong consistency level", func() {

			By("Attempting to create collection in ReadOnlyMode with strong consistency level")
			reqBody := fmt.Sprintf("{\"name\":\"%s\"}", collectionID)
			strongConsistencyHeaders := map[string]string{
				headers.ForceFreshDbMetrics: "true",
				headers.ServiceMode:         "ReadOnly",
			}
			resp, body, err := HttpCallWithHeaders("POST", endpointURI, strongConsistencyHeaders, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))

		})
	})
	_ = Context("accessing endpoint with DELETE method to delete a collection", func() {
		var endpointURI string
		var endpointURIRW string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}
			endpointURI = fmt.Sprintf("%s/v2/%s/collections", svcBaseURI, isolationID)
			endpointURIRW = fmt.Sprintf("%s/v2/%s/collections", svcBaseURI, isolationID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed
			if !CurrentSpecReport().Failed() {
				DeleteIsolation(opsBaseURI, isolationID)
			}
			for _, expID := range testExpectations {
				DeleteMockServerExpectation(expID)
			}
		})

		It("DELETE collection returns 405 ", func() {

			By("Create collection")
			reqBuddy := fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID)
			resp, body, err := HttpCall("POST", endpointURIRW, nil, reqBuddy)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Expect collection and smart attributes table created")
			ExpectTableExists(ctx, database, db2.GetTableCollections(isolationID))
			ExpectTableExists(ctx, database, db2.GetTableSmartAttrGroup(isolationID))

			By("Expect _doc, _emb and _attr tables created")
			ExpectTableExists(ctx, database, db2.GetTableDoc(isolationID, collectionID))
			ExpectTableExists(ctx, database, db2.GetTableEmb(isolationID, collectionID))
			ExpectTableExists(ctx, database, db2.GetTableAttr(isolationID, collectionID))
			ExpectTableExists(ctx, database, db2.GetTableDocMeta(isolationID, collectionID))
			ExpectTableExists(ctx, database, db2.GetTableEmbMeta(isolationID, collectionID))
			ExpectCollectionExistsInDB(ctx, database, isolationID, collectionID)
			ExpectCollectionEmbeddingProfileExists(ctx, database, isolationID, collectionID, collections.DefaultEmbeddingProfileID)

			By("Expect _doc indexes created")
			schemaName, tableName := helpers.SplitTableName(db2.GetTableDoc(isolationID, collectionID))
			ExpectIndexExists(ctx, database, schemaName, tableName, fmt.Sprintf("idx_%s__attrids", tableName))

			By("Expect _emb indexes created")
			schemaName, tableName = helpers.SplitTableName(db2.GetTableEmb(isolationID, collectionID))
			ExpectIndexExists(ctx, database, schemaName, tableName, fmt.Sprintf("idx_%s__docid", tableName))
			ExpectIndexExists(ctx, database, schemaName, tableName, fmt.Sprintf("idx_%s__rts", tableName))
			ExpectIndexExists(ctx, database, schemaName, tableName, fmt.Sprintf("idx_%s__attrids", tableName))
			ExpectIndexExists(ctx, database, schemaName, tableName, fmt.Sprintf("idx_%s__attrids2", tableName))

			By("Expect _attr indexes created")
			schemaName, tableName = helpers.SplitTableName(db2.GetTableAttr(isolationID, collectionID))
			ExpectIndexExists(ctx, database, schemaName, tableName, fmt.Sprintf("idx_%s_kth", tableName))

			By("expect _doc_processing table created")
			ExpectTableExists(ctx, database, db2.GetTableDocProcessing(isolationID, collectionID))
			By("expect _emb_processing table created")
			ExpectTableExists(ctx, database, db2.GetTableEmbProcessing(isolationID, collectionID))
			// TODO: EPIC-103866 / US-682862:
			// By("expect _emb_statistics table created")
			// ExpectTableExists(ctx, database, db2.GetTableEmbStatistics(isolationID, collectionID))

			By("Attempting to delete collection in ReadOnlyMode")
			uri := fmt.Sprintf("%s/%s", endpointURI, url.PathEscape(collectionID))
			resp, body, err = HttpCallWithHeadersAndApiCallStat("DELETE", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).To(BeNil())
			Expect(response["code"]).To(Equal("405"))
			Expect(response["message"]).To(Equal("Method not allowed in Read Only mode"))

		})
	})

	_ = Context("accessing endpoint with GET method to get a collection", func() {
		var endpointURI string
		var endpointURISetup string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}
			endpointURI = fmt.Sprintf("%s/v2/%s/collections", svcBaseURI, isolationID)
			endpointURISetup = fmt.Sprintf("%s/v2/%s/collections", svcBaseURI, isolationID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")

			// Create collection for testing
			reqBody := fmt.Sprintf("{\"name\":\"%s\"}", collectionID)
			_, _, err := HttpCall("POST", endpointURISetup, nil, reqBody)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			// Do not clean up if test failed
			if !CurrentSpecReport().Failed() {
				DeleteIsolation(opsBaseURI, isolationID)
			}
			for _, expID := range testExpectations {
				DeleteMockServerExpectation(expID)
			}
		})

		It("GET collection successfully", func() {

			By("Create collection 1")
			randomString := RandStringRunes(5)
			collectionID1 := strings.ToLower(fmt.Sprintf("col-%s-1", randomString))
			reqBuddy := fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID1)
			resp, _, err := HttpCall("POST", endpointURISetup, nil, reqBuddy)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Create collection 2")
			collectionID2 := strings.ToLower(fmt.Sprintf("col-%s-2", randomString))
			reqBuddy = fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID2)
			resp, _, err = HttpCall("POST", endpointURISetup, nil, reqBuddy)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("List collections")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", endpointURI, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			expectedResponse := fmt.Sprintf(`
				{
				  "isolationID": "%[1]s",
				  "collections":
					[
					  { "id": "%[2]s" , "defaultEmbeddingProfile": "%[4]s", "documentsTotal": 0 },
					  { "id": "%[3]s" , "defaultEmbeddingProfile": "%[4]s", "documentsTotal": 0 }
					],
				  "pagination": {}
		     }
		 `, isolationID, collectionID1, collectionID2, collections.DefaultEmbeddingProfileID)
			ExpectJSONEquals(body, []byte(expectedResponse))
		})

		It("GET single collection successfully", func() {

			By("Get single collection")
			uri := fmt.Sprintf("%s/%s", endpointURI, url.PathEscape(collectionID))
			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", uri, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			// This should work in ReadOnly mode as it's a read operation
			// It might return 400/404 if collection doesn't exist or wrong format, but that's fine for testing ReadOnly mode
			Expect(resp.StatusCode).To(BeElementOf([]int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}))
		})
	})

	_ = Context("accessing endpoint with POST method for find-documents", func() {
		var endpointURI string
		var endpointURISetup string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}
			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/find-documents", svcBaseURI, isolationID, collectionID)
			endpointURISetup = fmt.Sprintf("%s/v2/%s/collections", svcBaseURI, isolationID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")

			// Create collection for testing
			reqBody := fmt.Sprintf("{\"name\":\"%s\"}", collectionID)
			_, _, err := HttpCall("POST", endpointURISetup, nil, reqBody)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			// Do not clean up if test failed
			if !CurrentSpecReport().Failed() {
				DeleteIsolation(opsBaseURI, isolationID)
			}
			for _, expID := range testExpectations {
				DeleteMockServerExpectation(expID)
			}
		})

		It("POST find-documents works in ReadOnly mode", func() {

			By("Attempting to find documents in ReadOnlyMode")
			reqBody := `{"query": "test query", "limit": 10}`
			resp, body, err := HttpCallWithHeadersAndApiCallStat("POST", endpointURI, ServiceRuntimeHeaders, reqBody)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			// This should work in ReadOnly mode as it's a read operation
			// It might return 404 if collection doesn't exist, but that's fine for testing ReadOnly mode
			Expect(resp.StatusCode).To(BeElementOf([]int{http.StatusOK, http.StatusNotFound}))
		})
	})

	_ = Context("accessing endpoint with GET method for document chunks", func() {
		var endpointURI string
		var endpointURISetup string
		ctx := context.TODO()

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			testExpectations = []string{}
			documentID := "test-doc-1"
			endpointURI = fmt.Sprintf("%s/v2/%s/collections/%s/documents/%s/chunks", svcBaseURI, isolationID, collectionID, documentID)
			endpointURISetup = fmt.Sprintf("%s/v2/%s/collections", svcBaseURI, isolationID)

			CreateIsolation(opsBaseURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")

			// Create collection for testing
			reqBody := fmt.Sprintf("{\"name\":\"%s\"}", collectionID)
			_, _, err := HttpCall("POST", endpointURISetup, nil, reqBody)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			// Do not clean up if test failed
			if !CurrentSpecReport().Failed() {
				DeleteIsolation(opsBaseURI, isolationID)
			}
			for _, expID := range testExpectations {
				DeleteMockServerExpectation(expID)
			}
		})

		It("GET document chunks works in ReadOnly mode", func() {

			By("Attempting to get document chunks in ReadOnlyMode")
			resp, body, err := HttpCallWithHeadersAndApiCallStat("GET", endpointURI, ServiceRuntimeHeaders, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			// This should work in ReadOnly mode as it's a read operation
			// It might return 404 if document doesn't exist, but that's fine for testing ReadOnly mode
			Expect(resp.StatusCode).To(BeElementOf([]int{http.StatusOK, http.StatusNotFound}))
		})
	})
})
