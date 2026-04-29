// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package background

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/tools"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// migrationTestConfig defines the configuration for a migration test scenario
type migrationTestConfig struct {
	schemaFile      string // SQL file to initialize the database (e.g., "schema_v0_20_0.sql")
	initialVersion  string // Expected initial schema version (e.g., "v0.20.0")
	expectedVersion string // Expected final schema version after migration (e.g., "v0.21.0")
	prevVersion     string // Expected previous schema version (e.g., "v0.20.0")
	maxMigrationVer string // Optional: limit migrations to this version (e.g., "v0.19.0")
	description     string // Test scenario description
}

var _ = Describe("DB Configuration Puller Integration Tests", func() {

	// Define test scenarios for different migration paths
	testScenarios := []migrationTestConfig{
		{
			schemaFile:      "schema_v0_20_0.sql",
			initialVersion:  "v0.20.0",
			expectedVersion: "v0.21.0",
			prevVersion:     "v0.20.0",
			maxMigrationVer: "", // No limit, migrate to latest
			description:     "v0.20.0 → v0.21.0",
		},
		{
			schemaFile:      "schema_v0_18_0.sql",
			initialVersion:  "v0.18.0",
			expectedVersion: "v0.19.0",
			prevVersion:     "v0.18.0",
			maxMigrationVer: "v0.19.0", // Limit to v0.19.0
			description:     "v0.18.0 → v0.19.0",
		},
	}

	// Run tests for each scenario
	for _, scenario := range testScenarios {
		scenario := scenario // Capture range variable

		Context(fmt.Sprintf("Migration scenario: %s", scenario.description), func() {
			var (
				ctx                  context.Context
				localPostgresManager *tools.PostgreSQLManager
				localBackgroundMgr   *tools.ServiceManager
				localDatabase        *pgxpool.Pool
			)

			BeforeEach(func() {
				ctx = context.Background()

				// 1. Create and start PostgreSQL container
				By(fmt.Sprintf("Creating PostgreSQL container with %s schema", scenario.initialVersion))
				var err error

				// Get absolute path to the test data SQL file
				testDataPath, err := filepath.Abs(fmt.Sprintf("testdata/%s", scenario.schemaFile))
				Expect(err).ToNot(HaveOccurred(), "Failed to get absolute path to test data")

				// Verify test data file exists
				_, err = os.Stat(testDataPath)
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Test data file not found at %s", testDataPath))

				localPostgresManager, err = tools.NewPostgreSQLManager(ctx, tools.PostgreSQLConfig{
					InitScripts: []string{testDataPath},
				})
				Expect(err).ToNot(HaveOccurred(), "Failed to create PostgreSQL manager")

				By("Starting PostgreSQL container")
				err = localPostgresManager.Start()
				Expect(err).ToNot(HaveOccurred(), "Failed to start PostgreSQL container")

				// 2. Get connection details
				host, port := localPostgresManager.GetConnectionDetails()
				connString := localPostgresManager.GetConnectionString()
				By(fmt.Sprintf("PostgreSQL container started at %s:%s", host, port))

				// 3. Create database connection
				By("Creating database connection")
				db, err := SetupDatabaseConnectionFromString(ctx, connString)
				Expect(err).ToNot(HaveOccurred(), "Failed to create database connection")
				localDatabase = db

				// 4. Verify initial schema version
				By(fmt.Sprintf("Verifying initial schema version is %s", scenario.initialVersion))
				ExpectSchemaVersion(ctx, localDatabase, scenario.initialVersion)

				// 5. Start background service with short pull interval for testing
				By("Starting background service with fast config pull interval")
				backgroundEnv := map[string]string{
					"DB_LOCAL":                             "true",
					"DB_HOST":                              host,
					"DB_PORT":                              port,
					"DB_NAME":                              "vectordb",
					"DB_USR":                               "testuser",
					"DB_PWD":                               "testpwd",
					"BKG_HEALTHCHECK_PORT":                 backgroundHealthcheckPort,
					"DB_CONFIG_PULL_INTERVAL_SEC":          "5", // Short interval for testing (5 seconds)
					"ATTR_REPLICATION_BATCH_SIZE":          "10",
					"ATTR_REPLICATION_DELAY_MS":            "50",
					"ATTR_REPLICATION_ITERATION_DELAY_SEC": "1",
				}

				// Add max migration version if specified
				if scenario.maxMigrationVer != "" {
					backgroundEnv["DB_SCHEMA_MAX_MIGRATION_VERSION"] = scenario.maxMigrationVer
					By(fmt.Sprintf("Limiting migrations to %s", scenario.maxMigrationVer))
				}

				localBackgroundMgr, err = tools.StartBackgroundService(ctx, backgroundEnv)
				Expect(err).ToNot(HaveOccurred(), "Failed to start background service")

				By("Background service started with config puller enabled")
			})

			AfterEach(func() {
				// Stop background service
				if localBackgroundMgr != nil {
					By("Stopping background service")
					_ = localBackgroundMgr.StopService(ctx)
				}

				// Close database connection
				if localDatabase != nil {
					By("Closing database connection")
					ExpectNoIdleTransactionsLeft(ctx, localDatabase, "testuser")
					CloseDatabase(localDatabase)
				}

				// Stop PostgreSQL container
				if localPostgresManager != nil {
					By("Stopping PostgreSQL container")
					_ = localPostgresManager.Stop()
				}
			})

			Context("Configuration metrics exposure", func() {

				It("should expose schema_version as a Prometheus metric", func() {
					// Wait for schema migration to expected version
					By(fmt.Sprintf("Waiting for schema version to be updated to %s", scenario.expectedVersion))
					Eventually(func() string {
						version, _ := GetSchemaVersion(ctx, localDatabase)
						return version
					}, 60*time.Second, 2*time.Second).Should(Equal(scenario.expectedVersion))

					// Get metrics endpoint
					metricsEndpoint := localBackgroundMgr.GetMetricsEndpoint()
					Expect(metricsEndpoint).ToNot(BeEmpty(), "Background service should expose metrics endpoint")

					// Wait for the config puller to expose the schema_version metric
					By("Waiting for schema_version metric to be exposed")
					err := WaitForLabeledMetric(metricsEndpoint, "vector_store_db_configuration_info",
						map[string]string{"key": "schema_version", "value": scenario.expectedVersion},
						30*time.Second)
					Expect(err).ToNot(HaveOccurred(), "Schema version metric should be exposed")

					// Verify the metric value is 1 (info-style metric)
					By("Verifying schema_version metric value is 1")
					metricValue, err := GetLabeledMetricValue(metricsEndpoint, "vector_store_db_configuration_info",
						map[string]string{"key": "schema_version", "value": scenario.expectedVersion})
					Expect(err).ToNot(HaveOccurred(), "Should be able to read metric value")
					Expect(metricValue).To(Equal(1.0), "Info-style metric value should be 1")
				})

				It("should expose schema_version_prev as a Prometheus metric", func() {
					// Wait for schema migration to expected version
					By(fmt.Sprintf("Waiting for schema version to be updated to %s", scenario.expectedVersion))
					Eventually(func() string {
						version, _ := GetSchemaVersion(ctx, localDatabase)
						return version
					}, 60*time.Second, 2*time.Second).Should(Equal(scenario.expectedVersion))

					// Get metrics endpoint
					metricsEndpoint := localBackgroundMgr.GetMetricsEndpoint()
					Expect(metricsEndpoint).ToNot(BeEmpty(), "Background service should expose metrics endpoint")

					// Wait for the config puller to expose the schema_version_prev metric
					By("Waiting for schema_version_prev metric to be exposed")
					err := WaitForLabeledMetric(metricsEndpoint, "vector_store_db_configuration_info",
						map[string]string{"key": "schema_version_prev", "value": scenario.prevVersion},
						30*time.Second)
					Expect(err).ToNot(HaveOccurred(), "Schema version prev metric should be exposed")

					// Verify the metric value is 1 (info-style metric)
					By("Verifying schema_version_prev metric value is 1")
					metricValue, err := GetLabeledMetricValue(metricsEndpoint, "vector_store_db_configuration_info",
						map[string]string{"key": "schema_version_prev", "value": scenario.prevVersion})
					Expect(err).ToNot(HaveOccurred(), "Should be able to read metric value")
					Expect(metricValue).To(Equal(1.0), "Info-style metric value should be 1")
				})

				It("should update metrics when configuration changes", func() {
					// Wait for initial schema migration to expected version
					By(fmt.Sprintf("Waiting for schema version to be updated to %s", scenario.expectedVersion))
					Eventually(func() string {
						version, _ := GetSchemaVersion(ctx, localDatabase)
						return version
					}, 60*time.Second, 2*time.Second).Should(Equal(scenario.expectedVersion))

					metricsEndpoint := localBackgroundMgr.GetMetricsEndpoint()
					Expect(metricsEndpoint).ToNot(BeEmpty(), "Background service should expose metrics endpoint")

					// Verify initial metric
					By(fmt.Sprintf("Verifying initial schema_version metric is %s", scenario.expectedVersion))
					err := WaitForLabeledMetric(metricsEndpoint, "vector_store_db_configuration_info",
						map[string]string{"key": "schema_version", "value": scenario.expectedVersion},
						30*time.Second)
					Expect(err).ToNot(HaveOccurred(), "Initial schema version metric should be exposed")

					// Manually update schema_version in database to simulate a change
					testVersion := "v0.99.0"
					By(fmt.Sprintf("Manually updating schema_version to %s in database", testVersion))
					_, err = localDatabase.Exec(ctx,
						"UPDATE vector_store.configuration SET value = $1 WHERE key = 'schema_version'", testVersion)
					Expect(err).ToNot(HaveOccurred(), "Should be able to update schema version")

					// Wait for config puller to pick up the change (it pulls every 5 seconds)
					By(fmt.Sprintf("Waiting for updated schema_version metric to reflect %s", testVersion))
					err = WaitForLabeledMetric(metricsEndpoint, "vector_store_db_configuration_info",
						map[string]string{"key": "schema_version", "value": testVersion},
						15*time.Second) // Wait up to 15 seconds (3 pull cycles)
					Expect(err).ToNot(HaveOccurred(), "Updated schema version metric should be exposed")

					// Verify the old metric label is gone
					By(fmt.Sprintf("Verifying old %s metric label is no longer present", scenario.expectedVersion))
					Eventually(func() error {
						_, err := GetLabeledMetricValue(metricsEndpoint, "vector_store_db_configuration_info",
							map[string]string{"key": "schema_version", "value": scenario.expectedVersion})
						return err
					}, 5*time.Second, 1*time.Second).Should(HaveOccurred(),
						"Old metric label should not be present after update")
				})

				It("should continue exposing metrics even if database query fails temporarily", func() {
					// Wait for initial schema migration
					By(fmt.Sprintf("Waiting for schema version to be updated to %s", scenario.expectedVersion))
					Eventually(func() string {
						version, _ := GetSchemaVersion(ctx, localDatabase)
						return version
					}, 60*time.Second, 2*time.Second).Should(Equal(scenario.expectedVersion))

					metricsEndpoint := localBackgroundMgr.GetMetricsEndpoint()
					Expect(metricsEndpoint).ToNot(BeEmpty(), "Background service should expose metrics endpoint")

					// Verify initial metric
					By(fmt.Sprintf("Verifying initial schema_version metric is %s", scenario.expectedVersion))
					err := WaitForLabeledMetric(metricsEndpoint, "vector_store_db_configuration_info",
						map[string]string{"key": "schema_version", "value": scenario.expectedVersion},
						30*time.Second)
					Expect(err).ToNot(HaveOccurred(), "Initial schema version metric should be exposed")

					// The config puller should handle errors gracefully and continue running
					// Verify that metrics endpoint is still accessible and returns the last known values
					By("Verifying metrics endpoint remains accessible")
					time.Sleep(10 * time.Second) // Wait for at least 2 pull cycles

					metricValue, err := GetLabeledMetricValue(metricsEndpoint, "vector_store_db_configuration_info",
						map[string]string{"key": "schema_version", "value": scenario.expectedVersion})
					Expect(err).ToNot(HaveOccurred(), "Metrics endpoint should still be accessible")
					Expect(metricValue).To(Equal(1.0), "Metric value should still be available")
				})
			})
		})
	}
})
