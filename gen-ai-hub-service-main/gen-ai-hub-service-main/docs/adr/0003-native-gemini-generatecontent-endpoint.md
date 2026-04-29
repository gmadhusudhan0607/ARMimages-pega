# ADR-0003: Use Native Gemini /generateContent Endpoint for Image Generation

- **Status**: Accepted
- **Date**: 2026-03-20
- **Deciders**: GenAI Hub Service Team
- **Related ADRs**: None

## Context

The GenAI Hub Service needs to support Google's Gemini 3.1 Flash image generation models (gemini-3.1-flash-image-preview) on GCP Vertex AI.

### Problem Statement

How should we expose Gemini's image generation capability through the gateway while maintaining API simplicity and avoiding unnecessary complexity?

### Requirements

- **Functional**: Support Gemini native image generation API
- **Simplicity**: Minimize infrastructure-level API transformations
- **Consistency**: Follow existing patterns for provider-native APIs
- **Maintainability**: Keep code simple and avoid complex request/response mapping
- **Performance**: Minimize latency and processing overhead

### Constraints

- **Gemini models do not support Imagen API**: Gemini image generation requires the native `/generateContent` endpoint
- **OpenAI API in Vertex AI is chat-only**: The `/chat/completions` endpoint does not support image generation
- **Different response formats**: OpenAI (data[].b64_json) vs Gemini (candidates[].content.parts[].inlineData)
- **Different request structures**: OpenAI (simple prompt string) vs Gemini (structured contents array)

## Decision

**Add native `/generateContent` endpoint for Gemini image generation models instead of attempting API transformation.**

The service exposes Gemini's native API format at:
```
POST /google/deployments/{modelId}/generateContent
```

### Implementation

```go
// cmd/service/main.go
google.POST("/deployments/:modelId/generateContent",
    api.HandleExperimentalModelChatCompletionRequest(ctx, mapping))
```

**Model Configuration**:
```yaml
# distribution/genai-hub-service-helm/.../models/gemini-3-1-flash-image-preview.yaml
- name: gemini-3.1-flash-image-preview
  targetAPI: "/generateContent"
  path: "/google/deployments/gemini-3.1-flash-image-preview/generateContent"
  capabilities:
    completions: false
    embeddings: false
    image: true
```

**Request Format** (Native Gemini):
```json
{
  "contents": [
    {
      "role": "user",
      "parts": [
        {"text": "Generate an image of a sunset over mountains"}
      ]
    }
  ],
  "generationConfig": {
    "temperature": 1.0,
    "topP": 0.95,
    "responseModalities": ["IMAGE"]
  }
}
```

**Response Format** (Native Gemini):
```json
{
  "candidates": [
    {
      "content": {
        "parts": [
          {
            "inlineData": {
              "mimeType": "image/jpeg",
              "data": "<base64-encoded-image>"
            }
          }
        ]
      }
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 10,
    "candidatesTokenCount": 0,
    "totalTokenCount": 10
  }
}
```

### Why This Solution?

1. **Functional requirement**: Gemini models only work with their native API
2. **Simplicity**: Direct passthrough, no complex transformation logic
3. **Consistency**: Matches pattern used for OpenAI image generation (`/images/generations`)
4. **Maintainability**: Gateway acts as router, not translator
5. **Performance**: Zero overhead from request/response transformation

## Alternatives Considered

### Alternative 1: API Transformation Layer (OpenAI → Gemini)

**Description**: Accept OpenAI-format requests at `/images/generations`, transform to Gemini format at infrastructure level.

**Pros**:
- Single client-facing API format
- Clients don't need to learn Gemini API
- Potential for API abstraction

**Cons**:
- **Complex transformation logic**: Must map OpenAI prompt → Gemini contents structure
- **Response mapping complexity**: Must map Gemini candidates → OpenAI data array
- **Infrastructure-level transformation**: Violates gateway's "router not translator" principle
- **Maintenance burden**: Must keep transformation logic synchronized with API changes
- **Performance overhead**: Additional processing latency
- **Error handling complexity**: Must translate errors between formats
- **Limited feature support**: May not support all Gemini-specific features

**Rejected because**: Adds unnecessary complexity at the infrastructure level. The gateway's role is to route requests, not translate between APIs. Transformation logic is better handled by client SDKs or application code.

### Alternative 2: Use Imagen API Endpoints

**Description**: Route Gemini models through Vertex AI's Imagen-compatible endpoints.

**Pros**:
- Potentially simpler integration
- Could reuse Imagen patterns

**Cons**:
- **Does not work**: Gemini models do not support Imagen API format
- **Technical limitation**: Vertex AI documentation explicitly states Gemini uses `/generateContent`
- **Forced solution**: Would require workarounds for fundamental incompatibility

**Rejected because**: Not technically feasible. Gemini models require native `/generateContent` endpoint.

### Alternative 3: Native /generateContent Endpoint (CHOSEN)

**Description**: Expose Gemini's native `/generateContent` API through the gateway.

**Pros**:
- **Works out of the box**: No transformation needed
- **Simple implementation**: Direct request passthrough
- **Full feature support**: All Gemini capabilities available
- **Low maintenance**: No API mapping to maintain
- **Clear separation**: Different providers expose different APIs
- **Performance**: No transformation overhead

**Cons**:
- Clients must use Gemini-specific format for Gemini models
- Multiple API formats to document (OpenAI, Gemini, Imagen)

**Chosen because**: Simplicity, maintainability, and alignment with gateway's routing responsibility.

## Consequences

### Positive

- ✅ **Functional**: Gemini image generation works correctly
- ✅ **Simple code**: ~5 LoC to add route, reuses existing handler
- ✅ **No transformation logic**: Gateway remains a router, not a translator
- ✅ **Full feature support**: All Gemini capabilities accessible
- ✅ **Low maintenance**: No API mapping logic to maintain
- ✅ **Performance**: Zero transformation overhead
- ✅ **Extensible**: Easy to add more Gemini models

### Negative

- ❌ **Multiple API formats**: Clients must understand different provider formats
- ❌ **Documentation burden**: Must document Gemini API separately from OpenAI
- ❌ **Client complexity**: Applications need provider-specific code paths

### Neutral

- Different providers expose different APIs (OpenAI, Gemini, Imagen)
- Gateway acts as router, application layer handles format differences
- Clients can use provider SDKs (Google Vertex AI SDK, OpenAI SDK) directly

## Implementation

### Completed

1. ✅ Add `/generateContent` HTTP route (cmd/service/main.go:260)
2. ✅ Add model specifications with `image` capability
3. ✅ Add Helm configuration for 3 model variants
4. ✅ Add integration tests for endpoint routing
5. ✅ Update OpenAPI specification with endpoint documentation
6. ✅ Add URLPathGenerateContent test constant

### Files Modified

- `cmd/service/main.go` - Added POST route for /generateContent
- `distribution/genai-hub-service-helm/.../models/gemini-3-1-flash-image-preview.yaml` - Model config
- `internal/models/specs/gcp/vertex/google/gemini-3.1-flash-image-preview.yaml` - Model spec
- `test/integration/service/mappings_test.go` - Integration tests
- `test/integration/service/vars_test.go` - Test constants
- `test/integration/request-processing/mapping_20090.yaml` - Test mappings
- `apidocs/spec.yaml` - OpenAPI documentation

### Testing

- 183/183 integration tests passing
- Live test infrastructure supports image generation
- Make target: `make test-live-image`

## References

- [Vertex AI GenerateContent API](https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/inference)
- [Gemini Image Generation Guide](https://cloud.google.com/vertex-ai/docs/generative-ai/image/overview)
- [Implementation PR](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/445)
- OpenAI Image Generation API (for comparison)
- Imagen API Documentation (for comparison)

## Notes

This decision establishes the pattern for provider-native APIs in the gateway: when provider APIs are fundamentally incompatible (different request/response structures, different capabilities), expose them natively rather than forcing transformation.

The gateway's responsibility is **routing** (directing requests to the right backend), not **translation** (converting between API formats). Translation logic belongs in client SDKs or application code where it can be tested, versioned, and evolved independently.

### Future Direction: API Detection Over Model Detection

For Vertex AI endpoints, the gateway should continue migrating toward **API detection** rather than **model detection** for routing decisions.

**Current approach**: Route based on model name patterns (e.g., "gemini-*" → /generateContent, "imagen-*" → /predict)

**Target approach**: Route based on the API endpoint being called, with model configuration specifying the `targetAPI`:

```yaml
# Model specifies which API it supports
- name: gemini-3.1-flash-image-preview
  targetAPI: "/generateContent"  # Model declares its API
  path: "/google/deployments/gemini-3.1-flash-image-preview/generateContent"
```

**Benefits**:
- Explicit model-to-API mapping in configuration
- No inference from model name patterns
- Easier to add new models without code changes
- Clear separation: model config declares API, router uses config
- Supports cases where same API serves multiple model families

**Migration path**: Continue expanding `targetAPI` configuration field to all Vertex AI models, reducing reliance on model name pattern matching in routing logic.
