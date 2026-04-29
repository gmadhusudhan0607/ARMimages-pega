//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package max_tokens

// requestModelRegistry holds all model configurations for request building
var requestModelRegistry = map[string]*RequestModelConfig{
	// GPT-3.5-Turbo (default/latest)
	"gpt-35-turbo-": {
		ModelName:    "gpt-35-turbo",
		ModelVersion: "",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  16385,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-3.5-Turbo-0301 (legacy, no function calling)
	"gpt-35-turbo-0301": {
		ModelName:    "gpt-35-turbo",
		ModelVersion: "0301",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     false,
			SupportsVision:    false,
			MaxContextTokens:  4097,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2023-05-15": {
				Version:         "2023-05-15",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user"},
			},
		},
		DefaultAPIVersion: "2023-05-15",
	},

	// GPT-3.5-Turbo-0613 (first with function calling)
	"gpt-35-turbo-0613": {
		ModelName:    "gpt-35-turbo",
		ModelVersion: "0613",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  4097,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2023-05-15": {
				Version:         "2023-05-15",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user"},
			},
			"2024-02-01": {
				Version:         "2024-02-01",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice"},
			},
		},
		DefaultAPIVersion: "2024-02-01",
	},

	// GPT-3.5-Turbo-1106 (improved instruction following)
	"gpt-35-turbo-1106": {
		ModelName:    "gpt-35-turbo",
		ModelVersion: "1106",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  16385,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2023-05-15": {
				Version:         "2023-05-15",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user"},
			},
			"2024-02-01": {
				Version:         "2024-02-01",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice"},
			},
			"2024-06-01": {
				Version:         "2024-06-01",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice"},
			},
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-3.5-Turbo-16k (legacy 16k context)
	"gpt-35-turbo-16k": {
		ModelName:    "gpt-35-turbo",
		ModelVersion: "16k",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  16385,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2023-05-15": {
				Version:         "2023-05-15",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user"},
			},
			"2024-02-01": {
				Version:         "2024-02-01",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice"},
			},
		},
		DefaultAPIVersion: "2024-02-01",
	},

	// GPT-3.5-Turbo-16k-0613 (specific 16k version)
	"gpt-35-turbo-16k-0613": {
		ModelName:    "gpt-35-turbo",
		ModelVersion: "16k-0613",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  16385,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2023-05-15": {
				Version:         "2023-05-15",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user"},
			},
			"2024-02-01": {
				Version:         "2024-02-01",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice"},
			},
		},
		DefaultAPIVersion: "2024-02-01",
	},

	// GPT-3.5-Turbo-0125
	"gpt-35-turbo-0125": {
		ModelName:    "gpt-35-turbo",
		ModelVersion: "0125",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  16384,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-4-0613
	"gpt-4-0613": {
		ModelName:    "gpt-4",
		ModelVersion: "0613",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  8192,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-4-0125-preview
	"gpt-4-0125-preview": {
		ModelName:    "gpt-4",
		ModelVersion: "0125-preview",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  16384,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-4-preview-1106
	"gpt-4-preview-1106": {
		ModelName:    "gpt-4-preview",
		ModelVersion: "1106",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  128000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-4-vision-preview-1106
	"gpt-4-vision-preview-1106": {
		ModelName:    "gpt-4-vision-preview",
		ModelVersion: "1106",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  128000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-4o-2024-11-20
	"gpt-4o-2024-11-20": {
		ModelName:    "gpt-4o",
		ModelVersion: "2024-11-20",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  128000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-4o-2024-08-06
	"gpt-4o-2024-08-06": {
		ModelName:    "gpt-4o",
		ModelVersion: "2024-08-06",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  128000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-4o-2024-05-13
	"gpt-4o-2024-05-13": {
		ModelName:    "gpt-4o",
		ModelVersion: "2024-05-13",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  128000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GPT-4o-mini-2024-07-18
	"gpt-4o-mini-2024-07-18": {
		ModelName:    "gpt-4o-mini",
		ModelVersion: "2024-07-18",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  128000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "stop", "presence_penalty", "frequency_penalty", "user", "tools", "tool_choice", "response_format"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// AWS Bedrock - Anthropic Claude models
	"claude-3-haiku-v1": {
		ModelName:    "claude-3-haiku",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  200000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"claude-3-5-sonnet-v2": {
		ModelName:    "claude-3-5-sonnet",
		ModelVersion: "v2",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  200000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"claude-3-5-haiku-v1": {
		ModelName:    "claude-3-5-haiku",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  200000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"claude-3-7-sonnet-v1": {
		ModelName:    "claude-3-7-sonnet",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  200000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// Claude Haiku 4.5 (default/latest)
	"claude-haiku-4-5": {
		ModelName:    "claude-haiku-4-5",
		ModelVersion: "",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  200000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// Claude Sonnet 4.5 (default/latest)
	"claude-sonnet-4-5": {
		ModelName:    "claude-sonnet-4-5",
		ModelVersion: "",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  200000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// AWS Bedrock - Amazon Nova models
	"nova-lite-v1": {
		ModelName:    "nova-lite",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     false,
			SupportsVision:    true,
			MaxContextTokens:  300000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"nova-micro-v1": {
		ModelName:    "nova-micro",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     false,
			SupportsVision:    false,
			MaxContextTokens:  128000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"nova-premier-v1": {
		ModelName:    "nova-premier",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  1000000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"nova-pro-v1": {
		ModelName:    "nova-pro",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  300000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"titan-embed-text-v2": {
		ModelName:    "titan-embed-text",
		ModelVersion: "v2",
		Capabilities: RequestCapabilities{
			SupportsStreaming: false,
			SupportsTools:     false,
			SupportsVision:    false,
			MaxContextTokens:  50000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"inputText"},
				SupportedFields: []string{"inputText", "dimensions", "normalize"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// AWS Bedrock - Meta Llama models
	"llama-3-2-90b-instruct-v1:0": {
		ModelName:    "llama-3-2-90b-instruct",
		ModelVersion: "v1:0",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     false,
			SupportsVision:    false,
			MaxContextTokens:  128000,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "top_k"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"llama3-8b-instruct-v1": {
		ModelName:    "llama3-8b-instruct",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  8192,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"messages"},
				SupportedFields: []string{"messages", "max_tokens", "temperature", "stream", "top_p", "top_k"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	// GCP Vertex - Google Gemini models
	"gemini-1.0-pro-v1": {
		ModelName:    "gemini-1.0-pro",
		ModelVersion: "v1",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    false,
			MaxContextTokens:  30720,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"contents"},
				SupportedFields: []string{"contents", "max_output_tokens", "temperature", "top_p", "top_k", "candidate_count", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"gemini-1.5-pro-002": {
		ModelName:    "gemini-1.5-pro",
		ModelVersion: "002",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  1048576,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"contents"},
				SupportedFields: []string{"contents", "max_output_tokens", "temperature", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"gemini-1.5-flash-002": {
		ModelName:    "gemini-1.5-flash",
		ModelVersion: "002",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  1048576,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"contents"},
				SupportedFields: []string{"contents", "max_output_tokens", "temperature", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"gemini-2.0-flash-001": {
		ModelName:    "gemini-2.0-flash",
		ModelVersion: "001",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  1048576,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"contents"},
				SupportedFields: []string{"contents", "max_output_tokens", "temperature", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"gemini-2.5-pro-001": {
		ModelName:    "gemini-2.5-pro",
		ModelVersion: "001",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  1048576,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"contents"},
				SupportedFields: []string{"contents", "max_output_tokens", "temperature", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"gemini-2.5-flash-001": {
		ModelName:    "gemini-2.5-flash",
		ModelVersion: "001",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  1048576,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"contents"},
				SupportedFields: []string{"contents", "max_output_tokens", "temperature", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},

	"gemini-2.5-flash-lite-001": {
		ModelName:    "gemini-2.5-flash-lite",
		ModelVersion: "001",
		Capabilities: RequestCapabilities{
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsVision:    true,
			MaxContextTokens:  1048576,
		},
		SupportedAPIVersions: map[string]*APIVersionInfo{
			"2024-10-21": {
				Version:         "2024-10-21",
				TokenLimitField: MaxTokensField,
				RequiredFields:  []string{"contents"},
				SupportedFields: []string{"contents", "max_output_tokens", "temperature", "top_p", "top_k", "stop_sequences"},
			},
		},
		DefaultAPIVersion: "2024-10-21",
	},
}

// GetRequestModelConfig retrieves a request model configuration by model name and version
func GetRequestModelConfig(modelName, modelVersion string) *RequestModelConfig {
	// Validate inputs - return nil for invalid inputs
	if modelName == "" {
		return nil
	}

	// Build the registry key by concatenating modelName and modelVersion
	registryKey := modelName + "-" + modelVersion
	config, exists := requestModelRegistry[registryKey]
	if !exists {
		return nil
	}

	// Return a copy to prevent modifications
	configCopy := *config
	return &configCopy
}
