//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package functions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/testutils"
)

func CreateMockServerExpectation(mockServerURL, jsonData string) *Expectation {
	By("-> Creating mockserver expectation")
	uri := fmt.Sprintf("%s/mockserver/expectation", mockServerURL)

	resp, body, err := ExpectHttpCall("PUT", uri, nil, jsonData)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated), fmt.Sprintf("failed to create expectation : %s", string(body)))

	mr := make([]Expectation, 1)
	err = json.Unmarshal(body, &mr)
	Expect(err).To(BeNil())
	By(fmt.Sprintf("-> Created expectation: %s", mr[0].Id))
	return &mr[0]
}

func CreateModelMockExpectation(mockServerURL string, model *Model, path, body, testID string) {
	By(fmt.Sprintf("-> Creating mockserver expectation for model %s", model.Name))
	redirectUrlPath := ParseRegexParameters(model.RedirectUrl, `https?://(.*):(\d+)(?P<path>.*)`)["path"]
	if body == "" {
		body = "{}"
	}

	// Remove parameters from URL path if any
	u, err := url.Parse(path)
	Expect(err).To(BeNil())
	path = u.Path

	req := fmt.Sprintf(MockCreateModelExpectationReqWithPathAndBodyTpl, fmt.Sprintf("%s%s", redirectUrlPath, path), body, testID)
	model.Expectation = CreateMockServerExpectation(mockServerURL, req)
}

func CreateAwsMockExpectation(mockAwsServerURL string, genaiInfraConfig *GenAIInfraConfig, path, header string, response string) {
	By("-> Creating aws mockserver expectation")
	var req string
	if header == "" {
		req = fmt.Sprintf(MockCreatePostExpectationReqWithPathAndResponseBodyTpl, path, response)
	} else {
		req = fmt.Sprintf(MockCreatePostExpectationReqWithPathHeaderAndResponseBodyTpl, path, header, response)
	}
	expectation := CreateMockServerExpectation(mockAwsServerURL, req)
	genaiInfraConfig.Expectations = append(genaiInfraConfig.Expectations, *expectation)
}

func CreateBuddyMockExpectation(mockServerURL string, buddy *Buddy, testID string) {
	By(fmt.Sprintf("-> Creating mockserver expectation for buddy %s", buddy.Name))
	redirectUrlPath := ParseRegexParameters(buddy.RedirectUrl, `https?://(.*):(\d+)(?P<path>.*)`)["path"]
	req := fmt.Sprintf(MockCreateModelExpectationReqWithPathTpl, redirectUrlPath, testID)
	buddy.Expectation = CreateMockServerExpectation(mockServerURL, req)
}

func CreateOpsMappingsMockExpectation(mockServerURL string, infraMappings *InfraMappings) {
	By("-> Creating mockserver expectation for /mappings endpoint")
	jsonData, err := json.Marshal(infraMappings.Configs)
	escapedJSON := strings.ReplaceAll(string(jsonData), `"`, `\"`)
	Expect(err).To(BeNil())
	req := fmt.Sprintf(MockCreateGetExpectationReqWithPathAndResponseBodyTpl, "/v1/mappings", escapedJSON)
	infraMappings.Expectation = CreateMockServerExpectation(mockServerURL, req)
}

func CreateOktaTokenExpectation(mockServerURL, testID string) {
	By("-> Creating mockserver expectation for okta token")
	token := testutils.TokenBody{AccessToken: testID}
	jsonData, err := json.Marshal(token)
	//escapedJSON := strings.ReplaceAll(jsonData, `"`, `\"`)
	Expect(err).To(BeNil())
	req := fmt.Sprintf(MockCreatePostExpectationReqWithPathAndResponseBodyTpl, "/okta/v1/token", jsonData)
	CreateMockServerExpectation(mockServerURL, req)
}
