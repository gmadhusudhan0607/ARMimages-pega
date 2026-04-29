#!/bin/bash

NAMESPACE="genai-vector-store"
POD_DELAY=5
MAX_RETRIES=10
RETRY_INTERVAL=5

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}" || exit 1

# Check if /tmp/sax.token exists, else return error and exit
if [ ! -f /tmp/sax.token ]; then
  echo "/tmp/sax.token not found!"
  exit 1
fi

# Read token from /tmp/sax.token and pass as env var to the job
export K6_TOKEN=$(cat /tmp/sax.token)

# Create or recreate the k6-token secret from /tmp/sax.token (mounted as SAX_TOKEN)
kubectl -n ${NAMESPACE} delete secret k6-token --ignore-not-found
kubectl -n ${NAMESPACE} create secret generic k6-token --from-file=SAX_TOKEN=/tmp/sax.token


# Delete and recreate ConfigMap from your k6 scripts directory
kubectl -n ${NAMESPACE} delete configmap k6-scripts --ignore-not-found
kubectl -n ${NAMESPACE} create configmap k6-scripts --from-file=scripts

# Delete and recreate the k6-scripts-runner ConfigMap to ensure latest version is used
kubectl -n ${NAMESPACE} delete configmap k6-scripts-runner --ignore-not-found
kubectl -n ${NAMESPACE} create configmap k6-scripts-runner --from-file=k6-scripts-runner.sh

# Delete existing job if it exists
kubectl -n ${NAMESPACE} delete job k6-job --ignore-not-found

# Apply the k6 test configuration
kubectl -n ${NAMESPACE} apply -f k6-config.yaml

# Apply the k6 load test job
kubectl -n ${NAMESPACE} apply -f k6-job.yaml

echo "Job created. To view logs, run:"
echo "kubectl -n ${NAMESPACE} logs -f job/k6-job "

# Initial delay before first log attempt
echo "Waiting ${POD_DELAY} second(s) for job pod(s) to start..."
sleep ${POD_DELAY}

# Function to check if pod is ready
function is_pod_ready() {
  local pod_status
  pod_status=$(kubectl -n ${NAMESPACE} get pod -l job-name=k6-job -o jsonpath='{.items[0].status.phase}' 2>/dev/null)
  if [[ "${pod_status}" == "Running" ]]; then
    return 0
  else
    echo "Pod status: ${pod_status:-PodInitializing}"
    return 1
  fi
}

# Trap SIGINT (Ctrl+C) to kill all background log streaming processes
trap 'echo "\nStopping all log streams..."; jobs -p | xargs -r kill; exit 130' INT

# Try to follow logs with retries for all pods
retry_count=0
while [ $retry_count -lt $MAX_RETRIES ]; do
  pod_names=( $(kubectl -n ${NAMESPACE} get pods -l job-name=k6-job -o jsonpath='{.items[*].metadata.name}') )
  if [ ${#pod_names[@]} -gt 0 ]; then
    echo "Pods are ready. Streaming logs from all pods... (kubectl -n ${NAMESPACE} logs -f \"$pod\" & ) "
    for pod in "${pod_names[@]}"; do
      echo "\n--- Logs from pod: $pod ---"
      kubectl -n ${NAMESPACE} logs -f "$pod" &
    done
    wait
    break
  else
    echo "Pods not ready yet. Retry ${retry_count}/${MAX_RETRIES} - Waiting ${RETRY_INTERVAL} seconds..."
    retry_count=$((retry_count + 1))
    sleep ${RETRY_INTERVAL}
  fi
  if [ $retry_count -eq $MAX_RETRIES ]; then
    echo "Maximum retries (${MAX_RETRIES}) reached. Pods may still be initializing."
    echo "Try running manually: kubectl -n ${NAMESPACE} get pods -l job-name=k6-job"
    exit 1
  fi
  done
