Analyze code changes and update AI assistant configuration files (`CLAUDE.md`, agents, commands) to stay in sync with the codebase. Accepts an optional argument: pass "full" or "review all" for a complete audit; otherwise only branch changes are checked.

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
   - Read key source files, package structure, build commands, config, and conventions
   - No diff needed — compare AI configs against actual code

3. Map changed areas to AI config files that may need updates:

   | Changed area | Candidate AI config |
   |---|---|
   | Core packages (`internal/db`, `internal/resources`, `internal/workers`, etc.) | `CLAUDE.md` Architecture / Core Packages section |
   | Build/test commands (`Makefile`, `build.gradle.kts`) | `CLAUDE.md` Build & Test Commands section |
   | Binary entrypoints, ports (`cmd/`) | `CLAUDE.md` Architecture / Entry Points section |
   | Code conventions (new linting rules, new patterns) | `CLAUDE.md` Code Conventions section |
   | Test commands or test structure (`src/integTest/`, testcontainers) | `CLAUDE.md` Testing section |
   | Distribution / SCE / Helm / Terraform (`distribution/`) | `.claude/agents/go-infra-engineer.md` |
   | DB schema, pgvector indexes, resources layer | `.claude/agents/db-developer.md` |
   | Agent scope changes | `.claude/agents/<agent>.md` |
   | Command workflows | `.claude/commands/<command>.md` |

4. For `CLAUDE.md`:
   - Read the file in full
   - Check each section against actual code:
     - **Build & Test Commands**: verify against `Makefile` targets — run `make help` or read Makefile directly
     - **Architecture / Entry Points**: verify binary names and roles match `cmd/` subdirectories
     - **Core Packages**: verify listed packages exist in `internal/` and descriptions are accurate
     - **Code Conventions**: verify against `.golangci.yml`, actual import patterns, recent code
     - **Agent Team Workflow**: verify agent list matches `.claude/agents/` directory
   - Note specific discrepancies

5. For agent files (`.claude/agents/`):
   - Read each file
   - Check that referenced paths, package names, make targets, and scope boundaries are still accurate
   - Verify the `description` field still matches the agent's actual use cases
   - Pay special attention to `db-developer.md` — schema details (table names, index parameters, isolation model) must match actual code in `internal/db/` and `internal/resources/`

6. For command files (`.claude/commands/`):
   - Read each file
   - Verify referenced bash commands, make targets, and file paths still work
   - Check that kubectl/pegacloud commands match current cluster/namespace conventions

7. Update each file with minimal, targeted fixes:
   - Fix inaccurate commands, stale paths, outdated package descriptions
   - Add brief entries for genuinely new components or conventions
   - Remove references to deleted code or deprecated workflows

8. Report a summary of what was updated:
   - List each file changed with a one-line explanation of what was fixed
   - If nothing needed updating, report "All AI configs are current"

## Rules
- Read before writing — never update a file without reading it first and verifying the discrepancy against source code.
- `CLAUDE.md` must stay concise — only fix incorrect info or add critical new info. Do not add verbose explanations.
- Do NOT touch `docs/`, `apidocs/`, or README files — those belong to `/project-update-documentation`.
- Do not duplicate information that already exists in `docs/` — CLAUDE.md should reference docs, not repeat them.
- Keep the same heading structure, bullet style, and table format as each existing file.
- If a file needs large-scale rewriting, flag it to the user instead of rewriting silently.
