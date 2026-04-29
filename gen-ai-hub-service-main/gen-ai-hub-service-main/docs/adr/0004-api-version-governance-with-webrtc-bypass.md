# ADR-0004: API Version Governance with WebRTC Bypass

- **Status**: Accepted
- **Date**: 2026-03-25
- **Deciders**: GenAI Hub Service Team
- **Related ADRs**: None

## Context

The GenAI Hub Service proxies Azure OpenAI API requests, which require an `api-version` query parameter. Different Azure OpenAI API versions have different schemas, features, and behaviors. Clients have been sending various `api-version` values (some old, some invalid), leading to maintenance burden and inconsistent behavior.

### Problem Statement

**Challenge**: Azure OpenAI API versioning creates operational and maintenance issues:

1. **Version fragmentation**: Clients send different `api-version` values (`2021-01-01`, `2022-12-01`, `2023-05-15`, `2024-02-01`, `2024-06-01`, etc.)
2. **Breaking changes**: Azure introduces breaking changes between versions
3. **Maintenance burden**: Service must validate and maintain a list of "supported" versions
4. **Client coupling**: Clients are tightly coupled to specific Azure API versions
5. **Deployment complexity**: Updating supported versions requires service redeployment
6. **Testing complexity**: Must test against multiple API version combinations
7. **WebRTC special case**: Realtime WebRTC endpoints use path-based versioning (`/v1/realtime/`), not query parameters

### Requirements

- Decouple clients from Azure API version management
- Ensure consistent API behavior across all clients
- Simplify service deployment (no client coordination)
- Maintain backward compatibility with existing clients
- Support zero-downtime upgrades
- Handle WebRTC realtime endpoints correctly

### Assumptions

- Azure maintains backward compatibility within major versions
- The service team can monitor Azure API changes and update the governed version
- Clients trust the service to select an appropriate API version
- WebRTC endpoints will continue using path-based versioning

## Decision

**Implement dual API version governance strategy:**

1. **Standard OpenAI REST endpoints** (`/chat/completions`, `/embeddings`, `/images/generations`):
   - **Override-always**: Ignore client-provided `api-version`, always use service-governed version (`2024-10-21`)

2. **WebRTC Realtime endpoints** (`/v1/realtime/client_secrets`, `/v1/realtime/calls`):
   - **Bypass**: Passthrough client-provided `api-version` (or no parameter if not provided)

### Chosen Solution

**Implementation** (`cmd/service/api/common.go` + `cmd/service/api/models.go`):

Handler-level api-version injection approach: instead of modifying incoming requests, add the governed api-version only when constructing outbound URLs to Azure.

```go
// Helper function in common.go
func setApiVersionParam(url string) string {
    const governedApiVersion = "2024-10-21"
    separator := "?"
    if strings.Contains(url, "?") {
        separator = "&"
    }
    return url + separator + "api-version=" + governedApiVersion
}

// Applied in handlers (models.go)
func HandleChatCompletionRequest(...) {
    // Extract path without query params
    operationPath := strings.TrimPrefix(c.Request.URL.Path, PrefixPath)

    // Add api-version when constructing outbound URL
    modelUrl := setApiVersionParam(GetEntityEndpointUrl(m.RedirectURL, operationPath))

    // Forward to Azure with governed version
    CallTarget(c, ctx, modelUrl, saxAuthEnabled)
}

// WebRTC endpoints: no setApiVersionParam() call
// Handled by separate handlers that don't use this pattern
```

**Behavior**:

| Client Request | Service Forwards |
|----------------|------------------|
| `/openai/.../chat/completions?api-version=2021-01-01` | `?api-version=2024-10-21` |
| `/openai/.../chat/completions` (no param) | `?api-version=2024-10-21` |
| `/openai/.../v1/realtime/calls?api-version=2021-01-01` | `?api-version=2021-01-01` (passthrough) |
| `/openai/.../v1/realtime/calls` (no param) | (no param - passthrough) |

### Why This Solution?

#### Standard Endpoints: Override-Always

1. **Decoupling**: Clients no longer need to know Azure API versions
2. **Consistency**: All clients use the same tested API version
3. **Simplicity**: No validation logic, no rejection errors
4. **Flexibility**: Service can update API version without client changes
5. **Zero-downtime**: New clients work immediately with any input

#### WebRTC Endpoints: Bypass

1. **Protocol difference**: WebRTC uses path-based versioning (`/v1/realtime/`), not query params
2. **Azure spec**: Realtime endpoints don't define `?api-version=` parameter
3. **Architecture correctness**: Different protocols warrant different handling
4. **Forward compatibility**: If Azure adds query params later, passthrough won't break
5. **Implementation**: WebRTC handlers simply don't call `setApiVersionParam()` - clean and explicit

## Alternatives Considered

### Alternative 1: Validation with Rejection (Previous Implementation)

**Description**: Maintain a list of supported `api-version` values, reject unsupported versions with 400 errors.

```go
supportedApiVersions := map[string]bool{
    "2024-10-21": true,
    "2024-06-01": true,
    "2024-02-01": true,
    "2023-05-15": true,
    "2022-12-01": true,
}

if !supportedApiVersions[apiVersion] {
    return 400 Bad Request
}
```

**Pros**:
- Explicit control over allowed versions
- Client gets immediate feedback on unsupported versions
- Clear documentation of supported versions

**Cons**:
- ❌ Maintenance burden: Must update supported list regularly
- ❌ Breaking changes: Adding/removing versions breaks clients
- ❌ Deployment coordination: Can't deprecate versions without client updates
- ❌ Client errors: Legitimate requests fail with 400
- ❌ Testing complexity: Must test all version combinations

**Rejected because**: Creates tight coupling between service and clients. Every Azure API version update requires coordinated deployment.

### Alternative 2: Full Passthrough (No Governance)

**Description**: Forward client-provided `api-version` unchanged to Azure.

```go
// No modification - passthrough
// Client sends: ?api-version=2021-01-01
// Azure receives: ?api-version=2021-01-01
```

**Pros**:
- Simplest implementation
- Clients have full control
- No service logic needed

**Cons**:
- ❌ Version fragmentation: Different clients use different versions
- ❌ Breaking changes: Azure deprecations break clients directly
- ❌ Testing burden: Must test against all versions clients might send
- ❌ No protection: Clients can send invalid/broken versions

**Rejected because**: Doesn't solve the original problem - still have version fragmentation and client coupling.

### Alternative 3: Per-Endpoint Governance

**Description**: Different governed versions per endpoint type (chat, embeddings, images).

```go
var governedVersions = map[string]string{
    "/chat/completions": "2024-10-21",
    "/embeddings":       "2024-06-01",
    "/images/generations": "2024-02-01",
}
```

**Pros**:
- Flexibility for endpoints with different maturity levels
- Can adopt new versions gradually

**Cons**:
- ❌ Complexity: More configuration to maintain
- ❌ Inconsistency: Same model, different versions across endpoints
- ❌ Testing burden: Must test multiple version combinations
- ❌ Unclear benefit: Azure versions generally apply uniformly

**Rejected because**: Unnecessary complexity without clear benefit. Azure API versions typically apply across all endpoints.

### Alternative 4: Uniform Governance for WebRTC

**Description**: Apply governance to WebRTC endpoints too (then strip param before forwarding).

```go
// Set governed version for all paths
query.Set("api-version", governedApiVersion)

// Strip for WebRTC before forwarding
if strings.Contains(path, "/v1/realtime/") {
    query.Del("api-version")
}
```

**Pros**:
- Uniform behavior across all endpoints
- Simpler mental model

**Cons**:
- ❌ Incorrect architecture: WebRTC doesn't use query param versioning
- ❌ Wasted work: Set param then immediately remove it
- ❌ Misleading: Logs would show api-version for endpoints that don't use it

**Rejected because**: Violates architectural correctness. WebRTC uses path-based versioning; applying query param governance doesn't make sense.

## Consequences

### Positive

- ✅ **Decoupled clients**: Clients no longer need to track Azure API versions
- ✅ **Simplified deployment**: Can update API version without client coordination
- ✅ **Consistent behavior**: All clients use same tested version
- ✅ **Zero-downtime**: Backward compatible, no client changes required
- ✅ **Reduced maintenance**: No validation logic, no supported version list
- ✅ **Fewer errors**: No 400 rejections for unsupported versions
- ✅ **Clear testing**: Single API version to test and validate
- ✅ **Correct architecture**: WebRTC handled per its protocol design

### Negative

- ❌ **Client visibility**: Clients can't control Azure API version (intentional trade-off)
- ❌ **Version changes**: Service update changes API version for all clients simultaneously
- ❌ **Azure deprecations**: Service must monitor Azure API changes proactively
- ❌ **Debugging complexity**: Client-provided version != actual version used
- ❌ **Dual strategy**: Two different approaches (governance vs bypass) adds conceptual complexity

### Neutral

- Governed version (`2024-10-21`) must be updated via service deployment
- Metrics will show consistent `api-version` across all requests (simplifies analysis)
- Client retry logic doesn't need to handle 400 "invalid api-version" errors
- WebRTC bypass requires explicit documentation to avoid confusion

## Implementation

### Code Locations

**Core Logic**:
- `cmd/service/api/common.go:128-140` - `setApiVersionParam()` helper function
- `cmd/service/api/models.go:194,337,415` - Handler-level application in:
  - `HandleImageGenerationRequest()`
  - `HandleChatCompletionRequest()`
  - `HandleEmbeddingsRequest()`

**Tests**:
- `cmd/service/api/common_test.go:213-242` - Unit tests for `setApiVersionParam()`
- `cmd/service/main_test.go` - Integration tests

**Documentation**:
- `apidocs/spec.yaml` - OpenAPI spec updated (api-version marked optional)
- `docs/adr/0004-api-version-governance-with-webrtc-bypass.md` - This ADR

### Implementation Architecture

**Handler-level injection approach**:
1. Incoming requests pass through unchanged (client's api-version preserved in request)
2. Handlers extract path without query parameters using `URL.Path` (not `URL.RequestURI()`)
3. Handlers construct outbound URL and wrap with `setApiVersionParam()`
4. Outbound call to Azure always includes `?api-version=2024-10-21`

**WebRTC bypass**:
- WebRTC handlers (`HandleWebRTCClientSecretsRequest`, `HandleWebRTCCallsRequest`) don't call `setApiVersionParam()`
- Client's api-version (or no parameter) passes through unchanged

**Benefits of handler-level approach**:
- Cleaner separation of concerns (OpenAI-specific logic in OpenAI handlers)
- Incoming requests unchanged (simplifies debugging/logging)
- Explicit control per endpoint type (no middleware path matching logic)
- Reduced code complexity (~100 lines removed from middleware)

### Migration Path

1. ✅ Implement governance middleware (US-740710) - Initial approach
2. ✅ Remove validation logic and supported version map
3. ✅ Update tests (unit, integration, live)
4. ✅ Merge with WebRTC support (US-738621 from main)
5. ✅ Resolve conflicts (combine governance + bypass)
6. ✅ Add WebRTC bypass tests
7. ✅ Document dual strategy in ADR
8. ✅ Refactor to handler-level approach (cleaner architecture)
9. ✅ Update tests for handler-level implementation
10. 🔄 Deploy to staging
11. 🔄 Monitor for Azure API errors
12. 🔄 Deploy to production

### Estimated Effort

- Core implementation (middleware): 4 hours ✅
- Unit tests: 2 hours ✅
- Integration tests: 1 hour ✅
- Documentation: 2 hours ✅
- Merge + conflict resolution: 2 hours ✅
- Refactor to handler-level: 2 hours ✅
- ADR update: 1 hour ✅
- **Total: 14 hours ✅**

### Risks

**Risk**: Azure introduces breaking changes in `2024-10-21`
**Mitigation**:
- Monitor Azure OpenAI API changelog
- Test against Azure staging before updating governed version
- Maintain rollback capability via feature flag

**Risk**: Clients depend on specific Azure API version behavior
**Mitigation**:
- Document that api-version is service-controlled
- Provide upgrade path in release notes
- Monitor error rates after deployment

**Risk**: WebRTC bypass might be forgotten for new endpoints
**Mitigation**:
- WebRTC handlers are separate from standard OpenAI handlers
- Code review catches inappropriate `setApiVersionParam()` calls
- ADR documents the dual strategy clearly

**Risk**: Confusion between governance and bypass strategies
**Mitigation**:
- Comprehensive documentation (this ADR)
- Code comments explaining rationale
- Test coverage demonstrates both behaviors

## References

- [US-740710: API Version Governance Implementation](https://jira.pega.com/browse/US-740710)
- [US-738621: WebRTC Realtime Support](https://jira.pega.com/browse/US-738621)
- [Azure OpenAI API Versioning](https://learn.microsoft.com/en-us/azure/ai-services/openai/api-version-deprecation)
- [Azure OpenAI Realtime WebRTC](https://learn.microsoft.com/en-us/azure/foundry/openai/how-to/realtime-audio-webrtc)
- [Implementation Summary](../../IMPLEMENTATION_SUMMARY.md)
- [Strategy Review](../../API_VERSION_STRATEGY_REVIEW.md)

## Notes

### Why Two Strategies?

The dual strategy (governance vs bypass) reflects architectural reality:

- **Standard REST APIs**: Use query parameter versioning → Service governs query param
- **WebRTC APIs**: Use path versioning (`/v1/realtime/`) → Service doesn't modify path

This isn't inconsistency; it's **correct handling of different protocols**.

### Future Enhancements

Potential improvements (out of scope for this ADR):

1. **Per-model governance**: Different API versions for different models
2. **Response header**: Add `X-Genai-Gateway-Api-Version: 2024-10-21` for observability
3. **Dynamic configuration**: Load governed version from config (not hardcoded)
4. **Version negotiation**: Allow clients to request "latest" or "stable"
5. **Graceful deprecation**: Warn clients when using deprecated versions (before override)

### Azure API Version Compatibility

**Governed version: `2024-10-21`**

Tested with:
- `gpt-35-turbo` (chat completions)
- `gpt-4` (chat completions)
- `text-embedding-ada-002` (embeddings)
- `dall-e-3` (image generation)

**Known Azure API versions** (for reference):
- `2024-10-21` - Current governed version
- `2024-06-01` - Previous stable
- `2024-02-01` - Widely used
- `2023-05-15` - Legacy
- `2022-12-01` - Legacy

### Testing Strategy

**Unit Tests** (3 cases in `common_test.go`):
- URL without query params → adds `?api-version=2024-10-21`
- URL with existing query params → adds `&api-version=2024-10-21`
- URL already has api-version → appends another (Azure uses last value)

**Integration Tests**:
- Full request flow from client to Azure
- Verifies handlers apply governance correctly

### Monitoring Recommendations

After deployment, monitor:
- Azure API error rates (watch for 400/404 from Azure)
- Client error rates (should decrease - no more 400 from gateway)
- Response time (should be unchanged)
- Prometheus metrics `X-Genai-Gateway-Api-Version` header values

### Rollback Plan

If issues arise:
1. Deploy previous version (validation-based) from `origin/main~1`
2. Or add feature flag: `DISABLE_API_VERSION_GOVERNANCE=true`
3. Or update `governedApiVersion` constant to known-good version

### Documentation Updates Needed

- [x] Update `docs/guides/architecture.md` with governance explanation
- [x] Update `README.md` with api-version behavior
- [ ] Update client SDK documentation (if exists)
- [ ] Add runbook entry for updating governed version

## Implementation Evolution

### Initial Approach: Middleware-based (Implemented)

The initial implementation used middleware to modify incoming requests:
- Middleware intercepted all requests early in the chain
- Modified `c.Request.URL.RawQuery` to set governed api-version
- All downstream code saw the modified request
- Required path matching logic to identify OpenAI vs WebRTC endpoints

**Code location**: `internal/request/middleware/request_handler.go:governAPIVersion()`

### Refined Approach: Handler-based (Current)

After review, refactored to handler-level injection:
- Incoming requests pass through unchanged
- Handlers add api-version only when constructing outbound URLs
- Cleaner separation: OpenAI-specific logic in OpenAI handlers
- WebRTC handlers naturally bypass by not calling the helper

**Code location**: `cmd/service/api/common.go:setApiVersionParam()` + `models.go` handlers

### Why the Refactor?

**Benefits of handler-level approach**:
1. **Cleaner architecture**: api-version concerns belong in Azure OpenAI handlers, not global middleware
2. **Simpler request flow**: Incoming request unchanged → easier debugging, logging, metrics
3. **Explicit control**: Each handler explicitly chooses to govern or not (vs middleware path matching)
4. **Less code**: ~100 lines removed (middleware + tests), replaced with ~15 lines (helper + calls)
5. **Better separation**: Standard REST vs WebRTC handled by different handlers naturally

**Trade-off**:
- Must remember to call `setApiVersionParam()` in new Azure OpenAI handlers
- Mitigated by: code review, ADR documentation, existing pattern to follow

The handler-based approach maintains the same external behavior (governance + bypass) with cleaner internal architecture.
