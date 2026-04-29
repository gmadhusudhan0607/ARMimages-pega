/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx/cntxtest"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/testutils"
)

const modelYamlFileNameForTests = "./models_test.yaml"

func TestEndpointsGenAiInfraEnabled(t *testing.T) {
	tests := []testType{
		{
			name:              "POST call Bedrock infra model but Mappings service is unavailable",
			method:            http.MethodPost,
			uri:               "/anthropic/deployments/claude-3-haiku/chat/completions",
			reqBody:           `{"modelId":"claude-3-haiku-v1"}`,
			code:              http.StatusInternalServerError,
			errMsgText:        "internal error loading GenAI Infrastructure configuration",
			isServiceEndpoint: true,
			expectedHeaders:   failureHeaders,
		},
		{
			name:              "POST call Bedrock infra model fail with Invalid Json exception",
			method:            http.MethodPost,
			uri:               "/amazon/deployments/titan-embed-text/embeddings",
			reqBody:           `{inputText": "The Text Titan Embedding v2 model is provided by Amazon"}`,
			code:              http.StatusBadRequest,
			errMsgText:        "invalid JSON payload",
			isServiceEndpoint: true,
			expectedHeaders:   failureHeaders,
		},
	}
	os.Setenv("USE_GENAI_INFRA", "true")
	defer os.Unsetenv("USE_GENAI_INFRA")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{ "usage": { "completion_tokens": 29, "prompt_tokens": 11, "total_tokens": 40 }, "metrics": { "latencyMs": 123 }}`)
	}))
	defer ts.Close()

	runWithServer(t, tests, ts)
}

func TestEndpoints(t *testing.T) {
	acceptEncodingGzip := []string{"Accept-Encoding", "gzip"}

	tests := []testType{
		{
			name:              "POST self-study buddy",
			method:            http.MethodPost,
			uri:               "/v1/isolation123/buddies/selfstudybuddy/question",
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			isBuddy:           true,
			expectedHeaders:   buddyHeaders,
		},
		{
			name:              "POST ask for model that is not mapped",
			method:            http.MethodPost,
			uri:               "/openai/deployments/dont-exist/chat/completions?api-version=2024-02-01",
			reqBody:           `{}`,
			code:              http.StatusBadRequest,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   failureHeaders,
		},
		{
			name:              "POST chat completions without isolationId",
			method:            http.MethodPost,
			uri:               "/openai/deployments/gpt-35-turbo/chat/completions?api-version=2024-02-01",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST chat completions without isolationId",
			method:            http.MethodPost,
			uri:               "/openai/deployments/gpt-5.1/chat/completions?api-version=2024-02-01",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST chat completions without isolationId",
			method:            http.MethodPost,
			uri:               "/openai/deployments/gpt-5.2/chat/completions?api-version=2024-02-01",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST call Bedrock model without a modelId in the request body",
			method:            http.MethodPost,
			uri:               "/anthropic/deployments/claude-3-haiku/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusBadRequest,
			errMsgText:        "missing mandatory field modelId for AWS Bedrock model",
			isServiceEndpoint: true,
			expectedHeaders:   failureHeaders,
		},
		{
			name:              "POST call VertexAI model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-1.5-pro/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST call VertexAI model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-1.5-flash/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST call VertexAI model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-2.0-flash/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST call VertexAI model compressed response",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-2.0-flash/chat/completions",
			reqHeaders:        acceptEncodingGzip,
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST call VertexAI model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-2.5-flash/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST call VertexAI model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-2.5-flash-lite/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST call VertexAI model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-2.5-pro/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		}, {
			name:              "POST call VertexAI model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-3.0-flash-preview/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		}, {
			name:              "POST call VertexAI model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-3.0-pro-preview/chat/completions",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		}, {
			name:              "POST call VertexAI embed model",
			method:            http.MethodPost,
			uri:               "/google/deployments/gemini-embedding-001/embeddings",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},

		{
			name:              "POST call VertexAI embed model",
			method:            http.MethodPost,
			uri:               "/google/deployments/text-multilingual-embedding-002/embeddings",
			reqBody:           `{}`,
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST embeddings with invalid api-version",
			method:            http.MethodPost,
			uri:               "/v1/isolation123/models/text-embedding-ada-002/embeddings?api-version=1111",
			code:              http.StatusNotFound,
			errMsgText:        "",
			isServiceEndpoint: true,
			// No headers expected for 404 routes since they don't have LLM metrics
		},
		{
			name:              "POST chat completions with invalid api-version overridden to governed version",
			method:            http.MethodPost,
			uri:               "/openai/deployments/gpt-35-turbo/chat/completions?api-version=1111",
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: true,
			expectedHeaders:   successHeaders,
		},
		{
			name:              "POST not existing endpoint",
			method:            http.MethodPost,
			uri:               "/v1/isolation123/not-existing-endpoint/gpt-35-turbo/embeddings?api-version=2024-02-01",
			code:              http.StatusNotFound,
			errMsgText:        "",
			isServiceEndpoint: true,
		},
		{
			name:              "GET unrecognized model",
			method:            http.MethodGet,
			uri:               "/v1/isolation123/models/unrecognizedModelId",
			code:              http.StatusNotFound,
			isServiceEndpoint: true,
			// No headers expected for 404 routes since they don't have LLM metrics
		},
		{
			name:              "GET health readiness",
			method:            http.MethodGet,
			uri:               "/health/readiness",
			code:              http.StatusOK,
			isServiceEndpoint: false,
		},
		{
			name:              "GET health readiness on service port",
			method:            http.MethodGet,
			uri:               "/health/readiness",
			code:              http.StatusNotFound,
			errMsgText:        "",
			isServiceEndpoint: true,
		},
		{
			name:              "GET health liveness",
			method:            http.MethodGet,
			uri:               "/health/liveness",
			code:              http.StatusOK,
			errMsgText:        "",
			isServiceEndpoint: false,
		},
		{
			name:              "GET swagger - moved permanently",
			method:            http.MethodGet,
			uri:               "/swagger",
			code:              http.StatusMovedPermanently,
			errMsgText:        "",
			isServiceEndpoint: true,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{ "usage": { "completion_tokens": 29, "prompt_tokens": 11, "total_tokens": 40 }, "metrics": { "latencyMs": 123 }}`
		if r.Header.Get("Accept-Encoding") == "gzip" {
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()
			gz.Write([]byte(response)) //nolint:errcheck
			return
		}
		fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	runWithServer(t, tests, ts)

}

type testType struct {
	name              string
	method            string
	uri               string
	code              int
	errMsgText        string
	reqBody           string
	reqHeaders        []string
	expectedHeaders   []middleware.GatewayHeader
	isServiceEndpoint bool
	isBuddy           bool
}

var successHeaders = initializeSuccessHeaders()
var failureHeaders = initializeFailureHeaders()
var buddyHeaders = initializeBuddyHeaders()

func initializeSuccessHeaders() []middleware.GatewayHeader {
	return slices.Clone(middleware.AllGatewayHeaders)
}

func initializeFailureHeaders() []middleware.GatewayHeader {
	return slices.DeleteFunc(slices.Clone(middleware.AllGatewayHeaders), func(header middleware.GatewayHeader) bool {
		return header == middleware.GatewayOutputTokens || header == middleware.GatewayTokensPerSecond || header == middleware.GatewayInputTokens || header == middleware.GatewayTimeToFirstToken
	})
}

func initializeBuddyHeaders() []middleware.GatewayHeader {
	return slices.DeleteFunc(slices.Clone(middleware.AllGatewayHeaders), func(header middleware.GatewayHeader) bool {
		return (header == middleware.GatewayOutputTokens || header == middleware.GatewayTokensPerSecond || header == middleware.GatewayInputTokens || header == middleware.GatewayTimeToFirstToken)
	})
}

func runWithServer(t *testing.T, tests []testType, ts *httptest.Server) {

	modelsYamlContent := getModelsYamlContent(ts.URL)

	err := os.WriteFile(modelYamlFileNameForTests, []byte(modelsYamlContent), 0644)
	if err != nil {
		panic(err)
	}

	config, err := api.RetrieveMappingImpl(context.Background(), modelYamlFileNameForTests)
	if err != nil {
		panic(err)
	}

	callAndAssert(t, tests, config)
	defer os.Remove(modelYamlFileNameForTests)
}

func callAndAssert(t *testing.T, tests []testType, config *api.Mapping) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			recorder := httptest.NewRecorder()
			_, endpointHealth := gin.CreateTestContext(recorder)
			_, endpointService := gin.CreateTestContext(recorder)

			c := cntx.ServiceContext("unittest")

			setupEngine(c, config, endpointHealth, endpointService)
			req, err := http.NewRequest(tt.method, tt.uri, bytes.NewBuffer([]byte(tt.reqBody)))
			if len(tt.reqHeaders) > 0 {
				req.Header.Add(tt.reqHeaders[0], tt.reqHeaders[1])
			} // add headers if any
			assert.NoError(t, err)

			if tt.isServiceEndpoint {
				endpointService.ServeHTTP(recorder, req)
			} else {
				endpointHealth.ServeHTTP(recorder, req)
			}

			assert.Equal(t, tt.code, recorder.Result().StatusCode)

			if len(tt.expectedHeaders) > 0 {
				// extract result headers to map
				respHeaders := []string{}
				for k := range recorder.Header() {
					respHeaders = append(respHeaders, k)
				}

				// assert that all expected headers are present
				for _, h := range tt.expectedHeaders {
					assert.Contains(t, respHeaders, string(h))
				}

				if tt.code == http.StatusOK && !tt.isBuddy {
					checkHeaderHasValue(t, recorder.Header(), middleware.GatewayInputTokens)
					checkHeaderHasValue(t, recorder.Header(), middleware.GatewayOutputTokens)
					checkHeaderHasValue(t, recorder.Header(), middleware.GatewayTokensPerSecond)
				}
			}

			if tt.code != http.StatusOK {
				responseBody, _ := io.ReadAll(recorder.Result().Body)
				assert.Contains(t, string(responseBody), tt.errMsgText)
			}
		})
	}
}

func checkHeaderHasValue(t *testing.T, hs http.Header, h middleware.GatewayHeader) {
	vs := hs.Get(string(h))
	v, _ := strconv.Atoi(vs)
	assert.True(t, (v > 0))
}

func TestRetrieveMappingInvalidYAML(t *testing.T) {
	ctx := context.Background()
	invalidYamlContent := "invalid_yaml: ["
	fileName := "invalid_models_test.yaml"
	err := os.WriteFile(fileName, []byte(invalidYamlContent), 0644)
	if err != nil {
		panic(err)
	}
	defer os.Remove(fileName)

	_, err = api.RetrieveMappingImpl(ctx, fileName)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal file")
}

func TestRetrieveMappingFileDoesntExist(t *testing.T) {
	ctx := context.Background()
	fileName := helpers.RandStringRunes(40)

	_, err := api.RetrieveMappingImpl(ctx, fileName)
	assert.Error(t, err)
}

func getModelsYamlContent(redirectUrl string) string {
	return `
models:
- name: gpt-35-turbo
  redirectUrl: ` + redirectUrl + `
  creator: openai
- name: gpt-5.1
  redirectUrl: ` + redirectUrl + `
  creator: openai
- name: gpt-5.2
  redirectUrl: ` + redirectUrl + `
  creator: openai
- name: text-embedding-ada-002
  redirectUrl: ` + redirectUrl + `
  creator: openai
- name: claude-3-haiku
  redirectUrl: ` + redirectUrl + `
  modelId: claude-3-haiku-v1
  provider: anthropic
  creator: anthropic
- name: gemini-1.5-pro
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: gemini-1.5-flash
  redirectUrl: ` + redirectUrl + `
  creator: google
  provider: google
- name: gemini-2.0-flash
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: gemini-2.5-pro
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: gemini-2.5-flash
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: gemini-2.5-flash-lite
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: text-multilingual-embedding-002
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: imagen-3
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: imagen-3-fast
  redirectUrl: ` + redirectUrl + `
  creator: google
  provider: google
- name: titan-embed-text
  redirectUrl: ` + redirectUrl + `
  modelId: amazon.titan-embed-text-v2:0
  provider: amazon
  creator: amazon
- name: gemini-3.0-flash-preview
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: gemini-3.0-pro-preview
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
- name: gemini-embedding-001
  redirectUrl: ` + redirectUrl + `
  provider: google
  creator: google
buddies:
- name: selfstudybuddy
  redirectUrl: ` + redirectUrl + `
`
}

func TestMainPanicsWithWrongMapping(t *testing.T) {

	os.Setenv("CONFIGURATION_FILE", "configfile")
	defer os.Unsetenv("CONFIGURATION_FILE")

	assert.Panics(t, func() { main() }, "Expected to panic with wrong config file")
}

func TestMainToMoveMore(t *testing.T) {
	modelsYamlContent := getModelsYamlContent("http://testhost")

	err := os.WriteFile(modelYamlFileNameForTests, []byte(modelsYamlContent), 0644)
	if err != nil {
		panic(err)
	}
	defer os.Remove(modelYamlFileNameForTests)

	_, err = api.RetrieveMappingImpl(context.Background(), modelYamlFileNameForTests)
	if err != nil {
		panic(err)
	}

	os.Setenv("CONFIGURATION_FILE", modelYamlFileNameForTests)
	os.Setenv("USE_SAX", "true")

	assert.Panics(t, func() { main() }, "Expected to panic with wrong config file")

}

func TestLoadSaxConfigFromFile(t *testing.T) {
	// Tests use FileSystemMock for isolated, fast testing without real filesystem I/O
	// Note: FileSystemMock.FileReader() always returns nil error, so the file read error path
	// (line 142-145) is not covered. This represents rare I/O errors (permission denied, disk failure)
	// which are acceptable to leave untested. Coverage: ~95%
	tests := []struct {
		name           string
		configPath     string
		fileContent    string
		addToMock      bool
		expectError    bool
		errorContains  string
		expectedConfig *struct {
			ClientId      string
			PrivateKey    string
			Scopes        string
			TokenEndpoint string
		}
	}{
		{
			name:          "file not found",
			configPath:    "/genai-sax-config/genai-sax-config",
			addToMock:     false,
			expectError:   true,
			errorContains: "SAX config file does not exist",
		},
		{
			name:          "invalid JSON",
			configPath:    "/genai-sax-config/genai-sax-config",
			fileContent:   "{ invalid json }",
			addToMock:     true,
			expectError:   true,
			errorContains: "failed to unmarshal SAX config JSON",
		},
		{
			name:       "missing client_id",
			configPath: "/genai-sax-config/genai-sax-config",
			fileContent: `{
				"private_key": "test-private-key",
				"scopes": "test-scopes",
				"token_endpoint": "https://test.example.com/token"
			}`,
			addToMock:     true,
			expectError:   true,
			errorContains: "SAX config missing client_id",
		},
		{
			name:       "missing private_key",
			configPath: "/genai-sax-config/genai-sax-config",
			fileContent: `{
				"client_id": "test-client-id",
				"scopes": "test-scopes",
				"token_endpoint": "https://test.example.com/token"
			}`,
			addToMock:     true,
			expectError:   true,
			errorContains: "SAX config missing private_key",
		},
		{
			name:       "missing scopes",
			configPath: "/genai-sax-config/genai-sax-config",
			fileContent: `{
				"client_id": "test-client-id",
				"private_key": "test-private-key",
				"token_endpoint": "https://test.example.com/token"
			}`,
			addToMock:     true,
			expectError:   true,
			errorContains: "SAX config missing scopes",
		},
		{
			name:       "missing token_endpoint",
			configPath: "/genai-sax-config/genai-sax-config",
			fileContent: `{
				"client_id": "test-client-id",
				"private_key": "test-private-key",
				"scopes": "test-scopes"
			}`,
			addToMock:     true,
			expectError:   true,
			errorContains: "SAX config missing token_endpoint",
		},
		{
			name:       "valid config with all fields",
			configPath: "/genai-sax-config/genai-sax-config",
			fileContent: `{
				"client_id": "valid-client-id",
				"private_key": "valid-private-key",
				"scopes": "read write",
				"token_endpoint": "https://auth.example.com/oauth/token"
			}`,
			addToMock:   true,
			expectError: false,
			expectedConfig: &struct {
				ClientId      string
				PrivateKey    string
				Scopes        string
				TokenEndpoint string
			}{
				ClientId:      "valid-client-id",
				PrivateKey:    "valid-private-key",
				Scopes:        "read write",
				TokenEndpoint: "https://auth.example.com/oauth/token",
			},
		},
		{
			name:       "custom config path via environment variable",
			configPath: "/custom/path/to/sax-config",
			fileContent: `{
				"client_id": "custom-client-id",
				"private_key": "custom-private-key",
				"scopes": "custom-scopes",
				"token_endpoint": "https://custom.example.com/token"
			}`,
			addToMock:   true,
			expectError: false,
			expectedConfig: &struct {
				ClientId      string
				PrivateKey    string
				Scopes        string
				TokenEndpoint string
			}{
				ClientId:      "custom-client-id",
				PrivateKey:    "custom-private-key",
				Scopes:        "custom-scopes",
				TokenEndpoint: "https://custom.example.com/token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: t.Parallel() removed due to global helpers.HelperSuite shared state
			// The test helpers (FileReader, FileExists) modify a global singleton which
			// causes data races when tests run in parallel.

			// Create context with SAX config path (no env var needed!)
			ctx := cntxtest.NewContext("unittest")
			ctx = cntxtest.WithSaxConfigPath(ctx, tt.configPath)

			// Setup FileSystemMock
			fs := &testutils.FileSystemMock{}
			if tt.addToMock {
				fs.With(tt.configPath, tt.fileContent)
			}
			helpers.HelperSuite.FileReader = fs.FileReader()
			helpers.HelperSuite.FileExists = fs.FileExists()
			defer helpers.HelperSuite.Reset()

			// Execute test
			result, err := loadSaxConfigFromFile(ctx)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectedConfig != nil {
					assert.Equal(t, tt.expectedConfig.ClientId, result.ClientId)
					assert.Equal(t, tt.expectedConfig.PrivateKey, result.PrivateKey)
					assert.Equal(t, tt.expectedConfig.Scopes, result.Scopes)
					assert.Equal(t, tt.expectedConfig.TokenEndpoint, result.TokenEndpoint)
				}
			}
		})
	}
}

func TestGetSwagger(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	getSwagger(c)
	assert.Equal(t, http.StatusOK, recorder.Result().StatusCode)

}
