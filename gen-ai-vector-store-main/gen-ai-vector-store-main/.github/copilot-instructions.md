# Copilot Instructions: GenAI Vector Store

## Project overview

**GenAI Vector Store** is a backend REST API service built in Go that provides vector storage and retrieval capabilities
for Pega's GenAI features. It manages vector embeddings, supports semantic search, and integrates with multiple embedding
providers (OpenAI Ada, AWS Titan, Google Vertex AI). The service uses PostgreSQL with pgvector extension for vector storage.

**Tech Stack:**

- **Language:** Go 1.25+
- **Build:** Gradle (multi-module project)
- **Framework:** Gin (HTTP routing)
- **Database:** PostgreSQL with pgvector extension (pgx/v5 driver)
- **Testing:** Ginkgo/Gomega (BDD for integration tests), standard testing package (unit tests)
- **Linting/Formatting:** golangci-lint, staticcheck, go fmt, go vet
- **Mocking:** mockery
- **Dependencies:** AWS SDK v2, GCP SDK, OpenTelemetry, Prometheus, go-sax (internal auth). More in go.mod file
- **CI/CD:** Jenkins (SDEA pipeline), GitHub Actions (linting)
- **Deployment:** Docker, Helm charts, Terraform

**Folder structure:**

- `cmd/service/` — Main service entrypoint (REST API)
- `cmd/ops/` — Operations/admin service entrypoint
- `cmd/background/` — Background worker service entrypoint
- `cmd/middleware/` — Shared HTTP middleware (logging, metrics, auth, error handling)
- `internal/` — Core business logic (not importable outside this repo):
    - `config/` — Configuration management
    - `db/` — Database connection and utilities
    - `embedders/` — Embedding provider implementations (ada, titan, google, random)
    - `errors/` — Custom error types
    - `headers/` — HTTP header handling
    - `helpers/` — Utility functions
    - `http_client/` — HTTP client configuration
    - `indexer/` — Vector indexing logic
    - `log/` — Structured logging (zap)
    - `metrics/` — Prometheus metrics
    - `pagination/` — API pagination utilities
    - `queue/` — Background job queue
    - `resources/` — Resource management
    - `sax/` — SAX authentication integration
    - `schema/` — Database schema definitions
    - `smart_chunking/` — Text chunking for embeddings
    - `sql/` — SQL queries and functions
    - `workers/` — Background worker implementations
    - `aws/`, `gcp/` — Cloud provider adapters
- `apidocs/` — OpenAPI/Swagger documentation
- `mocks/` — Generated mocks
- `distribution/` — Gradle modules for Docker, Helm, SCE, Terraform
- `src/integTest/` — Integration tests (Ginkgo/Gomega)
- `src/perfTest/` — Performance tests (K6)
- `build.gradle.kts`, `settings.gradle.kts`, `gradle.properties` — Multi-module Gradle build
- `.golangci.yaml` — Linter configuration
- `go.mod` — Go module dependencies

---

## Custom Agents

This repository defines custom agents in `.github/agents/`. Each agent is a specialist - pick the one that best fits the task. For most GitHub issues, start with `task-planner`.

| Agent | When to use |
|-------|-------------|
| `task-planner` | Multi-step features, bug fixes, or refactoring spanning 2+ packages. Think first, plan backward from the goal, then execute atomically. Use this as the default entry point for complex tasks. |
| `go-developer` | Go application code: handlers, middleware, business logic in `cmd/` and `internal/` (excluding DB layer and infrastructure). |
| `db-developer` | Database layer: SQL queries, schema changes, pgvector index management, migrations, `internal/db`, `internal/schema`, `internal/sql`, `internal/resources`. |
| `go-infra-engineer` | Infrastructure-as-code: SCE definitions, Terraform, Helm charts, `distribution/`, env var coordination. |
| `go-test-developer` | Writing or modifying test code: unit tests, Ginkgo/Gomega integration tests, testcontainers tests, Pact consumer tests. Does not run tests. |
| `qa-tester` | Run build verification (`make build`) and unit tests (`make test`). Diagnose build or unit test failures. |
| `qa-integration-tester` | Run integration tests (Ginkgo, docker-compose) and background tests (testcontainers). Diagnose integration failures. |
| `reviewer` | Final code review after all tests pass. Checks anti-patterns, duplication, backward compatibility, and requirement coverage. |
| `security-reviewer` | Security audits: SQL injection via pgvector, tenant isolation leaks, SAX token handling, input validation. |
| `rubber-duck` | Pre-implementation design validation. Use before writing code when a task has hidden complexity (schema migrations, HNSW index costs, breaking changes, deployment strategy). |
| `debugger` | Systematic bug investigation using hypothesis testing. Specialized for VS failure modes: pgvector queries, tenant isolation, SAX auth, embedding provider errors, background worker hangs. |
| `git-committer` | Git workflow: committing, rebasing, branch management, pull request preparation, merge conflict resolution. |

**Typical flow for a GitHub issue:**
1. `task-planner` - plan and implement
2. `qa-tester` - verify build + unit tests
3. `reviewer` + `security-reviewer` - final review
4. `git-committer` - prepare PR

---

## Coding Agent Workflow (for unattended execution)

When working on a GitHub Issue as a Coding Agent, follow these rules strictly.

### Input Format

You will receive tasks as GitHub Issues. The issue title contains the Agile Studio
work item ID (e.g. `US-734574: Fix flaky integration tests`). Parse it and use it
in commit messages and PR title.
Do NOT attempt to access GitHub PR URLs or API endpoints - you don't have that capability.
Focus exclusively on the codebase available on the filesystem.

### PR Title Rule

PR title MUST follow format: `<AS-ID>: <description>` where AS-ID is the Agile Studio
work item from the issue title.

Examples:
- `US-734574: Fix flaky integration tests`
- `BUG-980744: Fix nil pointer in bulk delete handler`

### Mandatory Steps Before Creating PR

1. Read and understand the issue fully
2. Plan the implementation (use task-planner agent approach)
3. Implement changes following project conventions (see Style guide below)
4. Run `make build` to verify compilation and linting
5. Run `make test` to verify unit tests pass
6. Scan diff for secrets: `grep -rE '(password|apiKey|token|secret)=\S+' . --include='*.go' --include='*.yaml'`
7. Create PR with proper title and structured description

### Rules

- Do NOT skip, combine, or deduplicate any of the above steps
- Do NOT create PRs that fail `make build`
- Do NOT commit secrets, passwords, API keys, or tokens
- Do NOT modify files outside the established project structure
- Do NOT add dependencies not aligned with existing stack (see AGENTS.md)
- If unsure about scope, implement the MINIMUM viable change
- Prefer small, focused PRs over large sweeping changes

---

## Review Behavior & Priorities

**Review Output Style:**

- **Always provide suggested changes / inline diffs**, not just comments.
- Example:
  ```diff
  - return userId, err
  + if err != nil {
  +     return "", fmt.Errorf("failed to fetch user: %w", err)
  + }
  + return userId, nil
  ```
- Avoid low-value commentary ("looks fine", "LGTM") unless genuinely actionable.

**Style References:**

- Before submitting any review make sure to read and understand **[Google Go Style Guide](https://google.github.io/styleguide/go/)**.
- Always cite **[Google Go Style Guide](https://google.github.io/styleguide/go/)** if you reference any style or best practice.
- Example: "Per Google Go Style Guide, prefer `fooID` over `fooId` for consistency. Link to source: https://google.github.io/styleguide/go/ ."

**When Submitting a Review:**

- **Take your time and think longer, your review can significantly reduce future human review effort.**
- **Read documentation for packages imported in go.mod.**
- **Search for potential issues or bugs introduced** (nil deref, error handling, concurrency, performance, security).
- **Be familiar with their purpose and changes in imported versions**
- **Documentation updated** (README, godoc comments, examples if public API).
- **No forbidden cross-layer imports** (internal packages don't import from cmd/).
- **Dependency changes justified** (if adding/updating deps: why, version, license).
- **Refer to below style guide** for common pitfalls and best practices.
- **For new code enforce proper usage of libraries and patterns** (e.g., Ginkgo/Gomega for integration tests, context.Context for cancellations, dependency injection for decoupling).
- **Propose improvements** (better error messages, logging, abstractions, simplifications).

---

## Style guide

**Architecture:**

- Internal packages **must not** import from `cmd/` packages
- Use dependency injection: pass interfaces to decouple layers
- Follow established middleware patterns in `cmd/middleware/`
- Use `internal/config` for configuration, `internal/log` for logging
- **Fail fast** - flag code that silently swallows errors or uses fallbacks instead of returning errors
- New Go files **must have a Pega copyright header** (verify it exists, format may vary)

**Code Organization:**

- Avoid **utility catch-all packages** (e.g., adding unrelated helpers to `internal/helpers/`)
- Avoid **god objects** (structs with too many responsibilities)
- Keep packages **small and cohesive**; extract focused shared utilities
- Maintain clear separation between API handlers, middleware, and business logic
- New endpoints should follow established **API versioning** (v1, v2, ...)

**Naming:**

- Avoid **stuttering** in package names (e.g., `config.ConfigStruct` → `config.Settings`)
- Avoid **ambiguous or generic names** (`x`, `tmp`, `data`, `process`)
- Use **clear, intention-revealing names** (`embeddingID`, `vectorStore`, `chunkSize`)

**Global State:**

- Avoid **mutable global variables**
- Avoid **singletons without DI**, which hinder testability
- Always pass dependencies through constructors
- Use `log.GetNamedLogger()` pattern for loggers

**Error Handling:**

- Do not **ignore errors** silently (e.g. `_ = fn()` without comment)
- Avoid **panics in library code**; return errors rather than crashing
- Always **check errors** and wrap them with context (use `%w`)
- Use structured error types from `internal/errors` package

**Database Operations:**

- Always use `context.Context` for database operations
- Use `pgxpool.Pool` for connection pooling (follow patterns in `internal/db`)
- Prefer **bulk operations** to reduce database round-trips
- Use prepared statements and parameterized queries (prevent SQL injection)
- **Flag PRs that increase query count** unnecessarily - look for N+1 patterns

**Concurrency:**

- Avoid **goroutine leaks** by implementing cancellation mechanisms
- Avoid **shared mutable state** without synchronization (prevents data races)
- Avoid **unbounded goroutines**; enforce concurrency limits
- Use `context.Context`, `sync.Mutex`, and `errgroup` as appropriate

**Performance & Security**

- Prevent **resource leaks** (e.g. always `defer rows.Close()`, set HTTP client timeouts)
- Avoid **unbounded slice growth** (preallocate capacity when possible)
- Rigorously **validate external inputs** to prevent SQL injection, path traversal, etc.
- Apply input sanitization and safe defaults consistently
- Use `helpers.GetEnvOrDefault()` for environment variable access

**Logging:**

- Use structured logging with `zap.Logger` from `internal/log`
- **Never** use `fmt.Print*` or `log.Print*`
- Include relevant context in log messages (request IDs, operation names)
- Use appropriate log levels (Debug, Info, Warn, Error)

**Testing**

- Use **Ginkgo/Gomega BDD framework** for integration tests in `src/integTest/`
- Use **standard testing package** for unit tests alongside source code
- Integration tests should follow conventions in `src/integTest/README.md`
- Avoid **unit tests depending on external systems** (real DBs, cloud APIs)
- Avoid **flaky tests** (e.g. use of `time.Sleep`, race conditions)
- **Mock external services** using mockery-generated mocks (preferred for consistency)
- Verify `ExpectNoIdleTransactionsLeft` is called in `AfterSuite` for database tests

**Environment Variables:**

- If PR adds/modifies env vars, verify `docs/environment-variables.md` is updated

**Dependencies:**

- Prefer libraries already in use: gin, zap, pgx/v5, ginkgo/gomega, prometheus
- New dependencies should be justified in the PR description

**Key Principles:**

- **Prioritize readability, simplicity, and maintainability.**
- **Design for statelessness, scalability, and resilience.**
- **Enforce security best practices for cloud-native workloads.**
- **Reduce database load and number of queries where possible.**
- **Fail fast** - prefer returning errors over silent fallbacks.

**Note:** This file is general-purpose for *all* PR review tasks in this repo. It is not tied to a
single feature or PR. Follow these instructions consistently to maximize code quality and minimize additional human
reviewer effort.
