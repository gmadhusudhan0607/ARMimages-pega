//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package emulation_mode_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/gomega"
)

// TestContext holds common test setup data
type TestContext struct {
	IsolationID  string
	CollectionID string
	BaseURL      string
}

// NewTestContext creates a new test context with generated IDs
func NewTestContext(baseURL string) *TestContext {
	return &TestContext{
		IsolationID:  strings.ToLower(fmt.Sprintf("iso-%s", RandStringRunes(10))),
		CollectionID: strings.ToLower(fmt.Sprintf("col-%s", RandStringRunes(5))),
		BaseURL:      baseURL,
	}
}

// BuildEndpointURI constructs endpoint URIs based on API version and path components
func (tc *TestContext) BuildEndpointURI(version, path string, pathParams ...string) string {
	// Start with base pattern: baseURL + version + isolationID
	uri := fmt.Sprintf("%s%s/%s", tc.BaseURL, version, tc.IsolationID)

	// Add the path as-is (no automatic collection ID insertion)
	uri += path

	// Append additional path parameters
	for _, param := range pathParams {
		uri += "/" + url.PathEscape(param)
	}

	return uri
}

// BuildQueryParams adds query parameters to a URI
func BuildQueryParams(baseURI string, params map[string]string) string {
	u, err := url.Parse(baseURI)
	if err != nil {
		panic(fmt.Sprintf("invalid URI: %s", baseURI))
	}

	query := u.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	return u.String()
}

// HTTPTestHelper encapsulates common HTTP test operations
type HTTPTestHelper struct {
	Context *TestContext
}

// NewHTTPTestHelper creates a new HTTP test helper
func NewHTTPTestHelper(ctx *TestContext) *HTTPTestHelper {
	return &HTTPTestHelper{Context: ctx}
}

// MakeAPICall performs an HTTP call and validates basic response structure
func (h *HTTPTestHelper) MakeAPICall(method, uri, body string, expectedStatus int) (*http.Response, []byte) {
	resp, respBody, err := HttpCallWithHeadersAndApiCallStat(method, uri, ServiceRuntimeHeaders, body)
	Expect(err).To(BeNil())
	Expect(resp).NotTo(BeNil())
	Expect(respBody).NotTo(BeNil())
	Expect(resp.StatusCode).To(Equal(expectedStatus))

	return resp, respBody
}

// ValidateCommonHeaders checks for required response headers
func (h *HTTPTestHelper) ValidateCommonHeaders(resp *http.Response) {
	ExpectHeadersCommon(resp)
	ExpectHeadersDatabase(resp)
}

// MakeAPICallWithValidation performs API call with header validation
func (h *HTTPTestHelper) MakeAPICallWithValidation(method, uri, body string, expectedStatus int) (*http.Response, []byte) {
	resp, respBody := h.MakeAPICall(method, uri, body, expectedStatus)
	h.ValidateCommonHeaders(resp)
	return resp, respBody
}

// DocumentEndpointBuilder helps build document-related endpoints
type DocumentEndpointBuilder struct {
	baseURI string
}

// NewDocumentEndpointBuilder creates a new document endpoint builder
func NewDocumentEndpointBuilder(baseURI string) *DocumentEndpointBuilder {
	return &DocumentEndpointBuilder{baseURI: baseURI}
}

// GetEndpoint builds GET document endpoint
func (d *DocumentEndpointBuilder) GetEndpoint(docID string) string {
	return d.buildDocumentEndpoint(docID)
}

// PatchEndpoint builds PATCH document endpoint
func (d *DocumentEndpointBuilder) PatchEndpoint(docID string) string {
	return d.buildDocumentEndpoint(docID)
}

// DeleteEndpoint builds DELETE document endpoint
func (d *DocumentEndpointBuilder) DeleteEndpoint(docID string) string {
	return d.buildDocumentEndpoint(docID)
}

func (d *DocumentEndpointBuilder) buildDocumentEndpoint(docID string) string {
	u, err := url.Parse(d.baseURI)
	if err != nil {
		panic(fmt.Sprintf("invalid base URI: %s", d.baseURI))
	}
	u.Path = fmt.Sprintf("%s/%s", u.Path, url.PathEscape(docID))
	return u.String()
}

// ResponseValidator provides validation helpers for different response types
type ResponseValidator struct{}

// NewResponseValidator creates a new response validator
func NewResponseValidator() *ResponseValidator {
	return &ResponseValidator{}
}

// ValidateWithSchema performs schema validation and returns typed response
func (v *ResponseValidator) ValidateWithSchema(body []byte, validatorFunc func([]byte) (interface{}, error), description string) interface{} {
	result, err := validatorFunc(body)
	Expect(err).To(BeNil(), fmt.Sprintf("Schema validation failed for %s", description))
	Expect(result).NotTo(BeNil(), fmt.Sprintf("Response should not be nil for %s", description))
	return result
}

// MultipartFormHelper helps with multipart form operations
type MultipartFormHelper struct{}

// NewMultipartFormHelper creates a new multipart form helper
func NewMultipartFormHelper() *MultipartFormHelper {
	return &MultipartFormHelper{}
}

// BuildDocumentUploadParts creates multipart form parts for document upload
func (m *MultipartFormHelper) BuildDocumentUploadParts(documentID, filePath string, attributes interface{}) []MultiformPart {
	return []MultiformPart{
		{Type: "field", Name: "documentID", Value: documentID},
		{Type: "field", Name: "documentAttributes", Value: attributes},
		{Type: "file", Name: "documentFile", Value: GetAbsPath(filePath)},
	}
}
