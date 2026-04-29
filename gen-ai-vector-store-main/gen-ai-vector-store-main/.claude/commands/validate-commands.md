Validate Claude Code commands for correctness, security, clarity, and actionability.

## Steps

1. List all command files:
   ```bash
   ls -1 .claude/commands/*.md
   ```

2. For each command file, check against the validation checklist below.

3. Report findings grouped by severity:
   - **Critical** - security issues, must fix before use
   - **Major** - reliability issues, should fix soon
   - **Minor** - clarity/usability improvements, nice to have

4. Generate a summary table with stats per command.

## Validation Checklist

### Security

- [ ] **Token masking**: curl commands with `-H "Authorization: Bearer $TOKEN"` should not leak tokens in logs
  - In output: `echo "Using token: ${TOKEN:0:20}...<REDACTED>"`
  - In reports: explicitly mask tokens as `<REDACTED>`

- [ ] **Credentials handling**: DB passwords, AWS/GCP secrets should NOT be echoed to stdout
  - Store in env vars or temp files with `chmod 600`
  - Use `export VAR=$(...)` instead of `echo $(...)` for credentials

- [ ] **Destructive operations**: commands that restart pods, delete resources, or modify shared state should:
  - Have explicit warnings in the prerequisites section
  - Require user confirmation before execution (blocking prompt)
  - Follow the "ask before acting" principle from CLAUDE.md

- [ ] **Secret exposure**: temp files containing tokens/credentials should be cleaned up via trap
  ```bash
  cleanup() { rm -f tmp/sensitive-*; }
  trap cleanup EXIT
  ```

### Reliability

- [ ] **Port-forward validation**: after starting port-forward, verify connectivity with timeout
  ```bash
  kubectl port-forward ... &
  for i in {1..10}; do
    curl -s --max-time 1 http://localhost:8080/... && break
    sleep 1
  done
  ```

- [ ] **Cleanup on failure**: use trap to clean up temp files and background processes
  ```bash
  trap cleanup EXIT
  cleanup() {
    rm -f tmp/cmd-*
    pkill -f "kubectl port-forward"
  }
  ```

- [ ] **Pod name patterns**: kubectl filters should handle both Deployment and StatefulSet pod naming
  - Deployment: `pod/name-<hash>-<hash>`
  - StatefulSet: `pod/name-0`, `pod/name-1`
  - Background: `pod/name-background-<hash>-<hash>`

- [ ] **Timeout handling**: long-running operations (polling, wait) should have max timeout and clear failure messages
  ```bash
  for i in $(seq 1 12); do
    [ $i -eq 12 ] && echo "TIMEOUT after 60s" && exit 1
    ...
  done
  ```

- [ ] **Error propagation**: failed commands should stop execution (`set -e` or explicit checks)
  ```bash
  RESULT=$(command) || { echo "ERROR: command failed"; exit 1; }
  ```

### Clarity

- [ ] **Prerequisites clarity**: required env vars, tools, and setup should be listed at the top with examples
  ```markdown
  | Variable | Required | Default | Description |
  ```

- [ ] **Step numbering**: use consistent step numbering (0 for validation, 1-N for main flow)

- [ ] **Failure guidance**: when tests/checks fail, tell the user what to do next (not just report failure)

- [ ] **Priority in checklists**: long checklists (>50 items) should have priority guide
  ```markdown
  ## Priority Guide
  - **Critical** = blocks merge
  - **Major** = should fix
  - **Minor** = nice-to-have
  ```

- [ ] **Hardcoded patterns**: avoid hardcoded paths, formats, or assumptions without noting variability
  - Example: SAX secret format varies by env - add note to check with `sax list`

- [ ] **Command output**: show progress for long operations (`[ $((i % 10)) -eq 0 ] && echo "Progress: $i/$N"`)

### Actionability

- [ ] **Clear success criteria**: each step should have expected output (HTTP code, message, etc.)
  ```markdown
  Expected: **201 Created**
  ```

- [ ] **Troubleshooting table**: common errors mapped to causes and solutions

- [ ] **Modularity**: commands that require multiple tools should allow partial execution
  - Example: diagnostics can run DB checks without API tokens

- [ ] **Idempotency**: re-running the command should be safe (or warn if not)

### Portability

- [ ] **Cloud-agnostic**: detect cloud provider from env vars, not hardcoded
  ```bash
  CLOUD_PROVIDER=$(kubectl get deployment ... -o jsonpath='{...CLOUD_PROVIDER...}')
  ```

- [ ] **Tool availability**: check for required tools before use
  ```bash
  command -v jq >/dev/null || { echo "ERROR: jq not found"; exit 1; }
  ```

- [ ] **Path assumptions**: use repo-relative paths (`tmp/`, `.claude/`) not absolute

### UX / Usability

- [ ] **Zero-friction start**: can the user invoke this command with a single phrase and have it work without prior setup? If not, is the setup guided?

- [ ] **First-run experience**: if required env vars are missing, does the command ask the user for values (not just error out)? Does it explain where to find them?

- [ ] **Persistent config**: for commands with env var prerequisites, is there a mechanism to save config for reuse (e.g., `tmp/e2e-env.sh` pattern) so the user doesn't re-enter values every time?

- [ ] **Graceful pre-check**: does the command verify prerequisites (kubectl context, required tools, env vars) before starting the main flow, with clear actionable error messages?

- [ ] **Progress feedback**: for long-running operations (polling, log collection, perf test), does the command show progress so the user knows it's working?

- [ ] **Scope clarity**: is it clear what the command will and won't do? Are destructive operations explicitly called out before execution?

- [ ] **Partial execution**: for complex commands (diagnostics, e2e-test), can the user run only the relevant parts? Is this documented?

## Output Format

```markdown
# Command Validation Report

## Summary

| Command | Critical | Major | Minor | Status |
|---------|----------|-------|-------|--------|
| <command>.md | 0 | 0 | 1 | OK |
| <command>.md | 1 | 2 | 0 | ISSUES |
| ... | | | | |

## Critical Issues (must fix before use)

### <command>.md
- **Line N**: <description of the issue>
  - Fix: <suggested fix>

## Major Issues (should fix soon)

### <command>.md
- **Line N**: <description of the issue>
  - Fix: <suggested fix>

## Minor Issues (nice to have)

### <command>.md
- **Line N**: <description of the issue>
  - Fix: <suggested fix>

## Positive Observations

- <patterns that are well implemented>

## Statistics

- Total commands: N
- Commands with critical issues: N
- Commands with major issues: N
```

## Rules

- This skill is read-only - do NOT modify any command files during validation
- Report issues with specific line numbers and fix suggestions
- Group findings by severity (Critical > Major > Minor)
- Include positive observations to highlight good patterns
- If user asks to fix issues, create a separate task list and wait for approval before modifying files
