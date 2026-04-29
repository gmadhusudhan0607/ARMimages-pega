# GenAI Hub Service — Copilot Instructions

<!--
================================================================================
SINGLE SOURCE OF TRUTH for repository-wide agent instructions.

This file is what the GitHub Copilot coding agent reads server-side on Pega's
self-hosted runners (see KnowledgeHub: SDLC:Github_Copilot_Agent). It is also
surfaced to Copilot Chat in the browser for PR reviews.

`AGENTS.md` at the repo root is a **relative symlink to this file** so that
the Copilot CLI and any other `AGENTS.md`-aware tooling see identical content.

DO NOT create new parallel instruction files (`CLAUDE.md`, `.cursor/rules`,
`.claude/`, new `.opencode/`-resident instructions, etc.) — they will drift.
Existing `.opencode/` files are temporarily retained as legacy compatibility
artifacts only; they are not authoritative and must not be modified or added to
as part of normal repository changes. If a tool reads a different path, add a
relative symlink that points here instead of creating or updating duplicate
instructions.

A CI workflow at `.github/workflows/instructions-drift-check.yml` enforces
these invariants when the relevant instruction files are changed in a PR.
================================================================================
-->

## CRITICAL: Secrets Protection

**NEVER add passwords, API keys, tokens, or sensitive information to ANY file.**

### Rules

- **NEVER** hardcode credentials in files, logs, PRs, or commit messages
- **ALWAYS** use empty strings (`artifactoryPassword=`) or placeholders (`****`) in configuration files
- **ALWAYS** use environment variables or secrets management for actual credentials

### Pre-Commit Secret Scan (MANDATORY)

Before every `git add` or commit, run:

```bash
grep -rE "(password|apiKey|token|secret)=\S+" . --exclude-dir=.git
```

- If secrets found: STOP, replace with empty values, re-scan
- Only proceed when scan is clean

### Example

```properties
# CORRECT
artifactoryPassword=
apiKey=

# WRONG
artifactoryPassword=MySecret123
```

**If secrets are committed:** remove immediately, report issue, rotate credentials.

---

## CRITICAL: Pull Request Title Requirement

**When creating a Pull Request from an issue, you MUST use the EXACT same title as the issue.**

**RULE: The PR title MUST be identical to the issue title. Copy it exactly — do not paraphrase, summarize, or create a new title.**

#### Examples

- Issue #24: "ENHANCEMENT-725045: Ensure copilot draft PR has AS/OA title & unit test validation"
- PR title: "ENHANCEMENT-725045: Ensure copilot draft PR has AS/OA title & unit test validation"
- NOT: "Update golang.org/x/net to v0.17.0 to address security vulnerability"

**Remember:**

- Extract the issue title FIRST before doing any work
- Use it as-is for the PR title
- This includes work item IDs, prefixes, and any special characters

---

## IMPORTANT: Workflow Execution

**Do NOT skip, combine, or deduplicate any stages in the canonical workflow.**

All steps must be executed independently and completely, even if they seem similar or redundant. Each stage serves a specific purpose and must be performed.

---

## Project Overview

**GenAI Hub Service** ("Gateway") — the central API gateway through which all Pega Infinity instances and other systems connect to call LLMs (AWS Bedrock, Azure OpenAI, GCP Vertex AI). Go service deployed on Kubernetes.

Two services in this repo:

| Service | Entry point | Port |
|---|---|---|
| GenAI Hub Service | `cmd/service/main.go` | 8080 |
| GenAI Gateway Ops | `cmd/ops/main.go` | 8081 |

Both expose health on 8082 (`/health/liveness`, `/health/readiness`). Stack: Go (version in `go.mod`), Gin, Zap, OpenTelemetry, Prometheus. Build via `make` locally, Gradle in CI.

Infrastructure SCEs live under `distribution/` and span four Pega products: **GenAIGatewayServiceProduct**, **GenAIInfrastructure**, **GenAIInfrastructureGCP**, **GenAIPrivateModels**. See `SCE_TO_PRODUCT_MAPPING.md` for the full mapping.

## Documentation Structure

Before working on a task, read the relevant guide from `docs/guides/`:

| Guide | Read when… |
|---|---|
| `docs/guides/building_and_testing.md` | Building/testing; running locally; `make` targets; `make test`, `make lint`, `make integration-test-*`, `make test-live` |
| `docs/guides/architecture.md` | Understanding two-service architecture, model registry, request pipeline, configuration system, distribution layout |
| `docs/guides/infrastructure_coordination.md` | Infrastructure/SCE changes; adding env vars (3-layer); adding Bedrock/Vertex models; zero-downtime constraints |
| `docs/guides/code_conventions.md` | Writing new code; patterns; test helpers; naming/structure; import grouping; error handling |
| `docs/guides/model_lifecycle.md` | Adding/updating deprecation dates; `model-metadata.yaml` ConfigMap; lifecycle fields in API responses |
| `docs/adr/` | Understanding *why* architectural decisions were made. See `docs/adr/README.md`. |

## Testing Requirements

- **Always run `make test` before creating a PR** — regardless of the perceived scope of the change. See `docs/guides/building_and_testing.md` for full command reference.
- Tests pass → proceed. Tests fail → fix them; do NOT open a PR with failing tests.
- CI (Jenkins + SonarQube) enforces **≥80% coverage on new code**.
- Run `make lint` and `make staticcheck` before PR.
- Do **not** introduce env-var-based test configuration that could cause parallel-test flakes — use programmatic setup.

## Critical Code Rules

Detailed conventions live in `docs/guides/code_conventions.md`. Non-negotiables:

1. **Zero-downtime**: all changes MUST be forward- and backward-compatible. No breaking changes.
2. **No dead code**: remove unused functions, types, variables.
3. **Pattern priority**: Go stdlib > Kubernetes/Uber-Go > other open source.
4. **No magic strings**: use named constants.
5. **Table-driven tests** for repetitive cases.
6. **Assign unused returns** with `_, _ =` (SonarQube).
7. **Nested `*test` packages** (stdlib `httptest` pattern) — see ADR-0002.

### Internal Dependency Note

This project depends on `github.com/Pega-CloudEngineering/go-sax` — a private Pega module. The `copilot-setup-steps.yml` workflow supplies `.netrc` credentials and sets `GOPRIVATE=github.com/Pega-CloudEngineering/*`.

## Infrastructure Changes

Before touching `distribution/`:

1. Read `docs/guides/infrastructure_coordination.md`.
2. Consult `SCE_TO_PRODUCT_MAPPING.md` to identify product and resource type.
3. Control-plane upgrades first, backing services second.
4. Maintain backward compatibility.

---

## Agent Team Workflow

Specialist agents live under `.github/agents/*.md`. Each file is self-contained and declares its trigger description in its frontmatter. Three categories:

### Workflow Orchestrators (entry points)

Orchestrator agents coordinate the specialist agents below. **For a new work item, default to `jarvis`.** Invoke `brain` or `toast` only when the request explicitly calls for specification-only work or spec-driven implementation.

| Orchestrator | Use when… | Outputs |
|---|---|---|
| `jarvis` | **DEFAULT for new work items.** Implement end-to-end: branch → design → code → tests → QA → review | Committed code + tests + reviewed PR |
| `brain` | Specification only: analyze impacts, design approach, break down tasks; no implementation | Spec in PR body; optional new ADR in `docs/adr/` |
| `toast` | A specification already exists (e.g., written by `brain`); execute it | Committed code + tests + reviewed PR |

**Default-orchestrator rule (cloud coding agent)**: when an issue is assigned to the Copilot coding agent, invoke the `jarvis` agent as the entry point unless the issue explicitly asks for specification-only work (use `brain`) or for implementing an already-written spec (use `toast`).

**Invocation in the Copilot CLI**: use `/jarvis <args>`, `/brain <args>`, or `/toast <args>` — these are thin shims at `.github/prompts/{jarvis,brain,toast}.prompt.md` that dispatch the corresponding orchestrator agent. **Source of truth for each workflow is the `.github/agents/` file, not the prompt shim** — `.github/prompts/*.prompt.md` is a Copilot-CLI-only feature and is NOT picked up by the cloud coding agent.

**Invocation in opencode** (local harness only): commands at `.opencode/commands/{jarvis,brain,toast}.md` duplicate content with the orchestrator agents for now and will be unified in a future cleanup.

### Specialist agents

Dispatched by the orchestrators; can also be called directly when the work is scoped to one area.

| Agent | Use when… |
|---|---|
| `git-committer` | Committing, rebasing, branch creation (`ENHANCEMENT-{number}/short-description` or `BUG-{number}/short-description`), opening PRs |
| `rubber-duck` | Design review and specification before implementation |
| `go-developer` | Go application code in `cmd/` and `internal/` (handlers, middleware, business logic, bug fixes) |
| `go-infra-engineer` | Terraform, Helm, SCE definitions, model specs, environment variable coordination |
| `go-test-developer` | Writing unit, integration, and live test code |
| `qa-tester` | Running `make build` and `make test`; diagnosing unit-test failures |
| `qa-integration-tester` | Running Docker-based integration tests |
| `qa-test-live` | Running live tests against real LLM providers |
| `qa-apidocs` | Verifying the OpenAPI spec matches the code |
| `reviewer` | Final code review after tests pass |
| `security-reviewer` | OWASP audits, credential scans, SSRF/injection analysis |
| `pega-cloud-provisioner` | Pega Cloud self-service resource provisioning workflows |

### Task Delegation

**FIRST STEP**: When starting a new Agile Studio Next work item, **immediately** dispatch `git-committer` to create the feature branch `ENHANCEMENT-{number}/short-description` or `BUG-{number}/short-description` BEFORE dispatching any developer agents. All work must happen on the feature branch, never on `main`. (The `jarvis`, `brain`, and `toast` orchestrators already enforce this.)

When implementing a user story, the developer agent MUST also:

1. **Create live tests** if the feature is testable end-to-end. Live tests go in `test/live/`:
   - Prompt-based: add a prompt directory under `test/live/prompts/{name}/` with `system-prompt`, `user-prompt`, `embeddings-input`.
   - Programmatic (e.g., large payloads): add Go test functions in `test/live/runner/` following existing suite/runner patterns.
   - Config-based: add a config directory under `test/live/configs/{name}/`.
2. **Update model specs** (`internal/models/specs/`) and **model metadata** (`distribution/.../model-metadata.yaml`) when model capabilities change.
3. **Update relevant README.md files** when adding new features, test types, or changing project structure.

### Canonical Workflow

The canonical end-to-end workflow is defined in `.github/agents/jarvis.md`. Variants:

- **Specification-only**: use `brain` — produces a spec in the PR body and may create an ADR.
- **Spec-driven implementation**: use `toast` — assumes a spec already exists.
- **Model additions**: `jarvis` handles them; update specs/metadata and add appropriate live tests (prompt-based, programmatic, or config-based).
- **Bug fixes**: `jarvis` handles them; reproduce with `qa-tester` or existing live tests first, then add/update regression tests before code changes.

---

## Pull Request Descriptions

- **PR Title**: use exact issue title (see CRITICAL section above).
- **Description**: clearly describe what was fixed or changed; link the issue number; include test-pass evidence; list modified files.

## What NOT to do

- Do not modify `.github/workflows/copilot-setup-steps.yml` unless specifically asked.
- Do not modify `pipeline/configuration.properties` (Jenkins CI config) unless specifically asked.
- Do not modify `distribution/` Terraform or Helm files for routine code changes.
- Do not add new direct dependencies without justification — prefer stdlib solutions.
- Do not skip running `make test` or `make lint` to verify changes.
- Do not introduce environment-variable-based test configuration that could cause parallel test failures — use programmatic test setup.
- Do not create parallel instruction files (`CLAUDE.md`, `.claude/`, `.cursor/rules`, etc.). If a new tool needs a different path, add a relative symlink to this file.
- Do not add agent-memory directories (`.claude/agent-memory/`, `.opencode/agent-memory/`, `MEMORY.md`). Institutional knowledge belongs in `docs/guides/` where it is reviewable. The drift-check workflow will fail the PR if these return.
- Do not add workflow orchestration (multi-stage agent chains) as `.github/prompts/*.prompt.md` files alone — those are a Copilot-CLI-only feature and are NOT picked up by the cloud coding agent. Workflow orchestrators belong in `.github/agents/` (see `jarvis`, `brain`, `toast`); `.github/prompts/` should only contain thin shims that dispatch to the corresponding agent.
