/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package mapping

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient/saxtypes"
)

// STSClient defines an interface for the AWS STS client
type STSClient interface {
	AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
}

func (a AwsStsCredentialProvider) LoadSaxConfig() (*saxtypes.SaxAuthClientConfig, error) {
	saxConfig := saxtypes.SaxAuthClientConfig{}

	// Check whether the genai-config-file exists
	if _, err := helperSuite.FileExists(a.saxConfigPath); err != nil {
		return &saxConfig, fmt.Errorf("sax config file does not exist: %w", err)
	}

	// Read the file contents
	content, err := helperSuite.FileReader(a.saxConfigPath)
	if err != nil {
		return &saxConfig, fmt.Errorf("failed to read Sax config file: %w", err)
	}

	err = json.Unmarshal(content, &saxConfig)
	if err != nil {
		return &saxConfig, fmt.Errorf("failed to unmarshal Sax config file: %w", err)
	}

	return &saxConfig, nil
}

// CredentialsProvider provides AWS credentials
type CredentialsProvider interface {
	GetCredentials() (*aws.CredentialsCache, error)
}

type AwsStsCredentialProvider struct {
	stsClient     STSClient
	saxConfigPath string
	roleArn       string
}

func (a *AwsStsCredentialProvider) GetCredentials() (*aws.CredentialsCache, error) {
	c, err := a.LoadSaxConfig()
	if err != nil {
		return nil, err
	}

	var pk []byte
	if pk, err = c.GetPrivateKeyPEMFormat(); err != nil {
		return nil, err
	}
	token, err := saxclient.GetJwtValidTo(c.ClientId, c.Scopes, c.TokenEndpoint, pk, time.Now().Add(time.Minute))
	if err != nil {
		return nil, err
	}

	roleArn := a.roleArn
	sessionName := helperSuite.RandStringRunes(10)
	assumeRoleInput := sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          &roleArn,
		RoleSessionName:  &sessionName,
		WebIdentityToken: &token,
	}

	result, err := a.stsClient.AssumeRoleWithWebIdentity(context.Background(), &assumeRoleInput)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	// Use the assumed role credentials to create a credential
	creds := aws.NewCredentialsCache(
		credentials.NewStaticCredentialsProvider(
			*result.Credentials.AccessKeyId,
			*result.Credentials.SecretAccessKey,
			*result.Credentials.SessionToken),
	)

	return creds, nil
}

func NewAwsCredentialProvider() *AwsStsCredentialProvider {
	llmsRegion := helperSuite.GetEnvOrPanic("LLM_MODELS_REGION")
	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(llmsRegion))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	awsPartition := "aws"
	if strings.Contains(llmsRegion, "gov") {
		awsPartition = "aws-us-gov"
	}

	roleArn := fmt.Sprintf("arn:%s:iam::%s:role/genai-oidcrole-get-secrets-%s-%s",
		awsPartition,
		helperSuite.GetEnvOrPanic("LLM_ACCOUNT_ID"),
		helperSuite.GetEnvOrPanic("STAGE_NAME"),
		helperSuite.GetEnvOrPanic("SAX_CELL"))

	return &AwsStsCredentialProvider{
		stsClient:     sts.NewFromConfig(awsCfg),
		saxConfigPath: helperSuite.GetEnvOrPanic("SAX_CONFIG_PATH"),
		roleArn:       roleArn,
	}

}
