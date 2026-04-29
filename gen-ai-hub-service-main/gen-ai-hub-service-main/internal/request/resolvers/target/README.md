# Target Resolver Package

This package provides the `TargetResolver` component that analyzes incoming HTTP requests and resolves comprehensive target routing information including targetURL, targetType, infrastructure, provider, creator, model metadata, etc.

## Overview

The TargetResolver integrates with three configuration sources:
1. **CONFIGURATION_FILE** (static YAML) - Azure OpenAI, GCP Vertex, Buddies, AWS Bedrock (legacy)
2. **MAPPING_ENDPOINT** (dynamic HTTP) - AWS Bedrock auto-mapping
3. **MODELS_DEFAULTS_ENDPOINT** (HTTP) - Default fast/smart models

## Architecture

The resolver uses a pipeline pattern with enrichment stages:

```
TargetResolver
├── Configuration Loaders
│   ├── StaticMappingLoader (CONFIGURATION_FILE)
│   ├── MappingEndpointClient (MAPPING_ENDPOINT)
│   └── DefaultsEndpointClient (MODELS_DEFAULTS_ENDPOINT)
├── Enrichment Pipeline
│   ├── extractBasicInfo
│   ├── determineTargetType
│   ├── fetchModelConfiguration
│   ├── enrichWithInfrastructure
│   ├── enrichWithModelMetadata
│   └── constructTargetURL
└── Cache Layer (optional optimization)
```

## Core Types

### TargetType
Represents the type of target endpoint:
- `TargetTypeLLM` - LLM model endpoints (Azure OpenAI, AWS Bedrock, GCP Vertex)
- `TargetTypeBuddy` - Buddy endpoints
- `TargetTypeOther` - Other types of endpoints
- `TargetTypeNone` - Local endpoints (health, swagger, models, etc.)

### ResolvedTarget
Primary output structure containing:
- **Required**: `targetURL`, `targetType`
- **Optional**: `infrastructure`, `provider`, `creator`, `modelName`, `modelVersion`, `modelID`

### ResolutionRequest
Internal working context that accumulates information through pipeline stages:
- `GinContext` - The Gin HTTP context
- `Target` - The ResolvedTarget being built
- `Metadata` - Stage-specific data

## Files

### types.go
Core type definitions including `TargetType`, `ResolvedTarget`, `ResolutionRequest`, `EnrichmentStage`, and `ResolutionError`.

### target_resolver.go
Main resolver implementation with:
- `NewTargetResolver()` - Constructor that initializes all configuration sources
- `Resolve()` - Main entry point for request resolution
- Pipeline stage execution logic

### clients.go
HTTP clients for dynamic configuration:

#### MappingClient
Handles communication with the MAPPING_ENDPOINT for fetching dynamic AWS Bedrock model configurations.

**Features:**
- Thread-safe caching with `sync.RWMutex`
- 5-minute cache TTL (Time-To-Live)
- 10-second HTTP timeout
- Context-aware requests
- Automatic cache invalidation after expiry

**Usage:**
```go
client := NewMappingClient("https://mapping-endpoint.example.com")
models, err := client.GetModels(ctx)
if err != nil {
    // Handle error
}
// models is []infra.ModelConfig
```

**Caching Behavior:**
- First call fetches from endpoint and caches result
- Subsequent calls within 5 minutes return cached data (cache hit)
- After 5 minutes, next call fetches fresh data and updates cache
- Cache is thread-safe for concurrent access

#### DefaultsClient
Handles communication with the MODELS_DEFAULTS_ENDPOINT for fetching default fast/smart model configurations.

**Features:**
- 10-second HTTP timeout
- Context-aware requests
- Proper error handling with status code checks
- Returns structured default model configuration

**Usage:**
```go
client := NewDefaultsClient("https://defaults-endpoint.example.com")
defaults, err := client.GetDefaults(ctx)
if err != nil {
    // Handle error
}
// Access defaults.Fast and defaults.Smart
```

**Response Structure:**
The `DefaultModelConfig` contains:
- `Fast` - Default fast model configuration
- `Smart` - Default smart model configuration

Each `ModelDefault` includes:
- `ModelID` - The model identifier
- `Provider` - The provider name
- `Creator` - The model creator/vendor

**Note:** Environment variable overrides (`SMART_MODEL_OVERRIDE`, `FAST_MODEL_OVERRIDE`) are handled by the calling code, not within the client itself. This keeps the client focused on HTTP communication.

### loaders.go
Configuration loading utilities:
- `loadStaticMapping()` - Loads YAML configuration file
- `findModelInMapping()` - Helper to find models
- `findBuddyInMapping()` - Helper to find buddies

### private_models.go
Private model support for Azure OpenAI:

**Features:**
- Loads private Azure OpenAI model configurations from mounted directory
- Filters YAML files with prefix "private-model-"
- Returns only active models
- Thread-safe loading on-demand (no caching)
- Graceful error handling (skips invalid files)

**Directory Structure:**
```
/private-model-config/
├── private-model-customer1.yaml
├── private-model-customer2.yaml
└── private-model-internal.yaml
```

**YAML Format:**
```yaml
models:
  - name: "my-private-gpt-4o"
    modelId: "gpt-4o-2024-11-20"
    redirectUrl: "https://private.openai.azure.com"
    infrastructure: "azure"
    provider: "Azure"
    creator: "openai"
    active: true
    # ... other model properties
```

**Functions:**
- `loadPrivateModels(ctx, privateModelDir)` - Loads all active private models from directory
- `checkPrivateModels(ctx, modelName)` - Searches for a specific private model by name
  - Returns: `(model, found, error)`
  - `found=true` only if model exists and is active

**Resolution Priority:**
For Azure OpenAI requests, the resolver checks private models **first** before falling back to the static mapping. This allows customers to override standard models with their private deployments.

**Integration:**
Private models are integrated into the `fetchFromStaticMapping` stage:
```go
if infra == "azure" {
    privateModel, found, err := r.checkPrivateModels(ctx, modelName)
    if err == nil && found && privateModel != nil {
        req.Metadata["modelConfig"] = privateModel
        req.Metadata["isPrivateModel"] = true
        return nil
    }
}
// Fall back to static mapping...
```

## Usage Example

```go
import (
    "context"
    "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/request/resolvers/target"
)

// Initialize resolver
resolver, err := target.NewTargetResolver(
    "/path/to/config.yaml",
    "https://mapping-endpoint.example.com",
    "https://defaults-endpoint.example.com",
)
if err != nil {
    // Handle error
}

// Resolve target for a request
target, err := resolver.Resolve(ctx, ginContext)
if err != nil {
    // Handle resolution error
}

// Use resolved target
fmt.Printf("Target URL: %s\n", target.TargetURL)
fmt.Printf("Target Type: %s\n", target.TargetType)
fmt.Printf("Infrastructure: %s\n", target.Infrastructure)
```

## Implementation Status

### Phase 1: Core Infrastructure (COMPLETE)
- ✅ Type definitions (types.go)
- ✅ Main resolver structure (target_resolver.go)
- ✅ HTTP clients (clients.go)
- ✅ Configuration loaders (loaders.go)

### Phase 2: Enrichment Pipeline Stages (COMPLETE)
- ✅ extractBasicInfo
- ✅ determineTargetType
- ✅ fetchModelConfiguration
- ✅ enrichWithInfrastructure
- ✅ enrichWithModelMetadata
- ✅ constructTargetURL

### Phase 3: HTTP Clients Implementation (COMPLETE)
- ✅ MappingClient with caching (5-minute TTL, thread-safe)
- ✅ DefaultsClient with proper error handling
- ✅ Context-aware HTTP requests
- ✅ Timeout configuration (10 seconds)

### Phase 4: Private Models Support (COMPLETE) ✅
- ✅ Private model resolution for Azure OpenAI
- ✅ Directory-based configuration loading (`/private-model-config`)
- ✅ Prefix-based file filtering ("private-model-")
- ✅ Active-only model filtering
- ✅ Priority resolution (private models checked first)
- ✅ Graceful error handling for invalid files
- ✅ Integration with fetchFromStaticMapping stage
- ✅ privateModelDir parameter in NewTargetResolver constructor

**Implementation Details:**
- Private models are loaded on-demand from the configured directory
- Only files with prefix "private-model-" are processed
- YAML files are parsed and only active models are returned
- For Azure OpenAI routes, private models are checked before static mapping
- If a private model is found, it's used with `isPrivateModel=true` metadata

### Phase 5: Testing (COMPLETE) ✅
- ✅ Unit tests for each enrichment stage
- ✅ Unit tests for HTTP clients (MappingClient, DefaultsClient)
- ✅ Unit tests for helper functions (extractVersion, extractCreatorFromModelId)
- ✅ Integration tests for full pipeline (Azure OpenAI, Buddies, Local endpoints)
- ✅ Error handling tests (model not found, buddy not found, etc.)
- ✅ Edge case tests (empty paths, complex query parameters, etc.)
- ✅ Private model tests (loading, checking, priority)
- ✅ Loader tests (static mapping, file not found, empty path)
- ✅ Performance benchmarks (Resolve, extractBasicInfo, extractVersion, cache performance)

**Test File:** `internal/request/resolvers/target_resolver_test.go`

**Test Coverage:**
- **Stage Tests**: Each of the 6 enrichment stages has dedicated unit tests
- **Client Tests**: HTTP clients tested with mock servers and error scenarios
- **Integration Tests**: Full pipeline tests for different route types
- **Benchmark Tests**: Performance benchmarks to ensure < 1ms resolution time
- **Total Tests**: 40+ test functions covering all major functionality

**Running Tests:**
```bash
# Run all tests
go test ./internal/request/resolvers/...

# Run with verbose output
go test -v ./internal/request/resolvers/...

# Run benchmarks
go test -bench=. ./internal/request/resolvers/...

# Run specific test
go test -run TestTargetResolver_FullPipeline_AzureOpenAI ./internal/request/resolvers/...
```

### Phase 6: Integration (TODO)
- [ ] Integrate with RequestModificationMiddleware
- [ ] Feature flag support
- [ ] Monitoring and metrics

## Performance Targets

- Resolution time: < 1ms per request (average)
- Memory allocation: < 5KB per resolution
- Cache hit ratio: > 90% for MAPPING_ENDPOINT

## Error Handling

The resolver uses `ResolutionError` to provide detailed error information:
- `Stage` - Which pipeline stage failed
- `Reason` - Why it failed
- `Details` - Additional context

## Dependencies

- `github.com/gin-gonic/gin` - HTTP framework
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/Pega-CloudEngineering/gen-ai-hub-service/cmd/service/api` - API types
- `github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/infra` - Infrastructure models

## Future Enhancements

1. Request modification based on resolved target
2. Intelligent routing with load balancing
3. Circuit breakers for unhealthy targets
4. Telemetry and metrics tracking
5. Redis-backed configuration cache
6. Real-time configuration updates
