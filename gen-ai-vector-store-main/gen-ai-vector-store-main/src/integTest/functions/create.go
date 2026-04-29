/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	attributesgroup "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/attributes_group"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/tools"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type ExpectationResponse struct {
	ID string `json:"id"`
}

var genaiUrl = GetEnvOfDefault("GENAI_GATEWAY_SERVICE_URL", "Http://localhost:11080")

// CreateGenericEmbeddingMock creates a generic embedding mock expectation without headers or body matchers
// This is useful when you want the mock to match any request to the embedding endpoint
func CreateGenericEmbeddingMock() string {
	GinkgoHelper()
	expTpl := ReadTestDataFile("request-timeout/exp_Lab API-EMB-0.json")
	var expData map[string]interface{}
	err := json.Unmarshal([]byte(expTpl), &expData)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal generic embedding mock: %v", err))
	}

	// Remove the body matcher and headers so it matches ANY request
	if httpReq, ok := expData["httpRequest"].(map[string]interface{}); ok {
		delete(httpReq, "body")
		delete(httpReq, "headers")
	}

	modifiedExp, err := json.Marshal(expData)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal generic embedding mock: %v", err))
	}

	return string(modifiedExp)
}

func CreateMockServerExpectation(jsonData string) string {
	GinkgoHelper()
	uri := fmt.Sprintf("%s/mockserver/expectation", genaiUrl)

	resp, body, err := HttpCall("PUT", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	mr := make([]ExpectationResponse, 1)
	err = json.Unmarshal(body, &mr)
	Expect(err).To(BeNil())
	By(fmt.Sprintf(" -> Created mockserver expectation: %s", mr[0].ID))
	return mr[0].ID
}

func CreateMockServerExpectationFromFile(file, isolationID, collectionID string) string {
	GinkgoHelper()
	By(fmt.Sprintf("-> Creating mockserver expectation from file %s", file))
	expTpl := ReadTestDataFile(file)
	jsonData := injectHeader(expTpl, "vs-isolation-id", isolationID)
	jsonData = injectHeader(jsonData, "vs-collection-id", collectionID)
	return CreateMockServerExpectation(jsonData)
}

// TODO: use CreateMockServerExpectationFromFile1 function instead of CreateMockServerExpectationFromFile after merging all tests

func CreateMockServerExpectationFromFile1(file, isolationID, collectionID string) string {
	GinkgoHelper()
	By(fmt.Sprintf("-> Creating mockserver expectation from file %s", file))
	expTpl := ReadFile(file)
	jsonData := injectHeader(expTpl, "vs-isolation-id", isolationID)
	jsonData = injectHeader(jsonData, "vs-collection-id", collectionID)
	return CreateMockServerExpectation(jsonData)
}

func CreateMockServerExpectationsFromDir(dir, isolationID, collectionID string) (expIDs []string) {
	GinkgoHelper()
	path, err := os.Getwd()
	Expect(err).To(BeNil())
	dir = fmt.Sprintf("%s/data/%s", path, dir)

	By(fmt.Sprintf("-> Creating mockserver expectations from dir %s", dir))
	f, err := os.Open(dir)
	Expect(err).To(BeNil())

	files, err := f.Readdir(0)
	Expect(err).To(BeNil())

	for _, file := range files {
		fPath := fmt.Sprintf("%s/%s", dir, file.Name())
		expId := CreateMockServerExpectationFromFile1(fPath, isolationID, collectionID)
		expIDs = append(expIDs, expId)
	}
	return expIDs
}

// CreateAdaExpectationsFromDir creates WireMock expectations for the Ada embedder endpoint
// using all mock-server expectation templates found in the given directory. This mirrors
// the behavior of CreateMockServerExpectationsFromDir but targets WireMock instead of the
// legacy mock server.
func CreateAdaExpectationsFromDir(wiremockMgr *tools.WireMockManager, dir string) (expIDs []string) {
	GinkgoHelper()
	path, err := os.Getwd()
	Expect(err).To(BeNil())
	dir = fmt.Sprintf("%s/data/%s", path, dir)

	By(fmt.Sprintf("-> Creating WireMock ADA expectations from dir %s", dir))
	f, err := os.Open(dir)
	Expect(err).To(BeNil())

	files, err := f.Readdir(0)
	Expect(err).To(BeNil())

	for _, file := range files {
		fPath := fmt.Sprintf("%s/%s", dir, file.Name())
		expTpl := ReadFile(fPath)
		mockID, err := CreateAdaExpectationFromTpl(wiremockMgr, expTpl)
		Expect(err).To(BeNil())
		expIDs = append(expIDs, mockID)
	}
	return expIDs
}

func GetAbsPath(fileName string) string {
	GinkgoHelper()
	curDirPath, err := os.Getwd()
	Expect(err).To(BeNil())
	return fmt.Sprintf("%s/data/%s", curDirPath, fileName)
}

func CreateIsolation(baseURI, isolationID, maxStorageSize string) {
	GinkgoHelper()
	By(fmt.Sprintf("Creating isolation %s", isolationID))
	uri := fmt.Sprintf("%s/v1/isolations", baseURI)
	var jsonData = fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "%s" }`, isolationID, maxStorageSize)

	resp, body, err := HttpCall("POST", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("Failed to create expectation: %s", body))
	Expect(string(body)).To(Equal(fmt.Sprintf(`{"id":"%s"}`, isolationID)))
}

func CreateIsolationWithPDCEndpoint(baseURI, isolationID, maxStorageSize, pdcEndpointURL string) {
	GinkgoHelper()
	By(fmt.Sprintf("Creating isolation %s with PDC endpoint %s", isolationID, pdcEndpointURL))
	uri := fmt.Sprintf("%s/v1/isolations", baseURI)
	var jsonData = fmt.Sprintf(`{ "id": "%s", "maxStorageSize": "%s", "pdcEndpointURL": "%s" }`,
		isolationID, maxStorageSize, pdcEndpointURL)

	resp, body, err := HttpCall("POST", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK), fmt.Sprintf("Failed to create isolation: %s", body))
	Expect(string(body)).To(Equal(fmt.Sprintf(`{"id":"%s"}`, isolationID)))
}

func CreateCollection(baseURI, isolationID, collectionID string) {
	GinkgoHelper()
	By(fmt.Sprintf("Creating collection %s in isolation %s", collectionID, isolationID))
	uri := fmt.Sprintf("%s/v2/%s/collections", baseURI, isolationID)
	var jsonData = fmt.Sprintf(`{ "collectionID": "%s" }`, collectionID)

	resp, body, err := HttpCall("POST", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated), fmt.Sprintf("Failed to create collection: %s", body))
}

func CreateAttributeGroup(ctx context.Context, dbPool *pgxpool.Pool, endpointURI, isolationID, description string, attributes []string) (attrGroupID string) {
	GinkgoHelper()
	By(fmt.Sprintf("Creating attributes group in isolation '%s' with attributes: %s ", isolationID, attributes))

	requestBody := fmt.Sprintf(`{ "description": "%s",  "attributes": [] }`, description)
	if len(attributes) > 0 {
		requestBody = fmt.Sprintf(`{ "description": "%s", "attributes": [ "%s" ] }`, description, strings.Join(attributes, `", "`))
	}

	resp, body, err := HttpCall("POST", endpointURI, nil, requestBody)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	respBody := attributesgroup.AttributesGroup{}
	err = json.Unmarshal(body, &respBody)
	Expect(err).To(BeNil())
	Expect(respBody.GroupID).NotTo(BeEmpty())
	Expect(len(respBody.Attributes)).To(Equal(len(attributes)))
	for _, attr := range attributes {
		Expect(respBody.Attributes).To(ContainElement(attr))
	}

	By(fmt.Sprintf("Expect attributes group '%s' exists in DB with expected attributes", respBody.GroupID))
	ExpectAttributesGroupExistsInDB(ctx, dbPool, isolationID, respBody.GroupID, description, attributes)
	return respBody.GroupID
}
