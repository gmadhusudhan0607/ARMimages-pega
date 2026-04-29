# Vector Store Diagnostics

## Description
Diagnoses Vector Store issues by checking isolations via API, inspecting database state, reviewing document processing status, and verifying SCE configuration.

## Trigger
When user asks to:
- "diagnose VS"
- "check isolation in VS"
- "why is isolation missing"
- "check VS database"
- "documents stuck in processing"
- "troubleshoot Vector Store"
- "VS diagnostics"

## Prerequisites / Required Tools

**CLI tools:**
- `kubectl` - configured with context to the VS cluster (required)
- `psql` - local PostgreSQL client (optional, for DB proxy access)
- `jq` - JSON parsing (optional)
- `aws` CLI - to retrieve DB credentials from AWS Secrets Manager (optional, AWS clusters)
- `gcloud` CLI - to retrieve DB credentials from GCP Secret Manager (optional, GCP clusters)
- `pegacloud` CLI - to check SCE products on deployment (optional)

**Port-forward must be active** (from E2E setup or manually):
- Service: `localhost:8080` -> VS service pod
- Ops: `localhost:8081` -> VS ops pod

**Configuration (auto-loaded from `.claude/test.env` if present):**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SAX_SECRET_ID` | For API checks | - | SAX secret ID for service token (auto-generates TOKEN) |
| `SAX_OPS_SECRET_ID` | For ops API checks | - | SAX secret ID for ops token (auto-generates OPS_TOKEN) |
| `VS_NAMESPACE` | No | `genai-vector-store` | K8s namespace |

> First-time setup: `cp .claude/test.env.example .claude/test.env` and fill in SAX secret IDs.

## Steps

### Step 0: Load config and validate prerequisites

Load configuration from `.claude/test.env` (if exists), then generate tokens as needed:
```bash
# Auto-load config
if [ -f .claude/test.env ]; then
  set -a; source .claude/test.env; set +a
  echo "Loaded .claude/test.env"
fi

# Generate tokens from SAX secret IDs if TOKEN/OPS_TOKEN not already set
if [ -z "$TOKEN" ] && [ -n "$SAX_SECRET_ID" ]; then
  TOKEN=$(sax issue --secret-id "$SAX_SECRET_ID" 2>&1 | grep -A1 "Access Token:" | tail -1 | tr -d '[:space:]')
fi
if [ -z "$OPS_TOKEN" ] && [ -n "$SAX_OPS_SECRET_ID" ]; then
  OPS_TOKEN=$(sax issue --secret-id "$SAX_OPS_SECRET_ID" 2>&1 | grep -A1 "Access Token:" | tail -1 | tr -d '[:space:]')
fi

# Report status
[ -n "$TOKEN" ] && echo "TOKEN: ready" || echo "WARNING: TOKEN not available - service API checks will fail"
[ -n "$OPS_TOKEN" ] && echo "OPS_TOKEN: ready" || echo "WARNING: OPS_TOKEN not available - ops API checks will fail"
echo "DB-only diagnostics (steps 2-4) can still work without tokens."
```
If tokens can't be generated (no SAX secret IDs configured), tell the user to set up `.claude/test.env`. DB-only diagnostics can proceed without tokens.

### Step 1: Check isolation via API

```bash
# Check collections (service API)
curl -s http://localhost:8080/v2/<ISOLATION_ID>/collections \
  -H "Authorization: Bearer $TOKEN" | jq .

# Check isolation details (ops API - uses separate token)
curl -s http://localhost:8081/v1/isolations/<ISOLATION_ID> \
  -H "Authorization: Bearer $OPS_TOKEN" | jq .
```
- **200 + collections list** - isolation exists, OK
- **404 "isolation not found"** - isolation does not exist in VS database

### Step 2: Database access

Determine which DB access method is available:

```bash
NS="${VS_NAMESPACE:-genai-vector-store}"

# Check if DB proxy is enabled on VS
DB_PROXY=$(kubectl get deployment genai-vector-store -n $NS \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ENABLE_DB_PROXY")].value}' 2>/dev/null)

# Check if db-tools pod exists
DB_TOOLS=$(kubectl get deployment db-tools -n $NS --no-headers 2>/dev/null | wc -l | tr -d ' ')

echo "DB_PROXY=$DB_PROXY  DB_TOOLS=$DB_TOOLS"
```

| DB_PROXY | db-tools | Method |
|----------|----------|--------|
| `true` | any | **Option A: Port-forward + psql** (best) |
| empty/false | >=1 | **Option B: kubectl exec db-tools** |
| empty/false | 0 | **Option C: Set ENABLE_DB_PROXY** (requires pod restart!) |

#### Option A: Port-forward + psql (when ENABLE_DB_PROXY=true)

```bash
kubectl port-forward deployment/genai-vector-store -n $NS 35432:35432 &
```

Detect cloud provider and fetch DB credentials accordingly:
```bash
# Detect cloud provider from deployment env
CLOUD_PROVIDER=$(kubectl get deployment genai-vector-store -n $NS \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="CLOUD_PROVIDER")].value}' 2>/dev/null)

DB_INSTANCE=$(kubectl get deployment genai-vector-store-ops -n $NS \
  -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="DB_INSTANCE")].value}')

if [ "$CLOUD_PROVIDER" = "gcp" ]; then
  # GCP: credentials in Secret Manager
  GCP_PROJECT=$(kubectl get deployment genai-vector-store -n $NS \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="GCP_PROJECT_ID")].value}' 2>/dev/null)
  gcloud secrets versions access latest \
    --secret="/dbms/$DB_INSTANCE/mastercredentials" \
    --project="$GCP_PROJECT" | jq -r '.username, .password'
else
  # AWS (default): credentials in Secrets Manager
  aws secretsmanager get-secret-value --profile ai \
    --secret-id "/dbms/$DB_INSTANCE/mastercredentials" \
    --query 'SecretString' --output text | jq -r '.username, .password'
fi
```

```bash
psql -h localhost -p 35432 -U <user> -d pega
```

#### Option B: db-tools (when db-tools pod exists)

```bash
kubectl exec deployment/db-tools -n $NS -- \
  ./psql -c "SELECT iso_id, iso_prefix, max_storage_size, created_at FROM vector_store.isolations;"
```

Note: `psql` is in the current directory (`./psql`), not in PATH.

#### Option C: Set ENABLE_DB_PROXY (last resort)

> WARNING: `kubectl set env` causes a pod restart! This interrupts active jobs (document processing, embedding queue). Requires explicit user confirmation before proceeding.

Set on the service deployment (not background - background processes documents):

```bash
# 1. Enable proxy (restarts pod!)
kubectl set env deployment/genai-vector-store -n $NS ENABLE_DB_PROXY=true

# 2. Wait for restart
kubectl rollout status deployment/genai-vector-store -n $NS --timeout=120s

# 3. Port-forward
kubectl port-forward deployment/genai-vector-store -n $NS 35432:35432 &

# 4. Connect
psql -h localhost -p 35432 -U <user> -d pega
```

**After diagnostics - revert:**
```bash
kubectl set env deployment/genai-vector-store -n $NS ENABLE_DB_PROXY-
```

### Step 3: DB Schema reference

VS uses per-isolation schemas with hashed prefixes. There is no single `collections` or `documents` table.

```
vector_store.isolations              - iso_id, iso_prefix (maps isolation UUID to schema prefix)
vector_store.embedding_queue         - embedding processing queue
vector_store.configuration           - global configuration

vector_store_<iso_prefix>.collections         - col_id, col_prefix, default_emb_profile
vector_store_<iso_prefix>.emb_profiles        - embedding profiles
vector_store_<iso_prefix>.t_<col_prefix>_doc  - documents per collection
vector_store_<iso_prefix>.t_<col_prefix>_emb  - embeddings per collection
vector_store_<iso_prefix>.t_<col_prefix>_attr - attributes per collection
vector_store_<iso_prefix>.t_<col_prefix>_doc_meta       - document metadata
vector_store_<iso_prefix>.t_<col_prefix>_doc_processing  - document processing status
vector_store_<iso_prefix>.t_<col_prefix>_emb_meta       - embedding metadata
vector_store_<iso_prefix>.t_<col_prefix>_emb_processing  - embedding processing status
```

### Step 4: Useful SQL queries

All queries are SELECT only - no data modification.

```sql
-- All isolations
SELECT iso_id, iso_prefix, max_storage_size, created_at, modified_at
FROM vector_store.isolations;

-- Embedding queue size (how many waiting for processing)
SELECT COUNT(*) AS queue_size FROM vector_store.embedding_queue;

-- Schemas per isolation
SELECT schema_name FROM information_schema.schemata
WHERE schema_name LIKE 'vector_store_%' ORDER BY schema_name;

-- Collections in an isolation (replace <ISO_PREFIX>)
SELECT col_id, col_prefix, default_emb_profile, record_timestamp
FROM vector_store_<ISO_PREFIX>.collections;

-- Documents in a collection (replace <ISO_PREFIX> and <COL_PREFIX>)
SELECT * FROM vector_store_<ISO_PREFIX>.t_<COL_PREFIX>_doc LIMIT 20;

-- Embedding count in a collection
SELECT COUNT(*) AS emb_count FROM vector_store_<ISO_PREFIX>.t_<COL_PREFIX>_emb;

-- Document attributes
SELECT * FROM vector_store_<ISO_PREFIX>.t_<COL_PREFIX>_attr LIMIT 20;

-- Data size per isolation (approximate)
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) AS size
FROM pg_tables
WHERE schemaname LIKE 'vector_store_%'
ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC
LIMIT 20;
```

**Script: count documents across all collections in an isolation:**

```bash
ISO_PREFIX=<iso_prefix>

for COL_PREFIX in $(kubectl exec deployment/db-tools -n $NS -- \
  ./psql -t -c "SELECT col_prefix FROM vector_store_$ISO_PREFIX.collections;" | tr -d ' '); do

  DOC_COUNT=$(kubectl exec deployment/db-tools -n $NS -- \
    ./psql -t -c "SELECT COUNT(*) FROM vector_store_$ISO_PREFIX.t_${COL_PREFIX}_doc;" | tr -d ' ')

  EMB_COUNT=$(kubectl exec deployment/db-tools -n $NS -- \
    ./psql -t -c "SELECT COUNT(*) FROM vector_store_$ISO_PREFIX.t_${COL_PREFIX}_emb;" | tr -d ' ')

  echo "$COL_PREFIX: $DOC_COUNT docs, $EMB_COUNT embeddings"
done
```

### Step 5: Check SCE (if isolation should exist but doesn't)

```bash
pegacloud describe deployment <DEPLOYMENT> -o json | \
  jq '[.resources | to_entries[] | .value[] | select(.productName != null) | {productName, guid}]'
```

Look for:
- `GenAIVectorStore` - the VS infrastructure service itself
- `GenAIVectorStoreIsolation` - the SCE that creates isolations (if missing, isolations won't be created automatically)

### Step 6: Quick Performance Check

Quick latency check - send a few query requests and analyze VS response headers to identify bottleneck. This is a lightweight version of the full perf-test skill - useful during diagnostics to quickly spot if the problem is embedding, DB, or overhead.

**Requires:** port-forward active + `$TOKEN` set + existing isolation/collection with data.

```bash
ISO=<isolation-id>
COLLECTION=<collection-id>
REQUESTS=10

echo "request,request_ms,processing_ms,overhead_ms,db_ms,embedding_ms,emb_net_ms,emb_calls,gtw_response_ms" > tmp/vs-quick-perf.csv

for i in $(seq 1 $REQUESTS); do
  RESP=$(curl -s -i -X POST "http://localhost:8080/v1/$ISO/collections/$COLLECTION/query/chunks" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"filters": {"query": "test query"}, "topK": 3}')

  vs_req=$(echo "$RESP" | grep -i 'X-Genai-Vectorstore-Request-Duration-Ms' | awk '{print $2}' | tr -d '\r')
  vs_proc=$(echo "$RESP" | grep -i 'X-Genai-Vectorstore-Processing-Duration-Ms' | awk '{print $2}' | tr -d '\r')
  vs_oh=$(echo "$RESP" | grep -i 'X-Genai-Vectorstore-Overhead-Ms' | awk '{print $2}' | tr -d '\r')
  vs_db=$(echo "$RESP" | grep -i 'X-Genai-Vectorstore-Db-Query-Time-Ms' | awk '{print $2}' | tr -d '\r')
  vs_emb=$(echo "$RESP" | grep -i 'X-Genai-Vectorstore-Embedding-Time-Ms:' | awk '{print $2}' | tr -d '\r')
  vs_emb_net=$(echo "$RESP" | grep -i 'X-Genai-Vectorstore-Embedding-Net-Overhead-Ms' | awk '{print $2}' | tr -d '\r')
  vs_emb_calls=$(echo "$RESP" | grep -i 'X-Genai-Vectorstore-Embedding-Calls-Count' | awk '{print $2}' | tr -d '\r')
  gtw_resp=$(echo "$RESP" | grep -i 'X-Genai-Gateway-Response-Time-Ms' | awk '{print $2}' | tr -d '\r')

  echo "$i,${vs_req:-},${vs_proc:-},${vs_oh:-},${vs_db:-},${vs_emb:-},${vs_emb_net:-},${vs_emb_calls:-},${gtw_resp:-}" >> tmp/vs-quick-perf.csv
done

# Mini report - averages
echo "=== Quick Perf Check ($REQUESTS requests) ==="
tail -n +2 tmp/vs-quick-perf.csv | awk -F',' '
{
  if ($2!="") { req+=$2; req_n++ }
  if ($5!="") { db+=$5; db_n++ }
  if ($6!="") { emb+=$6; emb_n++ }
  if ($4!="") { oh+=$4; oh_n++ }
  if ($9!="") { gtw+=$9; gtw_n++ }
}
END {
  if (req_n>0) printf "Request-Duration avg: %.1f ms\n", req/req_n
  if (db_n>0)  printf "DB-Query avg:         %.1f ms\n", db/db_n
  if (emb_n>0) printf "Embedding avg:        %.1f ms\n", emb/emb_n
  if (oh_n>0)  printf "Overhead avg:         %.1f ms\n", oh/oh_n
  if (gtw_n>0) printf "Gateway avg:          %.1f ms\n", gtw/gtw_n
  if (req_n>0 && db_n>0 && emb_n>0)
    printf "Bottleneck: %s\n", (emb/emb_n > db/db_n) ? "EMBEDDING" : "DB"
}'
```

> For a full test with percentiles, comparison mode, and report generation, use the perf-test skill.

## Troubleshooting Table

| Problem | Cause | Solution |
|---------|-------|----------|
| 404 "isolation not found" | Isolation missing from DB | Check if SCE `GenAIVectorStoreIsolation` is installed on the deployment |
| Documents stuck in PROCESSING | Background pod not processing | Check background logs: `kubectl logs deployment/genai-vector-store-background -n $NS --tail=100` |
| Token rejected (401) | Wrong audience or expired token | Decode token: `echo $TOKEN \| cut -d. -f2 \| base64 -d \| jq '{guid, aud, exp}'` |
| guid mismatch | Used environmentguid instead of CUSTOMER_DEPLOYMENT_ID | Verify isolation ID from token (see E2E test setup notes) |
| query returns 400 | Wrong request body format | Use `{"filters": {"query": "..."}, "topK": N}` (not `{"query": "..."}`) |
| find-documents returns 500 | Known bug in some VS versions | Use `GET /v2/{iso}/collections` (documentsTotal field) + SQL instead |
| Slow queries (high latency) | Embedding, DB, or overhead bottleneck | Run Step 6 (Quick Perf Check) to identify dominant component. For full analysis use perf-test skill |

## Rules
- Before starting, check which env vars are set and inform the user about missing ones. For API checks `TOKEN` is required; for DB access `kubectl` context is required. Tell the user what to set if missing, with examples.
- **Secret redaction**: when printing env vars, DB credentials, or tokens in output, ALWAYS mask sensitive values: show `${VAR:0:8}...<REDACTED>` for tokens, `***` for passwords. Never print full token or password values. Patterns to redact: `*TOKEN*`, `*PASSWORD*`, `*SECRET*`, `*KEY*`, `*CREDENTIAL*`.
- ONLY read-only operations (SELECT, GET, describe) - do not modify data.
- Option C (set ENABLE_DB_PROXY) requires explicit user confirmation before executing - it restarts the pod.
- Do not delete or modify data in the database.
- If the user describes a symptom, prioritize diagnostics relevant to that symptom first.
- Use `--profile ai` for AWS CLI calls (not the default profile).

## Notes
- DB schema uses hashed prefixes (`iso_prefix`, `col_prefix`) - you cannot guess them, must look up in the `isolations` / `collections` tables first.
- `db-tools` pod has `psql` in the current directory (`./psql`), not in system PATH.
- Background pod processes the embedding queue - if documents are stuck, that's where to look first.
- The `find-documents` endpoint (500 bug) is a known issue in some versions - use alternatives.
