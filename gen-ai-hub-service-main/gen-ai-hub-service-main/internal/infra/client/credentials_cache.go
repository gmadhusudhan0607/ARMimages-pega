/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package client

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// Credential cache management
var (
	credentialsCache *CredentialsCache
	cacheOnce        sync.Once
)

const defaultCacheExpirationBuffer = 5 * time.Minute

// CredentialsCache provides thread-safe caching of AWS credentials
type CredentialsCache struct {
	cache            sync.Map // Thread-safe map for concurrent access
	expirationBuffer time.Duration
}

// cacheEntry represents a cached AWS credential with expiration metadata
type cacheEntry struct {
	Credentials *aws.Credentials
	ExpiresAt   time.Time
}

// getCredentialsCache returns the singleton credentials cache instance
func getCredentialsCache() *CredentialsCache {
	cacheOnce.Do(func() {
		credentialsCache = &CredentialsCache{
			expirationBuffer: defaultCacheExpirationBuffer,
		}
	})
	return credentialsCache
}

// generateCacheKey creates a unique cache key from infraModel properties and JWT token.
func generateCacheKey(jwt string, infraModel infra.ModelConfig) string {
	plainKey := fmt.Sprintf("%s:%s:%s", infraModel.OIDCRole, infraModel.Region, jwt)

	hashKey := fnv.New64a()
	hashKey.Write([]byte(plainKey))

	// Return 16-character hexadecimal string
	return fmt.Sprintf("%016x", hashKey.Sum64())
}

// Get retrieves cached credentials if they exist and are not expired
func (cc *CredentialsCache) Get(key string) (aws.Credentials, bool) {
	value, ok := cc.cache.Load(key)

	if !ok {
		return aws.Credentials{}, false
	}

	cached, ok := value.(cacheEntry)
	if !ok {
		cc.cache.Delete(key)
		return aws.Credentials{}, false
	}

	if cc.isExpired(cached) {
		cc.cache.Delete(key)
		return aws.Credentials{}, false
	}

	return *cached.Credentials, true
}

// Set stores credentials in the cache with an expiration buffer. Returns false if pre-req for caching is not met
func (cc *CredentialsCache) Set(key string, creds aws.Credentials) bool {

	// do not chache if credential is shorter than expiration buffer
	if creds.Expires.Before(time.Now().Add(cc.expirationBuffer)) {
		return false
	}

	entryExpiration := creds.Expires.Add(-cc.expirationBuffer)

	cached := cacheEntry{
		Credentials: &creds,
		ExpiresAt:   entryExpiration,
	}
	cc.cache.Store(key, cached)
	return true
}

func (cc *CredentialsCache) clear() {
	cc.cache.Clear()
}

// isExpired checks if a cache entry has expired
func (cc *CredentialsCache) isExpired(cached cacheEntry) bool {
	return time.Now().After(cached.ExpiresAt)
}
