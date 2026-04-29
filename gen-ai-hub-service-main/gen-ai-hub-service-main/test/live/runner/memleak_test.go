/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"
)

// defaultMemleakRounds is the number of measurement rounds.
// Override with the MEMLEAK_ROUNDS environment variable.
const defaultMemleakRounds = 5

// TestMemLeak runs the test suite multiple rounds with profiling snapshots to detect memory leaks.
//
// Flow:
//  1. Start one configuration (via standard TestMain setup)
//  2. Warm-up run (populate caches, trigger lazy init — no snapshot)
//  3. N measurement rounds: run suite → force GC → capture snapshot
//  4. Compare first vs last snapshot and report leaks
//
// More rounds make detection more reliable by separating real leaks from heap noise.
//
// Activate with MEMLEAK=true. Configure with MEMLEAK_ROUNDS,
// LEAK_HEAP_THRESHOLD_PCT, LEAK_GOROUTINE_THRESHOLD, LEAK_THREAD_THRESHOLD.
func TestMemLeak(t *testing.T) {
	if strings.ToLower(os.Getenv("MEMLEAK")) != "true" {
		t.Skip("Memory leak detection disabled (set MEMLEAK=true to enable)")
	}

	if len(environments) == 0 {
		t.Fatal("No environments available for memory leak detection")
	}
	ce := environments[0]
	env := ce.env

	healthURL, err := resolveHealthcheckURL(env)
	if err != nil {
		t.Fatalf("Cannot resolve healthcheck URL for pprof: %v", err)
	}

	if len(filteredPrompts) == 0 {
		t.Fatal("No prompts available for memory leak detection")
	}

	rounds := int(getEnvInt64("MEMLEAK_ROUNDS", int64(defaultMemleakRounds)))
	if rounds < 1 {
		rounds = 1
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Printf("  Memory Leak Detection (run %s)\n", env.UniqueID)
	fmt.Printf("  Config:      %s\n", ce.name)
	fmt.Printf("  Prompts:     %d\n", len(filteredPrompts))
	fmt.Printf("  Chat targets:      %d\n", len(env.ChatCompletionTargets))
	fmt.Printf("  Embedding targets: %d\n", len(env.EmbeddingTargets))
	fmt.Printf("  Healthcheck URL:   %s\n", healthURL)
	fmt.Printf("  Measurement rounds: %d\n", rounds)
	fmt.Printf("  GET endpoints:     /, /models, /models/defaults, /health/liveness, /health/readiness\n")
	fmt.Println("================================================================================")
	fmt.Println()

	// --- Warm-up run (populate caches, trigger lazy init, no snapshot) ---
	fmt.Println("=== Warm-up: Initial run (no snapshot) ===")
	warmup := runTolerantSuite(ce.name, env, filteredPrompts, healthURL)
	fmt.Println()
	warmup.Print("Warm-up")
	time.Sleep(2 * time.Second)

	// --- Measurement rounds: run suite → capture snapshot ---
	snapshots := make([]ProfileSnapshot, 0, rounds)
	for i := 1; i <= rounds; i++ {
		label := fmt.Sprintf("round%d", i)

		fmt.Println()
		fmt.Printf("=== Round %d/%d ===\n", i, rounds)
		results := runTolerantSuite(ce.name, env, filteredPrompts, healthURL)
		fmt.Println()
		results.Print(fmt.Sprintf("Round %d", i))
		time.Sleep(2 * time.Second)

		fmt.Printf("  Capturing snapshot %d ...\n", i)
		snap, err := CaptureAndSaveSnapshot(healthURL, env.UniqueID, label)
		if err != nil {
			t.Fatalf("Failed to capture snapshot %d: %v", i, err)
		}
		snapshots = append(snapshots, snap)
		fmt.Printf("  Snapshot %d: %s\n", i, snap)
	}

	// --- Summary table with round-over-round deltas ---
	fmt.Println()
	fmt.Println("=== Snapshot Summary ===")
	fmt.Printf("  %-8s  %15s  %10s  %10s  %10s  %10s  %10s\n",
		"Round", "Heap InUse", "Heap Delta", "Objects", "Goroutines", "Gor Delta", "Threads")
	for i, snap := range snapshots {
		heapDelta := "—"
		gorDelta := "—"
		if i > 0 {
			heapDelta = formatBytesSign(snap.HeapInUseBytes - snapshots[i-1].HeapInUseBytes)
			gorDelta = fmt.Sprintf("%+d", snap.GoroutineCount-snapshots[i-1].GoroutineCount)
		}
		fmt.Printf("  %-8d  %15s  %10s  %10d  %10d  %10s  %10d\n",
			i+1, formatBytes(snap.HeapInUseBytes), heapDelta, snap.HeapInUseObjects,
			snap.GoroutineCount, gorDelta, snap.ThreadCount)
	}

	// --- Compare first vs last snapshot ---
	first := snapshots[0]
	last := snapshots[len(snapshots)-1]

	fmt.Println()
	fmt.Printf("=== Memory Leak Detection Results (round 1 vs round %d) ===\n", rounds)
	report := CompareSnapshots(first, last)
	fmt.Print(report.Format())

	fmt.Println()
	fmt.Println("  Saved pprof profiles for offline analysis:")
	for _, pt := range []string{"heap", "goroutine", "allocs"} {
		fmt.Printf("    go tool pprof -base %s %s\n",
			ProfileFilePath(env.UniqueID, "round1", pt),
			ProfileFilePath(env.UniqueID, fmt.Sprintf("round%d", rounds), pt))
	}
	fmt.Println()

	if report.HasLeaks() {
		t.Errorf("Memory leak detected!\n%s", report.Format())
	}
}

// defaultMemleakGETRepetitions is the number of times each GET endpoint is called per round.
// Override with the MEMLEAK_GET_REPETITIONS environment variable.
const defaultMemleakGETRepetitions = 50

// TestMemLeakGET runs only the non-inference GET endpoints (/models, /models/defaults,
// /health/liveness, /health/readiness) in a memory leak detection loop.
//
// This isolates the GET endpoint code paths from inference workload, making it easier
// to detect leaks in model listing, default resolution, and health check handlers.
//
// Activate with MEMLEAK=true. Configure with MEMLEAK_ROUNDS, MEMLEAK_GET_REPETITIONS,
// LEAK_HEAP_THRESHOLD_PCT, LEAK_GOROUTINE_THRESHOLD, LEAK_THREAD_THRESHOLD.
func TestMemLeakGET(t *testing.T) {
	if strings.ToLower(os.Getenv("MEMLEAK")) != "true" {
		t.Skip("Memory leak detection disabled (set MEMLEAK=true to enable)")
	}

	if len(environments) == 0 {
		t.Fatal("No environments available for memory leak detection")
	}
	ce := environments[0]
	env := ce.env

	healthURL, err := resolveHealthcheckURL(env)
	if err != nil {
		t.Fatalf("Cannot resolve healthcheck URL for pprof: %v", err)
	}

	rounds := int(getEnvInt64("MEMLEAK_ROUNDS", int64(defaultMemleakRounds)))
	if rounds < 1 {
		rounds = 1
	}

	repetitions := int(getEnvInt64("MEMLEAK_GET_REPETITIONS", int64(defaultMemleakGETRepetitions)))
	if repetitions < 1 {
		repetitions = 1
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Printf("  Memory Leak Detection — GET endpoints only (run %s)\n", env.UniqueID)
	fmt.Printf("  Config:            %s\n", ce.name)
	fmt.Printf("  Healthcheck URL:   %s\n", healthURL)
	fmt.Printf("  Measurement rounds: %d\n", rounds)
	fmt.Printf("  Repetitions/round: %d\n", repetitions)
	fmt.Printf("  Endpoints:         GET /, GET /models, GET /models/defaults,\n")
	fmt.Printf("                     GET /health/liveness, GET /health/readiness\n")
	fmt.Println("================================================================================")
	fmt.Println()

	// --- Warm-up run ---
	fmt.Println("=== Warm-up: Initial run (no snapshot) ===")
	warmup := runTolerantGETSuite(env, healthURL, repetitions)
	fmt.Println()
	warmup.Print("Warm-up")
	time.Sleep(2 * time.Second)

	// --- Measurement rounds ---
	snapshots := make([]ProfileSnapshot, 0, rounds)
	for i := 1; i <= rounds; i++ {
		label := fmt.Sprintf("round%d", i)

		fmt.Println()
		fmt.Printf("=== Round %d/%d ===\n", i, rounds)
		results := runTolerantGETSuite(env, healthURL, repetitions)
		fmt.Println()
		results.Print(fmt.Sprintf("Round %d", i))
		time.Sleep(2 * time.Second)

		fmt.Printf("  Capturing snapshot %d ...\n", i)
		snap, err := CaptureAndSaveSnapshot(healthURL, env.UniqueID, label)
		if err != nil {
			t.Fatalf("Failed to capture snapshot %d: %v", i, err)
		}
		snapshots = append(snapshots, snap)
		fmt.Printf("  Snapshot %d: %s\n", i, snap)
	}

	// --- Summary table ---
	fmt.Println()
	fmt.Println("=== Snapshot Summary ===")
	fmt.Printf("  %-8s  %15s  %10s  %10s  %10s  %10s  %10s\n",
		"Round", "Heap InUse", "Heap Delta", "Objects", "Goroutines", "Gor Delta", "Threads")
	for i, snap := range snapshots {
		heapDelta := "—"
		gorDelta := "—"
		if i > 0 {
			heapDelta = formatBytesSign(snap.HeapInUseBytes - snapshots[i-1].HeapInUseBytes)
			gorDelta = fmt.Sprintf("%+d", snap.GoroutineCount-snapshots[i-1].GoroutineCount)
		}
		fmt.Printf("  %-8d  %15s  %10s  %10d  %10d  %10s  %10d\n",
			i+1, formatBytes(snap.HeapInUseBytes), heapDelta, snap.HeapInUseObjects,
			snap.GoroutineCount, gorDelta, snap.ThreadCount)
	}

	// --- Compare first vs last ---
	first := snapshots[0]
	last := snapshots[len(snapshots)-1]

	fmt.Println()
	fmt.Printf("=== Memory Leak Detection Results (round 1 vs round %d) ===\n", rounds)
	report := CompareSnapshots(first, last)
	fmt.Print(report.Format())

	fmt.Println()
	fmt.Println("  Saved pprof profiles for offline analysis:")
	for _, pt := range []string{"heap", "goroutine", "allocs"} {
		fmt.Printf("    go tool pprof -base %s %s\n",
			ProfileFilePath(env.UniqueID, "round1", pt),
			ProfileFilePath(env.UniqueID, fmt.Sprintf("round%d", rounds), pt))
	}
	fmt.Println()

	if report.HasLeaks() {
		t.Errorf("Memory leak detected!\n%s", report.Format())
	}
}

// runTolerantGETSuite runs only non-inference GET endpoints in parallel.
// Used by TestMemLeakGET to isolate GET endpoint leak detection from inference workload.
func runTolerantGETSuite(env *TestEnvironment, healthcheckURL string, repetitions int) tolerantResults {
	var (
		results tolerantResults
		mu      sync.Mutex
		wg      sync.WaitGroup
	)
	client := &http.Client{Timeout: 30 * time.Second}

	var tasks []tolerantTask
	for i := 0; i < repetitions; i++ {
		idx := i + 1

		// GET / (public — exercises swagger/root handler and full middleware stack)
		rootURL := env.SvcBaseURL + "/"
		tasks = append(tasks, tolerantTask{
			label:   fmt.Sprintf("GET / [%d/%d]", idx, repetitions),
			run:     func() error { return tolerantGET(client, rootURL, "") },
			success: &results.RootSuccess,
			failure: &results.RootFailure,
		})

		modelsURL := env.SvcBaseURL + "/models"
		tasks = append(tasks, tolerantTask{
			label:   fmt.Sprintf("GET /models [%d/%d]", idx, repetitions),
			run:     func() error { return tolerantGET(client, modelsURL, env.JWTToken) },
			success: &results.ModelsSuccess,
			failure: &results.ModelsFailure,
		})

		defaultsURL := env.SvcBaseURL + "/models/defaults"
		tasks = append(tasks, tolerantTask{
			label:   fmt.Sprintf("GET /models/defaults [%d/%d]", idx, repetitions),
			run:     func() error { return tolerantGET(client, defaultsURL, env.JWTToken) },
			success: &results.DefaultsSuccess,
			failure: &results.DefaultsFailure,
		})

		if healthcheckURL != "" {
			livenessURL := healthcheckURL + "/health/liveness"
			tasks = append(tasks, tolerantTask{
				label:   fmt.Sprintf("GET /health/liveness [%d/%d]", idx, repetitions),
				run:     func() error { return tolerantGET(client, livenessURL, "") },
				success: &results.HealthSuccess,
				failure: &results.HealthFailure,
			})

			readinessURL := healthcheckURL + "/health/readiness"
			tasks = append(tasks, tolerantTask{
				label:   fmt.Sprintf("GET /health/readiness [%d/%d]", idx, repetitions),
				run:     func() error { return tolerantGET(client, readinessURL, "") },
				success: &results.HealthSuccess,
				failure: &results.HealthFailure,
			})
		}
	}

	for _, task := range tasks {
		wg.Add(1)
		go task.execute(&wg, &mu)
	}

	wg.Wait()
	return results
}

// tolerantResults tracks the outcome of a tolerant test suite run.
type tolerantResults struct {
	ChatSuccess      int
	ChatFailure      int
	StreamSuccess    int
	StreamFailure    int
	EmbeddingSuccess int
	EmbeddingFailure int

	// Non-inference endpoint results
	RootSuccess     int
	RootFailure     int
	ModelsSuccess   int
	ModelsFailure   int
	DefaultsSuccess int
	DefaultsFailure int
	HealthSuccess   int
	HealthFailure   int
}

func (r *tolerantResults) Total() int {
	return r.ChatSuccess + r.ChatFailure +
		r.StreamSuccess + r.StreamFailure +
		r.EmbeddingSuccess + r.EmbeddingFailure +
		r.RootSuccess + r.RootFailure +
		r.ModelsSuccess + r.ModelsFailure +
		r.DefaultsSuccess + r.DefaultsFailure +
		r.HealthSuccess + r.HealthFailure
}

func (r *tolerantResults) TotalSuccess() int {
	return r.ChatSuccess + r.StreamSuccess + r.EmbeddingSuccess +
		r.RootSuccess + r.ModelsSuccess + r.DefaultsSuccess + r.HealthSuccess
}

func (r *tolerantResults) Print(phase string) {
	fmt.Printf("  %s results: %d/%d succeeded\n", phase, r.TotalSuccess(), r.Total())
	fmt.Printf("    Chat completion:  %d OK, %d failed\n", r.ChatSuccess, r.ChatFailure)
	fmt.Printf("    Streaming:        %d OK, %d failed\n", r.StreamSuccess, r.StreamFailure)
	fmt.Printf("    Embeddings:       %d OK, %d failed\n", r.EmbeddingSuccess, r.EmbeddingFailure)
	fmt.Printf("    GET /:            %d OK, %d failed\n", r.RootSuccess, r.RootFailure)
	fmt.Printf("    GET /models:      %d OK, %d failed\n", r.ModelsSuccess, r.ModelsFailure)
	fmt.Printf("    GET /models/defaults: %d OK, %d failed\n", r.DefaultsSuccess, r.DefaultsFailure)
	fmt.Printf("    GET /health:      %d OK, %d failed\n", r.HealthSuccess, r.HealthFailure)
}

// testPath builds a test name in the same format as Go's testing.T.Run nesting,
// replacing spaces with underscores just like the test framework does.
func testPath(configName, promptName, testType, subTest string) string {
	return fmt.Sprintf("TestLive/%s/%s/%s/%s",
		configName, promptName, testType, strings.ReplaceAll(subTest, " ", "_"))
}

// tolerantTask represents a single test to run in parallel within the tolerant suite.
type tolerantTask struct {
	label   string
	run     func() error
	success *int
	failure *int
}

func (task tolerantTask) execute(wg *sync.WaitGroup, mu *sync.Mutex) {
	defer wg.Done()
	err := task.run()
	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		fmt.Printf("    %s ... FAILED (ignored: %v)\n", task.label, err)
		*task.failure++
	} else {
		fmt.Printf("    %s ... OK\n", task.label)
		*task.success++
	}
}

// runTolerantSuite runs all test types against all targets and prompts in parallel,
// including non-inference endpoints (models, defaults, health).
// Failures are logged but never cause a test failure — the goal is to exercise
// the service code paths so that memory leaks can be detected.
func runTolerantSuite(configName string, env *TestEnvironment, prompts []string, healthcheckURL string) tolerantResults {
	var (
		results tolerantResults
		mu      sync.Mutex
		wg      sync.WaitGroup
	)
	client := &http.Client{Timeout: DefaultTimeout}

	type testSpec struct {
		testType string
		subFmt   string
		targets  []ModelTarget
		payload  func(ModelTarget) ([]byte, error)
		url      func(ModelTarget) string
		success  *int
		failure  *int
	}

	var tasks []tolerantTask
	for _, promptPath := range prompts {
		promptName := filepath.Base(promptPath)
		systemPrompt := filepath.Join(promptPath, "system-prompt")
		userPrompt := filepath.Join(promptPath, "user-prompt")
		embeddingsInput := filepath.Join(promptPath, "embeddings-input")

		specs := []testSpec{
			{
				testType: "ChatCompletion",
				subFmt:   "should complete successfully for %s",
				targets:  env.ChatCompletionTargets,
				payload: func(t ModelTarget) ([]byte, error) {
					return buildChatPayloadTolerant(t, systemPrompt, userPrompt, false)
				},
				url:     func(t ModelTarget) string { return env.SvcBaseURL + t.RequestPath() },
				success: &results.ChatSuccess,
				failure: &results.ChatFailure,
			},
			{
				testType: "ChatCompletionStreaming",
				subFmt:   "should stream successfully for %s",
				targets:  env.ChatCompletionTargets,
				payload: func(t ModelTarget) ([]byte, error) {
					return buildChatPayloadTolerant(t, systemPrompt, userPrompt, true)
				},
				url:     func(t ModelTarget) string { return env.SvcBaseURL + t.StreamingRequestPath() },
				success: &results.StreamSuccess,
				failure: &results.StreamFailure,
			},
			{
				testType: "Embeddings",
				subFmt:   "should embed successfully for %s",
				targets:  env.EmbeddingTargets,
				payload:  func(t ModelTarget) ([]byte, error) { return buildEmbeddingPayloadTolerant(t, embeddingsInput) },
				url:      func(t ModelTarget) string { return env.SvcBaseURL + t.EmbeddingsPath() },
				success:  &results.EmbeddingSuccess,
				failure:  &results.EmbeddingFailure,
			},
		}

		for _, spec := range specs {
			for _, target := range spec.targets {
				label := testPath(configName, promptName, spec.testType, fmt.Sprintf(spec.subFmt, target))
				payload, err := spec.payload(target)
				if err != nil {
					fmt.Printf("    %s ... FAILED (ignored: payload build: %v)\n", label, err)
					*spec.failure++
					continue
				}
				url := spec.url(target)
				tasks = append(tasks, tolerantTask{
					label:   label,
					run:     func() error { return tolerantRequest(client, env, url, payload) },
					success: spec.success,
					failure: spec.failure,
				})
			}
		}
	}

	// Non-inference GET endpoints — run multiple times per round to match inference load.
	// The repetition count scales with the number of chat targets (minimum 5).
	getRepetitions := len(env.ChatCompletionTargets)
	if getRepetitions < 5 {
		getRepetitions = 5
	}

	for i := 0; i < getRepetitions; i++ {
		idx := i + 1

		// GET / (public — exercises swagger/root handler and full middleware stack)
		rootURL := env.SvcBaseURL + "/"
		tasks = append(tasks, tolerantTask{
			label:   fmt.Sprintf("GET / [%d/%d]", idx, getRepetitions),
			run:     func() error { return tolerantGET(client, rootURL, "") },
			success: &results.RootSuccess,
			failure: &results.RootFailure,
		})

		// GET /models (authenticated — exercises model listing and provider aggregation)
		modelsURL := env.SvcBaseURL + "/models"
		tasks = append(tasks, tolerantTask{
			label:   fmt.Sprintf("GET /models [%d/%d]", idx, getRepetitions),
			run:     func() error { return tolerantGET(client, modelsURL, env.JWTToken) },
			success: &results.ModelsSuccess,
			failure: &results.ModelsFailure,
		})

		// GET /models/defaults (authenticated — exercises default model resolution)
		defaultsURL := env.SvcBaseURL + "/models/defaults"
		tasks = append(tasks, tolerantTask{
			label:   fmt.Sprintf("GET /models/defaults [%d/%d]", idx, getRepetitions),
			run:     func() error { return tolerantGET(client, defaultsURL, env.JWTToken) },
			success: &results.DefaultsSuccess,
			failure: &results.DefaultsFailure,
		})

		// GET /health/liveness (no auth — healthcheck port)
		if healthcheckURL != "" {
			livenessURL := healthcheckURL + "/health/liveness"
			tasks = append(tasks, tolerantTask{
				label:   fmt.Sprintf("GET /health/liveness [%d/%d]", idx, getRepetitions),
				run:     func() error { return tolerantGET(client, livenessURL, "") },
				success: &results.HealthSuccess,
				failure: &results.HealthFailure,
			})

			// GET /health/readiness (no auth — healthcheck port)
			readinessURL := healthcheckURL + "/health/readiness"
			tasks = append(tasks, tolerantTask{
				label:   fmt.Sprintf("GET /health/readiness [%d/%d]", idx, getRepetitions),
				run:     func() error { return tolerantGET(client, readinessURL, "") },
				success: &results.HealthSuccess,
				failure: &results.HealthFailure,
			})
		}
	}

	for _, task := range tasks {
		wg.Add(1)
		go task.execute(&wg, &mu)
	}

	wg.Wait()
	return results
}

// tolerantRequest sends an authorized POST request, drains the response body,
// and returns an error on non-200 status. Failures are returned as errors instead of calling t.Fatal,
// so that the memory leak test can continue exercising code paths regardless of individual results.
func tolerantRequest(client *http.Client, env *TestEnvironment, url string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+env.JWTToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// tolerantGET sends a GET request, drains the response body, and returns an error on non-200 status.
// If authToken is non-empty, an Authorization header is included.
// Like tolerantRequest, failures are returned as errors to allow memleak detection to continue.
func tolerantGET(client *http.Client, url, authToken string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

// --- Tolerant payload builders (return errors instead of calling t.Fatal) ---

func readAndEncodeFileTolerant(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	encoded, err := json.Marshal(string(data))
	if err != nil {
		return "", fmt.Errorf("json encode: %w", err)
	}
	return string(encoded), nil
}

func renderAndValidateTolerant(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render template %q: %w", tmpl.Name(), err)
	}
	var check map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		return nil, fmt.Errorf("invalid JSON from template %q: %w", tmpl.Name(), err)
	}
	return buf.Bytes(), nil
}

func buildChatPayloadTolerant(target ModelTarget, systemPromptPath, userPromptPath string, streaming bool) ([]byte, error) {
	systemPrompt, err := readAndEncodeFileTolerant(systemPromptPath)
	if err != nil {
		return nil, err
	}
	userPrompt, err := readAndEncodeFileTolerant(userPromptPath)
	if err != nil {
		return nil, err
	}

	data := chatCompletionData{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
	}

	var tmpl *template.Template
	switch target.Provider {
	case ProviderGoogle:
		modelJSON, err := json.Marshal(fmt.Sprintf("google/%s", target.Model))
		if err != nil {
			return nil, fmt.Errorf("json encode model: %w", err)
		}
		data.Model = string(modelJSON)
		if streaming {
			tmpl = chatCompletionGoogleStreamTemplate
		} else {
			tmpl = chatCompletionGoogleTemplate
		}
	case ProviderAnthropic, ProviderAmazon:
		tmpl = converseTemplate
	default:
		if streaming {
			tmpl = chatCompletionStreamTemplate
		} else {
			tmpl = chatCompletionTemplate
		}
	}

	return renderAndValidateTolerant(tmpl, data)
}

func buildEmbeddingPayloadTolerant(target ModelTarget, inputPath string) ([]byte, error) {
	input, err := readAndEncodeFileTolerant(inputPath)
	if err != nil {
		return nil, err
	}

	data := embeddingData{Input: input}

	tmpl := embeddingsOpenAITemplate
	if target.Provider == ProviderAmazon {
		if isNovaModel(target.Model) {
			tmpl = embeddingsAmazonNovaTemplate
		} else {
			tmpl = embeddingsAmazonTemplate
		}
	}

	return renderAndValidateTolerant(tmpl, data)
}
