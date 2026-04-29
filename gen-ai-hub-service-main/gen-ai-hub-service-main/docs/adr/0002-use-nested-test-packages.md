# ADR-0002: Use Nested Test Packages Following Go Stdlib Convention

- **Status**: Accepted
- **Date**: 2026-03-10
- **Deciders**: GenAI Hub Service Team
- **Related ADRs**: None

## Context

The GenAI Hub Service needs test helper functions that can be shared across packages while being protected from accidental import by production code. Specifically, we need context helpers for parallel-safe testing without environment variable conflicts.

### Problem Statement

How should we organize test helper functions so they:
1. Can be shared across multiple test packages
2. Cannot be imported by production code (enforced by validation)
3. Follow idiomatic Go patterns
4. Avoid import cycles

### Requirements

- Test helpers must be reusable across packages
- Must prevent production code from importing test utilities
- Must avoid import cycles (e.g., cntx_test importing testutils importing cntx)
- Should follow Go community best practices
- Must support parallel test execution

### Assumptions

- Import validation test (`TestNoTestutilImportInNonTestFiles`) will catch violations
- Developers are familiar with Go stdlib patterns like `net/http/httptest`

## Decision

**Use nested test packages following the Go standard library convention.**

Create test helper packages as nested subpackages with `*test` suffix (e.g., `cntx/cntxtest`), following the pattern used by Go stdlib (`net/http/httptest`, `testing/iotest`, etc.).

### Structure

```
internal/cntx/
├── context.go              # Core functions + exported test helpers
├── context_test.go         # Uses cntx.NewTestContext() directly
└── cntxtest/
    ├── cntxtest.go         # Public API wrapping cntx functions
    └── cntxtest_test.go    # Tests for the test helpers
```

### Implementation

**Core functions in main package** (`internal/cntx/context.go`):
```go
// NewTestContext creates a test context with default values
func NewTestContext(name string) context.Context {
    ctx := context.Background()
    l := getLogger()

    ctx = context.WithValue(ctx, platformTypeKey, "infinity")
    ctx = context.WithValue(ctx, useSaxKey, false)
    ctx = context.WithValue(ctx, saxConfigPathKey, "/genai-sax-config/genai-sax-config")
    ctx = context.WithValue(ctx, useGenAIInfraKey, false)

    return context.WithValue(ctx, loggerKey, l.Named(name))
}
```

**Wrapper in nested test package** (`internal/cntx/cntxtest/cntxtest.go`):
```go
// Package cntxtest provides utilities for testing code that uses cntx.
// It follows the Go standard library pattern (e.g., net/http/httptest).
package cntxtest

import "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"

// NewContext creates a test context with default values
func NewContext(name string) context.Context {
    return cntx.NewTestContext(name)
}

// WithSaxConfigPath sets the SAX config path
func WithSaxConfigPath(ctx context.Context, path string) context.Context {
    return cntx.WithSaxConfigPath(ctx, path)
}
```

**Usage patterns**:
```go
// Same package tests - direct call
// internal/cntx/context_test.go
ctx := NewTestContext("test")

// Other package tests - via cntxtest
// cmd/service/main_test.go
import "internal/cntx/cntxtest"
ctx := cntxtest.NewContext("test")
ctx = cntxtest.WithSaxConfigPath(ctx, "/custom/path")
```

### Why This Solution?

1. **Go stdlib convention**: Matches `net/http/httptest`, `testing/iotest`, familiar to Go developers
2. **Clear naming**: `cntxtest` obviously indicates testing utilities
3. **No import cycles**: cntx_test uses cntx directly, others use cntxtest
4. **Discoverability**: Test helpers nested under relevant package
5. **Testable**: Test helpers themselves have test coverage
6. **Validated**: Protected by enhanced import validation

## Alternatives Considered

### Alternative 1: Central testutils Package

**Description**: Place all test helpers in a central `internal/testutils` package with wrapper functions.

**Pros**:
- Single location for all test utilities
- Simple mental model: "test stuff goes in testutils"
- Already has import validation

**Cons**:
- Doesn't follow Go stdlib convention
- Indirection: testutils just wraps cntx functions
- Two different APIs: cntx tests use cntx.*, others use testutils.*
- Less discoverable (not nested under cntx)
- Functions exported from cntx but meant only for tests

**Rejected because**: Doesn't follow established Go patterns. Developers familiar with `net/http/httptest` won't recognize this structure.

### Alternative 2: Keep in testing.go (non-test file)

**Description**: Export functions from `internal/cntx/testing.go` for use by other packages.

**Pros**:
- Simplest: functions in one file
- No nested packages

**Cons**:
- No import protection (testing.go is a regular source file)
- Can be imported by production code accidentally
- Doesn't clearly signal "test-only" nature

**Rejected because**: No protection against production imports. Regular source files can be imported anywhere.

## Consequences

### Positive

- ✅ **Idiomatic Go**: Follows patterns from stdlib (`net/http/httptest`)
- ✅ **Clear intent**: Package name indicates test-only utilities
- ✅ **No import cycles**: Proper separation of concerns
- ✅ **Discoverable**: Nested under `cntx`, easy to find
- ✅ **Protected**: Import validation prevents production use
- ✅ **Testable**: Test helpers have their own test coverage
- ✅ **Extensible**: Easy to add more domain-specific helpers
- ✅ **Parallel-safe**: Tests run without environment variable conflicts

### Negative

- ❌ **More directory nesting**: `internal/cntx/cntxtest/` vs flat structure
- ❌ **Two APIs**: cntx tests use `cntx.*`, others use `cntxtest.*`
- ❌ **Exported from cntx**: Functions like `NewTestContext()` technically public

### Neutral

- Import validation extended to check both `testutils` and `cntxtest`
- Pattern can be replicated for other packages (e.g., `models/modelstest`)

## Implementation

### Migration Path

1. ✅ Create `internal/cntx/cntxtest/` directory
2. ✅ Add `cntxtest.go` with wrapper functions
3. ✅ Add `cntxtest_test.go` with tests
4. ✅ Update `TestNoTestutilImportInNonTestFiles` to check `cntxtest`
5. ✅ Update test files to use `cntxtest.NewContext()`
6. ✅ Document pattern in `.github/copilot-instructions.md`
7. ✅ Create comparison document

### Estimated Effort

- Implementation: 2 hours ✅ Complete
- Testing: 1 hour ✅ Complete
- Documentation: 2 hours ✅ Complete
- **Total: 5 hours**

### Risks

**Risk**: Developers unfamiliar with nested test package pattern
**Mitigation**: Document in `.github/copilot-instructions.md` with clear examples and references to Go stdlib

**Risk**: Import cycles if not careful
**Mitigation**: Pattern clearly documented, test helpers can only import cntx, not vice versa

## References

- [Test Context Approaches Comparison (original)](../../TEST_CONTEXT_APPROACHES.md)
- [Go stdlib: net/http/httptest](https://pkg.go.dev/net/http/httptest)
- [Go stdlib: testing/iotest](https://pkg.go.dev/testing/iotest)
- [Go stdlib: testing/fstest](https://pkg.go.dev/testing/fstest)
- [copilot-instructions.md: Test Helper Packages](../../.github/copilot-instructions.md#test-helper-packages)
- [Implementation PR](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/422)

## Notes

This pattern should be used for future test helper packages. When creating test utilities for other packages (e.g., `models`, `request`, etc.), follow the same nested `*test` package convention.

**Pattern Selection Hierarchy** (from `.github/copilot-instructions.md`):
1. Go Standard Library (highest priority) ✅ Used for this decision
2. Reference Go Codebases (Kubernetes, kubectl, Uber Go)
3. Other Open Source Projects (lowest priority)
