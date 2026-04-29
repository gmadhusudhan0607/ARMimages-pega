//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package readonly_mode_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/tools"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OPS Readonly Mode Suite")
}

var (
	// Service ports — assigned dynamically in BeforeSuite to avoid conflicts
	// between parallel test runs on the same machine.
	mainServicePort           string
	mainHealthcheckPort       string
	opsPort                   string
	opsHealthcheckPort        string
	backgroundHealthcheckPort string

	// Service URIs — initialized in BeforeSuite after port allocation
	baseSvcURI string
	baseOpsURI string
	baseOpsURL string // Alias for backwards compatibility with existing tests

	// Test infrastructure
	postgresManager          *tools.PostgreSQLManager
	backgroundServiceManager *tools.ServiceManager
	mainServiceManager       *tools.ServiceManager
	opsServiceManager        *tools.ServiceManager
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
	backgroundHealthcheckPort, err = tools.FindFreePort()
	Expect(err).ToNot(HaveOccurred(), "Failed to find free port for background healthcheck")

	baseSvcURI = fmt.Sprintf("http://localhost:%s", mainServicePort)
	baseOpsURI = fmt.Sprintf("http://localhost:%s", opsPort)
	baseOpsURL = baseOpsURI

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

	// Create and start PostgreSQL container
	By("Creating PostgreSQL container with latest schema")
	postgresManager, err = tools.NewPostgreSQLManager(ctx, tools.PostgreSQLConfig{})
	Expect(err).ToNot(HaveOccurred(), "Failed to create PostgreSQL manager")

	By("Starting PostgreSQL container")
	err = postgresManager.Start()
	Expect(err).ToNot(HaveOccurred(), "Failed to start PostgreSQL container")

	// Get connection details
	dbHost, dbPort := postgresManager.GetConnectionDetails()
	dbConnString := postgresManager.GetConnectionString()
	By(fmt.Sprintf("PostgreSQL container started at %s:%s", dbHost, dbPort))

	// Create database connection
	By("Creating database connection")
	db, err := SetupDatabaseConnectionFromString(ctx, dbConnString)
	Expect(err).ToNot(HaveOccurred(), "Failed to create database connection")
	database = db

	// Get WireMock URL and expose it for test helpers
	wiremockURL := wiremockManager.GetBaseURL()
	By(fmt.Sprintf("WireMock container available at %s", wiremockURL))
	os.Setenv("GENAI_GATEWAY_SERVICE_URL", wiremockURL)

	// Start background service FIRST (must be started before main service)
	By("Starting background service in readonly mode")
	backgroundServiceEnv := map[string]string{
		"LOG_LEVEL":                        "DEBUG",
		"DB_LOCAL":                         "true",
		"DB_HOST":                          dbHost,
		"DB_PORT":                          dbPort,
		"DB_NAME":                          "vectordb",
		"DB_USR":                           "testuser",
		"DB_PWD":                           "testpwd",
		"BKG_HEALTHCHECK_PORT":             backgroundHealthcheckPort,
		"GENAI_GATEWAY_SERVICE_URL":        wiremockURL,
		"DEFAULT_EMBEDDING_PROFILE":        "openai-text-embedding-ada-002",
		"SAX_DISABLED":                     "true",
		"SAX_CLIENT_DISABLED":              "true",
		"DOCUMENT_STATUS_UPDATE_PERIOD_MS": "2000",
		"ENABLE_RUNTIME_HEADER_CONFIG":     "true",
	}
	backgroundServiceManager, err = tools.StartBackgroundService(ctx, backgroundServiceEnv)
	Expect(err).ToNot(HaveOccurred(), "Failed to start background service")

	// Start main service SECOND (after background service)
	By("Starting main service in readonly mode")
	mainServiceEnv := map[string]string{
		"LOG_LEVEL":                    "DEBUG",
		"DB_LOCAL":                     "true",
		"DB_HOST":                      dbHost,
		"DB_PORT":                      dbPort,
		"DB_NAME":                      "vectordb",
		"DB_USR":                       "testuser",
		"DB_PWD":                       "testpwd",
		"SERVICE_PORT":                 mainServicePort,
		"SERVICE_HEALTHCHECK_PORT":     mainHealthcheckPort,
		"GENAI_GATEWAY_SERVICE_URL":    wiremockURL,
		"DEFAULT_EMBEDDING_PROFILE":    "openai-text-embedding-ada-002",
		"SAX_DISABLED":                 "true",
		"SAX_CLIENT_DISABLED":          "true",
		"ENABLE_RUNTIME_HEADER_CONFIG": "true",
	}
	mainServiceManager, err = tools.StartMainService(ctx, mainServiceEnv)
	Expect(err).ToNot(HaveOccurred(), "Failed to start main service")

	// Start ops service LAST (after main service)
	By("Starting ops service in readonly mode")
	opsServiceEnv := map[string]string{
		"LOG_LEVEL":                    "DEBUG",
		"DB_LOCAL":                     "true",
		"DB_HOST":                      dbHost,
		"DB_PORT":                      dbPort,
		"DB_NAME":                      "vectordb",
		"DB_USR":                       "testuser",
		"DB_PWD":                       "testpwd",
		"OPS_PORT":                     opsPort,
		"OPS_HEALTHCHECK_PORT":         opsHealthcheckPort,
		"SAX_DISABLED":                 "true",
		"SAX_CLIENT_DISABLED":          "true",
		"ENABLE_RUNTIME_HEADER_CONFIG": "true",
	}
	opsServiceManager, err = tools.StartOpsService(ctx, opsServiceEnv)
	Expect(err).ToNot(HaveOccurred(), "Failed to start ops service")

	By("All services started successfully in readonly mode")
})

var _ = AfterSuite(func() {
	ctx := context.Background()

	// Ensure we tested all OPS endpoints for readonly mode
	swaggerSpecURI := fmt.Sprintf("%s/swagger/ops.yaml", baseOpsURI)
	ExpectAllOpsEndpointsTestedForReadOnlyMode(swaggerSpecURI)

	// Stop services in reverse order
	// Stop ops service first
	if opsServiceManager != nil {
		By("Stopping ops service")
		_ = opsServiceManager.StopService(ctx)
	}

	// Stop main service second
	if mainServiceManager != nil {
		By("Stopping main service")
		_ = mainServiceManager.StopService(ctx)
	}

	// Stop background service last
	if backgroundServiceManager != nil {
		By("Stopping background service")
		_ = backgroundServiceManager.StopService(ctx)
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
