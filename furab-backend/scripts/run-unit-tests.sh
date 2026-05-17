#!/bin/bash
# run-unit-tests.sh
# Runs unit tests for all microservices.
# Usage: ./scripts/run-unit-tests.sh [service-name]

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "  Furab Backend - Unit Tests"
echo "=========================================="

TOTAL=0
PASSED=0
FAILED=0

run_test() {
    local service=$1
    local service_path="$ROOT_DIR/services/$service"

    if [ ! -d "$service_path/test/unit" ]; then
        echo -e "${YELLOW}[SKIP]${NC} $service (no unit tests)"
        return
    fi

    TOTAL=$((TOTAL + 1))
    echo -e "\n--- Testing: $service ---"
    
    cd "$service_path"
    if go test ./test/unit/... -v -count=1 2>&1; then
        PASSED=$((PASSED + 1))
        echo -e "${GREEN}[PASS]${NC} $service"
    else
        FAILED=$((FAILED + 1))
        echo -e "${RED}[FAIL]${NC} $service"
    fi
}

# Run for specific service or all
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

if [ $FAILED -gt 0 ]; then
    exit 1
fi
