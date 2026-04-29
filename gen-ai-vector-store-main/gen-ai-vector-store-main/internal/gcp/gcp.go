/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package gcp

import (
	"context"
	"encoding/json"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1beta2"
	"cloud.google.com/go/secretmanager/apiv1beta2/secretmanagerpb"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"go.uber.org/zap"
)

var logger = log.GetNamedLogger("gcp-secret")

// DBSecret is used to store username and password to postgres database
type DBSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func GetCredentials(ctx context.Context) (user, password string, err error) {
	if helpers.IsDBLocal() {
		return helpers.GetEnvOrPanic("DB_USR"), helpers.GetEnvOrPanic("DB_PWD"), nil
	} else {
		secretVersion := helpers.GetEnvOrPanic("DB_SECRET")
		content, err := getSecretVersionData(secretVersion)
		if err != nil {
			panic(fmt.Sprintf("failed to read credentials from %s : %s", secretVersion, err))
		}

		secret := &DBSecret{}
		if err := json.Unmarshal(content, secret); err != nil {
			logger.Error("unable to parse secret", zap.String("secretVersion", secretVersion), zap.Error(err))
			return "", "", err
		}
		return secret.Username, secret.Password, nil
	}
}

func GetSaxCredentials(log *zap.Logger) (string, error) {
	secretName := helpers.GetEnvOrPanic("SAX_CLIENT_SECRET")
	secretNameWithVersion := fmt.Sprintf("%s/versions/latest", secretName)

	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secretmanager client: %v", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretNameWithVersion,
	}
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Error("unable to read secret", zap.String("secretName", secretName), zap.Error(err))
		return "", err
	}
	return string(result.Payload.Data), nil

}

func getSecretVersionData(version string) ([]byte, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %v", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{Name: version}
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to access secret version: %v", err)
	}
	return result.Payload.Data, nil
}
