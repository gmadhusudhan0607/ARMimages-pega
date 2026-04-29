/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package aws

import (
	"context"
	"fmt"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/goccy/go-json"
	"go.uber.org/zap"
)

var logger = log.GetNamedLogger("aws-secret")

// DBSecret is used to store username and password to postgres database
type DBSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SecretsManagerClientFactory interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// GetDBSecret creates a service root context with a logger attached.
func GetDBSecret(ctx context.Context, log *zap.Logger, sm SecretsManagerClientFactory, secretArn string) (*DBSecret, error) {
	secretStr, err := GetSecretAsString(ctx, log, sm, secretArn)
	if err != nil {
		log.Error("unable to read secret", zap.String("secretArn", secretArn), zap.Error(err))
		return nil, err
	}

	secret := &DBSecret{}
	if err := json.Unmarshal([]byte(secretStr), secret); err != nil {
		log.Error("unable to parse secret", zap.String("secretArn", secretArn), zap.Error(err))
		return nil, err
	}
	return secret, nil
}

func GetSecretAsString(ctx context.Context, log *zap.Logger, sm SecretsManagerClientFactory, secretArn string) (string, error) {
	s, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretArn),
	})
	if err != nil {
		log.Error("unable to read secret", zap.String("secretArn", secretArn), zap.Error(err))
		return "", err
	}
	return aws.ToString(s.SecretString), nil
}

func GetCredentials(ctx context.Context, sm SecretsManagerClientFactory) (user, password string, err error) {
	if helpers.IsDBLocal() {
		return helpers.GetEnvOrPanic("DB_USR"), helpers.GetEnvOrPanic("DB_PWD"), nil
	} else {
		secretArn := helpers.GetEnvOrPanic("DB_SECRET")
		secret, err := GetDBSecret(ctx, logger, sm, secretArn)
		if err != nil {
			panic(fmt.Sprintf("failed to read credentials from %s : %s", secretArn, err))
		}
		return secret.Username, secret.Password, nil
	}
}
