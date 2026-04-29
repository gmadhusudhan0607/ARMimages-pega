/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package config

import (
	"context"
	"fmt"
	"os"

	awssecret "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/aws"
	gcpsecret "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/gcp"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type DatabaseType int

const (
	DatabaseTypeLocal DatabaseType = iota
	DatabaseTypeCloudAWS
	DatabaseTypeCloudGCP
)

const (
	// Multi-pod aware default: 20 x 3 pools per pod * 3 pods = 180 total connections
	// Safe for all AWS RDS instances (cost-optimized-large = 185 limit)
	defaultMaxConnections = 20
)

type DatabaseConfig struct {
	Type                  DatabaseType
	Host                  string
	Port                  string
	DbName                string
	User                  string
	Password              string
	CloudAccount          string
	CloudRegion           string
	CloudDBInstance       string
	AwsConfig             aws.Config
	CloudDbWithPrivateIP  bool
	MaxConnectionsDefault int64
	MaxConnectionEnvVar   string
}

func (dbCfg *DatabaseConfig) ForGeneric() *DatabaseConfig {
	copy := *dbCfg
	copy.MaxConnectionEnvVar = "MAX_CONNS_GENERIC"
	copy.MaxConnectionsDefault = 10

	return &copy
}

func (dbCfg *DatabaseConfig) ForIngestion() *DatabaseConfig {
	copy := *dbCfg
	copy.MaxConnectionEnvVar = "MAX_CONNS_INGESTION"
	copy.MaxConnectionsDefault = 30
	return &copy
}

func (dbCfg *DatabaseConfig) ForSearch() *DatabaseConfig {
	copy := *dbCfg
	copy.MaxConnectionEnvVar = "MAX_CONNS_SEARCH"
	copy.MaxConnectionsDefault = 20

	return &copy
}

func LoadDatabaseConfig() (*DatabaseConfig, error) {
	ctx := context.Background()
	var dbCfg *DatabaseConfig
	var err error
	dbType := getDatabaseType()
	switch dbType {
	case DatabaseTypeLocal:
		dbCfg, err = loadDBConfigLocal()
	case DatabaseTypeCloudGCP:
		dbCfg, err = loadDBConfigGCP(ctx)
	case DatabaseTypeCloudAWS:
		dbCfg, err = loadDBConfigAWS(ctx)
	default:
		panic(fmt.Sprintf("unknown database type: %d", dbType))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load database config: %w", err)
	}

	dbCfg.MaxConnectionEnvVar = "MAX_CONNS"
	dbCfg.MaxConnectionsDefault = defaultMaxConnections

	return dbCfg, nil
}

func loadDBConfigLocal() (*DatabaseConfig, error) {
	host, present := os.LookupEnv("DB_HOST")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_HOST' is required")
	}
	port, present := os.LookupEnv("DB_PORT")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_PORT' is required")
	}
	dbName, present := os.LookupEnv("DB_NAME")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_NAME' is required")
	}
	user, present := os.LookupEnv("DB_USR")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_USR' is required")
	}
	password, present := os.LookupEnv("DB_PWD")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_PWD' is required")
	}

	return &DatabaseConfig{
		Host:     host,
		Port:     port,
		DbName:   dbName,
		User:     user,
		Password: password,
		Type:     DatabaseTypeLocal,
	}, nil
}

func loadDBConfigAWS(ctx context.Context) (*DatabaseConfig, error) {

	host, present := os.LookupEnv("DB_HOST")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_HOST' is required")
	}
	port, present := os.LookupEnv("DB_PORT")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_PORT' is required")
	}
	dbName, present := os.LookupEnv("DB_NAME")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_NAME' is required")
	}
	secretARN, present := os.LookupEnv("DB_SECRET")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_SECRET' is required")
	}

	//retrieve region from DB_SECRET_ARN
	region, err := getRegionFromARN(secretARN)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve region from db secret: %w", err)
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		panic(fmt.Errorf("failed to initialize AWS config: %w", err))
	}
	user, password, err := awssecret.GetCredentials(ctx, secretsmanager.NewFromConfig(awsCfg))
	if err != nil {
		panic(fmt.Sprintf("failed to load AWS credentials: %s", err))
	}
	return &DatabaseConfig{
		Host:        host,
		Port:        port,
		DbName:      dbName,
		User:        user,
		Password:    password,
		Type:        DatabaseTypeCloudAWS,
		CloudRegion: region,
		AwsConfig:   awsCfg,
	}, nil
}

func loadDBConfigGCP(ctx context.Context) (*DatabaseConfig, error) {

	host, present := os.LookupEnv("DB_HOST")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_HOST' is required")
	}
	port, present := os.LookupEnv("DB_PORT")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_PORT' is required")
	}
	dbName, present := os.LookupEnv("DB_NAME")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_NAME' is required")
	}
	region, present := os.LookupEnv("REGION")
	if !present {
		return nil, fmt.Errorf("environment variable 'REGION' is required")
	}
	account, present := os.LookupEnv("ACCOUNT")
	if !present {
		return nil, fmt.Errorf("environment variable 'ACCOUNT' is required")
	}
	dbInstance, present := os.LookupEnv("DB_INSTANCE")
	if !present {
		return nil, fmt.Errorf("environment variable 'DB_INSTANCE' is required")
	}
	withPrivateIP := os.Getenv("DB_WITH_PRIVATE_IP") == "true"

	user, password, err := gcpsecret.GetCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load GCP credentials: %s", err)
	}

	return &DatabaseConfig{
		Host:                 host,
		Port:                 port,
		DbName:               dbName,
		User:                 user,
		Password:             password,
		Type:                 DatabaseTypeCloudGCP,
		CloudRegion:          region,
		CloudAccount:         account,
		CloudDBInstance:      dbInstance,
		CloudDbWithPrivateIP: withPrivateIP,
	}, nil
}

func (c *DatabaseConfig) ToConnString() string {
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s ",
		c.Host, c.Port, c.DbName, c.User, c.Password)
}

func (c *DatabaseConfig) ToConnStringMasked() string {
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s ",
		c.Host, c.Port, c.DbName, c.User, "******")
}

func getDatabaseType() DatabaseType {
	if helpers.GetEnvOrDefault("DB_LOCAL", "false") == "true" {
		return DatabaseTypeLocal
	}
	provider := helpers.GetEnvOrDefault("CLOUD_PROVIDER", "aws")
	switch provider {
	case "gcp":
		return DatabaseTypeCloudGCP
	default:
		return DatabaseTypeCloudAWS
	}
}

func getRegionFromARN(secretArn string) (string, error) {
	parsedArn, err := arn.Parse(secretArn)
	if err != nil {
		return "", fmt.Errorf("invalid ARN format: %s", secretArn)
	}
	return parsedArn.Region, nil
}
