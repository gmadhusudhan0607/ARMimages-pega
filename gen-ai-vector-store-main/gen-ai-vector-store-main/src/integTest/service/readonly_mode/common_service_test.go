//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"fmt"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ServiceReadOnlyAllowedEndpoints defines service endpoints that should be allowed in readonly mode
// Everything that is not defined in this list must be considered as blocked
var ServiceReadOnlyAllowedEndpoints = map[string][]string{
	"GET": {
		"/v1/{isolationID}/collections/{collectionName}/documents/{documentID}",
		"/v1/{isolationID}/smart-attributes-group",
		"/v1/{isolationID}/smart-attributes-group/{groupID}",
		"/v2/{isolationID}/collections",
		"/v2/{isolationID}/collections/{collectionID}",
		"/v2/{isolationID}/collections/{collectionID}/documents/{documentID}/chunks",
	},
	"POST": {
		// POST operations used for complex queries/filtering (read operations)
		"/v1/{isolationID}/collections/{collectionName}/documents",
		"/v1/{isolationID}/collections/{collectionName}/query/chunks",
		"/v1/{isolationID}/collections/{collectionName}/query/documents",
		"/v1/{isolationID}/collections/{collectionName}/attributes",
		"/v2/{isolationID}/collections/{collectionID}/find-documents",
	},
}

// ServiceRuntimeHeaders emulates read-only mode behavior by sending the appropriate header
var ServiceRuntimeHeaders = map[string]string{
	headers.ServiceMode: string(config.ServiceModeReadOnly),
}

// ExpectAllServiceEndpointsTestedForReadOnlyMode validates that all service endpoints are covered by readonly mode tests
func ExpectAllServiceEndpointsTestedForReadOnlyMode(swaggerSpecURI string) {
	By("Validating all service endpoints are tested for readonly mode")

	// Get all endpoints from swagger spec
	specEndpoints, err := GetNormalizedSpecEndpointsWithResponseCodes(swaggerSpecURI)
	Expect(err).To(BeNil())

	// Filter service endpoints (exclude ops, health, etc.)
	serviceEndpoints := filterServiceEndpoints(specEndpoints)

	// Validate coverage
	for endpoint, methods := range serviceEndpoints {
		for method := range methods {
			if isServiceEndpoint(endpoint) {
				validateEndpointTested(endpoint, method)
			}
		}
	}

	// Print coverage summary
	printCoverageSummary()
}

func filterServiceEndpoints(endpoints map[string]map[string]map[int]int) map[string]map[string]map[int]int {
	filtered := make(map[string]map[string]map[int]int)

	for endpoint, methods := range endpoints {
		// Include endpoints that match service patterns
		if isServiceEndpoint(endpoint) {
			filtered[endpoint] = methods
		}
	}

	return filtered
}

func isServiceEndpoint(endpoint string) bool {
	// Service endpoints start with /v1/{isolationID} or /v2/{isolationID}
	// Exclude ops endpoints (/v1/ops/, /v1/isolations, etc.)
	return (strings.HasPrefix(endpoint, "/v1/{isolationID}") ||
		strings.HasPrefix(endpoint, "/v2/{isolationID}")) &&
		!strings.Contains(endpoint, "/ops/")
}

func validateEndpointTested(endpoint, method string) {
	// Check if endpoint was called during tests
	if stats, exists := EndpointsCallsStats[endpoint]; exists {
		if _, methodExists := stats[method]; methodExists {
			return // Endpoint was tested
		}
	}

	// Log untested endpoint
	fmt.Printf("WARNING: Endpoint %s %s was not tested\n", method, endpoint)
}

func printCoverageSummary() {
	fmt.Println("\n=== Service ReadOnly Mode Test Coverage Summary ===")

	totalEndpoints := 0
	testedEndpoints := 0

	// Count allowed endpoints (only these are explicitly defined now)
	for method, endpoints := range ServiceReadOnlyAllowedEndpoints {
		for _, endpoint := range endpoints {
			totalEndpoints++
			if isEndpointTested(endpoint, method) {
				testedEndpoints++
			}
		}
	}

	coverage := float64(testedEndpoints) / float64(totalEndpoints) * 100
	fmt.Printf("Coverage: %d/%d endpoints tested (%.1f%%)\n", testedEndpoints, totalEndpoints, coverage)

	if coverage < 100 {
		Fail(fmt.Sprintf("Incomplete endpoint coverage: %.1f%% (expected 100%%)", coverage))
	}
}

func isEndpointTested(endpoint, method string) bool {
	if stats, exists := EndpointsCallsStats[endpoint]; exists {
		_, methodExists := stats[method]
		return methodExists
	}
	return false
}
