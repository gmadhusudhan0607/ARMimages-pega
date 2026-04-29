Start a new feature: create branch, run design review with rubber-duck, then dispatch the right implementation agents.

Usage: `/start-feature US-XXXXX short-description`

Examples:
- `/start-feature US-736080 add-bulk-search-endpoint`
- `/start-feature BUG-981234 fix-context-leak-in-background-worker`

## Steps

1. **Parse arguments** from `$ARGUMENTS`:
   - Extract work item ID (pattern: `US-\d+` or `BUG-\d+` or `TASK-\d+`)
   - Extract description (remaining text after the ID, spaces replaced with dashes)
   - If no ID found, ask the user: "What's the work item ID? (US-XXXXX or BUG-XXXXX)"

2. **Check git state**:
   ```bash
   git status --short
   git rev-parse --abbrev-ref HEAD
   ```
   - If there are uncommitted changes, warn the user and ask whether to stash them or stop.
   - If already on a feature branch for this ID, skip branch creation and continue from step 4.

3. **Create feature branch**:
   ```bash
   git fetch origin main
   git checkout -b feature/<ID>-<description> origin/main
   ```
   For bug fixes: `bugfix/<ID>-<description>`
   Confirm: "Created branch `feature/<ID>-<description>` from latest main."

4. **Run rubber-duck design review** — use the `rubber-duck` agent with the following context:
   - Work item ID and description
   - Ask the user to briefly describe what they want to build (if not already clear from the arguments)
   - The rubber-duck agent will surface risks, migration implications, and produce an Implementation Brief

5. **After rubber-duck completes**, present the Implementation Brief and ask:
   "Ready to implement? I'll dispatch agents based on the brief. Confirm or adjust:"
   - Show which agents will be used (based on what the brief covers: Go code / DB / infra / tests)
   - Wait for user confirmation

6. **Dispatch implementation agents** in parallel based on the brief:
   - Changes in `cmd/` or `internal/` (non-DB) → `go-developer`
   - Changes in `internal/db/`, `internal/schema/`, `internal/sql/`, `internal/resources/` → `db-developer`
   - Changes in `distribution/` or env vars → `go-infra-engineer`
   - All implementations → also dispatch `go-test-developer` after implementation agents complete

7. **Show progress dashboard** while agents work:
   ```
   | Agent              | Status  | Task                          |
   |--------------------|---------|-------------------------------|
   | go-developer       | Running | Implement handler + indexer   |
   | db-developer       | Running | Add schema migration           |
   | go-test-developer  | Waiting | Write tests (after dev)        |
   ```
   Update after each agent completes. Report results immediately.

8. **After all implementation + tests done**, dispatch in parallel:
   - `qa-tester` — run `make build` + `make test`
   - Ask user if they want integration tests now or later

9. **After QA passes**, dispatch in parallel:
   - `reviewer`
   - `security-reviewer`

10. **After reviews complete**, summarize findings and ask:
    "Ready to commit and create PR? I'll use git-committer."
    If yes → dispatch `git-committer` with the work item ID and a summary of changes.

## Rules
- Never skip rubber-duck for features that involve schema changes or new endpoints — these have migration/compatibility risk
- If the user says "skip rubber-duck" or "I already know what to build" — respect it and go straight to step 5
- Always show the progress dashboard when multiple agents are running
- If any agent fails, stop and report clearly — do not continue to the next step with broken code
- The branch must exist before any implementation agents start
