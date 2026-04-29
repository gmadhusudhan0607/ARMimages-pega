/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/mapping"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestHandleGetDefaultsRequest(t *testing.T) {
	// Setup logger context
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()
	ctx = WithLogger(ctx, logger)

	tests := []struct {
		name              string
		setupEnv          func()
		mockCredsProvider func() mapping.CredentialsProvider
		mockClientFactory func() mapping.ClientFactory
		expectedStatus    int
		checkBody         func(t *testing.T, body []byte)
	}{
		{
			name: "GenAI infra disabled",
			setupEnv: func() {
				t.Setenv("USE_GENAI_INFRA", "false")
			},
			mockCredsProvider: func() mapping.CredentialsProvider {
				return &mockCredentialsProvider{}
			},
			mockClientFactory: func() mapping.ClientFactory {
				return func(creds *aws.CredentialsCache, region string) mapping.SecretsManagerClient {
					return &mockSecretsManagerClient{}
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var config infra.DefaultModelConfig
				err := json.Unmarshal(body, &config)
				assert.NoError(t, err)
				assert.Equal(t, "", config.Fast)
				assert.Equal(t, "", config.Smart)
			},
		},
		{
			name: "Credentials error",
			setupEnv: func() {
				t.Setenv("USE_GENAI_INFRA", "true")
				t.Setenv("STAGE_NAME", "test")
				t.Setenv("SAX_CELL", "test-cell")
				t.Setenv("LLM_MODELS_REGION", "us-east-1")
			},
			mockCredsProvider: func() mapping.CredentialsProvider {
				mock := &mockCredentialsProvider{}
				mock.On("GetCredentials").Return(nil, errors.New("credential error"))
				return mock
			},
			mockClientFactory: func() mapping.ClientFactory {
				return func(creds *aws.CredentialsCache, region string) mapping.SecretsManagerClient {
					return &mockSecretsManagerClient{}
				}
			},
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body []byte) {
				var response map[string]string
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "Internal error occurred")
			},
		},
		{
			name: "Default model mapping error",
			setupEnv: func() {
				t.Setenv("USE_GENAI_INFRA", "true")
				t.Setenv("STAGE_NAME", "test")
				t.Setenv("SAX_CELL", "test-cell")
				t.Setenv("LLM_MODELS_REGION", "us-east-1")
			},
			mockCredsProvider: func() mapping.CredentialsProvider {
				mock := &mockCredentialsProvider{}
				mock.On("GetCredentials").Return(newMockCredentialsCache(), nil)
				return mock
			},
			mockClientFactory: func() mapping.ClientFactory {
				return func(creds *aws.CredentialsCache, region string) mapping.SecretsManagerClient {
					client := &mockSecretsManagerClient{}
					client.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).
						Return(nil, errors.New("failed to load mapping"))
					return client
				}
			},
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body []byte) {
				var response map[string]string
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "failed to load smart and fast defaults")
			},
		},
		{
			name: "Successful response with Pro field disabled (default)",
			setupEnv: func() {
				t.Setenv("USE_GENAI_INFRA", "true")
				t.Setenv("STAGE_NAME", "test")
				t.Setenv("SAX_CELL", "test-cell")
				t.Setenv("LLM_MODELS_REGION", "us-east-1")
				// ENABLE_PRO_MODEL_DEFAULT not set, defaults to false
			},
			mockCredsProvider: func() mapping.CredentialsProvider {
				mock := &mockCredentialsProvider{}
				mock.On("GetCredentials").Return(newMockCredentialsCache(), nil)
				return mock
			},
			mockClientFactory: func() mapping.ClientFactory {
				return func(creds *aws.CredentialsCache, region string) mapping.SecretsManagerClient {
					client := &mockSecretsManagerClient{}
					secretValue := `{"fast": "fast-model-name", "smart": "smart-model-name", "pro": "pro-model-name"}`
					client.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).
						Return(&secretsmanager.GetSecretValueOutput{
							SecretString: &secretValue,
						}, nil)
					return client
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				// When Pro is disabled, response should not include Pro field
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "fast-model-name", response["fast"])
				assert.Equal(t, "smart-model-name", response["smart"])
				assert.Nil(t, response["pro"], "Pro field should not be present when feature flag is disabled")
			},
		},
		{
			name: "Successful response with Pro field enabled",
			setupEnv: func() {
				t.Setenv("USE_GENAI_INFRA", "true")
				t.Setenv("STAGE_NAME", "test")
				t.Setenv("SAX_CELL", "test-cell")
				t.Setenv("LLM_MODELS_REGION", "us-east-1")
				t.Setenv("ENABLE_PRO_MODEL_DEFAULT", "true")
			},
			mockCredsProvider: func() mapping.CredentialsProvider {
				mock := &mockCredentialsProvider{}
				mock.On("GetCredentials").Return(newMockCredentialsCache(), nil)
				return mock
			},
			mockClientFactory: func() mapping.ClientFactory {
				return func(creds *aws.CredentialsCache, region string) mapping.SecretsManagerClient {
					client := &mockSecretsManagerClient{}
					secretValue := `{"fast": "fast-model-name", "smart": "smart-model-name", "pro": "pro-model-name"}`
					client.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).
						Return(&secretsmanager.GetSecretValueOutput{
							SecretString: &secretValue,
						}, nil)
					return client
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				// When Pro is enabled, all three fields should be present
				var config infra.DefaultModelConfig
				err := json.Unmarshal(body, &config)
				assert.NoError(t, err)
				assert.Equal(t, "fast-model-name", config.Fast)
				assert.Equal(t, "smart-model-name", config.Smart)
				assert.Equal(t, "pro-model-name", config.Pro)
			},
		},
		{
			name: "Pro field enabled but not in secret",
			setupEnv: func() {
				t.Setenv("USE_GENAI_INFRA", "true")
				t.Setenv("STAGE_NAME", "test")
				t.Setenv("SAX_CELL", "test-cell")
				t.Setenv("LLM_MODELS_REGION", "us-east-1")
				t.Setenv("ENABLE_PRO_MODEL_DEFAULT", "true")
			},
			mockCredsProvider: func() mapping.CredentialsProvider {
				mock := &mockCredentialsProvider{}
				mock.On("GetCredentials").Return(newMockCredentialsCache(), nil)
				return mock
			},
			mockClientFactory: func() mapping.ClientFactory {
				return func(creds *aws.CredentialsCache, region string) mapping.SecretsManagerClient {
					client := &mockSecretsManagerClient{}
					secretValue := `{"fast": "fast-model-name", "smart": "smart-model-name"}`
					client.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).
						Return(&secretsmanager.GetSecretValueOutput{
							SecretString: &secretValue,
						}, nil)
					return client
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				// When Pro is enabled but not in secret (empty string), Pro field should be omitted due to omitempty
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "fast-model-name", response["fast"])
				assert.Equal(t, "smart-model-name", response["smart"])
				_, hasProField := response["pro"]
				assert.False(t, hasProField, "Pro field should not be present when empty (omitempty)")
			},
		},
		{
			name: "Pro field disabled explicitly",
			setupEnv: func() {
				t.Setenv("USE_GENAI_INFRA", "true")
				t.Setenv("STAGE_NAME", "test")
				t.Setenv("SAX_CELL", "test-cell")
				t.Setenv("LLM_MODELS_REGION", "us-east-1")
				t.Setenv("ENABLE_PRO_MODEL_DEFAULT", "false")
			},
			mockCredsProvider: func() mapping.CredentialsProvider {
				mock := &mockCredentialsProvider{}
				mock.On("GetCredentials").Return(newMockCredentialsCache(), nil)
				return mock
			},
			mockClientFactory: func() mapping.ClientFactory {
				return func(creds *aws.CredentialsCache, region string) mapping.SecretsManagerClient {
					client := &mockSecretsManagerClient{}
					secretValue := `{"fast": "fast-model-name", "smart": "smart-model-name", "pro": "pro-model-name"}`
					client.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).
						Return(&secretsmanager.GetSecretValueOutput{
							SecretString: &secretValue,
						}, nil)
					return client
				}
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				// When Pro is explicitly disabled, response should not include Pro field
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "fast-model-name", response["fast"])
				assert.Equal(t, "smart-model-name", response["smart"])
				assert.Nil(t, response["pro"], "Pro field should not be present when feature flag is false")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			tt.setupEnv()

			// Create a new gin context for testing
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Create the handler and call it
			handler := HandleGetDefaultsRequest(ctx, tt.mockCredsProvider(), tt.mockClientFactory())
			handler(c)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check response body
			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

// Mock implementations
type mockCredentialsProvider struct {
	mock.Mock
}

func (m *mockCredentialsProvider) GetCredentials() (*aws.CredentialsCache, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aws.CredentialsCache), args.Error(1)
}

func newMockCredentialsCache() *aws.CredentialsCache {
	// Create and return an actual CredentialsCache instance
	return aws.NewCredentialsCache(
		credentials.NewStaticCredentialsProvider(
			"test-access-key",
			"test-secret-key",
			"test-session-token",
		),
	)
}

// Mock implementation of SecretsManagerClient
type mockSecretsManagerClient struct {
	mock.Mock
}

func (m *mockSecretsManagerClient) ListSecrets(
	ctx context.Context,
	params *secretsmanager.ListSecretsInput,
	optFns ...func(*secretsmanager.Options),
) (*secretsmanager.ListSecretsOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*secretsmanager.ListSecretsOutput), args.Error(1)
}

func (m *mockSecretsManagerClient) GetSecretValue(
	ctx context.Context,
	params *secretsmanager.GetSecretValueInput,
	optFns ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	args := m.Called(ctx, params, optFns)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

// WithLogger adds a logger to the context (replicating cntx.WithLogger functionality)
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	// Create a new context with the logger using the same key as in cntx package
	type loggerKeyType struct{}
	var loggerKey = &loggerKeyType{}
	return context.WithValue(ctx, loggerKey, logger)
}
