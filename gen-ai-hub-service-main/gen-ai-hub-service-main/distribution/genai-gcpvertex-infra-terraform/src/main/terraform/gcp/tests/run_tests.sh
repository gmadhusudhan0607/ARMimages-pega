#!/bin/bash
# Copyright (c) 2025 Pegasystems Inc.
# All rights reserved.

# Test runner for GCP Cloud Function
# This script sets up a virtual environment and runs the tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== GCP Cloud Function Test Runner ==="
echo ""

# Check if virtual environment exists
if [ ! -d "venv" ]; then
    echo "Creating virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
echo "Activating virtual environment..."
source venv/bin/activate

# Install dependencies
echo "Installing dependencies..."
# First install the base requirements (from templates directory)
cp ../templates/requirements.txt.tpl requirements.txt
pip install -q --upgrade pip
pip install -q -r requirements.txt

# Then install dev dependencies
pip install -q -r requirements-dev.txt

# Copy template to actual main.py for testing (from templates directory)
echo "Preparing main.py from template..."
cp ../templates/main.py.tpl main.py

# Run tests
echo ""
echo "Running tests..."
pytest test_main.py "$@"

# Cleanup
rm -f main.py requirements.txt

echo ""
echo "=== Tests Complete ==="
