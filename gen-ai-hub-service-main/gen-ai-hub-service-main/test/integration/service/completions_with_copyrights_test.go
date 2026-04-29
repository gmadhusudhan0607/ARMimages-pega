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
		mappings, err = LoadMappingFromFile(mappingFile1)
		Expect(err).To(BeNil())
		println(testID)
	})

	AfterAll(func() {
		// Cleanup
		for i := range mappings.Models {
			if mappings.Models[i].Expectation != nil && mappings.Models[i].Expectation.Id != "" {
				DeleteMockServerExpectation(mockServerURL, mappings.Models[i].Expectation.Id)
			}
		}

		// Additional checks
		// Disabled due to US-731110
		// for i := range mappings.Models {
		// 	if mappings.Models[i].Capabilities.Completions {
		// 		Expect(mappings.Models[i].CoveredByTest).To(Equal(true), fmt.Sprintf("Copyrights validation for %s not covered by any test", mappings.Models[i].Name))
		// 	}
		// }
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
	})

	_ = Context("calling chat/completions (REQUEST_PROCESSING_COPYRIGHT_PROTECTION=true) ", func() {
		XIt("model gpt-35-turbo must add copyrights to prompts", Label("fail"), func() {
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		XIt("model gpt-4-preview must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4-preview")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		XIt("model gpt-4-vision-preview must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4-vision-preview")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		XIt("model gpt-4o must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4o")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		XIt("model gpt-4o-mini must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4o-mini")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-5 must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-5-mini must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-mini")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-5-nano must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-nano")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-5-chat must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5-chat")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-4.1 must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-4.1-mini must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1-mini")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-4.1-nano must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1-nano")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-5.1 must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5.1")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
		XIt("model gpt-5.2 must add copyrights to prompts", func() {
			model := GetModelByName(mappings.Models, "gpt-5.2")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionExpectedBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
	})
})
