# Request Processing Package

This package provides request processing capabilities for the Gen AI Hub Service, including token adjustment strategies for optimizing AI model requests.

## Token Adjustment Strategies

The service supports multiple strategies for adjusting the `max_tokens` parameter in AI model requests. The strategy is controlled by the `REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY` environment variable.

### Configuration Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY` | Strategy to use for token adjustment | `DISABLED` | No |
| `REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE` | Base value for token calculations | `-1` | Strategy dependent |
| `REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED` | Force adjustment when suggested value is lower than original | `false` | No |
| `REQUEST_PROCESSING_CACHE_SIZE` | Maximum cache entries for P95-P99 strategies. **Note:** This is ignored for AUTO_INCREASING which always uses cache size of 1 | `1000` | No |

## Supported Strategies

### 1. DISABLED Strategy

**Status:** ✅ Implemented

**Description:** All request processing is completely bypassed. No processor is created, no metrics are collected, and the original request is passed through unchanged with minimal overhead.

**Implementation Details:**
- Handled at the middleware level (`internal/request/middleware/request_handler.go`)
- The strategy check happens early in the request pipeline before any processing
- When detected, the entire processing pipeline is skipped for maximum efficiency
- No processor instance is created, avoiding unnecessary object allocation

**Configuration:**
```bash
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=DISABLED
```

**Behavior:**
- All processing is skipped entirely at the middleware level
- No processor creation
- No metrics collection  
- No request body processing
- Minimal overhead - acts as pure proxy
- Request body is immediately restored and passed through unchanged

### 2. MONITORING_ONLY Strategy

**Status:** ✅ Implemented

**Description:** No token adjustment is performed. The original request is passed through unchanged, metrics are collected for monitoring purposes.

**Configuration:**
```bash
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=MONITORING_ONLY
```

**Behavior:**
- Always returns `false` for `ShouldAdjust()`
- Never modifies the `max_tokens` parameter

| Original Request | Forced | Result |
|------------------|--------|--------|
| No `max_tokens` | false | No change |
| No `max_tokens` | true | No change |
| Has `max_tokens` | false | No change |
| Has `max_tokens` | true | No change |

### 3. FIXED Strategy

**Status:** ✅ Implemented

**Description:** Sets `max_tokens` to a fixed value specified by `REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE`.

**Configuration:**
```bash
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED
REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE=1000
```

**Requirements:**
- `REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE` must be > 0

**Behavior:**
The strategy respects the `REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED` flag:

| Original Request | Forced | Suggested vs Original | Action | Result |
|------------------|--------|----------------------|--------|--------|
| No `max_tokens` | false | N/A | Insert | Uses fixed value (min with model max) |
| No `max_tokens` | true | N/A | Insert | Uses fixed value (min with model max) |
| Has `max_tokens` (e.g., 1000) | false | Any | Skip | Original value preserved (1000) |
| Has `max_tokens` (e.g., 1000) | true | Suggested (500) < Original | Override | Uses fixed value (500) |
| Has `max_tokens` (e.g., 1000) | true | Suggested (1500) >= Original | Skip | Original value preserved (1000) |

**Value Calculation:**
- Uses `min(configured_value, model_maximum)` if model maximum is available
- Uses `configured_value` if no model maximum is specified

### 4. AUTO_INCREASING Strategy

**Status:** ✅ Implemented

**Description:** Dynamically adjusts `max_tokens` based on historical usage patterns stored in an in-memory cache. The strategy learns from actual token usage and adjusts future requests accordingly.

**Configuration:**
```bash
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=AUTO_INCREASING
REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE=1000
```

**Requirements:**
- `REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE` must be > 0

**Behavior:**
Similar to FIXED strategy regarding the forced flag:

| Original Request | Forced | Suggested vs Original | Action | Result |
|------------------|--------|----------------------|--------|--------|
| No `max_tokens` | false | N/A | Insert | Uses auto-adjusted value |
| No `max_tokens` | true | N/A | Insert | Uses auto-adjusted value |
| Has `max_tokens` (e.g., 1000) | false | Any | Skip | Original value preserved (1000) |
| Has `max_tokens` (e.g., 1000) | true | Suggested (800) < Original | Override | Uses auto-adjusted value (800) |
| Has `max_tokens` (e.g., 1000) | true | Suggested (1200) >= Original | Skip | Original value preserved (1000) |

**Cache Mechanism:**
- **Cache Key:** Combination of `IsolationID/Infrastructure/Provider/Creator/ModelName/ModelVersion`
- **Cache Size:** Always fixed to 1 entry per unique cache key (REQUEST_PROCESSING_CACHE_SIZE is ignored)
- **Cache Implementation:** Uses singleton `TokenCache` instance shared across all requests
- **Cache Logic:** 
  - If cached value exists: uses `max(cached_value, config_value)`
  - If no cached value: uses `config_value` as starting point
- **Cache Updates:** After successful responses, computes candidate value as `max(used_tokens, config_value)`, then updates cache **only if candidate > current_cached_value** (auto-increasing behavior)
- **Cache Behavior:** Values only increase, never decrease. FIFO eviction when new cache keys are added beyond capacity

**Value Calculation:**
1. Look up cached value for the model configuration
2. If cached value exists: `adjusted_value = max(cached_value, config_value)`
3. If no cached value: `adjusted_value = config_value`
4. Apply model limit: `final_value = min(adjusted_value, model_maximum)`

**Learning Process:**
- After each successful response, the cache is updated with the actual tokens used
- The cache stores the maximum of `actual_used_tokens` and `config_value`
- This ensures the service learns from high-usage patterns while maintaining a minimum baseline

### 5. Percentile Strategies (P95, P96, P97, P98, P99)

**Status:** ✅ Implemented

**Description:** These strategies adjust tokens based on percentile calculations of historical usage patterns. They maintain a cache of token usage samples and calculate the specified percentile to determine the optimal `max_tokens` value.

**Available Strategy Names:**
- `P95` - 95th percentile adjustment
- `P96` - 96th percentile adjustment  
- `P97` - 97th percentile adjustment
- `P98` - 98th percentile adjustment
- `P99` - 99th percentile adjustment

**Configuration:**
```bash
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=P95  # or P96, P97, P98, P99
REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE=1000
REQUEST_PROCESSING_CACHE_SIZE=1000
```

**Requirements:**
- `REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE` must be > 0

**Behavior:**
Similar to AUTO_INCREASING strategy regarding the forced flag:

| Original Request | Forced | Suggested vs Original | Action | Result |
|------------------|--------|----------------------|--------|--------|
| No `max_tokens` | false | N/A | Insert | Uses percentile value |
| No `max_tokens` | true | N/A | Insert | Uses percentile value |
| Has `max_tokens` (e.g., 1000) | false | Any | Skip | Original value preserved (1000) |
| Has `max_tokens` (e.g., 1000) | true | Suggested (850) < Original | Override | Uses percentile value (850) |
| Has `max_tokens` (e.g., 1000) | true | Suggested (1100) >= Original | Skip | Original value preserved (1000) |

**Cache Mechanism:**
- **Cache Key:** Combination of `IsolationID/Infrastructure/Provider/Creator/ModelName/ModelVersion`
- **Cache Implementation:** Uses singleton `PercentileTokenCache` instance shared across all percentile strategies (P95-P99)
- **Cache Logic:** 
  - If cached samples exist: calculates specified percentile (P95, P96, P97, P98, or P99)
  - If no cached samples: uses `config_value` as starting point
- **Cache Updates:** After successful responses, stores `max(used_tokens, config_value)`
- **Cache Capacity:** Limited by `REQUEST_PROCESSING_CACHE_SIZE` per cache key (respects this configuration, unlike AUTO_INCREASING)

**Percentile Calculation:**
1. Look up cached samples for the model configuration
2. If samples exist: calculate the specified percentile from sorted samples
3. If no samples: use `config_value`
4. Use `max(percentile_value, config_value)` to ensure minimum baseline
5. Apply model limit: `final_value = min(adjusted_value, model_maximum)`

**Learning Process:**
- After each successful response, the cache is updated with `max(actual_used_tokens, config_value)`
- The cache maintains up to `REQUEST_PROCESSING_CACHE_SIZE` samples per unique key
- Percentile calculation provides more sophisticated adjustment than a simple maximum (AUTO_INCREASING)

**Example Usage:**
```bash
# Use P95 percentile with 800 token baseline
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=P95
REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE=800
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED=false
REQUEST_PROCESSING_CACHE_SIZE=2000
```

 
## Performance Considerations

- **DISABLED:** Zero overhead - bypassed at middleware level before any processing begins
- **MONITORING_ONLY:** Minimal overhead - processes requests but never modifies them
- **FIXED:** Low overhead - simple value substitution with no cache operations
- **AUTO_INCREASING:** Low overhead with singleton in-memory cache (1 entry per model configuration)
  - Thread-safe cache operations
  - Minimal memory footprint due to fixed cache size of 1 per key
- **Percentile Strategies (P95-P99):** Moderate overhead due to percentile calculations
  - Shared singleton cache across all percentile strategies
  - Memory usage scales with `REQUEST_PROCESSING_CACHE_SIZE` configuration
  - Percentile calculation complexity is O(n log n) where n is number of cached samples
- **Cache Efficiency:**
  - Singleton pattern ensures cache is shared across all requests
  - FIFO eviction policy for cache capacity management
  - Thread-safe operations prevent race conditions

## Usage Examples

### Example 1: Fixed Token Limit
```bash
# Set fixed 2000 tokens, only for requests without max_tokens
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED
REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE=2000
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED=false
```

### Example 2: Forced Fixed Token Limit
```bash
# Override requests only when suggested value (1500) is lower than original
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=FIXED
REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE=1500
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED=true
```

**Forcing Behavior Examples:**
- Request has `max_tokens: 2000` → Updates to 1500 (suggested 1500 < 2000, so it updates)
- Request has `max_tokens: 1000` → Keeps 1000 (suggested 1500 >= 1000, so it keeps original)
- Request has `max_tokens: 1500` → Keeps 1500 (suggested equals original)
- Request has no `max_tokens` → Sets to 1500 (always inserts when missing)

### Example 3: Auto-Adjustment with Learning
```bash
# Enable auto-adjustment with 800 token baseline
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=AUTO_INCREASING
REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE=800
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED=false
REQUEST_PROCESSING_CACHE_SIZE=2000
```

### Example 4: Aggressive Auto-Adjustment
```bash
# Force auto-adjustment when suggested value is lower than original
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_STRATEGY=AUTO_INCREASING
REQUEST_PROCESSING_OUTPUT_TOKENS_BASE_VALUE=1000
REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED=true
REQUEST_PROCESSING_CACHE_SIZE=5000
```

## Architecture

The token adjustment system consists of:

- **Middleware Layer:** (`internal/request/middleware/request_handler.go`)
  - Handles DISABLED strategy check early in the pipeline
  - Manages request body processing and processor creation
  - Coordinates between metadata, processors, and response handling
- **Strategy Interface:** (`internal/request/processors/strategies/token_strategy.go`)
  - `TokenAdjustmentStrategy` defines the contract for all strategies
  - Methods: `ShouldAdjust()`, `GetAdjustedTokens()`, `PostProcessResponse()`
- **Strategy Factory:** (`internal/request/processors/strategies/factory.go`)
  - Creates appropriate strategy instances based on configuration
  - Manages singleton cache instances for AUTO_INCREASING and percentile strategies
  - Validates configuration requirements for each strategy type
- **Cache System:** Thread-safe in-memory caching
  - `TokenCache`: For AUTO_INCREASING strategy (fixed size 1 per key)
  - `PercentileTokenCache`: For P95-P99 strategies (configurable size per key)
  - Both use singleton pattern to ensure shared state across requests
- **Configuration Provider:** (`internal/request/config/config.go`)
  - Loads and validates environment-based configuration
  - Singleton pattern ensures configuration is loaded once
  - Validates strategy-specific requirements
- **Processor System:** (`internal/request/processors/`)
  - Request processors implement token adjustment logic
  - Support for different model types (chat, embedding, image)
  - Extension system for provider-specific request/response handling

## Forcing Behavior Details

The `REQUEST_PROCESSING_OUTPUT_TOKENS_ADJUSTMENT_FORCED` flag controls whether to override existing `max_tokens` values in requests:

### When FORCED=false (Default)
- **Requests without `max_tokens`:** Always insert the suggested value
- **Requests with `max_tokens`:** Always preserve the original value

### When FORCED=true
- **Requests without `max_tokens`:** Always insert the suggested value
- **Requests with `max_tokens`:** 
  - If suggested value < original value: Update to suggested value
  - If suggested value >= original value: Preserve original value

This ensures that forcing only reduces token limits when necessary, never increases them beyond what the client originally requested.
 