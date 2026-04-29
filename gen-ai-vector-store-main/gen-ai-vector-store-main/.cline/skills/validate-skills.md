# Skill Quality Validator

## Description
Validates Cline skills for correctness, security, reliability, clarity, and actionability. Checks all `.cline/skills/*.md` files against best practices and reports issues grouped by severity.

## Trigger
When user asks to:
- "validate skills"
- "check skill quality"
- "audit skills"
- "review skills for issues"
- "check skills for security problems"

## Steps

1. **List all skill files**
   ```bash
   ls -1 .cline/skills/*.md
   ```

2. **For each skill file, check against validation checklist**

   ### Security Checks

   - [ ] **Token masking**: curl commands with `-H "Authorization: Bearer $TOKEN"` should not leak tokens
     - In output: `echo "Using token: ${TOKEN:0:20}...<REDACTED>"`
     - In reports: explicitly mask as `<REDACTED>`

   - [ ] **Credentials handling**: DB passwords, AWS/GCP secrets should NOT be echoed to stdout
     - Store in env vars or temp files with `chmod 600`
     - Use `export VAR=$(...)` instead of `echo $(...)`

   - [ ] **Destructive operations**: commands that restart pods, delete resources, or modify shared state should:
     - Have explicit warnings in Prerequisites/Steps
     - Require user confirmation before execution (blocking prompt)
     - Follow "ask before acting" principle from CLAUDE.md

   - [ ] **Secret exposure**: temp files with tokens/credentials cleaned up via trap
     ```bash
     trap cleanup EXIT
     cleanup() { rm -f tmp/sensitive-*; }
     ```

   ### Reliability Checks

   - [ ] **Port-forward validation**: verify connectivity after starting port-forward
     ```bash
     kubectl port-forward ... &
     for i in {1..10}; do
       curl -s --max-time 1 http://localhost:8080/... && break
       sleep 1
     done
     ```

   - [ ] **Cleanup on failure**: use trap for temp files and background processes

   - [ ] **Pod name patterns**: kubectl filters handle both Deployment and StatefulSet
     - Deployment: `pod/name-<hash>-<hash>`
     - StatefulSet: `pod/name-0`, `pod/name-1`

   - [ ] **Timeout handling**: long operations (polling, wait) have max timeout
     ```bash
     for i in $(seq 1 12); do
       [ $i -eq 12 ] && echo "TIMEOUT" && exit 1
       ...
     done
     ```

   - [ ] **Error propagation**: failed commands stop execution

   ### Clarity Checks

   - [ ] **Prerequisites section**: required env vars, tools, setup listed with examples
     ```markdown
     | Variable | Required | Default | Description |
     ```

   - [ ] **Step numbering**: consistent numbering (0 for validation, 1-N for main flow)

   - [ ] **Failure guidance**: when checks fail, tell user what to do next

   - [ ] **Priority in checklists**: long checklists (>50 items) have priority guide

   - [ ] **Hardcoded patterns**: note when paths/formats vary by environment

   - [ ] **Progress output**: long operations show progress

   ### Actionability Checks

   - [ ] **Success criteria**: each step has expected output (HTTP code, message)

   - [ ] **Troubleshooting**: common errors mapped to causes and solutions

   - [ ] **Modularity**: skills requiring multiple tools allow partial execution

   - [ ] **Idempotency**: re-running is safe (or warns if not)

   ### Portability Checks

   - [ ] **Cloud-agnostic**: detect cloud provider from env, not hardcoded

   - [ ] **Tool availability**: check for required tools before use

   - [ ] **Path assumptions**: use repo-relative paths

   ### UX / Usability Checks

   - [ ] **Zero-friction start**: can the user invoke this skill with a single phrase and have it work without prior setup? If not, is the setup guided?

   - [ ] **First-run experience**: if required env vars are missing, does the skill ask the user for values (not just error out)? Does it explain where to find them?

   - [ ] **Persistent config**: for skills with env var prerequisites, is there a mechanism to save config for reuse (e.g., `tmp/e2e-env.sh` pattern) so the user doesn't re-enter values every time?

   - [ ] **Graceful pre-check**: does the skill verify prerequisites (kubectl context, required tools, env vars) before starting the main flow, with clear actionable error messages?

   - [ ] **Progress feedback**: for long-running operations (polling, log collection, perf test), does the skill show progress so the user knows it's working?

   - [ ] **Scope clarity**: is it clear what the skill will and won't do? Are destructive operations explicitly called out before execution?

   - [ ] **Partial execution**: for complex skills (diagnostics, e2e-test), can the user run only the relevant parts? Is this documented?

3. **Group findings by severity**

   - **Critical** - security issues, must fix before use
   - **Major** - reliability issues, should fix soon
   - **Minor** - clarity/usability improvements, nice to have

4. **Generate summary report**

   ```markdown
   # Skill Validation Report

   ## Summary

   | Skill | Critical | Major | Minor | Status |
   |-------|----------|-------|-------|--------|
   | <skill>.md | 0 | 0 | 1 | OK |
   | <skill>.md | 1 | 2 | 0 | ISSUES |
   | ... | | | | |

   ## Critical Issues (must fix)

   ### <skill>.md
   - **Line N**: <description of the issue>
     - Fix: <suggested fix>

   ## Major Issues (should fix)

   ### <skill>.md
   - **Line N**: <description of the issue>
     - Fix: <suggested fix>

   ## Minor Issues (nice to have)

   ### <skill>.md
   - **Line N**: <description of the issue>
     - Fix: <suggested fix>

   ## Positive Observations

   - <patterns that are well implemented>

   ## Statistics

   - Total skills: N
   - Skills with critical issues: N
   - Skills with major issues: N
   ```

## Rules

- This skill is **read-only** - do NOT modify any skill files during validation
- Report issues with specific line numbers and fix suggestions
- Group by severity: Critical > Major > Minor
- Include positive observations to highlight good patterns
- If user asks to fix issues, create task list and wait for approval before modifying

## Notes

- Format specific to Cline skills (Title, Description, Trigger, Steps, Rules, Notes)
- Cline skills are source of truth - fixes here should sync to Claude Code via `/sync-to-claude`
- After fixing issues, sync to Claude Code and validate there too
