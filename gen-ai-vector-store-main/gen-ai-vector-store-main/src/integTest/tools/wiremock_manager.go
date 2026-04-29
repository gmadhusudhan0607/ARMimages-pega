// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// WireMockManager manages the lifecycle of a WireMock container
type WireMockManager struct {
	container testcontainers.Container
	baseURL   string
	adminURL  string
	host      string
	port      string
	config    WireMockConfig
	ctx       context.Context
}

// WireMockConfig holds configuration for starting a WireMock container
type WireMockConfig struct {
	Image          string        // Container image (default: "docker-dev.bin.pega.io/wiremock/wiremock:latest")
	Port           string        // WireMock port (default: "8080")
	StartupTimeout time.Duration // Container startup timeout (default: 30s)
	KeepRunning    bool          // Keep container running after tests for debugging
	ContainerLabel string        // Label to identify test containers (default: "genai-vector-store-wiremock-test")
	Verbose        bool          // Enable verbose logging
	DisableBanner  bool          // Disable WireMock banner (default: true)
}

// NewWireMockManager creates a new WireMock manager with the given configuration.
// The container is not started until Start() is called.
func NewWireMockManager(ctx context.Context, config WireMockConfig) (*WireMockManager, error) {
	// Apply defaults
	if config.Image == "" {
		config.Image = "docker-dev.bin.pega.io/wiremock/wiremock:latest"
	}
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.StartupTimeout == 0 {
		config.StartupTimeout = 30 * time.Second
	}
	if config.ContainerLabel == "" {
		config.ContainerLabel = "genai-vector-store-wiremock-test"
	}
	// Default to disabling banner for cleaner output
	if !config.Verbose {
		config.DisableBanner = true
	}

	// Create manager with configuration (container not started yet)
	mgr := &WireMockManager{
		config: config,
		ctx:    ctx,
	}

	return mgr, nil
}

// Start starts the WireMock container and waits for it to be ready
func (wm *WireMockManager) Start() error {
	if wm.container != nil {
		return fmt.Errorf("container already started")
	}

	// Check if KEEP mode is enabled
	keepRunning := wm.config.KeepRunning || os.Getenv("KEEP") == "true"

	// Build WireMock command arguments
	args := []string{}
	if wm.config.Verbose {
		args = append(args, "--verbose")
	}
	if wm.config.DisableBanner {
		args = append(args, "--disable-banner")
	}

	// Create container request with labels for identification and cleanup
	req := testcontainers.ContainerRequest{
		Image:        wm.config.Image,
		ExposedPorts: []string{fmt.Sprintf("%s/tcp", wm.config.Port)},
		Labels: map[string]string{
			"test-container": wm.config.ContainerLabel,
			"created-at":     fmt.Sprintf("%d", time.Now().Unix()),
		},
		WaitingFor: wait.ForHTTP("/__admin/health").
			WithPort(nat.Port(wm.config.Port + "/tcp")).
			WithStartupTimeout(wm.config.StartupTimeout),
	}

	// Add command arguments if any
	if len(args) > 0 {
		req.Cmd = args
	}

	// Use background context if KEEP mode is enabled to prevent container termination
	// when the test context is cancelled
	containerCtx := wm.ctx
	if keepRunning {
		containerCtx = context.Background()
		fmt.Println("KEEP=true: Using background context to persist container beyond test lifecycle")
	}

	// Start container
	container, err := testcontainers.GenericContainer(containerCtx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to start WireMock container: %w", err)
	}

	wm.container = container

	// Extract connection details
	if err := wm.updateConnectionDetails(); err != nil {
		_ = wm.container.Terminate(wm.ctx)
		wm.container = nil
		return fmt.Errorf("failed to get connection details: %w", err)
	}

	// Print connection URLs for easy troubleshooting
	fmt.Printf("\033[32m -> WireMock Base URL: %s\033[0m\n", wm.baseURL)
	fmt.Printf("\033[32m -> WireMock Admin URL: %s\033[0m\n", wm.adminURL)

	return nil
}

// updateConnectionDetails extracts and caches connection details from the container
func (wm *WireMockManager) updateConnectionDetails() error {
	host, err := wm.container.Host(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := wm.container.MappedPort(context.Background(), nat.Port(wm.config.Port))
	if err != nil {
		return fmt.Errorf("failed to get mapped port: %w", err)
	}

	wm.host = host
	wm.port = mappedPort.Port()
	wm.baseURL = fmt.Sprintf("http://%s:%s", wm.host, wm.port)
	wm.adminURL = fmt.Sprintf("http://%s:%s/__admin", wm.host, wm.port)

	return nil
}

// GetBaseURL returns the base URL for making requests to WireMock
func (wm *WireMockManager) GetBaseURL() string {
	return wm.baseURL
}

// GetAdminURL returns the admin API URL for managing WireMock
func (wm *WireMockManager) GetAdminURL() string {
	return wm.adminURL
}

// GetConnectionDetails returns the host and port for the container
func (wm *WireMockManager) GetConnectionDetails() (host, port string) {
	return wm.host, wm.port
}

// GetContainerID returns the container ID for debugging
func (wm *WireMockManager) GetContainerID() string {
	return wm.container.GetContainerID()
}

// IsRunning checks if the container is still running
func (wm *WireMockManager) IsRunning() bool {
	if wm.container == nil {
		return false
	}
	state, err := wm.container.State(wm.ctx)
	if err != nil {
		return false
	}
	return state.Running
}

// CreateMapping creates a WireMock mapping programmatically
// The mapping parameter should be a valid WireMock stub mapping JSON structure
// Returns the unique ID of the created mapping which can be used for selective deletion
func (wm *WireMockManager) CreateMapping(mapping interface{}) (string, error) {
	if wm.container == nil {
		return "", fmt.Errorf("container not started")
	}

	// Convert to JSON
	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return "", fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// Post to WireMock admin API
	resp, err := http.Post(
		fmt.Sprintf("%s/mappings", wm.adminURL),
		"application/json",
		bytes.NewBuffer(mappingJSON),
	)
	if err != nil {
		return "", fmt.Errorf("failed to post mapping to WireMock: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create mapping, status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to extract the mapping ID
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode mapping response: %w", err)
	}

	return result.ID, nil
}

// CreateStub creates a simple WireMock stub mapping with request and response
// This is a convenience method for common use cases
// Returns the unique ID of the created mapping which can be used for selective deletion
func (wm *WireMockManager) CreateStub(request, response interface{}) (string, error) {
	mapping := map[string]interface{}{
		"request":  request,
		"response": response,
	}
	return wm.CreateMapping(mapping)
}

// DeleteMapping removes a specific mapping by ID
func (wm *WireMockManager) DeleteMapping(id string) error {
	if wm.container == nil {
		return fmt.Errorf("container not started")
	}

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/mappings/%s", wm.adminURL, id),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete mapping, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ResetMappings removes all mappings from WireMock
func (wm *WireMockManager) ResetMappings() error {
	if wm.container == nil {
		return fmt.Errorf("container not started")
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/mappings", wm.adminURL), nil)
	if err != nil {
		return fmt.Errorf("failed to create reset request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reset mappings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to reset mappings, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetMapping retrieves a specific mapping by its ID from WireMock
func (wm *WireMockManager) GetMapping(id string) (map[string]interface{}, error) {
	if wm.container == nil {
		return nil, fmt.Errorf("container not started")
	}

	resp, err := http.Get(fmt.Sprintf("%s/mappings/%s", wm.adminURL, id))
	if err != nil {
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get mapping, status %d: %s", resp.StatusCode, string(body))
	}

	var mapping map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&mapping); err != nil {
		return nil, fmt.Errorf("failed to decode mapping response: %w", err)
	}

	return mapping, nil
}

// GetAllMappings retrieves all current mappings from WireMock
func (wm *WireMockManager) GetAllMappings() ([]map[string]interface{}, error) {
	if wm.container == nil {
		return nil, fmt.Errorf("container not started")
	}

	resp, err := http.Get(fmt.Sprintf("%s/mappings", wm.adminURL))
	if err != nil {
		return nil, fmt.Errorf("failed to get mappings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get mappings, status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Mappings []map[string]interface{} `json:"mappings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode mappings response: %w", err)
	}

	return result.Mappings, nil
}

// VerifyRequest verifies that a request matching the given pattern was made
// Returns the count of matching requests
func (wm *WireMockManager) VerifyRequest(requestPattern interface{}) (int, error) {
	if wm.container == nil {
		return 0, fmt.Errorf("container not started")
	}

	// Convert to JSON
	patternJSON, err := json.Marshal(requestPattern)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request pattern: %w", err)
	}

	// Post to WireMock verification endpoint
	resp, err := http.Post(
		fmt.Sprintf("%s/requests/count", wm.adminURL),
		"application/json",
		bytes.NewBuffer(patternJSON),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to verify request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to verify request, status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Count int `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode verification response: %w", err)
	}

	return result.Count, nil
}

// GetRequests retrieves all request journal entries from WireMock
// Each request entry includes the stubMapping field which indicates which stub handled the request
func (wm *WireMockManager) GetRequests() ([]map[string]interface{}, error) {
	if wm.container == nil {
		return nil, fmt.Errorf("container not started")
	}

	resp, err := http.Get(fmt.Sprintf("%s/requests", wm.adminURL))
	if err != nil {
		return nil, fmt.Errorf("failed to get requests: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get requests, status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Requests []map[string]interface{} `json:"requests"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode requests response: %w", err)
	}

	return result.Requests, nil
}

// Stop stops the container unless KeepRunning is set or KEEP environment variable is set
func (wm *WireMockManager) Stop() error {
	if wm.container == nil {
		return nil
	}

	// Parse KEEP mode configuration
	keepConfig, err := ParseKeepMode()
	if err != nil {
		return fmt.Errorf("failed to parse KEEP mode: %w", err)
	}

	// Check if we should keep the container running
	keepRunning := wm.config.KeepRunning || keepConfig.Enabled

	if keepRunning {
		containerID := wm.GetContainerID()

		if keepConfig.Enabled {
			// KEEP mode with duration - schedule cleanup
			fmt.Println("  ================================================================================")
			fmt.Printf("    KEEP mode: Container will remain running for %s\n", keepConfig.Duration)
			fmt.Printf("    Container ID: %s\n", containerID)
			fmt.Printf("    Base URL: %s\n", wm.baseURL)
			fmt.Printf("    Admin URL: %s\n", wm.adminURL)
			fmt.Printf("    To stop manually: docker stop %s\n", containerID)
			fmt.Printf("    Auto-cleanup scheduled at: %s\n", time.Now().Add(keepConfig.Duration).Format(time.RFC3339))

			// Schedule cleanup using background context so it survives test completion
			keepConfig.ScheduleCleanup(context.Background(), fmt.Sprintf("WireMock container %s", containerID), func() error {
				return wm.terminateContainer()
			})
		} else {
			// KeepRunning from config - no auto-cleanup
			fmt.Println("  ================================================================================")
			fmt.Println("    Container will remain running (KeepRunning=true in config)")
			fmt.Printf("    Container ID: %s\n", containerID)
			fmt.Printf("    Base URL: %s\n", wm.baseURL)
			fmt.Printf("    Admin URL: %s\n", wm.adminURL)
			fmt.Printf("    To stop manually: docker stop %s\n", containerID)
		}

		return nil
	}

	// Normal cleanup - terminate immediately
	return wm.terminateContainer()
}

// terminateContainer terminates the WireMock container
func (wm *WireMockManager) terminateContainer() error {
	if wm.container == nil {
		return nil
	}

	if err := wm.container.Terminate(wm.ctx); err != nil {
		return fmt.Errorf("failed to terminate container: %w", err)
	}

	return nil
}

// CleanupOrphanedWireMockContainers removes old test containers from previous runs
// This should be called at the beginning of test suites to ensure a clean state
// Note: This ALWAYS cleans up old containers regardless of KEEP setting to ensure fresh test runs
func CleanupOrphanedWireMockContainers(ctx context.Context, labelFilter string) error {
	if labelFilter == "" {
		labelFilter = "genai-vector-store-wiremock-test"
	}

	// Use docker CLI to find and remove containers with the test label
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

	fmt.Printf("Found %d orphaned WireMock container(s) from previous runs, cleaning up...\n", len(containerIDs))

	for _, containerID := range containerIDs {
		// Always force remove containers to ensure clean test setup
		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerID)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to remove container %s: %v\n", containerID, err)
		} else {
			fmt.Printf("Removed orphaned WireMock container: %s\n", containerID)
		}
	}

	return nil
}
