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

		ExpectUniqUrls(mappings)
	})

	AfterAll(func() {
		// Cleanup
		for _, model := range mappings.Models {
			DeleteModelExpectation(mockServerURL, model)
		}
		for _, buddy := range mappings.Buddies {
			DeleteBuddyExpectation(mockServerURL, buddy)
		}

		// Additional checks/ make sere we covered all model/buddy urls
		for _, model := range mappings.Models {
			ExpectMockServerExpectationCalledForModel(model, fmt.Sprintf("Redirection for %s not covered by any test", model.Name))
		}
		for _, buddy := range mappings.Buddies {
			ExpectMockServerExpectationCalledForBuddy(buddy, fmt.Sprintf("Redirection for %s not covered by any test", buddy.Name))
		}

	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
	})

	_ = Context("calling buddy", func() {

		It("must redirect to buddy selfstudybuddy", func() {
			buddy := GetBuddyByName(mappings.Buddies, "selfstudybuddy")
			CreateBuddyMockExpectation(mockServerURL, buddy, testID)
			ExpectBuddyCall(buddy, "iso-1", URLPathQuestion, testID)
			ExpectExpectationMatchedForBuddy(mockServerURL, buddy, 1, 1)
			DeleteMockServerExpectation(mockServerURL, buddy.Expectation.Id)
		})
	})

	_ = Context("calling model", func() {

		It("must redirect to model gpt-35-turbo", func() {
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-4-preview", func() {
			model := GetModelByName(mappings.Models, "gpt-4-preview")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-4-vision-preview", func() {
			model := GetModelByName(mappings.Models, "gpt-4-vision-preview")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model text-embedding-ada-002", func() {
			model := GetModelByName(mappings.Models, "text-embedding-ada-002")
			prepareAndInvokeModel(model, UrlPathEmbeddings, "", testID)
		})

		It("must redirect to model text-embedding-3-large", func() {
			model := GetModelByName(mappings.Models, "text-embedding-3-large")
			prepareAndInvokeModel(model, UrlPathEmbeddings, "", testID)
		})

		It("must redirect to model text-embedding-3-small", func() {
			model := GetModelByName(mappings.Models, "text-embedding-3-small")
			prepareAndInvokeModel(model, UrlPathEmbeddings, "", testID)
		})

		It("must redirect to model dall-e-3", func() {
			model := GetModelByName(mappings.Models, "dall-e-3")
			prepareAndInvokeModel(model, URLPathImageGenerations, "", testID)
		})

		It("must redirect to model gpt-image-1.5", func() {
			model := GetModelByName(mappings.Models, "gpt-image-1.5")
			prepareAndInvokeModel(model, URLPathImageGenerations, "", testID)
		})

		It("must redirect to model claude-3-haiku", func() {
			model := GetModelByName(mappings.Models, "claude-3-haiku")
			prepareAndInvokeModel(model, URLPathChatCompletions, `{"modelId":"anthropic.claude-3-haiku-20240307-v1:0"}`, testID)
		})

		It("must redirect to model gemini-1.5-pro", func() {
			model := GetModelByName(mappings.Models, "gemini-1.5-pro")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gemini-1.5-flash", func() {
			model := GetModelByName(mappings.Models, "gemini-1.5-flash")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gemini-2.0-flash", func() {
			model := GetModelByName(mappings.Models, "gemini-2.0-flash")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gemini-2.5-flash", func() {
			model := GetModelByName(mappings.Models, "gemini-2.5-flash")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gemini-2.5-flash-lite", func() {
			model := GetModelByName(mappings.Models, "gemini-2.5-flash-lite")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gemini-2.5-pro", func() {
			model := GetModelByName(mappings.Models, "gemini-2.5-pro")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model text-multilingual-embedding-002", func() {
			model := GetModelByName(mappings.Models, "text-multilingual-embedding-002")
			prepareAndInvokeModel(model, UrlPathEmbeddings, "", testID)
		})

		It("must redirect to model llama3-8b-instruct", func() {
			model := GetModelByName(mappings.Models, "llama3-8b-instruct")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-4o", func() {
			model := GetModelByName(mappings.Models, "gpt-4o")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-4o-mini", func() {
			model := GetModelByName(mappings.Models, "gpt-4o-mini")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model claude-3-5-haiku", func() {
			model := GetModelByName(mappings.Models, "claude-3-5-haiku")
			prepareAndInvokeModel(model, URLPathChatCompletions, `{"modelId":"anthropic.claude-3-5-haiku-20241022-v1:0"}`, testID)
		})

		It("must redirect to model claude-3-5-sonnet", func() {
			model := GetModelByName(mappings.Models, "claude-3-5-sonnet")
			prepareAndInvokeModel(model, URLPathChatCompletions, `{"modelId":"anthropic.claude-3-5-sonnet-20241022-v2:0"}`, testID)
		})

		It("must redirect to model claude-haiku-4-5", func() {
			model := GetModelByName(mappings.Models, "claude-haiku-4-5")
			prepareAndInvokeModel(model, URLPathChatCompletions, `{"modelId":"anthropic.claude-haiku-4-5-20251001-v1:0"}`, testID)
		})

		It("must redirect to model claude-sonnet-4-5", func() {
			model := GetModelByName(mappings.Models, "claude-sonnet-4-5")
			prepareAndInvokeModel(model, URLPathChatCompletions, `{"modelId":"anthropic.claude-sonnet-4-5-20250929-v1:0"}`, testID)
		})

		It("must redirect to model imagen-3", func() {
			model := GetModelByName(mappings.Models, "imagen-3")
			prepareAndInvokeModel(model, URLPathImageGenerations, "", testID)
		})

		It("must redirect to model imagen-3-fast", func() {
			model := GetModelByName(mappings.Models, "imagen-3-fast")
			prepareAndInvokeModel(model, URLPathImageGenerations, "", testID)
		})

		It("must redirect to model imagen-4.0", func() {
			model := GetModelByName(mappings.Models, "imagen-4.0")
			prepareAndInvokeModel(model, URLPathImageGenerations, "", testID)
		})

		It("must redirect to model imagen-4.0-fast", func() {
			model := GetModelByName(mappings.Models, "imagen-4.0-fast")
			prepareAndInvokeModel(model, URLPathImageGenerations, "", testID)
		})

		It("must redirect to model imagen-4.0-ultra", func() {
			model := GetModelByName(mappings.Models, "imagen-4.0-ultra")
			prepareAndInvokeModel(model, URLPathImageGenerations, "", testID)
		})

		It("must redirect to model gemini-3.1-flash-image-preview", func() {
			model := GetModelByName(mappings.Models, "gemini-3.1-flash-image-preview")
			prepareAndInvokeModel(model, URLPathGenerateContent, "", testID)
		})

		It("must redirect to model gemini-2.5-flash-image", func() {
			model := GetModelByName(mappings.Models, "gemini-2.5-flash-image")
			prepareAndInvokeModel(model, URLPathGenerateContent, "", testID)
		})

		It("must redirect to model titan-embed-text", func() {
			model := GetModelByName(mappings.Models, "titan-embed-text")
			prepareAndInvokeModel(model, UrlPathEmbeddings, "", testID)
		})

		It("must redirect to model gpt-5", func() {
			model := GetModelByName(mappings.Models, "gpt-5")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-5-mini", func() {
			model := GetModelByName(mappings.Models, "gpt-5-mini")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-5-nano", func() {
			model := GetModelByName(mappings.Models, "gpt-5-nano")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-5-chat", func() {
			model := GetModelByName(mappings.Models, "gpt-5-chat")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-4.1", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-4.1-mini", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1-mini")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-4.1-nano", func() {
			model := GetModelByName(mappings.Models, "gpt-4.1-nano")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-5.1", func() {
			model := GetModelByName(mappings.Models, "gpt-5.1")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gpt-5.2", func() {
			model := GetModelByName(mappings.Models, "gpt-5.2")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gemini-3.0-flash-preview", func() {
			model := GetModelByName(mappings.Models, "gemini-3.0-flash-preview")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gemini-3.0-pro-preview", func() {
			model := GetModelByName(mappings.Models, "gemini-3.0-pro-preview")
			prepareAndInvokeModel(model, URLPathChatCompletions, "", testID)
		})

		It("must redirect to model gemini-embedding-001", func() {
			model := GetModelByName(mappings.Models, "gemini-embedding-001")
			prepareAndInvokeModel(model, UrlPathEmbeddings, "", testID)
		})

		It("must redirect to model gpt-realtime", func() {
			model := GetModelByName(mappings.Models, "gpt-realtime")
			prepareAndInvokeModel(model, URLPathRealtimeClientSecrets, "", testID)
		})

		It("must redirect to model gpt-realtime-mini", func() {
			model := GetModelByName(mappings.Models, "gpt-realtime-mini")
			prepareAndInvokeModel(model, URLPathRealtimeClientSecrets, "", testID)
		})

		It("must redirect to model gpt-realtime-1.5", func() {
			model := GetModelByName(mappings.Models, "gpt-realtime-1.5")
			prepareAndInvokeModel(model, URLPathRealtimeClientSecrets, "", testID)
		})

	})
})

func prepareAndInvokeModel(model *Model, targetUrl, body, testID string) {
	CreateModelMockExpectation(mockServerURL, model, targetUrl, "{}", testID)
	ExpectModelCall(model, targetUrl, body, testID)
	ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
	DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
}
