#!/bin/bash
# run-functional-tests.sh
# Runs functional tests for all microservices.
# Requires running PostgreSQL database.
# Usage: ./scripts/run-functional-tests.sh [service-name]

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "  Furab Backend - Functional Tests"
echo "=========================================="
echo ""
echo "Prerequisites: PostgreSQL must be running"
echo ""

TOTAL=0
PASSED=0
FAILED=0

run_test() {
    local service=$1
    local service_path="$ROOT_DIR/services/$service"

    if [ ! -d "$service_path/test/functional" ]; then
        echo "[SKIP] $service (no functional tests)"
        return
    fi

    TOTAL=$((TOTAL + 1))
    echo ""
    echo "--- Functional Testing: $service ---"
    
    cd "$service_path"
    if go test ./test/functional/... -v -tags=functional -count=1 2>&1; then
        PASSED=$((PASSED + 1))
        echo "[PASS] $service"
    else
        FAILED=$((FAILED + 1))
        echo "[FAIL] $service"
    fi
}

if [ -n "$1" ]; then
    run_test "$1"
else
    for dir in "$ROOT_DIR"/services/*/; do
        service=$(basename "$dir")
        run_test "$service"
    done
fi

echo ""
echo "=========================================="
echo "  Results: $PASSED/$TOTAL passed, $FAILED failed"
echo "=========================================="
