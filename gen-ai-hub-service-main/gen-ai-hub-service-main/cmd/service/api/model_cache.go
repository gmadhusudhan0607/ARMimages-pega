/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package api

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"go.uber.org/zap"
)

// PopulateTimeout is the maximum duration the cache will wait for all provider
// fetches to complete before aborting. Exported so tests can verify behaviour.
const PopulateTimeout = 30 * time.Second

// DefaultCacheTTL is the default time-to-live for the model cache. After this
// duration the cache is considered stale and the next request will trigger
// re-population. This allows automatic recovery from transient provider
// failures (e.g. Azure auth errors) and picks up model list changes after
// deployments.
const DefaultCacheTTL = 10 * time.Minute

// cacheTTLEnvVar is the environment variable that overrides DefaultCacheTTL.
// Accepts any value parseable by time.ParseDuration (e.g. "30s", "2m", "5m").
// This is intentionally not an SCE input — it is meant for operational testing
// (e.g. memory leak detection with a short TTL) and does not require a
// deployment change to adjust.
const cacheTTLEnvVar = "MODEL_CACHE_TTL"

// CacheTTLFromEnv returns the cache TTL from the MODEL_CACHE_TTL environment
// variable if set and valid, otherwise returns DefaultCacheTTL. Invalid values
// are logged as warnings and fall back to the default.
func CacheTTLFromEnv(logger *zap.SugaredLogger) time.Duration {
	raw := os.Getenv(cacheTTLEnvVar)
	if raw == "" {
		return DefaultCacheTTL
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		logger.Warnf("Invalid %s value %q: %v — using default %s", cacheTTLEnvVar, raw, err, DefaultCacheTTL)
		return DefaultCacheTTL
	}

	if d <= 0 {
		logger.Warnf("Invalid %s value %q: must be positive — using default %s", cacheTTLEnvVar, raw, DefaultCacheTTL)
		return DefaultCacheTTL
	}

	logger.Infof("Model cache TTL set to %s via %s", d, cacheTTLEnvVar)
	return d
}

// cacheSnapshot holds an immutable point-in-time copy of the cached data.
// Stored behind an atomic.Pointer so readers never block on writers.
type cacheSnapshot struct {
	models   []ModelInfo
	warnings []string
	expiry   time.Time
}

// ModelListCache is a lazy, on-demand cache for the enriched model list shared
// by the /models and /models/defaults endpoints. The cache is populated on the
// first request that carries a valid SAX token (via the gin context), ensuring
// that provider calls (e.g. Azure APIM) can forward the caller's credentials.
//
// After the TTL expires, the next request triggers re-population and all
// concurrent readers block until fresh data is available. This ensures callers
// never receive arbitrarily stale data (the gap between expiry and the next
// request is unbounded — it could be seconds or days).
//
// Reads of non-expired data are lock-free via atomic.Pointer; only population
// is serialised by mu.
type ModelListCache struct {
	snap atomic.Pointer[cacheSnapshot]
	mu   sync.Mutex // serialises population only

	ttl     time.Duration
	checker ContextChecker
	logger  *zap.SugaredLogger
}

// NewModelListCache creates an empty cache with the given TTL. The cache will
// be populated on the first call to GetModels that supplies a request context
// with a gin context (and therefore a SAX token in the Authorization header).
// After each population the cache remains valid for the TTL duration; once
// expired the next request re-populates it synchronously (all concurrent
// readers block until fresh data is available).
func NewModelListCache(ctx context.Context, checker ContextChecker, ttl time.Duration) *ModelListCache {
	return &ModelListCache{
		ttl:     ttl,
		checker: checker,
		logger:  cntx.LoggerFromContext(ctx).Sugar(),
	}
}

// GetModels returns the cached models and warnings. On the first call the
// cache is populated synchronously using the provided context (which should
// carry the gin context so that SAX credentials are forwarded to providers).
// After the TTL expires, the next request re-populates the cache and all
// concurrent readers block until fresh data is ready — expired data is never
// served because the gap between expiry and the next request is unbounded.
//
// If the cache contains warnings (indicating a provider failure) and the
// current request carries an Authorization header, the cache is treated as
// expired so that the fresh credentials can be used to retry the failed
// provider immediately rather than waiting for the TTL to elapse.
func (c *ModelListCache) GetModels(ctx context.Context) ([]ModelInfo, []string) {
	// Fast path: lock-free read via atomic load.
	if snap := c.snap.Load(); snap != nil && time.Now().Before(snap.expiry) {
		if len(snap.warnings) == 0 || !hasAuthToken(ctx) {
			return copySnapshot(snap)
		}
		// Cache has warnings and caller has credentials that might fix
		// the failed provider — fall through to re-populate.
		c.logger.Info("Cache has warnings and request carries auth token — triggering early re-population")
	}

	// Slow path: cache is nil (cold start) or expired.
	// Block until the populating goroutine finishes — we never serve stale
	// data because we don't know how long ago the cache expired.
	c.mu.Lock()

	// Double-check: another goroutine may have populated while we waited.
	if snap := c.snap.Load(); snap != nil && time.Now().Before(snap.expiry) {
		if len(snap.warnings) == 0 || !hasAuthToken(ctx) {
			c.mu.Unlock()
			return copySnapshot(snap)
		}
	}

	populateCtx, cancel := context.WithTimeout(ctx, PopulateTimeout)
	defer cancel()

	c.populate(populateCtx)
	c.mu.Unlock()

	return copySnapshot(c.snap.Load())
}

// hasAuthToken reports whether the context carries a gin request with a
// non-empty Authorization header. This is used to decide whether an
// authenticated caller can potentially fix a provider failure recorded in
// the cache (e.g. an Azure APIM 401 caused by a missing token).
func hasAuthToken(ctx context.Context) bool {
	gc := cntx.GetGinContext(ctx)
	if gc == nil {
		return false
	}
	return gc.GetHeader("Authorization") != ""
}

// copySnapshot returns defensive copies of the snapshot's models and warnings.
func copySnapshot(snap *cacheSnapshot) ([]ModelInfo, []string) {
	models := make([]ModelInfo, len(snap.models))
	copy(models, snap.models)
	warnings := make([]string, len(snap.warnings))
	copy(warnings, snap.warnings)
	return models, warnings
}

// populate fetches models from all providers, deduplicates and enriches them,
// then atomically stores the results. Must be called while holding c.mu.
func (c *ModelListCache) populate(ctx context.Context) {
	allModels, warnings := collectModelsFromProviders(ctx, c.checker)

	merged := deduplicateModels(allModels)
	final := enrichModels(ctx, merged)

	snap := &cacheSnapshot{
		models:   final,
		warnings: warnings,
		expiry:   time.Now().Add(c.ttl),
	}
	c.snap.Store(snap)

	c.logger.Infof("Model cache populated: %d models, %d warnings (TTL %s, expires %s)",
		len(final), len(warnings), c.ttl, snap.expiry.Format(time.RFC3339))
	if len(warnings) > 0 {
		c.logger.Warnf("Model cache population warnings: %v", warnings)
	}
}
