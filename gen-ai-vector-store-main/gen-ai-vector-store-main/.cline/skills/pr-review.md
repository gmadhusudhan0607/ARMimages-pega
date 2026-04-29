# PR Code Review

## Description
Performs a thorough code review of the current branch's changes compared to `main`, checking against Go idioms, project conventions, and security best practices.

## Trigger
When user asks to:
- "review my code"
- "review this PR"
- "check my changes"
- "code review"
- "review before merge"

## Steps

1. **Get the diff and changed files**
   ```bash
   git diff main...HEAD --stat
   git diff main...HEAD
   ```

2. **Read every changed file in full** to understand context (not just diff hunks).

3. **Review all changes** against the checklist below.

4. **Output findings** in the format specified at the end.

## Review Checklist

### Architecture & Structure
- [ ] Changes follow directory structure: `cmd/service/`, `cmd/ops/`, `cmd/background/` entry points, `cmd/middleware/` shared middleware, `internal/` core logic
- [ ] API versioning respected - breaking changes require new version
- [ ] OpenAPI specs in `apidocs/` updated if endpoints changed
- [ ] No new `main.go` outside `cmd/` subdirectories

### Go Code Quality
- [ ] Copyright header: `/* Copyright (c) <year> Pegasystems Inc. ... */` (block comment)
- [ ] `snake_case.go` file naming
- [ ] No bare `panic` - errors handled explicitly
- [ ] Proper error wrapping: `fmt.Errorf("context: %w", err)`
- [ ] `context.Context` as first parameter for DB/HTTP calls
- [ ] Proper `defer` for resource cleanup
- [ ] No dead code, debug code, or TODO comments

### Logging
- [ ] Uses `log.GetNamedLogger("name")` for zap initialization
- [ ] Structured fields: `logger.Info("msg", zap.String("key", val))`
- [ ] No `fmt.Print*`, `log.Print*`, or `println`
- [ ] No sensitive data in logs

### Error Handling
- [ ] Fail fast - return errors, no silent fallbacks
- [ ] Proper error wrapping with context
- [ ] Correct HTTP status codes
- [ ] Consistent error response format

### Database (pgx/pgxpool)
- [ ] Uses `pgxpool.Pool` for connections
- [ ] Bulk queries preferred over N+1
- [ ] Proper transaction handling with rollback
- [ ] Parameterized queries (no SQL injection)

### Configuration
- [ ] Uses `helpers.GetEnvOrDefault()` for env vars
- [ ] Follows `internal/config` patterns
- [ ] `docs/environment-variables.md` updated if env vars changed
- [ ] No hardcoded secrets

### Testing
- [ ] Unit tests alongside source code (`*_test.go`)
- [ ] Standard `testing` for unit, Ginkgo/Gomega for integration
- [ ] Mocks via `go tool mockery` (no manual mocks)
- [ ] Happy path + error cases covered
- [ ] `ExpectNoIdleTransactionsLeft` in DB test `AfterSuite`

### Dependencies
- [ ] Only approved libs (gin, zap, pgx/v5, ginkgo/gomega, prometheus, aws-sdk-go-v2, go-sax)
- [ ] No new HTTP frameworks, logging libs, or DB drivers

### Security
- [ ] Input validation on request parameters
- [ ] SAX authentication enforced where required
- [ ] Parameterized SQL queries
- [ ] No command injection, path traversal, or SSRF

### Performance
- [ ] Minimize DB round-trips (bulk ops, SQL functions)
- [ ] Proper connection pool usage
- [ ] Context timeouts for external calls

## Output Format

### Critical Issues (must fix before merge)
- `file:line` - description, why it matters, suggested fix

### Major Issues (should fix)
- `file:line` - description, impact if not fixed, recommendation

### Minor Issues / Suggestions
- `file:line` - suggestion, why it would be better

### Security Concerns
- `file:line` - issue, potential impact, how to fix

### Positive Observations
- What was done well and why it is valuable

### Educational Notes
- Patterns or concepts worth learning from this review
