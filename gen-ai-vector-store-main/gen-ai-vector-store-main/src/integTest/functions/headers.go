/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package test_functions

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ServerConfigurationHeaders contains common headers used for server configuration in tests
var ServerConfigurationHeaders = map[string]string{
	headers.ForceFreshDbMetrics: "true",
}

func ExpectHeadersCommon(resp *http.Response) {
	GinkgoHelper()

	By("Check common headers in the response")
	Expect(resp.Header.Get(headers.RequestDurationMs)).NotTo(BeEmpty())

	requestDuration, err := strconv.Atoi(resp.Header.Get(headers.RequestDurationMs))
	Expect(err).To(BeNil())
	Expect(requestDuration).To(BeNumerically(">=", 0))
}

func ExpectHeadersEmbedding(resp *http.Response, embeddingDurationMs, callsCount int) {
	GinkgoHelper()

	By("Check embedding headers in the response")
	Expect(resp.Header.Get(headers.ModelId)).NotTo(BeEmpty())
	Expect(resp.Header.Get(headers.ModelVersion)).NotTo(BeEmpty())
	Expect(resp.Header.Get(headers.EmbeddingTimeMs)).NotTo(BeEmpty())
	Expect(resp.Header.Get(headers.EmbeddingCallsCount)).NotTo(BeEmpty())

	embeddingTime, err := strconv.Atoi(resp.Header.Get(headers.EmbeddingTimeMs))
	Expect(err).To(BeNil())
	Expect(embeddingTime).To(BeNumerically(">=", embeddingDurationMs))

	embeddingCalls, err := strconv.Atoi(resp.Header.Get(headers.EmbeddingCallsCount))
	Expect(err).To(BeNil())
	Expect(embeddingCalls).To(Equal(callsCount))
}

func ExpectHeadersDatabase(resp *http.Response) {
	GinkgoHelper()

	By("Check db headers in the response")
	Expect(resp.Header.Get(headers.DbQueryTimeMs)).NotTo(BeEmpty())

	// Validate the headers have proper values
	dbQueryTime, err := strconv.Atoi(resp.Header.Get(headers.DbQueryTimeMs))
	Expect(err).To(BeNil())
	Expect(dbQueryTime).To(BeNumerically(">=", 0))
}

func ExpectHeadersItemsCount(resp *http.Response, itemsCount int) {
	GinkgoHelper()

	By("Check items count headers in the response")

	Expect(resp.Header.Get(headers.ResponseReturnedItemsCount)).NotTo(BeEmpty())

	returnedItemsCount, err := strconv.Atoi(resp.Header.Get(headers.ResponseReturnedItemsCount))
	Expect(err).To(BeNil())
	Expect(returnedItemsCount).To(Equal(itemsCount))
}

// HeaderCheckType defines the type of header check
// "exists" - header exists and not empty
// "equals" - header exists and value equals expected
// "between" - header exists and value is between two numbers (inclusive)
type HeaderCheckType string

const (
	HeaderExists         HeaderCheckType = "exists"
	HeaderEquals         HeaderCheckType = "equals"
	HeaderBetween        HeaderCheckType = "between"
	HeaderGreaterOrEqual HeaderCheckType = "greaterOrEqual"
)

// HeaderCheck describes a single header validation
// For "equals", Expected should be string
// For "between", Expected should be [2]int
// For "exists", Expected is ignored
// Example:
//
//	HeaderCheck{Name: "X-Genai-Vectorstore-Model-Id", Type: HeaderExists}
//	HeaderCheck{Name: "X-Genai-Vectorstore-Model-Version", Type: HeaderEquals, Expected: "v1"}
//	HeaderCheck{Name: "X-Genai-Vectorstore-Request-Duration-Ms", Type: HeaderBetween, Expected: [2]int{0, 10000}}
type HeaderCheck struct {
	Name     string
	Type     HeaderCheckType
	Expected interface{}
}

// ExpectHeaderExistsAndNotEmpty checks if a header exists and is not empty
func ExpectHeaderExistsAndNotEmpty(resp *http.Response, headerName string) {
	GinkgoHelper()
	By(fmt.Sprintf("Expect header '%s' exists and is not empty", headerName))
	headerValue := resp.Header.Get(headerName)
	Expect(headerValue).NotTo(BeEmpty(), fmt.Sprintf("Header '%s' should exist and not be empty", headerName))
}

// ExpectHeaderExistsAndEquals checks if a header exists and has the expected value
func ExpectHeaderExistsAndEquals(resp *http.Response, headerName, expectedValue string) {
	GinkgoHelper()
	By(fmt.Sprintf("Expect header '%s' exists and equals '%s'", headerName, expectedValue))
	headerValue := resp.Header.Get(headerName)
	Expect(headerValue).To(Equal(expectedValue), fmt.Sprintf("Header '%s' should equal '%s' but was '%s'", headerName, expectedValue, headerValue))
}

// ExpectHeaderExistsAndBetween checks if a header exists and its numeric value is between min and max (inclusive)
func ExpectHeaderExistsAndBetween(resp *http.Response, headerName string, min, max int) {
	GinkgoHelper()
	By(fmt.Sprintf("Expect header '%s' exists and value is between %d and %d", headerName, min, max))
	headerValue := resp.Header.Get(headerName)
	Expect(headerValue).NotTo(BeEmpty(), fmt.Sprintf("Header '%s' should exist and not be empty", headerName))
	numericValue, err := strconv.Atoi(headerValue)
	Expect(err).To(BeNil(), fmt.Sprintf("Header '%s' value '%s' should be a valid integer", headerName, headerValue))
	Expect(numericValue).To(BeNumerically(">=", min), fmt.Sprintf("Header '%s' value %d should be >= %d", headerName, numericValue, min))
	Expect(numericValue).To(BeNumerically("<=", max), fmt.Sprintf("Header '%s' value %d should be <= %d", headerName, numericValue, max))
}

// ExpectHeaderExistsAndGreaterOrEqual checks if a header exists and its numeric value is greater or equal to min
func ExpectHeaderExistsAndGreaterOrEqual(resp *http.Response, headerName string, min int) {
	GinkgoHelper()
	By(fmt.Sprintf("Expect header '%s' exists and value is >= %d", headerName, min))
	headerValue := resp.Header.Get(headerName)
	Expect(headerValue).NotTo(BeEmpty(), fmt.Sprintf("Header '%s' should exist and not be empty", headerName))
	numericValue, err := strconv.Atoi(headerValue)
	Expect(err).To(BeNil(), fmt.Sprintf("Header '%s' value '%s' should be a valid integer", headerName, headerValue))
	Expect(numericValue).To(BeNumerically(">=", min), fmt.Sprintf("Header '%s' value %d should be >= %d", headerName, numericValue, min))
}

// ExpectHeadersFlexible checks a set of headers with flexible rules
func ExpectHeadersFlexible(resp *http.Response, checks []HeaderCheck) {
	GinkgoHelper()
	By("Check multiple headers with flexible rules")
	for _, check := range checks {
		switch check.Type {
		case HeaderExists:
			ExpectHeaderExistsAndNotEmpty(resp, check.Name)
		case HeaderEquals:
			ExpectHeaderExistsAndNotEmpty(resp, check.Name)
			// Convert Expected to string using fmt.Sprintf
			expectedValue := fmt.Sprintf("%v", check.Expected)
			ExpectHeaderExistsAndEquals(resp, check.Name, expectedValue)
		case HeaderBetween:
			ExpectHeaderExistsAndNotEmpty(resp, check.Name)
			bounds, ok := check.Expected.([2]int)
			Expect(ok).To(BeTrue(), fmt.Sprintf("Expected value for header '%s' should be [2]int array", check.Name))
			ExpectHeaderExistsAndBetween(resp, check.Name, bounds[0], bounds[1])
		case HeaderGreaterOrEqual:
			ExpectHeaderExistsAndNotEmpty(resp, check.Name)
			minVal, ok := check.Expected.(int)
			Expect(ok).To(BeTrue(), fmt.Sprintf("Expected value for header '%s' should be int", check.Name))
			ExpectHeaderExistsAndGreaterOrEqual(resp, check.Name, minVal)
		default:
			Fail(fmt.Sprintf("Unknown header check type: %s", check.Type))
		}
	}
}

func ExpectHeadersProcessingOverhead(resp *http.Response) {
	GinkgoHelper()
	By("Check processing overhead headers in the response")

	for _, h := range []string{headers.ProcessingDurationMs, headers.OverheadMs, headers.EmbeddingNetOverheadMs} {
		Expect(resp.Header.Get(h)).NotTo(BeEmpty())
		val, err := strconv.Atoi(resp.Header.Get(h))
		Expect(err).To(BeNil())
		Expect(val).To(BeNumerically(">=", 0))
	}
}

// ExpectAllHeadersCoveredByTestChecks asserts that all expected header names are present in the check slice
func ExpectAllHeadersCoveredByTestChecks(expectedHeaderNames []string, checks []HeaderCheck) {
	GinkgoHelper()
	for _, hName := range expectedHeaderNames {
		found := false
		for _, check := range checks {
			if check.Name == hName {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), fmt.Sprintf(
			"Expected header '%s' not found in checks but defined in swagger spec. "+
				"Fix test by adding missed header check or remove from swagger spec of not applicable.", hName))
	}
}
