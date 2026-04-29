#!/bin/sh
set -e

echo "K6 Scripts Runner Starting..."
echo "TEST_SCRIPTS: ${TEST_SCRIPTS}"
echo "PARALLEL_MODE: ${PARALLEL_MODE}"

# Validate required environment variables
if [ -z "$TEST_SCRIPTS" ]; then
    echo "ERROR: TEST_SCRIPTS environment variable is not set or is empty"
    echo "Please configure TEST_SCRIPTS in k6-config.yaml ConfigMap"
    exit 1
fi

# Default values if not set
PARALLEL_MODE="${PARALLEL_MODE:-false}"

# Parse TEST_SCRIPTS - supports both formats (with or without trailing commas)
# 1. Remove trailing commas from each line
# 2. Remove leading/trailing whitespace from each line
# 3. Filter out empty lines
# 4. Filter out comment lines (starting with #)
SCRIPTS=$(echo "$TEST_SCRIPTS" | sed 's/,[[:space:]]*$//' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//' | grep -v '^$' | grep -v '^#')

echo "Parsed scripts to run: $SCRIPTS"

# Check if running in parallel mode
if [ "$PARALLEL_MODE" = "true" ]; then
    echo "Running scripts in PARALLEL mode"
    PIDS=""
    
    for script in $SCRIPTS; do
        script_file="/tests/${script}"
        script_name=$(basename "$script" .js)
        output_file="${script_name}-results.json"
        
        if [ -f "$script_file" ]; then
            echo "Starting script: $script_file (output: $output_file)"
            k6 run --out json="$output_file" "$script_file" &
            PIDS="$PIDS $!"
        else
            echo "ERROR: Script not found: $script_file"
            exit 1
        fi
    done
    
    # Wait for all background processes
    echo "Waiting for all scripts to complete..."
    for pid in $PIDS; do
        wait $pid
        exit_code=$?
        if [ $exit_code -ne 0 ]; then
            echo "ERROR: Process $pid failed with exit code $exit_code"
            exit $exit_code
        fi
    done
    
    echo "All scripts completed successfully in parallel mode"
else
    echo "Running scripts in SEQUENTIAL mode"
    
    for script in $SCRIPTS; do
        script_file="/tests/${script}"
        script_name=$(basename "$script" .js)
        output_file="${script_name}-results.json"
        
        if [ -f "$script_file" ]; then
            echo "Running script: $script_file (output: $output_file)"
            k6 run --out json="$output_file" "$script_file"
            echo "Completed: $script"
        else
            echo "ERROR: Script not found: $script_file"
            exit 1
        fi
    done
    
    echo "All scripts completed successfully in sequential mode"
fi

echo "Tests completed successfully."
