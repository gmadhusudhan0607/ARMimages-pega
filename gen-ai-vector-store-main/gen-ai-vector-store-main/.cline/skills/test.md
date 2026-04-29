# Smart Test Runner

## Description
Detects which parts of the codebase were affected by recent changes and runs only the relevant tests, saving time compared to a full test suite.

## Trigger
When user asks to:
- "run tests"
- "test my changes"
- "test affected code"
- "run unit tests"
- "run integration tests"

## Steps

1. **Identify changed files**
   ```bash
   git diff --name-only HEAD~1
   ```
   Or against main if the user requests:
   ```bash
   git diff --name-only main...HEAD
   ```

2. **Map files to test commands**

   | Path prefix | Test command | Notes |
   |---|---|---|
   | `internal/` | `make test` | Unit tests for core packages |
   | `cmd/service/` | `make test` | Service entry point |
   | `cmd/ops/` | `make test` | Ops entry point |
   | `cmd/background/` | `make test` + `make integtest-background` | Background + integration |
   | `cmd/middleware/` | `make test` | Shared middleware |
   | `src/integTest/background/` | `make integtest-background` | Background integration |
   | `src/integTest/service/` | `make integration-test-run-locally` | Service integration |
   | `src/integTest/ops/` | `make integration-test-run-locally` | Ops integration |

   - `internal/` changes -> always run `make test`
   - Only integration test files changed -> run only relevant integration target
   - No testable files -> report no tests needed

3. **Run tests** - report progress as they run

4. **Report results**
   - Per suite: PASSED / FAILED with summary
   - On failure: show last 50 lines of failing output
   - Total: `X of Y test suites passed`

## Optional: Single Test

**Unit test:**
```bash
go test ./internal/package/... -run TestName -v
```

**Integration test (focused):**
```bash
FOCUS='pattern' make integtest-background
FOCUS='pattern' make integration-test-run-locally
```

**Keep containers for debugging:**
```bash
KEEP=60s FOCUS='pattern' make integtest-background
```

## Alternative: Gradle

Integration tests can also be run via Gradle directly:
```bash
./gradlew integrationTest          # all integration tests
./gradlew componentTest            # pact contract tests
```

Use Gradle when:
- The user explicitly asks for `./gradlew`
- CI/CD pipeline compatibility is important (pipeline uses Gradle)
- You need Gradle-specific options (e.g., `--tests` filter, `--info`)

Prefer `make` targets by default - they wrap Gradle/Go and add env setup, fmt, vet, lint.

## Rules
- Use `make` targets for full runs - never `go test` directly (Makefile sets env, runs fmt/vet/lint)
- Use `go test` directly only for single targeted tests
- Do not modify any code - read-only + test execution
- Report failures clearly but do not auto-fix unless asked

## Notes
- Background integration tests use testcontainers (self-contained, no external infra)
- Other integration tests need running services + DB (docker-compose)
- Makefile targets handle environment setup automatically
