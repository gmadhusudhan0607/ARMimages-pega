/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"bufio"
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/test/integration/functions"
	"github.com/google/uuid"
)

const (
	opsReadinessTimeout   = 15 * time.Second
	readinessPollInterval = 2 * time.Second
)

// logOutputEnabled controls whether service stdout/stderr is streamed to the console.
// Set LOG_OUTPUT=true to enable. Logs are always written to log files regardless.
var logOutputEnabled = os.Getenv("LOG_OUTPUT") == "true"

// Aliases for the shared verbose logging helpers from the functions package.
var (
	logVerbose  = functions.LogVerbose
	logVerbosef = functions.LogVerbosef
)

// TestEnvironment holds all shared state for live test scenarios.
type TestEnvironment struct {
	UniqueID               string
	SvcBaseURL             string
	SvcHealthcheckURL      string
	JWTToken               string
	SaxCell                string
	ChatCompletionTargets  []ModelTarget
	EmbeddingTargets       []ModelTarget
	ImageGenerationTargets []ModelTarget
	AllModels              []modelInfoDTO // cached /models response for filtering without extra HTTP calls

	svcManager *functions.ServiceManager
	opsManager *functions.ServiceManager
}

// serviceState holds the intermediate state during environment setup.
type serviceState struct {
	uniqueID          string
	svcBaseURL        string
	svcHealthcheckURL string
	saxCell           string
	svcManager        *functions.ServiceManager
	opsManager        *functions.ServiceManager
	isExternal        bool
}

// Cleanup stops any running services.
func (s *serviceState) Cleanup() {
	if s.svcManager != nil {
		s.svcManager.StopService()
	}
	if s.opsManager != nil {
		s.opsManager.StopService()
	}
}

// SetupEnvironment starts ops + hub-service, resolves JWT, and returns a ready environment.
// svcEnvFile configures genai-hub-service (e.g., "env.genai-hub-service").
// opsEnvFile configures genai-gateway-ops (e.g., "env.genai-gateway-ops").
//
// External service support:
// Set both OPS_URL and SERVICE_URL to use already-deployed services.
// Both must be provided together, or neither.
func SetupEnvironment(svcEnvFile, opsEnvFile string) (*TestEnvironment, error) {
	// Step 1: Setup services (local or external)
	services, err := setupServices(svcEnvFile, opsEnvFile)
	if err != nil {
		return nil, err
	}

	// Step 2: Setup authentication and fetch all models (single GET /models call).
	// FetchAllModels both verifies the token and returns the model list.
	// On HTTP 401, we invalidate caches, re-issue the token, and retry once.
	jwtToken, allModels, err := setupAuthAndFetchModels(services.svcBaseURL, services.saxCell)
	if err != nil {
		services.Cleanup()
		return nil, err
	}

	chatTargets := discoverTargetsFromCache(allModels, ModelTypeChatCompletion, "chat completion")
	embeddingTargets := discoverTargetsFromCache(allModels, ModelTypeEmbedding, "embedding")
	imageGenerationTargets := discoverTargetsFromCache(allModels, ModelTypeImageGeneration, "image generation")

	// Print setup summary
	if services.isExternal {
		logVerbose("=== Setup complete: using external ops and hub-service ===")
	} else {
		logVerbose("=== Setup complete: genai-gateway-ops and genai-hub-service are running ===")
	}

	return &TestEnvironment{
		UniqueID:               services.uniqueID,
		SvcBaseURL:             services.svcBaseURL,
		SvcHealthcheckURL:      services.svcHealthcheckURL,
		JWTToken:               jwtToken,
		SaxCell:                services.saxCell,
		ChatCompletionTargets:  chatTargets,
		EmbeddingTargets:       embeddingTargets,
		ImageGenerationTargets: imageGenerationTargets,
		AllModels:              allModels,
		svcManager:             services.svcManager,
		opsManager:             services.opsManager,
	}, nil
}

// setupServices handles the service setup phase - either using external URLs or starting local services.
func setupServices(svcEnvFile, opsEnvFile string) (*serviceState, error) {
	externalOpsURL := os.Getenv("OPS_URL")
	externalServiceURL := os.Getenv("SERVICE_URL")
	useExternalServices := externalOpsURL != "" && externalServiceURL != ""

	uniqueID := generateUniqueID()
	logVerbosef("=== Environment unique ID: %s ===\n", uniqueID)

	if useExternalServices {
		return setupExternalServices(externalOpsURL, externalServiceURL, uniqueID)
	}
	return setupLocalServices(svcEnvFile, opsEnvFile, uniqueID)
}

// setupExternalServices configures the environment to use already-deployed services.
func setupExternalServices(opsURL, serviceURL, uniqueID string) (*serviceState, error) {
	logVerbosef("=== Using external ops service: %s ===\n", opsURL)
	logVerbosef("=== Using external hub-service: %s ===\n", serviceURL)

	return &serviceState{
		uniqueID:          uniqueID,
		svcBaseURL:        serviceURL,
		svcHealthcheckURL: "", // External service - no healthcheck URL needed
		saxCell:           "us",
		svcManager:        nil,
		opsManager:        nil,
		isExternal:        true,
	}, nil
}

// setupLocalServices starts local ops and hub-service instances.
func setupLocalServices(svcEnvFile, opsEnvFile, uniqueID string) (*serviceState, error) {
	// Start genai-gateway-ops
	opsManager, opsBaseURL, saxCell, err := startOpsService(opsEnvFile, uniqueID)
	if err != nil {
		return nil, err
	}

	// Start genai-hub-service
	svcManager, svcBaseURL, svcHealthcheckURL, err := startHubServiceWithOps(svcEnvFile, opsBaseURL, uniqueID)
	if err != nil {
		opsManager.StopService()
		return nil, err
	}

	return &serviceState{
		uniqueID:          uniqueID,
		svcBaseURL:        svcBaseURL,
		svcHealthcheckURL: svcHealthcheckURL,
		saxCell:           saxCell,
		svcManager:        svcManager,
		opsManager:        opsManager,
		isExternal:        false,
	}, nil
}

// startHubServiceWithOps loads config, injects ops endpoints, and starts the hub-service.
func startHubServiceWithOps(svcEnvFile, opsBaseURL, uniqueID string) (*functions.ServiceManager, string, string, error) {
	envVars, err := loadEnvFile(svcEnvFile)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to load env file '%s': %w", svcEnvFile, err)
	}

	// Dynamically extract MODEL_METADATA_PATH and CONFIGURATION_FILE from Helm templates
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to find project root: %w", err)
	}
	modelMetadataPath, err := extractModelMetadata(projectRoot, uniqueID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to extract model metadata: %w", err)
	}
	mappingPath, err := generateMappingYAML(projectRoot, uniqueID, envVars)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate mapping.yaml: %w", err)
	}

	// Configure endpoints pointing to the ops service
	envVars["CONFIGURATION_FILE"] = mappingPath
	envVars["MODEL_METADATA_PATH"] = modelMetadataPath
	envVars["LOG_LEVEL"] = resolveLogLevel()
	if _, exists := envVars["LOG_TRUNCATE_LONG_STRINGS"]; !exists {
		envVars["LOG_TRUNCATE_LONG_STRINGS"] = "1000"
	}
	envVars["MAPPING_ENDPOINT"] = fmt.Sprintf("%s/v1/mappings", opsBaseURL)
	envVars["MONITORING_ENDPOINT"] = fmt.Sprintf("%s/v1/events", opsBaseURL)
	envVars["MODELS_DEFAULTS_ENDPOINT"] = fmt.Sprintf("%s/v1/models/defaults", opsBaseURL)

	return startHubService(envVars, uniqueID)
}

// setupAuthAndFetchModels resolves the JWT token, fetches all models from /models (single request),
// and returns both. This combines authentication verification with model discovery to avoid
// separate GET /models calls. On HTTP 401, it invalidates the cached JWT and ARN, re-resolves,
// and retries once.
func setupAuthAndFetchModels(svcBaseURL, saxCell string) (string, []modelInfoDTO, error) {
	jwtToken, err := ResolveJWTToken(saxCell)
	if err != nil {
		return "", nil, err
	}

	logVerbose("Fetching all models from /models (single request, also verifies auth)...")
	allModels, err := FetchAllModels(svcBaseURL, jwtToken)
	if err != nil {
		// Check if it's a 401 — if so, refresh token and retry once.
		if strings.Contains(err.Error(), "HTTP 401") {
			logVerbose("  GET /models returned 401, invalidating caches and re-issuing token...")
			jwtCacheForCell(saxCell).invalidate()
			arnCacheForCell(saxCell).invalidate()
			jwtToken, err = ResolveJWTToken(saxCell)
			if err != nil {
				return "", nil, err
			}
			allModels, err = FetchAllModels(svcBaseURL, jwtToken)
			if err != nil {
				return "", nil, fmt.Errorf("GET /models failed after token refresh: %w", err)
			}
		} else {
			return "", nil, fmt.Errorf("GET /models failed: %w", err)
		}
	}
	logVerbosef("  ✓ GET /models returned %d model(s)\n", len(allModels))

	return jwtToken, allModels, nil
}

// discoverTargetsFromCache filters pre-fetched models by type and applies the MODEL env filter.
// This avoids making separate GET /models requests for each model type.
func discoverTargetsFromCache(allModels []modelInfoDTO, modelType, label string) []ModelTarget {
	targets := FilterByType(allModels, modelType)

	// Filter targets by MODEL env var if set.
	if modelFilter := os.Getenv("MODEL"); modelFilter != "" {
		targets = filterTargets(targets, modelFilter)
	}

	logVerbosef("  ✓ Using %d %s target(s):\n", len(targets), label)
	for _, t := range targets {
		printTargetWithLifecycle(t)
	}

	return targets
}

// filterTargets returns only the targets whose model name or provider/model string matches
// modelFilter. Returns nil (empty) if no targets match — callers should treat this as "skip".
func filterTargets(targets []ModelTarget, modelFilter string) []ModelTarget {
	var filtered []ModelTarget
	for _, t := range targets {
		if t.Model == modelFilter || t.String() == modelFilter {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) > 0 {
		logVerbosef("  ✓ Filtered by MODEL=%s: %d target(s)\n", modelFilter, len(filtered))
	}
	return filtered
}

// startOpsService loads config from opsEnvFile, allocates dynamic ports, starts the
// genai-gateway-ops service, and waits for it to become ready.
// Returns the ServiceManager, the ops base URL, the SAX cell, and any error.
func startOpsService(opsEnvFile string, uniqueID string) (*functions.ServiceManager, string, string, error) {
	opsPort, err := functions.GetFreePort()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get free port for ops service: %w", err)
	}
	opsHealthcheckPort, err := functions.GetFreePort()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get free port for ops healthcheck: %w", err)
	}

	opsEnvVars, err := loadEnvFile(opsEnvFile)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to load ops env file %s: %w", opsEnvFile, err)
	}

	// Extract SAX_CELL and dynamically resolve SAX_CONFIG_PATH from AWS Secrets Manager.
	saxCell := opsEnvVars["SAX_CELL"]
	if saxCell == "" {
		saxCell = "us"
	}
	saxConfigPath, err := ResolveSAXConfigPath(saxCell)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to resolve SAX config for cell %s: %w", saxCell, err)
	}
	opsEnvVars["SAX_CONFIG_PATH"] = saxConfigPath

	opsEnvVars["OPS_PORT"] = strconv.Itoa(opsPort)
	opsEnvVars["SERVICE_HEALTHCHECK_PORT"] = strconv.Itoa(opsHealthcheckPort)
	opsEnvVars["LOG_LEVEL"] = resolveLogLevel()

	printEnvVars("genai-gateway-ops", opsEnvVars)

	opsManager, err := functions.NewOpsServiceManager(opsEnvVars)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create OpsServiceManager: %w", err)
	}
	opsManager.SetUniqueID(uniqueID)
	opsManager.SetLogToConsole(logOutputEnabled)

	if err = opsManager.StartService(); err != nil {
		return nil, "", "", fmt.Errorf("failed to start genai-gateway-ops: %w", err)
	}

	// Wait for ops service to be ready via /health/readiness endpoint
	opsBaseURL := fmt.Sprintf("http://localhost:%d", opsPort)
	opsReadinessURL := fmt.Sprintf("http://localhost:%d/health/readiness", opsHealthcheckPort)
	logVerbosef("Polling %s for ops readiness (max %s, every %s)...\n", opsReadinessURL, opsReadinessTimeout, readinessPollInterval)
	if err := waitForOpsReadiness(opsReadinessURL, opsReadinessTimeout); err != nil {
		opsManager.StopService()
		return nil, "", "", fmt.Errorf("genai-gateway-ops readiness check failed: %w", err)
	}
	logVerbose("Ops service is ready.")

	return opsManager, opsBaseURL, saxCell, nil
}

// startHubService allocates dynamic ports, starts the genai-hub-service with the
// given environment variables, and waits for it to accept connections.
// Returns the ServiceManager, the service base URL, the healthcheck URL, and any error.
func startHubService(envVars map[string]string, uniqueID string) (*functions.ServiceManager, string, string, error) {
	svcPort, err := functions.GetFreePort()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get free port for service: %w", err)
	}
	svcHealthcheckPort, err := functions.GetFreePort()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get free port for service healthcheck: %w", err)
	}

	svcBaseURL := fmt.Sprintf("http://localhost:%d", svcPort)
	svcHealthcheckURL := fmt.Sprintf("http://localhost:%d", svcHealthcheckPort)

	envVars["SERVICE_PORT"] = strconv.Itoa(svcPort)
	envVars["SERVICE_HEALTHCHECK_PORT"] = strconv.Itoa(svcHealthcheckPort)

	printEnvVars("genai-hub-service", envVars)

	svcManager, err := functions.NewServiceManager(envVars)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create ServiceManager: %w", err)
	}
	svcManager.SetUniqueID(uniqueID)
	svcManager.SetLogToConsole(logOutputEnabled)

	if err = svcManager.StartService(); err != nil {
		return nil, "", "", fmt.Errorf("failed to start genai-hub-service: %w", err)
	}

	return svcManager, svcBaseURL, svcHealthcheckURL, nil
}

// Teardown stops all services.
func (env *TestEnvironment) Teardown() {
	if env.svcManager != nil {
		env.svcManager.StopService()
		logVerbose("genai-hub-service stopped")
	}
	if env.opsManager != nil {
		env.opsManager.StopService()
		logVerbose("genai-gateway-ops stopped")
	}
}

// helmConfigMapDataKey is the key inside the ConfigMap data section that holds model metadata.
const helmConfigMapDataKey = "model-metadata.yaml"

// extractModelMetadata extracts model metadata from the Helm template file
// and writes it to a temp file. Returns the path to the temp file.
//
// The Helm template is a ConfigMap with Helm directives ({{ ... }}).
// We strip those lines so the remainder is valid YAML, then parse
// .data["model-metadata.yaml"] using Go's yaml.v3 — no external tools needed.
func extractModelMetadata(projectRoot string, uniqueID string) (string, error) {
	helmFile := projectRoot + "/distribution/genai-hub-service-helm/src/main/helm/templates/model-metadata.yaml"

	raw, err := os.ReadFile(helmFile)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", helmFile, err)
	}

	// Strip lines containing Helm directives so the remainder is valid YAML.
	var cleaned bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "{{") {
			cleaned.WriteString(line)
			cleaned.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan %s: %w", helmFile, err)
	}

	// Parse the ConfigMap and extract the data key.
	var configMap struct {
		Data map[string]string `yaml:"data"`
	}
	if err := yaml.Unmarshal(cleaned.Bytes(), &configMap); err != nil {
		return "", fmt.Errorf("failed to parse %s (after stripping Helm directives): %w", helmFile, err)
	}
	metadata, ok := configMap.Data[helmConfigMapDataKey]
	if !ok {
		return "", fmt.Errorf("key %q not found in data section of %s", helmConfigMapDataKey, helmFile)
	}

	tmpFile := fmt.Sprintf("/tmp/live-test-model-metadata-%s.yaml", uniqueID)
	if err := os.WriteFile(tmpFile, []byte(metadata), 0644); err != nil {
		return "", fmt.Errorf("failed to write model metadata to %s: %w", tmpFile, err)
	}
	logVerbose("  MODEL_METADATA_PATH  = " + tmpFile + " (extracted from " + helmFile + ")")
	return tmpFile, nil
}

// generateMappingYAML builds mapping.yaml from Helm configuration model/buddy files.
// It concatenates the individual YAML files and resolves Helm template variables
// using values from the env file (GENAI_URL, DEMO_GCP_VERTEX_URL, etc.).
// Returns the path to the generated temp file.
func generateMappingYAML(projectRoot, uniqueID string, envVars map[string]string) (string, error) {
	helmDir := projectRoot + "/distribution/genai-hub-service-helm/src/main/helm/configuration"

	// Template variable replacements — map Helm values to env vars.
	replacements := map[string]string{
		"{{ .Release.Namespace }}":         "genai-hub-service",
		"{{ .Values.GenAIURL }}":           envVars["GENAI_URL"],
		"{{ .Values.DemoGcpVertexURL }}":   envVars["DEMO_GCP_VERTEX_URL"],
		"{{ .Values.DemoAwsBedrockURL }}":  envVars["DEMO_AWS_BEDROCK_URL"],
		"{{ .Values.SelfStudyBuddyURLv1}}": envVars["SELF_STUDY_BUDDY_URL"],
	}

	// Build the mapping.yaml with models and buddies sections.
	var mapping bytes.Buffer
	for _, section := range []struct{ name, subdir string }{
		{"models", "models"},
		{"buddies", "buddies"},
	} {
		content, err := readAndReplaceHelmYAML(filepath.Join(helmDir, section.subdir), replacements)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", section.subdir, err)
		}
		mapping.WriteString(section.name + ":\n")
		for _, line := range strings.Split(content, "\n") {
			if line != "" {
				mapping.WriteString("  ")
				mapping.WriteString(line)
			}
			mapping.WriteByte('\n')
		}
	}

	tmpFile := fmt.Sprintf("/tmp/live-test-mapping-%s.yaml", uniqueID)
	if err := os.WriteFile(tmpFile, mapping.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write mapping to %s: %w", tmpFile, err)
	}
	logVerbose("  CONFIGURATION_FILE   = " + tmpFile + " (generated from Helm configuration)")
	return tmpFile, nil
}

// readAndReplaceHelmYAML reads all YAML files from dir, applies template replacements,
// and returns the concatenated result.
func readAndReplaceHelmYAML(dir string, replacements map[string]string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return "", fmt.Errorf("failed to glob %s: %w", dir, err)
	}
	sort.Strings(files)
	var buf bytes.Buffer
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", f, err)
		}
		content := string(data)
		for old, new := range replacements {
			if new != "" {
				content = strings.ReplaceAll(content, old, new)
			}
		}
		buf.WriteString(content)
		buf.WriteByte('\n')
	}
	return buf.String(), nil
}

// findProjectRoot walks up from cwd looking for go.mod to find the project root.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod) starting from %s", dir)
		}
		dir = parent
	}
}

// waitForOpsReadiness polls the ops /readiness healthcheck endpoint until it returns
// an HTTP 200 response, or the timeout expires. The /readiness endpoint returns 503
// when mappings are not yet loaded, and 200 when the service is fully ready.
func waitForOpsReadiness(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 1 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(readinessPollInterval)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			logVerbosef("  ✓ %s responded with HTTP %d\n", url, resp.StatusCode)
			return nil
		}
		logVerbosef("  ⏳ %s responded with HTTP %d, retrying...\n", url, resp.StatusCode)
		time.Sleep(readinessPollInterval)
	}
	hint := ""
	if os.Getenv("LOG_LEVEL") != "debug" {
		hint = "\nHint: re-run with VERBOSE=2 to see service logs for debugging"
	}
	return fmt.Errorf("endpoint %s did not become ready within %v%s", url, timeout, hint)
}

// printEnvVars prints all environment variables for a service in sorted order.
func printEnvVars(serviceName string, envVars map[string]string) {
	logVerbose("====================================================================================")
	logVerbosef("--- %s environment (%d vars) ---\n", serviceName, len(envVars))

	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		logVerbosef("  %-35s = %s\n", k, envVars[k])
	}
	logVerbose("")
}

// printTargetWithLifecycle prints a model target with its lifecycle status annotation.
func printTargetWithLifecycle(t ModelTarget) {
	if t.Lifecycle != "" && t.Lifecycle != "Generally Available" {
		logVerbosef("    - %s [%s]\n", t, t.Lifecycle)
	} else {
		logVerbosef("    - %s\n", t)
	}
}

// resolveLogLevel returns the LOG_LEVEL from the environment variable,
// defaulting to "info" if not set.
func resolveLogLevel() string {
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		return level
	}
	return "info"
}

// generateUniqueID generates a UUID (v7) to uniquely identify a test environment,
// allowing multiple environments to run in parallel without file conflicts.
func generateUniqueID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// loadEnvFile parses a key=value env file (lines starting with # are ignored)
// and returns a map of environment variables.
// After parsing, any values that look like local file paths are validated for existence.
func loadEnvFile(path string) (map[string]string, error) {
	if path == "" {
		return nil, fmt.Errorf("env file path is empty - when using external services, you must provide BOTH OPS_URL and SERVICE_URL, or specify CONFIG to use a local service config")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file '%s': %w", path, err)
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if err := validateEnvFilePaths(env, path); err != nil {
		return nil, err
	}
	return env, nil
}

// validateEnvFilePaths checks that env values which look like local file paths
// (absolute paths that are not URLs) actually exist on disk.
// This catches missing prerequisite files early, before the service fails to start.
func validateEnvFilePaths(env map[string]string, envFilePath string) error {
	var missing []string
	for key, value := range env {
		if !strings.HasPrefix(value, "/") {
			continue
		}
		// Skip values that contain URL-like patterns (e.g. embedded in a longer string)
		if strings.Contains(value, "://") {
			continue
		}
		// Check if the path has a file extension — bare directory prefixes like /v1/mappings are API paths
		if filepath.Ext(value) == "" {
			continue
		}
		if _, err := os.Stat(value); err != nil {
			missing = append(missing, fmt.Sprintf("  %s=%s", key, value))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("env file %s references files that do not exist:\n%s",
			envFilePath, strings.Join(missing, "\n"))
	}
	return nil
}
