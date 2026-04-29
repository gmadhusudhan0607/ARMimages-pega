Run a full E2E test of the Vector Store API on a live environment - from isolation creation through document upsert, query, and cleanup.

## Prerequisites / Required Tools

**CLI tools (must be available):**
- `kubectl` - configured with context to the VS cluster (read + port-forward access)
- `sax` CLI (`~/go/bin/sax` or in PATH) - SAX token generation
- `curl` - HTTP requests
- `jq` - JSON parsing
- `uuidgen` - isolation ID generation

**Configuration (auto-loaded from `.claude/test.env` if present):**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SAX_SECRET_ID` | Yes | - | SAX secret ID for service token (`sax/backing-services/<guid>`) |
| `SAX_OPS_SECRET_ID` | Yes (full E2E) | - | SAX secret ID for ops token - required for isolation create/delete (steps 1b, 10). Different secret than service. |
| `VS_NAMESPACE` | No | `genai-vector-store` | K8s namespace (may have suffix, e.g. `-titan`) |
| `VS_REPORT_DIR` | No | `tmp/reports` | Directory for E2E test reports (repo-relative or absolute) |

> First-time setup: `cp .claude/test.env.example .claude/test.env` and fill in SAX secret IDs.

## Steps

### Step 0: Load config, validate prerequisites and setup

Load configuration from `.claude/test.env` (if exists), then check required values:
```bash
# Auto-load config
if [ -f .claude/test.env ]; then
  set -a; source .claude/test.env; set +a
  echo "Loaded .claude/test.env"
fi

if [ -z "$SAX_SECRET_ID" ]; then
  echo "ERROR: SAX_SECRET_ID is not set."
  echo "  Setup: cp .claude/test.env.example .claude/test.env && edit .claude/test.env"
fi
if [ -z "$SAX_OPS_SECRET_ID" ]; then
  echo "WARNING: SAX_OPS_SECRET_ID is not set - isolation create/delete (steps 1b, 10) will fail."
fi
```
If `SAX_SECRET_ID` is not set (and `.claude/test.env` doesn't exist or is empty), stop and tell the user to set up the config file. Do not proceed without it.

1. Set namespace:
   ```bash
   NS="${VS_NAMESPACE:-genai-vector-store}"
   ```

2. Port-forward to VS pods:
   ```bash
   # Filter by pod name pattern - label selectors may match db-tools or other helper pods
   VS_POD=$(kubectl get pods -n $NS --no-headers -o name | grep "pod/genai-vector-store-[a-z0-9]*-[a-z0-9]*$" | head -1)
   OPS_POD=$(kubectl get pods -n $NS --no-headers -o name | grep "pod/genai-vector-store-ops" | head -1)
   echo "VS_POD=$VS_POD OPS_POD=$OPS_POD"
   kubectl port-forward -n $NS $VS_POD 8080:8080 >/dev/null 2>&1 &
   kubectl port-forward -n $NS $OPS_POD 8081:8080 >/dev/null 2>&1 &
   sleep 3
   ```

3. Generate SAX token:
   ```bash
   TOKEN=$(sax issue --secret-id "$SAX_SECRET_ID" 2>&1 | grep -A1 "Access Token:" | tail -1 | tr -d '[:space:]')
   ```
   If `SAX_OPS_SECRET_ID` is set:
   ```bash
   OPS_TOKEN=$(sax issue --secret-id "$SAX_OPS_SECRET_ID" 2>&1 | grep -A1 "Access Token:" | tail -1 | tr -d '[:space:]')
   ```
   Otherwise ops may not require auth (SAX disabled on ops).

4. Generate isolation ID:
   ```bash
   ISO=$(uuidgen | tr '[:upper:]' '[:lower:]')
   ```

5. Verify connectivity:
   ```bash
   curl -s --max-time 5 "http://localhost:8080/v2/probe/collections" 2>&1 | grep -q "isolation" && echo "VS OK" || echo "VS not ready"
   ```

6. Set base URLs:
   ```bash
   BASE=http://localhost:8080
   OPS=http://localhost:8081
   ```

**Important notes:**
- `environmentguid` (from pega-web labels) is NOT the same as isolation ID (`CUSTOMER_DEPLOYMENT_ID` from configmap). The SAX token contains the real `guid` claim that VS validates.
- Verify isolation ID from token: `echo "$TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | jq -r '.guid'`
- Service and ops use **different SAX secrets** - they are separate services with separate auth.
- SAX on ops may be disabled on some environments - in that case ops API does not require a token.
- Never inline long JSON in curl. Always write to `tmp/vs-*.json` (repo-relative) and use `curl -d @tmp/file.json`.
- **Optional - full response logging:** If the user asks to save raw responses (e.g., "save full responses", "I need headers"), use `curl -D tmp/vs-resp-step-{N}-headers.txt` and save body to `tmp/vs-resp-step-{N}-body.json`. Do not do this by default.

### Step 1: Check if isolation exists

```bash
curl -s $BASE/v2/$ISO/collections -H "Authorization: Bearer $TOKEN" | jq .
```
- **200** - isolation exists, proceed to step 2
- **404 "isolation not found"** - create it in step 1b
- **401** - token expired, refresh
- **403** - isolation ID mismatch with token

### Step 1b: Create isolation (only if step 1 returned 404)

```bash
curl -s -X POST $OPS/v1/isolations \
  -H "Authorization: Bearer $OPS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"id\": \"$ISO\", \"maxStorageSize\": \"1GB\"}" | jq .
```
Expected: **201 Created**

Track whether you created the isolation (needed for cleanup decision in step 10).

### Step 2: Create collection

```bash
COLLECTION="cline-test-collection"

curl -s -X POST $BASE/v2/$ISO/collections \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"collectionID\": \"$COLLECTION\"}" | jq .
```
Expected: **201 Created**

### Step 3: Upsert document

Write the test document to a temp file, then upsert:

```bash
DOC_ID="cline-test-DOC-1"

mkdir -p tmp
cat > tmp/vs-e2e-doc.json << 'DOCEOF'
{
  "id": "cline-test-DOC-1",
  "chunks": [
    {
      "content": "A report definition is a tool used in database interactions for retrieving required details based on certain conditions.",
      "attributes": [
        {"name": "section", "type": "string", "value": ["overview"]}
      ]
    },
    {
      "content": "Pega Infinity is a low-code platform that enables organizations to build and deploy enterprise applications quickly.",
      "attributes": [
        {"name": "section", "type": "string", "value": ["introduction"]}
      ]
    }
  ],
  "attributes": [
    {"name": "version", "type": "string", "value": ["8.8", "24.1"]},
    {"name": "category", "type": "string", "value": ["documentation"]}
  ]
}
DOCEOF

curl -s -X PUT "$BASE/v1/$ISO/collections/$COLLECTION/documents?consistencyLevel=eventual" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @tmp/vs-e2e-doc.json | jq .
```
Expected: **202 Accepted** (async processing)

### Step 4: Poll document status

Poll every 5s, timeout after 60s (12 attempts):

```bash
for i in $(seq 1 12); do
  RESPONSE=$(curl -s "$BASE/v1/$ISO/collections/$COLLECTION/documents/$DOC_ID" \
    -H "Authorization: Bearer $TOKEN")
  STATUS=$(echo "$RESPONSE" | jq -r '.status')
  echo "Poll $i: status=$STATUS"
  [ "$STATUS" = "COMPLETED" ] && break
  [ "$STATUS" = "ERROR" ] && echo "ERROR: $(echo "$RESPONSE" | jq -r '.errorMessage')" && break
  sleep 5
done
```
- **COMPLETED** - proceed to step 5
- **ERROR** - check `.errorMessage`, mark as FAIL
- **Timeout** - mark as FAIL, check background pod logs

### Step 5: Query chunks (semantic search)

```bash
curl -s -X POST "$BASE/v1/$ISO/collections/$COLLECTION/query/chunks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"filters": {"query": "What is Pega Infinity?"}, "topK": 3}' | jq '.[] | {content: .content[:80], distance}'
```
Expected: **200** with results. The chunk about "Pega Infinity is a low-code platform..." should have the lowest distance (best match).

### Step 6: Query documents

```bash
curl -s -X POST "$BASE/v1/$ISO/collections/$COLLECTION/query/documents" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"filters": {"query": "report definition"}, "topK": 3}' | jq '.[] | {documentID, distance}'
```
Expected: **200** with the test document matched.

### Step 7: Get document chunks (v2)

```bash
curl -s "$BASE/v2/$ISO/collections/$COLLECTION/documents/$DOC_ID/chunks" \
  -H "Authorization: Bearer $TOKEN" | jq '{totalChunks: (.chunks | length), chunks: [.chunks[] | {content: .content[:60]}]}'
```
Expected: **200** with 2 chunks.

### Step 8: Delete document

```bash
curl -s -X DELETE "$BASE/v1/$ISO/collections/$COLLECTION/documents/$DOC_ID" \
  -H "Authorization: Bearer $TOKEN" | jq .
```
Expected: **200** or **204**

### Step 9: Delete collection

```bash
curl -s -X DELETE "$BASE/v2/$ISO/collections/$COLLECTION" \
  -H "Authorization: Bearer $TOKEN" | jq .
```
Expected: **200** or **204**

### Step 10: Delete isolation (only if you created it in step 1b)

```bash
curl -s -X DELETE "$OPS/v1/isolations/$ISO" \
  -H "Authorization: Bearer $OPS_TOKEN" 2>&1
```
Expected: **200** or **204**

If the isolation existed before the test (step 1 returned 200) - do NOT delete it.

## Custom Scenarios

When the user requests non-standard tests (e.g., "upload 15 documents", "use 3 collections"), adapt the default flow:

**Parameters the user may specify:**
- Number of documents (e.g., "upload 15 documents")
- Content type (lorem ipsum, random text, custom content)
- Interval between documents (e.g., "every 10 seconds")
- Number of collections (e.g., "3 collections with 5 documents each")
- Custom query text

**Naming conventions:**
- Documents: `cline-test-DOC-{N}` (N = 1, 2, 3, ...)
- Collections: `cline-test-collection` (single) or `cline-test-collection-{M}` (multiple)
- Generate content as needed (lorem ipsum, documentation fragments, whatever the user wants)

**Flow per document:**
- Upsert -> poll status -> verify (or skip verify if user wants speed)
- Delay between documents: `sleep $INTERVAL` (if specified)

**Tracking:**
- Maintain a table per document in the report: doc ID, status, processing time

**Cleanup:**
- Delete ONLY resources with `cline-test-` prefix
- Loop over all created documents and collections

## Report

**By default:** print a summary table with verdict directly in the conversation. No file is saved unless the user asks.

**If the user asks to save a report** (e.g., "save report", "write report to file"):

**Directory:** `${VS_REPORT_DIR:-tmp/reports}` (repo-relative by default, create if missing).
**File:** `vs-e2e-report-<deployment>-<YYYY-MM-DD>-<N>.md`
where `<N>` is a sequence number (01, 02, ...). Check existing files to avoid overwriting.
```bash
REPORT_DIR="${VS_REPORT_DIR:-tmp/reports}"
mkdir -p "$REPORT_DIR"
```

**If called from upgrade validation (step 2.5):** do NOT generate a separate E2E report. Results go into the upgrade report.

**Report template (when saved to file):**

```markdown
# Vector Store E2E Test Report

**Date:** <YYYY-MM-DD HH:MM>
**Environment:** <deployment-name>
**Cluster:** <cluster-name>
**Namespace:** <namespace>
**VS Version:** <version>
**Isolation ID:** <ID>

## Results

| Step | Operation | Status | HTTP | Details |
|------|-----------|--------|------|---------|
| 1 | Check/Create Isolation | PASS/FAIL | 200/201 | |
| 2 | Create Collection (`cline-test-collection`) | PASS/FAIL | 201 | |
| 3 | Upsert Document (`cline-test-DOC-1`) | PASS/FAIL | 202 | |
| 4 | Document Processing -> COMPLETED | PASS/FAIL | | Polling: Xa over Ys |
| 5 | Query Chunks ("What is Pega Infinity?") | PASS/FAIL | 200 | |
| 6 | Query Documents ("report definition") | PASS/FAIL | 200 | |
| 7 | Get Document Chunks v2 | PASS/FAIL | 200 | |
| 8 | Delete Document | PASS/FAIL | 200/204 | |
| 9 | Delete Collection | PASS/FAIL | 200/204 | |

## Verdict: PASS / FAIL

## Step Details

(For each step: full request with method, URL, headers, body + full response with status code, headers, body.
Mask token as `<REDACTED>` in Authorization header.
For polling: show only summary + last poll request/response.)

## Errors (if any)
```

For custom scenarios: extend the results table with N document rows.

## Error Handling

| HTTP Status | Meaning | Action |
|-------------|---------|--------|
| 401 Unauthorized | Token expired or invalid | Refresh SAX token and retry |
| 403 Forbidden | Isolation ID mismatch with token | Verify isolation ID matches token's `guid` claim |
| 404 Not Found | Resource doesn't exist | Check isolation/collection/document ID |
| 500 Internal Server Error | Server error | Check VS logs, retry once |

| Document Status | Meaning | Action |
|-----------------|---------|--------|
| PROCESSING (timeout) | Stuck in queue | Check background pod logs, embedding queue |
| ERROR | Processing failed | Check `.errorMessage`, check embedding provider |

**Troubleshooting on failure:**
1. Check background pod logs: `kubectl logs deployment/genai-vector-store-background -n $NS --tail=50`
2. Check embedding queue (DB): `SELECT COUNT(*) FROM vector_store.embedding_queue;`
3. Check document error: `curl -s "$BASE/v1/$ISO/collections/$COLLECTION/documents/$DOC_ID" -H "Authorization: Bearer $TOKEN" | jq '.errorMessage'`

## Rules
- Before starting, verify that required env vars (`SAX_SECRET_ID`) are set. If missing, stop and tell the user what to set with an example value. Do not proceed without them.
- Safety: prefix ALL test resources with `cline-test-`. Delete ONLY resources with this prefix.
- If the isolation existed before the test - do NOT delete it in cleanup.
- JSON body: always write to `tmp/vs-*.json` (repo-relative), use `curl -d @tmp/file.json`.
- Cleanup `tmp/vs-*` files after completion (if standalone, not if called from upgrade validation).
- Kill port-forward processes after completion.
- Report always in English.
- Use `curl -i` or `-v` to capture response headers for the report.
