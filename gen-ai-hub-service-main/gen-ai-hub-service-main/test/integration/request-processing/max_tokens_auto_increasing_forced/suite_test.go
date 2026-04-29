//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens_auto_increasing_forced_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service suite (max_tokens STRATEGY='AUTO_INCREASING' + FORCED)")
}

// Configuration variables derived from env vars
var mockServerURL = "http://localhost:11818"
var mappingsEndpointPath = "/api/mappings"
var mappingEndpointFile = "../mapping-endpoint.json"
var defaultsEndpointPath = "/api/defaults"
var defaultsEndpointFile = "../defaults-endpoint.json"
var mappingFile = "../mapping_20090.yaml"
var svcPort = 20090
var svcHealthcheckPort = 20092
var testDir = os.Getenv("PWD")

// Define environment variables directly in the test
// Used by test and service manager to run service in propper mode
var serviceEnvVars = map[string]string{
	"LOG_LEVEL": "DEBUG",

	// Four required configuration sources for NewTargetResolver
	"CONFIGURATION_FILE":       "test/integration/request-processing/mapping_20090.yaml",
	"MAPPING_ENDPOINT":         fmt.Sprintf("%s%s", mockServerURL, mappingsEndpointPath),
	"MODELS_DEFAULTS_ENDPOINT": fmt.Sprintf("%s%s", mockServerURL, defaultsEndpointPath),
	"PRIVATE_MODEL_CONFIG_DIR": "test/integration/request-processing/max_tokens_auto_increasing_forced/private-models-test-dir",

	"SERVICE_PORT":             strconv.Itoa(svcPort),
	"SERVICE_HEALTHCHECK_PORT": strconv.Itoa(svcHealthcheckPort),
	"REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY": "AUTO_INCREASING",
	"REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE":          "1022",
	"REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED":   "true",

	// Enable providers and GenAI infra models (middle routing path)
	"ENABLED_PROVIDERS": "Azure,Bedrock,Vertex",
	"USE_GENAI_INFRA":   "true",
	"USE_AUTO_MAPPING":  "true",

	"GENAI_URL":           "http://localhost:11818/remote/gen-ai-url",
	"DEMO_GCP_VERTEX_URL": "http://localhost:11818/remote/demo-gcp-vertex",
}

var svcBaseURL = fmt.Sprintf("http://localhost:%d", svcPort)
var svcHealthcheckUrl = fmt.Sprintf("http://localhost:%d", svcHealthcheckPort)
var metricsUrl = fmt.Sprintf("%s/metrics", svcHealthcheckUrl)

var serviceManager *functions.ServiceManager

var _ = BeforeSuite(func() {
	var err error

	println("====================================================================================")
	println("Environment configuration:")
	println("  testDir = " + testDir)
	println("  mappingFile = " + mappingFile)
	println("  mockServerURL = " + mockServerURL)
	println("  svcBaseURL = " + svcBaseURL)
	println("  svcHealthcheckUrl = " + svcHealthcheckUrl)
	println("")

	privateModelsDir := "private-models-test-dir"

	// Create private models directory and files
	err = os.MkdirAll(privateModelsDir, 0755)
	Expect(err).NotTo(HaveOccurred())

	err = functions.CreatePrivateModelFiles(privateModelsDir)
	Expect(err).NotTo(HaveOccurred())

	// Start WireMock server
	functions.StartWireMock(mockServerURL)

	// Start genai-hub-service with all configuration sources
	fmt.Println("Starting genai-hub-service with complete configuration...")
	serviceManager, err = functions.NewServiceManager(serviceEnvVars)
	Expect(err).NotTo(HaveOccurred())
	err = serviceManager.StartService()
	Expect(err).NotTo(HaveOccurred())
	fmt.Println("=== BeforeSuite setup completed successfully ===")

})

var _ = AfterSuite(func() {
	println("")
	println("=== AfterSuite cleanup starting ===")

	// Clean up private models directory
	privateModelsDir := "private-models-test-dir"
	err := os.RemoveAll(privateModelsDir)
	if err != nil {
		fmt.Printf("Warning: Failed to remove private models directory: %v\n", err)
	}

	// Check if KEEP is set to skip stopping the service and mock
	keepService := os.Getenv("KEEP") == "true"

	// Stop WireMock server
	if keepService {
		fmt.Println("KEEP=true: Skipping WireMock stop")
	} else {
		functions.StopWireMock()
		fmt.Println("WireMock stopped")
	}

	// Stop the service
	if serviceManager != nil {
		if keepService {
			fmt.Println("KEEP=true: Skipping service stop")
		} else {
			serviceManager.StopService()
			fmt.Println("Service stopped")
		}
	}
})
