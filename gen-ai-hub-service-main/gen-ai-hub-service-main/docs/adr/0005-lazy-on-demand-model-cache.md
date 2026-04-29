# ADR-0005: Lazy On-Demand Model Cache with TTL-Based Expiry

- **Status**: Proposed
- **Date**: 2026-04-15
- **Deciders**: GenAI Hub Service Team
- **Related ADRs**: None

## Context

The GenAI Hub Service exposes `/models` and `/models/defaults` endpoints that return an aggregated, enriched list of all available models across three cloud providers: AWS Bedrock, GCP Vertex AI, and Azure OpenAI. Every call to these endpoints previously required real-time fetches to all three providers, deduplication, and enrichment against the local `model-metadata.yaml` ConfigMap.

### Problem Statement

How should the service cache the aggregated model list to eliminate redundant provider fetches while ensuring data freshness, correctness under concurrent access, and resilience to transient provider failures?

### Requirements

- **Freshness**: Model data must not be arbitrarily stale; a bounded TTL controls maximum staleness
- **Correctness**: Concurrent requests must never observe partially-populated or torn data
- **Resilience**: A single provider failure must not prevent serving models from the other two providers
- **Performance**: Cached reads must be lock-free; provider fetches must run in parallel
- **Credential flow**: Provider calls (especially Azure APIM) require the caller's SAX credentials from the HTTP request context
- **Zero-downtime compatibility**: Cache behavior must be safe across rolling deployments
- **Simplicity**: Prefer standard library patterns over external caching libraries

### Assumptions

- The `/models` response changes infrequently (model additions/removals happen via deployments, not at runtime)
- A 10-minute TTL is acceptable for production; shorter TTLs are useful for operational testing
- The `Authorization` header on incoming requests carries valid SAX credentials needed by Azure APIM
- All three providers are expected to be configured in production; any subset may be configured in dev/test

## Decision

**Use a lazy, on-demand model cache that populates on first request, serves lock-free reads for the TTL duration, and blocks all readers on expiry until fresh data is fetched from all providers in parallel.**

### Chosen Solution

The cache stores an immutable snapshot (the model list, provider warnings, and an absolute expiry timestamp). Reads are lock-free via an atomic reference. Writes (population) are serialized so that exactly one population occurs per expiry cycle, even under concurrent access.

#### Request Flow

1. **Fast path** (non-expired cache, no warnings): The cached snapshot is returned via an atomic read. A defensive copy is returned to prevent callers from mutating cached data. No lock is touched.

2. **Slow path** (expired cache or cold start): All concurrent readers wait while a single goroutine re-populates the cache. Population:
   - Fetches models from all configured providers in parallel
   - Merges results in deterministic order regardless of provider response timing
   - Records provider errors as warnings (never cancels the other fetches)
   - Deduplicates models by name, combining access paths
   - Enriches models against the metadata ConfigMap (lifecycle, capabilities, parameters)
   - Stores the new snapshot atomically
   
   After population completes, each waiting goroutine sees the fresh snapshot and returns it without re-populating.

3. **Warned cache with credentials**: If the cached snapshot has warnings (e.g., Azure returned 401) and the incoming request carries an `Authorization` header, the cache bypasses the TTL and re-populates eagerly. The reasoning is that the previous population failed due to missing credentials, and the current request provides credentials that may fix the issue. Requests without credentials return the warned cache as-is.

#### TTL Configuration

- Default: 10 minutes
- Configurable via `MODEL_CACHE_TTL` environment variable (Go duration string: `"30s"`, `"5m"`, `"10m"`)
- Invalid values log a warning and fall back to the default
- Intentionally an env var (not an SCE input) so it can be tuned for operational testing without requiring a deployment change

#### Population Timeout

Provider fetches are bounded by a 30-second timeout. If providers do not respond within this window, whatever results arrived are cached and timed-out providers appear as warnings.

#### Startup Behavior

The cache starts empty (no background initialization). The first HTTP request to `/models` or `/models/defaults` triggers population. Both endpoints share the same cache instance.

### Why This Solution?

1. **Lazy initialization solves the credential problem**: Provider calls need SAX credentials from the HTTP request context. A background goroutine has no request context and therefore no credentials. Populating on-demand ensures credentials are always available.

2. **Blocking on expiry prevents stale data**: The gap between TTL expiry and the next request is unbounded (seconds, hours, or days). Returning arbitrarily stale data is worse than blocking for 1-2 seconds while fresh data is fetched. Observed re-population latency in production: ~0.3 seconds additional.

3. **Lock-free reads eliminate contention**: The vast majority of requests hit a non-expired cache and never touch a lock. An atomic reference provides safe, zero-contention reads.

4. **Parallel provider fetches minimize population latency**: Population time is the maximum of the three providers (~1-2 seconds), not the sum (~3-6 seconds).

5. **Immutable snapshots eliminate data races**: Once created, a snapshot is never modified. Readers get a defensive copy; writers create a new snapshot and atomically replace the reference.

## Alternatives Considered

### Alternative 1: Background-Refresh Goroutine

**Description**: A cache with start/stop lifecycle that spawns a background goroutine to refresh models on a configurable interval, independent of incoming requests.

**Pros**:
- Pre-warmed cache: first request is always fast
- Refresh happens independently of user traffic
- Predictable, clock-driven refresh schedule

**Cons**:
- Background goroutine has no HTTP request context and therefore no SAX credentials
- Azure APIM calls fail with 401 when credentials are unavailable
- Requires lifecycle management (start/stop, graceful shutdown)
- Goroutine leak risk if stop is not called

**Rejected because**: The fundamental problem is credential availability. Azure APIM requires the caller's SAX token, which is only present in the HTTP request context. A background goroutine cannot obtain these credentials. This was the original implementation (commit `af34a97`) and was explicitly removed in commit `d2dbf07`.

### Alternative 2: Stale-While-Revalidate

**Description**: When the cache expires, the first request starts re-population asynchronously. Concurrent requests that fail to acquire the population lock return the stale cached data. On cold start (no stale data), requests block.

**Pros**:
- No request blocks on re-population (except cold start)
- Lower worst-case latency for concurrent requests during refresh
- Common pattern in CDN/HTTP caching

**Cons**:
- The gap between TTL expiry and the next request is unbounded; stale data could be hours or days old
- After a deployment that changes the model list, stale data reflects the old deployment
- More complex code: try-lock semantics, stale-data fallback, cold-start special case
- Harder to reason about correctness (two valid cache states simultaneously)

**Rejected because**: The unbounded staleness window makes this unsuitable. A model list from before a deployment (which may have added/removed models or changed metadata) should not be served to clients. The re-population latency in production (~1.3 seconds total, ~0.3 seconds above baseline) does not justify the added complexity. This was implemented in commit `c3936c8` and explicitly removed in commit `84d4e2c`.

### Alternative 3: No Cache (Direct Fetch Per Request)

**Description**: Every request to `/models` fetches from all providers, deduplicates, and enriches.

**Pros**:
- Always fresh data
- Simplest code (no cache state)

**Cons**:
- Every request incurs 1-2 seconds of provider fetch latency
- Multiplies outbound traffic to providers by request volume
- Risk of rate limiting from Azure APIM or Vertex AI
- Unnecessary load on Bedrock/Vertex/Azure for data that changes infrequently

**Rejected because**: Unacceptable latency and outbound traffic. The `/models` endpoint is called frequently by clients (including Autopilot/Assistant) and should respond in milliseconds when possible.

## Consequences

### Positive

- Lock-free reads: cached `/models` responses serve in ~0.7-1.0 seconds (dominated by network latency through port-forward, not computation)
- Parallel provider fetches: population completes in the time of the slowest provider, not the sum
- Server resource stability: goroutine count stable at 16-19 under sustained load (verified via pprof)
- Resilience: single provider failures degrade gracefully to a partial model list with warnings
- Deterministic output: model list order is reproducible regardless of provider response timing
- Configurable TTL enables operational testing without deployment changes

### Negative

- First request after startup or TTL expiry blocks for 1-2 seconds while providers are fetched
- All concurrent requests during population block (no stale fallback)
- If all three providers fail, an empty model list is cached for the full TTL duration
- The `model-metadata.yaml` enrichment filters out models without metadata, which can silently hide newly-added models if metadata is not updated

### Neutral

- The cache is per-pod (not shared across replicas); each pod independently populates its cache
- The `MODEL_CACHE_TTL` env var is set to `10m` in Helm values; changing it requires a pod restart
- Both `/models` and `/models/defaults` share one cache instance; `/models/defaults` resolves default model IDs per-request from env vars and the ops endpoint

## Implementation

### Location

The cache is implemented in `cmd/service/api/` alongside the model handlers. Both `/models` and `/models/defaults` share a single cache instance, constructed at service startup and injected into the route handlers.

### Configuration

| Setting | Default | Env Var | Description |
|---------|---------|---------|-------------|
| Cache TTL | 10 minutes | `MODEL_CACHE_TTL` | How long a snapshot is valid before re-population |
| Populate timeout | 30 seconds | — | Maximum time to wait for all providers during population |

### Testing

- Unit tests cover: defensive copy, TTL expiry, concurrent access, warned-cache eager re-population, population timeout, TTL parsing
- Live tests validate: model count consistency across TTL boundaries, server goroutine stability, response time patterns (cached vs. re-populated)
- Live test report: `docs/test-reports/2026-04-15-cacheAzure-live-test-report.md`

### Risks

**Risk**: All providers fail simultaneously and empty model list is cached for full TTL
**Mitigation**: Warnings are returned to the client; operators can monitor for warning responses. The TTL bounds the duration of the degraded state. A pod restart clears the cache.

**Risk**: `model-metadata.yaml` is corrupt or missing
**Mitigation**: The enrichment step returns an empty list if metadata is unavailable, which is cached. The pod's readiness probe remains healthy, but `/models` returns zero models. Operators can detect this via monitoring.

**Risk**: Cache TTL set too low causes excessive provider traffic
**Mitigation**: TTL parsing validates that the value is positive. The default (10 minutes) is intentionally conservative. Setting a very low TTL (e.g., `1s`) in production would increase outbound traffic but not break correctness.

## References

- [Live test report](../test-reports/2026-04-15-cacheAzure-live-test-report.md)

## Notes

### Design Evolution

The cache design evolved through three phases, each addressing limitations of the previous approach:

1. **Background-refresh goroutine** (commit `af34a97`): Failed because the background goroutine had no HTTP request context and therefore no SAX credentials for Azure APIM calls.

2. **Stale-while-revalidate** (commits `c3936c8` through `76cf231`): Addressed the credential problem by populating on-demand, but introduced complexity around stale data. The unbounded staleness window (between TTL expiry and the next request) made it possible to serve model lists from before a deployment.

3. **Blocking on expiry** (commit `84d4e2c`, current): Simplified to always block concurrent readers until fresh data is available. The observed re-population latency (~0.3 seconds above baseline) makes blocking acceptable.

Each phase was driven by concrete failures observed during development and live testing, not theoretical concerns.
