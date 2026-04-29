/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/resolvers/target"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/apidocs"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api/vertex"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/health"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/otel"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/client"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/monitoring"
	requestmiddleware "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/saxclient/saxtypes"
)

const (
	servicePort     = "8080"
	healthcheckPort = "8082"
)

func init() {
	gin.DisableConsoleColor()
	gin.SetMode(gin.ReleaseMode)

}

func main() {

	contextName := "genai-hub-service"

	ctx := cntx.ServiceContext(contextName)
	l := cntx.LoggerFromContext(ctx).Sugar()
	defer l.Sync() //nolint:errcheck

	configFile := helpers.GetEnvOrPanic("CONFIGURATION_FILE")
	l.Infof("using mapping file: %s", configFile)
	mapping, err := api.RetrieveMappingImpl(ctx, configFile)
	if err != nil {
		panic(err)
	}

	// Initialize TargetResolver with configuration
	mappingEndpoint := helpers.GetEnvOrDefault("MAPPING_ENDPOINT", "")
	defaultsEndpoint := helpers.GetEnvOrDefault("MODELS_DEFAULTS_ENDPOINT", "")
	privateModelDir := helpers.GetEnvOrDefault("PRIVATE_MODEL_CONFIG_DIR", "/private-model-config")

	targetResolver, err := target.NewTargetResolver(configFile, mappingEndpoint, defaultsEndpoint, privateModelDir)
	if err != nil {
		l.Errorf("Failed to initialize TargetResolver: %v", err)
		panic(err)
	}
	l.Info("TargetResolver initialized successfully")

	// Set resolver in middleware
	requestmiddleware.SetGlobalTargetResolver(targetResolver)

	if cntx.IsUseSax(ctx) {
		// Load SAX config from file mounted by External Secrets Operator
		saxClientConfig, err := loadSaxConfigFromFile(ctx)
		if err != nil {
			l.Errorf("Error loading SaxConfiguration from file: %s", err.Error())
			panic(err)
		}
		l.Debugf("SaxConfig loaded from file successfully")
		ctx = cntx.ContextWithSaxClientConfig(ctx, saxClientConfig)
	}

	tp, err := otel.InitTracer(ctx)
	if err != nil {
		panic(err)
	}
	defer tp.Shutdown(ctx) //nolint:errcheck

	g := errgroup.Group{}

	healthEngine := gin.New()
	serviceEngine := gin.New()

	setupEngine(ctx, mapping, healthEngine, serviceEngine)

	defer l.Sync() //nolint:errcheck

	g.Go(func() error {
		// run in a separate goroutine to have /health endpoint exposed on 8082 port by default or check env variable for local testing
		port := fmt.Sprintf(":%s", helpers.GetEnvOrDefault("SERVICE_HEALTHCHECK_PORT", healthcheckPort))
		l.Infof("running service healthcheck on %s", port)
		return healthEngine.Run(port)
	})

	g.Go(func() error {
		// run on a default port - 8080 or check env variable for local testing
		port := fmt.Sprintf(":%s", helpers.GetEnvOrDefault("SERVICE_PORT", servicePort))
		l.Infof("running service on %s", port)
		return serviceEngine.Run(port)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

// loadSaxConfigFromFile loads SAX configuration from file mounted by External Secrets Operator
func loadSaxConfigFromFile(ctx context.Context) (*saxtypes.SaxAuthClientConfig, error) {
	l := cntx.LoggerFromContext(ctx).Sugar()

	// Get SAX config path from context (set at service startup)
	saxConfigPath := cntx.GetSaxConfigPath(ctx)
	l.Debugf("Loading SAX config from file: %s", saxConfigPath)

	// Check if file exists
	if _, err := helpers.HelperSuite.FileExists(saxConfigPath); err != nil {
		return nil, fmt.Errorf("SAX config file does not exist at %s: %w", saxConfigPath, err)
	}

	// Read file contents
	content, err := helpers.HelperSuite.FileReader(saxConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SAX config file: %w", err)
	}

	// Unmarshal JSON
	var saxConfig saxtypes.SaxAuthClientConfig
	if err := json.Unmarshal(content, &saxConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SAX config JSON: %w", err)
	}

	// Validate required fields
	if saxConfig.ClientId == "" {
		return nil, fmt.Errorf("SAX config missing client_id")
	}
	if saxConfig.PrivateKey == "" {
		return nil, fmt.Errorf("SAX config missing private_key")
	}
	if saxConfig.Scopes == "" {
		return nil, fmt.Errorf("SAX config missing scopes")
	}
	if saxConfig.TokenEndpoint == "" {
		return nil, fmt.Errorf("SAX config missing token_endpoint")
	}

	return &saxConfig, nil
}

func setupEngine(ctx context.Context, mapping *api.Mapping, healthEngine *gin.Engine, serviceEngine *gin.Engine) {
	l := cntx.LoggerFromContext(ctx)
	defer l.Sync() //nolint:errcheck

	serviceEngine.NoRoute(middleware.HttpMetricsMiddleware(ctx), func(c *gin.Context) {
		l.Sugar().Infof("route not found: %s", c.Request.RequestURI)
		c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": "404 page not found"})
	})

	healthEngine.Use(
		ginzap.Ginzap(l.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true), // to decrease bloated logs caused by liveness/readiness checks
		ginzap.RecoveryWithZap(l, true),
	)

	healthEngine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	debug := healthEngine.Group("/debug/pprof")
	{
		debug.GET("/", gin.WrapF(pprof.Index))
		debug.GET("/heap", gin.WrapH(pprof.Handler("heap")))
		debug.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		debug.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
		debug.GET("/block", gin.WrapH(pprof.Handler("block")))
		debug.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
		debug.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
		debug.GET("/cmdline", gin.WrapF(pprof.Cmdline))
		debug.GET("/profile", gin.WrapF(pprof.Profile))
		debug.GET("/symbol", gin.WrapF(pprof.Symbol))
		debug.GET("/trace", gin.WrapF(pprof.Trace))
	}

	h := healthEngine.Group("/health")
	{
		h.GET("/liveness", health.GetLiveness)
		if cntx.IsUseGenAiInfraModels(ctx) && helpers.HelperSuite.GetEnvOrFalse("USE_AUTO_MAPPING") {
			h.GET("/readiness", health.GetReadinessDependingOnMappings(ctx, infra.GetInfraModelsForContext))
		} else {
			h.GET("/readiness", health.GetReadiness)
		}
	}

	serviceEngine.Use(
		otelgin.Middleware("service"),
		middleware.RequestLoggerMiddleware(ctx),
		middleware.ResponseLoggerMiddleware(ctx),
		ginzap.Ginzap(l.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true),
		ginzap.RecoveryWithZap(l, true),
		middleware.HttpMetricsMiddleware(ctx),
		requestmiddleware.RequestModificationMiddleware(ctx),
	)

	serviceEngine.GET("/", getSwagger)
	serviceEngine.StaticFile("./swagger/spec.yaml", "./apidocs/spec.yaml")
	serviceEngine.StaticFile("./swagger/", "./apidocs/static/swagger/views/swagger-ui/index.html")

	// Create model list cache (populated lazily on the first /models request)
	checker := &api.DefaultContextChecker{}
	cacheTTL := api.CacheTTLFromEnv(cntx.LoggerFromContext(ctx).Sugar())
	modelCache := api.NewModelListCache(ctx, checker, cacheTTL)

	serviceEngine.GET("/models", middleware.UasValidator(ctx), middleware.SaxRequestEnrichment(ctx), api.HandleCachedGetModelsRequest(ctx, checker, modelCache))
	serviceEngine.GET("/models/defaults", middleware.UasValidator(ctx), middleware.SaxRequestEnrichment(ctx), api.HandleCachedGetDefaultModelsRequest(ctx, checker, modelCache))

	// Endpoint https://{api-url}/openai/deployments/{modelId}/chat/completions?api-version={api-version}
	// was introduced in Pega Infinity 23 and extended with further explicit endpoints for Infinity 24.1
	// e.g. https://{api-url}/openai/deployments/{modelId}/embeddings?api-version={api-version}
	o := serviceEngine.Group("/openai",
		middleware.ProviderEnabled("Azure"),
		middleware.UasValidator(ctx),
		middleware.SaxRequestEnrichment(ctx),
		middleware.LLMMetricsMiddleware(ctx),
		monitoring.RequestReporter(ctx))
	{
		o.POST("/deployments/:modelId/images/generations", api.HandleImageGenerationRequest(ctx, mapping))
		o.POST("/deployments/:modelId/chat/completions", api.HandleChatCompletionRequest(ctx, mapping))
		o.POST("/deployments/:modelId/embeddings", api.HandleEmbeddingsRequest(ctx, mapping))
		o.POST("/deployments/:modelId/v1/realtime/client_secrets", api.HandleRealtimeProxyRequest(ctx))
		o.POST("/deployments/:modelId/v1/realtime/calls", api.HandleRealtimeProxyRequest(ctx))
	}
	//Endpoint {api-url}/v1/{isolationId}/buddies/selfstudybuddy/question
	b := serviceEngine.Group("/v1/:isolationId/buddies", middleware.UasValidator(ctx), middleware.SaxRequestEnrichment(ctx), middleware.LLMMetricsMiddleware(ctx))
	{
		b.POST("/:buddyId/question", api.HandleBuddyRequest(ctx, mapping))
	}

	// AWS Bedrock model deployments
	{
		setupBedrockRoutes(serviceEngine, ctx, mapping, "anthropic", cntx.IsUseGenAiInfraModels(ctx))
		setupBedrockRoutes(serviceEngine, ctx, mapping, "meta", cntx.IsUseGenAiInfraModels(ctx))
		setupBedrockRoutes(serviceEngine, ctx, mapping, "amazon", cntx.IsUseGenAiInfraModels(ctx))
		setupBedrockRoutes(serviceEngine, ctx, mapping, "mistral", cntx.IsUseGenAiInfraModels(ctx))
	}

	google := serviceEngine.Group("/google", middleware.ProviderEnabled("Vertex"), middleware.UasValidator(ctx), middleware.SaxRequestEnrichment(ctx), middleware.LLMMetricsMiddleware(ctx), monitoring.RequestReporter(ctx))
	{
		// Simple forwarder pattern: Delegates GCP-specific API logic to cloud function
		// Cloud function handles OpenAI SDK compatibility and native Vertex AI endpoints
		google.POST("/deployments/:modelId/chat/completions", api.HandleExperimentalModelChatCompletionRequest(ctx, mapping))
		google.POST("/deployments/:modelId/embeddings", api.HandleExperimentalModelChatCompletionRequest(ctx, mapping))

		// Gemini native image generation via generateContent endpoint
		google.POST("/deployments/:modelId/generateContent", api.HandleExperimentalModelChatCompletionRequest(ctx, mapping))

		// Dedicated middleware chain: Complex request validation and transformation for Imagen
		// Requires specialized handling that differs from standard Gemini models
		google.POST("/deployments/:modelId/images/generations",
			vertex.CheckVertexImagenRequest(ctx),          // Validate request structure
			vertex.SelectImagenModelMapping(ctx, mapping), // Resolve model configuration
			vertex.CallImagenApi(ctx),                     // Execute Imagen API call
		)
	}
}

func setupBedrockRoutes(serviceEngine *gin.Engine, ctx context.Context, mapping *api.Mapping, groupName string, useGenAiInfraModels bool) {
	group := serviceEngine.Group(groupName, middleware.ProviderEnabled("Bedrock"), middleware.UasValidator(ctx), middleware.SaxRequestEnrichment(ctx), middleware.LLMMetricsMiddleware(ctx), monitoring.RequestReporter(ctx))

	endpoint := "/deployments/:modelId/*targetApi"

	// call models form mapping file (DemoAwsBedrockEndpoint input) - already deprecate and to be removed
	if !useGenAiInfraModels {
		group.POST(endpoint,
			api.ValidateBedrockConverseRequest(ctx),
			api.SelectModelMapping(ctx, mapping),
			api.DoBedrockConverseCall(ctx),
		)
		return
	}

	// use mappings imported by External Secrets Operations (SCE inputs for GenAI Infra models) - to be deprecated soon
	if !helpers.HelperSuite.GetEnvOrFalse("USE_AUTO_MAPPING") {
		group.POST(endpoint,
			api.HandleBedrockModelCall(ctx, infra.LoadInfraModelsForContext, client.DoConverse),
		)
		return
	}

	// use mappings loaded by the mapping synchronizer from GenAI Gateway Ops sidecar (new default)
	group.POST(endpoint,
		api.HandleBedrockModelCall(ctx, infra.GetInfraModelsForContext, client.DoConverse),
	)

}

func getSwagger(c *gin.Context) {
	f, err := apidocs.FS.ReadFile("spec.yaml")
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.String(http.StatusOK, string(f))
}
