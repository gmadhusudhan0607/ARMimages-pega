package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	jsoniter "github.com/json-iterator/go"
)

type MockedSecretsManagerClient struct {
	Secret       *DBSecret
	SecretString string
	Error        error
}

func (m *MockedSecretsManagerClient) GetSecretValue(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Secret == nil {
		return &secretsmanager.GetSecretValueOutput{
			SecretString: &m.SecretString,
		}, nil
	}
	secretString, _ := jsoniter.MarshalToString(m.Secret)
	return &secretsmanager.GetSecretValueOutput{
		SecretString: &secretString,
	}, nil
}
