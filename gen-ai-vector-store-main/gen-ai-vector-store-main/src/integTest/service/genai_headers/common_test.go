//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package cross_functional_test

import (
	"fmt"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Wraps HttpCall and HttpCallMultipartForm to track API endpoint calls
// Functions moved to src/integTest/functions/common.go

func ExpectAllEndpointsTested() {
	By("Verifying all endpoints were called at least once")

	// Check only these response codes
	codesToCheck := []int{200, 201, 202}

	// Get all normalized endpoints/methods/response codes from spec
	swaggerSpecURI := fmt.Sprintf("%s/", svcBaseURI)
	specEndpoints, err := GetNormalizedSpecEndpointsWithResponseCodes(swaggerSpecURI)
	Expect(err).To(BeNil(), "Failed to get spec endpoints: %v", err)

	codeIsChecked := func(code int) bool {
		for _, c := range codesToCheck {
			if c == code {
				return true
			}
		}
		return false
	}

	shouldSkipEndpoint := func(endpoint, method string) bool {
		switch endpoint {
		case "/v1/{isolationID}/collections/{collectionName}/file",
			"/v1/{isolationID}/collections/{collectionName}/file/text":
			// File upload endpoints are not implemented in this test environment,
			// so we do not enforce coverage for them here (any method/response).
			return true
		default:
			return false
		}
	}

	red := "\033[31m"
	green := "\033[32m"
	reset := "\033[0m"
	tested := []string{}
	missing := []string{}

	for endpoint, methods := range specEndpoints {
		for method, codes := range methods {
			for code := range codes {
				if shouldSkipEndpoint(endpoint, method) {
					continue
				}
				if !codeIsChecked(code) {
					continue
				}
				if EndpointsCallsStats[endpoint] != nil && EndpointsCallsStats[endpoint][method] != nil && EndpointsCallsStats[endpoint][method][code] > 0 {
					tested = append(tested, fmt.Sprintf(green+"TESTED"+reset+":   %-40s %-6s %d", endpoint, method, code))
				} else {
					missing = append(missing, fmt.Sprintf(red+"MISSED"+reset+":   %-40s %-6s %d", endpoint, method, code))
				}
			}
		}
	}

	if len(missing) > 0 {
		msg := "\n==== Endpoint Coverage Report ===="

		msg += "\n\nMissed endpoints:"
		for _, m := range missing {
			msg += "\n  " + m
		}

		msg += "\n\nTested endpoints:"
		for _, t := range tested {
			msg += "\n  " + t
		}

		// Print EndpointsCallsStats items for debugging
		msg += "\n\nEndpoints call statistic:"
		for endpoint, methods := range EndpointsCallsStats {
			for method, statuses := range methods {
				for status, count := range statuses {
					msg += fmt.Sprintf("\n  %s %s %d : %d", endpoint, method, status, count)
				}
			}
		}

		msg += "\n\nTotal tested: %d, missed: %d\n"
		msg = fmt.Sprintf(msg, len(tested), len(missing))
		Expect(len(missing)).To(Equal(0), msg)
	}
}
