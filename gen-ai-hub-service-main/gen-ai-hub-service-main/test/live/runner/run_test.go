/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	configsDir = "../configs"
	promptsDir = "../prompts"
)

// configEnvironment holds a started environment and its config name.
type configEnvironment struct {
	name string
	env  *TestEnvironment
}

var environments []configEnvironment

// filteredPrompts is populated in TestMain and reused in TestLive
// so that PROMPT env filtering is applied consistently.
var filteredPrompts []string

func TestMain(m *testing.M) {
	configFilter := os.Getenv("CONFIG")
	promptFilter := os.Getenv("PROMPT")

	// Check for external service URLs
	externalOpsURL := os.Getenv("OPS_URL")
	externalServiceURL := os.Getenv("SERVICE_URL")

	// Validate: both must be provided together, or neither
	if (externalOpsURL != "") != (externalServiceURL != "") {
		if externalOpsURL != "" {
			log.Fatalf("Error: OPS_URL is set but SERVICE_URL is missing. Both must be provided together.\n" +
				"Usage: make test-live RUN=all OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080")
		} else {
			log.Fatalf("Error: SERVICE_URL is set but OPS_URL is missing. Both must be provided together.\n" +
				"Usage: make test-live RUN=all OPS_URL=http://localhost:8081 SERVICE_URL=http://localhost:8080")
		}
	}

	useExternalServices := externalOpsURL != "" && externalServiceURL != ""

	prompts := discoverDirs(promptsDir)
	if len(prompts) == 0 {
		log.Fatalf("No prompts found in %s", promptsDir)
	}

	if promptFilter != "" {
		filtered := filterByName(prompts, promptFilter)
		if len(filtered) == 0 {
			log.Fatalf("PROMPT=%q not found in %s (available: %v)", promptFilter, promptsDir, dirNames(prompts))
		}
		prompts = filtered
	}

	filteredPrompts = prompts

	// When external URLs are provided, configs are completely irrelevant
	if useExternalServices {
		logVerbose("=== Using external OPS and SERVICE - configs are ignored ===")
		logVerbosef("  OPS_URL:     %s\n", externalOpsURL)
		logVerbosef("  SERVICE_URL: %s\n", externalServiceURL)
		logVerbosef("=== Live test matrix: 1 environment × %d prompt(s) ===\n", len(prompts))
		for _, p := range prompts {
			logVerbosef("  prompt: %s\n", filepath.Base(p))
		}
		logVerbose("")

		// Setup single environment using external services (empty paths - won't be used)
		logVerbose("=== Setting up environment with external services ===")
		env, err := SetupEnvironment("", "")
		if err != nil {
			log.Fatalf("Failed to setup environment with external services: %v", err)
		}
		environments = append(environments, configEnvironment{name: "external", env: env})
	} else {
		// Standard behavior: discover and use configs
		configs := discoverDirs(configsDir)
		if len(configs) == 0 {
			log.Fatalf("No configurations found in %s", configsDir)
		}

		if configFilter != "" {
			filtered := filterByName(configs, configFilter)
			if len(filtered) == 0 {
				log.Fatalf("CONFIG=%q not found in %s (available: %v)", configFilter, configsDir, dirNames(configs))
			}
			configs = filtered
		}

		logVerbosef("=== Live test matrix: %d config(s) × %d prompt(s) ===\n", len(configs), len(prompts))
		for _, c := range configs {
			logVerbosef("  config: %s\n", filepath.Base(c))
		}
		for _, p := range prompts {
			logVerbosef("  prompt: %s\n", filepath.Base(p))
		}
		logVerbose("")

		// Start an environment for each configuration
		for _, cfgPath := range configs {
			cfgName := filepath.Base(cfgPath)
			svcEnvFile := filepath.Join(cfgPath, "env.genai-hub-service")
			opsEnvFile := filepath.Join(cfgPath, "env.genai-gateway-ops")

			logVerbosef("=== Setting up environment for config: %s ===\n", cfgName)
			env, err := SetupEnvironment(svcEnvFile, opsEnvFile)
			if err != nil {
				// Tear down any already-started environments
				for _, e := range environments {
					e.env.Teardown()
				}
				log.Fatalf("Failed to setup environment for config %q: %v", cfgName, err)
			}
			environments = append(environments, configEnvironment{name: cfgName, env: env})
		}
	}

	exitCode := m.Run()

	logVerbose("")
	logVerbose("=== Cleanup starting ===")
	for _, e := range environments {
		logVerbosef("Tearing down environment for config: %s\n", e.name)
		e.env.Teardown()
	}

	os.Exit(exitCode)
}

func TestLive(t *testing.T) {
	for _, ce := range environments {
		t.Run(ce.name, func(t *testing.T) {
			for _, promptPath := range filteredPrompts {
				promptName := filepath.Base(promptPath)

				t.Run(promptName, func(t *testing.T) {
					systemPrompt := filepath.Join(promptPath, "system-prompt")
					userPrompt := filepath.Join(promptPath, "user-prompt")
					embeddingsInput := filepath.Join(promptPath, "embeddings-input")

					// Image generation prompts only run ImageGeneration tests
					isImageGenerationPrompt := strings.HasPrefix(promptName, "image-generation")

					if !isImageGenerationPrompt {
						// Only run chat and embedding tests for non-image-generation prompts
						t.Run("ChatCompletion", func(t *testing.T) {
							RunChatCompletionSuite(t, ce.env, systemPrompt, userPrompt, DefaultTimeout)
						})

						t.Run("ChatCompletionStreaming", func(t *testing.T) {
							RunStreamingChatCompletionSuite(t, ce.env, systemPrompt, userPrompt, DefaultTimeout)
						})

						t.Run("Embeddings", func(t *testing.T) {
							RunEmbeddingSuite(t, ce.env, embeddingsInput, DefaultTimeout)
						})
					}

					if isImageGenerationPrompt {
						// Only run image generation tests for image-generation prompt
						t.Run("ImageGeneration", func(t *testing.T) {
							RunImageGenerationSuite(t, ce.env, userPrompt, DefaultTimeout)
						})
					}
				})
			}
		})
	}
}

func TestLiveModelsDiscovery(t *testing.T) {
	for _, ce := range environments {
		t.Run(ce.name, func(t *testing.T) {
			t.Run("RealtimeModelsPresent", func(t *testing.T) {
				targets, err := FetchRealtimeTargets(ce.env.SvcBaseURL, ce.env.JWTToken)
				if err != nil {
					t.Fatalf("Failed to fetch realtime targets from /models: %v", err)
				}
				if len(targets) == 0 {
					t.Error("Expected realtime models in /models response, but found none")
				}

				// Verify expected model families are present.
				// Models may appear with version suffix (e.g. gpt-realtime-mini-2025-10-06)
				// so we match by exact name or name followed by a date suffix.
				expectedFamilies := []string{"gpt-realtime", "gpt-realtime-mini", "gpt-realtime-1.5"}
				for _, target := range targets {
					t.Logf("Found realtime model: %s", target.String())
				}
				for _, family := range expectedFamilies {
					found := false
					for _, target := range targets {
						if target.Model == family || strings.HasPrefix(target.Model, family+"-20") {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected model family %q in /models response, but not found", family)
					}
				}
			})
		})
	}
}

func TestLiveWebRTC(t *testing.T) {
	for _, ce := range environments {
		t.Run(ce.name, func(t *testing.T) {
			for _, promptPath := range filteredPrompts {
				promptName := filepath.Base(promptPath)
				if promptName == "image-generation" {
					continue
				}
				t.Run(promptName, func(t *testing.T) {
					systemPrompt := readPromptFile(t, filepath.Join(promptPath, "system-prompt"))
					userPrompt := readPromptFile(t, filepath.Join(promptPath, "user-prompt"))

					models := discoverRealtimeModels(t, ce.env)
					for _, model := range models {
						t.Run(model, func(t *testing.T) {
							RunWebRTCRealtimeTest(t, ce.env.SvcBaseURL, ce.env.JWTToken, model, systemPrompt, userPrompt)
						})
					}
				})
			}
		})
	}
}

func TestLiveWebRTCAudio(t *testing.T) {
	for _, ce := range environments {
		t.Run(ce.name, func(t *testing.T) {
			for _, promptPath := range filteredPrompts {
				promptName := filepath.Base(promptPath)
				if promptName == "image-generation" {
					continue
				}
				t.Run(promptName, func(t *testing.T) {
					systemPrompt := readPromptFile(t, filepath.Join(promptPath, "system-prompt"))
					userPrompt := readPromptFile(t, filepath.Join(promptPath, "user-prompt"))

					models := discoverRealtimeModels(t, ce.env)
					for _, model := range models {
						t.Run(model, func(t *testing.T) {
							RunWebRTCRealtimeAudioTest(t, ce.env.SvcBaseURL, ce.env.JWTToken, model, systemPrompt, userPrompt)
						})
					}
				})
			}
		})
	}
}

// readPromptFile reads a prompt file and returns its contents as a trimmed string.
func readPromptFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read prompt file %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}

// discoverRealtimeModels discovers realtime models, applying the MODEL env filter if set.
// Uses cached AllModels from environment setup when available to avoid an extra GET /models call.
// Falls back to the hardcoded RealtimeModels list if no models are found.
func discoverRealtimeModels(t *testing.T, env *TestEnvironment) []string {
	t.Helper()

	// Use cached models from environment setup if available, otherwise fetch fresh.
	var discovered []ModelTarget
	if len(env.AllModels) > 0 {
		discovered = FilterByType(env.AllModels, ModelTypeRealtime)
	} else {
		var err error
		discovered, err = FetchRealtimeTargets(env.SvcBaseURL, env.JWTToken)
		if err != nil {
			t.Logf("Warning: failed to discover realtime models from /models: %v, falling back to hardcoded list", err)
		}
	}

	var models []string
	if len(discovered) > 0 {
		for _, d := range discovered {
			models = append(models, d.Model)
		}
	} else {
		models = RealtimeModels
	}

	if modelFilter := os.Getenv("MODEL"); modelFilter != "" {
		var filtered []string
		for _, m := range models {
			if m == modelFilter {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}

	if len(models) == 0 {
		t.Skip("No realtime models discovered or matching MODEL filter")
	}

	return models
}
