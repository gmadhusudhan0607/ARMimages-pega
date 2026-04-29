# Log Analyzer

## Description
Analyzes Kubernetes pod logs collected in `tmp/logs/` to identify errors, warnings, and recurring issues across Vector Store services.

## Trigger
When user asks to:
- "analyze logs"
- "find errors in logs"
- "what's wrong in the logs"
- "check logs for issues"
- "debug from logs"

## Steps

### Step 0: Verify log files exist

```bash
if [ ! -d tmp/logs ] || [ -z "$(ls -A tmp/logs/*.log 2>/dev/null)" ]; then
  echo "No log files found in tmp/logs/."
  echo "Run the collect-logs skill first: ask Cline to 'collect pod logs'"
  exit 1
fi
ls -lh tmp/logs/*.log
```
If no logs found, stop and tell the user to run the collect-logs skill first.

1. **List available log files**
   ```bash
   ls -lh tmp/logs/*.log
   ```

2. **Scan each log file for problems** (case-insensitive):
   - `"level":"error"`, `"level":"dpanic"`, `"level":"fatal"`
   - `"level":"warn"`
   - `stacktrace`, `panic`, `runtime error`
   - `OOMKilled`, `CrashLoopBackOff`, `timeout`, `context deadline exceeded`
   - `connection refused`, `connection reset`, `connection pool exhausted`
   - `pgx`, `pool exhausted`, `conn closed`, `database`
   - `sax`, `authentication failed`, `unauthorized`, `403`
   - `embedding`, `embedder`, `provider error`, `rate limit`
   - HTTP 5xx: `"status":5`, `status_code.*5[0-9][0-9]`

3. **Extract context** (3-5 lines around each issue) to understand the cause.

4. **Group by service** (derived from pod name in filename):
   - `genai-vector-store-<hash>` -> service
   - `genai-vector-store-ops-<hash>` -> ops
   - `genai-vector-store-background-<hash>` -> background

5. **Produce report**:

   ### Summary
   - Total log files analyzed: N
   - Services with issues / Services healthy

   ### Issues by Service
   Per service: errors (count + first/last occurrence), warnings, stack traces

   ### Patterns & Recommendations
   - Recurring issues, cross-service correlations, suggested actions

## Rules
- Read-only - do not modify log files
- Large files (>10MB): sample last 5000 lines
- Deduplicate repeated identical errors - show count + first/last occurrence
- Focus on actionable issues, not routine info messages
- If user specifies a service or symptom, prioritize accordingly

## Notes
- Log format is zap JSON: `{"level":"error","ts":1234567890.123,"caller":"pkg/file.go:42","msg":"...","stacktrace":"..."}`
- Key VS-specific patterns to watch: pgx errors, pool exhaustion, SAX auth failures, embedding service timeouts, context deadline exceeded
