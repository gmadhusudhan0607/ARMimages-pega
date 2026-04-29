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

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/workers/dbconfigpuller"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/background/dbproxy"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/cmd/background/health"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/config"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/dbmetrics"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/usagemetrics"
	_ "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/workers" // Import for metrics registration
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/sql"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/workers/maintenance"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	serviceName     = "genai-vector-store-background"
	healthcheckPort = "8082"
	dbProxyPort     = "35432"
)

var logger = log.GetNamedLogger(serviceName)

func init() {
	gin.DisableConsoleColor()
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	defer func() { _ = logger.Sync() }() //nolint:errcheck
	logger.Info("Starting vector-store-background service")
	logger.Info("Configured GENAI gateway service URL", zap.String("url", helpers.GetEnvOrDefault("GENAI_GATEWAY_SERVICE_URL", "")))
	logger.Info("Configured GENAI gateway custom config", zap.String("config", helpers.GetEnvOrDefault("GENAI_GATEWAY_CUSTOM_CONFIG", "")))
	logger.Info("Default embedding profile", zap.String("profile", helpers.GetEnvOrDefault("DEFAULT_EMBEDDING_PROFILE", "")))
	logger.Info("Troubleshooting mode enabled", zap.Bool("enabled", helpers.IsTroubleshootingMode()))

	ctx := context.Background()

	g := errgroup.Group{}
	healthEngine := gin.New()
	setupEngine(healthEngine)

	// For Dev purposes only
	if helpers.GetEnvOrDefault("ENABLE_DB_PROXY", "false") == "true" {
		logger.Info("starting dbproxy", zap.String("port", dbProxyPort))
		g.Go(func() error {
			proxy := dbproxy.NewProxy(
				helpers.GetEnvOrPanic("DB_HOST"),
				fmt.Sprintf(":%s", helpers.GetEnvOrPanic("DB_PORT")))
			return proxy.Start(dbProxyPort)
		})
	}

	dbConfig, err := config.LoadDatabaseConfig()
	if err != nil {
		logger.Error("unable to load DB config", zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}
	database, err := db.NewDatabase(ctx, dbConfig)
	if err != nil {
		logger.Error("unable to initialize DB connection", zap.String("connStringMasked", dbConfig.ToConnStringMasked()), zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}

	g.Go(func() error {
		// run in a separate goroutine to have /health endpoint exposed on 8082 port by default or check env variable for local testing
		port := fmt.Sprintf(":%s", helpers.GetEnvOrDefault("BKG_HEALTHCHECK_PORT", healthcheckPort))
		logger.Info("running service healthcheck", zap.String("port", port))
		return healthEngine.Run(port)
	})

	// Start database metrics collection in background
	g.Go(func() error {
		prometheusCollector := dbmetrics.NewPrometheusCollector(database)
		return prometheusCollector.GetDbMetricsHandler(ctx)()
	})

	logger.Sugar().Info("starting embeddings rescheduler")
	if err = sql.SetupDatabaseForBackground(ctx, logger, database); err != nil {
		logger.Error("unable to setup DB", zap.Error(err))
		_ = logger.Sync() // flush logs before exit
		os.Exit(1)
	}
	logger.Info("DB setup successfully completed")

	// Start attribute replication
	logger.Info("Starting attribute replication process")
	g.Go(func() error {
		worker := maintenance.NewAttributesReplicator(database)
		return worker.RunReplication(ctx)
	})

	// Start DB configuration puller
	logger.Info("Starting DB configuration puller")
	g.Go(func() error {
		puller := dbconfigpuller.NewDBConfigPuller(database)
		return puller.Run(ctx)
	})

	// Start DB metrics uploader to PDC
	logger.Info("Starting DB metrics uploader to PDC")
	g.Go(func() error {
		isoManager := isolations.NewManager(database, logger)
		dbMetricsUploader := usagemetrics.NewDBMetricsUploader(database, isoManager)
		return dbMetricsUploader.StartBackgroundUploader(ctx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func setupEngine(healthEngine *gin.Engine) {

	healthEngine.NoRoute(func(c *gin.Context) {
		logger.Info("route not found", zap.String("method", c.Request.Method), zap.String("uri", c.Request.RequestURI))
		c.JSON(http.StatusNotFound, gin.H{"code": "404", "message": "404 page not found.", "method": c.Request.Method, "uri": c.Request.RequestURI})
	})

	healthEngine.Use(
		ginzap.Ginzap(logger.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true), // to decrease bloated logs caused by liveness/readiness checks
		ginzap.RecoveryWithZap(logger, true),
	)

	if helpers.IsTroubleshootingMode() {
		pprof.Register(healthEngine)
	}

	//wrapping the prometheus handler into a gin middleware handler
	healthEngine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	h := healthEngine.Group("/health")
	{
		h.GET("/liveness", health.GetLiveness)
		h.GET("/readiness", health.GetReadiness)
	}
}
