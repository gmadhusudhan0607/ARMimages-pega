//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package service_test

import (
	"fmt"
	"strings"

	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tests SVC:", Ordered, func() {

	var mappings Mappings
	var err error
	var testID string

	BeforeAll(func() {
		mappings, err = LoadMappingFromFile(mappingFile)
		Expect(err).To(BeNil())
		println(testID)
	})

	AfterAll(func() {
		// Cleanup
		for _, model := range mappings.Models {
			if model.Expectation != nil && model.Expectation.Id != "" {
				DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
			}
		}

		// Additional checks
		for _, model := range mappings.Models {
			if model.Capabilities.Completions {
				Expect(model.CoveredByTest).To(Equal(true), fmt.Sprintf("Copyrights validation for %s not covered by any test", model.Name))
			}
		}
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
	})

	_ = Context("calling chat/completions (REQUEST_PROCESSING_COPYRIGHT_PROTECTION=false)", func() {
		It("model gpt-35-turbo must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		It("model gpt-4-preview must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4-preview")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		It("model gpt-4-vision-preview must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4-vision-preview")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		It("model gpt-4o must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4o")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		It("model gpt-4o-mini must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4o-mini")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-5 must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-5-mini must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-mini")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-5-nano must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-nano")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-5-chat must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-chat")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-4.1 must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-4.1-mini must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1-mini")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-4.1-nano must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1-nano")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-5.1 must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5.1")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		It("model gpt-5.2 must not add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5.2")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
	})
})
