/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/apidocs"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/service/api"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/service/apiV2"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/service/background"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/service/health"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/service/otel"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/servicemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/usagemetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sax"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	serviceName     = "genai-vector-store"
	saxScopeRead    = "pega.genai-vector-store:read"
	saxScopeWrite   = "pega.genai-vector-store:write"
	servicePort     = "8080"
	healthcheckPort = "8082"
)

var logger = log.GetNamedLogger(serviceName)

func init() {
	gin.DisableConsoleColor()
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	defer func() { _ = logger.Sync() }() //nolint:errcheck
	logger.Info("Starting vector-store service")

	logger.Info("Configured GENAI gateway service URL", zap.String("url", helpers.GetEnvOrDefault("GENAI_GATEWAY_SERVICE_URL", "")))
	logger.Info("Configured GENAI gateway custom config", zap.String("config", helpers.GetEnvOrDefault("GENAI_GATEWAY_CUSTOM_CONFIG", "")))
	logger.Info("Configured smart chunking service URL", zap.String("url", helpers.GetEnvOrDefault("GENAI_SMART_CHUNKING_SERVICE_URL", "")))
	logger.Info("PGVector distance precision", zap.String("precision", helpers.GetEnvOrDefault("PGVECTOR_DISTANCE_PRECISION", "0")))
	logger.Info("Isolation auto-creation enabled", zap.Bool("enabled", helpers.IsIsolationAutoCreationEnabled()))

	logger.Info("SAX enabled", zap.Bool("enabled", !helpers.IsSaxDisabled()))
	logger.Info("SAX client enabled", zap.Bool("enabled", !helpers.IsSaxClientDisabled()))
	logger.Info("IsolationID verification enabled", zap.Bool("enabled", !helpers.IsIsolationIDVerificationDisabled()))
	logger.Info("Default embedding profile", zap.String("profile", helpers.GetEnvOrDefault("DEFAULT_EMBEDDING_PROFILE", "")))
	logger.Info("Read-only mode enabled", zap.Bool("enabled", helpers.IsReadOnlyMode()))
	logger.Info("Troubleshooting mode enabled", zap.Bool("enabled", helpers.IsTroubleshootingMode()))
	logger.Info("Semantic search index encouragement enabled", zap.Bool("enabled", helpers.IsEncourageSemSearchIndexUseEnabled()))
	logger.Info("Runtime configuration via headers enabled", zap.Bool("enabled", helpers.IsRuntimeConfigurationViaHeadersEnabled()))
	logger.Info("Emulation mode enabled", zap.Bool("enabled", helpers.IsEmulationEnabled()))
	logger.Info("Legacy attributes IDs usage enabled", zap.Bool("enabled", helpers.UseLegacyAttributesIDs()))
	logger.Info("Usage metrics enabled", zap.Bool("enabled", helpers.IsUsageMetricsEnabled()))
	if helpers.IsUsageMetricsEnabled() {
		logger.Info("Usage metrics configuration",
			zap.Int("uploadIntervalSeconds", helpers.GetUsageMetricsUploadIntervalSeconds()),
			zap.Int("maxPayloadSizeBytes", helpers.GetUsageMetricsMaxPayloadSizeBytes()),
			zap.Int("retryCount", helpers.GetUsageMetricsRetryCount()),
			zap.Int("requestTimeoutSeconds", helpers.GetUsageMetricsRequestTimeoutSeconds()))
	}
	if helpers.IsEmulationEnabled() {
		logger.Info("Emulation configuration",
			zap.Int64("min_time_ms", helpers.GetEmulationMinTime()),
			zap.Int64("max_time_ms", helpers.GetEmulationMaxTime()))
	}
	if httpRequestTimeout := helpers.GetEnvOrDefault("HTTP_REQUEST_TIMEOUT", ""); httpRequestTimeout != "" {
		logger.Info("Request timeout configured", zap.String("timeout", httpRequestTimeout))
	}
	if httpRequestBackgroundTimeout := helpers.GetEnvOrDefault("HTTP_REQUEST_BACKGROUND_TIMEOUT", ""); httpRequestBackgroundTimeout != "" {
		logger.Info("Async processing timeout configured", zap.String("timeout", httpRequestBackgroundTimeout))
	}

	ctx := context.Background()

	// Initialize OTEL tracing
	tp, err := otel.SetupTracing(ctx)
	if err != nil {
		logger.Error("failed to initialize OTEL tracer", zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		logger.Error("unable to load DB config", zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}

	genericDB, err := db.NewDatabase(ctx, dbConfig.ForGeneric())
	if err != nil {
		logger.Error("unable to initialize DB connection", zap.String("connStringMasked", dbConfig.ToConnStringMasked()), zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}

	if err = sql.SetupDatabaseForService(ctx, logger, genericDB); err != nil {
		logger.Error("unable to setup DB", zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}
	logger.Info("DB setup successfully completed")

	ingestionDB, err := db.NewDatabase(ctx, dbConfig.ForIngestion())
	if err != nil {
		logger.Error("unable to initialize DB 2nd connection", zap.String("connStringMasked", dbConfig.ToConnStringMasked()), zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}

	searchDB, err := db.NewDatabase(ctx, dbConfig.ForSearch())
	if err != nil {
		logger.Error("unable to initialize DB 3rd connection", zap.String("connStringMasked", dbConfig.ToConnStringMasked()), zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}

	g := errgroup.Group{}

	// Initialize usage metrics collector if enabled
	var isoManager isolations.IsoManager
	if helpers.IsUsageMetricsEnabled() {
		// Configure usage metrics collector with environment variables
		usageMetricsConfig := usagemetrics.Config{
			Enabled:               helpers.IsUsageMetricsEnabled(),
			UploadIntervalSeconds: helpers.GetUsageMetricsUploadIntervalSeconds(),
			MaxPayloadSizeBytes:   helpers.GetUsageMetricsMaxPayloadSizeBytes(),
			RetryCount:            helpers.GetUsageMetricsRetryCount(),
			RequestTimeoutSeconds: helpers.GetUsageMetricsRequestTimeoutSeconds(),
		}

		usagemetrics.SetCollectorConfig(usageMetricsConfig)

		// Create isolation manager for usage metrics middleware
		isoManager = isolations.NewManager(genericDB, logger)

		// Start background uploader
		collector := usagemetrics.GetCollector()
		uploader := usagemetrics.NewUploader(collector, isoManager)

		g.Go(func() error {
			uploader.StartBackgroundUploader(ctx)
			return nil
		})

		logger.Info("Usage metrics collector initialized and background uploader started")
	}

	healthEngine := gin.New()
	serviceEngine := gin.New()

	middlewareChain := []gin.HandlerFunc{
		middleware.ErrorHandler(),
		middleware.RequestTimeoutMiddleware(),
		middleware.RuntimeConfigMiddleware,     // Runtime configuration from headers
		middleware.PathNormalizationMiddleware, // Needed for ReadOnlyMiddleware and MetricsMiddleware
		otelgin.Middleware(serviceName,
			otelgin.WithSpanNameFormatter(func(r *http.Request) string {
				// Get the handler path using Gin's context
				// We need to use the URL path as a fallback since we don't have access to c.FullPath() directly here
				path := r.URL.Path
				return fmt.Sprintf("%s %s %s", serviceName, r.Method, path)
			}),
		),
		middleware.EmulationMiddleware(logger),
		middleware.DatabaseHandler(middleware.DatabasesConfig{
			Generic: genericDB,
			Ingest:  ingestionDB,
			Search:  searchDB,
		}), // Database connections
		middleware.NewReadOnlyMiddleware(), // Read-only mode protection
		middleware.ServiceMetricsMiddleware,
	}

	// Add usage metrics middleware if enabled
	if helpers.IsUsageMetricsEnabled() {
		middlewareChain = append(middlewareChain, middleware.UsageMetricsMiddleware(isoManager))
	}

	middlewareChain = append(middlewareChain,
		middleware.PrometheusGinMiddleware(), // Includes SAX JWT cache metrics if caching is enabled
		middleware.GenaiResponseHeadersMiddleware,
		middleware.ContextInfoMidleware,
		middleware.RequestLoggerMiddleware(ctx),
		middleware.ResponseLoggerMiddleware(ctx),
	)

	serviceEngine.Use(middlewareChain...)

	setupEngine(healthEngine, serviceEngine)

	if !helpers.IsReadOnlyMode() {
		backgroundCtx := servicemetrics.WithMetrics(ctx) // to prevent log warning about missing metrics in background worker

		g.Go(background.GetEmbeddingsProcessingHandler2(backgroundCtx, genericDB))
		g.Go(background.GetDocStatusUpdaterHandler(backgroundCtx, genericDB))
	}

	g.Go(func() error {
		// run in a separate goroutine to have /health endpoint exposed on 8082 port by default or check env variable for local testing
		port := fmt.Sprintf(":%s", helpers.GetEnvOrDefault("SERVICE_HEALTHCHECK_PORT", healthcheckPort))
		logger.Info("running service healthcheck", zap.String("port", port))
		return healthEngine.Run(port)
	})

	g.Go(func() error {
		// run on a default port - 8080 or check env variable for local testing
		port := fmt.Sprintf(":%s", helpers.GetEnvOrDefault("SERVICE_PORT", servicePort))
		logger.Info("running service", zap.String("port", port))
		return serviceEngine.Run(port)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func setupEngine(healthEngine *gin.Engine, serviceEngine *gin.Engine) {

	saxAuth := getSaxValidator()
	isolationValidator := sax.NewIsolationValidator()

	serviceEngine.NoRoute(func(c *gin.Context) {
		logger.Info("route not found", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))
		c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": "404 page not found"})
	})

	healthEngine.Use(
		ginzap.Ginzap(logger.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true), // to decrease bloated logs caused by liveness/readiness checks
		ginzap.RecoveryWithZap(logger, true),
	)

	h := healthEngine.Group("/health")
	{
		h.GET("/liveness", health.GetLiveness)
		h.GET("/readiness", health.GetReadiness)
	}

	//wrapping the prometheus handler into a gin middleware handler
	healthEngine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	serviceEngine.Use(
		ginzap.Ginzap(logger.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true),
		ginzap.RecoveryWithZap(logger, true),
	)

	if helpers.IsTroubleshootingMode() {
		pprof.Register(serviceEngine)
		serviceEngine.GET("/debug/memory", func(c *gin.Context) {
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			memLimit := debug.SetMemoryLimit(-1) // Get current memory limit
			c.JSON(http.StatusOK, gin.H{
				"alloc_MB":       mem.Alloc / 1024 / 1024,
				"totalAlloc_MB":  mem.TotalAlloc / 1024 / 1024,
				"sys_MB":         mem.Sys / 1024 / 1024,
				"numGC":          mem.NumGC,
				"memoryLimit_MB": memLimit / 1024 / 1024,
			})
		})
	}

	serviceEngine.GET("/", getSwagger)
	serviceEngine.StaticFile("./swagger/service.yaml", "./apidocs/service.yaml")
	serviceEngine.StaticFile("./swagger/", "./apidocs/static/swagger/views/swagger-ui/service/index.html")

	v := serviceEngine.Group("/v1/:isolationID/collections/:collectionName")
	{
		//POST (instead of GET) is used for retrieving documents to satisfy the requirement of having attributes filtering capability
		v.POST("/documents", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.RetrieveDocuments)
		v.GET("/documents/:documentID", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.GetDocument)
		v.PUT("/documents", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.PutDocument)
		v.DELETE("/documents", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.DeleteDocuments)
		v.PATCH("/documents/:documentID", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.PatchDocument)
		v.DELETE("/documents/:documentID", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.DeleteDocument)
		v.POST("/document/delete-by-id", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.DeleteDocumentById)

		v.PUT("/file", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.PutDocumentFile)
		v.PUT("/file/text", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.PutDocumentFileText)

		q := v.Group("/query")
		{
			//POST (instead of GET) is used for retrieving chunks to satisfy the requirement of having query parameters capability
			q.POST("/chunks", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.QueryChunks)
			//POST (instead of GET) is used for retrieving documents ids to satisfy the requirement of having query parameters capability
			q.POST("/documents", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.QueryDocuments)
		}
		//POST (instead of GET) is used for retrieving attributes to satisfy the requirement of having attributes filtering capability
		v.POST("/attributes", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.QueryAttributes)
	}

	v2 := serviceEngine.Group("/v2/:isolationID")
	{
		v2.GET("/collections", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), apiV2.GetCollections)
		v2.GET("/collections/:collectionID", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), apiV2.GetCollection)
		v2.POST("/collections", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), apiV2.PostCollection)
		v2.DELETE("/collections/:collectionID", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), apiV2.DeleteCollection)

		v2.GET("/collections/:collectionID/documents/:documentID/chunks", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), apiV2.GetDocumentChunks)
		v2.POST("/collections/:collectionID/find-documents", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), apiV2.FindDocuments)
	}

	ag := serviceEngine.Group("/v1/:isolationID/smart-attributes-group")
	{
		ag.GET("/", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.ListSmartAttributesGroups)
		ag.POST("/", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.PostSmartAttributesGroup)
		ag.PUT("/", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.PutSmartAttributesGroup)
		ag.GET("/:groupID", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.GetSmartAttributesGroup)
		ag.PUT("/:groupID", saxAuth.ValidateRequest(saxScopeWrite), isolationValidator.Validate(), api.PutSmartAttributesGroup)
		ag.DELETE("/:groupID", saxAuth.ValidateRequest(saxScopeRead), isolationValidator.Validate(), api.DeleteSmartAttributesGroup)
	}

}

//// Function to print all registered routes
//func printRoutes(router *gin.Engine) {
//	fmt.Println("Registered Routes:")
//	for _, route := range router.Routes() {
//		fmt.Printf("%s %s\n", route.Method, route.Path)
//	}
//}

func getSwagger(c *gin.Context) {
	f, err := apidocs.FS.ReadFile("service.yaml")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "500", "message": fmt.Sprintf("Failed to read service.yaml: %v", err)})
		return
	}
	c.String(http.StatusOK, string(f))
}

func getSaxValidator() sax.Validator {
	if helpers.IsSaxDisabled() {
		logger.Info("SAX validation disabled")
		return sax.NewValidatorMock()
	}
	logger.Info("SAX validation enabled")
	cfg := sax.Config{
		Audience:     helpers.GetEnvOrPanic("SAX_AUDIENCE"),
		Issuer:       helpers.GetEnvOrPanic("SAX_ISSUER"),
		JWKSEndpoint: helpers.GetEnvOrPanic("SAX_JWKS_ENDPOINT"),
	}
	validator, err := sax.New(cfg)
	if err != nil {
		panic("failed to create SAX validator: " + err.Error())
	}

	// Wrap with caching if enabled
	if helpers.IsSaxTokenCacheEnabled() {
		logger.Info("SAX token caching enabled")
		return sax.NewCachedValidator(validator)
	}

	logger.Info("SAX token caching disabled")
	return validator
}
