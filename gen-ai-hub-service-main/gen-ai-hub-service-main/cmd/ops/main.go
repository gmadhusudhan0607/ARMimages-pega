/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/ops/api"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/health"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra/mapping"
	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/middleware"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	opsPort             = "8081"
	healthcheckPort     = "8082"
	serviceName         = "genai-ops"
	defaultTaskInterval = "5m"
)

var defaultTaskIntervalDuration, _ = time.ParseDuration(defaultTaskInterval)
var helperSuite = helpers.HelperSuite

type Server struct {
	router *gin.Engine
	port   string
}

// New method wrapping router.Run using context condition.
func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:              ":" + s.port,
		Handler:           s.router,
		ReadHeaderTimeout: 30 * time.Second,
	}

	// Start the server in a separate goroutine
	go func() {
		<-ctx.Done()                           // Wait for context cancellation
		_ = srv.Shutdown(context.Background()) // Gracefully shut down the server
	}()

	// Start the server and return any errors
	return srv.ListenAndServe()
}

type ScheduledTask struct {
	jobName  string
	task     func() error
	interval time.Duration
}

func (st *ScheduledTask) Start(ctx context.Context) error {

	if err := st.task(); err != nil {
		cntx.LoggerFromContext(ctx).Sugar().Errorf("Scheduled task %s failed in first execution: %v", st.jobName, err)
	}

	if st.interval == 0 {
		cntx.LoggerFromContext(ctx).Sugar().Errorf("Scheduled task %s has no interval set, skipping scheduling.", st.jobName)
		return nil
	}

	ticker := time.NewTicker(st.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := st.task(); err != nil {
				cntx.LoggerFromContext(ctx).Sugar().Errorf("Scheduled task %s failed: %v", st.jobName, err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

type GenAIGatewayOpsApp struct {
	opsServer      *Server
	healthServer   *Server
	mappingFetcher *ScheduledTask // added mappingFetcher field
}

// Run starts the scheduled task and both servers concurrently using an errgroup.
func (app *GenAIGatewayOpsApp) Run(ctx context.Context) *errgroup.Group {
	var g errgroup.Group

	if cntx.IsUseGenAiInfraModels(ctx) && helperSuite.GetEnvOrFalse("USE_AUTO_MAPPING") {
		g.Go(func() error {
			return app.mappingFetcher.Start(ctx)
		})
	}

	g.Go(func() error {
		return app.opsServer.Start(ctx)
	})
	g.Go(func() error {
		return app.healthServer.Start(ctx)
	})
	return &g
}

func init() {
	gin.DisableConsoleColor()
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	contextName := helperSuite.GetEnvOrDefault("CONTEXT_NAME", serviceName)
	if helperSuite.CreateServiceContext == nil { // if context was already set, this is running in test mode
		helperSuite.CreateServiceContext = cntx.ServiceContext
	}
	ctx := helperSuite.CreateServiceContext(contextName)

	l := cntx.LoggerFromContext(ctx).Named("opsmain").Sugar()

	var t *ScheduledTask
	awsCredsProvider := mapping.NewAwsCredentialProvider()
	mappings := mapping.NewSyncMappingStore()
	if cntx.IsUseGenAiInfraModels(ctx) && helperSuite.GetEnvOrFalse("USE_AUTO_MAPPING") {
		awsCredProvider := mapping.NewAwsCredentialProvider()
		modelMappingService := mapping.NewModelMappingService(mappings, mapping.LoadData, awsCredProvider)
		t = setupAWSMappingSynchronizer(ctx, modelMappingService.Execute)
	} else {
		l.Debugf("GenAI Infra is NOT enabled. Skipping to configure mapping synchronizer.")
	}

	opsEngine := setupOpsServer(ctx, mappings, awsCredsProvider)
	opsPortUsed := helperSuite.GetEnvOrDefault("OPS_PORT", opsPort)
	o := &Server{router: opsEngine, port: opsPortUsed}

	healthEngine := setupHealthServer(ctx, mappings)
	healthPortUsed := helperSuite.GetEnvOrDefault("SERVICE_HEALTHCHECK_PORT", healthcheckPort)
	h := &Server{router: healthEngine, port: healthPortUsed}

	app := &GenAIGatewayOpsApp{
		opsServer:      o,
		healthServer:   h,
		mappingFetcher: t,
	}
	g := app.Run(ctx)

	if err := g.Wait(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func setupAWSMappingSynchronizer(ctx context.Context, task func() error) *ScheduledTask {

	l := cntx.LoggerFromContext(ctx).Sugar()
	l.Debugf("GenAI Infra is enabled. Configuring scheduled mappings synchronizer.")

	// Parse the fetchMappingInterval string
	mappingRefreshInterval := helperSuite.GetEnvOrDefault("GENAI_INFRA_MAPPING_REFRESH_INTERVAL", defaultTaskInterval)
	fetchMappingInterval, err := time.ParseDuration(mappingRefreshInterval)
	if err != nil {
		l.Warnf("error parsing fetchMappingInterval - using default interval of 5m. %w", err)
		fetchMappingInterval = defaultTaskIntervalDuration
	}

	// Create ScheduledTask wrapping getModelMappings with a 1-minute interval.
	mappingTask := &ScheduledTask{
		jobName:  "AWSGenAIInfraMappingFetcher",
		task:     task,
		interval: fetchMappingInterval,
	}
	return mappingTask
}

// New function to setup Health Server.
func setupHealthServer(ctx context.Context, mappings *mapping.SyncMappingStore) *gin.Engine {
	l := cntx.LoggerFromContext(ctx)
	l.Debug("Configuring Health Server")

	router := gin.New()
	router.Use(
		ginzap.Ginzap(l.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true),
		ginzap.RecoveryWithZap(l, true),
	)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	h := router.Group("/health")
	{
		h.GET("/liveness", health.GetLiveness)
		h.GET("/readiness", health.GetOpsReadiness(ctx, mappings))
	}
	return router
}

// New function to setup Ops Server.
func setupOpsServer(ctx context.Context, mappings *mapping.SyncMappingStore, awsCredsProvider mapping.CredentialsProvider) *gin.Engine {
	l := cntx.LoggerFromContext(ctx)
	l.Debug("Configuring Ops Server")
	router := gin.New()
	router.Use(
		middleware.RequestLoggerMiddleware(ctx),
		middleware.ResponseLoggerMiddleware(ctx),
		ginzap.Ginzap(l.WithOptions(zap.IncreaseLevel(zap.ErrorLevel)), time.RFC3339, true),
		ginzap.RecoveryWithZap(l, true),
	)
	ops := router.Group("/v1")
	{
		ops.GET("/isolations/:isolationId/metrics", api.HandleGetIsolationMetrics(ctx))
		ops.POST("/events", api.HandlePostEventRequest(ctx))
		ops.GET("/mappings", api.HandleGetMappingsRequest(ctx, mappings))
		ops.GET("/models/defaults", api.HandleGetDefaultsRequest(ctx, awsCredsProvider, mapping.NewAuthenticatedClient))
	}
	return router
}
