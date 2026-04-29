//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package functions

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func ExpectHttpCall(method, uri string, headers map[string]string, body string) (response *http.Response, respBody []byte, err error) {
	By(fmt.Sprintf("-> HTTP/%s: %s", method, uri))
	request, reqErr := http.NewRequest(method, uri, bytes.NewBuffer([]byte(body)))
	Expect(reqErr).To(BeNil())
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	resp, err := (&http.Client{Transport: &http.Transport{DisableCompression: true}}).Do(request)
	Expect(err).To(BeNil())
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(resp.Body)

	return resp, rawBody, err
}

func ExpectServiceIsAccessible(uri string) {
	By(fmt.Sprintf("Expect service is accesible on %s", uri))
	u, err := url.Parse(uri)
	Expect(err).To(BeNil())
	Expect(isPortAccessible(u.Hostname(), u.Port())).To(Equal(true), fmt.Sprintf("service %s not accssible", uri))
}

func ExpectModelCall(model *Model, path, body, testID string) *http.Response {
	url := fmt.Sprintf("%s%s", model.ModelUrl, path)
	By(fmt.Sprintf("-> Calling %s", url))
	headers := map[string]string{
		"test-id":                      testID,
		"X-Genai-Gateway-Isolation-ID": testID,
	}
	if body == "" || body == "{}" {
		body = fmt.Sprintf("{\"modelId\": \"%s\"}", model.ModelId)
	}
	resp := ExpectCallWithHeaders(url, body, headers)
	ExpectPerformanceMetricHeadersArePresent(resp)
	return resp
}

func ExpectModelCallWithJwt(model *Model, path, body, testID string) *http.Response {
	url := fmt.Sprintf("%s%s", model.ModelUrl, path)
	By(fmt.Sprintf("-> Calling %s", url))
	headers := map[string]string{
		"test-id":                      testID,
		"X-Genai-Gateway-Isolation-ID": testID,
		"Authorization":                "Bearer SAXTOKEN",
	}
	if body == "" || body == "{}" {
		body = fmt.Sprintf("{\"modelId\": \"%s\"}", model.ModelId)
	}
	resp := ExpectCallWithHeaders(url, body, headers)
	ExpectPerformanceMetricHeadersArePresent(resp)
	return resp
}

func ExpectCallWithHeaders(url, body string, headers map[string]string) *http.Response {
	By(fmt.Sprintf("-> Calling %s", url))
	resp, _, err := ExpectHttpCall("POST", url, headers, body)
	Expect(err).To(BeNil())
	return resp
}

func ExpectBuddyCall(buddy *Buddy, isolationID, path, testID string) {
	By(fmt.Sprintf("Verifying redirection %s", buddy.Name))
	url := strings.Replace(buddy.BuddyUrl, ":isolationID", isolationID, -1)
	url = fmt.Sprintf("%s%s", url, path)
	headers := map[string]string{"test-id": testID}
	_, _, err := ExpectHttpCall("POST", url, headers, "")
	Expect(err).To(BeNil())
}

func ExpectMockServerExpectationCalledForModel(model Model, msg string) {
	Expect(model.Expectation).NotTo(BeNil(), msg)
	Expect(model.CoveredByTest).To(Equal(true), msg)
}

func ExpectMockServerExpectationCalledForBuddy(buddy Buddy, msg string) {
	if buddy.Expectation != nil {
		Expect(buddy.CoveredByTest).To(Equal(true), msg)
	}
}

func ExpectExpectationMatchedForModel(mockServerURL string, model *Model, atLeast, atMost int) {
	By(fmt.Sprintf("Verify expectation '%s' for model '%s' matched for atLeast %d time(s), atMost %d time(s)", model.Expectation.Id, model.Name, atLeast, atLeast))
	uri := fmt.Sprintf("%s/mockserver/verify", mockServerURL)
	inReqBody := fmt.Sprintf(MockValidateExpectationReqTpl, model.Expectation.Id, atLeast, atMost)
	resp, body, err := ExpectHttpCall("PUT", uri, nil, inReqBody)
	Expect(err).To(BeNil())
	Expect(body).NotTo(BeNil())
	Expect(resp).NotTo(BeNil())
	errMsg := fmt.Sprintf("expectation %s for %s (%s): %s", model.Expectation.Id, model.Name, model.RedirectUrl, body)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted), errMsg)
	model.CoveredByTest = true
}

func ExpectExpectationMatchedForBuddy(mockServerURL string, buddy *Buddy, atLeast, atMost int) {
	By(fmt.Sprintf("Verify epectation '%s' for buddy '%s' matched for atLeast %d time(s), atMost %d time(s)", buddy.Expectation.Id, buddy.Name, atLeast, atLeast))
	uri := fmt.Sprintf("%s/mockserver/verify", mockServerURL)
	inReqBody := fmt.Sprintf(MockValidateExpectationReqTpl, buddy.Expectation.Id, atLeast, atMost)
	resp, body, err := ExpectHttpCall("PUT", uri, nil, inReqBody)
	Expect(err).To(BeNil())
	Expect(body).NotTo(BeNil())
	Expect(resp).NotTo(BeNil())
	errMsg := fmt.Sprintf("expectation %s for %s (%s): %s", buddy.Expectation.Id, buddy.Name, buddy.RedirectUrl, body)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted), errMsg)
	buddy.CoveredByTest = true
}

func ExpectExpectationMatchedForAws(mockServerURL string, infraModel *GenAIInfraConfig, atLeast, atMost int) {
	uri := fmt.Sprintf("%s/mockserver/verify", mockServerURL)
	for _, exp := range infraModel.Expectations {
		By(fmt.Sprintf("Verify AWS expectation %s for %s matched at least %d time(s), atMost %d time(s)", exp.Id, exp.HttpRequest.Path, atLeast, atMost))
		inReqBody := fmt.Sprintf(MockValidateExpectationReqTpl, exp.Id, atLeast, atMost)
		resp, body, err := ExpectHttpCall("PUT", uri, nil, inReqBody)
		Expect(err).To(BeNil())
		Expect(body).NotTo(BeNil())
		Expect(resp).NotTo(BeNil())
		errMsg := fmt.Sprintf("expectation %s for %s: %s", exp.Id, infraModel.ModelId, body)
		Expect(resp.StatusCode).To(Equal(http.StatusAccepted), errMsg)
	}
}

func ExpectUniqUrls(mappings Mappings) {
	// Expect all urls in mappings are uniq

	var modelUrls []string
	for _, m := range mappings.Models {
		Expect(slices.Contains(modelUrls, m.ModelUrl)).To(BeFalse())
		modelUrls = append(modelUrls, m.ModelUrl)
	}
	for _, b := range mappings.Buddies {
		Expect(slices.Contains(modelUrls, b.BuddyUrl)).To(BeFalse())
		modelUrls = append(modelUrls, b.BuddyUrl)
	}
}

func ExpectOpsCall(port, method, path, body, testID string) (int, string) {
	uri := fmt.Sprintf("%s%s", "http://localhost:"+port, path)
	By(fmt.Sprintf("-> Calling %s", uri))
	headers := map[string]string{"test-id": testID}
	resp, respBody, err := ExpectHttpCall(method, uri, headers, body)
	Expect(err).ShouldNot(HaveOccurred())
	return resp.StatusCode, string(respBody)
}

func ExpectPerformanceMetricHeadersArePresent(r *http.Response) {
	// get response header with the Trailers declaration
	By(fmt.Sprintf("-> Checking Performance Metrics are present for response status %s", r.Status))
	Expect(r).ToNot(BeNil())

	ts := []string{}
	var a http.Header
	if len(r.Trailer) > 0 {
		a = r.Trailer
	} else {
		a = r.Header
	}
	for k := range a {
		ts = append(ts, k)
	}

	var expectedMetadata []middleware.GatewayHeader
	if r.StatusCode == http.StatusOK {
		expectedMetadata = middleware.AllGatewayHeaders
	} else {
		expectedMetadata = middleware.FailedInferenceHeaders
	}
	// iterate the custom headers declared in metrics_middleware.go and check they are declared as trailers
	for _, t := range expectedMetadata {
		Expect(ts).To(ContainElement(string(t)),
			fmt.Sprintf("Expected Trailer/Header metadata %s not found. Expected metadata are: %s", t, ts))
	}
}
