#!/bin/sh
set -e

echo "[Setup] Waiting for Temporal server at ${TEMPORAL_CLI_ADDRESS}..."

# Wait for Temporal server to be healthy
max_attempts=60
attempt=0
while [ $attempt -lt $max_attempts ]; do
  if temporal operator cluster health --address "${TEMPORAL_CLI_ADDRESS}" 2>/dev/null; then
    echo "[Setup] Temporal server is ready!"
    break
  fi
  attempt=$((attempt + 1))
  echo "  Attempt $attempt/$max_attempts..."
  sleep 5
done

if [ $attempt -eq $max_attempts ]; then
  echo "[Setup] ERROR: Temporal server not available after $max_attempts attempts"
  exit 1
fi

# Configure default namespace with retention
echo "[Setup] Configuring default namespace with 30-day retention..."
temporal operator namespace update default \
  --address "${TEMPORAL_CLI_ADDRESS}" \
  --retention 30d \
  --history-archival-state enabled \
  --visibility-archival-state enabled 2>&1 || echo "[Setup] Namespace update had warnings (may already be configured)"

echo "[Setup] Listing namespaces..."
temporal operator namespace list --address "${TEMPORAL_CLI_ADDRESS}" 2>&1 || true

echo "[Setup] Done! Namespace configuration complete."
