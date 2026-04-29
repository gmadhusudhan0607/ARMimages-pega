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
	"time"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing SVC /v1/{isolationID}/collections/{collectionName}/attributes ", func() {

	var isolationID string
	var collectionID string
	var ctx = context.TODO()
	var testExpectations []string

	_ = Context("calling service", func() {
		var endpointURI string
		var jsonData string
		var body []byte
		var resp *http.Response
		var items []Item

		BeforeEach(func() {
			testExpectations = []string{}
			isolationID = strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10)))
			collectionID = strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5)))
			endpointURI = fmt.Sprintf("%s/v1/%s/collections/%s/attributes", baseURI, isolationID, collectionID)

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
			//v.POST("/attributes", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.QueryAttributes)
			ExpectServiceReturns404IfIsolationDoesNotExist("POST", endpointURI)
		})

		It("test01: List requested attributes if ASYNC / COMPLETED documents", func() {
			// Mock ADA using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("attributes/test01/documents", baseURI, isolationID, collectionID)
			By("Waiting for completion")
			for _, docID := range docIDs {
				WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Validate response returns all attributes when retrieveAttributes not set")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(6))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-not-set.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 6)

			By("Validate response returns all attributes when retrieveAttributes empty")
			jsonData = ReadTestDataFile("attributes/test02/query-retrieveAttributes-empty.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(6))
			ExpectResponseMatchFromFile(string(body), "attributes/test02/response-retrieveAttributes-empty.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 6)

			By("Validate response returns 1 attribute")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-1.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-1.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 1)

			By("Validate response returns 2 attribute")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-2.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(2))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-2.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 2)

			By("Validate response returns 3 attribute")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-3.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-3.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 3)

			By("Validate response ignore non-existent attributes")
			jsonData = ReadTestDataFile("attributes/test01/query-retrieveAttributes-not-existent.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			ExpectResponseMatchFromFile(string(body), "attributes/test01/response-retrieveAttributes-not-existent.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 3)
		})

		It("test02: List requested attributes if ASYNC / IN_PROGRESS documents", func() {
			// Mock ADA using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDir("attributes/test02/documents", baseURI, isolationID, collectionID)
			// Wait for background processing to be completed before cleanup to avoid not needed errors in logs
			defer func() {
				By("Waiting for completion")
				for _, docID := range docIDs {
					WaitForDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
				}
			}()

			By("Validate response returns all attributes when retrieveAttributes not set")
			jsonData = ReadTestDataFile("attributes/test02/query-retrieveAttributes-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(6))
			ExpectResponseMatchFromFile(string(body), "attributes/test02/response-retrieveAttributes-not-set.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 6)

			By("Validate response returns all attributes when retrieveAttributes empty")
			jsonData = ReadTestDataFile("attributes/test02/query-retrieveAttributes-empty.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(6))
			ExpectResponseMatchFromFile(string(body), "attributes/test02/response-retrieveAttributes-empty.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 6)

			By("Validate response returns 1 attribute")
			jsonData = ReadTestDataFile("attributes/test02/query-retrieveAttributes-1.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			ExpectResponseMatchFromFile(string(body), "attributes/test02/response-retrieveAttributes-1.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 1)

			By("Validate response returns 2 attribute")
			jsonData = ReadTestDataFile("attributes/test02/query-retrieveAttributes-2.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(2))
			ExpectResponseMatchFromFile(string(body), "attributes/test02/response-retrieveAttributes-2.json")

			By("Validate response returns 3 attribute")
			jsonData = ReadTestDataFile("attributes/test02/query-retrieveAttributes-3.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			ExpectResponseMatchFromFile(string(body), "attributes/test02/response-retrieveAttributes-3.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 3)

			By("Validate response ignore non-existent attributes")
			jsonData = ReadTestDataFile("attributes/test02/query-retrieveAttributes-not-existent.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			ExpectResponseMatchFromFile(string(body), "attributes/test02/response-retrieveAttributes-not-existent.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 3)
		})

		It("test03: List requested attributes if SYNC / COMPLETED documents", func() {
			// Mock ADA using WireMock
			mockID, err := CreateExpectationEmbeddingAda(wiremockManager, isolationID)
			Expect(err).To(BeNil())
			testExpectations = []string{mockID}

			By("Insert documents")
			docIDs := UpsertDocumentsFromDirWithStrongConsistencyLevel("attributes/test02/documents", baseURI, isolationID, collectionID)
			// TODO: remove sleep after US-608485
			time.Sleep(time.Second * 3)
			for _, docID := range docIDs {
				ExpectDocumentStatusInDB(context.Background(), database, isolationID, collectionID, docID, "COMPLETED")
			}

			By("Validate response returns all attributes when retrieveAttributes not set")
			jsonData = ReadTestDataFile("attributes/test03/query-retrieveAttributes-not-set.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(6))
			ExpectResponseMatchFromFile(string(body), "attributes/test03/response-retrieveAttributes-not-set.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 6)

			By("Validate response returns all attributes when retrieveAttributes empty")
			jsonData = ReadTestDataFile("attributes/test03/query-retrieveAttributes-empty.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(6))
			ExpectResponseMatchFromFile(string(body), "attributes/test03/response-retrieveAttributes-empty.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 6)

			By("Validate response returns 1 attribute")
			jsonData = ReadTestDataFile("attributes/test03/query-retrieveAttributes-1.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(1))
			ExpectResponseMatchFromFile(string(body), "attributes/test03/response-retrieveAttributes-1.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 1)

			By("Validate response returns 2 attribute")
			jsonData = ReadTestDataFile("attributes/test03/query-retrieveAttributes-2.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(2))
			ExpectResponseMatchFromFile(string(body), "attributes/test03/response-retrieveAttributes-2.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 2)

			By("Validate response returns 3 attribute")
			jsonData = ReadTestDataFile("attributes/test03/query-retrieveAttributes-3.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			ExpectResponseMatchFromFile(string(body), "attributes/test03/response-retrieveAttributes-3.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 3)

			By("Validate response ignore non-existent attributes")
			jsonData = ReadTestDataFile("attributes/test03/query-retrieveAttributes-not-existent.json")
			resp, body, err = HttpCall("POST", endpointURI, nil, jsonData)
			Expect(err).To(BeNil())
			Expect(body).NotTo(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			err = json.Unmarshal(body, &items)
			Expect(err).To(BeNil())
			Expect(len(items)).To(Equal(3))
			ExpectResponseMatchFromFile(string(body), "attributes/test03/response-retrieveAttributes-not-existent.json")
			ExpectHeadersCommon(resp)
			ExpectHeadersDatabase(resp)
			ExpectHeadersItemsCount(resp, 3)
		})
	})
})
