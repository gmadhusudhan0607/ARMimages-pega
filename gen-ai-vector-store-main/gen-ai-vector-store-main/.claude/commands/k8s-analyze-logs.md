Analyze Kubernetes pod logs collected in `tmp/logs/` and identify issues.

## Steps

1. List all log files available:
   ```bash
   ls -lh tmp/logs/*.log
   ```
   If `tmp/logs/` is empty or missing, tell the user to run `/k8s-collect-logs` first and stop.

2. For each log file, scan for problems by searching for these patterns (case-insensitive):
   - `"level":"error"`, `"level": "error"`, `"level":"dpanic"`, `"level":"fatal"`
   - `"level":"warn"`, `"level": "warn"`
   - `stacktrace`, `panic`, `runtime error`
   - `OOMKilled`, `CrashLoopBackOff`, `timeout`, `context deadline exceeded`
   - `connection refused`, `connection reset`, `connection pool exhausted`
   - `pgx`, `pool exhausted`, `conn closed`, `database`
   - `sax`, `authentication failed`, `unauthorized`, `403`
   - `embedding`, `embedder`, `provider error`, `rate limit`
   - HTTP 5xx status codes: `"status":5`, `status_code.*5[0-9][0-9]`

3. For each issue found, extract the surrounding context (3-5 lines before and after) to understand the cause.

4. Group findings by service (derived from pod name in the log filename):
   - `genai-vector-store-<hash>` -> service
   - `genai-vector-store-ops-<hash>` -> ops
   - `genai-vector-store-background-<hash>` -> background

5. Produce a report in this format:

   ### Summary
   - Total log files analyzed: N
   - Services with issues: list
   - Services healthy: list

   ### Issues by Service

   #### service
   **Errors (N occurrences)**
   - `timestamp` - error message (file:line-in-log)
   - Root cause analysis if pattern is clear

   **Warnings (N occurrences)**
   - `timestamp` - warning message

   #### ops
   ...

   #### background
   ...

   ### Patterns & Recommendations
   - Recurring issues (e.g., repeated connection timeouts, pool exhaustion)
   - Correlations across services (e.g., embedding timeout causing background worker failures)
   - Suggested actions

## Rules
- Do NOT modify any log files - this skill is read-only.
- If a log file is very large (>10MB), sample the last 5000 lines instead of reading the entire file.
- Deduplicate repeated identical errors - report the count and first/last occurrence instead of listing each one.
- Focus on actionable issues, not routine info-level messages.
- If the user specifies a service name, only analyze logs for that service.
- If the user describes a specific symptom (e.g., "search requests timing out"), prioritize searching for related patterns first.

## Notes
- Log format is zap JSON: `{"level":"error","ts":1234567890.123,"caller":"pkg/file.go:42","msg":"...","stacktrace":"..."}`
- Key VS-specific patterns to watch: pgx errors, pool exhaustion, SAX auth failures, embedding service timeouts, context deadline exceeded
