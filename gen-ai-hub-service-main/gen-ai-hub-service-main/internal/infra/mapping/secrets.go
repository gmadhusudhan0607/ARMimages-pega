/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package mapping

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
)

type SyncMappingStore struct {
	data []infra.ModelConfig
	mu   sync.RWMutex
}

func (mappings *SyncMappingStore) Read() []infra.ModelConfig {
	mappings.mu.RLock()
	defer mappings.mu.RUnlock()
	return mappings.data
}

func (mappings *SyncMappingStore) Write(s []infra.ModelConfig) {
	mappings.mu.Lock()
	defer mappings.mu.Unlock()
	mappings.data = s
}

func NewSyncMappingStore() *SyncMappingStore {
	return &SyncMappingStore{
		data: []infra.ModelConfig{},
	}
}

// SecretsManagerClient defines an interface for the AWS Secrets Manager client
type SecretsManagerClient interface {
	ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// ClientFactory is a function type that creates a SecretsManagerClient
type ClientFactory func(creds *aws.CredentialsCache, region string) SecretsManagerClient

func NewAuthenticatedClient(creds *aws.CredentialsCache, region string) SecretsManagerClient {

	// Create a new AWS config with the provided credentials
	cfg, _ := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(creds),
		config.WithRegion(region),
	)

	// Create a new Secrets Manager client with the provided credentials
	// It is needed to have a new Config object as parameter feed the credentiasl properly.
	return secretsmanager.NewFromConfig(cfg)
}

func ListInfraSecrets(c SecretsManagerClient, stage, saxCell string) ([]string, error) {
	secretPrefix := fmt.Sprintf("genai_infra/%s/%s/", stage, saxCell)

	var nextToken *string
	secretNames := []string{}

	for {
		secretsManagerListInput := &secretsmanager.ListSecretsInput{
			Filters: []types.Filter{
				{
					Key:    "name",
					Values: []string{secretPrefix},
				},
			},
			NextToken: nextToken,
		}

		result, err := c.ListSecrets(context.Background(), secretsManagerListInput)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range result.SecretList {
			if strings.HasPrefix(*secret.Name, secretPrefix) {
				secretNames = append(secretNames, *secret.Name)
			}
		}
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return secretNames, nil
}

func GetModelMappings(c SecretsManagerClient, secretNames []string) ([]infra.ModelConfig, error) {
	modelMappings := []infra.ModelConfig{}

	for _, secretName := range secretNames {
		newMap := infra.ModelConfig{}

		secretsManagerInput := &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		}

		result, err := c.GetSecretValue(context.TODO(), secretsManagerInput)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret value: %w", err)
		}

		if result == nil {
			return nil, fmt.Errorf("secret is nil for %s", secretName)
		}

		if result.SecretString == nil {
			return nil, fmt.Errorf("secret string is nil for %s", secretName)
		}

		err = json.Unmarshal([]byte(*result.SecretString), &newMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal model mappings for %s", secretName)
		}

		if newMap.Inactive {
			// Skip inactive mappings
			continue
		}
		modelMappings = append(modelMappings, newMap)
	}

	return modelMappings, nil
}

// ModelMappingService orchestrates the retrieval of model mappings
type ModelMappingService struct {
	stage              string
	saxCell            string
	awsRegion          string
	mappingStore       *SyncMappingStore
	secretLoader       SecretLoaderFunction
	credentialProvider CredentialsProvider
}

func NewModelMappingService(s *SyncMappingStore, f SecretLoaderFunction, c CredentialsProvider) *ModelMappingService {

	return &ModelMappingService{
		stage:              helperSuite.GetEnvOrPanic("STAGE_NAME"),
		saxCell:            helperSuite.GetEnvOrPanic("SAX_CELL"),
		awsRegion:          helperSuite.GetEnvOrPanic("LLM_MODELS_REGION"),
		mappingStore:       s,
		secretLoader:       f,
		credentialProvider: c,
	}
}

type SecretLoaderFunction func(string, string, SecretsManagerClient, *SyncMappingStore) error

func (s *ModelMappingService) Execute() error {
	cp := s.credentialProvider

	creds, err := cp.GetCredentials()
	if err != nil {
		return err
	}

	// Get the list of secrets
	secretsProvider := NewAuthenticatedClient(creds, s.awsRegion)
	err = s.secretLoader(s.stage, s.saxCell, secretsProvider, s.mappingStore)
	return err
}

// LoadData fetches and updates model mapping data
func LoadData(stage, saxCell string, c SecretsManagerClient, store *SyncMappingStore) error {

	secretNames, err := ListInfraSecrets(c, stage, saxCell)
	if err != nil {
		return err
	}

	// Get model mappings
	modelMappings, err := GetModelMappings(c, secretNames)
	if err != nil {
		return err
	}

	// Update the data
	store.Write(modelMappings)

	return nil
}

func LoadDefaultModelMapping(c SecretsManagerClient, stage, saxCell string) (*infra.DefaultModelConfig, error) {
	secretName := fmt.Sprintf("genai_infra/defaults/%s/%s/defaults", stage, saxCell)
	fmt.Printf("Fetching secret: %s\n", secretName)

	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}
	if strings.ContainsAny(secretName, "*") {
		return nil, fmt.Errorf("invalid character '*' in secret name: %s", secretName)
	}
	result, err := c.GetSecretValue(context.TODO(), input)

	log.Printf("Fetching secret: %s", secretName)

	if err != nil {
		return nil, fmt.Errorf("failed to get default model mapping: %w", err)
	}

	if result.SecretString == nil {
		return nil, fmt.Errorf("secret string is nil for %s", secretName)
	}

	var mapping infra.DefaultModelConfig
	if err := json.Unmarshal([]byte(*result.SecretString), &mapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default model mapping: %w", err)
	}

	return &mapping, nil
}
