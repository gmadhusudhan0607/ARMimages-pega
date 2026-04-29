//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

import (
	"fmt"
	"net/http"

	functions "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ExpectLLMCall is a local implementation to replace functions.ExpectModelCall()
// This function makes HTTP calls to model endpoints with proper headers and test IDs
func ExpectLLMCall(isolationID string, url, body string, headers map[string]string) (*http.Response, []byte) {
	// Extend provided headers by adding test-id and authorization
	extendedHeaders := make(map[string]string)

	// Copy provided headers
	for k, v := range headers {
		extendedHeaders[k] = v
	}

	// Add test-id header
	extendedHeaders["X-Genai-Gateway-Isolation-ID"] = isolationID

	// Generate and add Authorization header using the isolationID as isolation ID
	authHeader := CreateAuthHeaderWithIsolationID(isolationID)
	extendedHeaders["Authorization"] = authHeader

	By(fmt.Sprintf("-> Calling %s", url))
	resp, respBody, err := functions.ExpectHttpCall("POST", url, extendedHeaders, body)
	Expect(err).To(BeNil())
	return resp, respBody
}

// ExpectLLMCallSuccessWithExactResponse makes an HTTP call and verifies the exact response
func ExpectLLMCallSuccessWithExactResponse(isolationID, requestUrl, requestBody, expectedResponseBody string) string {
	// Generate and add Authorization header using the isolationID
	authHeader := CreateAuthHeaderWithIsolationID(isolationID)

	headers := map[string]string{
		"X-Genai-Gateway-Isolation-ID": isolationID,
		"Content-Type":                 "application/json",
		"Authorization":                authHeader,
	}

	resp, body, err := functions.ExpectHttpCall("POST", requestUrl, headers, requestBody)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(200), fmt.Sprintf("Expected status code 200 but got %d", resp.StatusCode))

	// Return the actual response body for exact comparison
	actualResponse := string(body)
	Expect(actualResponse).To(Equal(expectedResponseBody),
		fmt.Sprintf("Response body was modified. Expected: %s, Got: %s", expectedResponseBody, actualResponse))

	return actualResponse
}
