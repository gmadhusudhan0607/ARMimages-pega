# AI Tooling

This repo has AI-assisted skills for [Claude Code](https://claude.ai/code) and [Cline](https://github.com/cline/cline) (VS Code extension). Both tools can run the same set of operations - the skills are maintained in two formats.

## Available Skills

### Development

| Skill | Claude Code | Cline | Description |
|-------|-------------|-------|-------------|
| commit | `/commit` | auto-trigger | Stage changes and create a commit with work item ID prefix |
| test | `/test` | auto-trigger | Detect changed files and run relevant unit/integration tests |
| pr-description | `/pr-description` | auto-trigger | Generate PR description from branch diff |
| pr-review | `/pr-review` | auto-trigger | Code review against project conventions and Go best practices |

### Kubernetes & Ops

| Skill | Claude Code | Cline | Description |
|-------|-------------|-------|-------------|
| k8s-collect-logs | `/k8s-collect-logs [namespace]` | auto-trigger | Collect logs from VS pods to `tmp/logs/` |
| k8s-analyze-logs | `/k8s-analyze-logs` | auto-trigger | Analyze collected logs for errors and patterns |

### Vector Store Testing & Diagnostics

| Skill | Claude Code | Cline | Description |
|-------|-------------|-------|-------------|
| e2e-test | `/e2e-test` | auto-trigger | Full E2E test: isolation -> upsert -> query -> cleanup |
| diagnostics | `/diagnostics` | auto-trigger | Check isolations, DB state, document processing, SCE config |
| perf-test | `/perf-test` | auto-trigger | Measure latency (p50/p95/p99), throughput, response headers |

### Sync & Quality

| Skill | Claude Code | Cline | Description |
|-------|-------------|-------|-------------|
| sync-from-cline | `/sync-from-cline` | - | Sync Cline skills to Claude Code format |
| sync-to-claude | - | auto-trigger | Same (Cline-side equivalent) |
| validate-commands | `/validate-commands` | - | Validate Claude commands for security, reliability, clarity |
| validate-skills | - | auto-trigger | Validate Cline skills for security, reliability, clarity |

## File Locations

```
.claude/commands/*.md     <- Claude Code skills (invoked via /command-name)
.cline/skills/*.md        <- Cline skills (auto-triggered by conversation context)
```

## Maintaining Skills

**Source of truth:** `.cline/skills/*.md`

When creating or updating skills:
1. Make changes in `.cline/skills/`
2. Validate quality in Cline: "validate skills"
3. Sync to Claude Code format:
   - From Claude Code: `/sync-from-cline`
   - From Cline: "sync skills to Claude Code"
4. Validate Claude commands: `/validate-commands` (checks security, reliability, clarity)

**Why Cline is source:**
- Richer format (Title, Description, Trigger sections)
- Auto-trigger context patterns
- More established workflow

**Direction:** `.cline/skills/` → `.claude/commands/`

Never directly edit `.claude/commands/` for content changes - those are sync targets. Only edit them for Claude-specific format fixes after sync.

## Prerequisites

### For all skills

- Git repository cloned locally

### For K8s / testing skills

**CLI tools:**
- `kubectl` - configured with context to the target cluster
- `sax` CLI (`~/go/bin/sax` or in PATH) - SAX token generation
- `curl`, `jq`, `uuidgen`

**Environment variables:**

| Variable | Used by | Required | Default | Description |
|----------|---------|----------|---------|-------------|
| `VS_NAMESPACE` | e2e, diagnostics, perf | No | `genai-vector-store` | K8s namespace |
| `SAX_SECRET_ID` | e2e, perf | Yes | - | SAX secret ID for **service** (`sax/backing-services/<guid>`) |
| `SAX_OPS_SECRET_ID` | e2e | Yes (full E2E) | - | SAX secret ID for **ops** - different secret, needed for isolation create/delete |
| `VS_REPORT_DIR` | e2e, perf | No | `tmp/reports` | Where to save reports |
| `TOKEN` | diagnostics | For API checks | - | SAX Bearer token |

**Setup options:**

**Option 1 (recommended):** Use config file - set once, auto-loaded by all skills
```bash
cp .claude/test.env.example .claude/test.env
# Edit .claude/test.env with your values (gitignored, stays local)
# Skills auto-load this file - no need to export manually
```

**Option 2:** Export manually each session
```bash
export SAX_SECRET_ID='sax/backing-services/<your-service-guid>'
export SAX_OPS_SECRET_ID='sax/<your-ops-secret>'
export VS_NAMESPACE='genai-vector-store-titan'
```

The skills validate these at startup and will tell you what's missing.

## Artifacts

All generated files go to `tmp/` (repo-relative, gitignored):

| Path | Content | Cleanup |
|------|---------|---------|
| `tmp/vs-*.json` | Temporary request/response bodies | Auto-cleaned after test |
| `tmp/vs-resp-*` | Full response dumps (on demand) | Auto-cleaned after test |
| `tmp/vs-perf-*` | Perf test CSV data and curl format | Kept with report |
| `tmp/reports/` | Saved reports (E2E, perf) | Manual cleanup |
| `tmp/logs/` | Collected K8s pod logs | Manual cleanup |

## Quick Start

### Run an E2E test (Claude Code)

```
export SAX_SECRET_ID='sax/backing-services/<guid>'
```
Then in Claude Code: `/e2e-test`

### Run diagnostics (Claude Code)

Set up port-forward and token, then: `/diagnostics`

### Run a perf test (Claude Code)

```
export SAX_SECRET_ID='sax/backing-services/<guid>'
```
Then: `/perf-test` - the skill will ask for isolation ID, collection, and iterations.

### Cline

Open the project in VS Code with Cline extension. Ask naturally:
- "run E2E test on VS"
- "diagnose why isolation is 404"
- "benchmark VS query, 200 iterations"
