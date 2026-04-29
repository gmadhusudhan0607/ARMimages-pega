//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package service_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OPS Suite")
}

var mappingFile = functions.GetEnvOfDefault("CONFIGURATION_FILE", "./../../../build/config/mapping.yaml")

// with redirect do second container with REQUEST_PROCESSING_COPYRIGHT_PROTECTION=true
var mappingFile1 = functions.GetEnvOfDefault("CONFIGURATION_FILE1", "./../../../build/config/mapping1.yaml")

var mappingFilePrivateModel = functions.GetEnvOfDefault("CONFIGURATION_FILE_PRIVATE_MODEL", "./../../../build/config/mapping-private-model.yaml")
var privateModelConfigPath = functions.GetEnvOfDefault("PRIVATE_MODEL_FILE_PATH", "./../../../build/private-model-config")

var mappingFileGenAiInfra = functions.GetEnvOfDefault("CONFIGURATION_FILE_GENAI_INFRA", "./../../../build/config/mapping-genai-infra.yaml")
var dirSecretsGenAiInfra = functions.GetEnvOfDefault("DIR_SECRETS_GENAI_INFRA", "./../genai-infra-config")

var mockServerURL = functions.GetEnvOfDefault("GENAI_URL", "http://localhost:11080")
var svcPort = functions.GetEnvOfDefault("SERVICE_PORT", "8080")
var svcHealthcheckPort = functions.GetEnvOfDefault("SERVICE_HEALTHCHECK_PORT", "18082")

var svcBaseURL string

var _ = BeforeSuite(func() {
	initialSetup()
	functions.ExpectServiceIsAccessible(svcBaseURL)
	functions.ExpectServiceIsAccessible(mockServerURL)
})

func initialSetup() {
	println("====================================================================================")
	println("Environment configuration:")
	println("  svcPort = " + svcPort)
	println("  svcHealthcheckPort = " + svcHealthcheckPort)
	println("  mappingFile = " + mappingFile)
	println("  mappingFile1 = " + mappingFile1)
	println("  mappingFileGenAiInfra = " + mappingFileGenAiInfra)
	println("")

	svcBaseURL = fmt.Sprintf("http://localhost:%s", svcPort)

	println("  svcBaseURL = " + svcBaseURL)
}
