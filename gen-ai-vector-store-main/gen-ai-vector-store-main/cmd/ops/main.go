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
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/apidocs"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/middleware"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/ops/api"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/ops/health"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sax"
)

const (
	serviceName = "genai-vector-store-ops"

	authScopeIsolationRead   = "pega.genai-vector-store-ops:isolations.read"
	authScopeOperationsWrite = "pega.genai-vector-store-ops:operations.write"
	authScopeOperationsRead  = "pega.genai-vector-store-ops:operations.read"
	authScopeIsolationWrite  = "pega.genai-vector-store-ops:isolations.write"
	authScopeSwagger         = "pega.genai-vector-store-ops:swagger"

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
	logger.Info("Starting vector-store-ops service")
	logger.Info("Configured GENAI gateway service URL", zap.String("url", helpers.GetEnvOrDefault("GENAI_GATEWAY_SERVICE_URL", "")))
	logger.Info("Configured GENAI gateway custom config", zap.String("config", helpers.GetEnvOrDefault("GENAI_GATEWAY_CUSTOM_CONFIG", "")))
	logger.Info("Default embedding profile", zap.String("profile", helpers.GetEnvOrDefault("DEFAULT_EMBEDDING_PROFILE", "")))
	logger.Info("Troubleshooting mode enabled", zap.Bool("enabled", helpers.IsTroubleshootingMode()))
	logger.Info("SAX enabled", zap.Bool("enabled", !helpers.IsSaxDisabled()))
	logger.Info("SAX client enabled", zap.Bool("enabled", !helpers.IsSaxClientDisabled()))

	// TEMPORARY  Workaround for the issue with updating pgvector version here andin main service the same
	time.Sleep(time.Second * 1)

	ctx := context.Background()

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		logger.Error("unable to load DB config", zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}
	genericDB, err := db.NewDatabase(ctx, dbConfig.ForGeneric())
	if err != nil {
		logger.Error("unable to initialize DB connection", zap.String("connString", dbConfig.ToConnStringMasked()), zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}

	if err = sql.SetupDatabaseForOps(ctx, logger, genericDB); err != nil {
		logger.Error("unable to setup DB", zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}
	logger.Info("DB setup successfully completed")

	ingestionDB, err := db.NewDatabase(ctx, dbConfig.ForIngestion())
	if err != nil {
		logger.Error("unable to initialize DB 2nd connection", zap.String("connString", dbConfig.ToConnStringMasked()), zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}

	g := errgroup.Group{}

	healthEngine := gin.New()
	serviceEngine := gin.New()

	serviceEngine.Use(
		middleware.ErrorHandler(),
		middleware.RuntimeConfigMiddleware,     // Runtime configuration from headers
		middleware.PathNormalizationMiddleware, // Needed for ReadOnlyMiddleware
		middleware.DatabaseHandler(middleware.DatabasesConfig{
			Generic: genericDB,
			Ingest:  ingestionDB,
			Search:  nil,
		}),
		middleware.NewReadOnlyMiddleware(), // Read-only mode protection
		middleware.RequestLoggerMiddleware(ctx),
		middleware.ResponseLoggerMiddleware(ctx),
	)
	setupEngine(healthEngine, serviceEngine)

	g.Go(func() error {
		// run in a separate goroutine to have /health endpoint exposed on 8082 port by default or check env variable for local testing
		port := fmt.Sprintf(":%s", helpers.GetEnvOrDefault("OPS_HEALTHCHECK_PORT", healthcheckPort))
		logger.Info("running service healthcheck", zap.String("port", port))
		return healthEngine.Run(port)
	})

	g.Go(func() error {
		// run on a default port - 8080 or check env variable for local testing
		port := fmt.Sprintf(":%s", helpers.GetEnvOrDefault("OPS_PORT", servicePort))
		logger.Info("running service", zap.String("port", port))
		return serviceEngine.Run(port)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func setupEngine(healthEngine *gin.Engine, serviceEngine *gin.Engine) {
	healthEngine.Use(
		ginzap.Ginzap(logger.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true), // to decrease bloated logs caused by liveness/readiness checks
		ginzap.RecoveryWithZap(logger, true),
	)

	h := healthEngine.Group("/health")
	{
		h.GET("/liveness", health.GetLiveness)
		h.GET("/readiness", health.GetReadiness)
	}

	serviceEngine.Use(
		ginzap.Ginzap(logger.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true),
		ginzap.RecoveryWithZap(logger, true),
		middleware.ServiceMetricsMiddleware,
	)
	sv := getSaxValidator()

	if helpers.IsTroubleshootingMode() {
		pprof.Register(serviceEngine)
	}

	serviceEngine.GET("/swagger/ops.yaml", sv.ValidateRequest(authScopeSwagger), getSwagger)
	v := serviceEngine.Group("/v1/isolations")
	{
		v.GET(":isolationID", sv.ValidateRequest(authScopeIsolationRead), api.GetIsolation)
		v.POST("", sv.ValidateRequest(authScopeIsolationWrite), api.PostIsolation)
		v.PUT(":isolationID", sv.ValidateRequest(authScopeIsolationWrite), api.PutIsolation)
		v.DELETE(":isolationID", sv.ValidateRequest(authScopeIsolationWrite), api.DeleteIsolation)
	}

	//Read only endpoints added as a workaround for mrdr operations on isolation while on "non-active" mode
	ro := serviceEngine.Group("/v1/isolationsRO")
	{
		ro.GET(":isolationID", sv.ValidateRequest(authScopeIsolationRead), api.GetIsolationRO)
		ro.POST("", sv.ValidateRequest(authScopeIsolationWrite), api.PostIsolationRO)
		ro.PUT(":isolationID", sv.ValidateRequest(authScopeIsolationWrite), api.PutIsolationRO)
		ro.DELETE(":isolationID", sv.ValidateRequest(authScopeIsolationWrite), api.DeleteIsolationRO)
	}

	o := serviceEngine.Group("/v1/ops/:isolationID")
	{
		o.POST("documents", sv.ValidateRequest(authScopeOperationsRead), api.RetrieveDocumentsMetrics)
		o.POST("documentsDetails", sv.ValidateRequest(authScopeOperationsRead), api.RetrieveDocumentsMetricsDetails)
	}

	repl := serviceEngine.Group("/v1/db")
	{
		repl.GET("/configuration", sv.ValidateRequest(authScopeOperationsRead), api.GetConfiguration)
		repl.GET("/size", sv.ValidateRequest(authScopeOperationsRead), api.GetDatabaseSize)
	}
}

func getSwagger(c *gin.Context) {
	f, err := apidocs.FS.ReadFile("ops.yaml")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
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
