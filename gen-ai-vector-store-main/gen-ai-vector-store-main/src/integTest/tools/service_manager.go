// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package tools

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ServiceManager manages the lifecycle of a service process0
type ServiceManager struct {
	cmd         *exec.Cmd
	serviceName string
	healthURL   string
	logFile     *os.File
	port        string
	servicePort string // Port where the service API is exposed (different from health check port)
}

// ServiceConfig holds configuration for starting a service
type ServiceConfig struct {
	SourcePath      string
	BinaryPath      string
	ServiceName     string
	HealthCheckPort string
	ServicePort     string // Port where the service API is exposed (empty for background service)
	Environment     map[string]string
	LogFilePath     string
}

// newServiceManager starts a service with the given configuration (private constructor)
func newServiceManager(ctx context.Context, config ServiceConfig) (*ServiceManager, error) {
	// Build service binary if not exists
	if _, err := os.Stat(config.BinaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("service binary not found at %s, please build it first", config.BinaryPath)
	}

	// Create log file directory if it doesn't exist
	logDir := filepath.Dir(config.LogFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file for service output
	logFile, err := os.Create(config.LogFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Create command
	cmd := exec.CommandContext(ctx, config.BinaryPath)

	// Set up environment variables
	cmd.Env = os.Environ()
	for key, value := range config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Redirect stdout and stderr to log file
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Start the service
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to start service: %w", err)
	}

	sm := &ServiceManager{
		cmd:         cmd,
		serviceName: config.ServiceName,
		healthURL:   fmt.Sprintf("http://localhost:%s", config.HealthCheckPort),
		logFile:     logFile,
		port:        config.HealthCheckPort,
		servicePort: config.ServicePort,
	}

	return sm, nil
}

// WaitForReady waits for the service to become ready by checking health endpoints
func (sm *ServiceManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		// Check if the process is still running
		if !sm.IsRunning() {
			logs, _ := sm.GetLogs()
			return fmt.Errorf("service process exited unexpectedly, logs:\n%s", logs)
		}

		// Try readiness endpoint
		resp, err := client.Get(sm.healthURL + "/health/readiness")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			logs, _ := sm.GetLogs()
			return fmt.Errorf("context cancelled while waiting for service, logs:\n%s", logs)
		case <-time.After(500 * time.Millisecond):
			continue
		}
	}

	logs, _ := sm.GetLogs()
	return fmt.Errorf("service failed to become ready within %v, logs:\n%s", timeout, logs)
}

// StopService gracefully stops the service
func (sm *ServiceManager) StopService(ctx context.Context) error {
	if sm.cmd == nil || sm.cmd.Process == nil {
		return nil
	}

	// Parse KEEP mode configuration
	keepConfig, err := ParseKeepMode()
	if err != nil {
		return fmt.Errorf("failed to parse KEEP mode: %w", err)
	}

	// If KEEP mode is enabled, schedule cleanup and return
	if keepConfig.Enabled {
		pid := sm.cmd.Process.Pid
		logPath, err := filepath.Abs(sm.logFile.Name())
		if err != nil {
			// Fallback to relative path if absolute path cannot be determined
			logPath = sm.logFile.Name()
		}

		fmt.Println("  ================================================================================")
		fmt.Printf("    KEEP mode: Service '%s' will remain running for %s\n", sm.serviceName, keepConfig.Duration)
		fmt.Printf("    Process ID: %d\n", pid)
		fmt.Printf("    Log file: %s\n", logPath)
		fmt.Printf("    Health check: %s\n", sm.GetHealthcheckEndpoint())
		fmt.Printf("    Metrics: %s\n", sm.GetMetricsEndpoint())
		fmt.Printf("    To stop manually: kill %d\n", pid)
		fmt.Printf("    Auto-cleanup scheduled at: %s\n", time.Now().Add(keepConfig.Duration).Format(time.RFC3339))

		// Schedule cleanup using background context so it survives test completion
		keepConfig.ScheduleCleanup(context.Background(), fmt.Sprintf("Service '%s' (PID %d)", sm.serviceName, pid), func() error {
			return sm.terminateProcess()
		})

		return nil
	}

	// Normal cleanup - terminate immediately
	return sm.terminateProcess()
}

// terminateProcess gracefully terminates the service process
func (sm *ServiceManager) terminateProcess() error {
	if sm.cmd == nil || sm.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown
	if err := sm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for graceful shutdown (5 seconds)
	done := make(chan error, 1)
	go func() {
		done <- sm.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill if graceful shutdown times out
		if err := sm.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		<-done // Wait for process to actually exit
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" {
			return fmt.Errorf("service exited with error: %w", err)
		}
	}

	// Close log file
	if sm.logFile != nil {
		sm.logFile.Close()
	}

	return nil
}

// GetLogs returns the captured service logs
func (sm *ServiceManager) GetLogs() (string, error) {
	if sm.logFile == nil {
		return "", fmt.Errorf("no log file available")
	}

	// Sync to ensure all writes are flushed
	_ = sm.logFile.Sync()

	// Read a log file
	content, err := os.ReadFile(sm.logFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	return string(content), nil
}

// IsRunning checks if the service process is still running
func (sm *ServiceManager) IsRunning() bool {
	if sm.cmd == nil || sm.cmd.Process == nil {
		return false
	}

	// Send signal 0 to check if a process exists
	err := sm.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// GetMetricsEndpoint returns the URL for the Prometheus metrics endpoint
// Returns empty string if the service doesn't expose metrics (e.g., ops service)
func (sm *ServiceManager) GetMetricsEndpoint() string {
	// Ops service doesn't expose Prometheus metrics
	if sm.serviceName == "ops-test" {
		return "N/A"
	}
	// Background and main services expose metrics on health check port
	return fmt.Sprintf("http://localhost:%s/metrics", sm.port)
}

// GetHealthcheckEndpoint returns the URL for the health check endpoint
func (sm *ServiceManager) GetHealthcheckEndpoint() string {
	return fmt.Sprintf("http://localhost:%s/health/readiness", sm.port)
}

// GetServiceEndpoint returns the URL for the service endpoint
// Returns empty string if the service doesn't expose a service endpoint (e.g., background service)
func (sm *ServiceManager) GetServiceEndpoint() string {
	// Background service doesn't expose a service endpoint
	if sm.serviceName == "background-test" {
		return "N/A"
	}

	// Use the stored service port from the environment
	if sm.servicePort != "" {
		return fmt.Sprintf("http://localhost:%s", sm.servicePort)
	}

	// Fallback to default ports if service port wasn't provided
	var defaultPort string
	switch sm.serviceName {
	case "main-test":
		defaultPort = "8080"
	case "ops-test":
		defaultPort = "8090"
	default:
		return ""
	}

	return fmt.Sprintf("http://localhost:%s", defaultPort)
}

// printGreen prints text in green color using ANSI escape codes
func printGreen(text string) {
	fmt.Printf("\033[32m%s\033[0m\n", text)
}

// findProjectRoot finds the project root directory by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Join(dir, "..")
		parentAbs, err := filepath.Abs(parent)
		if err != nil {
			return "", err
		}

		// If we've reached the root, stop
		if parentAbs == dir {
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}

		dir = parentAbs
	}
}

// buildServiceBinary builds a service binary for testing using the provided configuration
// This function now uses the global build cache to avoid unnecessary rebuilds
func buildServiceBinary(ctx context.Context, config ServiceConfig) error {
	cache := GetBuildCache()
	_, err := cache.EnsureBinary(ctx, config)
	return err
}

// StartBackgroundService builds and starts the background service with given configuration
func StartBackgroundService(ctx context.Context, env map[string]string) (*ServiceManager, error) {
	// Extract health check port from environment
	healthCheckPort, ok := env["BKG_HEALTHCHECK_PORT"]
	if !ok {
		return nil, fmt.Errorf("BKG_HEALTHCHECK_PORT must be specified in environment map")
	}

	// Find project root to build absolute binary path
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	// Create service configuration
	config := ServiceConfig{
		SourcePath:      "./cmd/background",
		BinaryPath:      "bin/background-test",
		ServiceName:     "background-test",
		HealthCheckPort: healthCheckPort,
		ServicePort:     "", // Background service doesn't expose a service API
		Environment:     env,
		LogFilePath:     "logs/background-test.log",
	}

	// Build the binary
	println("Building background service...")
	if err := buildServiceBinary(ctx, config); err != nil {
		return nil, err
	}

	// Update BinaryPath to absolute path for service manager
	config.BinaryPath = filepath.Join(projectRoot, config.BinaryPath)

	// Start the service
	println("Starting background service...")
	manager, err := newServiceManager(ctx, config)
	if err != nil {
		return nil, err
	}

	err = manager.WaitForReady(ctx, 30*time.Second)
	if err != nil {
		_ = manager.StopService(ctx)
		return nil, fmt.Errorf("background service failed to become ready: %w", err)
	}

	// Print service URLs and log file location
	printGreen(fmt.Sprintf(" -> Service URL: %s", manager.GetServiceEndpoint()))
	printGreen(fmt.Sprintf(" -> Metrics URL: %s", manager.GetMetricsEndpoint()))
	printGreen(fmt.Sprintf(" -> HealthCheck URL: %s", manager.GetHealthcheckEndpoint()))
	logPath, _ := filepath.Abs(manager.logFile.Name())
	printGreen(fmt.Sprintf(" -> Log file: %s", logPath))

	return manager, nil
}

// StartOpsService builds and starts the ops service with given configuration
func StartOpsService(ctx context.Context, env map[string]string) (*ServiceManager, error) {
	// Extract health check port from environment
	opsHealthcheckPort, ok := env["OPS_HEALTHCHECK_PORT"]
	if !ok {
		return nil, fmt.Errorf("OPS_HEALTHCHECK_PORT must be specified in environment map")
	}

	// Extract service port from environment
	opsPort, ok := env["OPS_PORT"]
	if !ok {
		return nil, fmt.Errorf("OPS_PORT must be specified in environment map")
	}

	// Find project root to build absolute binary path
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	// Create service configuration
	config := ServiceConfig{
		SourcePath:      "./cmd/ops",
		BinaryPath:      "bin/ops-test",
		ServiceName:     "ops-test",
		HealthCheckPort: opsHealthcheckPort,
		ServicePort:     opsPort,
		Environment:     env,
		LogFilePath:     "logs/ops-test.log",
	}

	// Build the binary
	println("Building ops service...")
	if err := buildServiceBinary(ctx, config); err != nil {
		return nil, err
	}

	// Update BinaryPath to absolute path for service manager
	config.BinaryPath = filepath.Join(projectRoot, config.BinaryPath)

	// Start the service
	println("Starting ops service...")
	manager, err := newServiceManager(ctx, config)
	if err != nil {
		return nil, err
	}

	err = manager.WaitForReady(ctx, 30*time.Second)
	if err != nil {
		_ = manager.StopService(ctx)
		return nil, fmt.Errorf("ops service failed to become ready: %w", err)
	}

	// Print service URLs and log file location
	printGreen(fmt.Sprintf(" -> Service URL: %s", manager.GetServiceEndpoint()))
	printGreen(fmt.Sprintf(" -> Metrics URL: %s", manager.GetMetricsEndpoint()))
	printGreen(fmt.Sprintf(" -> HealthCheck URL: %s", manager.GetHealthcheckEndpoint()))
	logPath, _ := filepath.Abs(manager.logFile.Name())
	printGreen(fmt.Sprintf(" -> Log file: %s", logPath))

	return manager, nil
}

// StartMainService builds and starts the main service with given configuration
func StartMainService(ctx context.Context, env map[string]string) (*ServiceManager, error) {
	// Extract health check port from environment
	serviceHealthcheckPort, ok := env["SERVICE_HEALTHCHECK_PORT"]
	if !ok {
		return nil, fmt.Errorf("SERVICE_HEALTHCHECK_PORT must be specified in environment map")
	}

	// Extract service port from environment
	servicePort, ok := env["SERVICE_PORT"]
	if !ok {
		return nil, fmt.Errorf("SERVICE_PORT must be specified in environment map")
	}

	// Find project root to build absolute binary path
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	// Create service configuration
	config := ServiceConfig{
		SourcePath:      "./cmd/service",
		BinaryPath:      "bin/service-test",
		ServiceName:     "main-test",
		HealthCheckPort: serviceHealthcheckPort,
		ServicePort:     servicePort,
		Environment:     env,
		LogFilePath:     "logs/service-test.log",
	}

	// Build the binary
	println("Building main service...")
	if err := buildServiceBinary(ctx, config); err != nil {
		return nil, err
	}

	// Update BinaryPath to absolute path for service manager
	config.BinaryPath = filepath.Join(projectRoot, config.BinaryPath)

	// Start the service
	println("Starting main service...")
	manager, err := newServiceManager(ctx, config)
	if err != nil {
		return nil, err
	}

	err = manager.WaitForReady(ctx, 30*time.Second)
	if err != nil {
		_ = manager.StopService(ctx)
		return nil, fmt.Errorf("main service failed to become ready: %w", err)
	}

	// Print service URLs and log file location
	printGreen(fmt.Sprintf(" -> Service URL: %s", manager.GetServiceEndpoint()))
	printGreen(fmt.Sprintf(" -> Metrics URL: %s", manager.GetMetricsEndpoint()))
	printGreen(fmt.Sprintf(" -> HealthCheck URL: %s", manager.GetHealthcheckEndpoint()))
	logPath, _ := filepath.Abs(manager.logFile.Name())
	printGreen(fmt.Sprintf(" -> Log file: %s", logPath))

	return manager, nil
}

// FindFreePort finds an available TCP port by briefly binding to :0.
// The OS assigns a free port, which is then released for use by the caller.
// Note: there is a small TOCTOU window between releasing the port and the
// service binding to it, but in practice this is negligible.
func FindFreePort() (string, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", fmt.Errorf("failed to find free port: %w", err)
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port), nil
}

// StopAllServices gracefully stops all service managers
func StopAllServices(ctx context.Context, managers ...*ServiceManager) {
	for _, manager := range managers {
		if manager != nil {
			_ = manager.StopService(ctx)
		}
	}
}

// CleanupOrphanedServices kills orphaned test service processes from previous runs
// This should be called at the beginning of test suites to ensure a clean state
// Note: This ALWAYS cleans up old services regardless of KEEP setting to ensure fresh test runs
func CleanupOrphanedServices(ctx context.Context) error {
	// Find project root to locate test binaries
	projectRoot, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// List of test service binary names to clean up
	testBinaries := []string{
		"background-test",
		"service-test",
		"main-test",
		"ops-test",
	}

	for _, binaryName := range testBinaries {
		binaryPath := filepath.Join(projectRoot, "bin", binaryName)

		// Use pgrep to find processes running this binary
		cmd := exec.CommandContext(ctx, "pgrep", "-f", binaryPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// pgrep returns exit code 1 if no processes found, which is not an error
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
				continue
			}
			// Other errors are not critical, log and continue
			fmt.Printf("Warning: Could not search for %s processes: %v\n", binaryName, err)
			continue
		}

		pids := strings.Fields(strings.TrimSpace(string(output)))
		if len(pids) == 0 {
			continue
		}

		// Always clean up old processes to ensure fresh test setup
		// KEEP=true only applies to services from the CURRENT test run, not old ones
		fmt.Printf("Found %d orphaned %s process(es) from previous runs, cleaning up...\n", len(pids), binaryName)

		for _, pidStr := range pids {
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				fmt.Printf("Warning: Invalid PID %s: %v\n", pidStr, err)
				continue
			}

			// Send SIGTERM first for graceful shutdown
			if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
				fmt.Printf("Warning: Failed to send SIGTERM to PID %d: %v\n", pid, err)
				continue
			}

			// Wait a bit for graceful shutdown
			time.Sleep(500 * time.Millisecond)

			// Check if process is still running
			if err := syscall.Kill(pid, syscall.Signal(0)); err == nil {
				// Process still running, force kill
				if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
					fmt.Printf("Warning: Failed to kill PID %d: %v\n", pid, err)
				} else {
					fmt.Printf("Force killed orphaned process: %s (PID %d)\n", binaryName, pid)
				}
			} else {
				fmt.Printf("Terminated orphaned process: %s (PID %d)\n", binaryName, pid)
			}
		}
	}

	return nil
}
