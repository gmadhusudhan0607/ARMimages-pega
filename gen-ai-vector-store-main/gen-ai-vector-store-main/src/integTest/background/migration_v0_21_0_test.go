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

var _ = Describe("Migration v0.21.0 Integration Tests", func() {

	var (
		ctx                  context.Context
		localPostgresManager *tools.PostgreSQLManager
		localBackgroundMgr   *tools.ServiceManager
		localDatabase        *pgxpool.Pool
	)

	BeforeEach(func() {
		ctx = context.Background()

		// 1. Create and start PostgreSQL container
		By("Creating PostgreSQL container for v0.20.0 schema (pre-v0.21.0)")
		var err error

		// Get absolute path to the test data SQL file
		testDataPath, err := filepath.Abs("testdata/schema_v0_20_0.sql")
		Expect(err).ToNot(HaveOccurred(), "Failed to get absolute path to test data")

		// Verify test data file exists
		_, err = os.Stat(testDataPath)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Test data file not found at %s", testDataPath))

		localPostgresManager, err = tools.NewPostgreSQLManager(ctx, tools.PostgreSQLConfig{
			InitScripts: []string{testDataPath},
		})
		Expect(err).ToNot(HaveOccurred(), "Failed to create PostgreSQL manager")

		By("Starting PostgreSQL container with v0.20.0 schema")
		err = localPostgresManager.Start()
		Expect(err).ToNot(HaveOccurred(), "Failed to start PostgreSQL container")

		// 2. Get connection details
		host, port := localPostgresManager.GetConnectionDetails()
		connString := localPostgresManager.GetConnectionString()
		By(fmt.Sprintf("PostgreSQL container started at %s:%s", host, port))

		// 3. Verify initial schema version is v0.20.0
		By("Verifying initial schema version is v0.20.0")
		db, err := SetupDatabaseConnectionFromString(ctx, connString)
		Expect(err).ToNot(HaveOccurred(), "Failed to create database connection")
		localDatabase = db

		ExpectSchemaVersion(ctx, localDatabase, "v0.20.0")

		// 4. Verify pdc_endpoint_url column does NOT exist yet (v0.20.0 state)
		By("Verifying pdc_endpoint_url column does not exist in v0.20.0 schema")
		ExpectColumnDoesNotExist(ctx, localDatabase, "vector_store", "isolations", "pdc_endpoint_url")

		// 5. Start background service (this will trigger migration to v0.21.0)
		By("Starting background service to trigger migration")
		backgroundEnv := map[string]string{
			"DB_LOCAL":             "true",
			"DB_HOST":              host,
			"DB_PORT":              port,
			"DB_NAME":              "vectordb",
			"DB_USR":               "testuser",
			"DB_PWD":               "testpwd",
			"BKG_HEALTHCHECK_PORT": backgroundHealthcheckPort,
		}
		localBackgroundMgr, err = tools.StartBackgroundService(ctx, backgroundEnv)
		Expect(err).ToNot(HaveOccurred(), "Failed to start background service")

		By("Background service started - migration to v0.21.0 should be in progress")
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

	Context("Schema migration from v0.20.0 to v0.21.0", func() {

		It("should successfully migrate schema to v0.21.0 and add pdc_endpoint_url column", func() {
			// Wait for schema version to be updated to v0.21.0
			By("Waiting for schema version to be updated to v0.21.0")
			Eventually(func() string {
				version, _ := GetSchemaVersion(ctx, localDatabase)
				return version
			}, 60*time.Second, 2*time.Second).Should(Equal("v0.21.0"))

			// Verify pdc_endpoint_url column was added by migration
			By("Verifying pdc_endpoint_url column was added to isolations table")
			ExpectColumnExists(ctx, localDatabase, "vector_store", "isolations", "pdc_endpoint_url")

			// Verify the column is nullable (TEXT type, no NOT NULL constraint)
			By("Verifying pdc_endpoint_url column is nullable")
			query := `
				SELECT is_nullable, data_type 
				FROM information_schema.columns 
				WHERE table_schema = 'vector_store' 
				  AND table_name = 'isolations' 
				  AND column_name = 'pdc_endpoint_url'
			`
			var isNullable, dataType string
			err := localDatabase.QueryRow(ctx, query).Scan(&isNullable, &dataType)
			Expect(err).ToNot(HaveOccurred(), "Should be able to query column information")
			Expect(isNullable).To(Equal("YES"), "pdc_endpoint_url should be nullable")
			Expect(dataType).To(Equal("text"), "pdc_endpoint_url should be TEXT type")

			// Verify migration configuration was recorded
			By("Verifying migration configuration was recorded")
			configKey := "schema_migration_v0.21.0_add_pdc_endpoint_url"
			var configValue string
			configQuery := `SELECT value FROM vector_store.configuration WHERE key = $1`
			err = localDatabase.QueryRow(ctx, configQuery, configKey).Scan(&configValue)
			Expect(err).ToNot(HaveOccurred(), "Migration configuration should exist")
			Expect(configValue).To(Equal("completed"), "Migration should be marked as completed")
		})

		It("should preserve all existing isolation data during migration", func() {
			// Wait for migration to complete
			By("Waiting for schema version to be updated to v0.21.0")
			Eventually(func() string {
				version, _ := GetSchemaVersion(ctx, localDatabase)
				return version
			}, 60*time.Second, 2*time.Second).Should(Equal("v0.21.0"))

			// Verify all test isolations still exist with their original data
			By("Verifying all test isolations still exist")
			query := `
				SELECT iso_id, iso_prefix, max_storage_size, pdc_endpoint_url 
				FROM vector_store.isolations 
				ORDER BY iso_id
			`
			rows, err := localDatabase.Query(ctx, query)
			Expect(err).ToNot(HaveOccurred(), "Should be able to query isolations")
			defer rows.Close()

			isolations := make(map[string]struct {
				prefix         string
				maxStorageSize *string
				pdcEndpointURL *string
			})

			for rows.Next() {
				var isoID, isoPrefix string
				var maxStorageSize, pdcEndpointURL *string
				err := rows.Scan(&isoID, &isoPrefix, &maxStorageSize, &pdcEndpointURL)
				Expect(err).ToNot(HaveOccurred(), "Should be able to scan row")
				isolations[isoID] = struct {
					prefix         string
					maxStorageSize *string
					pdcEndpointURL *string
				}{isoPrefix, maxStorageSize, pdcEndpointURL}
			}

			// Verify we have all 4 test isolations
			Expect(len(isolations)).To(Equal(4), "Should have 4 test isolations")

			// Verify specific isolation data
			Expect(isolations["iso-empty"].prefix).To(Equal("e5af55acf1ba36f60b1e55a5d57f4b7d"))
			Expect(*isolations["iso-empty"].maxStorageSize).To(Equal("1GB"))
			Expect(isolations["iso-empty"].pdcEndpointURL).To(BeNil(), "pdc_endpoint_url should be NULL for existing isolations")

			Expect(isolations["iso-test-1"].prefix).To(Equal("6f909e9b46455b62a7337a75311a25eb"))
			Expect(*isolations["iso-test-1"].maxStorageSize).To(Equal("2GB"))
			Expect(isolations["iso-test-1"].pdcEndpointURL).To(BeNil())

			Expect(isolations["iso-test-2"].prefix).To(Equal("76a3008adb7cb8f988ba492ad034e815"))
			Expect(*isolations["iso-test-2"].maxStorageSize).To(Equal("5GB"))
			Expect(isolations["iso-test-2"].pdcEndpointURL).To(BeNil())

			Expect(isolations["iso-test-3"].prefix).To(Equal("c5a5bdd11f93ee7c40c46a71351b83f0"))
			Expect(isolations["iso-test-3"].maxStorageSize).To(BeNil(), "iso-test-3 has NULL max_storage_size")
			Expect(isolations["iso-test-3"].pdcEndpointURL).To(BeNil())
		})

		It("should allow inserting new isolations with pdc_endpoint_url after migration", func() {
			// Wait for migration to complete
			By("Waiting for schema version to be updated to v0.21.0")
			Eventually(func() string {
				version, _ := GetSchemaVersion(ctx, localDatabase)
				return version
			}, 60*time.Second, 2*time.Second).Should(Equal("v0.21.0"))

			// Insert a new isolation with pdc_endpoint_url
			By("Inserting new isolation with pdc_endpoint_url")
			insertQuery := `
				INSERT INTO vector_store.isolations 
					(iso_id, iso_prefix, max_storage_size, pdc_endpoint_url, created_at, modified_at)
				VALUES 
					($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`
			_, err := localDatabase.Exec(ctx, insertQuery,
				"iso-new-with-pdc",
				"newisohash123",
				"10GB",
				"https://pdc.example.com/endpoint")
			Expect(err).ToNot(HaveOccurred(), "Should be able to insert isolation with pdc_endpoint_url")

			// Verify the new isolation was inserted correctly
			By("Verifying new isolation with pdc_endpoint_url")
			var pdcURL string
			selectQuery := `SELECT pdc_endpoint_url FROM vector_store.isolations WHERE iso_id = $1`
			err = localDatabase.QueryRow(ctx, selectQuery, "iso-new-with-pdc").Scan(&pdcURL)
			Expect(err).ToNot(HaveOccurred(), "Should be able to query new isolation")
			Expect(pdcURL).To(Equal("https://pdc.example.com/endpoint"))

			// Insert another isolation without pdc_endpoint_url (NULL)
			By("Inserting new isolation without pdc_endpoint_url")
			_, err = localDatabase.Exec(ctx, insertQuery,
				"iso-new-without-pdc",
				"newisohash456",
				"5GB",
				nil)
			Expect(err).ToNot(HaveOccurred(), "Should be able to insert isolation without pdc_endpoint_url")

			// Verify NULL value is stored correctly
			By("Verifying new isolation without pdc_endpoint_url has NULL value")
			var pdcURLNull *string
			err = localDatabase.QueryRow(ctx, selectQuery, "iso-new-without-pdc").Scan(&pdcURLNull)
			Expect(err).ToNot(HaveOccurred(), "Should be able to query new isolation")
			Expect(pdcURLNull).To(BeNil(), "pdc_endpoint_url should be NULL")
		})
	})
})
