//
// Copyright (c) 2024 Pegasystems Inc.
// All rights reserved.
//

package endpoints_test

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
	os.Setenv("GIN_MODE", "release")
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Endpoints Suite")
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
	baseURI string
	opsURI  string

	// Test environment components
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

	baseURI = fmt.Sprintf("http://localhost:%s", mainServicePort)
	opsURI = fmt.Sprintf("http://localhost:%s", opsPort)

	// --- CLEANUP PREVIOUS RUN ---
	By("Cleaning up orphaned resources from previous test runs")
	if err := tools.CleanupOrphanedServices(ctx); err != nil {
		fmt.Printf("Warning: Failed to cleanup orphaned services: %v\n", err)
	}
	if err := tools.CleanupOrphanedContainers(ctx, "genai-vector-store-test"); err != nil {
		fmt.Printf("Warning: Failed to cleanup orphaned containers: %v\n", err)
	}
	if err := tools.CleanupOrphanedWireMockContainers(ctx, "genai-vector-store-test"); err != nil {
		fmt.Printf("Warning: Failed to cleanup orphaned WireMock containers: %v\n", err)
	}

	// --- PREBUILD BINARIES (new unified pattern) ---
	By("Pre-building test binaries for the suite")
	buildCache := tools.GetBuildCache()

	// Build background service binary
	_, err = buildCache.EnsureBinary(ctx, tools.ServiceConfig{
		SourcePath:  "./cmd/background",
		BinaryPath:  "bin/background-test",
		ServiceName: "background-test",
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to pre-build background service binary")

	// Build main service
	_, err = buildCache.EnsureBinary(ctx, tools.ServiceConfig{
		SourcePath:  "./cmd/service",
		BinaryPath:  "bin/service-test",
		ServiceName: "main-test",
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to pre-build main service binary")

	// Build ops service
	_, err = buildCache.EnsureBinary(ctx, tools.ServiceConfig{
		SourcePath:  "./cmd/ops",
		BinaryPath:  "bin/ops-test",
		ServiceName: "ops-test",
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to pre-build ops service binary")

	fmt.Println("All test binaries pre-built successfully")

	// --- START WIREMOCK ---
	By("Starting WireMock container for test suite")
	wiremockManager, err = tools.NewWireMockManager(ctx, tools.WireMockConfig{
		ContainerLabel: "genai-vector-store-test",
	})
	Expect(err).ToNot(HaveOccurred(), "Failed to create WireMock manager")

	err = wiremockManager.Start()
	Expect(err).ToNot(HaveOccurred(), "Failed to start WireMock container")

	wiremockURL := wiremockManager.GetBaseURL()
	By(fmt.Sprintf("WireMock available at %s", wiremockURL))
	os.Setenv("GENAI_GATEWAY_SERVICE_URL", wiremockURL)

	// --- START POSTGRES ---
	By("Creating PostgreSQL container with latest schema")
	postgresManager, err = tools.NewPostgreSQLManager(ctx, tools.PostgreSQLConfig{})
	Expect(err).ToNot(HaveOccurred(), "Failed to create PostgreSQL manager")

	By("Starting PostgreSQL container")
	err = postgresManager.Start()
	Expect(err).ToNot(HaveOccurred(), "Failed to start PostgreSQL container")

	dbHost, dbPort := postgresManager.GetConnectionDetails()
	dbConnectionString := postgresManager.GetConnectionString()
	By(fmt.Sprintf("PostgreSQL started at %s:%s", dbHost, dbPort))

	// DB connection
	By("Creating database connection")
	db, err := SetupDatabaseConnectionFromString(ctx, dbConnectionString)
	Expect(err).ToNot(HaveOccurred(), "Failed to create database connection")
	database = db

	// --- START BACKGROUND SERVICE ---
	By("Starting background service")
	backgroundServiceEnv := map[string]string{
		"LOG_LEVEL":                        "DEBUG",
		"DB_LOCAL":                         "true",
		"DB_HOST":                          dbHost,
		"DB_PORT":                          dbPort,
		"DB_NAME":                          "vectordb",
		"DB_USR":                           "testuser",
		"DB_PWD":                           "testpwd",
		"BKG_HEALTHCHECK_PORT":             bkgHealthcheckPort,
		"GENAI_GATEWAY_SERVICE_URL":        wiremockURL,
		"GENAI_SMART_CHUNKING_SERVICE_URL": wiremockURL,
		"DEFAULT_EMBEDDING_PROFILE":        "openai-text-embedding-ada-002",
		"SAX_DISABLED":                     "true",
		"SAX_CLIENT_DISABLED":              "true",
		"DOCUMENT_STATUS_UPDATE_PERIOD_MS": "2000",
	}
	backgroundServiceManager, err = tools.StartBackgroundService(ctx, backgroundServiceEnv)
	Expect(err).ToNot(HaveOccurred(), "Failed to start background service")

	// --- START MAIN SERVICE ---
	By("Starting main service")
	mainServiceEnv := map[string]string{
		"LOG_LEVEL":                        "DEBUG",
		"DB_LOCAL":                         "true",
		"DB_HOST":                          dbHost,
		"DB_PORT":                          dbPort,
		"DB_NAME":                          "vectordb",
		"DB_USR":                           "testuser",
		"DB_PWD":                           "testpwd",
		"SERVICE_PORT":                     mainServicePort,
		"SERVICE_HEALTHCHECK_PORT":         mainHealthcheckPort,
		"GENAI_GATEWAY_SERVICE_URL":        wiremockURL,
		"GENAI_SMART_CHUNKING_SERVICE_URL": wiremockURL,
		"DEFAULT_EMBEDDING_PROFILE":        "openai-text-embedding-ada-002",
		"SAX_DISABLED":                     "true",
		"SAX_CLIENT_DISABLED":              "true",
		"QUERY_EMBEDDING_TIMEOUT_MS":       "2000", // 2 second timeout for query embedding calls (tests use 3s delay to trigger timeout)
	}
	mainServiceManager, err = tools.StartMainService(ctx, mainServiceEnv)
	Expect(err).ToNot(HaveOccurred(), "Failed to start main service")
	// --- START OPS SERVICE ---
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
	if backgroundServiceManager != nil {
		By("Stopping background service")
		_ = backgroundServiceManager.StopService(ctx)
	}

	if mainServiceManager != nil {
		By("Stopping main service")
		_ = mainServiceManager.StopService(ctx)
	}

	if opsServiceManager != nil {
		By("Stopping ops service")
		_ = opsServiceManager.StopService(ctx)
	}

	if database != nil {
		By("Closing database connection")
		ExpectNoIdleTransactionsLeft(ctx, database, "testuser")
		CloseDatabase(database)
	}

	if postgresManager != nil {
		By("Stopping PostgreSQL container")
		_ = postgresManager.Stop()
	}

	if wiremockManager != nil {
		By("Stopping WireMock container")
		_ = wiremockManager.Stop()
	}
})
