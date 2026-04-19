#!/bin/bash
# Start Temporal server (uses auto-setup image's built-in start script)
set -e

# Ensure config is rendered
export BIND_ON_IP="${BIND_ON_IP:-127.0.0.1}"
export TEMPORAL_ADDRESS="${TEMPORAL_ADDRESS:-127.0.0.1:7233}"

# Render config template
if command -v dockerize &> /dev/null; then
  dockerize -template /etc/temporal/config/config_template.yaml:/etc/temporal/config/docker.yaml
fi

exec /etc/temporal/start-temporal.sh
