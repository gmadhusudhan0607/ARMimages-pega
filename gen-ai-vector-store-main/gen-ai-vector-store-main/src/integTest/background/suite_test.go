// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package background

import (
	"context"
	"fmt"
	"testing"

	"github.com/onsi/gomega/format"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/tools"
)

func TestBackgroundIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Background Processes Suite")
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
	baseURI string
	opsURI  string

	// Test infrastructure
	postgresManager    *tools.PostgreSQLManager
	backgroundManager  *tools.ServiceManager
	mainServiceManager *tools.ServiceManager
	opsServiceManager  *tools.ServiceManager
	wiremockManager    *tools.WireMockManager
	database           *pgxpool.Pool
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

	baseURI = fmt.Sprintf("http://localhost:%s", mainServicePort)
	opsURI = fmt.Sprintf("http://localhost:%s", opsPort)

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
})

var _ = AfterSuite(func() {
	// Stop WireMock container
	if wiremockManager != nil {
		_ = wiremockManager.Stop()
	}
})
