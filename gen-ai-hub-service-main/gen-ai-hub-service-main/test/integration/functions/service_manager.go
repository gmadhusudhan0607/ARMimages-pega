//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package functions

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

// ServiceManager manages the gen-ai-hub-service for integration tests
type ServiceManager struct {
	name         string
	uniqueID     string // unique identifier for parallel environment isolation (PID/log files)
	command      []string
	envVars      map[string]string
	cmd          *exec.Cmd
	logs         bytes.Buffer
	logFile      *os.File
	pidFilePath  string
	logToConsole bool
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewServiceManager creates a new ServiceManager with the provided environment variables
func NewServiceManager(envVars map[string]string) (*ServiceManager, error) {
	return NewServiceManagerWithCommand("genai-hub-service", []string{"go", "run", "cmd/service/main.go"}, envVars)
}

// NewOpsServiceManager creates a ServiceManager configured for the genai-gateway-ops service
func NewOpsServiceManager(envVars map[string]string) (*ServiceManager, error) {
	return NewServiceManagerWithCommand("genai-gateway-ops", []string{"go", "run", "cmd/ops/main.go"}, envVars)
}

// NewServiceManagerWithCommand creates a ServiceManager with a custom name and command
func NewServiceManagerWithCommand(name string, command []string, envVars map[string]string) (*ServiceManager, error) {
	if envVars == nil {
		return nil, fmt.Errorf("environment variables cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ServiceManager{
		name:    name,
		command: command,
		envVars: envVars,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// SetUniqueID sets a unique identifier used to isolate PID and log files,
// allowing multiple environments to run in parallel without file conflicts.
// When set, file names become e.g. "genai-hub-service-{uniqueID}.pid" instead of "genai-hub-service.pid".
func (sm *ServiceManager) SetUniqueID(id string) {
	sm.uniqueID = id
}

// fileBaseName returns the base name for PID and log files.
// When a unique ID is set, it follows the same naming convention as live test
// curl files (see test/live/runner/http.go newTestFilePrefix): "live-test-{id}-{name}".
func (sm *ServiceManager) fileBaseName() string {
	if sm.uniqueID != "" {
		return fmt.Sprintf("live-test-%s-%s", sm.uniqueID, sm.name)
	}
	return sm.name
}

// SetLogToConsole enables or disables streaming service logs to the console (os.Stdout).
// When enabled, service stdout/stderr will be printed to the console in real-time
// in addition to being captured in the in-memory buffer and log file.
func (sm *ServiceManager) SetLogToConsole(enabled bool) {
	sm.logToConsole = enabled
}

// VerboseEnabled is true when VERBOSE_RUNNER=true is set.
// Controls whether test runner progress messages are printed.
var VerboseEnabled = os.Getenv("VERBOSE_RUNNER") == "true"

// LogVerbose prints msg to stdout only when VERBOSE_RUNNER=true.
func LogVerbose(msg string) {
	if VerboseEnabled {
		fmt.Println(msg)
	}
}

// LogVerbosef prints a formatted message to stdout only when VERBOSE_RUNNER=true.
func LogVerbosef(format string, args ...any) {
	if VerboseEnabled {
		fmt.Printf(format, args...)
	}
}

// startService starts the gen-ai-hub-service with the configured environment variables
func (sm *ServiceManager) startService() error {
	if sm.cmd != nil {
		return fmt.Errorf("service is already running")
	}

	// Check and clean up ports before starting
	servicePort, exists := sm.envVars["SERVICE_PORT"]
	if exists {
		err := sm.ensurePortAvailable(servicePort)
		if err != nil {
			return fmt.Errorf("failed to make service port %s available: %w", servicePort, err)
		}
	}

	healthcheckPort, exists := sm.envVars["SERVICE_HEALTHCHECK_PORT"]
	if exists {
		err := sm.ensurePortAvailable(healthcheckPort)
		if err != nil {
			return fmt.Errorf("failed to make healthcheck port %s available: %w", healthcheckPort, err)
		}
	}

	// Create command to run the service
	sm.cmd = exec.CommandContext(sm.ctx, sm.command[0], sm.command[1:]...)

	// Set working directory to project root
	// Find project root by looking for go.mod file
	workingDir, err := sm.findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}
	sm.cmd.Dir = workingDir

	// Set environment variables
	sm.cmd.Env = os.Environ()
	for key, value := range sm.envVars {
		sm.cmd.Env = append(sm.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create log file for parallel logging
	err = sm.createLogFile()
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Create MultiWriter to write to buffer and file simultaneously,
	// optionally also streaming to console (os.Stdout) when logToConsole is enabled
	var multiWriter io.Writer
	if sm.logToConsole {
		multiWriter = io.MultiWriter(&sm.logs, sm.logFile, os.Stdout)
	} else {
		multiWriter = io.MultiWriter(&sm.logs, sm.logFile)
	}

	// Capture stdout and stderr for parallel logging
	sm.cmd.Stdout = multiWriter
	sm.cmd.Stderr = multiWriter

	// Set process group ID for proper process termination
	sm.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Start the service
	err = sm.cmd.Start()
	if err != nil {
		// Clean up log file if service start fails
		sm.closeLogFile()
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Save PID to file for process tracking
	err = sm.savePIDToFile(sm.cmd.Process.Pid)
	if err != nil {
		// Log warning but don't fail service startup
		fmt.Printf("Warning: Failed to save PID to file: %v\n", err)
	}

	return nil
}

// stopService terminates the gen-ai-hub-service process
func (sm *ServiceManager) stopService() error {
	if sm.cmd == nil || sm.cmd.Process == nil {
		return nil // Service not running
	}

	// Cancel context to signal shutdown
	if sm.cancel != nil {
		sm.cancel()
	}

	// Kill the entire process group (negative PID)
	// Since we set Setpgid: true when starting, we need to kill the whole group
	pgid := -sm.cmd.Process.Pid

	// Try graceful shutdown first
	err := syscall.Kill(pgid, syscall.SIGTERM)
	if err != nil {
		// If graceful shutdown fails, try to kill just the process
		_ = sm.cmd.Process.Kill()
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- sm.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Timeout - force kill the entire process group
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		<-done // Wait for Wait() to complete
	}

	// Close log file if it's open
	sm.closeLogFile()

	// Remove PID file since service has stopped
	sm.removePIDFile()

	sm.cmd = nil
	// Reset context for next run
	sm.ctx, sm.cancel = context.WithCancel(context.Background())
	return nil
}

// WaitForServiceReady waits for the service to become ready by checking the healthcheck endpoint
func (sm *ServiceManager) WaitForServiceReady(timeout time.Duration) error {
	if sm.envVars == nil {
		return fmt.Errorf("environment variables not configured")
	}

	healthcheckPort, exists := sm.envVars["SERVICE_HEALTHCHECK_PORT"]
	if !exists {
		return fmt.Errorf("SERVICE_HEALTHCHECK_PORT not configured")
	}

	healthcheckURL := fmt.Sprintf("http://localhost:%s/health/liveness", healthcheckPort)
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(healthcheckURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil // Service is ready
			}
		}

		// Check if service process has exited unexpectedly
		if sm.cmd != nil && sm.cmd.ProcessState != nil && sm.cmd.ProcessState.Exited() {
			return fmt.Errorf("service process exited unexpectedly")
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("service did not become ready within %v", timeout)
}

// GetServiceLogs returns the captured service logs
func (sm *ServiceManager) GetServiceLogs() (string, error) {
	return sm.logs.String(), nil
}

// StartService performs common setup operations for test suites.
// It handles pre-cleanup, service startup, and readiness check.
// Returns an error with detailed logs if any step fails.
func (sm *ServiceManager) StartService() error {
	// Print service configuration
	servicePort := sm.envVars["SERVICE_PORT"]
	healthcheckPort := sm.envVars["SERVICE_HEALTHCHECK_PORT"]
	LogVerbosef("Starting service with ports %s (service) and %s (healthcheck)...\n", servicePort, healthcheckPort)

	// Print log file location that will be created
	logPath := fmt.Sprintf("/tmp/%s.log", sm.fileBaseName())
	LogVerbosef("Service logs will be written to: %s\n", logPath)

	// Pre-cleanup: Check for existing PID file and terminate any orphaned processes
	pid, err := sm.readPIDFromFile()
	if err == nil {
		// PID file exists, check if process is still running
		if sm.isProcessRunning(pid) {
			LogVerbosef("Found existing service process with PID %d, terminating it...\n", pid)
			err = sm.terminateProcess(pid)
			if err != nil {
				fmt.Printf("Warning: Failed to terminate existing process %d: %v\n", pid, err)
			} else {
				LogVerbosef("Successfully terminated existing process %d\n", pid)
			}
		} else {
			LogVerbosef("Found stale PID file with PID %d (process not running)\n", pid)
		}
		// Remove stale PID file regardless of whether process was running
		sm.removePIDFile()
	}

	// Pre-cleanup: Stop any existing service instances managed by this ServiceManager
	_ = sm.stopService() // Ignore errors as service might not be running

	// Start the service
	err = sm.startService()
	if err != nil {
		logs, _ := sm.GetServiceLogs()
		return fmt.Errorf("failed to start service: %w\nService logs:\n%s", err, logs)
	}

	err = sm.WaitForServiceReady(60 * time.Second)
	if err != nil {
		logs, _ := sm.GetServiceLogs()
		// print logs from the sm.logFile file as well

		// read from log file
		if sm.logFile != nil {
			logFileContent, readErr := os.ReadFile(sm.logFile.Name())
			if readErr == nil {
				logs += "\n--- Log file content ---\n" + string(logFileContent)
			} else {
				logs += fmt.Sprintf("\n--- Failed to read log file: %v ---\n", readErr)
			}
		}
		fmt.Print(logs) // Print logs to make use of the variable
		return fmt.Errorf("service failed to become ready: %w\n", err)
	}

	LogVerbose("Service is ready to accept requests on port " + servicePort)
	return nil
}

// StopService performs common cleanup operations for test suites.
// It gracefully stops the service with a timeout and handles errors without panicking.
// This method is designed to be called in AfterSuite and will not return an error
// to prevent test suite cleanup from failing.
func (sm *ServiceManager) StopService() {
	if sm == nil {
		return
	}

	// Use a channel with timeout to prevent hanging in cleanup
	done := make(chan error, 1)
	go func() {
		done <- sm.stopService()
	}()

	select {
	case err := <-done:
		if err != nil {
			// Log warning but don't fail the cleanup
			fmt.Printf("Warning: Error stopping service during cleanup: %v\n", err)
		}
	case <-time.After(5 * time.Second):
		// Timeout - log warning but continue
		fmt.Printf("Warning: Service stop operation timed out after 5 seconds\n")
	}
}

// findProjectRoot finds the project root directory by looking for go.mod file
func (sm *ServiceManager) findProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Start from current directory and walk up until we find go.mod
	dir := currentDir
	for {
		goModPath := fmt.Sprintf("%s/go.mod", dir)
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, this is the project root
			return dir, nil
		}

		// Move up one directory
		parentDir := fmt.Sprintf("%s/..", dir)
		absParentDir, err := filepath.Abs(parentDir)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}

		// If we haven't moved up (reached root), stop
		if absParentDir == dir {
			break
		}
		dir = absParentDir
	}

	return "", fmt.Errorf("go.mod not found in any parent directory")
}

// isPortInUse checks if a given port is already in use
func (sm *ServiceManager) isPortInUse(port string) bool {
	conn, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return true // Port is in use
	}
	conn.Close()
	return false // Port is available
}

// createLogFile creates a log file in the temp directory for capturing service logs.
func (sm *ServiceManager) createLogFile() error {
	logPath := fmt.Sprintf("/tmp/%s.log", sm.fileBaseName())

	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file %s: %w", logPath, err)
	}

	sm.logFile = logFile
	return nil
}

// closeLogFile safely closes the log file if it's open
func (sm *ServiceManager) closeLogFile() {
	if sm.logFile != nil {
		_ = sm.logFile.Close() // Ignore errors during cleanup
		sm.logFile = nil
	}
}

// getPIDFilePath returns the path for the PID file
func (sm *ServiceManager) getPIDFilePath() string {
	if sm.pidFilePath == "" {
		sm.pidFilePath = fmt.Sprintf("/tmp/%s.pid", sm.fileBaseName())
	}
	return sm.pidFilePath
}

// savePIDToFile saves the process ID to a file
func (sm *ServiceManager) savePIDToFile(pid int) error {
	pidFile := sm.getPIDFilePath()
	content := strconv.Itoa(pid)

	err := os.WriteFile(pidFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write PID file %s: %w", pidFile, err)
	}

	return nil
}

// readPIDFromFile reads the process ID from the PID file
func (sm *ServiceManager) readPIDFromFile() (int, error) {
	pidFile := sm.getPIDFilePath()

	data, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("PID file does not exist")
		}
		return 0, fmt.Errorf("failed to read PID file %s: %w", pidFile, err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file %s: %w", pidFile, err)
	}

	return pid, nil
}

// removePIDFile removes the PID file
func (sm *ServiceManager) removePIDFile() {
	pidFile := sm.getPIDFilePath()
	_ = os.Remove(pidFile) // Ignore errors during cleanup
}

// isProcessRunning checks if a process with the given PID is running
func (sm *ServiceManager) isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Try to send signal 0 to check if process exists
	err := syscall.Kill(pid, 0)
	return err == nil
}

// terminateProcess terminates a process by PID with graceful/force approach
func (sm *ServiceManager) terminateProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	if !sm.isProcessRunning(pid) {
		return nil // Process not running
	}

	// Try graceful shutdown first (SIGTERM)
	err := syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to send SIGTERM to process %d: %w", pid, err)
	}

	// Wait for graceful shutdown with timeout
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Timeout - try force kill
			if sm.isProcessRunning(pid) {
				err = syscall.Kill(pid, syscall.SIGKILL)
				if err != nil {
					return fmt.Errorf("failed to force kill process %d: %w", pid, err)
				}
				// Wait a bit more for force kill to take effect
				time.Sleep(1 * time.Second)
			}
			return nil
		case <-ticker.C:
			if !sm.isProcessRunning(pid) {
				// Process has exited
				return nil
			}
		}
	}
}

// ensurePortAvailable ensures that the specified port is available by terminating any processes using it
func (sm *ServiceManager) ensurePortAvailable(port string) error {
	if !sm.isPortInUse(port) {
		return nil // Port is already available
	}

	// Find processes using this port and terminate them
	pids, err := sm.findProcessesUsingPort(port)
	if err != nil {
		return fmt.Errorf("failed to find processes using port %s: %w", port, err)
	}

	if len(pids) == 0 {
		// Port is in use but no processes found, might be a timing issue
		// Wait a bit and check again
		time.Sleep(1 * time.Second)
		if !sm.isPortInUse(port) {
			return nil
		}
		return fmt.Errorf("port %s is in use but no processes found", port)
	}

	// Terminate all processes using the port
	for _, pid := range pids {
		LogVerbosef("Found process %d using port %s, terminating it...\n", pid, port)
		err = sm.terminateProcess(pid)
		if err != nil {
			fmt.Printf("Warning: Failed to terminate process %d: %v\n", pid, err)
		} else {
			LogVerbosef("Successfully terminated process %d\n", pid)
		}
	}

	// Wait for port to become available
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("port %s did not become available after terminating processes", port)
		case <-ticker.C:
			if !sm.isPortInUse(port) {
				return nil // Port is now available
			}
		}
	}
}

// findProcessesUsingPort finds all process IDs that are using the specified port
func (sm *ServiceManager) findProcessesUsingPort(port string) ([]int, error) {
	// Use lsof to find processes using the port
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%s", port))
	output, err := cmd.Output()
	if err != nil {
		// lsof returns non-zero exit code when no processes found
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return []int{}, nil // No processes found
		}
		return nil, fmt.Errorf("failed to run lsof: %w", err)
	}

	// Parse PIDs from output
	var pids []int
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue // Skip invalid PIDs
		}
		pids = append(pids, pid)
	}

	return pids, nil
}
