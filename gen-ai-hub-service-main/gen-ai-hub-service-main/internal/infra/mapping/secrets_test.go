/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package mapping

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/testutils"
)

type mockSecretsManagerClient struct {
	listSecretsOutput *secretsmanager.ListSecretsOutput
	listSecretsError  error
}

func (m *mockSecretsManagerClient) ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
	if m.listSecretsError != nil {
		return nil, m.listSecretsError
	}
	return m.listSecretsOutput, nil
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return nil, nil
}

func TestListInfraSecrets(t *testing.T) {

	creds := aws.NewCredentialsCache(
		credentials.NewStaticCredentialsProvider(
			"accessKey",
			"secretKey",
			"sesionToken"),
	)

	testCases := []struct {
		name                string
		stage               string
		saxCell             string
		mockListOutput      *secretsmanager.ListSecretsOutput
		mockListError       error
		expectedSecrets     []string
		expectError         bool
		awsClient           bool
		awsCustomEndpoint   string
		errorStringContains string
	}{
		{
			name:    "no secrets returned",
			stage:   "dev",
			saxCell: "cellA",
			mockListOutput: &secretsmanager.ListSecretsOutput{
				SecretList: []types.SecretListEntry{},
			},
			mockListError:   nil,
			expectedSecrets: []string{},
		},
		{
			name:    "some matching secrets",
			stage:   "dev",
			saxCell: "cellB",
			mockListOutput: &secretsmanager.ListSecretsOutput{
				SecretList: []types.SecretListEntry{
					{Name: aws.String("genai_infra/dev/cellB/secret1")},
					{Name: aws.String("genai_infra/dev/cellB/secret2")},
					{Name: aws.String("unrelated_secret")},
				},
			},
			mockListError:   nil,
			expectedSecrets: []string{"genai_infra/dev/cellB/secret1", "genai_infra/dev/cellB/secret2"},
			expectError:     false,
		},
		{
			name:            "error on list",
			stage:           "dev",
			saxCell:         "cellC",
			mockListOutput:  nil,
			mockListError:   fmt.Errorf("some error"),
			expectedSecrets: []string{},
			expectError:     true,
		},
		{
			name:                "test client url",
			stage:               "dev",
			saxCell:             "cellC",
			expectedSecrets:     []string{},
			awsClient:           true,
			awsCustomEndpoint:   "http://my-test-endpoint.com",
			expectError:         true,
			errorStringContains: "http://my-test-endpoint.com/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockSecretsManagerClient{
				listSecretsOutput: tc.mockListOutput,
				listSecretsError:  tc.mockListError,
			}

			var secretNames []string
			var err error
			if tc.awsClient {
				os.Setenv("AWS_IGNORE_CONFIGURED_ENDPOINT_URLS", "false")
				os.Setenv("AWS_ENDPOINT_URL_SECRETS_MANAGER", tc.awsCustomEndpoint)
				cfg, _ := config.LoadDefaultConfig(context.Background(),
					config.WithCredentialsProvider(creds),
					config.WithRegion("us-east-1"),
				)
				c := secretsmanager.NewFromConfig(cfg)
				secretNames, err = ListInfraSecrets(c, tc.stage, tc.saxCell)
				os.Unsetenv("AWS_IGNORE_CONFIGURED_ENDPOINT_URLS")
				os.Unsetenv("AWS_ENDPOINT_URL_SECRETS_MANAGER")
			} else {
				secretNames, err = ListInfraSecrets(mockClient, tc.stage, tc.saxCell)
			}
			if tc.expectError {
				assert.Error(t, err)
				if tc.errorStringContains != "" {
					assert.Contains(t, err.Error(), tc.errorStringContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedSecrets, secretNames)
			}
		})
	}
}

func TestSyncMappingStore_ReadWrite(t *testing.T) {
	testCases := []struct {
		name string
		data []infra.ModelConfig
	}{
		{
			name: "write and read empty string",
			data: []infra.ModelConfig{},
		},
		{
			name: "write and read normal string",
			data: []infra.ModelConfig{infra.ModelConfig{}},
		},
		{
			name: "write and read special characters",
			data: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := &SyncMappingStore{}
			store.Write(tc.data)
			got := store.Read()
			assert.Equal(t, tc.data, got)
		})
	}
}

func TestNewAuthenticatedClient(t *testing.T) {
	creds := aws.NewCredentialsCache(
		credentials.NewStaticCredentialsProvider(
			"accessKey",
			"secretKey",
			"sesionToken"),
	)
	region := "us-east-1"

	c := NewAuthenticatedClient(creds, region)
	assert.IsType(t, c, &secretsmanager.Client{})
}

func TestGetModelMappings(t *testing.T) {
	m1 := infra.ModelConfig{
		ModelMapping: "model1",
		Endpoint:     "endpoint1",
	}
	m2 := infra.ModelConfig{
		ModelMapping: "model2",
		Endpoint:     "endpoint2",
	}
	// convert to JSON
	//mappingJSON := testutils.ToJSONString(m)

	testCases := []struct {
		name           string
		secretNames    []string
		mockClient     *testutils.MockSecretsManagerClient
		expectedOutput []infra.ModelConfig
		expectError    bool
	}{
		{
			name:        "successful single mapping",
			secretNames: []string{"secret1"},
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueOut: []*secretsmanager.GetSecretValueOutput{
					{
						Name:         aws.String("secret1"),
						SecretString: aws.String(`{"ModelMapping":"model1","Endpoint":"endpoint1"}`),
					},
				},
			},
			expectedOutput: []infra.ModelConfig{m1},
			expectError:    false,
		},
		{
			name:        "multiple secrets",
			secretNames: []string{"secret1", "secret2"},
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueOut: []*secretsmanager.GetSecretValueOutput{
					{
						Name:         aws.String("secret1"),
						SecretString: aws.String(`{"ModelMapping":"model1","Endpoint":"endpoint1"}`),
					},
					{
						Name:         aws.String("secret2"),
						SecretString: aws.String(`{"ModelMapping":"model2","Endpoint":"endpoint2"}`),
					},
				},
			},
			expectedOutput: []infra.ModelConfig{m1, m2},
			expectError:    false,
		},
		{
			name:        "inactive secret is not returned",
			secretNames: []string{"secret1", "secret2"},
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueOut: []*secretsmanager.GetSecretValueOutput{
					{
						Name:         aws.String("secret1"),
						SecretString: aws.String(`{"ModelMapping":"model1","Endpoint":"endpoint1", "Inactive": true}`),
					},
					{
						Name:         aws.String("secret2"),
						SecretString: aws.String(`{"ModelMapping":"model2","Endpoint":"endpoint2"}`),
					},
				},
			},
			expectedOutput: []infra.ModelConfig{
				{
					ModelMapping: "model2",
					Endpoint:     "endpoint2",
					Inactive:     false,
				},
			},
			expectError: false,
		},
		{
			name:        "GetSecretValue error",
			secretNames: []string{"secret1"},
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueErr: fmt.Errorf("secret access denied"),
			},
			expectError: true,
		},
		{
			name:        "nil secret value",
			secretNames: []string{"secret1"},
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueOut: []*secretsmanager.GetSecretValueOutput{
					{
						SecretString: nil,
					},
				},
			},
			expectError: true,
		},
		{
			name:           "empty secret list",
			secretNames:    []string{},
			mockClient:     &testutils.MockSecretsManagerClient{},
			expectedOutput: []infra.ModelConfig{},
			expectError:    false,
		},
		{
			name:        "invalid JSON in secret",
			secretNames: []string{"secret1"},
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueOut: []*secretsmanager.GetSecretValueOutput{
					{
						SecretString: aws.String(`invalid json`),
					},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := GetModelMappings(tc.mockClient, tc.secretNames)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedOutput, output)
			}
		})
	}
}

func TestLoadDefaultModelMapping(t *testing.T) {
	testCases := []struct {
		name           string
		stage          string
		saxCell        string
		mockClient     *testutils.MockSecretsManagerClient
		expectedConfig *infra.DefaultModelConfig
		expectError    bool
		errorContains  string
	}{
		{
			name:    "successful mapping retrieval",
			stage:   "dev",
			saxCell: "cellA",
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueOut: []*secretsmanager.GetSecretValueOutput{
					{
						Name:         aws.String("genai_infra/defaults/dev/cellA/defaults"),
						SecretString: aws.String(`{"Smart":"smart-model","Fast":"fast-model"}`),
					},
				},
			},
			expectedConfig: &infra.DefaultModelConfig{
				Smart: "smart-model",
				Fast:  "fast-model",
			},
			expectError: false,
		},
		{
			name:    "GetSecretValue error",
			stage:   "dev",
			saxCell: "cellB",
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueErr: fmt.Errorf("access denied"),
			},
			expectedConfig: nil,
			expectError:    true,
			errorContains:  "failed to get default model mapping",
		},
		{
			name:    "nil secret string",
			stage:   "dev",
			saxCell: "cellC",
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueOut: []*secretsmanager.GetSecretValueOutput{
					{
						Name:         aws.String("genai_infra/defaults/dev/cellC/defaults"),
						SecretString: nil,
					},
				},
			},
			expectedConfig: nil,
			expectError:    true,
			errorContains:  "secret string is nil",
		},
		{
			name:    "invalid JSON in secret",
			stage:   "dev",
			saxCell: "cellD",
			mockClient: &testutils.MockSecretsManagerClient{
				GetSecretValueOut: []*secretsmanager.GetSecretValueOutput{
					{
						Name:         aws.String("genai_infra/defaults/dev/cellD/defaults"),
						SecretString: aws.String(`invalid json`),
					},
				},
			},
			expectedConfig: nil,
			expectError:    true,
			errorContains:  "failed to unmarshal default model mapping",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := LoadDefaultModelMapping(tc.mockClient, tc.stage, tc.saxCell)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedConfig, config)
			}
		})
	}
}
