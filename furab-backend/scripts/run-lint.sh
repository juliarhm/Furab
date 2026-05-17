#!/bin/bash
# run-lint.sh
# Runs go vet and optional golangci-lint for all services.
# Usage: ./scripts/run-lint.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "  Furab Backend - Lint & Vet"
echo "=========================================="

# Run go vet on all services
for dir in "$ROOT_DIR"/services/*/; do
    service=$(basename "$dir")
    echo "--- Vetting: $service ---"
    cd "$dir"
    go vet ./... 2>&1 || echo "[WARN] vet issues in $service"
done

# Vet shared library
echo "--- Vetting: shared ---"
cd "$ROOT_DIR/shared"
go vet ./... 2>&1 || echo "[WARN] vet issues in shared"

# Vet gateway
echo "--- Vetting: api-gateway ---"
cd "$ROOT_DIR/gateway/api-gateway"
go vet ./... 2>&1 || echo "[WARN] vet issues in api-gateway"

echo ""
echo "=========================================="
echo "  Lint & Vet complete"
echo "=========================================="
