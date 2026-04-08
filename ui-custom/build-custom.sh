#!/bin/bash
# Build custom Temporal UI with API Keys feature

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building custom Temporal UI with API Keys..."

# Build frontend
echo "→ Building frontend..."
docker run --rm -v "$SCRIPT_DIR:/app" -w /app node:22-alpine sh -c "
    npm install -g pnpm@latest &&
    pnpm install --frozen-lockfile &&
    pnpm build:server
"

# Build backend (needs to run after frontend)
echo "→ Building backend..."
docker run --rm -v "$SCRIPT_DIR/server:/home/server-builder" -w /home/server-builder golang:1.23-alpine sh -c "
    apk add --no-cache make git &&
    go mod download &&
    make build-server
"

echo "✓ Build complete! Binary at: $SCRIPT_DIR/server/ui-server"
echo ""
echo "To build Docker image:"
echo "  docker build -f Dockerfile.custom -t temporal-ui-custom:latest ."
