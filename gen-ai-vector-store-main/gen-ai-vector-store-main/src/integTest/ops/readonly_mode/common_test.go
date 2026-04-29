//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/opsmetrics"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Wraps HttpCall and HttpCallMultipartForm to track API endpoint calls for readonly mode testing
// Functions moved to src/integTest/functions/common.go

var ServiceRuntimeHeaders = map[string]string{
	headers.ServiceMode: string(config.ServiceModeReadOnly),
}

// OpsReadOnlyAllowedEndpoints defines the predefined list of allowed OPS operations in read-only mode
// Similar to DefaultReadOnlyConfig in cmd/middleware/readonly.go
// All operations NOT in this list are considered write operations and should be blocked
var OpsReadOnlyAllowedEndpoints = map[string][]string{
	"GET": {
		"/v1/isolations/{isolationID}", // Get isolation details
		"/v1/db/configuration",         // Get DB configuration
		"/v1/db/size",                  // Get database size
	},
	"POST": {
		"/v1/ops/{isolationID}/documents",        // Get documents metrics (read operation using POST)
		"/v1/ops/{isolationID}/documentsDetails", // Get documents metrics details (read operation using POST)
	},
}

func ExpectAllOpsEndpointsTestedForReadOnlyMode(swaggerSpecURI string) {
	By("Verifying all OPS endpoints were tested for readonly mode at least once")

	// Check only these response codes for readonly mode testing
	codesToCheck := []int{200, 201, 202, 405}

	// Get all normalized endpoints/methods/response codes from OPS spec
	specEndpoints, err := GetNormalizedSpecEndpointsWithResponseCodes(swaggerSpecURI)
	Expect(err).To(BeNil(), "Failed to get OPS spec endpoints: %v", err)

	codeIsChecked := func(code int) bool {
		for _, c := range codesToCheck {
			if c == code {
				return true
			}
		}
		return false
	}

	red := "\033[31m"
	green := "\033[32m"
	reset := "\033[0m"
	tested := []string{}
	missing := []string{}
	readOnlyExpected := []string{}

	for endpoint, methods := range specEndpoints {
		for method, codes := range methods {
			for code := range codes {
				if !codeIsChecked(code) {
					continue
				}

				// Determine if this endpoint/method should be blocked in readonly mode
				isWriteOperation := isWriteOperationForOps(method, endpoint)
				expectedCode := code
				if isWriteOperation {
					expectedCode = 405 // Write operations should return 405 in readonly mode
				}

				if EndpointsCallsStats[endpoint] != nil && EndpointsCallsStats[endpoint][method] != nil && EndpointsCallsStats[endpoint][method][expectedCode] > 0 {
					if isWriteOperation {
						tested = append(tested, fmt.Sprintf(green+"TESTED (RO)"+reset+": %-40s %-6s %d (blocked)", endpoint, method, expectedCode))
					} else {
						tested = append(tested, fmt.Sprintf(green+"TESTED"+reset+":     %-40s %-6s %d", endpoint, method, expectedCode))
					}
				} else {
					if isWriteOperation {
						missing = append(missing, fmt.Sprintf(red+"MISSED (RO)"+reset+": %-40s %-6s %d (should be blocked)", endpoint, method, expectedCode))
						readOnlyExpected = append(readOnlyExpected, fmt.Sprintf("  %s %s -> should return 405", method, endpoint))
					} else {
						missing = append(missing, fmt.Sprintf(red+"MISSED"+reset+":     %-40s %-6s %d", endpoint, method, expectedCode))
					}
				}
			}
		}
	}

	if len(missing) > 0 {
		msg := "\n==== OPS Readonly Mode Endpoint Coverage Report ===="

		msg += "\n\nMissed endpoints:"
		for _, m := range missing {
			msg += "\n  " + m
		}

		if len(readOnlyExpected) > 0 {
			msg += "\n\nWrite operations that should be tested for readonly mode blocking:"
			for _, ro := range readOnlyExpected {
				msg += "\n" + ro
			}
		}

		msg += "\n\nTested endpoints:"
		for _, t := range tested {
			msg += "\n  " + t
		}

		// Print EndpointsCallsStats items for debugging
		msg += "\n\nEndpoints call statistics:"
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

// isWriteOperationForOps determines if an endpoint/method combination is a write operation for OPS
// Uses predefined list of allowed operations - all operations NOT in the allowed list are considered write operations
func isWriteOperationForOps(method, endpoint string) bool {
	// Check if this endpoint is allowed in read-only mode using the predefined list
	// If it's NOT allowed, then it's a write operation that should be blocked
	return !isOpsEndpointAllowedInReadOnlyMode(method, endpoint)
}

// isOpsEndpointAllowedInReadOnlyMode checks if the given method and path combination is allowed for OPS in read-only mode
// Similar to isEndpointAllowed in cmd/middleware/readonly.go but specific to OPS endpoints
func isOpsEndpointAllowedInReadOnlyMode(method, path string) bool {
	allowedPaths, exists := OpsReadOnlyAllowedEndpoints[method]
	if !exists {
		return false
	}

	for _, allowedPath := range allowedPaths {
		if opsPathMatches(path, allowedPath) {
			return true
		}
	}

	return false
}

// opsPathMatches checks if a request path matches an allowed path pattern for OPS endpoints
// Similar to pathMatches in cmd/middleware/readonly.go but simplified for OPS test patterns
func opsPathMatches(requestPath, allowedPattern string) bool {
	// Handle exact matches first
	if requestPath == allowedPattern {
		return true
	}

	// Handle parameter patterns (e.g., /v1/isolations/{isolationID})
	return opsGinPathMatches(requestPath, allowedPattern)
}

// opsGinPathMatches checks if a request path matches a pattern with parameters for OPS endpoints
// Similar to ginPathMatches in cmd/middleware/readonly.go but handles {param} syntax from OpenAPI spec
func opsGinPathMatches(requestPath, pattern string) bool {
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")

	// If different number of parts, they don't match
	if len(requestParts) != len(patternParts) {
		return false
	}

	// Check each part
	for i, patternPart := range patternParts {
		requestPart := requestParts[i]

		// If pattern part is in {param} format, it's a parameter - matches any value
		if strings.HasPrefix(patternPart, "{") && strings.HasSuffix(patternPart, "}") {
			continue
		}

		// Otherwise, must be exact match
		if requestPart != patternPart {
			return false
		}
	}

	return true
}

// HttpCallWithHeadersAndReadOnlyApiCallStat wraps HttpCallWithHeaders and tracks API endpoint calls for readonly mode testing
func HttpCallWithHeadersAndReadOnlyApiCallStat(method, uri string, headers map[string]string, reqBody string) (response *http.Response, respBody []byte, err error) {
	return HttpCallWithHeadersAndApiCallStat(method, uri, headers, reqBody)
}

// Common types for readonly mode testing
type GetIsolationResponse struct {
	ID             string    `json:"id"`
	MaxStorageSize string    `json:"maxStorageSize"`
	CreatedAt      time.Time `json:"createdAt"`
	ModifiedAt     time.Time `json:"modifiedAt"`
}

type DocumentMetricForCollectionResponse struct {
	ID               string                      `json:"id" binding:"required"`
	DocumentsMetrics opsmetrics.DocumentsMetrics `json:"documentsDetailsMetrics" binding:"required"`
}

func getCollectionMetricsByName(collections []DocumentMetricForCollectionResponse, name string) *DocumentMetricForCollectionResponse {
	for _, coll := range collections {
		if coll.ID == name {
			return &coll
		}
	}
	return nil
}

type DocumentMetricForIsolationResponse struct {
	DiskUsage             int64      `json:"diskUsage,omitempty"`
	DocumentsCount        int64      `json:"documentsCount,omitempty"`
	DocumentsModification *time.Time `json:"documentsModification,omitempty"`
}
