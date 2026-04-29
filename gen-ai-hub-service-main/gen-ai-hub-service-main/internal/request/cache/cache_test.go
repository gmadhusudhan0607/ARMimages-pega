/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package cache

import (
	"fmt"
	"sync"
	"testing"
)

func TestCacheKey_String(t *testing.T) {
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	expected := "iso1/infra1/openai/creator1/gpt-3.5-turbo/v1"
	if key.String() != expected {
		t.Errorf("Expected %s, got %s", expected, key.String())
	}
}

func TestNewTokenCache(t *testing.T) {
	// Test with positive maxSamplesPerIsolation
	cache := NewTokenCache(100)
	if cache.maxSamplesPerIsolation != 100 {
		t.Errorf("Expected maxSamplesPerIsolation to be 100, got %d", cache.maxSamplesPerIsolation)
	}

	// Test with zero maxSamplesPerIsolation (should use default)
	cache = NewTokenCache(0)
	if cache.maxSamplesPerIsolation != 1000 {
		t.Errorf("Expected maxSamplesPerIsolation to be 1000 (default), got %d", cache.maxSamplesPerIsolation)
	}

	// Test with negative maxSamplesPerIsolation (should use default)
	cache = NewTokenCache(-5)
	if cache.maxSamplesPerIsolation != 1000 {
		t.Errorf("Expected maxSamplesPerIsolation to be 1000 (default), got %d", cache.maxSamplesPerIsolation)
	}
}

func TestTokenCache_GetSet(t *testing.T) {
	cache := NewTokenCache(10)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Test Get on empty cache
	value, exists := cache.Get(key)
	if exists {
		t.Error("Expected key to not exist in empty cache")
	}
	if value != 0 {
		t.Errorf("Expected value to be 0, got %d", value)
	}

	// Test Set and Get
	cache.Set(key, 1024)
	value, exists = cache.Get(key)
	if !exists {
		t.Error("Expected key to exist after Set")
	}
	if value != 1024 {
		t.Errorf("Expected value to be 1024, got %d", value)
	}

	// Test overwriting existing key
	cache.Set(key, 2048)
	value, exists = cache.Get(key)
	if !exists {
		t.Error("Expected key to exist after overwrite")
	}
	if value != 2048 {
		t.Errorf("Expected value to be 2048, got %d", value)
	}
}

func TestTokenCache_Update(t *testing.T) {
	cache := NewTokenCache(10)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Test Update on non-existing key
	result := cache.Update(key, 1024)
	if result != 1024 {
		t.Errorf("Expected result to be 1024, got %d", result)
	}

	value, exists := cache.Get(key)
	if !exists {
		t.Error("Expected key to exist after Update")
	}
	if value != 1024 {
		t.Errorf("Expected value to be 1024, got %d", value)
	}

	// Test Update with higher value
	result = cache.Update(key, 2048)
	if result != 2048 {
		t.Errorf("Expected result to be 2048, got %d", result)
	}

	value, _ = cache.Get(key)
	if value != 2048 {
		t.Errorf("Expected value to be 2048, got %d", value)
	}

	// Test Update with lower value (should not update)
	result = cache.Update(key, 1024)
	if result != 2048 {
		t.Errorf("Expected result to be 2048 (unchanged), got %d", result)
	}

	value, _ = cache.Get(key)
	if value != 2048 {
		t.Errorf("Expected value to remain 2048, got %d", value)
	}
}

func TestTokenCache_MaxSamplesPerIsolation(t *testing.T) {
	cache := NewTokenCache(2) // Very small cache per isolation

	// Test with keys from the same isolation (should enforce per-isolation limit)
	key1_iso1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key2_iso1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model2", ModelVersion: "v1"}
	key3_iso1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model3", ModelVersion: "v1"}

	// Add two entries to same isolation
	cache.Set(key1_iso1, 1024)
	cache.Set(key2_iso1, 2048)

	if cache.GetIsolationSize("iso1") != 2 {
		t.Errorf("Expected iso1 cache size to be 2, got %d", cache.GetIsolationSize("iso1"))
	}

	// Add third entry to same isolation (should evict one from iso1)
	cache.Set(key3_iso1, 3072)

	if cache.GetIsolationSize("iso1") != 2 {
		t.Errorf("Expected iso1 cache size to remain 2, got %d", cache.GetIsolationSize("iso1"))
	}

	// Verify key3_iso1 exists
	value, exists := cache.Get(key3_iso1)
	if !exists {
		t.Error("Expected key3_iso1 to exist")
	}
	if value != 3072 {
		t.Errorf("Expected value to be 3072, got %d", value)
	}
}

func TestTokenCache_Size(t *testing.T) {
	cache := NewTokenCache(10)

	if cache.Size() != 0 {
		t.Errorf("Expected empty cache size to be 0, got %d", cache.Size())
	}

	key1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key2 := CacheKey{IsolationID: "iso2", Infrastructure: "infra2", Provider: "openai", Creator: "creator2", ModelName: "model2", ModelVersion: "v1"}

	cache.Set(key1, 1024)
	if cache.Size() != 1 {
		t.Errorf("Expected cache size to be 1, got %d", cache.Size())
	}

	cache.Set(key2, 2048)
	if cache.Size() != 2 {
		t.Errorf("Expected cache size to be 2, got %d", cache.Size())
	}
}

func TestTokenCache_Clear(t *testing.T) {
	cache := NewTokenCache(10)
	key := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}

	cache.Set(key, 1024)
	if cache.Size() != 1 {
		t.Errorf("Expected cache size to be 1, got %d", cache.Size())
	}

	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Expected cache size to be 0 after Clear, got %d", cache.Size())
	}

	_, exists := cache.Get(key)
	if exists {
		t.Error("Expected key to not exist after Clear")
	}
}

func TestTokenCache_GetAll(t *testing.T) {
	cache := NewTokenCache(10)
	key1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key2 := CacheKey{IsolationID: "iso2", Infrastructure: "infra2", Provider: "openai", Creator: "creator2", ModelName: "model2", ModelVersion: "v1"}

	cache.Set(key1, 1024)
	cache.Set(key2, 2048)

	all := cache.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected GetAll to return 2 entries, got %d", len(all))
	}

	if all[key1.String()] != 1024 {
		t.Errorf("Expected key1 value to be 1024, got %d", all[key1.String()])
	}

	if all[key2.String()] != 2048 {
		t.Errorf("Expected key2 value to be 2048, got %d", all[key2.String()])
	}
}

func TestTokenCache_PerIsolationLimits(t *testing.T) {
	cache := NewTokenCache(2) // Very small cache per isolation

	// Create keys from different isolations
	key1_iso1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key2_iso1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model2", ModelVersion: "v1"}
	key3_iso1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model3", ModelVersion: "v1"}

	key1_iso2 := CacheKey{IsolationID: "iso2", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key2_iso2 := CacheKey{IsolationID: "iso2", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model2", ModelVersion: "v1"}
	key3_iso2 := CacheKey{IsolationID: "iso2", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model3", ModelVersion: "v1"}

	// Add 2 entries to iso1 (should fit within limit)
	cache.Set(key1_iso1, 1000)
	cache.Set(key2_iso1, 2000)

	// Add 2 entries to iso2 (should fit within limit)
	cache.Set(key1_iso2, 3000)
	cache.Set(key2_iso2, 4000)

	// Verify both isolations have 2 entries each
	if cache.GetIsolationSize("iso1") != 2 {
		t.Errorf("Expected iso1 to have 2 entries, got %d", cache.GetIsolationSize("iso1"))
	}
	if cache.GetIsolationSize("iso2") != 2 {
		t.Errorf("Expected iso2 to have 2 entries, got %d", cache.GetIsolationSize("iso2"))
	}
	if cache.Size() != 4 {
		t.Errorf("Expected total cache size to be 4, got %d", cache.Size())
	}

	// Add third entry to iso1 (should evict one from iso1 only)
	cache.Set(key3_iso1, 5000)

	// Verify iso1 still has 2 entries (one was evicted)
	if cache.GetIsolationSize("iso1") != 2 {
		t.Errorf("Expected iso1 to still have 2 entries after eviction, got %d", cache.GetIsolationSize("iso1"))
	}
	// Verify iso2 is unaffected
	if cache.GetIsolationSize("iso2") != 2 {
		t.Errorf("Expected iso2 to still have 2 entries (unaffected), got %d", cache.GetIsolationSize("iso2"))
	}

	// Verify key3_iso1 exists
	value, exists := cache.Get(key3_iso1)
	if !exists {
		t.Error("Expected key3_iso1 to exist after being added")
	}
	if value != 5000 {
		t.Errorf("Expected key3_iso1 value to be 5000, got %d", value)
	}

	// Verify iso2 entries are still intact
	value, exists = cache.Get(key1_iso2)
	if !exists {
		t.Error("Expected key1_iso2 to still exist (iso2 should be unaffected)")
	}
	if value != 3000 {
		t.Errorf("Expected key1_iso2 value to be 3000, got %d", value)
	}

	value, exists = cache.Get(key2_iso2)
	if !exists {
		t.Error("Expected key2_iso2 to still exist (iso2 should be unaffected)")
	}
	if value != 4000 {
		t.Errorf("Expected key2_iso2 value to be 4000, got %d", value)
	}

	// Add third entry to iso2 (should evict one from iso2 only)
	cache.Set(key3_iso2, 6000)

	// Verify both isolations still have 2 entries each
	if cache.GetIsolationSize("iso1") != 2 {
		t.Errorf("Expected iso1 to still have 2 entries, got %d", cache.GetIsolationSize("iso1"))
	}
	if cache.GetIsolationSize("iso2") != 2 {
		t.Errorf("Expected iso2 to have 2 entries after eviction, got %d", cache.GetIsolationSize("iso2"))
	}
	if cache.Size() != 4 {
		t.Errorf("Expected total cache size to be 4, got %d", cache.Size())
	}
}

func TestTokenCache_GetIsolations(t *testing.T) {
	cache := NewTokenCache(10)

	// Initially, no isolations
	isolations := cache.GetIsolations()
	if len(isolations) != 0 {
		t.Errorf("Expected 0 isolations initially, got %d", len(isolations))
	}

	// Add entries for different isolations
	key1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key2 := CacheKey{IsolationID: "iso2", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key3 := CacheKey{IsolationID: "iso3", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}

	cache.Set(key1, 1000)
	cache.Set(key2, 2000)
	cache.Set(key3, 3000)

	isolations = cache.GetIsolations()
	if len(isolations) != 3 {
		t.Errorf("Expected 3 isolations, got %d", len(isolations))
	}

	// Check that all expected isolations are present
	isolationMap := make(map[string]bool)
	for _, iso := range isolations {
		isolationMap[iso] = true
	}

	if !isolationMap["iso1"] || !isolationMap["iso2"] || !isolationMap["iso3"] {
		t.Errorf("Expected isolations [iso1, iso2, iso3], got %v", isolations)
	}
}

func TestTokenCache_ConcurrentAccess(t *testing.T) {
	cache := NewTokenCache(100)
	key := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Test concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Update(key, id*numOperations+j)
			}
		}(i)
	}

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.Get(key)
			}
		}()
	}

	wg.Wait()

	// Verify cache is still functional
	value, exists := cache.Get(key)
	if !exists {
		t.Error("Expected key to exist after concurrent operations")
	}
	if value < 0 {
		t.Errorf("Expected value to be non-negative, got %d", value)
	}
}

// PercentileTokenCache Tests

func TestNewPercentileTokenCache(t *testing.T) {
	// Test with positive maxSamples
	cache := NewPercentileTokenCache(100)
	if cache.maxSamples != 100 {
		t.Errorf("Expected maxSamples to be 100, got %d", cache.maxSamples)
	}

	// Test with zero maxSamples (should use default)
	cache = NewPercentileTokenCache(0)
	if cache.maxSamples != 1000 {
		t.Errorf("Expected maxSamples to be 1000 (default), got %d", cache.maxSamples)
	}

	// Test with negative maxSamples (should use default)
	cache = NewPercentileTokenCache(-5)
	if cache.maxSamples != 1000 {
		t.Errorf("Expected maxSamples to be 1000 (default), got %d", cache.maxSamples)
	}
}

func TestPercentileTokenCache_AddSample(t *testing.T) {
	cache := NewPercentileTokenCache(10)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Test adding samples
	cache.AddSample(key, 100)
	cache.AddSample(key, 200)
	cache.AddSample(key, 300)

	if cache.GetSampleCount(key) != 3 {
		t.Errorf("Expected sample count to be 3, got %d", cache.GetSampleCount(key))
	}

	samples := cache.GetSamples(key)
	expected := []int{100, 200, 300}
	if len(samples) != len(expected) {
		t.Errorf("Expected %d samples, got %d", len(expected), len(samples))
	}
	for i, v := range expected {
		if samples[i] != v {
			t.Errorf("Expected sample[%d] to be %d, got %d", i, v, samples[i])
		}
	}
}

func TestPercentileTokenCache_MaxSamples(t *testing.T) {
	cache := NewPercentileTokenCache(3) // Very small cache for testing
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Add samples up to max
	cache.AddSample(key, 100)
	cache.AddSample(key, 200)
	cache.AddSample(key, 300)

	if cache.GetSampleCount(key) != 3 {
		t.Errorf("Expected sample count to be 3, got %d", cache.GetSampleCount(key))
	}

	// Add one more sample (should evict oldest)
	cache.AddSample(key, 400)

	if cache.GetSampleCount(key) != 3 {
		t.Errorf("Expected sample count to remain 3, got %d", cache.GetSampleCount(key))
	}

	samples := cache.GetSamples(key)
	expected := []int{200, 300, 400} // 100 should be evicted
	for i, v := range expected {
		if samples[i] != v {
			t.Errorf("Expected sample[%d] to be %d, got %d", i, v, samples[i])
		}
	}
}

func TestPercentileTokenCache_GetPercentile_NoSamples(t *testing.T) {
	cache := NewPercentileTokenCache(10)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Test with no samples
	value, exists := cache.GetPercentile(key, 95)
	if exists {
		t.Error("Expected GetPercentile to return false for key with no samples")
	}
	if value != 0 {
		t.Errorf("Expected value to be 0, got %d", value)
	}
}

func TestPercentileTokenCache_GetPercentile_SingleSample(t *testing.T) {
	cache := NewPercentileTokenCache(10)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	cache.AddSample(key, 100)

	// With single sample, all percentiles should return that sample
	for _, percentile := range []int{50, 95, 99} {
		value, exists := cache.GetPercentile(key, percentile)
		if !exists {
			t.Errorf("Expected GetPercentile(%d) to return true", percentile)
		}
		if value != 100 {
			t.Errorf("Expected P%d to be 100, got %d", percentile, value)
		}
	}
}

func TestPercentileTokenCache_GetPercentile_TwoSamples(t *testing.T) {
	cache := NewPercentileTokenCache(10)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	cache.AddSample(key, 100)
	cache.AddSample(key, 200)

	// P50 should be 100 (ceil(50/100 * 2) = ceil(1.0) = 1, index 0)
	value, exists := cache.GetPercentile(key, 50)
	if !exists {
		t.Error("Expected GetPercentile(50) to return true")
	}
	if value != 100 {
		t.Errorf("Expected P50 to be 100, got %d", value)
	}

	// P95 should be 200 (ceil(95/100 * 2) = ceil(1.9) = 2, index 1)
	value, exists = cache.GetPercentile(key, 95)
	if !exists {
		t.Error("Expected GetPercentile(95) to return true")
	}
	if value != 200 {
		t.Errorf("Expected P95 to be 200, got %d", value)
	}
}

func TestPercentileTokenCache_GetPercentile_TenSamples(t *testing.T) {
	cache := NewPercentileTokenCache(20)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Add samples: 10, 20, 30, 40, 50, 60, 70, 80, 90, 100
	for i := 1; i <= 10; i++ {
		cache.AddSample(key, i*10)
	}

	// P50 should be 50 (ceil(50/100 * 10) = ceil(5.0) = 5, index 4, value 50)
	value, exists := cache.GetPercentile(key, 50)
	if !exists {
		t.Error("Expected GetPercentile(50) to return true")
	}
	if value != 50 {
		t.Errorf("Expected P50 to be 50, got %d", value)
	}

	// P95 should be 100 (ceil(95/100 * 10) = ceil(9.5) = 10, index 9, value 100)
	value, exists = cache.GetPercentile(key, 95)
	if !exists {
		t.Error("Expected GetPercentile(95) to return true")
	}
	if value != 100 {
		t.Errorf("Expected P95 to be 100, got %d", value)
	}

	// P90 should be 90 (ceil(90/100 * 10) = ceil(9.0) = 9, index 8, value 90)
	value, exists = cache.GetPercentile(key, 90)
	if !exists {
		t.Error("Expected GetPercentile(90) to return true")
	}
	if value != 90 {
		t.Errorf("Expected P90 to be 90, got %d", value)
	}
}

func TestPercentileTokenCache_GetPercentile_TwentySamples(t *testing.T) {
	cache := NewPercentileTokenCache(30)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Add samples: 10, 20, 30, ..., 200
	for i := 1; i <= 20; i++ {
		cache.AddSample(key, i*10)
	}

	// P95 should be 190 (ceil(95/100 * 20) = ceil(19.0) = 19, index 18, value 190)
	value, exists := cache.GetPercentile(key, 95)
	if !exists {
		t.Error("Expected GetPercentile(95) to return true")
	}
	if value != 190 {
		t.Errorf("Expected P95 to be 190, got %d", value)
	}

	// P99 should be 200 (ceil(99/100 * 20) = ceil(19.8) = 20, index 19, value 200)
	value, exists = cache.GetPercentile(key, 99)
	if !exists {
		t.Error("Expected GetPercentile(99) to return true")
	}
	if value != 200 {
		t.Errorf("Expected P99 to be 200, got %d", value)
	}
}

func TestPercentileTokenCache_GetPercentile_UnsortedSamples(t *testing.T) {
	cache := NewPercentileTokenCache(10)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Add samples in random order
	samples := []int{50, 10, 90, 30, 70, 20, 80, 40, 60, 100}
	for _, sample := range samples {
		cache.AddSample(key, sample)
	}

	// P95 should be 100 (ceil(95/100 * 10) = ceil(9.5) = 10, index 9, value 100)
	value, exists := cache.GetPercentile(key, 95)
	if !exists {
		t.Error("Expected GetPercentile(95) to return true")
	}
	if value != 100 {
		t.Errorf("Expected P95 to be 100, got %d", value)
	}

	// P50 should be 50 (ceil(50/100 * 10) = ceil(5.0) = 5, index 4, value 50)
	value, exists = cache.GetPercentile(key, 50)
	if !exists {
		t.Error("Expected GetPercentile(50) to return true")
	}
	if value != 50 {
		t.Errorf("Expected P50 to be 50, got %d", value)
	}
}

func TestPercentileTokenCache_GetPercentile_VerifyPercentileProperty(t *testing.T) {
	cache := NewPercentileTokenCache(100)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	// Add 100 samples: 1, 2, 3, ..., 100
	for i := 1; i <= 100; i++ {
		cache.AddSample(key, i)
	}

	// Test that P95 is >= 95% of all samples
	p95Value, exists := cache.GetPercentile(key, 95)
	if !exists {
		t.Error("Expected GetPercentile(95) to return true")
	}

	samples := cache.GetSamples(key)
	countLessOrEqual := 0
	for _, sample := range samples {
		if sample <= p95Value {
			countLessOrEqual++
		}
	}

	percentageAtOrBelow := float64(countLessOrEqual) / float64(len(samples)) * 100
	if percentageAtOrBelow < 95.0 {
		t.Errorf("P95 value %d should be >= 95%% of samples, but only %.1f%% are <= P95", p95Value, percentageAtOrBelow)
	}

	// Test that P99 is >= 99% of all samples
	p99Value, exists := cache.GetPercentile(key, 99)
	if !exists {
		t.Error("Expected GetPercentile(99) to return true")
	}

	countLessOrEqual = 0
	for _, sample := range samples {
		if sample <= p99Value {
			countLessOrEqual++
		}
	}

	percentageAtOrBelow = float64(countLessOrEqual) / float64(len(samples)) * 100
	if percentageAtOrBelow < 99.0 {
		t.Errorf("P99 value %d should be >= 99%% of samples, but only %.1f%% are <= P99", p99Value, percentageAtOrBelow)
	}
}

func TestPercentileTokenCache_MultipleKeys(t *testing.T) {
	cache := NewPercentileTokenCache(10)
	key1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key2 := CacheKey{IsolationID: "iso2", Infrastructure: "infra2", Provider: "openai", Creator: "creator2", ModelName: "model2", ModelVersion: "v1"}

	// Add different samples for each key
	cache.AddSample(key1, 100)
	cache.AddSample(key1, 200)
	cache.AddSample(key2, 300)
	cache.AddSample(key2, 400)

	if cache.Size() != 2 {
		t.Errorf("Expected cache size to be 2, got %d", cache.Size())
	}

	// Verify samples are isolated per key
	samples1 := cache.GetSamples(key1)
	samples2 := cache.GetSamples(key2)

	if len(samples1) != 2 || samples1[0] != 100 || samples1[1] != 200 {
		t.Errorf("Expected key1 samples to be [100, 200], got %v", samples1)
	}

	if len(samples2) != 2 || samples2[0] != 300 || samples2[1] != 400 {
		t.Errorf("Expected key2 samples to be [300, 400], got %v", samples2)
	}
}

func TestPercentileTokenCache_Clear(t *testing.T) {
	cache := NewPercentileTokenCache(10)
	key := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	cache.AddSample(key, 100)
	cache.AddSample(key, 200)

	if cache.Size() != 1 {
		t.Errorf("Expected cache size to be 1, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size to be 0 after Clear, got %d", cache.Size())
	}

	_, exists := cache.GetPercentile(key, 95)
	if exists {
		t.Error("Expected GetPercentile to return false after Clear")
	}
}

func TestPercentileTokenCache_GetAllSamples(t *testing.T) {
	cache := NewPercentileTokenCache(10)
	key1 := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}
	key2 := CacheKey{IsolationID: "iso2", Infrastructure: "infra2", Provider: "openai", Creator: "creator2", ModelName: "model2", ModelVersion: "v1"}

	cache.AddSample(key1, 100)
	cache.AddSample(key1, 200)
	cache.AddSample(key2, 300)

	allSamples := cache.GetAllSamples()
	if len(allSamples) != 2 {
		t.Errorf("Expected GetAllSamples to return 2 keys, got %d", len(allSamples))
	}

	samples1 := allSamples[key1.String()]
	samples2 := allSamples[key2.String()]

	if len(samples1) != 2 || samples1[0] != 100 || samples1[1] != 200 {
		t.Errorf("Expected key1 samples to be [100, 200], got %v", samples1)
	}

	if len(samples2) != 1 || samples2[0] != 300 {
		t.Errorf("Expected key2 samples to be [300], got %v", samples2)
	}
}

func TestPercentileTokenCache_EmptyCache(t *testing.T) {
	cache := NewPercentileTokenCache(10)

	// Test various cache keys that don't exist
	testKeys := []CacheKey{
		{
			IsolationID:    "non-existent-1",
			Infrastructure: "test-infra",
			Provider:       "test-provider",
			Creator:        "test-creator",
			ModelName:      "test-model",
			ModelVersion:   "v1",
		},
		{
			IsolationID:    "non-existent-2",
			Infrastructure: "different-infra",
			Provider:       "different-provider",
			Creator:        "different-creator",
			ModelName:      "different-model",
			ModelVersion:   "v2",
		},
	}

	for i, key := range testKeys {
		t.Run(fmt.Sprintf("empty_cache_key_%d", i+1), func(t *testing.T) {
			// Test GetPercentile returns (0, false) for non-existent keys
			value, exists := cache.GetPercentile(key, 95)
			if exists {
				t.Errorf("Expected GetPercentile to return false for non-existent key, got true")
			}
			if value != 0 {
				t.Errorf("Expected GetPercentile to return 0 for non-existent key, got %d", value)
			}

			// Test GetSampleCount returns 0 for non-existent keys
			count := cache.GetSampleCount(key)
			if count != 0 {
				t.Errorf("Expected GetSampleCount to return 0 for non-existent key, got %d", count)
			}

			// Test GetSamples returns empty slice for non-existent keys
			samples := cache.GetSamples(key)
			if len(samples) != 0 {
				t.Errorf("Expected GetSamples to return empty slice for non-existent key, got %v", samples)
			}
		})
	}
}

func TestCacheKey_StringGeneration(t *testing.T) {
	tests := []struct {
		name        string
		key         CacheKey
		expected    string
		description string
	}{
		{
			name: "basic key",
			key: CacheKey{
				IsolationID:    "iso1",
				Infrastructure: "infra1",
				Provider:       "openai",
				Creator:        "creator1",
				ModelName:      "gpt-3.5-turbo",
				ModelVersion:   "v1",
			},
			expected:    "iso1/infra1/openai/creator1/gpt-3.5-turbo/v1",
			description: "Basic cache key string generation",
		},
		{
			name: "key with special characters",
			key: CacheKey{
				IsolationID:    "iso-test_123",
				Infrastructure: "infra.test",
				Provider:       "provider-name",
				Creator:        "creator_test",
				ModelName:      "model.name-v2",
				ModelVersion:   "v1.0.1",
			},
			expected:    "iso-test_123/infra.test/provider-name/creator_test/model.name-v2/v1.0.1",
			description: "Cache key with special characters",
		},
		{
			name: "key with empty fields",
			key: CacheKey{
				IsolationID:    "",
				Infrastructure: "infra1",
				Provider:       "",
				Creator:        "creator1",
				ModelName:      "model1",
				ModelVersion:   "",
			},
			expected:    "/infra1//creator1/model1/",
			description: "Cache key with empty fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.key.String()
			if result != tt.expected {
				t.Errorf("%s: Expected %s, got %s", tt.description, tt.expected, result)
			}
		})
	}
}

func TestCacheKey_Uniqueness(t *testing.T) {
	// Test that different combinations produce different cache key strings
	baseKey := CacheKey{
		IsolationID:    "iso1",
		Infrastructure: "infra1",
		Provider:       "openai",
		Creator:        "creator1",
		ModelName:      "gpt-3.5-turbo",
		ModelVersion:   "v1",
	}

	variations := []struct {
		name string
		key  CacheKey
	}{
		{
			name: "different_isolation_id",
			key: CacheKey{
				IsolationID:    "iso2",
				Infrastructure: baseKey.Infrastructure,
				Provider:       baseKey.Provider,
				Creator:        baseKey.Creator,
				ModelName:      baseKey.ModelName,
				ModelVersion:   baseKey.ModelVersion,
			},
		},
		{
			name: "different_infrastructure",
			key: CacheKey{
				IsolationID:    baseKey.IsolationID,
				Infrastructure: "infra2",
				Provider:       baseKey.Provider,
				Creator:        baseKey.Creator,
				ModelName:      baseKey.ModelName,
				ModelVersion:   baseKey.ModelVersion,
			},
		},
		{
			name: "different_provider",
			key: CacheKey{
				IsolationID:    baseKey.IsolationID,
				Infrastructure: baseKey.Infrastructure,
				Provider:       "anthropic",
				Creator:        baseKey.Creator,
				ModelName:      baseKey.ModelName,
				ModelVersion:   baseKey.ModelVersion,
			},
		},
		{
			name: "different_creator",
			key: CacheKey{
				IsolationID:    baseKey.IsolationID,
				Infrastructure: baseKey.Infrastructure,
				Provider:       baseKey.Provider,
				Creator:        "creator2",
				ModelName:      baseKey.ModelName,
				ModelVersion:   baseKey.ModelVersion,
			},
		},
		{
			name: "different_model_name",
			key: CacheKey{
				IsolationID:    baseKey.IsolationID,
				Infrastructure: baseKey.Infrastructure,
				Provider:       baseKey.Provider,
				Creator:        baseKey.Creator,
				ModelName:      "gpt-4",
				ModelVersion:   baseKey.ModelVersion,
			},
		},
		{
			name: "different_model_version",
			key: CacheKey{
				IsolationID:    baseKey.IsolationID,
				Infrastructure: baseKey.Infrastructure,
				Provider:       baseKey.Provider,
				Creator:        baseKey.Creator,
				ModelName:      baseKey.ModelName,
				ModelVersion:   "v2",
			},
		},
	}

	baseKeyString := baseKey.String()

	for _, variation := range variations {
		t.Run(variation.name, func(t *testing.T) {
			variationString := variation.key.String()
			if variationString == baseKeyString {
				t.Errorf("Expected different cache key strings, but got same: %s", variationString)
			}
		})
	}
}

func TestPercentileTokenCache_CacheIsolation(t *testing.T) {
	cache := NewPercentileTokenCache(20)

	// Create keys that differ only in one field
	key1 := CacheKey{
		IsolationID:    "isolation-1",
		Infrastructure: "test-infra",
		Provider:       "test-provider",
		Creator:        "test-creator",
		ModelName:      "test-model",
		ModelVersion:   "v1",
	}

	key2 := CacheKey{
		IsolationID:    "isolation-2", // Only difference
		Infrastructure: "test-infra",
		Provider:       "test-provider",
		Creator:        "test-creator",
		ModelName:      "test-model",
		ModelVersion:   "v1",
	}

	// Add samples to first key
	cache.AddSample(key1, 100)
	cache.AddSample(key1, 200)
	cache.AddSample(key1, 300)

	// Verify first key has samples
	if cache.GetSampleCount(key1) != 3 {
		t.Errorf("Expected key1 to have 3 samples, got %d", cache.GetSampleCount(key1))
	}

	// Verify second key is completely isolated (empty)
	if cache.GetSampleCount(key2) != 0 {
		t.Errorf("Expected key2 to have 0 samples (isolated), got %d", cache.GetSampleCount(key2))
	}

	// Test percentile calculation isolation
	p95_key1, exists1 := cache.GetPercentile(key1, 95)
	if !exists1 {
		t.Error("Expected key1 to have percentile data")
	}
	if p95_key1 != 300 { // P95 of [100, 200, 300] should be 300
		t.Errorf("Expected key1 P95 to be 300, got %d", p95_key1)
	}

	p95_key2, exists2 := cache.GetPercentile(key2, 95)
	if exists2 {
		t.Error("Expected key2 to have no percentile data (isolated)")
	}
	if p95_key2 != 0 {
		t.Errorf("Expected key2 P95 to be 0 (empty), got %d", p95_key2)
	}

	// Add samples to second key and verify isolation is maintained
	cache.AddSample(key2, 1000)
	cache.AddSample(key2, 2000)

	// Verify both keys maintain separate data
	if cache.GetSampleCount(key1) != 3 {
		t.Errorf("Expected key1 to still have 3 samples, got %d", cache.GetSampleCount(key1))
	}
	if cache.GetSampleCount(key2) != 2 {
		t.Errorf("Expected key2 to have 2 samples, got %d", cache.GetSampleCount(key2))
	}

	// Verify percentile calculations are separate
	p95_key1_after, _ := cache.GetPercentile(key1, 95)
	p95_key2_after, _ := cache.GetPercentile(key2, 95)

	if p95_key1_after != 300 {
		t.Errorf("Expected key1 P95 to remain 300, got %d", p95_key1_after)
	}
	if p95_key2_after != 2000 { // P95 of [1000, 2000] should be 2000
		t.Errorf("Expected key2 P95 to be 2000, got %d", p95_key2_after)
	}
}

func TestPercentileTokenCache_ConcurrentAccess(t *testing.T) {
	cache := NewPercentileTokenCache(100)
	key := CacheKey{IsolationID: "iso1", Infrastructure: "infra1", Provider: "openai", Creator: "creator1", ModelName: "model1", ModelVersion: "v1"}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	// Test concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.AddSample(key, id*numOperations+j)
			}
		}(i)
	}

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cache.GetPercentile(key, 95)
			}
		}()
	}

	wg.Wait()

	// Verify cache is still functional
	value, exists := cache.GetPercentile(key, 95)
	if !exists {
		t.Error("Expected GetPercentile to return true after concurrent operations")
	}
	if value < 0 {
		t.Errorf("Expected percentile value to be non-negative, got %d", value)
	}

	if cache.GetSampleCount(key) <= 0 {
		t.Error("Expected sample count to be positive after concurrent operations")
	}
}
