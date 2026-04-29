//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package endpoints_test

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	db2 "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/indexer"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/collections"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing collections", func() {

	var isolationID string
	var collectionID string
	var testExpectations []string
	print(collectionID)

	_ = Context("accessing endpoint", func() {
		var endpointURI string
		ctx := context.TODO()
		print(endpointURI)

		BeforeEach(func() {
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col # %d", rand.Intn(1000))) // with special character
			testExpectations = []string{}
			endpointURI = fmt.Sprintf("%s/v2/%s/collections", baseURI, isolationID)

			CreateIsolation(opsURI, isolationID, "1GB")
			ExpectIsolationExistsInDbWithMaxStorageSize(ctx, database, isolationID, "1GB")
		})

		AfterEach(func() {
			// Do not clean up if test failed
			if !CurrentSpecReport().Failed() {
				DeleteIsolation(opsURI, isolationID)
			}
			for _, expID := range testExpectations {
				err := DeleteExpectationIfExist(wiremockManager, expID)
				Expect(err).To(BeNil())
			}
		})

		It("test 404: return 404 if isolation does not exist", func() {

			// v2.GET("/collections", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), apiV2.GetCollections)
			ExpectServiceReturns404IfIsolationDoesNotExist("GET", endpointURI)

			// v2.GET("/collections/:collectionID", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), apiV2.GetCollection)
			ExpectServiceReturns404IfIsolationDoesNotExist("GET", fmt.Sprintf("%s/%s", endpointURI, "collection-1"))

			// v2.POST("/collections", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), apiV2.PostCollection)
			ExpectServiceReturns404IfIsolationDoesNotExist("POST", endpointURI)

			// v2.DELETE("/collections/:collectionID", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), apiV2.DeleteCollection)
			ExpectServiceReturns404IfIsolationDoesNotExist("DELETE", fmt.Sprintf("%s/%s", endpointURI, "collection-1"))

		})

		It("test1: expect collection automatically created when data injected using API v1", func() {
			// collectionID  must not contain special characters if using V1 API for post document.
			// Create V1 compatible collectionID
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))

			// Mock ADA using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document")
			uri := fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID, indexer.ConsistencyLevelStrong)
			resp, body, err := HttpCall("PUT", uri, nil, ReadTestDataFile("collections/test1/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			ExpectTableExists(ctx, database, db2.GetTableEmbeddingProfiles(isolationID))
			ExpectTableExists(ctx, database, db2.GetTableCollectionEmbeddingProfiles(isolationID))
			ExpectTableExists(ctx, database, db2.GetTableCollections(isolationID))
			ExpectTableExists(ctx, database, db2.GetTableSmartAttrGroup(isolationID))
			ExpectTableExists(ctx, database, db2.GetTableEmb(isolationID, collectionID))
			ExpectTableExists(ctx, database, db2.GetTableAttr(isolationID, collectionID))
			ExpectTableExists(ctx, database, db2.GetTableDocMeta(isolationID, collectionID))
			ExpectTableExists(ctx, database, db2.GetTableEmbMeta(isolationID, collectionID))
			ExpectCollectionExistsInDB(ctx, database, isolationID, collectionID)
			ExpectCollectionEmbeddingProfileExists(ctx, database, isolationID, collectionID, collections.DefaultEmbeddingProfileID)

		})

		It("test2: expect collection created and deleted successfully", func() {

			By("Create collection")
			reqBuddy := fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID)
			resp, body, err := HttpCall("POST", endpointURI, nil, reqBuddy)
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

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)

			By("Delete collection")
			uri := fmt.Sprintf("%s/%s", endpointURI, url.PathEscape(collectionID))
			resp, body, err = HttpCall("DELETE", uri, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			ExpectTableDoesNotExist(ctx, database, db2.GetTableDoc(isolationID, collectionID))
			ExpectTableDoesNotExist(ctx, database, db2.GetTableEmb(isolationID, collectionID))
			ExpectTableDoesNotExist(ctx, database, db2.GetTableAttr(isolationID, collectionID))
			ExpectTableDoesNotExist(ctx, database, db2.GetTableDocMeta(isolationID, collectionID))
			ExpectTableDoesNotExist(ctx, database, db2.GetTableEmbMeta(isolationID, collectionID))
			ExpectCollectionEmbeddingProfileDoesNotExist(ctx, database, isolationID, collectionID, collections.DefaultEmbeddingProfileID)

			ExpectTableDoesNotExist(ctx, database, db2.GetTableDocProcessing(isolationID, collectionID))
			ExpectTableDoesNotExist(ctx, database, db2.GetTableEmbProcessing(isolationID, collectionID))
			// TODO: EPIC-103866 / US-682862:
			// ExpectTableDoesNotExist(ctx, database, db2.GetTableEmbStatistics(isolationID, collectionID))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
		})

		It("test3: expect collection not fails when deleting not existent collection", func() {

			ExpectTableDoesNotExist(ctx, database, db2.GetTableDoc(isolationID, collectionID))
			ExpectTableDoesNotExist(ctx, database, db2.GetTableEmb(isolationID, collectionID))
			ExpectTableDoesNotExist(ctx, database, db2.GetTableAttr(isolationID, collectionID))

			By("Delete collection")
			uri := fmt.Sprintf("%s/%s", endpointURI, url.PathEscape(collectionID))
			resp, body, err := HttpCall("DELETE", uri, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

		})

		It("test4: expect list collection successfully", func() {

			By("Create collection 1")
			randomString := RandStringRunes(5)
			collectionID1 := strings.ToLower(fmt.Sprintf("col-%s-1", randomString))
			reqBuddy := fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID1)
			resp, _, err := HttpCall("POST", endpointURI, nil, reqBuddy)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Create collection 2")
			collectionID2 := strings.ToLower(fmt.Sprintf("col-%s-2", randomString))
			reqBuddy = fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID2)
			resp, _, err = HttpCall("POST", endpointURI, nil, reqBuddy)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("List collections")
			resp, body, err := HttpCall("GET", endpointURI, nil, "")
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

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 2)
		})

		It("test5: expect get collection successfully", func() {

			By("Create collection")
			reqBuddy := fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID)
			resp, body, err := HttpCall("POST", endpointURI, nil, reqBuddy)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Get collections")
			uri := fmt.Sprintf("%s/%s", endpointURI, url.PathEscape(collectionID))
			resp, body, err = HttpCall("GET", uri, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			expectedResp := fmt.Sprintf(`
				{ 
                  "id":"%[1]s",
                  "defaultEmbeddingProfile":"%[2]s",
				  "documentsTotal": 0
                }
            `, collectionID, collections.DefaultEmbeddingProfileID)
			ExpectJSONEquals(body, []byte(expectedResp))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 1)
		})

		It("test6: expect get collections successfully with 1 document in total", func() {

			By("Create collection 1")
			randomString := RandStringRunes(5)
			collectionID1 := strings.ToLower(fmt.Sprintf("col-%s-1", randomString))
			reqBuddy := fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID1)
			resp, _, err := HttpCall("POST", endpointURI, nil, reqBuddy)
			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Create collection 2")
			collectionID2 := strings.ToLower(fmt.Sprintf("col-%s-2", randomString))
			reqBuddy = fmt.Sprintf("{\"collectionID\":\"%s\"}", collectionID2)
			resp, _, err = HttpCall("POST", endpointURI, nil, reqBuddy)
			Expect(err).To(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			// Mock ADA using WireMock
			putEndpointURI := fmt.Sprintf("%s/v1/%s/collections/%s/documents?consistencyLevel=%s", baseURI, isolationID, collectionID1, indexer.ConsistencyLevelStrong)
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = append(testExpectations, mockID)

			By("Put document for strong consistency level")
			resp, body, err := HttpCall("PUT", putEndpointURI, nil, ReadTestDataFile("documents-put/test1/DOC-1.json"))
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			By("Wait for completion")
			WaitForDocumentStatusInDB(ctx, database, isolationID, collectionID1, "DOC-1", "COMPLETED")

			By("List collections")
			resp, body, err = HttpCall("GET", endpointURI, nil, "")
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			expectedResponse := fmt.Sprintf(`
				{
				  "isolationID": "%[1]s",
				  "collections":
					[
					  { "id": "%[2]s" , "defaultEmbeddingProfile": "%[4]s", "documentsTotal": 1 },
					  { "id": "%[3]s" , "defaultEmbeddingProfile": "%[4]s", "documentsTotal": 0 }
					],
				  "pagination": {}
		     }
		 `, isolationID, collectionID1, collectionID2, collections.DefaultEmbeddingProfileID)
			ExpectJSONEquals(body, []byte(expectedResponse))

			By("Check headers in the response")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 2)
		})
	})

})
