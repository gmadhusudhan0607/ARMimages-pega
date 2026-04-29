/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package sax

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/go-sax"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	defaultMaxCacheTTL     = 15 * time.Minute
	defaultMaxCacheSize    = 10000
	defaultCleanupInterval = 5 * time.Minute
	cacheHitAttribute      = attribute.Key("cache_hit")
	cacheSizeAttribute     = attribute.Key("cache_size")
)

var cachedValidatorLogger = log.GetNamedLogger("sax-cached-validator")

// CachedClaims represents cached JWT claims with metadata
type CachedClaims struct {
	Claims    sax.Claims
	ExpiresAt time.Time
	CachedAt  time.Time
}

// CachedValidator wraps an existing validator with caching capabilities
type CachedValidator struct {
	underlying   Validator
	cache        *cache.Cache
	maxCacheTTL  time.Duration
	maxCacheSize int
	tracer       trace.Tracer
}

// NewCachedValidator creates a new cached validator wrapping the provided validator
func NewCachedValidator(underlying Validator) *CachedValidator {
	maxTTL := defaultMaxCacheTTL
	if ttlStr := helpers.GetEnvOrDefault("SAX_TOKEN_CACHE_MAX_TTL", ""); ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			maxTTL = parsed
		} else {
			cachedValidatorLogger.Warn("Invalid SAX_TOKEN_CACHE_MAX_TTL, using default",
				zap.String("value", ttlStr),
				zap.Duration("default", defaultMaxCacheTTL))
		}
	}

	maxSize := defaultMaxCacheSize
	if sizeStr := helpers.GetEnvOrDefault("SAX_TOKEN_CACHE_MAX_SIZE", ""); sizeStr != "" {
		if parsed := helpers.ParseIntOrDefault(sizeStr, defaultMaxCacheSize); parsed > 0 {
			maxSize = parsed
		}
	}

	cleanupInterval := defaultCleanupInterval
	if intervalStr := helpers.GetEnvOrDefault("SAX_TOKEN_CACHE_CLEANUP_INTERVAL", ""); intervalStr != "" {
		if parsed, err := time.ParseDuration(intervalStr); err == nil {
			cleanupInterval = parsed
		} else {
			cachedValidatorLogger.Warn("Invalid SAX_TOKEN_CACHE_CLEANUP_INTERVAL, using default",
				zap.String("value", intervalStr),
				zap.Duration("default", defaultCleanupInterval))
		}
	}

	cv := &CachedValidator{
		underlying:   underlying,
		cache:        cache.New(maxTTL, cleanupInterval),
		maxCacheTTL:  maxTTL,
		maxCacheSize: maxSize,
		tracer:       otel.Tracer(helpers.LibraryNameFromPkgPath()),
	}

	cachedValidatorLogger.Info("JWT token caching enabled",
		zap.Duration("max_ttl", maxTTL),
		zap.Int("max_size", maxSize),
		zap.Duration("cleanup_interval", cleanupInterval))

	return cv
}

// ValidateRequest implements the Validator interface with caching
func (cv *CachedValidator) ValidateRequest(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the original trace context
		originalCtx := c.Request.Context()
		spanCtx := trace.SpanContextFromContext(originalCtx)

		ctx := trace.ContextWithSpanContext(originalCtx, spanCtx)
		ctx, span := cv.tracer.Start(ctx,
			serviceName+": cached_token_validation",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithLinks(trace.Link{SpanContext: spanCtx}),
		)
		defer span.End()

		c.Set("cached_token_validation_span", span)
		c.Request = c.Request.WithContext(ctx)

		// Extract JWT token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			cachedValidatorLogger.Debug("No authorization header provided, bypassing cache")
			span.SetAttributes(cacheHitAttribute.Bool(false))
			span.SetStatus(codes.Error, "No authorization header")
			c.AbortWithStatus(401)
			return
		}

		// Extract token from "Bearer <token>" format
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			cachedValidatorLogger.Debug("Invalid authorization token format, bypassing cache")
			span.SetAttributes(cacheHitAttribute.Bool(false))
			span.SetStatus(codes.Error, "Invalid authorization header format")
			c.AbortWithStatus(401)
			return
		}

		// Generate cache key from token signature
		cacheKey := cv.generateCacheKey(token)
		if cacheKey == "" {
			cachedValidatorLogger.Debug("Invalid JWT token format, cannot generate cache key")
			span.SetAttributes(cacheHitAttribute.Bool(false))
			span.SetStatus(codes.Error, "Invalid JWT token format")
			c.AbortWithStatus(401)
			return
		}

		// Check cache first
		if cachedClaims := cv.getCachedClaims(cacheKey, scopes); cachedClaims != nil {
			// Cache hit - set claims in context and continue
			c.Set(contextKeyClaims, cachedClaims.Claims)

			// Set context values for middleware metrics collection
			c.Set("sax_cache_hit", true)
			c.Set("sax_cache_size", cv.getCacheSize())

			cachedValidatorLogger.Debug("Using cached JWT claims",
				zap.Time("expires_at", cachedClaims.ExpiresAt),
				zap.Int("current_cache_size", cv.getCacheSize()))

			span.SetAttributes(
				cacheHitAttribute.Bool(true),
				cacheSizeAttribute.Int(cv.getCacheSize()),
			)
			span.SetStatus(codes.Ok, "Validation successful (cached)")

			c.Request = c.Request.WithContext(originalCtx)
			c.Next()
			return
		}

		// Cache miss - validate with underlying validator
		span.SetAttributes(cacheHitAttribute.Bool(false))

		// Set context values for middleware metrics collection
		c.Set("sax_cache_hit", false)
		c.Set("sax_cache_size", cv.getCacheSize())

		cachedValidatorLogger.Debug("Cache miss - validating with underlying validator",
			zap.Int("current_cache_size", cv.getCacheSize()))

		// Create a custom handler that captures the validated claims
		underlyingHandler := cv.underlying.ValidateRequest(scopes...)

		// Wrap the underlying handler to capture claims for caching
		wrappedHandler := func(c *gin.Context) {
			underlyingHandler(c)

			// If validation succeeded, cache the claims
			if !c.IsAborted() {
				if claims, exists := c.Get(contextKeyClaims); exists {
					if claimsData, ok := claims.(sax.Claims); ok {
						cv.cacheClaims(cacheKey, claimsData)
						cachedValidatorLogger.Debug("JWT validation completed and cached for future use",
							zap.Strings("scopes", claimsData.Scopes))
					}
				}
			} else {
				cachedValidatorLogger.Debug("JWT validation failed, not caching")
			}
		}

		wrappedHandler(c)

		if c.IsAborted() {
			span.SetAttributes(attrValidationSuccess.Bool(false))
			if len(c.Errors) > 0 {
				span.RecordError(c.Errors.Last().Err)
			}
			return
		}

		span.SetStatus(codes.Ok, "Validation successful")
		span.SetAttributes(
			attrValidationSuccess.Bool(true),
			cacheSizeAttribute.Int(cv.getCacheSize()),
		)

		c.Request = c.Request.WithContext(originalCtx)
	}
}

// generateCacheKey creates a cache key from the JWT token signature
func (cv *CachedValidator) generateCacheKey(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}

	// Use hash of signature for cache key (more secure than raw signature)
	hasher := sha256.New()
	hasher.Write([]byte(parts[2])) // signature part
	return hex.EncodeToString(hasher.Sum(nil))
}

// getCachedClaims retrieves cached claims if valid and contains required scopes
func (cv *CachedValidator) getCachedClaims(cacheKey string, requiredScopes []string) *CachedClaims {
	item, found := cv.cache.Get(cacheKey)
	if !found {
		cachedValidatorLogger.Debug("Cache miss - token not found in cache")
		return nil
	}

	cached, ok := item.(*CachedClaims)
	if !ok {
		cachedValidatorLogger.Debug("Cache miss - invalid cache entry type")
		return nil
	}

	// Check if token has expired
	if time.Now().After(cached.ExpiresAt) {
		// Token has expired, remove it from cache and return nil
		cv.cache.Delete(cacheKey)
		cachedValidatorLogger.Debug("Cache miss - cached token expired",
			zap.Time("expired_at", cached.ExpiresAt))
		return nil
	}

	// Check if cached entry has required scopes
	if !cv.hasRequiredScopes(cached.Claims.Scopes, requiredScopes) {
		cachedValidatorLogger.Debug("Cache miss - insufficient scopes in cached token",
			zap.Strings("required_scopes", requiredScopes),
			zap.Strings("available_scopes", cached.Claims.Scopes))
		return nil
	}

	return cached
}

// cacheClaims stores validated claims in cache with appropriate TTL
func (cv *CachedValidator) cacheClaims(cacheKey string, claims sax.Claims) {

	// Check cache size limit before adding
	if cv.cache.ItemCount() >= cv.maxCacheSize {
		// Skip caching if at max size
		cachedValidatorLogger.Debug("Cache size limit reached, skipping cache storage",
			zap.Int("current_size", cv.cache.ItemCount()),
			zap.Int("max_size", cv.maxCacheSize))
		return
	}

	// Extract expiration time from claims
	var expiresAt time.Time
	if claims.Expiry != nil {
		expiresAt = claims.Expiry.Time()
	}

	// If no expiration or invalid, don't cache
	if expiresAt.IsZero() {
		cachedValidatorLogger.Debug("Skipping cache - token has no expiration")
		return
	}

	if expiresAt.Before(time.Now()) {
		cachedValidatorLogger.Debug("Skipping cache - token already expired",
			zap.Time("expired_at", expiresAt))
		return
	}

	// Calculate TTL for this specific token
	now := time.Now()
	tokenTTL := expiresAt.Sub(now)

	// Limit cache TTL to maximum allowed
	if tokenTTL > cv.maxCacheTTL {
		tokenTTL = cv.maxCacheTTL
		expiresAt = now.Add(tokenTTL)
	}

	cachedClaims := &CachedClaims{
		Claims:    claims,
		ExpiresAt: expiresAt,
		CachedAt:  now,
	}

	cv.cache.Set(cacheKey, cachedClaims, tokenTTL)

	cachedValidatorLogger.Debug("Caching validated JWT token",
		zap.Strings("scopes", claims.Scopes),
		zap.Duration("ttl", tokenTTL),
		zap.Time("expires_at", expiresAt),
		zap.Int("cache_size", cv.cache.ItemCount()))
}

// hasRequiredScopes checks if cached scopes contain all required scopes
func (cv *CachedValidator) hasRequiredScopes(cachedScopes, requiredScopes []string) bool {
	if len(requiredScopes) == 0 {
		return true
	}

	scopeSet := make(map[string]bool)
	for _, scope := range cachedScopes {
		scopeSet[scope] = true
	}

	for _, required := range requiredScopes {
		if !scopeSet[required] {
			return false
		}
	}

	return true
}

// getCacheSize returns current cache size (thread-safe)
func (cv *CachedValidator) getCacheSize() int {
	return cv.cache.ItemCount()
}
