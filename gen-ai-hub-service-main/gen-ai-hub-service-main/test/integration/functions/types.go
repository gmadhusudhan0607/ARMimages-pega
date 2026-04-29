//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package functions

type Header []string

type Expectation struct {
	Id          string `json:"id"`
	HttpRequest struct {
		Method  string            `json:"method,omitempty"`
		Path    string            `json:"path,omitempty"`
		Headers map[string]Header `json:"headers,omitempty"`
		Body    interface{}       `json:"body"`
	}
	HttpResponse struct {
		Headers    map[string]Header `json:"headers,omitempty"`
		Body       interface{}       `json:"body"`
		StatusCode int               `json:"statusCode"`
	} `json:"httpResponse"`
	Times struct {
		RemainingTimes int  `json:"remainingTimes"`
		Unlimited      bool `json:"unlimited"`
	} `json:"times"`
	TimeToLive struct {
		TimeUnit   string `json:"timeUnit"`
		TimeToLive int    `json:"timeToLive"`
		Unlimited  bool   `json:"unlimited"`
	} `json:"timeToLive"`
}

type Capabilities struct {
	Completions     bool `json:"completions,omitempty"`
	Embeddings      bool `json:"embeddings,omitempty"`
	ImageGeneration bool `json:"image-generation,omitempty"`
}

type Model struct {
	Name           string       `yaml:"name" binding:"required"`
	ModelId        string       `yaml:"modelId" binding:"required"`
	ModelUrl       string       `yaml:"modelUrl" binding:"required"`
	RedirectUrl    string       `yaml:"redirectUrl" binding:"required"`
	Capabilities   Capabilities `yaml:"capabilities"`
	Provider       string       `yaml:"provider" binding:"required"`
	Infrastructure string       `yaml:"infrastructure"`

	// Additional fields for test
	Expectation   *Expectation
	CoveredByTest bool
}

type GenAIInfraConfig struct {
	ModelMapping string `json:"ModelMapping"`
	ModelId      string `json:"ModelId"`
	ModelArn     string `json:"ModelArn"`
	OIDCRole     string `json:"OIDCIAMRoleArn"`
	Endpoint     string `json:"Endpoint"`
	Path         string `json:"Path"`
	Region       string `json:"Region"`

	Expectations []Expectation
}

type InfraMappings struct {
	Configs     []GenAIInfraConfig
	Expectation *Expectation
}

type Buddy struct {
	Name        string `yaml:"name" binding:"required"`
	BuddyUrl    string `yaml:"buddyUrl" binding:"required"`
	RedirectUrl string `yaml:"redirectUrl" binding:"required"`

	// Additional fields for test
	Expectation   *Expectation
	CoveredByTest bool
}

type Mappings struct {
	Models  []Model `yaml:"models" binding:"required"`
	Buddies []Buddy `yaml:"buddies" binding:"required"`
}

type ChatCompletionsResponseType struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
