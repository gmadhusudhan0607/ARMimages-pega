//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package emulation_mode_test

import (
	"fmt"
	"strings"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/headers"
	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ServiceRuntimeHeaders emulates read-only mode behavior by sending the appropriate header
var ServiceRuntimeHeaders = map[string]string{
	headers.ServiceMode: string(config.ServiceModeEmulation),
}

// ExpectAllServiceEndpointsTestedForEmulationMode validates that all service endpoints are covered by Emulation mode tests
func ExpectAllServiceEndpointsTestedForEmulationMode(swaggerSpecURI string) {
	By("Validating all service endpoints are tested for Emulation mode")

	// Get all endpoints from swagger spec
	specEndpoints, err := GetNormalizedSpecEndpointsWithResponseCodes(swaggerSpecURI)
	Expect(err).To(BeNil())

	// Filter service endpoints (exclude ops, health, etc.)
	serviceEndpoints := filterServiceEndpoints(specEndpoints)

	totalEndpoints := 0
	testedEndpoints := 0

	// Validate coverage
	for endpoint, methods := range serviceEndpoints {
		for method := range methods {
			if isServiceEndpoint(endpoint) {
				totalEndpoints++

				// Check if endpoint was called during tests
				if isEndpointTested(endpoint, method) {
					testedEndpoints++
				} else {
					// Log untested endpoint
					fmt.Printf("WARNING: Endpoint %s %s was not tested\n", method, endpoint)
				}
			}
		}
	}

	// Print coverage summary
	fmt.Println("\n=== Service Emulation Mode Test Coverage Summary ===")

	coverage := float64(testedEndpoints) / float64(totalEndpoints) * 100
	fmt.Printf("Coverage: %d/%d endpoints tested (%.1f%%)\n", testedEndpoints, totalEndpoints, coverage)

	if coverage < 100 {
		Fail(fmt.Sprintf("Incomplete endpoint coverage: %.1f%% (expected 100%%)", coverage))
	}
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

func isEndpointTested(endpoint, method string) bool {
	if stats, exists := EndpointsCallsStats[endpoint]; exists {
		_, methodExists := stats[method]
		return methodExists
	}
	return false
}
