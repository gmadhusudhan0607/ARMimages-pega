/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package testutils

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// MockSTSClient is a mock implementation of the STSClient interface
type MockSTSClient struct {
	AssumeRoleOut *sts.AssumeRoleWithWebIdentityOutput
	Err           error
}

func (m *MockSTSClient) AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error) {
	return m.AssumeRoleOut, m.Err
}

// MockSecretsManagerClient is a mock implementation of the SecretsManagerClient interface
type MockSecretsManagerClient struct {
	ListSecretsOut    *secretsmanager.ListSecretsOutput
	ListSecretsErr    error
	GetSecretValueOut []*secretsmanager.GetSecretValueOutput
	GetSecretValueErr error
}

func (m *MockSecretsManagerClient) ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
	return m.ListSecretsOut, m.ListSecretsErr
}

func (m *MockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.GetSecretValueErr != nil {
		return nil, m.GetSecretValueErr
	}
	if len(m.GetSecretValueOut) == 0 {
		return nil, nil // Return nil if no secrets are available
	}
	if params.SecretId == nil || *params.SecretId == "" {
		return nil, nil // Return nil if SecretId is not provided
	}
	for _, secret := range m.GetSecretValueOut {
		if secret.Name != nil && *secret.Name == *params.SecretId {
			return secret, nil // Return the matching secret
		}
	}
	return nil, m.GetSecretValueErr

}

type TokenBody struct {
	AccessToken string `json:"access_token"`
}
