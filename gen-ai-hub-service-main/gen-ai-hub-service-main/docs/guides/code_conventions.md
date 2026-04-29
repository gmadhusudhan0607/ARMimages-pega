# Code Conventions and Project Rules

## Pattern Selection and Best Practices

When implementing a feature or solving a problem **without an established pattern in this codebase**, follow this priority order for pattern selection:

### 1. Go Standard Library (highest priority)

- Always check if the Go stdlib has a similar case or pattern
- Examples: `net/http/httptest`, `testing/iotest`, `io.Reader` interface pattern
- Stdlib patterns are idiomatic, well-documented, and familiar to Go developers

### 2. Reference Go Codebases

- Kubernetes: [github.com/kubernetes/kubernetes](https://github.com/kubernetes/kubernetes)
- kubectl: [github.com/kubernetes/kubectl](https://github.com/kubernetes/kubectl)
- Uber Go: [github.com/uber-go](https://github.com/uber-go) (zap, fx, etc.)
- These codebases demonstrate production-grade Go patterns at scale

### 3. Other Open Source Projects (lowest priority)

- Well-maintained projects with similar use cases
- Verify the pattern is idiomatic and widely adopted
- Check if the project follows Go best practices

**Example**: When creating test helpers (`cntx/cntxtest`), we followed the `net/http/httptest` pattern from Go stdlib rather than inventing our own approach.

**Rationale**: Using established patterns from recognized sources ensures:

- Idiomatic Go code that matches community expectations
- Well-tested patterns proven at scale
- Easier onboarding (developers recognize familiar patterns)
- Long-term maintainability and compatibility

## Documentation

- Always update `README.md` when adding or changing functionality in a directory that has one
- After editing Markdown files, run `npx markdownlint-cli <file>` and fix all errors before committing. Ignore MD013 (line-length)

## Code Quality

- After writing code, review it for readability and maintainability before committing
- Remove unused functions, types, and variables — do not leave dead code
- Assign unused return values explicitly with `_, _ =` to satisfy SonarQube (e.g. `_, _ = io.Copy(io.Discard, resp.Body)`)

## Error and Warning Handling

**Never silently discard errors or warnings.** Swallowed diagnostics make production systems difficult to troubleshoot. Every error and warning return value must be handled explicitly.

### Rules

1. **Never use `_ =` or `_ :=` on error returns** unless the function is truly fire-and-forget (e.g., `logger.Sync()`). If discarding is intentional, add an `//nolint:errcheck` comment explaining *why* it is safe
2. **Never use `_ =` on warning/diagnostic returns** (e.g., `warnings []string`). Log them at an appropriate level (`Warn` for operational warnings, `Debug` for informational)
3. **Log unexpected errors from goroutine coordination** (`errgroup.Wait()`, channel receives). Even when the current code guarantees no error, a future refactor could change that. Use a defensive `if err != nil` with a log statement rather than discarding with `_ =`
4. **Propagate or log — never swallow.** If a function returns an error you cannot propagate to the caller, log it with enough context to diagnose the issue (function name, relevant parameters)

### Examples

```go
// BAD — error silently discarded
_ = g.Wait()

// GOOD — defensive logging
if err := g.Wait(); err != nil {
    l.Errorf("unexpected errgroup error: %v", err)
}

// BAD — warnings silently discarded
models, _ := cache.GetModels(ctx)

// GOOD — warnings logged
models, warnings := cache.GetModels(ctx)
if len(warnings) > 0 {
    l.Warnf("model cache warnings: %v", warnings)
}
```

**Rationale**: Silent failures are the hardest class of production bugs to diagnose. Logging unexpected conditions costs nothing at runtime but saves hours of troubleshooting when something goes wrong.

## Constants and Naming

- Don't hardcode values that appear in comments — use named constants. Keep comments value-free so they don't go stale
- No magic strings for file paths or URLs — use constants
- Match existing naming conventions in the codebase (e.g. test output should match `go test` format like `TestLive/config/prompt/Type/subtest`)

## Code Structure

- Don't duplicate logic — if the same pattern appears 3+ times, extract it into a shared function or table-driven approach
- Use table-driven patterns for repetitive code blocks that differ only in parameters
- Don't use anonymous functions longer than 3 lines in goroutines — extract a named method

## Concurrency

- Use UUID v7 for unique IDs that could run in parallel — avoid small collision spaces

## Test Helper Packages

**Pattern**: Follow Go stdlib convention for test helper packages (like `net/http/httptest`)

When creating test utilities that need to be shared across packages:

1. **Create a nested test package** under the main package (e.g., `internal/cntx/cntxtest/`)
2. **Name clearly** - use `*test` suffix (e.g., `cntxtest`, `modeltest`)
3. **Export helpers from main package** - Core functions in main package (e.g., `cntx.NewTestContext()`)
4. **Wrapper in test package** - Test package provides public API for other packages

### Structure Example

```
internal/cntx/
├── context.go              # Core + exported test helpers (NewTestContext, etc.)
├── context_test.go         # Uses cntx.NewTestContext() directly (same package)
└── cntxtest/
    ├── cntxtest.go         # Public API wrapping cntx functions
    └── cntxtest_test.go    # Tests for the test helpers
```

### Usage

```go
// In cntx package tests - use directly (no import needed)
ctx := NewTestContext("test")

// In other package tests - use via cntxtest import
import "github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx/cntxtest"
ctx := cntxtest.NewContext("test")
ctx = cntxtest.WithSaxConfigPath(ctx, "/path")
```

### Import Protection

- Add package name to `TestNoTestutilImportInNonTestFiles` validation in `internal/testutils/testutils_test.go`
- This prevents test helpers from being imported by production code
- Test validation fails if non-test files import `testutils`, `cntxtest`, or other test-only packages

### Benefits

- Follows established Go patterns (familiar to Go developers)
- No import cycles (main package tests use direct calls, others use nested package)
- Clear discoverability (test helpers nested under relevant package)
- Protected by import validation (cannot be used in production code)
- Parallel-safe (no environment variable conflicts)

## Model Management

- Use the `make add-bedrock-model` or `make add-vertex-model` commands (see `building_and_testing.md`)
- For AWS Bedrock models, follow the complete workflow in `infrastructure_coordination.md` (infrastructure + runtime changes)
- Similar coordination is required for GCP Vertex models and private models
