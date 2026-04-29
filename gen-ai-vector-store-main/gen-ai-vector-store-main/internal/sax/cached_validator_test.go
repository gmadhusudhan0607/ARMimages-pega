/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package sax

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Pega-CloudEngineering/go-sax"
	"github.com/gin-gonic/gin"
	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockValidator struct {
	mock.Mock
}

func (m *mockValidator) ValidateRequest(scopes ...string) gin.HandlerFunc {
	args := m.Called(scopes)
	return args.Get(0).(gin.HandlerFunc)
}

func TestNewCachedValidator(t *testing.T) {
	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("SAX_TOKEN_CACHE_MAX_TTL")
		os.Unsetenv("SAX_TOKEN_CACHE_MAX_SIZE")
		os.Unsetenv("SAX_TOKEN_CACHE_CLEANUP_INTERVAL")
	}()

	t.Run("DefaultConfiguration", func(t *testing.T) {
		mockValidator := NewValidatorMock()
		cv := NewCachedValidator(mockValidator)

		assert.Equal(t, defaultMaxCacheTTL, cv.maxCacheTTL)
		assert.Equal(t, defaultMaxCacheSize, cv.maxCacheSize)
		assert.NotNil(t, cv.cache)
	})

	t.Run("CustomConfiguration", func(t *testing.T) {
		os.Setenv("SAX_TOKEN_CACHE_MAX_TTL", "10m")
		os.Setenv("SAX_TOKEN_CACHE_MAX_SIZE", "5000")
		os.Setenv("SAX_TOKEN_CACHE_CLEANUP_INTERVAL", "2m")

		mockValidator := NewValidatorMock()
		cv := NewCachedValidator(mockValidator)

		assert.Equal(t, 10*time.Minute, cv.maxCacheTTL)
		assert.Equal(t, 5000, cv.maxCacheSize)
	})

	t.Run("InvalidConfiguration", func(t *testing.T) {
		os.Setenv("SAX_TOKEN_CACHE_MAX_TTL", "invalid")
		os.Setenv("SAX_TOKEN_CACHE_MAX_SIZE", "invalid")
		os.Setenv("SAX_TOKEN_CACHE_CLEANUP_INTERVAL", "invalid")

		mockValidator := NewValidatorMock()
		cv := NewCachedValidator(mockValidator)

		// Should fall back to defaults
		assert.Equal(t, defaultMaxCacheTTL, cv.maxCacheTTL)
		assert.Equal(t, defaultMaxCacheSize, cv.maxCacheSize)
	})
}

func TestCachedValidator_GenerateCacheKey(t *testing.T) {
	mockValidator := NewValidatorMock()
	cv := NewCachedValidator(mockValidator)

	t.Run("ValidJWT", func(t *testing.T) {
		token := "header.payload.signature"
		key := cv.generateCacheKey(token)
		assert.NotEmpty(t, key)
		assert.Len(t, key, 64)
	})

	t.Run("InvalidJWT", func(t *testing.T) {
		token := "invalid.token"
		key := cv.generateCacheKey(token)
		assert.Empty(t, key)
	})

	t.Run("ConsistentKeys", func(t *testing.T) {
		token := "header.payload.signature"
		key1 := cv.generateCacheKey(token)
		key2 := cv.generateCacheKey(token)
		assert.Equal(t, key1, key2)
	})
}

func TestCachedValidator_HasRequiredScopes(t *testing.T) {
	mockValidator := NewValidatorMock()
	cv := NewCachedValidator(mockValidator)

	t.Run("NoRequiredScopes", func(t *testing.T) {
		cached := []string{"read", "write"}
		required := []string{}
		assert.True(t, cv.hasRequiredScopes(cached, required))
	})

	t.Run("HasAllRequiredScopes", func(t *testing.T) {
		cached := []string{"read", "write", "admin"}
		required := []string{"read", "write"}
		assert.True(t, cv.hasRequiredScopes(cached, required))
	})

	t.Run("MissingRequiredScopes", func(t *testing.T) {
		cached := []string{"read"}
		required := []string{"read", "write"}
		assert.False(t, cv.hasRequiredScopes(cached, required))
	})
}

func TestCachedValidator_CacheClaims(t *testing.T) {
	mockValidator := NewValidatorMock()
	cv := NewCachedValidator(mockValidator)

	t.Run("ValidClaims", func(t *testing.T) {
		claims := createTestClaims("user123", time.Now().Add(time.Hour))
		claims.Scopes = []string{"read"}
		cacheKey := "testkey"

		cv.cacheClaims(cacheKey, claims)

		item, found := cv.cache.Get(cacheKey)
		assert.True(t, found)

		cached, ok := item.(*CachedClaims)
		assert.True(t, ok)
		assert.Equal(t, claims, cached.Claims)
		assert.True(t, cached.ExpiresAt.After(time.Now()))
	})

	t.Run("ExpiredClaims", func(t *testing.T) {
		claims := createTestClaims("user123", time.Now().Add(-time.Hour))
		cacheKey := "expiredkey"

		cv.cacheClaims(cacheKey, claims)

		_, found := cv.cache.Get(cacheKey)
		assert.False(t, found) // Should not cache expired tokens
	})

	t.Run("NoExpirationClaims", func(t *testing.T) {
		claims := sax.Claims{
			Claims: jwt.Claims{
				Subject: "user123",
			},
		}
		cacheKey := "noexpkey"

		cv.cacheClaims(cacheKey, claims)

		_, found := cv.cache.Get(cacheKey)
		assert.False(t, found) // Should not cache tokens without expiration
	})

	t.Run("TTLLimitedByMaxTTL", func(t *testing.T) {
		// Set a very long expiration
		longExpiry := time.Now().Add(24 * time.Hour)
		claims := createTestClaims("user123", longExpiry)
		cacheKey := "longtokenkey"

		cv.cacheClaims(cacheKey, claims)

		item, found := cv.cache.Get(cacheKey)
		assert.True(t, found)

		cached, ok := item.(*CachedClaims)
		assert.True(t, ok)
		// Should be limited by maxCacheTTL (15 minutes by default)
		maxAllowedExpiry := time.Now().Add(cv.maxCacheTTL)
		assert.True(t, cached.ExpiresAt.Before(maxAllowedExpiry.Add(time.Second))) // Allow 1 second tolerance
	})
}

func TestCachedValidator_GetCachedClaims(t *testing.T) {
	mockValidator := NewValidatorMock()
	cv := NewCachedValidator(mockValidator)

	t.Run("CacheHit", func(t *testing.T) {
		cacheKey := "hitkey"
		claims := createTestClaims("user123", time.Now().Add(time.Hour))
		claims.Scopes = []string{"read"}

		cv.cacheClaims(cacheKey, claims)
		cached := cv.getCachedClaims(cacheKey, []string{"read"})

		assert.NotNil(t, cached)
		assert.Equal(t, claims, cached.Claims)
	})

	t.Run("CacheMiss", func(t *testing.T) {
		cached := cv.getCachedClaims("nonexistentkey", []string{"read"})
		assert.Nil(t, cached)
	})

	t.Run("ExpiredEntry", func(t *testing.T) {
		cacheKey := "expiredkey"
		expiredClaims := createTestClaims("user123", time.Now().Add(-time.Hour))
		expiredClaims.Scopes = []string{"read"}
		// Manually add expired entry
		cv.cache.Set(cacheKey, &CachedClaims{
			Claims:    expiredClaims,
			ExpiresAt: time.Now().Add(-time.Hour),
			CachedAt:  time.Now().Add(-time.Hour),
		}, time.Hour)

		cached := cv.getCachedClaims(cacheKey, []string{"read"})
		assert.Nil(t, cached) // Should return nil for expired entry and remove it from cache

		// Verify it was removed from cache
		_, found := cv.cache.Get(cacheKey)
		assert.False(t, found)
	})

	t.Run("InsufficientScopes", func(t *testing.T) {
		cacheKey := "scopekey"
		claims := createTestClaims("user123", time.Now().Add(time.Hour))
		claims.Scopes = []string{"read"}
		requiredScopes := []string{"read", "write"}

		cv.cacheClaims(cacheKey, claims)
		cached := cv.getCachedClaims(cacheKey, requiredScopes)

		assert.Nil(t, cached) // Should return nil when scopes don't match
	})
}

func TestCachedValidator_ValidateRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CacheHit", func(t *testing.T) {
		mockValidator := NewValidatorMock()
		cv := NewCachedValidator(mockValidator)

		// Pre-populate cache
		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjk5OTk5OTk5OTl9.signature"
		cacheKey := cv.generateCacheKey(token)
		claims := createTestClaims("1234567890", time.Now().Add(time.Hour))
		claims.Scopes = []string{"read"}
		cv.cacheClaims(cacheKey, claims)

		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		// Create handler
		handler := cv.ValidateRequest("read")

		// Set up next handler to verify claims were set
		// Execute handler
		handler(c)

		// Verify claims were set in context
		claimsValue, exists := c.Get(contextKeyClaims)
		assert.True(t, exists)
		assert.Equal(t, claims, claimsValue)
		assert.False(t, c.IsAborted())
	})

	t.Run("CacheMiss", func(t *testing.T) {
		mockValidator := &mockValidator{}
		cv := NewCachedValidator(mockValidator)

		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJleHAiOjk5OTk5OTk5OTl9.signature"

		// Mock underlying validator to succeed
		mockValidator.On("ValidateRequest", []string{"read"}).Return(gin.HandlerFunc(func(c *gin.Context) {
			// Simulate successful validation by setting claims
			claims := sax.Claims{
				Claims: jwt.Claims{
					Subject: "user123",
					Expiry:  jwt.NewNumericDate(time.Now().Add(time.Hour)),
				},
				Scopes: []string{"read"},
			}
			c.Set(contextKeyClaims, claims)
			c.Next()
		}))

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		handler := cv.ValidateRequest("read")

		handler(c)
		assert.False(t, c.IsAborted())

		// Verify token was cached after successful validation
		cacheKey := cv.generateCacheKey(token)
		cached := cv.getCachedClaims(cacheKey, []string{"read"})
		assert.NotNil(t, cached)

		mockValidator.AssertExpectations(t)
	})

	t.Run("NoAuthorizationHeader", func(t *testing.T) {
		mockValidator := &mockValidator{}
		cv := NewCachedValidator(mockValidator)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handler := cv.ValidateRequest("read")
		handler(c)

		assert.True(t, c.IsAborted())
	})

	t.Run("InvalidAuthorizationHeader", func(t *testing.T) {
		mockValidator := &mockValidator{}
		cv := NewCachedValidator(mockValidator)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("Authorization", "InvalidFormat")

		handler := cv.ValidateRequest("read")
		handler(c)

		assert.True(t, c.IsAborted())
	})
}

// TestCachedValidator_AutomaticCleanup tests that go-cache automatically cleans up expired entries
func TestCachedValidator_AutomaticCleanup(t *testing.T) {
	mockValidator := NewValidatorMock()

	// Set a very short cleanup interval for testing
	os.Setenv("SAX_TOKEN_CACHE_CLEANUP_INTERVAL", "50ms")
	defer os.Unsetenv("SAX_TOKEN_CACHE_CLEANUP_INTERVAL")

	cv := NewCachedValidator(mockValidator)

	// Add entries with very short TTL
	expiredClaims := createTestClaims("user1", time.Now().Add(10*time.Millisecond))
	expiredClaims.Scopes = []string{"read"}
	cv.cache.Set("expired1", &CachedClaims{
		Claims:    expiredClaims,
		ExpiresAt: time.Now().Add(10 * time.Millisecond),
		CachedAt:  time.Now(),
	}, 10*time.Millisecond)

	validClaims := createTestClaims("user3", time.Now().Add(time.Hour))
	validClaims.Scopes = []string{"read"}
	cv.cache.Set("valid", &CachedClaims{
		Claims:    validClaims,
		ExpiresAt: time.Now().Add(time.Hour),
		CachedAt:  time.Now(),
	}, time.Hour)

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Check results - expired entry should be gone
	_, expired1Exists := cv.cache.Get("expired1")
	_, validExists := cv.cache.Get("valid")

	assert.False(t, expired1Exists)
	assert.True(t, validExists)
}

func TestCachedValidator_BackgroundCleanup(t *testing.T) {
	// This test verifies that background cleanup runs periodically with go-cache
	mockValidator := NewValidatorMock()

	// Set a very short cleanup interval for testing
	os.Setenv("SAX_TOKEN_CACHE_CLEANUP_INTERVAL", "50ms")
	defer os.Unsetenv("SAX_TOKEN_CACHE_CLEANUP_INTERVAL")

	cv := NewCachedValidator(mockValidator)

	// Add an entry with very short TTL
	expiredClaims := createTestClaims("user1", time.Now().Add(10*time.Millisecond))
	expiredClaims.Scopes = []string{"read"}
	cv.cache.Set("expired", &CachedClaims{
		Claims:    expiredClaims,
		ExpiresAt: time.Now().Add(10 * time.Millisecond),
		CachedAt:  time.Now(),
	}, 10*time.Millisecond)

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Check that expired entry was removed by go-cache
	_, exists := cv.cache.Get("expired")
	assert.False(t, exists)
}

func TestCachedValidator_CacheSizeLimit(t *testing.T) {
	mockValidator := &mockValidator{}

	// Set a small cache size for testing
	os.Setenv("SAX_TOKEN_CACHE_MAX_SIZE", "3")
	defer os.Unsetenv("SAX_TOKEN_CACHE_MAX_SIZE")

	cv := NewCachedValidator(mockValidator)

	// Add entries up to the limit
	for i := 0; i < 5; i++ {
		claims := createTestClaims(fmt.Sprintf("user%d", i), time.Now().Add(time.Hour))
		claims.Scopes = []string{"read"}
		cv.cacheClaims(fmt.Sprintf("key%d", i), claims)
	}

	// Cache should not exceed the limit
	assert.True(t, cv.getCacheSize() <= 3)
}

// Helper function to create sax.Claims for testing
func createTestClaims(subject string, expiry time.Time) sax.Claims {
	expiryNumeric := jwt.NewNumericDate(expiry)
	return sax.Claims{
		Claims: jwt.Claims{
			Subject: subject,
			Expiry:  expiryNumeric,
		},
	}
}
