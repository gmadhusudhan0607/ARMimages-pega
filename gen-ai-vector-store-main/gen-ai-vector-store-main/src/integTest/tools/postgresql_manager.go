// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgreSQLManager manages the lifecycle of a PostgreSQL container with pgvector extension
type PostgreSQLManager struct {
	container  *postgres.PostgresContainer
	connString string
	host       string
	port       string
	config     PostgreSQLConfig
	ctx        context.Context
}

// PostgreSQLConfig holds configuration for starting a PostgreSQL container
type PostgreSQLConfig struct {
	Image          string        // Container image (default: "docker-dev.bin.pega.io/pgvector/pgvector:0.8.0-pg17")
	User           string        // Database user (default: "testuser")
	Password       string        // Database password (default: "testpwd")
	Database       string        // Database name (default: "vectordb")
	StartupTimeout time.Duration // Container startup timeout (default: 60s)
	KeepRunning    bool          // Keep container running after tests for debugging
	InitScripts    []string      // SQL files to execute on container initialization (mounted to /docker-entrypoint-initdb.d/)
	ContainerLabel string        // Label to identify test containers (default: "genai-vector-store-test")
}

// NewPostgreSQLManager creates a new PostgreSQL manager with the given configuration.
// The container is not started until Start() is called.
func NewPostgreSQLManager(ctx context.Context, config PostgreSQLConfig) (*PostgreSQLManager, error) {
	// Apply defaults
	if config.Image == "" {
		config.Image = "docker-dev.bin.pega.io/pgvector/pgvector:0.8.1-pg18"
	}
	if config.User == "" {
		config.User = "testuser"
	}
	if config.Password == "" {
		config.Password = "testpwd"
	}
	if config.Database == "" {
		config.Database = "vectordb"
	}
	if config.StartupTimeout == 0 {
		config.StartupTimeout = 60 * time.Second
	}
	if config.ContainerLabel == "" {
		config.ContainerLabel = "genai-vector-store-test"
	}

	// Create manager with configuration (container not started yet)
	mgr := &PostgreSQLManager{
		config: config,
		ctx:    ctx,
	}

	return mgr, nil
}

// Start starts the PostgreSQL container and waits for it to be ready
func (pm *PostgreSQLManager) Start() error {
	if pm.container != nil {
		return fmt.Errorf("container already started")
	}

	// Check if KEEP mode is enabled
	keepRunning := pm.config.KeepRunning || os.Getenv("KEEP") == "true"

	// Create container request with labels for identification and cleanup
	req := testcontainers.ContainerRequest{
		Image:        pm.config.Image,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     pm.config.User,
			"POSTGRES_PASSWORD": pm.config.Password,
			"POSTGRES_DB":       pm.config.Database,
		},
		Labels: map[string]string{
			"test-container": pm.config.ContainerLabel,
			"created-at":     fmt.Sprintf("%d", time.Now().Unix()),
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(pm.config.StartupTimeout),
	}

	// Mount init scripts if provided
	if len(pm.config.InitScripts) > 0 {
		req.Files = make([]testcontainers.ContainerFile, 0, len(pm.config.InitScripts))
		for i, scriptPath := range pm.config.InitScripts {
			// Read the script file
			scriptContent, err := os.ReadFile(scriptPath)
			if err != nil {
				return fmt.Errorf("failed to read init script %s: %w", scriptPath, err)
			}

			// Generate a numbered filename to ensure proper execution order
			// Format: 00-file.sql, 01-file.sql, etc.
			targetPath := fmt.Sprintf("/docker-entrypoint-initdb.d/%02d-%s", i, filepath.Base(scriptPath))

			req.Files = append(req.Files, testcontainers.ContainerFile{
				HostFilePath:      scriptPath,
				ContainerFilePath: targetPath,
				FileMode:          0644,
			})

			fmt.Printf("Mounting init script: %s -> %s (%d bytes)\n", scriptPath, targetPath, len(scriptContent))
		}
	}

	// Use background context if KEEP mode is enabled to prevent container termination
	// when the test context is cancelled
	containerCtx := pm.ctx
	if keepRunning {
		containerCtx = context.Background()
		fmt.Println("KEEP=true: Using background context to persist container beyond test lifecycle")
	}

	// Start container
	genericContainer, err := testcontainers.GenericContainer(containerCtx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	// Wrap in postgres container type for convenience methods
	pm.container = &postgres.PostgresContainer{
		Container: genericContainer,
	}

	// Extract connection details
	if err := pm.updateConnectionDetails(); err != nil {
		_ = pm.container.Terminate(pm.ctx)
		pm.container = nil
		return fmt.Errorf("failed to get connection details: %w", err)
	}

	// Print connection string in red for easy troubleshooting
	fmt.Printf("\033[32m -> PostgreSQL Connection String: %s\033[0m\n", pm.connString)

	return nil
}

// updateConnectionDetails extracts and caches connection details from the container
func (pm *PostgreSQLManager) updateConnectionDetails() error {
	host, err := pm.container.Host(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := pm.container.MappedPort(context.Background(), "5432")
	if err != nil {
		return fmt.Errorf("failed to get mapped port: %w", err)
	}

	pm.host = host
	pm.port = mappedPort.Port()
	pm.connString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pm.config.User, pm.config.Password, pm.host, pm.port, pm.config.Database)

	return nil
}

// GetConnectionString returns the full connection string for the container
func (pm *PostgreSQLManager) GetConnectionString() string {
	return pm.connString
}

// GetConnectionDetails returns the host and port for the container
func (pm *PostgreSQLManager) GetConnectionDetails() (host, port string) {
	return pm.host, pm.port
}

// GetContainerID returns the container ID for debugging
func (pm *PostgreSQLManager) GetContainerID() string {
	return pm.container.GetContainerID()
}

// IsRunning checks if the container is still running
func (pm *PostgreSQLManager) IsRunning() bool {
	if pm.container == nil {
		return false
	}
	state, err := pm.container.State(pm.ctx)
	if err != nil {
		return false
	}
	return state.Running
}

// LoadSQLFile executes a SQL file against the running container
func (pm *PostgreSQLManager) LoadSQLFile(sqlFilePath string) error {
	if pm.container == nil {
		return fmt.Errorf("container not started")
	}

	// Read SQL file
	sqlContent, err := os.ReadFile(sqlFilePath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file %s: %w", sqlFilePath, err)
	}

	// Execute SQL using psql
	cmd := []string{
		"psql",
		"-U", pm.config.User,
		"-d", pm.config.Database,
		"-c", string(sqlContent),
	}

	exitCode, output, err := pm.container.Exec(pm.ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to execute SQL file: %w", err)
	}

	if exitCode != 0 {
		outputBytes, _ := io.ReadAll(output)
		return fmt.Errorf("SQL execution failed with exit code %d: %s", exitCode, string(outputBytes))
	}

	fmt.Printf("Successfully loaded SQL file: %s\n", sqlFilePath)
	return nil
}

// ExecuteSQL executes raw SQL commands against the running container
func (pm *PostgreSQLManager) ExecuteSQL(sql string) error {
	if pm.container == nil {
		return fmt.Errorf("container not started")
	}

	cmd := []string{
		"psql",
		"-U", pm.config.User,
		"-d", pm.config.Database,
		"-c", sql,
	}

	exitCode, output, err := pm.container.Exec(pm.ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	if exitCode != 0 {
		outputBytes, _ := io.ReadAll(output)
		return fmt.Errorf("SQL execution failed with exit code %d: %s", exitCode, string(outputBytes))
	}

	return nil
}

// LoadDump loads a PostgreSQL dump file into the running container
// Supports different dump formats:
// - "sql" or "plain": Plain SQL text file (from pg_dump -Fp or pg_dumpall)
// - "custom": Custom format (from pg_dump -Fc)
// - "directory": Directory format (from pg_dump -Fd)
// - "tar": Tar format (from pg_dump -Ft)
func (pm *PostgreSQLManager) LoadDump(dumpFilePath, format string) error {
	if pm.container == nil {
		return fmt.Errorf("container not started")
	}

	// Validate format
	validFormats := map[string]bool{
		"sql":       true,
		"plain":     true,
		"custom":    true,
		"directory": true,
		"tar":       true,
	}

	if !validFormats[format] {
		return fmt.Errorf("invalid dump format: %s (valid: sql, plain, custom, directory, tar)", format)
	}

	// Copy dump file to container
	targetPath := "/tmp/dump_import"
	if err := pm.container.CopyFileToContainer(pm.ctx, dumpFilePath, targetPath, 0644); err != nil {
		return fmt.Errorf("failed to copy dump file to container: %w", err)
	}

	var cmd []string

	// Choose appropriate restore command based on format
	switch format {
	case "sql", "plain":
		// Plain SQL file - use psql
		cmd = []string{
			"psql",
			"-U", pm.config.User,
			"-d", pm.config.Database,
			"-f", targetPath,
		}
	case "custom", "tar":
		// Custom or tar format - use pg_restore
		cmd = []string{
			"pg_restore",
			"-U", pm.config.User,
			"-d", pm.config.Database,
			"--no-owner",
			"--no-acl",
			targetPath,
		}
	case "directory":
		// Directory format requires special handling
		return fmt.Errorf("directory format dumps must be extracted before loading")
	}

	// Execute restore command
	exitCode, output, err := pm.container.Exec(pm.ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to execute dump restore: %w", err)
	}

	if exitCode != 0 {
		outputBytes, _ := io.ReadAll(output)
		return fmt.Errorf("dump restore failed with exit code %d: %s", exitCode, string(outputBytes))
	}

	fmt.Printf("Successfully loaded dump file: %s (format: %s)\n", dumpFilePath, format)
	return nil
}

// Stop stops the container unless KeepRunning is set or KEEP environment variable is set
func (pm *PostgreSQLManager) Stop() error {
	if pm.container == nil {
		return nil
	}

	// Parse KEEP mode configuration
	keepConfig, err := ParseKeepMode()
	if err != nil {
		return fmt.Errorf("failed to parse KEEP mode: %w", err)
	}

	// Check if we should keep the container running
	keepRunning := pm.config.KeepRunning || keepConfig.Enabled

	if keepRunning {
		containerID := pm.GetContainerID()

		if keepConfig.Enabled {
			// KEEP mode with duration - schedule cleanup
			fmt.Println("  ================================================================================")
			fmt.Printf("    KEEP mode: Container will remain running for %s\n", keepConfig.Duration)
			fmt.Printf("    Container ID: %s\n", containerID)
			fmt.Printf("    Connection: %s\n", pm.connString)
			fmt.Printf("    To stop manually: docker stop %s\n", containerID)
			fmt.Printf("    Auto-cleanup scheduled at: %s\n", time.Now().Add(keepConfig.Duration).Format(time.RFC3339))

			// Schedule cleanup using background context so it survives test completion
			keepConfig.ScheduleCleanup(context.Background(), fmt.Sprintf("PostgreSQL container %s", containerID), func() error {
				return pm.terminateContainer()
			})
		} else {
			// KeepRunning from config - no auto-cleanup
			fmt.Println("  ================================================================================")
			fmt.Println("    Container will remain running (KeepRunning=true in config)")
			fmt.Printf("    Container ID: %s\n", containerID)
			fmt.Printf("    Connection: %s\n", pm.connString)
			fmt.Printf("    To stop manually: docker stop %s\n", containerID)
		}

		return nil
	}

	// Normal cleanup - terminate immediately
	return pm.terminateContainer()
}

// terminateContainer terminates the PostgreSQL container
func (pm *PostgreSQLManager) terminateContainer() error {
	if pm.container == nil {
		return nil
	}

	if err := pm.container.Terminate(pm.ctx); err != nil {
		return fmt.Errorf("failed to terminate container: %w", err)
	}

	return nil
}

// CleanupOrphanedContainers removes old test containers from previous runs
// This should be called at the beginning of test suites to ensure a clean state
// Note: This ALWAYS cleans up old containers regardless of KEEP setting to ensure fresh test runs
func CleanupOrphanedContainers(ctx context.Context, labelFilter string) error {
	if labelFilter == "" {
		labelFilter = "genai-vector-store-test"
	}

	// Use docker CLI to find and remove containers with the test label
	// We use CLI instead of testcontainers API for broader compatibility
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("label=test-container=%s", labelFilter), "--format", "{{.ID}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If docker is not available or returns an error, log and continue
		fmt.Printf("Warning: Could not list containers for cleanup: %v\n", err)
		return nil
	}

	containerIDs := strings.Fields(strings.TrimSpace(string(output)))
	if len(containerIDs) == 0 {
		return nil
	}

	fmt.Printf("Found %d orphaned test container(s) from previous runs, cleaning up...\n", len(containerIDs))

	for _, containerID := range containerIDs {
		// Always force remove containers to ensure clean test setup
		// KEEP=true only applies to containers from the CURRENT test run, not old ones
		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerID)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to remove container %s: %v\n", containerID, err)
		} else {
			fmt.Printf("Removed orphaned container: %s\n", containerID)
		}
	}

	return nil
}
