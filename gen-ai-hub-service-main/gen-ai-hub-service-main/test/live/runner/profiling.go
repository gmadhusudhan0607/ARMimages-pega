//
// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.
//

package runner

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/pprof/profile"
)

// defaultLeakHeapThresholdPct is the percentage of heap inuse_space growth that triggers a leak warning.
// Override with the LEAK_HEAP_THRESHOLD_PCT environment variable.
const defaultLeakHeapThresholdPct = 20.0

// defaultLeakHeapMinBytes is the minimum absolute heap growth (in bytes) required
// before the percentage threshold is evaluated. Small fluctuations from Go runtime
// internals (connection buffers, GC timing) are ignored when below this floor.
// Override with the LEAK_HEAP_MIN_BYTES environment variable.
const defaultLeakHeapMinBytes int64 = 5 * 1024 * 1024 // 5 MB

// defaultLeakGoroutineThreshold is the number of additional goroutines that triggers a leak warning.
// Override with the LEAK_GOROUTINE_THRESHOLD environment variable.
const defaultLeakGoroutineThreshold int64 = 5

// defaultLeakThreadThreshold is the number of additional OS threads that triggers a leak warning.
// Override with the LEAK_THREAD_THRESHOLD environment variable.
const defaultLeakThreadThreshold int64 = 100

// pprofHTTPTimeout is the timeout for each pprof HTTP request.
const pprofHTTPTimeout = 30 * time.Second

// pprofEndpoints defines the pprof profiles to capture.
// The heap endpoint uses ?gc=1 to force runtime.GC() before sampling.
var pprofEndpoints = []struct {
	name string
	path string
}{
	{"heap", "/debug/pprof/heap?gc=1"},
	{"goroutine", "/debug/pprof/goroutine"},
	{"threadcreate", "/debug/pprof/threadcreate"},
	{"allocs", "/debug/pprof/allocs"},
}

// ProfileSnapshot holds parsed pprof metric data captured at a single point in time.
type ProfileSnapshot struct {
	// Heap metrics (from /debug/pprof/heap?gc=1 — GC is forced before sampling)
	HeapInUseBytes   int64
	HeapInUseObjects int64

	// Goroutine count (from /debug/pprof/goroutine)
	GoroutineCount int64

	// OS thread count (from /debug/pprof/threadcreate)
	ThreadCount int64

	// Cumulative allocation metrics (from /debug/pprof/allocs)
	AllocBytes   int64
	AllocObjects int64
}

// String returns a human-readable one-line summary of the snapshot.
func (s ProfileSnapshot) String() string {
	return fmt.Sprintf(
		"Heap InUse: %s (%d objects), Goroutines: %d, Threads: %d, Allocs: %s (%d objects)",
		formatBytes(s.HeapInUseBytes), s.HeapInUseObjects,
		s.GoroutineCount,
		s.ThreadCount,
		formatBytes(s.AllocBytes), s.AllocObjects,
	)
}

// LeakReport summarises the differences between two ProfileSnapshots.
type LeakReport struct {
	// Heap deltas
	HeapInUseDeltaBytes   int64
	HeapInUseDeltaObjects int64
	HeapInUsePctGrowth    float64

	// Goroutine delta
	GoroutineDelta int64

	// Thread delta
	ThreadDelta int64

	// Alloc deltas (cumulative — expected to grow; informational only)
	AllocDeltaBytes   int64
	AllocDeltaObjects int64

	// Configured thresholds (set at construction time)
	heapThresholdPct   float64
	heapMinBytes       int64
	goroutineThreshold int64
	threadThreshold    int64
}

// HasLeaks returns true if any metric exceeds the configured thresholds.
// Heap growth must exceed both the percentage threshold AND the absolute minimum
// to be considered a leak — this avoids false positives from small Go runtime
// fluctuations (e.g. connection buffers, GC timing noise).
func (r *LeakReport) HasLeaks() bool {
	heapLeaked := r.HeapInUseDeltaBytes > r.heapMinBytes && r.HeapInUsePctGrowth > r.heapThresholdPct
	return heapLeaked ||
		r.GoroutineDelta > r.goroutineThreshold ||
		r.ThreadDelta > r.threadThreshold
}

// Format returns a printable multi-line diff report with pass/fail indicators per metric.
func (r *LeakReport) Format() string {
	var sb strings.Builder

	heapStatus := passFailIcon(!(r.HeapInUseDeltaBytes > r.heapMinBytes && r.HeapInUsePctGrowth > r.heapThresholdPct))
	goroutineStatus := passFailIcon(r.GoroutineDelta <= r.goroutineThreshold)
	threadStatus := passFailIcon(r.ThreadDelta <= r.threadThreshold)

	sb.WriteString("  Memory leak detection results:\n")
	sb.WriteString(fmt.Sprintf("    %-28s %s (%+.1f%%)  %s (threshold: %.0f%% AND > %s)\n",
		"Heap growth (inuse_space):",
		formatBytesSign(r.HeapInUseDeltaBytes),
		r.HeapInUsePctGrowth,
		heapStatus,
		r.heapThresholdPct,
		formatBytes(r.heapMinBytes),
	))
	sb.WriteString(fmt.Sprintf("    %-28s %+d objects\n",
		"Heap objects growth:",
		r.HeapInUseDeltaObjects,
	))
	sb.WriteString(fmt.Sprintf("    %-28s %+d  %s (threshold: %d)\n",
		"Goroutine delta:",
		r.GoroutineDelta,
		goroutineStatus,
		r.goroutineThreshold,
	))
	sb.WriteString(fmt.Sprintf("    %-28s %+d  %s (threshold: %d)\n",
		"Thread delta:",
		r.ThreadDelta,
		threadStatus,
		r.threadThreshold,
	))
	sb.WriteString(fmt.Sprintf("    %-28s %s (%+d objects)  [cumulative, informational]\n",
		"Alloc delta:",
		formatBytesSign(r.AllocDeltaBytes),
		r.AllocDeltaObjects,
	))
	return sb.String()
}

// fetchRawProfiles fetches all pprof endpoints and returns the raw data keyed by endpoint name.
func fetchRawProfiles(healthcheckBaseURL string) (map[string][]byte, error) {
	client := &http.Client{Timeout: pprofHTTPTimeout}

	raw := make(map[string][]byte, len(pprofEndpoints))
	for _, ep := range pprofEndpoints {
		data, err := fetchProfile(client, healthcheckBaseURL+ep.path)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s profile: %w", ep.name, err)
		}
		raw[ep.name] = data
	}
	return raw, nil
}

// parseSnapshot extracts metrics from raw pprof data into a ProfileSnapshot.
func parseSnapshot(raw map[string][]byte) (ProfileSnapshot, error) {
	var snap ProfileSnapshot

	heapInUse, err := sumProfileMetrics(raw["heap"], "inuse_space", "inuse_objects")
	if err != nil {
		return ProfileSnapshot{}, fmt.Errorf("failed to parse heap profile: %w", err)
	}
	snap.HeapInUseBytes = heapInUse[0]
	snap.HeapInUseObjects = heapInUse[1]

	snap.GoroutineCount = sumFirstSampleValue(raw["goroutine"])
	snap.ThreadCount = sumFirstSampleValue(raw["threadcreate"])

	allocs, err := sumProfileMetrics(raw["allocs"], "alloc_space", "alloc_objects")
	if err != nil {
		return ProfileSnapshot{}, fmt.Errorf("failed to parse allocs profile: %w", err)
	}
	snap.AllocBytes = allocs[0]
	snap.AllocObjects = allocs[1]

	return snap, nil
}

// CompareSnapshots computes the difference between two snapshots and returns a LeakReport.
// Thresholds are read from environment variables (or defaults when not set).
func CompareSnapshots(before, after ProfileSnapshot) LeakReport {
	heapDeltaBytes := after.HeapInUseBytes - before.HeapInUseBytes
	heapPctGrowth := 0.0
	if before.HeapInUseBytes > 0 {
		heapPctGrowth = float64(heapDeltaBytes) / float64(before.HeapInUseBytes) * 100.0
	}

	return LeakReport{
		HeapInUseDeltaBytes:   heapDeltaBytes,
		HeapInUseDeltaObjects: after.HeapInUseObjects - before.HeapInUseObjects,
		HeapInUsePctGrowth:    heapPctGrowth,
		GoroutineDelta:        after.GoroutineCount - before.GoroutineCount,
		ThreadDelta:           after.ThreadCount - before.ThreadCount,
		AllocDeltaBytes:       after.AllocBytes - before.AllocBytes,
		AllocDeltaObjects:     after.AllocObjects - before.AllocObjects,
		heapThresholdPct:      getEnvFloat("LEAK_HEAP_THRESHOLD_PCT", defaultLeakHeapThresholdPct),
		heapMinBytes:          getEnvInt64("LEAK_HEAP_MIN_BYTES", defaultLeakHeapMinBytes),
		goroutineThreshold:    getEnvInt64("LEAK_GOROUTINE_THRESHOLD", defaultLeakGoroutineThreshold),
		threadThreshold:       getEnvInt64("LEAK_THREAD_THRESHOLD", defaultLeakThreadThreshold),
	}
}

// ProfileFilePath returns the file path for a saved pprof profile.
// runID uniquely identifies the test run (to allow parallel execution on the same host).
func ProfileFilePath(runID, label, profileType string) string {
	return fmt.Sprintf("/tmp/live-test-memleak-%s-%s-%s.prof", runID, label, profileType)
}

// saveRawProfiles writes previously fetched raw pprof data to /tmp files for offline analysis.
func saveRawProfiles(raw map[string][]byte, runID, label string) {
	for _, ep := range pprofEndpoints {
		data, ok := raw[ep.name]
		if !ok {
			continue
		}
		path := ProfileFilePath(runID, label, ep.name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			fmt.Printf("    Warning: could not write %s profile to %s: %v\n", ep.name, path, err)
			continue
		}
		logVerbosef("    Saved %s profile → %s\n", ep.name, path)
	}
}

// CaptureAndSaveSnapshot fetches pprof profiles once, parses them into a snapshot,
// and saves the raw data to disk — avoiding a second round-trip to the service.
func CaptureAndSaveSnapshot(healthcheckBaseURL, runID, label string) (ProfileSnapshot, error) {
	raw, err := fetchRawProfiles(healthcheckBaseURL)
	if err != nil {
		return ProfileSnapshot{}, err
	}
	snap, err := parseSnapshot(raw)
	if err != nil {
		return ProfileSnapshot{}, err
	}
	saveRawProfiles(raw, runID, label)
	return snap, nil
}

// resolveHealthcheckURL returns the pprof-accessible healthcheck base URL for an environment.
// For locally started services this is env.SvcHealthcheckURL.
// For external services the caller should set the HEALTHCHECK_URL env variable.
func resolveHealthcheckURL(env *TestEnvironment) (string, error) {
	if env.SvcHealthcheckURL != "" {
		return env.SvcHealthcheckURL, nil
	}

	if u := os.Getenv("HEALTHCHECK_URL"); u != "" {
		return u, nil
	}

	return "", fmt.Errorf(
		"no healthcheck URL available for pprof: " +
			"SvcHealthcheckURL is empty (external service) and HEALTHCHECK_URL env var is not set. " +
			"Set HEALTHCHECK_URL=http://<host>:<port> to point at the service's healthcheck/pprof port",
	)
}

// ---- internal helpers ----

func fetchProfile(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, url, string(body))
	}
	return io.ReadAll(resp.Body)
}

// sumProfileMetrics parses a pprof profile and sums the values for the given sample type names.
// Returns a slice of sums in the same order as the requested type names.
func sumProfileMetrics(raw []byte, typeNames ...string) ([]int64, error) {
	p, err := profile.Parse(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	// Map each requested type name to its index in the profile's SampleType list.
	indices := make([]int, len(typeNames))
	for i := range indices {
		indices[i] = -1
	}
	for i, st := range p.SampleType {
		for j, name := range typeNames {
			if st.Type == name {
				indices[j] = i
			}
		}
	}

	sums := make([]int64, len(typeNames))
	for _, s := range p.Sample {
		for j, idx := range indices {
			if idx >= 0 && idx < len(s.Value) {
				sums[j] += s.Value[idx]
			}
		}
	}
	return sums, nil
}

// sumFirstSampleValue sums the first value of every sample (used for goroutine/thread counts).
func sumFirstSampleValue(raw []byte) int64 {
	p, err := profile.Parse(bytes.NewReader(raw))
	if err != nil {
		return 0
	}
	var total int64
	for _, s := range p.Sample {
		if len(s.Value) > 0 {
			total += s.Value[0]
		}
	}
	return total
}

// formatBytes converts bytes to a human-readable string (B / KB / MB / GB).
func formatBytes(b int64) string {
	abs := b
	if abs < 0 {
		abs = -b
	}
	switch {
	case abs >= 1024*1024*1024:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(1024*1024*1024))
	case abs >= 1024*1024:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(1024*1024))
	case abs >= 1024:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(1024))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// formatBytesSign is like formatBytes but always shows a +/- sign prefix.
func formatBytesSign(b int64) string {
	if b >= 0 {
		return "+" + formatBytes(b)
	}
	return formatBytes(b)
}

// passFailIcon returns a pass or fail indicator string.
func passFailIcon(ok bool) string {
	if ok {
		return "✓ OK"
	}
	return "✗ LEAK"
}

// getEnvFloat reads a float64 from an environment variable, returning def if not set or invalid.
func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// getEnvInt64 reads an int64 from an environment variable, returning def if not set or invalid.
func getEnvInt64(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return def
}
