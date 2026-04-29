/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package cache

import (
	"fmt"
	"math"
	"sort"
	"sync"
)

// CacheKey represents a unique identifier for a model configuration
type CacheKey struct {
	IsolationID    string
	Infrastructure string
	Provider       string
	Creator        string
	ModelName      string
	ModelVersion   string
}

// String returns a string representation of the cache key
func (k CacheKey) String() string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s",
		k.IsolationID, k.Infrastructure, k.Provider, k.Creator, k.ModelName, k.ModelVersion)
}

// TokenCache represents an in-memory cache for storing auto-adjusted token values
// Cache size limits are enforced per isolation ID
type TokenCache struct {
	mu                     sync.RWMutex
	cache                  map[string]map[string]int // isolationID -> (key -> value)
	maxSamplesPerIsolation int
}

// NewTokenCache creates a new TokenCache with the specified maximum number of samples per isolation
func NewTokenCache(maxSamplesPerIsolation int) *TokenCache {
	if maxSamplesPerIsolation <= 0 {
		maxSamplesPerIsolation = 1000 // Default value
	}

	return &TokenCache{
		cache:                  make(map[string]map[string]int),
		maxSamplesPerIsolation: maxSamplesPerIsolation,
	}
}

// Get retrieves the cached token value for the given key
func (tc *TokenCache) Get(key CacheKey) (int, bool) {
	isolationID := key.IsolationID
	keyWithoutIsolation := fmt.Sprintf("%s/%s/%s/%s/%s",
		key.Infrastructure, key.Provider, key.Creator, key.ModelName, key.ModelVersion)

	tc.mu.RLock()
	defer tc.mu.RUnlock()

	isolationCache, exists := tc.cache[isolationID]
	if !exists {
		return 0, false
	}

	value, exists := isolationCache[keyWithoutIsolation]
	return value, exists
}

// Set stores the token value for the given key
// If the isolation cache is at capacity, it removes the oldest entry (simple FIFO)
func (tc *TokenCache) Set(key CacheKey, value int) {
	isolationID := key.IsolationID
	keyWithoutIsolation := fmt.Sprintf("%s/%s/%s/%s/%s",
		key.Infrastructure, key.Provider, key.Creator, key.ModelName, key.ModelVersion)

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Ensure isolation cache exists
	if tc.cache[isolationID] == nil {
		tc.cache[isolationID] = make(map[string]int)
	}

	isolationCache := tc.cache[isolationID]

	// If key exists, just update
	if _, exists := isolationCache[keyWithoutIsolation]; exists {
		isolationCache[keyWithoutIsolation] = value
		return
	}

	// If isolation cache is at capacity, remove one entry (FIFO)
	if len(isolationCache) >= tc.maxSamplesPerIsolation {
		for k := range isolationCache {
			delete(isolationCache, k)
			break
		}
	}

	isolationCache[keyWithoutIsolation] = value
}

// Update updates the cached value only if the new value is greater than the current value
// Returns the final value stored in cache
func (tc *TokenCache) Update(key CacheKey, newValue int) int {
	isolationID := key.IsolationID
	keyWithoutIsolation := fmt.Sprintf("%s/%s/%s/%s/%s",
		key.Infrastructure, key.Provider, key.Creator, key.ModelName, key.ModelVersion)

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Ensure isolation cache exists
	if tc.cache[isolationID] == nil {
		tc.cache[isolationID] = make(map[string]int)
	}

	isolationCache := tc.cache[isolationID]
	currentValue, exists := isolationCache[keyWithoutIsolation]

	if !exists || newValue > currentValue {
		// If isolation cache is at capacity and key doesn't exist, remove one entry
		if !exists && len(isolationCache) >= tc.maxSamplesPerIsolation {
			for k := range isolationCache {
				delete(isolationCache, k)
				break
			}
		}
		isolationCache[keyWithoutIsolation] = newValue
		return newValue
	}

	return currentValue
}

// Size returns the current total number of entries across all isolations
func (tc *TokenCache) Size() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	total := 0
	for _, isolationCache := range tc.cache {
		total += len(isolationCache)
	}
	return total
}

// Clear removes all entries from the cache
func (tc *TokenCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache = make(map[string]map[string]int)
}

// GetAll returns a copy of all cache entries (for testing/debugging)
// Returns entries in the original full key format for backward compatibility
func (tc *TokenCache) GetAll() map[string]int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	result := make(map[string]int)
	for isolationID, isolationCache := range tc.cache {
		for keyWithoutIsolation, value := range isolationCache {
			fullKey := fmt.Sprintf("%s/%s", isolationID, keyWithoutIsolation)
			result[fullKey] = value
		}
	}
	return result
}

// GetIsolationSize returns the number of entries for a specific isolation
func (tc *TokenCache) GetIsolationSize(isolationID string) int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	isolationCache, exists := tc.cache[isolationID]
	if !exists {
		return 0
	}
	return len(isolationCache)
}

// GetIsolations returns a list of isolation IDs that have entries in the cache
func (tc *TokenCache) GetIsolations() []string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	isolations := make([]string, 0, len(tc.cache))
	for isolationID := range tc.cache {
		if len(tc.cache[isolationID]) > 0 {
			isolations = append(isolations, isolationID)
		}
	}
	return isolations
}

// PercentileTokenCache represents an in-memory cache for storing multiple token samples per key
// and calculating percentiles from those samples
type PercentileTokenCache struct {
	mu         sync.RWMutex
	samples    map[string][]int // key -> slice of token samples
	maxSamples int              // maximum samples per key
}

// NewPercentileTokenCache creates a new PercentileTokenCache with the specified maximum number of samples per key
func NewPercentileTokenCache(maxSamples int) *PercentileTokenCache {
	if maxSamples <= 0 {
		maxSamples = 1000 // Default value
	}

	return &PercentileTokenCache{
		samples:    make(map[string][]int),
		maxSamples: maxSamples,
	}
}

// AddSample adds a new token sample for the given key
// If the key already has maxSamples samples, removes the oldest sample (FIFO)
func (ptc *PercentileTokenCache) AddSample(key CacheKey, tokenValue int) {
	ptc.mu.Lock()
	defer ptc.mu.Unlock()

	keyStr := key.String()
	samples := ptc.samples[keyStr]

	// Add the new sample
	samples = append(samples, tokenValue)

	// If we exceed maxSamples, remove the oldest (first) sample
	if len(samples) > ptc.maxSamples {
		samples = samples[1:]
	}

	ptc.samples[keyStr] = samples
}

// GetPercentile calculates and returns the specified percentile from the cached samples
// Returns 0 and false if no samples exist for the key
func (ptc *PercentileTokenCache) GetPercentile(key CacheKey, percentile int) (int, bool) {
	ptc.mu.RLock()
	defer ptc.mu.RUnlock()

	keyStr := key.String()
	samples := ptc.samples[keyStr]

	if len(samples) == 0 {
		return 0, false
	}

	// Create a copy and sort it
	sortedSamples := make([]int, len(samples))
	copy(sortedSamples, samples)
	sort.Ints(sortedSamples)

	// Use proper percentile calculation (nearest rank method)
	// For Pxx: position = ceil(xx/100 * N)
	position := math.Ceil(float64(percentile) / 100.0 * float64(len(sortedSamples)))
	index := int(position) - 1 // Convert to 0-based index

	// Handle edge cases
	if index <= 0 {
		return sortedSamples[0], true
	}
	if index >= len(sortedSamples) {
		return sortedSamples[len(sortedSamples)-1], true
	}

	return sortedSamples[index], true
}

// GetSampleCount returns the number of samples stored for the given key
func (ptc *PercentileTokenCache) GetSampleCount(key CacheKey) int {
	ptc.mu.RLock()
	defer ptc.mu.RUnlock()

	keyStr := key.String()
	return len(ptc.samples[keyStr])
}

// GetSamples returns a copy of all samples for the given key (for testing/debugging)
func (ptc *PercentileTokenCache) GetSamples(key CacheKey) []int {
	ptc.mu.RLock()
	defer ptc.mu.RUnlock()

	keyStr := key.String()
	samples := ptc.samples[keyStr]

	if len(samples) == 0 {
		return nil
	}

	result := make([]int, len(samples))
	copy(result, samples)
	return result
}

// Size returns the total number of unique keys in the cache
func (ptc *PercentileTokenCache) Size() int {
	ptc.mu.RLock()
	defer ptc.mu.RUnlock()
	return len(ptc.samples)
}

// Clear removes all samples from the cache
func (ptc *PercentileTokenCache) Clear() {
	ptc.mu.Lock()
	defer ptc.mu.Unlock()
	ptc.samples = make(map[string][]int)
}

// GetAllSamples returns a copy of all samples for all keys (for testing/debugging)
func (ptc *PercentileTokenCache) GetAllSamples() map[string][]int {
	ptc.mu.RLock()
	defer ptc.mu.RUnlock()

	result := make(map[string][]int, len(ptc.samples))
	for k, v := range ptc.samples {
		if len(v) > 0 {
			samples := make([]int, len(v))
			copy(samples, v)
			result[k] = samples
		}
	}
	return result
}
