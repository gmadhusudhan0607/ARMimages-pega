// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/onsi/gomega/format"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/tools"
)

func TestUsageMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Usage Metrics Suite")
}

var (
	// Service ports — assigned dynamically in BeforeSuite to avoid conflicts
	// between parallel test runs on the same machine.
	mainServicePort     string
	mainHealthcheckPort string
	opsPort             string
	opsHealthcheckPort  string
	bkgHealthcheckPort  string

	// Service URIs — initialized in BeforeSuite after port allocation
	svcBaseURI string
	opsBaseURI string

	// Test infrastructure
	postgresManager          *tools.PostgreSQLManager
	mainServiceManager       *tools.ServiceManager
	opsServiceManager        *tools.ServiceManager
	backgroundServiceManager *tools.ServiceManager
	wiremockManager          *tools.WireMockManager
	database                 *pgxpool.Pool
)

var _ = BeforeSuite(func() {
	format.MaxLength = 0

	ctx := context.Background()

	// Allocate free ports dynamically to avoid conflicts with parallel test runs
	var err error
	mainServicePort, err = tools.FindFreePort()
	Expect(err).ToNot(HaveOccurred(), "Failed to find free port for main service")
	mainHealthcheckPort, err = tools.FindFreePort()
	Expect(err).ToNot(HaveOccurred(), "Failed to find free port for main healthcheck")
	opsPort, err = tools.FindFreePort()
	Expect(err).ToNot(HaveOccurred(), "Failed to find free port for ops service")
	opsHealthcheckPort, err = tools.FindFreePort()
	Expect(err).ToNot(HaveOccurred(), "Failed to find free port for ops healthcheck")
	bkgHealthcheckPort, err = tools.FindFreePort()
	Expect(err).ToNot(HaveOccurred(), "Failed to find free port for background healthcheck")

	svcBaseURI = fmt.Sprintf("http://localhost:%s", mainServicePort)
	opsBaseURI = fmt.Sprintf("http://localhost:%s", opsPort)

	// Clean up orphaned services and containers from previous test runs
	// This ensures a clean state even when KEEP=true was used in previous runs
	By("Cleaning up orphaned resources from previous test runs")
	if err := tools.CleanupOrphanedServices(ctx); err != nil {
		fmt.Printf("Warning: Failed to cleanup orphaned services: %v\n", err)
	}
	if err := tools.CleanupOrphanedContainers(ctx, "genai-vector-store-test"); err != nil {
		fmt.Printf("Warning: Failed to cleanup orphaned containers: %v\n", err)
	}
	if err := tools.CleanupOrphanedWireMockContainers(ctx, "genai-vector-store-test"); err != nil {
		fmt.Printf("Warning: Failed to cleanup orphaned WireMockContainers: %v\n", err)
	}

	// Pre-build all test binaries once at suite level
	// This significantly reduces test execution time by avoiding repeated builds
	By("Pre-building test service binaries for the suite")
	buildCache := tools.GetBuildCache()

	// Build background service binary
	_, err = buildCache.EnsureBinary(ctx, tools.ServiceConfig{
		SourcePath:  "./cmd/background",
		BinaryPath:  "bin/background-test",
		ServiceName: "background-test",
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to pre-build background service binary")

	// Build main service binary
	_, err = buildCache.EnsureBinary(ctx, tools.ServiceConfig{
		SourcePath:  "./cmd/service",
		BinaryPath:  "bin/service-test",
		ServiceName: "main-test",
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to pre-build main service binary")

	// Build ops service binary
	_, err = buildCache.EnsureBinary(ctx, tools.ServiceConfig{
		SourcePath:  "./cmd/ops",
		BinaryPath:  "bin/ops-test",
		ServiceName: "ops-test",
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to pre-build ops service binary")

	fmt.Println("All test binaries pre-built successfully")

	// Start WireMock container for suite
	By("Starting WireMock container for test suite")
	wiremockManager, err = tools.NewWireMockManager(ctx, tools.WireMockConfig{
		ContainerLabel: "genai-vector-store-test",
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to create WireMock manager")

	err = wiremockManager.Start()
	Expect(err).ToNot(HaveOccurred(), "Failed to start WireMock container")

	// 1. Create and start PostgreSQL container
	By("Creating PostgreSQL container with latest schema")
	postgresManager, err = tools.NewPostgreSQLManager(ctx, tools.PostgreSQLConfig{})
	Expect(err).ToNot(HaveOccurred(), "Failed to create PostgreSQL manager")

	By("Starting PostgreSQL container")
	err = postgresManager.Start()
	Expect(err).ToNot(HaveOccurred(), "Failed to start PostgreSQL container")

	// 2. Get connection details
	dbHost, dbPort := postgresManager.GetConnectionDetails()
	dbConnString := postgresManager.GetConnectionString()
	By(fmt.Sprintf("PostgreSQL container started at %s:%s", dbHost, dbPort))

	// 3. Create database connection
	By("Creating database connection")
	db, err := SetupDatabaseConnectionFromString(ctx, dbConnString)
	Expect(err).ToNot(HaveOccurred(), "Failed to create database connection")
	database = db

	// 4. Get WireMock URL
	wiremockURL := wiremockManager.GetBaseURL()
	By(fmt.Sprintf("WireMock container available at %s", wiremockURL))

	// 5. Start background service first to update database schema, then keep it running for retry processing
	By("Starting background service")
	backgroundServiceEnv := map[string]string{
		"LOG_LEVEL":                              "DEBUG",
		"DB_LOCAL":                               "true",
		"DB_HOST":                                dbHost,
		"DB_PORT":                                dbPort,
		"DB_NAME":                                "vectordb",
		"DB_USR":                                 "testuser",
		"DB_PWD":                                 "testpwd",
		"BKG_HEALTHCHECK_PORT":                   bkgHealthcheckPort,
		"GENAI_GATEWAY_SERVICE_URL":              wiremockURL,
		"DEFAULT_EMBEDDING_PROFILE":              "openai-text-embedding-ada-002",
		"SAX_DISABLED":                           "true",
		"SAX_CLIENT_DISABLED":                    "true",
		"DOCUMENT_STATUS_UPDATE_PERIOD_MS":       "2000", // Reduce from 30000ms to 2000ms for faster test execution
		"USAGE_METRICS_ENABLED":                  "true",
		"DB_METRICS_PDC_UPLOAD_INTERVAL_SECONDS": "1",
	}
	backgroundServiceManager, err = tools.StartBackgroundService(ctx, backgroundServiceEnv)
	Expect(err).ToNot(HaveOccurred(), "Failed to start background service")

	// 6. Start main service
	By("Starting main service")
	mainServiceEnv := map[string]string{
		"LOG_LEVEL":                             "DEBUG",
		"DB_LOCAL":                              "true",
		"DB_HOST":                               dbHost,
		"DB_PORT":                               dbPort,
		"DB_NAME":                               "vectordb",
		"DB_USR":                                "testuser",
		"DB_PWD":                                "testpwd",
		"SERVICE_PORT":                          mainServicePort,
		"SERVICE_HEALTHCHECK_PORT":              mainHealthcheckPort,
		"GENAI_GATEWAY_SERVICE_URL":             wiremockURL,
		"DEFAULT_EMBEDDING_PROFILE":             "openai-text-embedding-ada-002",
		"SAX_DISABLED":                          "true",
		"SAX_CLIENT_DISABLED":                   "true",
		"USAGE_METRICS_ENABLED":                 "false",
		"USAGE_METRICS_UPLOAD_INTERVAL_SECONDS": "1",
	}
	mainServiceManager, err = tools.StartMainService(ctx, mainServiceEnv)
	Expect(err).ToNot(HaveOccurred(), "Failed to start main service")

	// 7. Start ops service
	By("Starting ops service")
	opsServiceEnv := map[string]string{
		"LOG_LEVEL":            "DEBUG",
		"DB_LOCAL":             "true",
		"DB_HOST":              dbHost,
		"DB_PORT":              dbPort,
		"DB_NAME":              "vectordb",
		"DB_USR":               "testuser",
		"DB_PWD":               "testpwd",
		"OPS_PORT":             opsPort,
		"OPS_HEALTHCHECK_PORT": opsHealthcheckPort,
		"SAX_DISABLED":         "true",
		"SAX_CLIENT_DISABLED":  "true",
	}
	opsServiceManager, err = tools.StartOpsService(ctx, opsServiceEnv)
	Expect(err).ToNot(HaveOccurred(), "Failed to start ops service")

	By("All services started successfully")
})

var _ = AfterSuite(func() {
	ctx := context.Background()

	// Stop background service
	if backgroundServiceManager != nil {
		By("Stopping background service")
		_ = backgroundServiceManager.StopService(ctx)
	}

	// Stop main service
	if mainServiceManager != nil {
		By("Stopping main service")
		_ = mainServiceManager.StopService(ctx)
	}

	// Stop ops service
	if opsServiceManager != nil {
		By("Stopping ops service")
		_ = opsServiceManager.StopService(ctx)
	}

	// Close database connection
	if database != nil {
		By("Closing database connection")
		ExpectNoIdleTransactionsLeft(ctx, database, "testuser")
		CloseDatabase(database)
	}

	// Stop PostgreSQL container
	if postgresManager != nil {
		By("Stopping PostgreSQL container")
		_ = postgresManager.Stop()
	}

	// Stop WireMock container
	if wiremockManager != nil {
		By("Stopping WireMock container")
		_ = wiremockManager.Stop()
	}
})
