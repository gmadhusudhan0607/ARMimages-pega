Detect which parts of the codebase were affected by recent changes and run appropriate tests.

## Steps

1. Identify changed files compared to the last commit (or to `main` if the user specifies):
   ```bash
   git diff --name-only HEAD~1
   ```
   If the user says "test all changes" or "test vs main", use:
   ```bash
   git diff --name-only main...HEAD
   ```

2. Map changed files to the right test commands:

   | Path prefix | Test command | Notes |
   |---|---|---|
   | `internal/` | `make test` | Unit tests for core packages |
   | `cmd/service/` | `make test` | Service entry point |
   | `cmd/ops/` | `make test` | Ops entry point |
   | `cmd/background/` | `make test` + `make integtest-background` | Background worker + integration |
   | `cmd/middleware/` | `make test` | Shared middleware |
   | `src/integTest/background/` | `make integtest-background` | Background integration tests |
   | `src/integTest/service/` | `make integration-test-run-locally` | Service integration tests |
   | `src/integTest/ops/` | `make integration-test-run-locally` | Ops integration tests |
   | `src/integTest/functions/` or `src/integTest/tools/` | `make integration-test-run-locally` | Shared test utilities |

   - If `internal/` was changed, always run `make test` (unit tests).
   - If only integration test files changed, run only the relevant integration target.
   - If no testable files changed (docs, CI config, Makefile), report that no tests are needed.
   - When in doubt, run `make test` (unit tests are fast and safe).

3. Run the identified test commands. Report progress as tests run.

4. Report results:
   - For each test suite: PASSED or FAILED with a summary
   - If any test failed, show the failure output (last 50 lines)
   - Total: `X of Y test suites passed`

## Optional: Running a single test

If the user specifies a test file, test name, or package:

**Unit test:**
```bash
go test ./internal/package/... -run TestName -v
```

**Integration test (Ginkgo):**
```bash
FOCUS='test pattern' make integtest-background
# or
FOCUS='test pattern' make integration-test-run-locally
```

**Keep containers for debugging (background tests):**
```bash
KEEP=60s FOCUS='test pattern' make integtest-background
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
- Always use `make` targets for full test runs - never call `go test` directly for that (Makefile sets up env vars, runs fmt/vet/lint).
- Use `go test` directly only for running a single targeted test.
- Do not modify any code - this skill is read-only + test execution.
- If a test fails, report the failure clearly but do not attempt to fix it unless the user asks.
- Background integration tests use testcontainers (no external infra needed). Other integration tests need running services + DB.
