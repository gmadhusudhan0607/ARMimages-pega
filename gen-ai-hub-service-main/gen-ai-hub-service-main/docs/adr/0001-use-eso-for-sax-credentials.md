# ADR-0001: Use External Secrets Operator for SAX Credentials

- **Status**: Accepted
- **Date**: 2026-03-10
- **Deciders**: GenAI Hub Service Team
- **Related ADRs**: None

## Context

The GenAI Hub Service needs to inject ServiceAuthenticationClient (SAX) credentials at runtime for authenticating with Pega's Service Authentication Service. These credentials include:
- ClientId
- Scopes
- TokenEndpoint
- PrivateKey (sensitive)

### Problem Statement

How should we securely inject SAX credentials into the running service with minimal complexity and maximum operational excellence?

### Requirements

- **Security**: PrivateKey must not be exposed in environment variables or etcd
- **Fail-fast**: Credential errors should prevent pod from starting
- **Performance**: Minimal startup latency and container image size
- **Operational**: Clear error messages, Kubernetes-native patterns
- **Maintainability**: Simple code, minimal dependencies

### Assumptions

- External Secrets Operator (ESO) is deployed in all target clusters
- ServiceAuthenticationClientService SCE provides credentials via AWS Secrets Manager
- Secret rotation can tolerate pod restarts (10min refresh interval acceptable)

## Decision

**Use External Secrets Operator (ESO) to mount SAX credentials as a JSON file.**

All credentials (including PrivateKey) are stored in a single JSON file mounted to `/genai-sax-config/genai-sax-config` via ESO. The service reads and validates the file at startup.

### Implementation

```go
func loadSaxConfigFromFile(ctx context.Context) (*saxtypes.SaxAuthClientConfig, error) {
    saxConfigPath := helpers.GetEnvOrDefault("SAX_CONFIG_PATH", "/genai-sax-config/genai-sax-config")

    // Check file exists
    if _, err := helpers.HelperSuite.FileExists(saxConfigPath); err != nil {
        return nil, fmt.Errorf("SAX config file does not exist at %s: %w", saxConfigPath, err)
    }

    // Read and parse JSON
    content, err := helpers.HelperSuite.FileReader(saxConfigPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read SAX config file: %w", err)
    }

    var saxConfig saxtypes.SaxAuthClientConfig
    if err := json.Unmarshal(content, &saxConfig); err != nil {
        return nil, fmt.Errorf("failed to unmarshal SAX config JSON: %w", err)
    }

    // Validate required fields
    if saxConfig.ClientId == "" || saxConfig.PrivateKey == "" ||
       saxConfig.Scopes == "" || saxConfig.TokenEndpoint == "" {
        return nil, fmt.Errorf("SAX config missing required fields")
    }

    return &saxConfig, nil
}
```

### Why This Solution?

1. **Security**: PrivateKey on tmpfs mount (memory-only), not in env vars or etcd
2. **Simplicity**: No AWS SDK dependency (~50MB savings), no runtime AWS calls
3. **Fail-fast**: File read errors prevent pod readiness
4. **Consistency**: Same pattern as genai-gateway-ops and private-model-config
5. **Operational**: ESO handles refresh, clear error messages

## Alternatives Considered

### Alternative 1: Hybrid (Env Vars + AWS Runtime Fetch)

**Description**: Metadata (ClientId, Scopes, TokenEndpoint, SecretArn) in env vars, PrivateKey fetched at runtime from AWS Secrets Manager using AWS SDK.

**Pros**:
- PrivateKey never in environment variables
- Can rotate secrets without pod restart
- Metadata visible in pod description

**Cons**:
- Requires AWS SDK dependency (~50MB)
- Runtime network call to AWS (~50-200ms)
- Two failure modes: env var missing OR AWS fetch fails
- Requires IRSA + IAM permissions
- Error happens after pod starts (harder to debug)

**Rejected because**: Unnecessary complexity and dependencies. Runtime AWS calls add latency and failure modes. The current approach already exists in genai-gateway-ops.

### Alternative 2: Env-Var Only

**Description**: All credentials including PrivateKey passed as environment variables from SCE.

**Pros**:
- Simplest code (~15 LoC)
- No external dependencies
- Fastest startup
- Visible in pod spec

**Cons**:
- **SECURITY RISK**: PrivateKey in environment variables
- Visible in pod spec (`kubectl get pod -o yaml`)
- Stored in etcd
- Visible in logs if accidentally logged
- Violates security best practices

**Rejected because**: Unacceptable security risk. Private keys should never be in environment variables or etcd.

## Consequences

### Positive

- ✅ **Smaller container images**: ~50MB savings (no AWS SDK)
- ✅ **Faster startup**: ~150ms saved (no AWS network call)
- ✅ **Better security**: PrivateKey on tmpfs, not in env vars/etcd
- ✅ **Fail-fast**: Clear errors before pod reports ready
- ✅ **Simpler code**: ~35 LoC, only stdlib dependencies
- ✅ **Consistent pattern**: Matches ops service and private-model-config
- ✅ **Better debugging**: File read errors are clear and actionable

### Negative

- ❌ **Requires ESO**: Must be deployed in all target clusters
- ❌ **Secret rotation requires pod restart**: ESO refreshes mount every 10min, triggering restart
- ❌ **Less visible debugging**: Credentials not in `kubectl describe pod` env vars

### Neutral

- File mounted to tmpfs (memory-only filesystem)
- ESO refresh interval: 10 minutes (configurable)
- File permissions: 0400 (read-only)

## Implementation

### Migration Path

1. ✅ Create loadSaxConfigFromFile function
2. ✅ Add comprehensive tests with parallel execution
3. ✅ Update Helm chart to mount ESO secret
4. ✅ Remove AWS SDK dependencies
5. ✅ Update `.github/copilot-instructions.md` documentation
6. [ ] Deploy to dev environment
7. [ ] Validate in staging
8. [ ] Production rollout

### Estimated Effort

- Development: 4 hours ✅ Complete
- Testing: 4 hours ✅ Complete
- Documentation: 2 hours ✅ Complete
- Deployment validation: 4 hours
- **Total: 14 hours (~2 days)**

### Risks

**Risk**: ESO not deployed in target cluster
**Mitigation**: Pod fails with clear K8s event, ESO is already deployed in all production clusters

**Risk**: File corruption or malformed JSON
**Mitigation**: Explicit validation with clear error messages, pod fails readiness

**Risk**: Secret rotation disruption
**Mitigation**: 10min refresh interval, graceful pod restart

## References

- [SAX Credential Approach Comparison (original)](../../SAX_CREDENTIAL_APPROACH_COMPARISON.md)
- [External Secrets Operator](https://external-secrets.io/)
- [ServiceAuthenticationClientService SCE](../../../distribution/genai-service-authentication-client-sce/)
- [Implementation PR](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/422)

## Notes

This decision aligns with Pega's security best practices for credential management and follows established patterns in the genai-gateway-ops service.
