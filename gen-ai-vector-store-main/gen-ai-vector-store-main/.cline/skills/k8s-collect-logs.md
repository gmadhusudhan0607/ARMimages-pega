# Collect Kubernetes Pod Logs

## Description
Collects logs from Vector Store pods in a given namespace and saves them to files in the `tmp/logs/` directory.

## Trigger
When user asks to:
- "collect pod logs"
- "get kubernetes logs"
- "gather logs from VS pods"
- "fetch logs from vector store"
- "collect logs from genai-vector-store"

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

1. **Determine target namespace**
   - If user specified a namespace, use it directly (e.g., `genai-vector-store-titan`)
   - Otherwise discover available VS namespaces:
     ```bash
     kubectl get ns | grep genai-vector-store
     ```
     If only one exists, use it. If multiple, list them and ask which one.

2. **Discover pods in the namespace**
   ```bash
   kubectl -n <namespace> get pods -o wide
   ```
   Filter to VS pods only (skip `db-tools`, `pgbouncer`, helper pods). VS pod patterns:
   - `genai-vector-store-*` (service)
   - `genai-vector-store-ops-*` (ops)
   - `genai-vector-store-background-*` (background)

3. **Create `tmp/logs/` directory** if it doesn't exist

4. **For each VS pod, collect logs with timestamps**
   ```bash
   kubectl logs <pod-name> -n <namespace> --timestamps > tmp/logs/<pod-name>.log 2>&1
   ```

5. **Report summary** - list all log files created, sizes, and any failures

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

## Notes
- Default namespace: `genai-vector-store`
- Namespace with suffix: `genai-vector-store-<suffix>` (e.g., `-titan`, `-saturn`)
- Previous log files with same name will be overwritten
- Always filter out non-VS pods (db-tools, pgbouncer, etc.)
- Requires kubectl configured and authenticated with access to the namespace
