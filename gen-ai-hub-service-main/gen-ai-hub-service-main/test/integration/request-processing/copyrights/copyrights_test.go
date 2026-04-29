//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package copyrights_test

import (
	"fmt"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions/copyrights"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tests SVC:", Ordered, func() {

	var mappings Mappings
	var testID string
	var testWireMockExpectations []*WireMockExpectation

	BeforeAll(func() {
		var err error
		mappings, err = LoadMappingFromFile(mappingFile)
		Expect(err).NotTo(HaveOccurred())

		println(testID)
	})

	AfterAll(func() {
		// Cleanup WireMock mappings
		for _, mapping := range testWireMockExpectations {
			if mapping != nil && mapping.Id != "" {
				err := DeleteWireMockExpectation(mockServerURL, mapping.Id)
				Expect(err).NotTo(HaveOccurred())
			}
		}

		// Additional checks
		// Disabled due to US-731110
		// for _, model := range mappings.Models {

		// 	if model.Capabilities.Completions {
		// 		Expect(model.CoveredByTest).To(Equal(true), fmt.Sprintf("Copyrights validation for %s not covered by any test", model.Name))
		// 	}
		// }
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
		// Reset WireMock server before each test
		err := ResetWireMockServer(mockServerURL)
		Expect(err).NotTo(HaveOccurred())

		// Recreate mapping and defaults endpoint expectations after reset
		err = CreateMappingEndpointExpectation(mockServerURL, mappingsEndpointPath, mappingEndpointFile)
		Expect(err).To(BeNil())
		err = CreateDefaultsEndpointExpectation(mockServerURL, defaultsEndpointPath, defaultsEndpointFile)
		Expect(err).To(BeNil())

		testWireMockExpectations = []*WireMockExpectation{}
	})

	_ = Context("calling chat/completions (REQUEST_PROCESSING_COPYRIGHT_PROTECTION=true) ", func() {

		XIt("model gpt-35-turbo must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-4-preview must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4-preview")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-4-vision-preview must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4-vision-preview")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-4o must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4o")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-4o-mini must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4o-mini")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-4.1 must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-4.1-mini must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1-mini")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-4.1-nano must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1-nano")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-5 must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-5-mini must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-mini")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-5-nano must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-nano")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-5-chat must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-chat")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-5.1 must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5.1")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})

		XIt("model gpt-5.2 must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5.2")
			mapping, err := CreateWireMockCopyrightsExpectation(mockServerURL, testID, URLPathChatCompletions, chatCompletionExpectedBody, model)
			Expect(err).NotTo(HaveOccurred())
			testWireMockExpectations = append(testWireMockExpectations, mapping)

			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)

			err = VerifyWireMockCopyrightsExpectation(mockServerURL, 1, model, URLPathChatCompletions)
			Expect(err).NotTo(HaveOccurred())
			model.CoveredByTest = true
		})
	})
})
