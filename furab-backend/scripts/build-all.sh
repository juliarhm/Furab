#!/bin/bash
# build-all.sh
# Builds Docker images for all microservices.
# Usage: ./scripts/build-all.sh [tag]

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
TAG=${1:-latest}
REGISTRY=${DOCKER_REGISTRY:-"furab"}

echo "=========================================="
echo "  Furab Backend - Build All Images"
echo "  Tag: $TAG"
echo "=========================================="

cd "$ROOT_DIR"

for dir in services/*/; do
    service=$(basename "$dir")
    echo ""
    echo "--- Building: $service ---"
    
    if [ -f "$dir/Dockerfile" ]; then
        docker build \
            -t "${REGISTRY}/${service}:${TAG}" \
            -f "$dir/Dockerfile" \
            . 2>&1 || echo "[WARN] Build failed for $service"
        echo "[DONE] ${REGISTRY}/${service}:${TAG}"
    else
        echo "[SKIP] No Dockerfile found"
    fi
done

# Build gateway
echo ""
echo "--- Building: api-gateway ---"
if [ -f "gateway/api-gateway/Dockerfile" ]; then
    docker build \
        -t "${REGISTRY}/api-gateway:${TAG}" \
        -f "gateway/api-gateway/Dockerfile" \
        . 2>&1 || echo "[WARN] Build failed for api-gateway"
    echo "[DONE] ${REGISTRY}/api-gateway:${TAG}"
fi

echo ""
echo "=========================================="
echo "  All images built successfully"
echo "=========================================="
