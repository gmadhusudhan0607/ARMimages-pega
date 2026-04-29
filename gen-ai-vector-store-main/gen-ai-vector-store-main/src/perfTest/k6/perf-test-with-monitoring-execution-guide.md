# Performance Test With Monitoring Execution Guide

## Overview
This guide provides comprehensive instructions for executing load tests with complete data capture, real-time monitoring, and thorough post-test analysis.
It supports various test scenarios including single test execution, A/B comparisons, and performance validation.

## Prerequisites

### Required Access
- Kubernetes cluster access with appropriate namespace permissions
- kubectl configured with target cluster context
- SAX token file (typically at `/tmp/sax.token`)

### Required Tools
- **kubectl**: Kubernetes command-line tool
- **Python 3.x**: For analysis scripts
- **curl**: HTTP client for metrics collection
- **jq**: JSON processor (optional but recommended)
- **tmux or screen**: For persistent terminal sessions (recommended)
- **go**: For running Go-based monitoring tools

### Storage Requirements
- At least 5GB free disk space for logs and metrics

### Terminal Setup
- 4 terminal sessions recommended for parallel monitoring:
  - Terminal 1: Port forwarding and test control
  - Terminal 2: Service metrics monitoring
  - Terminal 3: Resource monitoring
  - Terminal 4: Database and async metrics

---

## Monitoring Tools Reference

### monitor_service.sh
Bash script for continuous service metrics monitoring via Prometheus `/metrics` endpoint.

**Features:**
- Collects goroutine count, memory allocation, active requests, circuit breaker state
- Real-time colored console display with threshold warnings
- Saves data to CSV for post-test analysis
- Alerts when goroutines > 1000 or memory > 300MB
- Requires `curl` and `jq` (optional for detailed memory stats)

**Usage:**
```bash
SERVICE_URL=http://localhost:28082 \
INTERVAL=10 \
OUTPUT_FILE=metrics_$(date +%Y%m%d_%H%M%S).csv \
./src/perfTest/scripts/monitor_service.sh
```

**When to Use:**
- Real-time monitoring during test execution
- Quick health checks
- Continuous data collection for time-series analysis

---

## Test Configuration

Default configuration can be customized based on test objectives:

```yaml
# Standard Load Test Configuration
virtual_users: 500
test_duration: 10m
ramp_up_time: 1m

# Light Load Test Configuration
virtual_users: 100
test_duration: 2m
ramp_up_time: 30s

# Stress Test Configuration
virtual_users: 1000
test_duration: 15m
ramp_up_time: 2m
```

**Test Scripts:**
- `put-documents.js`: Async document ingestion operations
- `query-documents-random.js`: Sync query operations

**Customize:** Edit `src/perfTest/k6/k6-config.yaml` for your specific needs.

---

## Test Execution Scenarios

### Scenario 1: Single Test Execution
Follow all phases (1-5) sequentially for a single test run.

### Scenario 2: A/B Comparison Testing
Execute the full process twice with different configurations:
1. First execution with Configuration A
2. Second execution with Configuration B
3. Use comparison scripts to analyze differences

### Scenario 3: Performance Baseline
Execute test to establish performance baseline:
1. Run with current production configuration
2. Archive results as baseline
3. Compare future tests against this baseline

---

## Phase 1: Pre-Test Setup

### Step 1.1: Create Test Environment

```bash
# Create timestamped test directory
export TEST_NAME="load_test_$(date +%Y%m%d_%H%M%S)"
mkdir -p $TEST_NAME/{k6_logs,service_logs,metrics,analysis,scripts}
cd $TEST_NAME

# Save test metadata
cat > test_metadata.json << EOF
{
  "test_name": "$TEST_NAME",
  "start_time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "cluster": "$(kubectl config current-context)",
  "namespace": "genai-vector-store",
  "service_version": "$(kubectl get deployment genai-vector-store -n genai-vector-store -o jsonpath='{.spec.template.spec.containers[0].image}')",
  "test_scenario": "single_execution",
  "virtual_users": 500,
  "duration": "10m"
}
EOF

echo "Test environment created: $TEST_NAME"
```

### Step 1.2: Verify Cluster and Service Status

```bash
# Set cluster context (replace with your cluster name)
kubectl config use-context <your-cluster-name>

# Verify connection
kubectl cluster-info

# Check service deployment
kubectl get deployment genai-vector-store -n genai-vector-store

# Check all pods are running
kubectl get pods -n genai-vector-store -l app.kubernetes.io/name=genai-vector-store

# Verify service version and image
kubectl describe deployment genai-vector-store -n genai-vector-store | grep Image

# Check environment configuration
kubectl exec deployment/genai-vector-store -n genai-vector-store -- env | grep -E "HTTP_REQUEST_TIMEOUT|BACKGROUND_TIMEOUT|LOG_LEVEL"

# Check current resource allocation
kubectl top pods -n genai-vector-store
kubectl top nodes
```

### Step 1.3: Setup Real-Time Log Streaming

**⚠️ CRITICAL:** Start log streaming BEFORE test execution to capture complete data.

```bash
# Create log streaming script
cat > scripts/stream_all_logs.sh << 'EOF'
#!/bin/bash
NAMESPACE=${1:-genai-vector-store}
LOG_DIR=${2:-service_logs}

echo "Starting log streaming for namespace: $NAMESPACE"
mkdir -p "$LOG_DIR"

# Get all service pods (exclude db-tools)
PODS=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=genai-vector-store,name!=db-tools -o jsonpath='{.items[*].metadata.name}')

# Start streaming logs from each pod individually
for pod in $PODS; do
  echo "Starting log stream for pod: $pod"
  kubectl logs -f $pod -n $NAMESPACE \
    --all-containers=true \
    --prefix=true \
    --timestamps \
    > "$LOG_DIR/${pod}_stream.log" 2>&1 &
  echo $! > "$LOG_DIR/${pod}_stream.pid"
  echo "  - Pod $pod streaming to ${pod}_stream.log (PID: $(cat $LOG_DIR/${pod}_stream.pid))"
done

# Stream combined deployment logs
echo "Starting combined deployment log stream"
kubectl logs -f deployment/genai-vector-store -n $NAMESPACE \
  --all-containers=true \
  --prefix=true \
  --timestamps \
  > "$LOG_DIR/deployment_stream.log" 2>&1 &
echo $! > "$LOG_DIR/deployment_stream.pid"

# Stream background service logs (if exists)
if kubectl get deployment genai-vector-store-background -n $NAMESPACE >/dev/null 2>&1; then
  echo "Starting background service log stream"
  kubectl logs -f deployment/genai-vector-store-background -n $NAMESPACE \
    --all-containers=true \
    --prefix=true \
    --timestamps \
    > "$LOG_DIR/background_stream.log" 2>&1 &
  echo $! > "$LOG_DIR/background_stream.pid"
fi

echo ""
echo "Log streaming started successfully!"
echo "PIDs saved in $LOG_DIR/*_stream.pid"
echo "Logs being written to $LOG_DIR/*_stream.log"
echo ""
echo "To stop streaming: for pid in $LOG_DIR/*.pid; do kill \$(cat \$pid) 2>/dev/null; done"
EOF

chmod +x scripts/stream_all_logs.sh

# Start log streaming
./scripts/stream_all_logs.sh genai-vector-store service_logs

# Wait for streaming to initialize
sleep 5

# Verify logs are being written
echo "Verifying log streams..."
ls -lh service_logs/*.log
tail -5 service_logs/deployment_stream.log
```

### Step 1.4: Setup Port Forwarding

```bash
# Terminal 1: Get specific pod name (exclude db-tools)
POD_NAME=$(kubectl get pods -n genai-vector-store \
  -l app.kubernetes.io/name=genai-vector-store,name!=db-tools \
  -o jsonpath='{.items[0].metadata.name}')
echo "Using pod: $POD_NAME"
echo "pod_name: $POD_NAME" >> test_metadata.json

# Port forward health/metrics endpoint (8082)
kubectl port-forward -n genai-vector-store pod/$POD_NAME 28082:8082 &
PF_HEALTH_PID=$!
echo "health_port_forward_pid: $PF_HEALTH_PID" >> test_metadata.json

# Port forward service API endpoint (8080)
kubectl port-forward -n genai-vector-store pod/$POD_NAME 28080:8080 &
PF_SERVICE_PID=$!
echo "service_port_forward_pid: $PF_SERVICE_PID" >> test_metadata.json

# Wait for port forwards to establish
sleep 3

# Verify connectivity
echo "Verifying port forward connectivity..."
curl -s http://localhost:28082/health/liveness && echo " ✓ Health endpoint OK"
curl -s http://localhost:28082/metrics | head -5 && echo " ✓ Metrics endpoint OK"

echo "Port forwarding established (PIDs: $PF_HEALTH_PID, $PF_SERVICE_PID)"
```

---

## Phase 2: Monitoring Setup

### Step 2.1: Start Service Metrics Collection (Terminal 2)

```bash
# Navigate to test directory
cd $TEST_NAME

# Start Prometheus metrics monitoring
SERVICE_URL=http://localhost:28082 \
INTERVAL=10 \
OUTPUT_FILE=metrics/service_metrics_$(date +%Y%m%d_%H%M%S).csv \
../src/perfTest/scripts/monitor_service.sh

# This runs continuously - keep terminal open
# Watch for warnings about goroutines or memory
```

### Step 2.2: Start Resource Monitoring (Terminal 3)

```bash
# Navigate to test directory
cd $TEST_NAME

# Create comprehensive metrics collector
cat > scripts/collect_pod_metrics.sh << 'EOF'
#!/bin/bash
OUTPUT_DIR=${1:-metrics}
INTERVAL=${2:-10}

echo "Starting pod metrics collection (interval: ${INTERVAL}s)"
mkdir -p "$OUTPUT_DIR"

while true; do
  TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  
  # Pod resource usage
  echo "=== $TIMESTAMP ===" >> "$OUTPUT_DIR/pod_resources.log"
  kubectl top pods -n genai-vector-store >> "$OUTPUT_DIR/pod_resources.log"
  
  # Node resource usage
  echo "=== $TIMESTAMP ===" >> "$OUTPUT_DIR/node_resources.log"
  kubectl top nodes >> "$OUTPUT_DIR/node_resources.log"
  
  # Pod status
  echo "=== $TIMESTAMP ===" >> "$OUTPUT_DIR/pod_status.log"
  kubectl get pods -n genai-vector-store >> "$OUTPUT_DIR/pod_status.log"
  
  sleep $INTERVAL
done
EOF

chmod +x scripts/collect_pod_metrics.sh
./scripts/collect_pod_metrics.sh metrics 10

# This runs continuously - keep terminal open
```

### Step 2.3: Start Application-Specific Monitoring (Terminal 4)

```bash
# Navigate to test directory
cd $TEST_NAME

# Monitor application-specific metrics (adjust based on your application)
while true; do
  clear
  echo "=== Application Metrics at $(date +%H:%M:%S) ==="
  echo ""
  echo "Service Metrics:"
  curl -s http://localhost:28082/metrics | grep -E "async_queue_size|async_processing_duration|go_goroutines|db_connections|http_requests_total"
  echo ""
  echo "Database Connections (if db-tools available):"
  kubectl exec -n genai-vector-store deployment/db-tools -- \
    psql -U $DB_USER -h $DB_HOST -d $DB_NAME \
    -c "SELECT count(*) as conn_count, state FROM pg_stat_activity GROUP BY state;" \
    2>/dev/null || echo "Database query not available"
  echo ""
  sleep 5
done
```

---

## Phase 3: Test Execution

### Step 3.1: Final Pre-Flight Check

```bash
# Navigate to test directory (Terminal 1)
cd $TEST_NAME

# Verify all monitoring is active
echo "Pre-flight checklist:"
echo "1. Log streaming active: $(ls -1 service_logs/*.pid 2>/dev/null | wc -l) streams"
echo "2. Port forwarding: Health(28082), Service(28080)"
echo "3. Metrics collection running in Terminal 2"
echo "4. Resource monitoring running in Terminal 3"
echo "5. Application monitoring running in Terminal 4"
echo ""
echo "Press Enter to continue with test deployment..."
read
```

### Step 3.2: Deploy K6 Load Test

```bash
# Update test start time in metadata
if command -v jq &> /dev/null; then
  jq '.test_execution_start = "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"' test_metadata.json > tmp.json && mv tmp.json test_metadata.json
else
  echo "  test_execution_start: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> test_metadata.json
fi

# Verify SAX token exists
ls -la /tmp/sax.token || echo "WARNING: SAX token not found at /tmp/sax.token"

# Deploy k6 test job
cd ../src/perfTest/k6
./run-test.sh

# Get k6 job pod name
K6_POD=$(kubectl get pods -n genai-vector-store -l job-name=k6-test --no-headers -o name | head -1)
echo "K6 Pod: ${K6_POD##*/}"

# Return to test directory
cd -
echo "k6_pod: ${K6_POD##*/}" >> test_metadata.json

# Stream k6 logs in real-time
kubectl logs -f $K6_POD -n genai-vector-store --all-containers=true --timestamps \
  > k6_logs/k6_stream.log 2>&1 &
echo $! > k6_logs/k6_stream.pid

echo "K6 test deployed and streaming logs"
```

### Step 3.3: Monitor Test Progress

```bash
# Watch k6 test status
echo "Monitoring k6 job status..."
kubectl get job k6-test -n genai-vector-store -w

# Or watch pod status (in another terminal if needed)
# kubectl get pod $K6_POD -n genai-vector-store -w
```

### Step 3.4: Real-Time Validation

While test is running, verify data collection (run in separate terminal or use tmux/screen):

```bash
cd $TEST_NAME

# Check log file sizes (should be growing)
watch -n 10 'du -sh service_logs/*.log k6_logs/*.log'

# Verify logs contain recent data
tail -f service_logs/deployment_stream.log

# Count logged requests (should increase)
watch -n 5 'grep -c "served request" service_logs/deployment_stream.log'

# Check for errors or warnings
watch -n 10 'grep -i "error\|warn" service_logs/deployment_stream.log | tail -10'

# Monitor pod health
kubectl get pods -n genai-vector-store --watch
```

---

## Phase 4: Post-Test Data Collection

### Step 4.1: Wait for Test Completion

```bash
# Navigate to test directory
cd $TEST_NAME

# Wait for k6 job to complete
echo "Waiting for k6 job to complete..."
kubectl wait --for=condition=complete job/k6-test -n genai-vector-store --timeout=20m

# Record test end time
if command -v jq &> /dev/null; then
  jq '.test_execution_end = "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"' test_metadata.json > tmp.json && mv tmp.json test_metadata.json
else
  echo "  test_execution_end: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> test_metadata.json
fi

echo "Test completed at $(date)"
```

### Step 4.2: Stop All Streaming

```bash
# Stop log streaming processes
echo "Stopping log streams..."
for pid_file in service_logs/*.pid k6_logs/*.pid; do
  if [ -f "$pid_file" ]; then
    PID=$(cat "$pid_file")
    kill $PID 2>/dev/null && echo "  Stopped PID $PID"
    rm "$pid_file"
  fi
done

# Stop metrics collection in Terminals 2, 3, 4 (Ctrl+C)
echo "Stop monitoring processes in Terminals 2, 3, and 4 (Ctrl+C)"
echo "Press Enter when all monitoring has been stopped..."
read
```

### Step 4.3: Collect Complete Logs (Backup)

```bash
# Get complete logs using kubectl (as backup)
echo "Collecting complete logs as backup..."

if command -v jq &> /dev/null; then
  TEST_START=$(jq -r '.test_execution_start' test_metadata.json)
else
  TEST_START=$(grep test_execution_start test_metadata.json | cut -d'"' -f4)
fi

# Service logs from all pods
for pod in $(kubectl get pods -n genai-vector-store -l app.kubernetes.io/name=genai-vector-store --no-headers -o name); do
  POD_NAME=${pod##*/}
  echo "  Collecting logs from $POD_NAME..."
  kubectl logs $pod -n genai-vector-store \
    --since-time=$TEST_START \
    --all-containers=true \
    --timestamps \
    > service_logs/${POD_NAME}_complete.log 2>&1
done

# Background service logs (if exists)
if kubectl get deployment genai-vector-store-background -n genai-vector-store >/dev/null 2>&1; then
  kubectl logs deployment/genai-vector-store-background -n genai-vector-store \
    --since-time=$TEST_START \
    --all-containers=true \
    --timestamps \
    > service_logs/background_complete.log 2>&1
fi

# K6 complete logs
kubectl logs job/k6-test -n genai-vector-store \
  --all-containers=true \
  > k6_logs/k6_complete.log 2>&1

echo "Backup log collection complete"
```

### Step 4.4: Collect Final Metrics

```bash
# Save final Prometheus metrics
curl -s http://localhost:28082/metrics > metrics/final_metrics.txt

# Save final memory debug info (if troubleshooting mode enabled)
curl -s http://localhost:28080/debug/memory > metrics/final_memory.json 2>/dev/null || \
  echo "Debug endpoint not available (troubleshooting mode disabled)"

# Save final pod status
kubectl get pods -n genai-vector-store -o wide > metrics/final_pod_status.txt
kubectl top pods -n genai-vector-store > metrics/final_pod_resources.txt
kubectl top nodes > metrics/final_node_resources.txt

# Save deployment info
kubectl describe deployment genai-vector-store -n genai-vector-store > metrics/deployment_info.txt

# Stop port forwarding
kill $PF_HEALTH_PID $PF_SERVICE_PID 2>/dev/null

echo "Final metrics collected"
```

### Step 4.5: Validate Log Completeness

```bash
# Create validation script
cat > scripts/validate_logs.py << 'EOF'
#!/usr/bin/env python3
import json
import os
from datetime import datetime

print("=" * 80)
print("LOG VALIDATION REPORT")
print("=" * 80)

# Load test metadata
try:
    with open('test_metadata.json') as f:
        content = f.read()
        # Handle both proper JSON and line-based format
        if content.strip().startswith('{'):
            metadata = json.loads(content)
        else:
            # Parse line-based format
            metadata = {}
            for line in content.split('\n'):
                if ':' in line:
                    key, value = line.split(':', 1)
                    metadata[key.strip()] = value.strip().strip('"')
except Exception as e:
    print(f"Error loading metadata: {e}")
    metadata = {}

if 'test_execution_start' in metadata and 'test_execution_end' in metadata:
    try:
        test_start = datetime.fromisoformat(metadata['test_execution_start'].replace('Z', '+00:00'))
        test_end = datetime.fromisoformat(metadata['test_execution_end'].replace('Z', '+00:00'))
        duration_minutes = (test_end - test_start).total_seconds() / 60
        
        print(f"\nTest Duration: {duration_minutes:.1f} minutes")
        print(f"Start: {test_start}")
        print(f"End: {test_end}")
    except Exception as e:
        print(f"Error parsing timestamps: {e}")
else:
    print("\nTest timing information not available")

# Check log files
print("\n" + "=" * 80)
print("SERVICE LOG FILES")
print("=" * 80)

service_logs = []
if os.path.exists('service_logs'):
    for log_file in os.listdir('service_logs'):
        if log_file.endswith('.log'):
            path = os.path.join('service_logs', log_file)
            size_mb = os.path.getsize(path) / 1024 / 1024
            
            # Count lines
            with open(path) as f:
                line_count = sum(1 for _ in f)
            
            service_logs.append({
                'file': log_file,
                'size_mb': size_mb,
                'lines': line_count
            })

    for log in sorted(service_logs, key=lambda x: x['size_mb'], reverse=True):
        print(f"  {log['file']:50s} {log['size_mb']:8.2f} MB  {log['lines']:>8,} lines")
else:
    print("  No service logs directory found")

# Check k6 logs
print("\n" + "=" * 80)
print("K6 LOG FILES")
print("=" * 80)

if os.path.exists('k6_logs'):
    for log_file in os.listdir('k6_logs'):
        if log_file.endswith('.log'):
            path = os.path.join('k6_logs', log_file)
            size_mb = os.path.getsize(path) / 1024 / 1024
            with open(path) as f:
                line_count = sum(1 for _ in f)
            print(f"  {log_file:50s} {size_mb:8.2f} MB  {line_count:>8,} lines")
else:
    print("  No k6 logs directory found")

# Validate request logging
print("\n" + "=" * 80)
print("REQUEST LOGGING VALIDATION")
print("=" * 80)

total_served = 0
for log in service_logs:
    if 'stream' in log['file'] or 'deployment' in log['file']:
        path = os.path.join('service_logs', log['file'])
        count = 0
        try:
            with open(path) as f:
                for line in f:
                    if '"msg":"served request"' in line or 'served request' in line:
                        count += 1
            total_served += count
            if count > 0:
                print(f"  {log['file']:50s} {count:>8,} served requests")
        except Exception as e:
            print(f"  Error processing {log['file']}: {e}")

print(f"\n  Total 'served request' logs: {total_served:,}")

# Check for errors
print("\n" + "=" * 80)
print("ERROR ANALYSIS")
print("=" * 80)

error_count = 0
warn_count = 0
for log in service_logs:
    path = os.path.join('service_logs', log['file'])
    try:
        with open(path) as f:
            for line in f:
                if '"level":"error"' in line or 'level=error' in line:
                    error_count += 1
                if '"level":"warn"' in line or 'level=warn' in line:
                    warn_count += 1
    except Exception as e:
        print(f"  Error processing {log['file']}: {e}")

print(f"  Error messages: {error_count:,}")
print(f"  Warning messages: {warn_count:,}")

print("\n" + "=" * 80)
print("VALIDATION SUMMARY")
print("=" * 80)

issues = []
if 'duration_minutes' in locals() and duration_minutes < 4:
    issues.append("⚠ Test duration less than expected")
if total_served < 50:
    issues.append("⚠ Very few 'served request' logs found")
if not any('stream' in log['file'] for log in service_logs):
    issues.append("⚠ No streaming logs found")

if issues:
    print("\nIssues Found:")
    for issue in issues:
        print(f"  {issue}")
else:
    print("\n✓ All validations passed")
    print("✓ Logs appear complete and valid")

print("=" * 80)
EOF

chmod +x scripts/validate_logs.py
python3 scripts/validate_logs.py
```

---

## Phase 5: Analysis

### Step 5.1: Analyze K6 Results

```bash
# Create k6 analysis script
cat > scripts/analyze_k6.py << 'EOF'
#!/usr/bin/env python3
import re
import json
import os

def analyze_k6_log(log_file):
    if not os.path.exists(log_file):
        return None, []
    
    with open(log_file, 'r') as f:
        content = f.read()
    
    metrics = {}
    
    # Extract key metrics
    patterns = {
        'total_requests': r'http_reqs.*?(\d+)',
        'requests_per_sec': r'http_reqs.*?([\d.]+)/s',
        'checks_passed': r'checks.*?(\d+) passed',
        'checks_failed': r'checks.*?(\d+) failed',
        'avg_duration': r'http_req_duration.*?avg=([\d.]+)ms',
        'median_duration': r'http_req_duration.*?med=([\d.]+)ms',
        'p90_duration': r'http_req_duration.*?p\(90\)=([\d.]+)',
        'p95_duration': r'http_req_duration.*?p\(95\)=([\d.]+)',
        'p99_duration': r'http_req_duration.*?p\(99\)=([\d.]+)',
        'max_duration': r'http_req_duration.*?max=([\d.]+)',
    }
    
    for key, pattern in patterns.items():
        match = re.search(pattern, content)
        if match:
            try:
                metrics[key] = float(match.group(1))
            except:
                metrics[key] = match.group(1)
    
    # Calculate success rate
    if 'checks_passed' in metrics and 'checks_failed' in metrics:
        total_checks = metrics['checks_passed'] + metrics['checks_failed']
        if total_checks > 0:
            metrics['success_rate'] = (metrics['checks_passed'] / total_checks) * 100
    
    # Count errors
    metrics['timeout_errors'] = content.count('request timeout')
    metrics['connection_errors'] = content.count('connection refused') + content.count('connection reset')
    metrics['auth_errors_401'] = content.count('status code: 401')
    metrics['server_errors_5xx'] = len(re.findall(r'status code: 5\d\d', content))
    
    # Extract test scenarios
    scenarios = []
    for line in content.split('\n'):
        if 'scenario' in line.lower() and ('query' in line.lower() or 'put' in line.lower() or 'iterations' in line.lower()):
            scenarios.append(line.strip())
    
    return metrics, scenarios

# Analyze k6 logs
print("=" * 80)
print("K6 TEST RESULTS ANALYSIS")
print("=" * 80)

log_files = ['k6_logs/k6_stream.log', 'k6_logs/k6_complete.log']
metrics = None
scenarios = []

for log_file in log_files:
    result_metrics, result_scenarios = analyze_k6_log(log_file)
    if result_metrics:
        metrics = result_metrics
        scenarios = result_scenarios
        print(f"\nAnalyzing: {log_file}")
        print("-" * 80)
        break

if metrics:
    print(f"\nRequest Metrics:")
    print(f"  Total Requests: {metrics.get('total_requests', 'N/A'):,}" if isinstance(metrics.get('total_requests'), (int, float)) else f"  Total Requests: {metrics.get('total_requests', 'N/A')}")
    print(f"  Requests/sec: {metrics.get('requests_per_sec', 'N/A')}")
    print(f"  Checks Passed: {metrics.get('checks_passed', 'N/A'):,}" if isinstance(metrics.get('checks_passed'), (int, float)) else f"  Checks Passed: {metrics.get('checks_passed', 'N/A')}")
    print(f"  Checks Failed: {metrics.get('checks_failed', 'N/A'):,}" if isinstance(metrics.get('checks_failed'), (int, float)) else f"  Checks Failed: {metrics.get('checks_failed', 'N/A')}")
    if 'success_rate' in metrics:
        print(f"  Success Rate: {metrics['success_rate']:.2f}%")
    
    print(f"\nDuration Metrics:")
    print(f"  Average: {metrics.get('avg_duration', 'N/A')} ms")
    print(f"  Median: {metrics.get('median_duration', 'N/A')} ms")
    print(f"  p90: {metrics.get('p90_duration', 'N/A')} s")
    print(f"  p95: {metrics.get('p95_duration', 'N/A')} s")
    print(f"  p99: {metrics.get('p99_duration', 'N/A')} s")
    print(f"  Max: {metrics.get('max_duration', 'N/A')} s")
    
    print(f"\nError Analysis:")
    print(f"  Timeout Errors: {metrics.get('timeout_errors', 0):,}")
    print(f"  Connection Errors: {metrics.get('connection_errors', 0):,}")
    print(f"  401 Auth Errors: {metrics.get('auth_errors_401', 0):,}")
    print(f"  5xx Server Errors: {metrics.get('server_errors_5xx', 0):,}")
    
    if scenarios:
        print(f"\nTest Scenarios Detected:")
        for scenario in scenarios[:5]:  # Limit to first 5
            print(f"  - {scenario}")
    
    # Save metrics
    os.makedirs('analysis', exist_ok=True)
    with open('analysis/k6_metrics.json', 'w') as f:
        json.dump(metrics, f, indent=2)
    
    print("\n✓ K6 metrics saved to analysis/k6_metrics.json")
else:
    print("\n⚠ No k6 log files found or unable to parse metrics")

print("\n" + "=" * 80)
EOF

chmod +x scripts/analyze_k6.py
python3 scripts/analyze_k6.py
```

### Step 5.2: Analyze Service Logs

```bash
# Create comprehensive service log analysis
cat > scripts/analyze_service_logs.py << 'EOF'
#!/usr/bin/env python3
import json
import os
from datetime import datetime
from collections import defaultdict
import statistics

print("=" * 80)
print("SERVICE LOGS ANALYSIS")
print("=" * 80)

# Load test metadata
try:
    with open('test_metadata.json') as f:
        content = f.read()
        if content.strip().startswith('{'):
            metadata = json.loads(content)
        else:
            metadata = {}
            for line in content.split('\n'):
                if ':' in line and '"' in line:
                    key = line.split(':')[0].strip().strip('"')
                    value = ':'.join(line.split(':')[1:]).strip().strip('",')
                    metadata[key] = value
except Exception as e:
    print(f"Error loading metadata: {e}")
    metadata = {}

# Parse test times
test_start = None
test_end = None
if 'test_execution_start' in metadata and 'test_execution_end' in metadata:
    try:
        test_start = datetime.fromisoformat(metadata['test_execution_start'].replace('Z', '+00:00'))
        test_end = datetime.fromisoformat(metadata['test_execution_end'].replace('Z', '+00:00'))
        print(f"\nTest Period: {test_start} to {test_end}")
    except Exception as e:
        print(f"Could not parse test times: {e}")

# Data structures
all_requests = []
query_requests = []
put_requests = []
status_codes = defaultdict(int)
error_messages = []
warning_messages = []

# Process service logs
log_files = []
if os.path.exists('service_logs'):
    for f in os.listdir('service_logs'):
        if f.endswith('_stream.log') or f.endswith('deployment_stream.log'):
            log_files.append(os.path.join('service_logs', f))

print(f"\nProcessing {len(log_files)} log files...")

for log_file in log_files:
    pod_name = os.path.basename(log_file).replace('_stream.log', '')
    
    with open(log_file, 'r') as f:
        for line in f:
            try:
                # Handle kubectl log format or pure JSON
                if line.startswith('{'):
                    data = json.loads(line)
                else:
                    parts = line.split(' ', 1)
                    if len(parts) < 2:
                        continue
                    data = json.loads(parts[1])
                
                ts = data.get('ts')
                if not ts:
                    continue
                
                ts_dt = datetime.fromtimestamp(ts, tz=datetime.timezone.utc) if isinstance(ts, (int, float)) else datetime.fromisoformat(str(ts).replace('Z', '+00:00'))
                
                # Filter by test period if available
                if test_start and test_end:
                    if not (test_start <= ts_dt <= test_end):
                        continue
                
                # Track errors and warnings
                level = data.get('level', '')
                if level == 'error':
                    error_messages.append(data)
                elif level == 'warn':
                    warning_messages.append(data)
                
                # Extract served request logs
                if data.get('msg') == 'served request':
                    uri = data.get('uri', '')
                    method = data.get('method', '')
                    status = data.get('status', 0)
                    duration_sec = data.get('duration_sec')
                    
                    request = {
                        'ts': ts_dt,
                        'pod': pod_name,
                        'method': method,
                        'uri': uri,
                        'status': status,
                        'duration_sec': duration_sec
                    }
                    
                    all_requests.append(request)
                    status_codes[status] += 1
                    
                    if method == 'PUT' and '/documents' in uri:
                        put_requests.append(request)
                    elif method == 'POST' and '/query' in uri:
                        query_requests.append(request)
            
            except (json.JSONDecodeError, ValueError, KeyError, AttributeError):
                continue

print(f"\nProcessed {len(log_files)} files")
print(f"Found {len(all_requests):,} served requests")
print(f"  Query requests (POST /query): {len(query_requests):,}")
print(f"  PUT requests (PUT /documents): {len(put_requests):,}")

# Analyze durations
def analyze_durations(requests, label):
    if not requests:
        print(f"\n{label}: NO DATA")
        return None
    
    durations = [r['duration_sec'] for r in requests if r.get('duration_sec') is not None]
    
    if not durations:
        print(f"\n{label}: NO DURATION DATA")
        return None
    
    sorted_durations = sorted(durations)
    
    stats = {
        'count': len(durations),
        'min': min(durations),
        'max': max(durations),
        'mean': statistics.mean(durations),
        'median': statistics.median(durations),
        'p90': sorted_durations[int(len(sorted_durations) * 0.90)] if len(sorted_durations) > 10 else sorted_durations[-1],
        'p95': sorted_durations[int(len(sorted_durations) * 0.95)] if len(sorted_durations) > 20 else sorted_durations[-1],
        'p99': sorted_durations[int(len(sorted_durations) * 0.99)] if len(sorted_durations) > 100 else sorted_durations[-1],
    }
    
    if len(durations) >= 2:
        stats['stdev'] = statistics.stdev(durations)
    
    print(f"\n{label}:")
    print(f"  Sample size: {stats['count']:,}")
    print(f"  Min: {stats['min']:.3f}s")
    print(f"  Max: {stats['max']:.3f}s")
    print(f"  Mean: {stats['mean']:.3f}s")
    print(f"  Median: {stats['median']:.3f}s")
    print(f"  p90: {stats['p90']:.3f}s")
    print(f"  p95: {stats['p95']:.3f}s")
    print(f"  p99: {stats['p99']:.3f}s")
    if 'stdev' in stats:
        print(f"  Std Dev: {stats['stdev']:.3f}s")
    
    return stats

# Analyze query and PUT requests
print("\n" + "=" * 80)
print("DURATION ANALYSIS")
print("=" * 80)

query_stats = analyze_durations(query_requests, "QUERY REQUESTS (POST /query)")
put_stats = analyze_durations(put_requests, "PUT REQUESTS (PUT /documents)")
all_stats = analyze_durations(all_requests, "ALL REQUESTS")

# Status code analysis
if all_requests:
    print("\n" + "=" * 80)
    print("STATUS CODE DISTRIBUTION")
    print("=" * 80)
    for status in sorted(status_codes.keys()):
        count = status_codes[status]
        pct = (count / len(all_requests) * 100)
        print(f"  {status}: {count:,} ({pct:.2f}%)")

# Error analysis
print("\n" + "=" * 80)
print("ERROR AND WARNING ANALYSIS")
print("=" * 80)
print(f"  Error messages: {len(error_messages):,}")
print(f"  Warning messages: {len(warning_messages):,}")

if error_messages:
    print(f"\nTop Error Messages:")
    error_counts = defaultdict(int)
    for err in error_messages:
        msg = err.get('msg', 'unknown')
        error_counts[msg] += 1
    
    for msg, count in sorted(error_counts.items(), key=lambda x: x[1], reverse=True)[:5]:
        print(f"  - {msg}: {count} occurrences")

# Save results
results = {
    'query_requests': len(query_requests),
    'put_requests': len(put_requests),
    'total_requests': len(all_requests),
    'query_stats': query_stats,
    'put_stats': put_stats,
    'all_stats': all_stats,
    'status_codes': dict(status_codes),
    'errors': len(error_messages),
    'warnings': len(warning_messages)
}

os.makedirs('analysis', exist_ok=True)
with open('analysis/service_log_analysis.json', 'w') as f:
    json.dump(results, f, indent=2, default=str)

print("\n" + "=" * 80)
print("✓ Analysis saved to analysis/service_log_analysis.json")
print("=" * 80)
EOF

chmod +x scripts/analyze_service_logs.py
python3 scripts/analyze_service_logs.py
```

### Step 5.3: Generate Comprehensive Report

```bash
# Create final analysis report
cat > analysis/test_report.md << 'EOF'
# Load Test Analysis Report

## Test Information

See `test_metadata.json` for complete test configuration.

## K6 Test Results

See `analysis/k6_metrics.json` for detailed metrics.

Key highlights:
- Total requests
- Success rate
- Duration percentiles
- Error counts

## Service Log Analysis

See `analysis/service_log_analysis.json` for detailed analysis.

Key highlights:
- Request counts by type
- Duration statistics
- Status code distribution
- Error and warning counts

## Key Findings

### Performance
1. Query Operations: [Review query_stats in service_log_analysis.json]
2. PUT Operations: [Review put_stats in service_log_analysis.json]
3. Overall Latency: [Review all_stats in service_log_analysis.json]

### Reliability
1. Success Rate: [Review status_codes in service_log_analysis.json]
2. Error Rate: [Review errors count in service_log_analysis.json]
3. Timeout Behavior: [Review timeout_errors in k6_metrics.json]

### Resource Utilization
1. Goroutines: [Review metrics/service_metrics_*.csv]
2. Memory Usage: [Review metrics/pod_resources.log]
3. Database Connections: [Review service logs for db connection patterns]

## Recommendations

Based on the analysis results:
1. [Add recommendations based on findings]
2. [Identify bottlenecks or issues]
3. [Suggest optimizations]

## Next Steps

1. Review detailed metrics in JSON files
2. Compare against baselines or requirements
3. Address any identified issues
4. Update performance baselines if applicable
EOF

echo "Comprehensive report template created: analysis/test_report.md"
echo "Review and customize the report based on your specific findings."
```

### Step 5.4: Create Comparison Script (for A/B Testing)

```bash
# Create A/B comparison script
cat > scripts/compare_tests.py << 'EOF'
#!/usr/bin/env python3
import json
import sys
import os

def load_analysis(test_dir):
    """Load analysis results from a test directory."""
    k6_path = os.path.join(test_dir, 'analysis', 'k6_metrics.json')
    service_path = os.path.join(test_dir, 'analysis', 'service_log_analysis.json')
    
    data = {'test_dir': test_dir}
    
    if os.path.exists(k6_path):
        with open(k6_path) as f:
            data['k6'] = json.load(f)
    
    if os.path.exists(service_path):
        with open(service_path) as f:
            data['service'] = json.load(f)
    
    return data

def compare_metric(name, value_a, value_b, lower_is_better=True):
    """Compare two metrics and determine improvement."""
    if value_a is None or value_b is None:
        return name, value_a, value_b, "N/A", "⚪"
    
    diff = value_b - value_a
    pct_change = (diff / value_a * 100) if value_a != 0 else 0
    
    if lower_is_better:
        symbol = "✓" if diff < 0 else "✗" if diff > 0 else "="
    else:
        symbol = "✓" if diff > 0 else "✗" if diff < 0 else "="
    
    return name, value_a, value_b, f"{pct_change:+.2f}%", symbol

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 compare_tests.py <test_dir_a> <test_dir_b>")
        print("Example: python3 compare_tests.py load_test_baseline load_test_optimized")
        sys.exit(1)
    
    test_a_dir = sys.argv[1]
    test_b_dir = sys.argv[2]
    
    print("=" * 80)
    print("LOAD TEST COMPARISON")
    print("=" * 80)
    
    data_a = load_analysis(test_a_dir)
    data_b = load_analysis(test_b_dir)
    
    print(f"\nTest A: {test_a_dir}")
    print(f"Test B: {test_b_dir}")
    
    # K6 Metrics Comparison
    if 'k6' in data_a and 'k6' in data_b:
        print("\n" + "=" * 80)
        print("K6 METRICS COMPARISON")
        print("=" * 80)
        print(f"\n{'Metric':<30} {'Test A':>15} {'Test B':>15} {'Change':>12} {'':>3}")
        print("-" * 80)
        
        metrics = [
            ('Total Requests', 'total_requests', False),
            ('Requests/sec', 'requests_per_sec', False),
            ('Success Rate %', 'success_rate', False),
            ('Avg Duration (ms)', 'avg_duration', True),
            ('p95 Duration (s)', 'p95_duration', True),
            ('p99 Duration (s)', 'p99_duration', True),
            ('Timeout Errors', 'timeout_errors', True),
            ('5xx Errors', 'server_errors_5xx', True),
        ]
        
        for label, key, lower_better in metrics:
            val_a = data_a['k6'].get(key)
            val_b = data_b['k6'].get(key)
            name, v_a, v_b, change, symbol = compare_metric(label, val_a, val_b, lower_better)
            
            v_a_str = f"{v_a:,.2f}" if isinstance(v_a, float) else f"{v_a:,}" if isinstance(v_a, int) else "N/A"
            v_b_str = f"{v_b:,.2f}" if isinstance(v_b, float) else f"{v_b:,}" if isinstance(v_b, int) else "N/A"
            
            print(f"{name:<30} {v_a_str:>15} {v_b_str:>15} {change:>12} {symbol:>3}")
    
    # Service Metrics Comparison
    if 'service' in data_a and 'service' in data_b:
        print("\n" + "=" * 80)
        print("SERVICE METRICS COMPARISON")
        print("=" * 80)
        print(f"\n{'Metric':<30} {'Test A':>15} {'Test B':>15} {'Change':>12} {'':>3}")
        print("-" * 80)
        
        # Query stats
        if data_a['service'].get('query_stats') and data_b['service'].get('query_stats'):
            print("\nQuery Operations:")
            for key in ['mean', 'median', 'p95', 'p99']:
                val_a = data_a['service']['query_stats'].get(key)
                val_b = data_b['service']['query_stats'].get(key)
                name, v_a, v_b, change, symbol = compare_metric(f"  {key} (s)", val_a, val_b, True)
                
                v_a_str = f"{v_a:.3f}" if v_a is not None else "N/A"
                v_b_str = f"{v_b:.3f}" if v_b is not None else "N/A"
                
                print(f"{name:<30} {v_a_str:>15} {v_b_str:>15} {change:>12} {symbol:>3}")
        
        # PUT stats
        if data_a['service'].get('put_stats') and data_b['service'].get('put_stats'):
            print("\nPUT Operations:")
            for key in ['mean', 'median', 'p95', 'p99']:
                val_a = data_a['service']['put_stats'].get(key)
                val_b = data_b['service']['put_stats'].get(key)
                name, v_a, v_b, change, symbol = compare_metric(f"  {key} (s)", val_a, val_b, True)
                
                v_a_str = f"{v_a:.3f}" if v_a is not None else "N/A"
                v_b_str = f"{v_b:.3f}" if v_b is not None else "N/A"
                
                print(f"{name:<30} {v_a_str:>15} {v_b_str:>15} {change:>12} {symbol:>3}")
        
        # Error counts
        print("\nErrors and Warnings:")
        for key, label in [('errors', 'Errors'), ('warnings', 'Warnings')]:
            val_a = data_a['service'].get(key, 0)
            val_b = data_b['service'].get(key, 0)
            name, v_a, v_b, change, symbol = compare_metric(f"  {label}", val_a, val_b, True)
            
            print(f"{name:<30} {v_a:>15,} {v_b:>15,} {change:>12} {symbol:>3}")
    
    print("\n" + "=" * 80)
    print("LEGEND: ✓ = Improvement, ✗ = Regression, = = No Change, ⚪ = N/A")
    print("=" * 80)
EOF

chmod +x scripts/compare_tests.py

echo "Comparison script created: scripts/compare_tests.py"
echo "Usage: python3 scripts/compare_tests.py <test_dir_a> <test_dir_b>"
```

---

## Troubleshooting

### Port Forward Connection Lost

```bash
# Kill existing port-forwards
ps aux | grep "port-forward" | grep -v grep | awk '{print $2}' | xargs kill 2>/dev/null

# Get new pod and restart
POD_NAME=$(kubectl get pods -n genai-vector-store \
  -l app.kubernetes.io/name=genai-vector-store,name!=db-tools \
  -o jsonpath='{.items[0].metadata.name}')

kubectl port-forward -n genai-vector-store pod/$POD_NAME 28082:8082 &
kubectl port-forward -n genai-vector-store pod/$POD_NAME 28080:8080 &

sleep 3
curl -s http://localhost:28082/health/liveness
```

### Log Streaming Stopped

```bash
# Check if processes are still running
ps aux | grep "kubectl logs -f"

# Restart streaming for specific pod
POD=<pod-name>
kubectl logs -f $POD -n genai-vector-store --all-containers=true \
  --prefix=true --timestamps >> service_logs/${POD}_stream.log 2>&1 &

# Check for pod restarts
kubectl get events -n genai-vector-store --sort-by='.lastTimestamp' | grep -i restart
```

### K6 Test Fails to Start

```bash
# Check job status
kubectl describe job k6-test -n genai-vector-store

# Check ConfigMaps
kubectl get configmap -n genai-vector-store | grep k6

# Check SAX token
kubectl get secret sax-token -n genai-vector-store -o yaml

# Check k6 pod logs
kubectl logs job/k6-test -n genai-vector-store --all-containers=true
```

### No Served Request Logs

```bash
# Check log level
kubectl get deployment genai-vector-store -n genai-vector-store -o yaml | grep -i log_level

# Check different log message formats
grep -E '"msg"|"message"' service_logs/deployment_stream.log | head -20

# Verify requests are reaching service
curl http://localhost:28082/metrics | grep http_requests_total
```

### Missing Metrics Data

```bash
# Verify Prometheus endpoint
curl http://localhost:28082/metrics | head -20

# Check if port-forward is working
lsof -i :28082

# Verify pod is healthy
kubectl get pods -n genai-vector-store -l app.kubernetes.io/name=genai-vector-store
kubectl describe pod $POD_NAME -n genai-vector-store
```

### Python Analysis Scripts Fail

```bash
# Check Python version
python3 --version

# Install required packages if needed
pip3 install --user <package_name>

# Run with verbose output
python3 -v scripts/analyze_k6.py
```

---

## Cleanup

```bash
# Navigate to parent directory
cd ..

# Stop all monitoring processes
pkill -f "kubectl logs -f"
pkill -f "monitor_service.sh"
pkill -f "collect_pod_metrics.sh"
pkill -f "kubectl port-forward"

# Delete k6 resources
kubectl delete job k6-test -n genai-vector-store 2>/dev/null
kubectl delete configmap k6-config k6-scripts -n genai-vector-store 2>/dev/null

# Archive test results
tar -czf ${TEST_NAME}.tar.gz $TEST_NAME/
echo "Test results archived: ${TEST_NAME}.tar.gz"
echo "Original directory: $TEST_NAME"
```

---

## Best Practices

### Before Test Execution
1. Always verify cluster connectivity and pod health
2. Ensure adequate disk space for logs (5GB minimum)
3. Start log streaming BEFORE deploying test
4. Verify port forwarding is stable
5. Check that all monitoring tools are running

### During Test Execution
6. Monitor logs are actively being written
7. Watch for pod restarts or crashes
8. Verify metrics are being collected
9. Keep monitoring terminals visible
10. Note any unusual behavior or errors

### After Test Execution
11. Wait for test to fully complete before stopping streaming
12. Collect backup logs using kubectl
13. Validate log completeness
14. Run analysis scripts immediately
15. Archive results before cleanup

### Data Integrity
16. Use timestamped directories for each test run
17. Never reuse test directories
18. Validate logs before considering test complete
19. Keep raw logs until analysis is confirmed
20. Archive results for future reference

---

## Success Criteria Template

Customize these criteria based on your specific requirements:

### Performance Targets
- [ ] Average query latency < 2s
- [ ] p95 query latency < 5s
- [ ] p99 query latency < 10s
- [ ] PUT operation latency < 30s
- [ ] Requests per second > 100

### Reliability Targets
- [ ] Success rate > 99%
- [ ] Error rate < 1%
- [ ] No pod restarts during test
- [ ] No out-of-memory errors
- [ ] Goroutines return to baseline post-test

### Resource Utilization
- [ ] Memory usage < 1GB per pod
- [ ] CPU usage < 80% sustained
- [ ] Database connections released properly
- [ ] No resource leaks detected

---

## Appendix: Quick Reference

### Common Commands

```bash
# Check pod status
kubectl get pods -n genai-vector-store

# View pod logs
kubectl logs -f <pod-name> -n genai-vector-store

# Check metrics endpoint
curl http://localhost:28082/metrics | grep <metric_name>

# Monitor resource usage
kubectl top pods -n genai-vector-store

# Check test status
kubectl get job k6-test -n genai-vector-store

# View test logs
kubectl logs job/k6-test -n genai-vector-store
```

### Important Files

- `test_metadata.json`: Test configuration and timing
- `service_logs/*_stream.log`: Real-time service logs
- `k6_logs/k6_stream.log`: K6 test execution logs
- `metrics/service_metrics_*.csv`: Time-series metrics
- `metrics/pod_resources.log`: Resource usage over time
- `analysis/k6_metrics.json`: Parsed K6 results
- `analysis/service_log_analysis.json`: Parsed service metrics
- `analysis/test_report.md`: Final analysis report

### Port Reference

- **28082**: Health and metrics endpoint
- **28080**: Service API endpoint
- **8082**: Pod health/metrics port
- **8080**: Pod service port

---

## Version History

- **v1.0**: Initial generic guide combining best practices from multiple test executions
