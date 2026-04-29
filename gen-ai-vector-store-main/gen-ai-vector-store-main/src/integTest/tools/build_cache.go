// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// BuildCache manages test binary builds with intelligent caching
type BuildCache struct {
	mu            sync.Mutex
	projectRoot   string
	buildInfo     map[string]*buildMetadata
	forceRebuild  bool
	buildAttempts map[string]int // Track build attempts to avoid infinite loops
}

type buildMetadata struct {
	binaryPath      string
	sourcePath      string
	lastBuildTime   time.Time
	sourceChecksum  string
	buildSuccessful bool
}

var (
	globalBuildCache   *BuildCache
	globalBuildCacheMu sync.Mutex
)

// GetBuildCache returns the global build cache instance (singleton)
func GetBuildCache() *BuildCache {
	globalBuildCacheMu.Lock()
	defer globalBuildCacheMu.Unlock()

	if globalBuildCache == nil {
		projectRoot, err := findProjectRoot()
		if err != nil {
			// Fallback to current directory if project root not found
			projectRoot, _ = os.Getwd()
		}

		globalBuildCache = &BuildCache{
			projectRoot:   projectRoot,
			buildInfo:     make(map[string]*buildMetadata),
			forceRebuild:  false,
			buildAttempts: make(map[string]int),
		}
	}

	return globalBuildCache
}

// ResetBuildCache forces a reset of the build cache (useful for testing)
func ResetBuildCache() {
	globalBuildCacheMu.Lock()
	defer globalBuildCacheMu.Unlock()
	globalBuildCache = nil
}

// SetForceRebuild enables or disables force rebuild mode
func (bc *BuildCache) SetForceRebuild(force bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.forceRebuild = force
}

// IsForceRebuild returns whether force rebuild is enabled
func (bc *BuildCache) IsForceRebuild() bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.forceRebuild
}

// EnsureBinary ensures a test binary is built, using cache when possible
// Returns the absolute path to the binary
func (bc *BuildCache) EnsureBinary(ctx context.Context, config ServiceConfig) (string, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Create absolute paths
	absoluteBinaryPath := filepath.Join(bc.projectRoot, config.BinaryPath)
	absoluteSourcePath := filepath.Join(bc.projectRoot, config.SourcePath)

	cacheKey := config.ServiceName

	// Check if we've attempted to build this too many times
	if attempts, exists := bc.buildAttempts[cacheKey]; exists && attempts > 3 {
		return "", fmt.Errorf("build failed after %d attempts for %s", attempts, config.ServiceName)
	}

	// Get or create build metadata
	metadata, exists := bc.buildInfo[cacheKey]
	if !exists {
		metadata = &buildMetadata{
			binaryPath: absoluteBinaryPath,
			sourcePath: absoluteSourcePath,
		}
		bc.buildInfo[cacheKey] = metadata
	}

	// Check if rebuild is needed
	needsRebuild, reason := bc.needsRebuild(metadata, absoluteSourcePath, absoluteBinaryPath)

	if !needsRebuild && !bc.forceRebuild {
		// Binary is up to date, return existing path
		fmt.Printf("Using cached binary for %s\n", config.ServiceName)
		return absoluteBinaryPath, nil
	}

	// Build is needed
	fmt.Printf("Building %s (%s)...\n", config.ServiceName, reason)

	// Track build attempt
	bc.buildAttempts[cacheKey]++

	// Ensure bin directory exists
	binDir := filepath.Dir(absoluteBinaryPath)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Build the service
	cmd := exec.CommandContext(ctx, "go", "build", "-o", config.BinaryPath, config.SourcePath)
	cmd.Dir = bc.projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		metadata.buildSuccessful = false
		return "", fmt.Errorf("failed to build service from %s: %w\nOutput: %s", config.SourcePath, err, output)
	}

	// Update metadata after successful build
	metadata.lastBuildTime = time.Now()
	metadata.buildSuccessful = true
	metadata.sourceChecksum, _ = bc.calculateSourceChecksum(absoluteSourcePath)

	// Reset build attempts on success
	bc.buildAttempts[cacheKey] = 0

	return absoluteBinaryPath, nil
}

// needsRebuild determines if a binary needs to be rebuilt
func (bc *BuildCache) needsRebuild(metadata *buildMetadata, sourcePath, binaryPath string) (bool, string) {
	// Force rebuild if requested
	if bc.forceRebuild {
		return true, "force rebuild requested"
	}

	// Check if binary exists
	binaryInfo, err := os.Stat(binaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "binary does not exist"
		}
		// If we can't stat the binary, rebuild to be safe
		return true, fmt.Sprintf("cannot stat binary: %v", err)
	}

	// Check if previous build not valid
	if !metadata.buildSuccessful {
		return true, "previous build not valid"
	}

	// Check if source directory has been modified
	sourceModTime, err := bc.getLatestModTimeInDir(sourcePath)
	if err != nil {
		// If we can't check source mod time, rebuild to be safe
		return true, fmt.Sprintf("cannot check source modification time: %v", err)
	}

	// If source is newer than binary, rebuild
	if sourceModTime.After(binaryInfo.ModTime()) {
		return true, "source files modified"
	}

	// Check if checksum has changed (more accurate but slower)
	currentChecksum, err := bc.calculateSourceChecksum(sourcePath)
	if err == nil && metadata.sourceChecksum != "" && currentChecksum != metadata.sourceChecksum {
		return true, "source checksum changed"
	}

	return false, ""
}

// getLatestModTimeInDir recursively finds the latest modification time in a directory
func (bc *BuildCache) getLatestModTimeInDir(dirPath string) (time.Time, error) {
	var latestModTime time.Time

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		if info.ModTime().After(latestModTime) {
			latestModTime = info.ModTime()
		}

		return nil
	})

	if err != nil {
		return time.Time{}, err
	}

	return latestModTime, nil
}

// calculateSourceChecksum calculates a checksum for all .go files in a directory
func (bc *BuildCache) calculateSourceChecksum(dirPath string) (string, error) {
	hasher := sha256.New()

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		// Read file and update hash
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(hasher, file); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ClearCache clears the build cache, forcing rebuilds on next use
func (bc *BuildCache) ClearCache() {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.buildInfo = make(map[string]*buildMetadata)
	bc.buildAttempts = make(map[string]int)
	bc.forceRebuild = false
}

// GetBuildInfo returns build information for debugging
func (bc *BuildCache) GetBuildInfo() map[string]string {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	info := make(map[string]string)
	for key, metadata := range bc.buildInfo {
		status := "not built"
		if metadata.buildSuccessful {
			status = fmt.Sprintf("built at %s", metadata.lastBuildTime.Format(time.RFC3339))
		}
		info[key] = status
	}

	return info
}
