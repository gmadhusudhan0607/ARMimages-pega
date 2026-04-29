# GenAI Vector Store — AI Agent Instructions

This file provides guidance to AI coding assistants (GitHub Copilot, OpenAI Codex, and others) when working with this repository.

## Project Overview

GenAI Vector Store is a Go microservice that stores, indexes, and retrieves AI-generated vector embeddings backed by PostgreSQL/pgvector. Three independent binaries share the `internal/` codebase: **service** (REST API - CRUD, search, bulk operations), **ops** (admin endpoints - schema management), **background** (async workers - migrations, re-embedding, indexing, cleanup). HTTP routing via Gin, database access via pgx/v5, structured logging via zap.

## Build & Test Commands

```bash
# Build (includes fmt, vet, lint, staticcheck)
make build

# Unit tests (includes fmt, vet, lint, mockery)
make test

# Run a single unit test
go test ./internal/package/... -run TestName -v

# Integration tests (Ginkgo, docker-compose infra)
make integration-test-run                    # all (requires running services + DB)
make integration-test-run-locally            # all (local services)
FOCUS='pattern' make integration-test-run-locally  # focused

# Background integration tests (testcontainers - self-contained)
make integtest-background                    # all background tests
FOCUS='pattern' make integtest-background    # focused
KEEP=60s FOCUS='pattern' make integtest-background  # keep containers for debugging

# Other integration test targets
make integtest-timeout                       # timeout/throttling tests
make integtest-reembedding                   # re-embedding tests
make integration-test-run-locally_readonly_mode
make integration-test-run-locally_emulation_mode

# Lint & static analysis
make lint                                    # golangci-lint
go tool staticcheck ./...

# Generate mocks
make mockery                                 # or: go tool mockery

# Pact contract tests
make pact-test

# Local dev
make run-service                             # service on default port
make run-ops                                 # ops on default port
make run-background                          # background on default port
```

## Architecture

### Entry Points (cmd/)

| Binary | Directory | Role |
|--------|-----------|------|
| **service** | `cmd/service/` | Main REST API - CRUD, search, bulk ops, health |
| **ops** | `cmd/ops/` | Admin API - schema management, maintenance |
| **background** | `cmd/background/` | Async workers - migrations, re-embedding, indexing, cleanup |
| **middleware** | `cmd/middleware/` | Shared Gin middleware (auth, logging, metrics, emulation) |

### Core Packages (internal/)

| Package | Purpose |
|---------|---------|
| `config` | Configuration via `helpers.GetEnvOrDefault()` |
| `db` | PostgreSQL via `pgxpool.Pool`, SQL functions, bulk queries |
| `embedders` | Embedding provider integrations (AWS Bedrock, GCP Vertex) |
| `indexer` | Document indexing - embeds chunks into vectors via LLM providers, stores in DB |
| `sax` | SAX authentication middleware |
| `log` | zap logger initialization (`log.GetNamedLogger()`) |
| `metrics` | Prometheus metrics collection |
| `workers` | Background job processing |
| `resources` | Data access layer - `collections`, `isolations`, `documents`, `embedings`, `attributes`, `attributes_group`, `filters` |
| `errors` | Error-to-HTTP-status mapping, unified API error responses |
| `pagination` | Cursor-based pagination (default 500, max 10k) |
| `http_client` | Retryable HTTP client with SAX auth (singleton, used for embedding calls) |
| `helpers` | Utility functions, env var access |
| `schema` | Database schema management |
| `sql` | SQL query builders and functions |
| `queue` | Internal job queue |

### Integration Tests (src/integTest/)

Ginkgo/Gomega BDD tests organized by service area:
- `src/integTest/service/` - service endpoint tests
- `src/integTest/ops/` - ops endpoint tests
- `src/integTest/background/` - background worker tests (testcontainers)
- `src/integTest/functions/` - shared test utilities
- `src/integTest/tools/` - test tooling

## Code Conventions

- **Copyright header** on all new Go files (block comment style):
  ```go
  /*
   * Copyright (c) <current-year> Pegasystems Inc.
   * All rights reserved.
   */
  ```
  Do not modify existing copyright years.
- **Logging**: Always use `log.GetNamedLogger("name")` for zap loggers. Never `fmt.Print*` or `log.Print*`. Use structured fields: `logger.Info("msg", zap.String("key", val))`.
- **File naming**: `snake_case.go`, `snake_case_test.go` for tests.
- **Go version**: 1.25 (from go.mod).
- **Config access**: `helpers.GetEnvOrDefault()` for env vars, `internal/config` for structured config.
- **Dependencies**: Only use approved libraries (gin, zap, pgx/v5, ginkgo/gomega, prometheus, aws-sdk-go-v2, go-sax). Do not introduce new HTTP frameworks, logging libs, or DB drivers.
- **Commit messages**: Prefix with Agile Studio work item ID (e.g., `US-736080: Add AI tooling config`).
- **Branch naming**: `feature/[Story ID]-description`, `bugfix/[Bug ID]-description`.
- **Env vars**: Any change must update `docs/environment-variables.md`.

## Error Handling

- **Fail fast** - return errors to the client, no fallbacks or silent recovery.
- **Wrap errors** with context: `fmt.Errorf("operation failed: %w", err)`.
- **Use `context.Context`** for all DB and HTTP operations.
- No bare `panic` - handle errors explicitly.
- Proper `defer` for resource cleanup.

## Testing

- **Unit tests**: Standard `testing` package, alongside source files (`*_test.go`). Run with `make test`.
- **Integration tests**: Ginkgo/Gomega BDD in `src/integTest/`. Read `src/integTest/README.md` before modifying.
- **Mocks**: Generated by `go tool mockery` (config in `.mockery.yaml`). Never create manual mocks.
- **Pact**: Consumer contract tests in `internal/embedders/pact/`.
- **Test isolation**: Use `ExpectNoIdleTransactionsLeft` for DB tests in `AfterSuite`.
- **Testcontainers**: Background integration tests use testcontainers (self-contained, no external infra).

## Anti-Patterns (Do NOT)

- Do NOT create files outside established project structure.
- Do NOT create `main.go` files outside `cmd/` subdirectories.
- Do NOT bypass middleware patterns or create direct DB connections outside `internal/db`.
- Do NOT implement custom logging (use zap) or custom config patterns (use `internal/config`).
- Do NOT duplicate middleware or create new HTTP server patterns (use Gin).
- Do NOT add dependencies not aligned with existing stack.
- Do NOT generate CLI tools (pegacloud-cli/ already exists separately).

## Key Documentation

- `docs/environment-variables.md` - All env vars with descriptions, defaults, service usage
- `src/integTest/README.md` - Integration test standards and patterns
- `apidocs/` - OpenAPI/Swagger specs
