// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package background

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/functions"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/src/integTest/tools"
	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Migration Switching Integration Tests", func() {

	var (
		ctx                  context.Context
		localPostgresManager *tools.PostgreSQLManager
		localBackgroundMgr   *tools.ServiceManager
		localMainServiceMgr  *tools.ServiceManager
		localDatabase        *pgxpool.Pool
		mockID               string // Track WireMock mock ID for cleanup
	)

	BeforeEach(func() {
		ctx = context.Background()

		// 1. Create and start PostgreSQL container with v0.18.0 schema
		By("Creating PostgreSQL container for v0.18.0 schema")
		var err error

		testDataPath, err := filepath.Abs("testdata/schema_v0_18_0.sql")
		Expect(err).ToNot(HaveOccurred(), "Failed to get absolute path to test data")

		_, err = os.Stat(testDataPath)
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Test data file not found at %s", testDataPath))

		localPostgresManager, err = tools.NewPostgreSQLManager(ctx, tools.PostgreSQLConfig{
			InitScripts: []string{testDataPath},
		})
		Expect(err).ToNot(HaveOccurred(), "Failed to create PostgreSQL manager")

		By("Starting PostgreSQL container with v0.18.0 schema")
		err = localPostgresManager.Start()
		Expect(err).ToNot(HaveOccurred(), "Failed to start PostgreSQL container")

		host, port := localPostgresManager.GetConnectionDetails()
		connString := localPostgresManager.GetConnectionString()
		By(fmt.Sprintf("PostgreSQL container started at %s:%s", host, port))

		// 2. Create database connection
		By("Creating database connection")
		db, err := SetupDatabaseConnectionFromString(ctx, connString)
		Expect(err).ToNot(HaveOccurred(), "Failed to create database connection")
		localDatabase = db

		// 3. Verify initial schema version is v0.18.0
		By("Verifying initial schema version is v0.18.0")
		ExpectSchemaVersion(ctx, localDatabase, "v0.18.0")

		// 4. Create WireMock expectations for embedder
		By("Creating WireMock expectations for iso-test-1")
		mockID, err = CreateExpectationEmbeddingAda(wiremockManager, "iso-test-1")
		Expect(err).ToNot(HaveOccurred(), "Failed to create WireMock expectations")

		// 5. Start background service (triggers migration)
		By("Starting background service to trigger migration")
		wiremockURL := wiremockManager.GetBaseURL()
		backgroundEnv := map[string]string{
			"LOG_LEVEL":                            "DEBUG",
			"DB_LOCAL":                             "true",
			"DB_HOST":                              host,
			"DB_PORT":                              port,
			"DB_NAME":                              "vectordb",
			"DB_USR":                               "testuser",
			"DB_PWD":                               "testpwd",
			"BKG_HEALTHCHECK_PORT":                 backgroundHealthcheckPort,
			"ATTR_REPLICATION_BATCH_SIZE":          "10", // Fast migration settings for testing
			"ATTR_REPLICATION_DELAY_MS":            "50", // Fast migration settings for testing
			"ATTR_REPLICATION_ITERATION_DELAY_SEC": "1",  // Fast migration settings for testing
			"GENAI_GATEWAY_SERVICE_URL":            wiremockURL,
			"SAX_DISABLED":                         "true",
			"SAX_CLIENT_DISABLED":                  "true", // Disable SAX client for testing (prevents HTTP client from requiring SAX_CLIENT_SECRET)
			// Limit migrations to v0.19.0 for this test
			"DB_SCHEMA_MAX_MIGRATION_VERSION": "v0.19.0",
		}
		localBackgroundMgr, err = tools.StartBackgroundService(ctx, backgroundEnv)
		Expect(err).ToNot(HaveOccurred(), "Failed to start background service")

		By("Background service started - migration to v0.19.0 should be in progress")

		// 6. Start main service with WireMock as embedder endpoint
		By("Starting main service with WireMock as embedder endpoint")
		mainServiceEnv := map[string]string{
			"LOG_LEVEL":                       "DEBUG",
			"DB_LOCAL":                        "true",
			"DB_HOST":                         host,
			"DB_PORT":                         port,
			"DB_NAME":                         "vectordb",
			"DB_USR":                          "testuser",
			"DB_PWD":                          "testpwd",
			"SERVICE_PORT":                    mainServicePort,
			"SVC_HEALTHCHECK_PORT":            mainHealthcheckPort, // For service manager validation
			"SERVICE_HEALTHCHECK_PORT":        mainHealthcheckPort, // For actual service binary
			"SAX_DISABLED":                    "true",              // Disable SAX for testing
			"SAX_CLIENT_DISABLED":             "true",              // Disable SAX client for testing (prevents HTTP client from requiring SAX_CLIENT_SECRET)
			"INJECT_TEST_HEADERS":             "true",              // Enable test header injection for embedder
			"GENAI_GATEWAY_SERVICE_URL":       wiremockURL,         // Point to WireMock for embeddings
			"RUNTIME_CONFIG_PULL_INTERVAL_MS": "1000",              // Enable runtime config puller with 1s interval to load schema
			// Set forced min version to v0.19.0 so service accepts v0.19.0 schema
			"DB_SCHEMA_FORCED_MIN_REQUIRED_VERSION": "v0.19.0",
		}
		localMainServiceMgr, err = tools.StartMainService(ctx, mainServiceEnv)
		Expect(err).ToNot(HaveOccurred(), "Failed to start main service")

		By("Main service started and ready")

		// Wait for schema/configuration to be loaded into the service
		// The runtime config puller runs every 1 second, so we need to wait at least 2-3 iterations
		// to ensure the schema has been loaded from the database
		By("Waiting for schema to be loaded into main service")
		time.Sleep(4 * time.Second)
	})

	AfterEach(func() {
		// Stop main service
		if localMainServiceMgr != nil {
			By("Stopping main service")
			_ = localMainServiceMgr.StopService(ctx)
		}

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

		// Delete WireMock expectation created by this test
		if mockID != "" {
			By("Deleting WireMock expectation")
			err := DeleteExpectationIfExist(wiremockManager, mockID)
			if err != nil {
				GinkgoWriter.Printf("Warning: Failed to delete WireMock expectation: %v\n", err)
			}
		}
	})

	Context("Query endpoint switching during migration", func() {

		It("should switch from old to new query functions when migration completes", func() {
			// Define query endpoints
			documentsURL := fmt.Sprintf("%s/v1/iso-test-1/collections/col-1a/query/documents", baseURI)
			chunksURL := fmt.Sprintf("%s/v1/iso-test-1/collections/col-1a/query/chunks", baseURI)

			// Define query payloads with maxDistance=1.0 to accept all results
			documentsQuery := map[string]interface{}{
				"filters": map[string]interface{}{
					"query": "test query",
				},
				"maxDistance": 1.0,
				"limit":       10,
			}

			chunksQuery := map[string]interface{}{
				"filters": map[string]interface{}{
					"query": "test query",
				},
				"maxDistance": 1.0,
				"limit":       10,
			}

			// Phase 1: Verify old functions are used BEFORE migration completes
			By("Verifying old functions are used before migration completes")

			// Check documents endpoint header
			Eventually(func() string {
				resp := makeQueryRequest(documentsURL, documentsQuery)
				if resp == nil {
					GinkgoWriter.Println("makeQueryRequest returned nil response")
					return ""
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					bodyBytes, _ := io.ReadAll(resp.Body)
					GinkgoWriter.Printf("Query returned non-200 status: %d, body: %s\n", resp.StatusCode, string(bodyBytes))
					return ""
				}
				header := resp.Header.Get("X-Genai-Vectorstore-Db-Schema-Migration")
				GinkgoWriter.Printf("Got header value: '%s'\n", header)
				return header
			}, 10*time.Second, 1*time.Second).Should(Equal("incompleted"),
				"Documents endpoint should use old functions (FindDocuments2) before migration")

			// Check chunks endpoint header
			Eventually(func() string {
				resp := makeQueryRequest(chunksURL, chunksQuery)
				if resp == nil {
					return ""
				}
				defer resp.Body.Close()
				return resp.Header.Get("X-Genai-Vectorstore-Db-Schema-Migration")
			}, 10*time.Second, 1*time.Second).Should(Equal("incompleted"),
				"Chunks endpoint should use old functions (FindChunks2) before migration")

			By("Confirmed: Old query functions are being used before migration")

			// Phase 2: Wait for migration to complete
			By("Waiting for migration to complete")
			err := WaitForAttributesMigration(ctx, localDatabase, "iso-test-1", "col-1a",
				"openai-text-embedding-ada-002", 60*time.Second)
			Expect(err).To(BeNil(), "Attribute replication should complete")

			By("Migration completed successfully")

			// Phase 3: Verify new functions are used AFTER migration completes
			By("Verifying new functions are used after migration completes")

			// Check documents endpoint header
			Eventually(func() string {
				resp := makeQueryRequest(documentsURL, documentsQuery)
				if resp == nil {
					return ""
				}
				defer resp.Body.Close()
				return resp.Header.Get("X-Genai-Vectorstore-Db-Schema-Migration")
			}, 10*time.Second, 1*time.Second).Should(Equal("completed"),
				"Documents endpoint should use new functions (FindDocuments) after migration")

			// Check chunks endpoint header
			Eventually(func() string {
				resp := makeQueryRequest(chunksURL, chunksQuery)
				if resp == nil {
					return ""
				}
				defer resp.Body.Close()
				return resp.Header.Get("X-Genai-Vectorstore-Db-Schema-Migration")
			}, 10*time.Second, 1*time.Second).Should(Equal("completed"),
				"Chunks endpoint should use new functions (FindChunks) after migration")

			By("Confirmed: New query functions are being used after migration")

			// Phase 4: Verify queries return valid results
			By("Verifying queries return valid results with new functions")

			// Query documents and verify response
			resp := makeQueryRequest(documentsURL, documentsQuery)
			Expect(resp).ToNot(BeNil(), "Documents query should return a response")
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK), "Documents query should return 200 OK")

			bodyBytes, err := io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred(), "Should be able to read documents response body")

			var docs []map[string]interface{}
			err = json.Unmarshal(bodyBytes, &docs)
			Expect(err).ToNot(HaveOccurred(), "Should be able to parse documents JSON response")
			Expect(len(docs)).To(BeNumerically(">", 0), "Documents query should return at least one result")

			By(fmt.Sprintf("Documents query returned %d results", len(docs)))

			// Query chunks and verify response
			resp = makeQueryRequest(chunksURL, chunksQuery)
			Expect(resp).ToNot(BeNil(), "Chunks query should return a response")
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK), "Chunks query should return 200 OK")

			bodyBytes, err = io.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred(), "Should be able to read chunks response body")

			var chunks []map[string]interface{}
			err = json.Unmarshal(bodyBytes, &chunks)
			Expect(err).ToNot(HaveOccurred(), "Should be able to parse chunks JSON response")
			Expect(len(chunks)).To(BeNumerically(">", 0), "Chunks query should return at least one result")

			By(fmt.Sprintf("Chunks query returned %d results", len(chunks)))

			By("Test completed: Migration switching verified successfully")

		})
	})
})

// makeQueryRequest is a helper function to make HTTP POST requests to query endpoints
func makeQueryRequest(url string, query map[string]interface{}) *http.Response {
	payload, err := json.Marshal(query)
	if err != nil {
		GinkgoWriter.Printf("Failed to marshal query: %v\n", err)
		return nil
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		GinkgoWriter.Printf("Failed to create request: %v\n", err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		GinkgoWriter.Printf("Failed to execute request: %v\n", err)
		return nil
	}

	return resp
}
