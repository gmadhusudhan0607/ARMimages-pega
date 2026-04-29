/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package types

import (
	"testing"
)

func TestInfrastructureConstants(t *testing.T) {
	tests := []struct {
		name     string
		infra    Infrastructure
		expected string
	}{
		{"AWS Infrastructure", InfrastructureAWS, "aws"},
		{"GCP Infrastructure", InfrastructureGCP, "gcp"},
		{"Azure Infrastructure", InfrastructureAzure, "azure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.infra) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.infra))
			}
		})
	}
}

func TestProviderConstants(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		expected string
	}{
		{"Google Provider", ProviderGoogle, "google"},
		{"Bedrock Provider", ProviderBedrock, "bedrock"},
		{"Anthropic Provider", ProviderAnthropic, "anthropic"},
		{"Meta Provider", ProviderMeta, "meta"},
		{"Amazon Provider", ProviderAmazon, "amazon"},
		{"Vertex Provider", ProviderVertex, "vertex"},
		{"Azure Provider", ProviderAzure, "azure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.provider) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.provider))
			}
		})
	}
}

func TestEndpointConstants(t *testing.T) {
	tests := []struct {
		name     string
		endpoint Endpoint
		expected string
	}{
		{"Chat Completions", EndpointChatCompletions, "chat/completions"},
		{"Embeddings", EndpointEmbeddings, "embeddings"},
		{"Images Generations", EndpointImagesGenerations, "images/generations"},
		{"Generate Images", EndpointGenerateImages, "generateImages"},
		{"Converse", EndpointConverse, "converse"},
		{"Converse Stream", EndpointConverseStream, "converse-stream"},
		{"Invoke", EndpointInvoke, "invoke"},
		{"Predict", EndpointPredict, "predict"},
		{"Invoke Stream", EndpointInvokeStream, "invoke-stream"},
		{"Generate Content", EndpointGenerateContent, "generateContent"},
		{"Invoke With Response Stream", EndpointInvokeWithResponseStream, "invoke-with-response-stream"},
		{"Stream Generate Content", EndpointStreamGenerateContent, "streamGenerateContent"},
		{"Responses", EndpointResponses, "v1/responses"},
		{"Realtime Client Secrets", EndpointRealtimeClientSecrets, "v1/realtime/client_secrets"},
		{"Realtime Calls", EndpointRealtimeCalls, "v1/realtime/calls"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.endpoint) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.endpoint))
			}
		})
	}
}

func TestCreatorConstants(t *testing.T) {
	tests := []struct {
		name     string
		creator  Creator
		expected string
	}{
		{"OpenAI Creator", CreatorOpenAI, "openai"},
		{"Google Creator", CreatorGoogle, "google"},
		{"Meta Creator", CreatorMeta, "meta"},
		{"Amazon Creator", CreatorAmazon, "amazon"},
		{"Anthropic Creator", CreatorAnthropic, "anthropic"},
		{"Bedrock Creator", CreatorBedrock, "bedrock"},
		{"Vertex Creator", CreatorVertex, "vertex"},
		{"Stability Creator", CreatorStability, "stability"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.creator) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.creator))
			}
		})
	}
}

func TestIsValidCreator(t *testing.T) {
	tests := []struct {
		name     string
		creator  Creator
		expected bool
	}{
		{"Valid OpenAI", CreatorOpenAI, true},
		{"Valid Google", CreatorGoogle, true},
		{"Valid Meta", CreatorMeta, true},
		{"Valid Amazon", CreatorAmazon, true},
		{"Valid Anthropic", CreatorAnthropic, true},
		{"Valid Bedrock", CreatorBedrock, true},
		{"Valid Vertex", CreatorVertex, true},
		{"Valid Stability", CreatorStability, true},
		{"Invalid Creator", Creator("invalid"), false},
		{"Empty Creator", Creator(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidCreator(tt.creator)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for creator %s", tt.expected, result, tt.creator)
			}
		})
	}
}

func TestModelKeyString(t *testing.T) {
	tests := []struct {
		name     string
		key      ModelKey
		expected string
	}{
		{
			name: "Complete ModelKey",
			key: ModelKey{
				Infrastructure: InfrastructureAWS,
				Provider:       ProviderBedrock,
				Creator:        CreatorAmazon,
				ModelName:      "nova-pro",
				Version:        "v1",
			},
			expected: "aws/bedrock/amazon/nova-pro/v1",
		},
		{
			name: "GCP ModelKey",
			key: ModelKey{
				Infrastructure: InfrastructureGCP,
				Provider:       ProviderVertex,
				Creator:        CreatorGoogle,
				ModelName:      "gemini-pro",
				Version:        "1.0",
			},
			expected: "gcp/vertex/google/gemini-pro/1.0",
		},
		{
			name: "Azure ModelKey",
			key: ModelKey{
				Infrastructure: InfrastructureAzure,
				Provider:       ProviderAzure,
				Creator:        CreatorOpenAI,
				ModelName:      "gpt-4",
				Version:        "0613",
			},
			expected: "azure/azure/openai/gpt-4/0613",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.key.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestModelGetModelKey(t *testing.T) {
	model := &Model{
		Infrastructure: InfrastructureAWS,
		Provider:       ProviderBedrock,
		Creator:        CreatorAmazon,
		Name:           "nova-pro",
		Version:        "v1",
	}

	expectedKey := ModelKey{
		Infrastructure: InfrastructureAWS,
		Provider:       ProviderBedrock,
		Creator:        CreatorAmazon,
		ModelName:      "nova-pro",
		Version:        "v1",
	}

	result := model.GetModelKey()
	if result != expectedKey {
		t.Errorf("Expected %+v, got %+v", expectedKey, result)
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		rawEndpoint string
		expected    Endpoint
		expectError bool
	}{
		{"Chat Completions", "chat/completions", EndpointChatCompletions, false},
		{"Chat Completions with trailing slash", "chat/completions/", EndpointChatCompletions, false},
		{"Embeddings", "embeddings", EndpointEmbeddings, false},
		{"Images Generations", "images/generations", EndpointImagesGenerations, false},
		{"Generate Images", "generateImages", EndpointGenerateImages, false},
		{"Converse", "converse", EndpointConverse, false},
		{"Converse Stream", "converse-stream", EndpointConverseStream, false},
		{"Invoke", "invoke", EndpointInvoke, false},
		{"Predict", "predict", EndpointPredict, false},
		{"Invoke Stream", "invoke-stream", EndpointInvokeStream, false},
		{"Generate Content", "generateContent", EndpointGenerateContent, false},
		{"Invoke With Response Stream", "invoke-with-response-stream", EndpointInvokeWithResponseStream, false},
		{"Stream Generate Content", "streamGenerateContent", EndpointStreamGenerateContent, false},
		{"Responses", "v1/responses", EndpointResponses, false},
		{"Realtime Client Secrets", "v1/realtime/client_secrets", EndpointRealtimeClientSecrets, false},
		{"Realtime Client Secrets with leading slash", "/v1/realtime/client_secrets", EndpointRealtimeClientSecrets, false},
		{"Realtime Calls", "v1/realtime/calls", EndpointRealtimeCalls, false},
		{"Realtime Calls with leading slash", "/v1/realtime/calls", EndpointRealtimeCalls, false},
		{"Unknown endpoint", "unknown/endpoint", "", true},
		{"Empty endpoint", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeEndpoint(tt.rawEndpoint)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for endpoint %s, but got none", tt.rawEndpoint)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for endpoint %s: %v", tt.rawEndpoint, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %s, got %s for endpoint %s", tt.expected, result, tt.rawEndpoint)
				}
			}
		})
	}
}

func TestParameterSpec(t *testing.T) {
	param := ParameterSpec{
		Title:       "Temperature",
		Description: "Controls randomness",
		Type:        "float",
		Default:     0.7,
		Maximum:     1.0,
		Minimum:     0.0,
		Required:    false,
	}

	if param.Title != "Temperature" {
		t.Errorf("Expected Title 'Temperature', got %s", param.Title)
	}
	if param.Description != "Controls randomness" {
		t.Errorf("Expected Description 'Controls randomness', got %s", param.Description)
	}
	if param.Type != "float" {
		t.Errorf("Expected Type 'float', got %s", param.Type)
	}
	if param.Default != 0.7 {
		t.Errorf("Expected Default 0.7, got %v", param.Default)
	}
	if param.Maximum != 1.0 {
		t.Errorf("Expected Maximum 1.0, got %v", param.Maximum)
	}
	if param.Minimum != 0.0 {
		t.Errorf("Expected Minimum 0.0, got %v", param.Minimum)
	}
	if param.Required != false {
		t.Errorf("Expected Required false, got %v", param.Required)
	}
}

func TestModelCapabilities(t *testing.T) {
	capabilities := ModelCapabilities{
		Features:         []string{"streaming", "functionCalling"},
		InputModalities:  []string{"text", "image"},
		OutputModalities: []string{"text"},
		MimeTypes:        []string{"image/png", "image/jpeg"},
	}

	if len(capabilities.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(capabilities.Features))
	}
	if capabilities.Features[0] != "streaming" {
		t.Errorf("Expected first feature 'streaming', got %s", capabilities.Features[0])
	}
	if capabilities.Features[1] != "functionCalling" {
		t.Errorf("Expected second feature 'functionCalling', got %s", capabilities.Features[1])
	}

	if len(capabilities.InputModalities) != 2 {
		t.Errorf("Expected 2 input modalities, got %d", len(capabilities.InputModalities))
	}
	if capabilities.InputModalities[0] != "text" {
		t.Errorf("Expected first input modality 'text', got %s", capabilities.InputModalities[0])
	}

	if len(capabilities.OutputModalities) != 1 {
		t.Errorf("Expected 1 output modality, got %d", len(capabilities.OutputModalities))
	}
	if capabilities.OutputModalities[0] != "text" {
		t.Errorf("Expected output modality 'text', got %s", capabilities.OutputModalities[0])
	}

	if len(capabilities.MimeTypes) != 2 {
		t.Errorf("Expected 2 mime types, got %d", len(capabilities.MimeTypes))
	}
}

func TestModelValidation(t *testing.T) {
	// Test creating a complete model
	model := &Model{
		KEY:                    "test-model-id",
		Name:                   "test-model",
		Version:                "v1",
		Label:                  "Test Model",
		FunctionalCapabilities: []FunctionalCapability{FunctionalCapabilityChatCompletion},
		Infrastructure:         InfrastructureAWS,
		Provider:               ProviderBedrock,
		Creator:                CreatorAmazon,
		Endpoints:              []Endpoint{"/chat/completions"},
		Capabilities: ModelCapabilities{
			Features:         []string{"streaming"},
			InputModalities:  []string{"text"},
			OutputModalities: []string{"text"},
		},
		Parameters: map[string]ParameterSpec{
			"temperature": {
				Title:       "Temperature",
				Description: "Controls randomness",
				Type:        "float",
				Default:     0.7,
				Required:    false,
			},
		},
	}

	// Verify all fields are set correctly
	if model.KEY != "test-model-id" {
		t.Errorf("Expected KEY 'test-model-id', got %s", model.KEY)
	}
	if model.Name != "test-model" {
		t.Errorf("Expected Name 'test-model', got %s", model.Name)
	}
	if model.Version != "v1" {
		t.Errorf("Expected Version 'v1', got %s", model.Version)
	}
	if model.Infrastructure != InfrastructureAWS {
		t.Errorf("Expected Infrastructure AWS, got %s", model.Infrastructure)
	}
	if model.Provider != ProviderBedrock {
		t.Errorf("Expected Provider Bedrock, got %s", model.Provider)
	}
	if model.Creator != CreatorAmazon {
		t.Errorf("Expected Creator Amazon, got %s", model.Creator)
	}
	if len(model.Endpoints) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(model.Endpoints))
	}
	if len(model.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(model.Parameters))
	}

	// Test GetModelKey method
	key := model.GetModelKey()
	expectedKeyString := "aws/bedrock/amazon/test-model/v1"
	if key.String() != expectedKeyString {
		t.Errorf("Expected key string %s, got %s", expectedKeyString, key.String())
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("Empty ModelKey String", func(t *testing.T) {
		key := ModelKey{}
		result := key.String()
		expected := "////"
		if result != expected {
			t.Errorf("Expected %s, got %s", expected, result)
		}
	})

	t.Run("NormalizeEndpoint with multiple slashes", func(t *testing.T) {
		result, err := NormalizeEndpoint("///chat/completions///")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result != EndpointChatCompletions {
			t.Errorf("Expected %s, got %s", EndpointChatCompletions, result)
		}
	})

	t.Run("IsValidCreator with special characters", func(t *testing.T) {
		result := IsValidCreator(Creator("open@ai"))
		if result {
			t.Errorf("Expected false for creator with special characters")
		}
	})
}
