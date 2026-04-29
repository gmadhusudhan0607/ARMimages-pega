//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package service_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	. "github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const URLPathChatCompletionsBedrock = "/chat/completions"

const AssumeRoleResponse = `
<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
	<AssumeRoleWithWebIdentityResult>
		<Audience>backing-services</Audience>
		<AssumedRoleUser>
			<AssumedRoleId>AROAQVIN3FT47557HRFU5:name2</AssumedRoleId>
			<Arn>arn:aws:sts::045663071481:assumed-role/genai-oidcrole-us-60et1/name2</Arn>
		</AssumedRoleUser>
		<Provider>arn:aws:iam::045663071481:oidc-provider/stg-fcp-us-1.oktapreview.com/oauth2/ausg5ldmi6IpvdpXX1d6</Provider>
		<Credentials>
			<AccessKeyId>ACCESS_KEY</AccessKeyId>
			<SecretAccessKey>SECRET_ACCESS_KEY</SecretAccessKey>
			<SessionToken>SESSION_TOKEN</SessionToken>
			<Expiration>2024-12-04T15:52:40Z</Expiration>
		</Credentials>
		<SubjectFromWebIdentityToken>0oaihrsvjtVKcN9541d7</SubjectFromWebIdentityToken>
	</AssumeRoleWithWebIdentityResult>
	<ResponseMetadata>
		<RequestId>a53c475f-7416-4690-b4ef-b8183447b6ba</RequestId>
	</ResponseMetadata>
</AssumeRoleWithWebIdentityResponse>
`

const ListSecretsResponse = `
{    "SecretList": [
        {
            "ARN": "arn:aws:secretsmanager:us-east-1:01234567890:secret:claude-3-haiku",
            "Name": "genai_infra/local/us/claude-3-haiku",
            "SecretVersionsToStages": {
                "terraform-20250422152706439300000002": [
                    "AWSCURRENT"
                ]
            }
        },
        {
            "ARN": "arn:aws:secretsmanager:us-east-1:01234567890:secret:titan-embed-text",
            "Name": "genai_infra/local/us/titan-embed-text",
            "SecretVersionsToStages": {
                "terraform-20250422152706439300000002": [
                    "AWSCURRENT"
                ]
            }
        },
    ]
}
`

const BedrockRuntimeConverseResponse = `
{
	"metrics": {
		"latencyMs":327
	},
	"output": {
		"message": {
			"content":[
				{
					"text": "Hello! How can I assist you today?"
				}],
			"role": "assistant"
		}
	},
	"stopReason": "end_turn",
	"usage": {
		"inputTokens": 10,
		"outputTokens": 12,
		"totalTokens": 22
	}
}`

var GetSecretResponse = `{
	"SecretString": %s
}`

var _ = Describe("Tests SVC:", Ordered, func() {

	var mappings Mappings
	var err error
	var testID string
	var secretsGenAiInfra []GenAIInfraConfig
	var infraMappings InfraMappings

	BeforeAll(func() {
		mappings, err = LoadMappingFromFile(mappingFileGenAiInfra)
		Expect(err).To(BeNil())
		secretsGenAiInfra, err = LoadMappingFromSecretsDir(
			fmt.Sprintf("%s/%s", dirSecretsGenAiInfra, "claude-3-haiku"),
			fmt.Sprintf("%s/%s", dirSecretsGenAiInfra, "titan-embed-text"),
		)
		infraMappings = InfraMappings{
			Configs: secretsGenAiInfra,
		}
		Expect(err).To(BeNil())
		Expect(secretsGenAiInfra).To(HaveLen(2))
		println(testID)
		ExpectUniqUrls(mappings)
	})

	AfterAll(func() {
		// Cleanup
		for _, model := range mappings.Models {
			DeleteModelExpectation(mockServerURL, model)
		}
		if infraMappings.Expectation != nil {
			DeleteMockServerExpectation(mockServerURL, infraMappings.Expectation.Id)
		}
	})

	BeforeEach(func() {
		testID = strings.ToLower(fmt.Sprintf("test-%s", RandStringRunes(10)))
	})

	// TODO: adopt Andrii suggest and do calls to different APIs automatically based on config file by parsing
	// 		 distribution/genai-awsbedrock-infra-sce/src/main/resources/metadata.json and reading TargetApi values
	_ = Context("calling model", func() {

		It("calls model mapped from GenAI Infra config using Converse API", Label("genai-infra"), func() {
			model := GetModelByName(mappings.Models, "claude-3-haiku")
			genaiInfraConfig := GetGenAiInfraByName(secretsGenAiInfra, "claude-3-haiku")

			CreateOktaTokenExpectation(mockServerURL, testID)

			// create expectation for the model that shouldn't be called from the mapping file
			CreateModelMockExpectation(mockServerURL, model, URLPathChatCompletionsBedrock, "{}", testID)

			// create expectation for STS Assume Role With WebIdentity call
			CreateAwsMockExpectation(mockServerURL, genaiInfraConfig, "/sts", "", strconv.Quote(AssumeRoleResponse))

			//escapedJSON := strings.ReplaceAll(ListSecretsResponse, `"`, `\"`)
			CreateAwsMockExpectation(mockServerURL, genaiInfraConfig, "/secretsmanager", "secretsmanager.ListSecrets", ListSecretsResponse)

			modelJson, _ := json.Marshal(genaiInfraConfig)
			j := fmt.Sprintf(GetSecretResponse, strconv.Quote(string(modelJson)))
			CreateAwsMockExpectation(mockServerURL, genaiInfraConfig, "/secretsmanager", "secretsmanager.GetSecretValue", j)

			time.Sleep(15000 * time.Millisecond)

			// create expectation for Bedrock Runtime Converse call
			bedrockPath := fmt.Sprintf("/model/%s/converse", url.PathEscape(genaiInfraConfig.ModelId))
			CreateAwsMockExpectation(mockServerURL, genaiInfraConfig, bedrockPath, "", BedrockRuntimeConverseResponse)

			// create expectation for Bedrock Runtime call
			ExpectModelCallWithJwt(model, URLPathChatCompletionsBedrock, "{}", testID)

			// assert mapping file model expectation of not being called
			ExpectExpectationMatchedForModel(mockServerURL, model, 0, 0)

			ExpectExpectationMatchedForAws(mockServerURL, genaiInfraConfig, 1, 2000)

			// Cleanup expectations
			DeleteMockServerExpectation(mockServerURL, model.Expectation.Id)
			for _, exp := range genaiInfraConfig.Expectations {
				DeleteMockServerExpectation(mockServerURL, exp.Id)
			}
		})
	})
})
