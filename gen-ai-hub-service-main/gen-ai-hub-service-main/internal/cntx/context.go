/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package cntx

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient/saxtypes"
)

type loggerKeyType struct{}

var loggerKey = &loggerKeyType{}

type awsConfigKeyType struct{}

var awsConfigKey = &awsConfigKeyType{}

type platformTypeKeyType struct{}

var platformTypeKey = &platformTypeKeyType{}

type saxConfigKeyType struct{}

var saxConfigKey = &saxConfigKeyType{}

type saxConfigPathKeyType struct{}

var saxConfigPathKey = &saxConfigPathKeyType{}

type useSaxKeyType struct{}

var useSaxKey = &useSaxKeyType{}

type useGCPVertexKeyType struct{}

var useGCPVertexKey = &useGCPVertexKeyType{}

type useAzureGenAIURLKeyType struct{}

var useAzureGenAIURLKey = &useAzureGenAIURLKeyType{}

type azureGenAIURLKeyType struct{}

var azureGenAIKey = &azureGenAIURLKeyType{}

type useGenAIInfraKeyType struct{}

var useGenAIInfraKey = &useGenAIInfraKeyType{}

type serviceNameType struct{}

var ServiceNameKey = &serviceNameType{}

var infraModelsDir string

type ginContextKeyType struct{}

var ginContextKey = &ginContextKeyType{}

// ServiceContext creates a service root context with a logger attached.
func ServiceContext(name string) context.Context {
	ctx := context.Background()
	l := getLogger()

	platformType := helpers.GetEnvOrDefault("PLATFORM_TYPE", "infinity")
	ctx = context.WithValue(ctx, platformTypeKey, platformType)

	useSax := helpers.GetEnvOrFalse("USE_SAX")
	ctx = context.WithValue(ctx, useSaxKey, useSax)

	saxConfigPath := helpers.GetEnvOrDefault("SAX_CONFIG_PATH", "/genai-sax-config/genai-sax-config")
	ctx = context.WithValue(ctx, saxConfigPathKey, saxConfigPath)

	useGenAiInfra := helpers.GetEnvOrFalse("USE_GENAI_INFRA")
	ctx = context.WithValue(ctx, useGenAIInfraKey, useGenAiInfra)

	ctx = context.WithValue(ctx, ServiceNameKey, name)

	useGCPVertexURL := helpers.GetEnvOrDefault("DEMO_GCP_VERTEX_URL", "")
	useGCPVertex := useGCPVertexURL != ""
	ctx = context.WithValue(ctx, useGCPVertexKey, useGCPVertex)

	getAzureGenAIURL := helpers.GetEnvOrDefault("GENAI_URL", "")
	useAzureGenAIURL := getAzureGenAIURL != ""
	ctx = context.WithValue(ctx, useAzureGenAIURLKey, useAzureGenAIURL)
	ctx = context.WithValue(ctx, azureGenAIKey, getAzureGenAIURL)

	infraModelsDir = helpers.GetEnvOrDefault("GENAI_INFRA_MODELS_DIR", "/genai-infra-config")

	l.Sugar().Infof("Context configured with flags platformType:%s, useSax:%t, useGenAiInfra:%t",
		ctx.Value(platformTypeKey), ctx.Value(useSaxKey), ctx.Value(useGenAIInfraKey))
	if useGenAiInfra {
		l.Sugar().Infof("GenAI Infra Models mount point: :%s", infraModelsDir)
	}

	return context.WithValue(ctx, loggerKey, l.Named(name))
}

func ContextWithAwsConfig(ctx context.Context, awsConfig *aws.Config) context.Context {

	return context.WithValue(ctx, awsConfigKey, awsConfig)
}

func GetAwsConfigFromContext(ctx context.Context) *aws.Config {
	if c, ok := ctx.Value(awsConfigKey).(*aws.Config); ok {
		return c
	}
	return nil
}

func ContextWithSaxClientConfig(ctx context.Context, config *saxtypes.SaxAuthClientConfig) context.Context {
	return context.WithValue(ctx, saxConfigKey, config)
}

func GetSaxClientConfigFromContext(ctx context.Context) *saxtypes.SaxAuthClientConfig {
	if s, ok := ctx.Value(saxConfigKey).(*saxtypes.SaxAuthClientConfig); ok {
		return s
	}
	return nil
}

func GetSaxConfigPath(ctx context.Context) string {
	if path, ok := ctx.Value(saxConfigPathKey).(string); ok {
		return path
	}
	return "/genai-sax-config/genai-sax-config"
}

// LoggerFromContext returns the logger associated with this context, or a no-op Logger.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return l
	}
	return getLogger()
}

func getLogger() *zap.Logger {
	defaultLoggerOnce.Do(func() {
		logLevel := helpers.GetEnvOrDefault("LOG_LEVEL", "INFO")
		if strings.ToUpper(logLevel) == "DEBUG" {
			defaultLogger, _ = zap.NewDevelopment()
		} else {
			defaultLogger, _ = zap.NewProduction()
		}
	})
	return defaultLogger
}

var (
	defaultLogger     *zap.Logger
	defaultLoggerOnce sync.Once
)

func IsLaunchpadPlatform(ctx context.Context) bool {
	return ctx.Value(platformTypeKey) == "launchpad"
}

func IsInfinityPlatform(ctx context.Context) bool {
	return ctx.Value(platformTypeKey) == "infinity"
}

func IsUseSax(ctx context.Context) bool {
	return trueOrFalse(ctx, useSaxKey)
}

func IsUseGenAiInfraModels(ctx context.Context) bool {
	return trueOrFalse(ctx, useGenAIInfraKey)
}

func IsUseGCPVertex(ctx context.Context) bool {
	val, ok := ctx.Value(useGCPVertexKey).(bool)
	return ok && val
}

func IsUseAzureGenAIURL(ctx context.Context) bool {
	val, ok := ctx.Value(useAzureGenAIURLKey).(bool)
	return ok && val
}

func AzureGenAIURL(ctx context.Context) string {
	val, ok := ctx.Value(azureGenAIKey).(string)
	if !ok {
		return ""
	}
	return val
}

func ContextWithGinContext(ctx context.Context, c *gin.Context) context.Context {
	return context.WithValue(ctx, ginContextKey, c)
}

func GetGinContext(ctx context.Context) *gin.Context {
	if c, ok := ctx.Value(ginContextKey).(*gin.Context); ok {
		return c
	}
	return nil
}

func trueOrFalse(ctx context.Context, key interface{}) bool {
	if ctx.Value(key) == nil {
		return false
	}
	return ctx.Value(key).(bool)
}

// MergeServiceContext merges service context values into a request context
// This preserves configuration values from service startup while keeping request-specific values
func MergeServiceContext(serviceCtx, requestCtx context.Context) context.Context {
	ctx := requestCtx

	// Copy all service configuration keys to request context
	if val := serviceCtx.Value(ServiceNameKey); val != nil {
		ctx = context.WithValue(ctx, ServiceNameKey, val)
	}

	if val := serviceCtx.Value(platformTypeKey); val != nil {
		ctx = context.WithValue(ctx, platformTypeKey, val)
	}

	if val := serviceCtx.Value(useSaxKey); val != nil {
		ctx = context.WithValue(ctx, useSaxKey, val)
	}

	if val := serviceCtx.Value(useGenAIInfraKey); val != nil {
		ctx = context.WithValue(ctx, useGenAIInfraKey, val)
	}

	if val := serviceCtx.Value(useGCPVertexKey); val != nil {
		ctx = context.WithValue(ctx, useGCPVertexKey, val)
	}

	if val := serviceCtx.Value(useAzureGenAIURLKey); val != nil {
		ctx = context.WithValue(ctx, useAzureGenAIURLKey, val)
	}

	if val := serviceCtx.Value(azureGenAIKey); val != nil {
		ctx = context.WithValue(ctx, azureGenAIKey, val)
	}

	if val := serviceCtx.Value(loggerKey); val != nil {
		ctx = context.WithValue(ctx, loggerKey, val)
	}

	if val := serviceCtx.Value(awsConfigKey); val != nil {
		ctx = context.WithValue(ctx, awsConfigKey, val)
	}

	if val := serviceCtx.Value(saxConfigKey); val != nil {
		ctx = context.WithValue(ctx, saxConfigKey, val)
	}

	if val := serviceCtx.Value(saxConfigPathKey); val != nil {
		ctx = context.WithValue(ctx, saxConfigPathKey, val)
	}

	return ctx
}

func GetInfraModelsDir(ctx context.Context) string {
	return infraModelsDir
}

func IsLLMProviderConfigured(ctx context.Context, provider string) bool {
	enabledProviders := helpers.GetEnabledProviders(ctx)
	return slices.Contains(enabledProviders, provider)
}

func IssueSaxClientToken(ctx context.Context) (string, error) {

	s := GetSaxClientConfigFromContext(ctx)
	if s == nil {
		e := fmt.Errorf("failed to load Sax Client configuration")
		return "", e
	}

	var pk []byte
	var err error
	if pk, err = s.GetPrivateKeyPEMFormat(); err != nil {
		e := fmt.Errorf("failed to get private key PEM format: %s", err)
		return "", e
	}

	var saxJwt string
	if saxJwt, err = saxclient.GetJwtValidTo(s.ClientId, s.Scopes, s.TokenEndpoint, pk, time.Now().Add(time.Minute)); err != nil {
		e := fmt.Errorf("failed to get JWT token: %s", err)
		return "", e
	}
	return saxJwt, nil
}

// NewTestContext creates a test context with default values for testing
// This allows tests to run in parallel without environment variable conflicts
func NewTestContext(name string) context.Context {
	ctx := context.Background()
	l := getLogger()

	// Use default values for test context
	ctx = context.WithValue(ctx, platformTypeKey, "infinity")
	ctx = context.WithValue(ctx, useSaxKey, false)
	ctx = context.WithValue(ctx, saxConfigPathKey, "/genai-sax-config/genai-sax-config")
	ctx = context.WithValue(ctx, useGenAIInfraKey, false)

	infraModelsDir = helpers.GetEnvOrDefault("GENAI_INFRA_MODELS_DIR", "/genai-infra-config")

	return context.WithValue(ctx, loggerKey, l.Named(name))
}

// WithSaxConfigPath sets the SAX config path in a context
func WithSaxConfigPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, saxConfigPathKey, path)
}

// WithUseGenAIInfra sets the useGenAIInfra flag in a context
func WithUseGenAIInfra(ctx context.Context, useGenAiInfra bool) context.Context {
	return context.WithValue(ctx, useGenAIInfraKey, useGenAiInfra)
}

// WithAzureGenAIURL sets the Azure GenAI URL in a context
func WithAzureGenAIURL(ctx context.Context, url string) context.Context {
	return context.WithValue(ctx, azureGenAIKey, url)
}

// ContextWithLogger replaces the logger stored in the context.
// Intended for tests that need to capture or override log output.
func ContextWithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}
