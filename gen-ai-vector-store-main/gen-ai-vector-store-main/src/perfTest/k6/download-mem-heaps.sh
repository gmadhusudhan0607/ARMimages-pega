#!/bin/bash

NAMESPACE="genai-vector-store"
LABEL="app.kubernetes.io/name=genai-vector-store"
PROFILE_DIR="/tmp/vs-mem-profile"
PORT_BASE=18080

# Create directory if not exists
mkdir -p "$PROFILE_DIR"

# Get all pod names with the label in the namespace
pods=( $(kubectl get pods -n "$NAMESPACE" -l "$LABEL" -o jsonpath='{.items[*].metadata.name}') )

if [ ${#pods[@]} -eq 0 ]; then
  echo "No pods found with label $LABEL in namespace $NAMESPACE."
  exit 1
fi

for i in "${!pods[@]}"; do
  pod="${pods[$i]}"
  # Exclude pods that start with "db-tools-"
  if [[ "$pod" == db-tools-* ]]; then
    echo "Skipping pod $pod"
    continue
  fi
  port=$((PORT_BASE + i))
  echo "Processing pod $pod on local port $port..."

  # Start port-forward in background
  kubectl port-forward -n "$NAMESPACE" "$pod" "$port":8080 &
  pf_pid=$!

  # Wait for port-forward to be ready
  for j in {1..10}; do
    if nc -z localhost "$port"; then
      break
    fi
    sleep 0.5
  done

  if ! nc -z localhost "$port"; then
    echo "Port-forward to $pod failed. Skipping."
    kill $pf_pid 2>/dev/null
    continue
  fi

  # Download heap profile
  curl -s -o "$PROFILE_DIR/${pod}-memory-profile.pb.gz" "http://localhost:$port/debug/pprof/heap"
  echo "Downloaded heap profile for $pod to $PROFILE_DIR/${pod}-memory-profile.pb.gz"

  # Stop port-forward
  kill $pf_pid
  wait $pf_pid 2>/dev/null

done

# Ask if user wants to pack all downloaded files into one zip file
echo -n "Do you want to pack all downloaded heap profiles into one zip file? [Y/N] (default: N): "
read answer
answer=${answer:-N}
if [[ "$answer" =~ ^[Yy]$ ]]; then
  ZIP_FILE="$PROFILE_DIR/vs-mem-profiles.zip"
  zip -j "$ZIP_FILE" "$PROFILE_DIR"/*-memory-profile.pb.gz
  echo "All heap profiles have been packed into $ZIP_FILE."
else
  echo "Skipping zip archive creation."
fi
