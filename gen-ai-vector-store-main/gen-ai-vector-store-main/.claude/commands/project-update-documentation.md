Analyze code changes and update project documentation (`docs/`, `apidocs/`, READMEs) to stay in sync with the codebase. Accepts an optional argument: pass "full" or "review all" for a complete audit; otherwise only branch changes are checked.

## Steps

1. Determine the mode from `$ARGUMENTS`:
   - If arguments contain "full", "review all", or "all" → **full review mode**
   - Otherwise → **branch mode** (default)

2. Gather changes based on the mode:

   **Branch mode** — get what changed on this branch vs main:
   ```bash
   git diff main...HEAD --stat
   git diff main...HEAD --name-only
   git diff main...HEAD
   ```

   **Full review mode** — scan the codebase broadly:
   - Read key source files, package structure, config, and API definitions
   - No diff needed — compare documentation against actual code

3. Map changed areas to documentation files that may need updates:

   | Changed area | Candidate docs |
   |---|---|
   | Binary entrypoints (`cmd/`) | `docs/architecture.md` or equivalent |
   | Core packages (`internal/`) | `docs/architecture.md` |
   | Environment variables (`helpers.GetEnvOrDefault`, `internal/config`) | `docs/environment-variables.md` |
   | API routes/handlers (`cmd/service/`, `cmd/ops/`) | `apidocs/` OpenAPI specs |
   | DB schema, migrations (`internal/schema/`, `internal/db/`) | relevant `docs/` architecture docs |
   | pgvector index config, HNSW parameters | relevant `docs/` |
   | Integration test structure (`src/integTest/`) | `src/integTest/README.md` |
   | Build/Makefile changes | `docs/` if build docs exist |

4. For each candidate doc file:
   - Read the doc file in full
   - Read the corresponding source code it documents
   - Compare: are descriptions, package paths, field names, env var names, defaults, and commands still accurate?
   - Note specific discrepancies

5. For `docs/environment-variables.md` (critical — always check when any env var changed):
   - Read the full file
   - Search changed code for `helpers.GetEnvOrDefault` calls and `internal/config` struct fields
   - Verify every env var is documented with: name, description, default, which services use it (service/ops/background)
   - Add missing vars, remove deleted vars, fix stale defaults

6. For OpenAPI specs (`apidocs/`), if API routes or models changed:
   - Read the current spec file
   - Read the actual route definitions and request/response structs
   - Check that endpoints, request/response schemas, status codes, and descriptions match

7. Update each doc file with minimal, targeted fixes:
   - Fix inaccurate descriptions, stale paths, wrong defaults, outdated field names
   - Add brief documentation for genuinely new features or components
   - Remove references to deleted code

8. Report a summary of what was updated:
   - List each file changed with a one-line explanation of what was fixed
   - If nothing needed updating, report "All documentation is current"

## Rules
- Read before writing — never update a doc without reading it first and verifying the discrepancy against source code.
- Minimal changes only — fix what is actually wrong or missing. Do not reorganize, reformat, or rephrase existing prose.
- Do not add speculative documentation for planned features or TODO items.
- OpenAPI specs are the source of truth for the API surface — update them from route/handler code, not the other way around.
- Do NOT touch `CLAUDE.md`, `.claude/agents/`, or `.claude/commands/` — those belong to `/project-update-agents-and-skills`.
- Keep the same markdown style, heading levels, and formatting conventions as the existing doc file.
- If a doc file needs large-scale rewriting, flag it to the user instead of rewriting silently.
