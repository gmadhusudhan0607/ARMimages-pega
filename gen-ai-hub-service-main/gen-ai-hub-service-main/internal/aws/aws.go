/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package awsclient

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func GetSecretValue(ctx context.Context, awsConfig *aws.Config, secretArn string) (string, error) {

	//create a secrets manager client
	secretsManagerClient := secretsmanager.NewFromConfig(*awsConfig)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretArn),
	}

	result, err := secretsManagerClient.GetSecretValue(ctx, input)

	if err != nil {
		return "", err
	}

	return *result.SecretString, nil

}
