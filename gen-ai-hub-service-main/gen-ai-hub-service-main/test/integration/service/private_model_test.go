//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package service_test

import (
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
)

var _ = Describe("Tests SVC with Private Models:", Ordered, func() {

	var mappings Mappings
	var privateModelMappings Mappings
	var err error
	var testID string

	BeforeAll(func() {

		// getting the default config mapping
		//-------------------------------------
		mappings, err = LoadMappingFromFile(mappingFilePrivateModel)
		Expect(err).To(BeNil())
		println(testID)

		// getting the private model config mapping
		//-----------------------------------------
		var fileList []string
		files, err := os.ReadDir(privateModelConfigPath)
		Expect(err).To(BeNil())

		for _, file := range files {
			if strings.HasPrefix(file.Name(), PrivateModelFilePrefix) {
				fileList = append(fileList, file.Name())
			}
		}

		for _, file := range fileList {

			absFilePath := fmt.Sprintf("%s/%s", privateModelConfigPath, file)
			content, err := os.ReadFile(absFilePath)
			Expect(err).To(BeNil())

			var modelList []Model

			err = yaml.Unmarshal(content, &modelList)
			Expect(err).To(BeNil())

			privateModelMappings.Models = append(privateModelMappings.Models, modelList...)
		}

	})

	AfterAll(func() {
		// Cleanup
		for _, model := range mappings.Models {
			if model.Expectation != nil && model.Expectation.Id != "" {
				DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
			}
		}
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
	})

	_ = Context("calling chat/completions and embeddings with a BYOM model)", func() {

		It("The private model config for the gpt-35-turbo model must be used", func() {
			model := GetModelByName(mappings.Models, "gpt-35-turbo")
			privateModel := GetModelByName(privateModelMappings.Models, "gpt-35-turbo")
			// getting the RedirectUrl from the private model config for setting the Mock expectation
			model.RedirectUrl = privateModel.RedirectUrl
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		It("The private model config for the gpt-4o-mini model must be used", func() {
			model := GetModelByName(mappings.Models, "gpt-4o-mini")
			privateModel := GetModelByName(privateModelMappings.Models, "gpt-4o-mini")
			// getting the RedirectUrl from the private model config for setting the Mock expectation
			model.RedirectUrl = privateModel.RedirectUrl
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		It("The private model config for the text-embedding-ada-002 model must be used", func() {
			model := GetModelByName(mappings.Models, "text-embedding-ada-002")
			privateModel := GetModelByName(privateModelMappings.Models, "text-embedding-ada-002")
			// getting the RedirectUrl from the private model config for setting the Mock expectation
			model.RedirectUrl = privateModel.RedirectUrl
			CreateModelMockExpectation(mockServerURL, model, UrlPathEmbeddings, textEmbeddingBody, testID)
			ExpectModelCall(model, UrlPathEmbeddings, textEmbeddingBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})

		It("The Default config for the gpt-4o model must be used", func() {
			model := GetModelByName(mappings.Models, "gpt-4o")
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectModelCall(model, URLPathChatCompletions, chatCompletionInBody, testID)
			ExpectExpectationMatchedForModel(mockServerURL, model, 1, 1)
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
		})
	})
})
