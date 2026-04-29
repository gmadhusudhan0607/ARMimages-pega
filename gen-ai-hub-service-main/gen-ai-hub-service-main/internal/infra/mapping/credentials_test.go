/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package mapping

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/assert"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient/saxtypes"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/testutils"
)

// mockSTSClient simulates STS responses for testing.
type mockSTSClient struct {
	assumeRoleOutput *sts.AssumeRoleWithWebIdentityOutput
	assumeRoleError  error
}

func (m mockSTSClient) AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error) {
	if m.assumeRoleError != nil {
		return nil, m.assumeRoleError
	}
	return m.assumeRoleOutput, nil
}

func httpOkFunc(url, method string, header http.Header, body io.ReadCloser) (*http.Request, *http.Response, error) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"access_token": "test-token"}`)),
		Header:     header,
	}
	return nil, resp, nil
}

var saxConfigMock = saxtypes.SaxAuthClientConfig{
	ClientId:      "mockClientId",
	Scopes:        "mockScope",
	TokenEndpoint: "mockToken",
	PrivateKey:    base64.StdEncoding.EncodeToString(testutils.GeneratePrivateKey()),
}

var saxBadKeyConfigMock = saxtypes.SaxAuthClientConfig{
	ClientId:      "mockClientId",
	Scopes:        "mockScope",
	TokenEndpoint: "mockToken",
	PrivateKey:    "bad-key",
}

func TestAwsStsCredentialProvider_GetCredentials(t *testing.T) {
	type fields struct {
		stsClient     STSClient
		saxConfigPath string
		roleArn       string
		saxConfig     *saxtypes.SaxAuthClientConfig
	}
	creds := types.Credentials{
		AccessKeyId:     aws.String("mockAccessKeyId"),
		SecretAccessKey: aws.String("mockSecretAccessKey"),
		SessionToken:    aws.String("mockSessionToken"),
	}
	tests := []struct {
		name    string
		fields  fields
		want    *aws.CredentialsCache
		wantErr bool
	}{
		{
			name: "Valid credentials",
			fields: fields{
				stsClient: mockSTSClient{
					assumeRoleOutput: &sts.AssumeRoleWithWebIdentityOutput{
						Credentials: &creds,
					},
					assumeRoleError: nil,
				},
				saxConfigPath: "/valid/path/to/config",
				roleArn:       "arn:aws:iam::123456789012:role/test-role",
				saxConfig:     &saxConfigMock,
			},
			want: aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(
					*creds.AccessKeyId,
					*creds.SecretAccessKey,
					*creds.SessionToken),
			),
			wantErr: false,
		},
		{
			name: "Invalid config path",
			fields: fields{
				stsClient:     mockSTSClient{ /* mock implementation */ },
				saxConfigPath: "/invalid/path/to/config",
				roleArn:       "arn:aws:iam::123456789012:role/test-role",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "STS client error",
			fields: fields{
				stsClient: mockSTSClient{
					assumeRoleOutput: nil,
					assumeRoleError:  errors.New("mock STS error"),
				},
				saxConfigPath: "/valid/path/to/config",
				roleArn:       "arn:aws:iam::123456789012:role/test-role",
				saxConfig:     &saxConfigMock,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "PK decode error",
			fields: fields{
				stsClient: mockSTSClient{
					assumeRoleOutput: nil,
					assumeRoleError:  errors.New("mock STS error"),
				},
				saxConfigPath: "/valid/path/to/config",
				roleArn:       "arn:aws:iam::123456789012:role/test-role",
				saxConfig:     &saxBadKeyConfigMock,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, _ := json.Marshal(tt.fields.saxConfig)
			fs := &testutils.FileSystemMock{}
			fs.With("/valid/path/to/config", string(str))
			helperSuite.FileExists = fs.FileExists()
			helperSuite.FileReader = fs.FileReader()
			helperSuite.HttpCaller = httpOkFunc
			defer helperSuite.Reset()

			a := &AwsStsCredentialProvider{
				stsClient:     tt.fields.stsClient,
				saxConfigPath: tt.fields.saxConfigPath,
				roleArn:       tt.fields.roleArn,
			}
			got, err := a.GetCredentials()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCredentials() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewAwsCredentialProvider(t *testing.T) {
	tests := []struct {
		name string
		want *AwsStsCredentialProvider
		string
		saxConfigPath   string
		llmAccountId    string
		stageName       string
		saxCell         string
		llmModelsRegion string
	}{
		{
			name: "Valid environment variables",
			want: &AwsStsCredentialProvider{
				saxConfigPath: "/valid/path/to/config",
				roleArn:       "arn:aws:iam::123456789012:role/genai-oidcrole-get-secrets-local-test",
			},
			llmModelsRegion: "us-west-2",
			saxConfigPath:   "/valid/path/to/config",
			stageName:       "local",
			llmAccountId:    "123456789012",
			saxCell:         "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LLM_MODELS_REGION", tt.llmModelsRegion)
			os.Setenv("SAX_CONFIG_PATH", tt.saxConfigPath)
			os.Setenv("LLM_ACCOUNT_ID", tt.llmAccountId)
			os.Setenv("STAGE_NAME", tt.stageName)
			os.Setenv("SAX_CELL", tt.saxCell)
			assert.NotPanics(t, func() {
				got := NewAwsCredentialProvider()
				assert.NotNil(t, got)
				assert.Equal(t, tt.want.saxConfigPath, got.saxConfigPath)
				assert.Equal(t, tt.want.roleArn, got.roleArn)
			})
			os.Unsetenv("LLM_MODELS_REGION")
			os.Unsetenv("SAX_CONFIG_PATH")
			os.Unsetenv("LLM_ACCOUNT_ID")
			os.Unsetenv("STAGE_NAME")
			os.Unsetenv("SAX_CELL")
		})
	}
}
