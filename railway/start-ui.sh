#!/bin/bash
# Start Temporal UI server with config overrides for monolith
set -e

# Override config to point to localhost services
export TEMPORAL_HOST="${TEMPORAL_HOST:-localhost}"
export CASDOOR_CLIENT_ID="${CASDOOR_CLIENT_ID:-temporal-ui}"
export CASDOOR_CLIENT_SECRET="${CASDOOR_CLIENT_SECRET:-temporal-ui-secret}"

# UI needs to connect to local temporal and casdoor
export TEMPORAL_GRPC_ADDRESS="127.0.0.1:7233"

# Update the config YAML on the fly
CONFIG_FILE="/home/ui-server/config/docker.yaml"

# Replace temporal address and casdoor URLs in config
sed -i "s|temporal:7233|127.0.0.1:7233|g" "$CONFIG_FILE"
sed -i "s|http://casdoor:8000|http://127.0.0.1:8000|g" "$CONFIG_FILE"
sed -i "s|http://{{ env \"TEMPORAL_HOST\" }}:8000|http://{{ env \"TEMPORAL_HOST\" }}:8000|g" "$CONFIG_FILE"

# Wait for temporal server to be available
echo "[ui] Waiting for Temporal server at 127.0.0.1:7233..."
max_attempts=60
attempt=0
while [ $attempt -lt $max_attempts ]; do
  if nc -z 127.0.0.1 7233 2>/dev/null; then
    echo "[ui] Temporal server is ready!"
    break
  fi
  attempt=$((attempt + 1))
  sleep 2
done

exec /home/ui-server/start-ui-server.sh
