Run a performance test against a Vector Store API endpoint - measure latency, throughput, and analyze VS response headers for component-level breakdown.

## Prerequisites / Required Tools

**CLI tools (must be available):**
- `kubectl` - configured with context to the VS cluster (read + port-forward access)
- `sax` CLI (`~/go/bin/sax` or in PATH) - SAX token generation
- `curl` - HTTP requests with timing support (`-w` flag)
- `jq` - JSON parsing
- `awk`, `sort` - stats calculation

**Configuration (auto-loaded from `.claude/test.env` if present):**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SAX_SECRET_ID` | Yes | - | SAX secret ID for service token |
| `SAX_OPS_SECRET_ID` | Self-Contained only | - | SAX secret ID for ops token (isolation create/delete) |
| `VS_NAMESPACE` | No | `genai-vector-store` | K8s namespace |
| `VS_REPORT_DIR` | No | `tmp/reports` | Directory for perf test reports |

> First-time setup: `cp .claude/test.env.example .claude/test.env` and fill in SAX secret IDs. After that, all skills pick them up automatically.

## VS Response Headers Reference

VS returns per-request component timing in response headers. These are the source of truth for latency breakdown.

### VS timing headers

| Header | Unit | Description |
|--------|------|-------------|
| `X-Genai-Vectorstore-Request-Duration-Ms` | ms | Total server-side request duration |
| `X-Genai-Vectorstore-Processing-Duration-Ms` | ms | Application-level processing time |
| `X-Genai-Vectorstore-Overhead-Ms` | ms | Internal overhead (request - processing) |
| `X-Genai-Vectorstore-Db-Query-Time-Ms` | ms | Database query time (pgvector search) |
| `X-Genai-Vectorstore-Embedding-Time-Ms` | ms | Total time calling embedding provider |
| `X-Genai-Vectorstore-Embedding-Net-Overhead-Ms` | ms | Network overhead for embedding calls |
| `X-Genai-Vectorstore-Embedding-Calls-Count` | count | Number of embedding API calls made |
| `X-Genai-Vectorstore-Embedding-Retry-Count` | count | Number of embedding retries (throttling, errors) |
| `X-Genai-Vectorstore-Documents-Count` | count | Number of documents in collection |
| `X-Genai-Vectorstore-Vectors-Count` | count | Number of vectors searched |
| `X-Genai-Vectorstore-Response-Returned-Items-Count` | count | Number of items returned in response |
| `X-Genai-Vectorstore-Model-Id` | string | Embedding model ID used |
| `X-Genai-Vectorstore-Model-Version` | string | Embedding model version |
| `X-Genai-Vectorstore-Db-Schema-Migration` | string | DB schema migration status |

### Gateway headers (from upstream GenAI Hub)

| Header | Unit | Description |
|--------|------|-------------|
| `X-Genai-Gateway-Response-Time-Ms` | ms | Gateway total response time |
| `X-Genai-Gateway-Model-Id` | string | Model ID used by gateway |
| `X-Genai-Gateway-Region` | string | Region where gateway processed request |
| `X-Genai-Gateway-Input-Tokens` | count | Input tokens consumed |
| `X-Genai-Gateway-Output-Tokens` | count | Output tokens generated |
| `X-Genai-Gateway-Tokens-Per-Second` | count | Token generation rate |
| `X-Genai-Gateway-Retry-Count` | count | Number of gateway retries |

> These headers are present on query endpoints (chunks, documents). Use them for component-level breakdown instead of relying solely on curl timing.

## Steps

### Step 0: Load config and validate prerequisites

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
```
If `SAX_SECRET_ID` is not set (and `.claude/test.env` doesn't exist or is empty), stop and tell the user to set up the config file. Do not proceed without it.

Verify port-forward is active (VS responds):
```bash
HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "http://localhost:8080/v2/probe/collections")
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ]; then
  echo "VS OK (HTTP $HTTP_CODE)"
else
  echo "VS not ready (HTTP $HTTP_CODE) - start port-forward first"
fi
```
> Note: With SAX enabled, probe returns 401 without a token - that's fine, it means the server is alive.

If port-forward is not active, set it up (same as E2E step 0) or tell the user to run `/e2e-test` first.

### Step 1: Configure test parameters

Determine from user input (or use defaults):

| Parameter | Default | Description |
|-----------|---------|-------------|
| Endpoint | `POST /v1/{iso}/collections/{col}/query/chunks` | Target API endpoint |
| Request body | `{"filters": {"query": "What is Pega Infinity?"}, "topK": 3}` | Request payload |
| Iterations | 100 | Number of requests to send |
| Warm-up | 5 | Requests to discard before measuring |
| Concurrency | 1 | Parallel requests (1 = sequential) |
| Isolation ID | (from user or env) | Must exist with data |
| Collection ID | (from user or env) | Must exist with data |

The user must provide an existing isolation + collection with data. If not specified, ask - or use **Self-Contained Mode** (see below).

Set up variables:
```bash
BASE=http://localhost:8080
ISO=<isolation-id>
COLLECTION=<collection-id>
ITERATIONS=100
WARMUP=5
CONCURRENCY=1
```

### Step 2: Prepare curl format and output files

```bash
mkdir -p tmp

# Request body
cat > tmp/vs-perf-request.json << 'EOF'
{"filters": {"query": "What is Pega Infinity?"}, "topK": 3}
EOF

# CSV header - includes VS header columns for per-request component data
echo "iteration,ttfb,total,http_code,size,vs_request_ms,vs_processing_ms,vs_overhead_ms,vs_db_ms,vs_embedding_ms,vs_emb_net_ms,vs_emb_calls,vs_emb_retries,gtw_response_ms,gtw_retries" > tmp/vs-perf-data.csv
```

### Step 3: Warm-up

Run a few requests to prime caches and connections (discard results):
```bash
for i in $(seq 1 $WARMUP); do
  curl -s -o /dev/null -X POST "$BASE/v1/$ISO/collections/$COLLECTION/query/chunks" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d @tmp/vs-perf-request.json
done
echo "Warm-up: $WARMUP requests done"
```

### Step 4: Run test iterations

Each request captures both curl timing AND VS response headers for component breakdown.

**Sequential (concurrency=1):**
```bash
for i in $(seq 1 $ITERATIONS); do
  # Capture full response with headers
  RESPONSE=$(curl -s -i -w '\n__CURL_TIMING__:%{time_starttransfer},%{time_total},%{http_code},%{size_download}' \
    -X POST "$BASE/v1/$ISO/collections/$COLLECTION/query/chunks" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d @tmp/vs-perf-request.json)

  # Extract curl timing
  TIMING=$(echo "$RESPONSE" | grep '__CURL_TIMING__:' | sed 's/__CURL_TIMING__://')

  # Extract VS headers (default to empty if missing)
  vs_request=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Request-Duration-Ms' | awk '{print $2}' | tr -d '\r')
  vs_processing=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Processing-Duration-Ms' | awk '{print $2}' | tr -d '\r')
  vs_overhead=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Overhead-Ms' | awk '{print $2}' | tr -d '\r')
  vs_db=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Db-Query-Time-Ms' | awk '{print $2}' | tr -d '\r')
  vs_emb=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Embedding-Time-Ms:' | awk '{print $2}' | tr -d '\r')
  vs_emb_net=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Embedding-Net-Overhead-Ms' | awk '{print $2}' | tr -d '\r')
  vs_emb_calls=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Embedding-Calls-Count' | awk '{print $2}' | tr -d '\r')
  vs_emb_retries=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Embedding-Retry-Count' | awk '{print $2}' | tr -d '\r')
  gtw_response=$(echo "$RESPONSE" | grep -i 'X-Genai-Gateway-Response-Time-Ms' | awk '{print $2}' | tr -d '\r')
  gtw_retries=$(echo "$RESPONSE" | grep -i 'X-Genai-Gateway-Retry-Count' | awk '{print $2}' | tr -d '\r')

  echo "$i,$TIMING,${vs_request:-},${vs_processing:-},${vs_overhead:-},${vs_db:-},${vs_emb:-},${vs_emb_net:-},${vs_emb_calls:-},${vs_emb_retries:-},${gtw_response:-},${gtw_retries:-}" >> tmp/vs-perf-data.csv

  [ $((i % 10)) -eq 0 ] && echo "Progress: $i/$ITERATIONS"
done
```

**Parallel (concurrency>1):**
```bash
perf_request() {
  local i=$1
  RESPONSE=$(curl -s -i -w '\n__CURL_TIMING__:%{time_starttransfer},%{time_total},%{http_code},%{size_download}' \
    -X POST "$BASE/v1/$ISO/collections/$COLLECTION/query/chunks" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d @tmp/vs-perf-request.json)

  TIMING=$(echo "$RESPONSE" | grep '__CURL_TIMING__:' | sed 's/__CURL_TIMING__://')
  vs_request=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Request-Duration-Ms' | awk '{print $2}' | tr -d '\r')
  vs_processing=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Processing-Duration-Ms' | awk '{print $2}' | tr -d '\r')
  vs_overhead=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Overhead-Ms' | awk '{print $2}' | tr -d '\r')
  vs_db=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Db-Query-Time-Ms' | awk '{print $2}' | tr -d '\r')
  vs_emb=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Embedding-Time-Ms:' | awk '{print $2}' | tr -d '\r')
  vs_emb_net=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Embedding-Net-Overhead-Ms' | awk '{print $2}' | tr -d '\r')
  vs_emb_calls=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Embedding-Calls-Count' | awk '{print $2}' | tr -d '\r')
  vs_emb_retries=$(echo "$RESPONSE" | grep -i 'X-Genai-Vectorstore-Embedding-Retry-Count' | awk '{print $2}' | tr -d '\r')
  gtw_response=$(echo "$RESPONSE" | grep -i 'X-Genai-Gateway-Response-Time-Ms' | awk '{print $2}' | tr -d '\r')
  gtw_retries=$(echo "$RESPONSE" | grep -i 'X-Genai-Gateway-Retry-Count' | awk '{print $2}' | tr -d '\r')

  echo "$i,$TIMING,${vs_request:-},${vs_processing:-},${vs_overhead:-},${vs_db:-},${vs_emb:-},${vs_emb_net:-},${vs_emb_calls:-},${vs_emb_retries:-},${gtw_response:-},${gtw_retries:-}"
}
export -f perf_request
export BASE ISO COLLECTION TOKEN

seq 1 $ITERATIONS | xargs -I{} -P $CONCURRENCY bash -c 'perf_request {}' >> tmp/vs-perf-data.csv
```

Also capture full response headers from one representative request for analysis:
```bash
curl -s -D tmp/vs-perf-sample-headers.txt -o tmp/vs-perf-sample-body.json \
  -X POST "$BASE/v1/$ISO/collections/$COLLECTION/query/chunks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @tmp/vs-perf-request.json
```

### Step 5: Calculate statistics

#### Curl timing stats (TTFB, total)
```bash
# Sort by total time, calculate percentiles
tail -n +2 tmp/vs-perf-data.csv | sort -t',' -k3 -n | awk -F',' '
BEGIN { errors=0 }
{
  ttfb[NR]=$2; total[NR]=$3; code=$4
  ttfb_sum+=$2; total_sum+=$3; count++
  if (code < 200 || code >= 300) errors++
}
END {
  printf "=== Latency (time_total, seconds) ===\n"
  printf "min=%.4f max=%.4f avg=%.4f\n", total[1], total[count], total_sum/count
  printf "p50=%.4f p95=%.4f p99=%.4f\n", total[int(count*0.50)], total[int(count*0.95)], total[int(count*0.99)]
  printf "\n=== Summary ===\n"
  printf "total=%d errors=%d error_rate=%.2f%%\n", count, errors, (errors/count)*100
  printf "throughput=%.1f req/s (sequential estimate)\n", count/total_sum
}'
```

For TTFB percentiles separately:
```bash
tail -n +2 tmp/vs-perf-data.csv | sort -t',' -k2 -n | awk -F',' '
{ ttfb[NR]=$2; sum+=$2; count++ }
END {
  printf "TTFB: min=%.4f max=%.4f avg=%.4f p50=%.4f p95=%.4f p99=%.4f\n",
    ttfb[1], ttfb[count], sum/count, ttfb[int(count*0.50)], ttfb[int(count*0.95)], ttfb[int(count*0.99)]
}'
```

#### Component breakdown from VS headers

Calculate percentiles for each VS timing component. CSV columns: 6=vs_request, 7=vs_processing, 8=vs_overhead, 9=vs_db, 10=vs_embedding, 11=vs_emb_net, 14=gtw_response.

```bash
# Component breakdown - percentiles per component (ms)
for COL_NAME_IDX in "Request-Duration:6" "Processing-Duration:7" "Overhead:8" "DB-Query:9" "Embedding:10" "Embedding-Net:11" "Gateway-Response:14"; do
  NAME="${COL_NAME_IDX%%:*}"
  IDX="${COL_NAME_IDX##*:}"
  echo "=== $NAME (ms) ==="
  tail -n +2 tmp/vs-perf-data.csv | awk -F',' -v idx="$IDX" '$idx != "" { vals[++n]=$idx+0; sum+=$idx+0 }
    END {
      if (n==0) { print "  (no data)"; next }
      # sort vals
      for (i=1; i<=n; i++) for (j=i+1; j<=n; j++) if (vals[i]>vals[j]) { t=vals[i]; vals[i]=vals[j]; vals[j]=t }
      printf "  min=%.1f max=%.1f avg=%.1f p50=%.1f p95=%.1f p99=%.1f (n=%d)\n",
        vals[1], vals[n], sum/n, vals[int(n*0.50)], vals[int(n*0.95)], vals[int(n*0.99)], n
    }'
done
```

### Step 6: Analyze response headers

Examine the sample response headers for VS-specific and gateway timing:
```bash
echo "=== VS Headers ==="
grep -i 'X-Genai-Vectorstore-' tmp/vs-perf-sample-headers.txt
echo ""
echo "=== Gateway Headers ==="
grep -i 'X-Genai-Gateway-' tmp/vs-perf-sample-headers.txt
```

Key things to look for:
- **Embedding retries > 0** - throttling by embedding provider, potential bottleneck
- **Gateway retry count > 0** - gateway-level retries, potential instability
- **DB query time >> embedding time** - DB is the bottleneck (index quality, data size)
- **Embedding time >> DB query time** - embedding provider is the bottleneck
- **Overhead >> 0** - internal VS overhead (auth, middleware, serialization)
- **Embedding-Net-Overhead high** - network latency to embedding provider

### Step 7: Generate report

```bash
REPORT_DIR="${VS_REPORT_DIR:-tmp/reports}"
mkdir -p "$REPORT_DIR"
```

**File:** `$REPORT_DIR/vs-perf-report-<deployment>-<YYYY-MM-DD>-<N>.md`

```markdown
# Vector Store Performance Test Report

**Date:** <YYYY-MM-DD HH:MM>
**Environment:** <deployment-name>
**Namespace:** <namespace>
**VS Version:** <version>
**Isolation ID:** <ID>
**Collection:** <collection-id>
**Mode:** standard | self-contained

## Test Configuration

| Parameter | Value |
|-----------|-------|
| Endpoint | POST /v1/{iso}/collections/{col}/query/chunks |
| Query | "What is Pega Infinity?" |
| Iterations | 100 |
| Warm-up | 5 |
| Concurrency | 1 |

## Summary

| Metric | Value |
|--------|-------|
| Total requests | 100 |
| Successful (2xx) | 100 |
| Failed | 0 |
| Error rate | 0.00% |
| Throughput | X.X req/s |

## Latency (time_total)

| Stat | Value (s) |
|------|-----------|
| min | |
| max | |
| avg | |
| p50 | |
| p95 | |
| p99 | |

## TTFB (time_starttransfer)

| Stat | Value (s) |
|------|-----------|
| min | |
| max | |
| avg | |
| p50 | |
| p95 | |
| p99 | |

## Component Breakdown (from VS headers, ms)

| Component | min | avg | p50 | p95 | p99 | max |
|-----------|-----|-----|-----|-----|-----|-----|
| Request-Duration | | | | | | |
| Processing-Duration | | | | | | |
| Overhead | | | | | | |
| DB-Query | | | | | | |
| Embedding | | | | | | |
| Embedding-Net | | | | | | |
| Gateway-Response | | | | | | |

### Bottleneck Analysis

- **Dominant component:** (embedding / db / overhead)
- **Embedding retries:** X (throttling indicator)
- **Gateway retries:** X
- **Embedding model:** <model-id> <model-version>

## Response Headers (sample)

(Full X-Genai-* headers from one representative request)

## Raw Data

CSV data: `tmp/vs-perf-data.csv`
Sample headers: `tmp/vs-perf-sample-headers.txt`
Sample body: `tmp/vs-perf-sample-body.json`
```

## Self-Contained Mode

When the user does NOT provide an existing isolation + collection, the perf test can set up its own test data, run the test, and clean up afterward.

**Trigger:** User asks for perf test but has no existing isolation/collection, or explicitly asks for "self-contained" / "standalone" mode.

**Additional prerequisites:** `SAX_OPS_SECRET_ID` must be set (for isolation create/delete via ops API). On environments with `ISOLATION_AUTO_CREATION=true`, ops token is not needed - isolation is created automatically on first upsert.

### Self-Contained Steps

**Setup phase:**
1. Generate isolation ID: `ISO=$(uuidgen | tr '[:upper:]' '[:lower:]')`
2. Create isolation via ops API (unless `ISOLATION_AUTO_CREATION=true`):
   ```bash
   curl -s -X POST "$OPS/v1/isolations" \
     -H "Authorization: Bearer $OPS_TOKEN" \
     -H "Content-Type: application/json" \
     -d "{\"isolationId\": \"$ISO\", \"maxStorageSize\": 1073741824}"
   ```
3. Create collection:
   ```bash
   curl -s -X PUT "$BASE/v1/$ISO/collections/perf-test-collection" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"description": "perf test temp collection"}'
   ```
4. Upsert test document (use standard E2E test document from `vs-test-data.md`):
   ```bash
   curl -s -X PUT "$BASE/v1/$ISO/collections/perf-test-collection/documents?consistencyLevel=eventual" \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d @tmp/vs-perf-test-doc.json
   ```
5. Poll document status until COMPLETED (max 60s)
6. Set `COLLECTION=perf-test-collection` and continue with normal Steps 2-7

**Cleanup phase (after report):**
```bash
# Delete isolation (removes all data)
curl -s -X DELETE "$OPS/v1/isolations/$ISO" \
  -H "Authorization: Bearer $OPS_TOKEN"
echo "Cleanup: isolation $ISO deleted"
```

> Always clean up self-contained resources. If cleanup fails, report the isolation ID so user can clean up manually.

## Supported Endpoints

Any VS API endpoint can be tested. Common targets:

| Endpoint | Method | Use case |
|----------|--------|----------|
| `/v1/{iso}/collections/{col}/query/chunks` | POST | Semantic search - chunk-level |
| `/v1/{iso}/collections/{col}/query/documents` | POST | Semantic search - document-level |
| `/v2/{iso}/collections/{col}/documents/{doc}/chunks` | GET | Get document chunks |
| `/v2/{iso}/collections` | GET | List collections |
| `/v1/{iso}/collections/{col}/documents?consistencyLevel=eventual` | PUT | Upsert (write perf) |

For write endpoints (upsert), each iteration should use a unique document ID to avoid conflicts:
```bash
DOC_ID="perf-test-DOC-$i"
```

## Comparing Results

The user may ask to compare before/after (e.g., before and after upgrade). For this:
1. Run perf test before change, save report with label (e.g., `vs-perf-report-BEFORE-*.md`)
2. Run perf test after change, save report with label (e.g., `vs-perf-report-AFTER-*.md`)
3. Compare key metrics side by side in a summary table

```markdown
## Comparison: BEFORE vs AFTER

| Metric | BEFORE | AFTER | Change |
|--------|--------|-------|--------|
| avg latency | X.XXXs | X.XXXs | +/-XX% |
| p95 latency | X.XXXs | X.XXXs | +/-XX% |
| p99 latency | X.XXXs | X.XXXs | +/-XX% |
| avg embedding (ms) | X.X | X.X | +/-XX% |
| avg db query (ms) | X.X | X.X | +/-XX% |
| avg overhead (ms) | X.X | X.X | +/-XX% |
| throughput | X req/s | X req/s | +/-XX% |
| error rate | X% | X% | |
```

## Rules
- Before starting, verify that required env vars (`SAX_SECRET_ID`) are set. If missing, stop and tell the user what to set with an example value. Do not proceed without them.
- The target isolation and collection must exist and contain data. If they don't, offer Self-Contained Mode or tell the user to run `/e2e-test` first (skip cleanup).
- Safety: for write endpoints, use `perf-test-` prefix on all created resources.
- Cleanup `tmp/vs-perf-*` files after the report is generated (keep only the report and CSV).
- Token may expire during long runs (>1h). If 401 errors spike, refresh the token mid-test.
- Report always in English.
- Do not modify existing data in the isolation - perf tests should be non-destructive (read endpoints) or use dedicated test resources (write endpoints).

## Notes
- Port-forward introduces network overhead - measured latency includes the kubectl tunnel. For absolute numbers, compare with direct pod-to-pod or service-to-service calls.
- Sequential throughput is limited by round-trip time. Use concurrency > 1 to measure server capacity under load.
- Long runs (>1000 iterations) may hit SAX token expiry (~1h). Monitor for 401 responses and refresh token if needed.
- For write perf tests, document processing is async - the measured latency is only the upsert acceptance time, not end-to-end processing.
- SAX credentials (AWS-issued) work on both AWS and GCP clusters - the same Okta staging issuer is used across cloud providers.
- The VS response headers give the real server-side breakdown. curl timing includes port-forward overhead, so always prefer VS headers for component analysis.
