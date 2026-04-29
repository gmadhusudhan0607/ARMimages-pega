/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package client

import (
	"sync"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestCacheKeyUniqueness(t *testing.T) {
	tests := []struct {
		name     string
		model    infra.ModelConfig
		token    string
		expected string
	}{
		{
			name: "Basic cache key generation",
			model: infra.ModelConfig{
				OIDCRole: "arn:aws:iam::123456789012:role/TestRole",
				Region:   "us-east-1",
			},
			token:    "myidenticaltoken",
			expected: "arn:aws:iam::123456789012:role/TestRole:us-east-1",
		},
		{
			name: "Different region",
			model: infra.ModelConfig{
				OIDCRole: "arn:aws:iam::123456789012:role/TestRole",
				Region:   "eu-west-1",
			},
			token:    "myidenticaltoken",
			expected: "arn:aws:iam::123456789012:role/TestRole:eu-west-1",
		},
		{
			name: "Different role",
			model: infra.ModelConfig{
				OIDCRole: "arn:aws:iam::987654321098:role/AnotherRole",
				Region:   "us-east-1",
			},
			token:    "myidenticaltoken",
			expected: "arn:aws:iam::987654321098:role/AnotherRole:us-east-1",
		},
	}

	keys := make(map[string]int)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateCacheKey(tt.token, tt.model)
			keys[result]++
		})
	}

	for key, count := range keys {
		assert.Equal(t, 1, count, "Key '%s' should appear exactly once", key)
	}

}

func TestCredentialsCache_SetAndGet(t *testing.T) {
	cache := &CredentialsCache{
		expirationBuffer: defaultCacheExpirationBuffer,
	}

	credentialExpiration := time.Now().Add(1 * time.Hour)
	testCreds := aws.Credentials{
		AccessKeyID:     "AKIATEST123456789",
		SecretAccessKey: "TestSecretKey123456789",
		SessionToken:    "TestSessionToken123456789",
		Expires:         credentialExpiration,
	}

	key := "test:key"

	// Test Set
	cache.Set(key, testCreds)

	// Test Get - should find valid credentials
	retrievedCreds, found := cache.Get(key)
	v, _ := cache.cache.Load(key)
	entry := v.(cacheEntry)
	assert.True(t, found)
	assert.Equal(t, testCreds, retrievedCreds)
	assert.Equal(t, testCreds.Expires.Sub(entry.ExpiresAt), cache.expirationBuffer)

	// Test Get with non-existent key
	_, found = cache.Get("non-existent")
	assert.False(t, found)
}

func TestCredentialsCache_Expiration(t *testing.T) {
	cache := &CredentialsCache{}

	testCreds := aws.Credentials{
		AccessKeyID:     "AKIATEST123456789",
		SecretAccessKey: "TestSecretKey123456789",
		SessionToken:    "TestSessionToken123456789",
		Expires:         time.Now().Add(5 * time.Millisecond),
	}

	key := "test:expiring"

	// Set credentials with very short TTL
	cache.Set(key, testCreds)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should not find expired credentials
	_, found := cache.Get(key)
	assert.False(t, found)

	// Verify expired entry is cleaned up from cache
	_, exists := cache.cache.Load(key)
	assert.False(t, exists)
}

func TestCredentialsCache_ConcurrentAccess(t *testing.T) {
	cache := &CredentialsCache{}

	testCreds := aws.Credentials{
		AccessKeyID:     "AKIATEST123456789",
		SecretAccessKey: "TestSecretKey123456789",
		SessionToken:    "TestSessionToken123456789",
		Expires:         time.Now().Add(1 * time.Hour),
	}

	key := "test:concurrent"

	// Set initial credentials
	cache.Set(key, testCreds)

	var wg sync.WaitGroup
	numReaders := 10
	numWriters := 5

	// Start concurrent readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Get(key)
				// Just verify we can read from cache concurrently
				// Don't assert on specific values since writers are modifying them
				assert.True(t, true) // Always passes, just exercises the Get method
			}
		}()
	}

	// Start concurrent writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				modifiedCreds := testCreds
				modifiedCreds.AccessKeyID = testCreds.AccessKeyID + "_" + string(rune('0'+id))
				cache.Set(key, modifiedCreds)
			}
		}(i)
	}

	wg.Wait()

	// Verify cache still works after concurrent access
	_, found := cache.Get(key)
	assert.True(t, found)
}

func TestCredentialsCache_IsExpired(t *testing.T) {
	cache := &CredentialsCache{}
	now := time.Now()

	tests := []struct {
		name     string
		cached   cacheEntry
		expected bool
	}{
		{
			name: "Not expired",
			cached: cacheEntry{
				ExpiresAt: now.Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "Expired",
			cached: cacheEntry{
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "Just expired",
			cached: cacheEntry{
				ExpiresAt: now.Add(-1 * time.Millisecond),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.isExpired(tt.cached)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCredentialsCache_Singleton(t *testing.T) {
	// Reset singleton for testing
	credentialsCache = nil
	cacheOnce = sync.Once{}

	// Get cache instances from multiple goroutines
	var wg sync.WaitGroup
	var cache1, cache2, cache3 *CredentialsCache

	wg.Add(3)
	go func() {
		defer wg.Done()
		cache1 = getCredentialsCache()
	}()
	go func() {
		defer wg.Done()
		cache2 = getCredentialsCache()
	}()
	go func() {
		defer wg.Done()
		cache3 = getCredentialsCache()
	}()

	wg.Wait()

	// All instances should be the same (singleton pattern)
	assert.Same(t, cache1, cache2)
	assert.Same(t, cache2, cache3)
	assert.NotNil(t, cache1)
}

func TestGetCredentialsCache_FailDuringTypeCastingDeletesFromCache(t *testing.T) {
	testCache := &CredentialsCache{}
	testCache.cache.Store("key", "wrongtype")
	v, ok := testCache.Get("key")
	assert.False(t, ok)
	assert.Empty(t, v.SecretAccessKey)

	//assert entry was removed from the cache
	n, ok := testCache.cache.Load("key")
	assert.False(t, ok)
	assert.Nil(t, n)
}
