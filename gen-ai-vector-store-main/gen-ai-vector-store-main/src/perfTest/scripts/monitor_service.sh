#!/bin/bash
# Copyright (c) 2025 Pegasystems Inc.
# All rights reserved.
#
# Service metrics monitoring script

SERVICE_URL=${SERVICE_URL:-http://localhost:28082}
INTERVAL=${INTERVAL:-10}
OUTPUT_FILE=${OUTPUT_FILE:-metrics_$(date +%Y%m%d_%H%M%S).csv}

# Colors for terminal output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo "Starting service metrics monitoring"
echo "Service URL: $SERVICE_URL"
echo "Interval: ${INTERVAL}s"
echo "Output file: $OUTPUT_FILE"
echo ""

# Create CSV header
echo "timestamp,goroutines,memory_alloc_mb,memory_sys_mb,active_requests,circuit_breaker_state" > "$OUTPUT_FILE"

while true; do
  TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  
  # Fetch metrics
  METRICS=$(curl -s "${SERVICE_URL}/metrics" 2>/dev/null)
  
  if [ $? -ne 0 ]; then
    echo -e "${RED}[ERROR]${NC} Failed to fetch metrics at $TIMESTAMP"
    sleep $INTERVAL
    continue
  fi
  
  # Extract metrics
  GOROUTINES=$(echo "$METRICS" | grep "^go_goroutines " | awk '{print $2}' | head -1)
  MEMORY_ALLOC=$(echo "$METRICS" | grep "^go_memstats_alloc_bytes " | awk '{print $2}' | head -1)
  MEMORY_SYS=$(echo "$METRICS" | grep "^go_memstats_sys_bytes " | awk '{print $2}' | head -1)
  ACTIVE_REQUESTS=$(echo "$METRICS" | grep "^http_requests_in_flight " | awk '{print $2}' | head -1)
  
  # Convert bytes to MB
  if [ -n "$MEMORY_ALLOC" ]; then
    MEMORY_ALLOC_MB=$(echo "scale=2; $MEMORY_ALLOC / 1048576" | bc)
  else
    MEMORY_ALLOC_MB="N/A"
  fi
  
  if [ -n "$MEMORY_SYS" ]; then
    MEMORY_SYS_MB=$(echo "scale=2; $MEMORY_SYS / 1048576" | bc)
  else
    MEMORY_SYS_MB="N/A"
  fi
  
  # Circuit breaker state
  CB_STATE=$(echo "$METRICS" | grep "circuit_breaker_state" | awk '{print $2}' | head -1)
  CB_STATE=${CB_STATE:-0}
  
  # Write to CSV
  echo "$TIMESTAMP,$GOROUTINES,$MEMORY_ALLOC_MB,$MEMORY_SYS_MB,$ACTIVE_REQUESTS,$CB_STATE" >> "$OUTPUT_FILE"
  
  # Console output with color coding
  echo -ne "\r[$(date +%H:%M:%S)] "
  
  # Goroutines warning
  if [ -n "$GOROUTINES" ] && [ "$GOROUTINES" -gt 1000 ]; then
    echo -ne "${RED}Goroutines: $GOROUTINES${NC} "
  else
    echo -ne "Goroutines: ${GOROUTINES:-N/A} "
  fi
  
  # Memory warning
  if [ -n "$MEMORY_ALLOC_MB" ] && [ $(echo "$MEMORY_ALLOC_MB > 300" | bc) -eq 1 ]; then
    echo -ne "${YELLOW}Memory: ${MEMORY_ALLOC_MB}MB${NC} "
  else
    echo -ne "Memory: ${MEMORY_ALLOC_MB:-N/A}MB "
  fi
  
  echo -ne "Active Requests: ${ACTIVE_REQUESTS:-N/A} "
  
  # Circuit breaker state
  if [ "$CB_STATE" != "0" ]; then
    echo -ne "${RED}CB: OPEN${NC}"
  fi
  
  sleep $INTERVAL
done
