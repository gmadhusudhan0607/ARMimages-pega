Collect logs from Kubernetes pods in the Vector Store namespace.

The namespace is provided as `$ARGUMENTS`. If not provided, default to `genai-vector-store`.

## Steps

### Step 0: Verify kubectl is available and context is active

```bash
kubectl cluster-info 2>&1 | head -3
```
If this fails, stop and tell the user:
- `kubectl` is not installed or not in PATH
- No active cluster context (`kubectl config current-context`)
- No access to the cluster (check VPN, credentials)

Do not proceed without a working kubectl context.

1. Determine the target namespace:
   - If `$ARGUMENTS` is provided, use it as the namespace (e.g., `genai-vector-store-titan`)
   - If not provided, discover available VS namespaces:
     ```bash
     kubectl get ns | grep genai-vector-store
     ```
     If only one exists, use it. If multiple, list them and ask the user which one to collect from.

2. Discover pods in the namespace:
   ```bash
   kubectl -n <namespace> get pods -o wide
   ```
   Filter out non-VS pods (e.g., `db-tools`, `pgbouncer`, helper pods). VS pods match patterns:
   - `genai-vector-store-*` (service pods)
   - `genai-vector-store-ops-*` (ops pods)
   - `genai-vector-store-background-*` (background pods)

3. Create `tmp/logs/` directory if it does not exist.

4. For each VS pod, collect logs with timestamps:
   ```bash
   kubectl logs <pod-name> -n <namespace> --timestamps > tmp/logs/<pod-name>.log 2>&1
   ```

5. Report a summary: list all log files created, their sizes, and any failures.

## Optional Parameters

If user specifies:
- **since**: Add `--since=<duration>` (e.g., "1h", "30m")
  ```bash
  kubectl logs <pod-name> -n <namespace> --timestamps --since=1h > tmp/logs/<pod-name>.log 2>&1
  ```
- **tail**: Add `--tail=<lines>` (e.g., 1000, 500)
- **service**: Only collect from matching pods (e.g., "just background" -> only `*-background-*` pods)

## Rules
- Logs may contain sensitive data (tokens, credentials, PII). Warn the user before collecting:
  `"Note: logs may contain sensitive data (tokens, credentials). tmp/ is gitignored but treat files with care."`
- Verify `tmp/` is gitignored before collecting: `git check-ignore tmp/ 2>&1`
- Do NOT print log file contents to the conversation - only report file paths and sizes
- Always filter out non-VS pods (db-tools, pgbouncer, etc.) - only collect from VS service/ops/background pods.
- If namespace doesn't exist or has no pods, report clearly and stop.
- Previous log files with the same name will be overwritten.

## Notes
- Default namespace: `genai-vector-store`
- Namespace with suffix: `genai-vector-store-<suffix>` (e.g., `-titan`, `-saturn`)
- Requires kubectl configured and authenticated with access to the namespace
