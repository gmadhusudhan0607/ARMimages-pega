# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this repository.

@AGENTS.md

## Agent Team Workflow

This repository uses specialized agents for complex workflows. Agent definitions are in `.claude/agents/`.

### Available Agents

**rubber-duck** (Opus) - Pre-Implementation Design Review
- Use when starting new features, schema changes, or anything with migration/compatibility risk
- Surfaces HNSW rebuild costs, isolation boundary risks, rolling upgrade implications before code is written

**go-developer** (Opus) - Go Application Code
- Scope: `cmd/`, `internal/` (except DB layer and infrastructure)
- Do NOT use for DB layer or test code

**db-developer** (Opus) - Database & pgvector Layer
- Scope: `internal/db/`, `internal/schema/`, `internal/sql/`, `internal/resources/`, migrations
- Use for schema changes, SQL queries, HNSW index tuning, migration planning

**go-infra-engineer** (Opus) - Infrastructure
- Scope: `distribution/` (SCE, Terraform, Helm), `docs/environment-variables.md`
- Use for SCE changes, env var coordination, Helm chart updates

**go-test-developer** (Opus) - Test Code
- Scope: unit tests, Ginkgo integration tests, testcontainers background tests, Pact
- Writes tests but does NOT run them

**git-committer** (Sonnet) - Git Workflow
- Commits, rebases, PRs; enforces branch naming and commit message conventions

**qa-tester** (Opus) - Build + Unit Tests
- Runs `make build` + `make test`, diagnoses failures

**qa-integration-tester** (Sonnet) - Integration Tests
- Runs Ginkgo tests, background testcontainers tests, Pact tests

**reviewer** (Opus) - Final Code Review
- Runs after all tests pass; checks anti-patterns, duplication, backward compatibility

**security-reviewer** (Opus) - Security Audit
- SQL injection, tenant isolation leaks, SAX token handling, credential exposure

### Typical Workflow: Feature Development

Use `/start-feature US-XXXXX short-description` to kick off the full workflow automatically.

Manual steps:
1. **rubber-duck** - design review, surface risks
2. **go-developer** + **db-developer** + **go-infra-engineer** - implementation (parallel if independent)
3. **go-test-developer** - write tests
4. **qa-tester** - verify build + unit tests
5. **qa-integration-tester** - verify integration tests
6. **reviewer** + **security-reviewer** - final review (parallel)
7. **git-committer** - commit + PR

### Commands vs Agents — kiedy co używać

**Użyj agenta** gdy:
- Robisz feature development (wieloetapowy, wiele plików, potrzeba izolacji kontekstu)
- Chcesz równoległą pracę (np. reviewer + security-reviewer jednocześnie)
- Zadanie wymaga specjalistycznej wiedzy domenowej (pgvector → db-developer, SCE → go-infra-engineer)

**Użyj command** gdy:
- Chcesz szybki, jednorazowy output w bieżącym kontekście
- `/test` — inteligentnie wykrywa które testy uruchomić na podstawie zmienionych plików
- `/commit` — bezpieczny commit z podglądem przed wykonaniem
- `/pr-review` — lekki quick review bez odpalania pełnych agentów
- `/diagnostics`, `/e2e-test`, `/k8s-*`, `/perf-test` — operacyjne one-offery

### Agent Memory

Each agent has a persistent memory directory at `.claude/agent-memory/<agent-name>/` relative to the git root. Resolve the absolute path at runtime with `git rev-parse --show-toplevel`. The directory already exists - write to it directly with the Write tool (do not run `mkdir` or check for its existence). Contents persist across conversations.

**IMPORTANT**: Agent memory paths must NEVER be hardcoded as absolute paths. Always use `$(git rev-parse --show-toplevel)/.claude/agent-memory/<agent-name>/`.

Guidelines:
- `MEMORY.md` is always loaded into the agent's system prompt - lines after 200 will be truncated, so keep it concise
- Create separate topic files (e.g., `debugging.md`, `patterns.md`) for detailed notes and link to them from MEMORY.md
- Update or remove memories that turn out to be wrong or outdated
- Organize memory semantically by topic, not chronologically
- Use the Write and Edit tools to update memory files

What to save:
- Stable patterns and conventions confirmed across multiple interactions
- Key architectural decisions, important file paths, and project structure
- Solutions to recurring problems and debugging insights

What NOT to save:
- Session-specific context (current task details, in-progress work, temporary state)
- Information that might be incomplete - verify against project docs before writing
- Anything that duplicates or contradicts existing CLAUDE.md instructions

When the user asks to remember something across sessions, save it. When the user corrects you on something from memory, update or remove the incorrect entry immediately. Since this memory is project-scope and shared via version control, tailor memories to this project.

### Progress Reporting

When dispatching background agents:
1. Show a progress dashboard with all dispatched agents and their status (Running/Done/Failed)
2. Report each agent's results immediately when it completes
3. Update the dashboard after each completion
4. Never silently wait — keep the user informed
