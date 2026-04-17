#!/bin/sh
# Setup namespaces with retention and archive
# Re-runs on every restart to ensure settings persist

echo "Waiting for Temporal server to be ready..."
for i in $(seq 1 30); do
  if temporal operator cluster health --address temporal:7233 2>/dev/null; then
    echo "Server is ready!"
    break
  fi
  echo "Attempt $i/30 - waiting for server..."
  sleep 2
done

echo "Configuring default namespace with 30-day retention and archive..."

temporal operator namespace update default \
  --address temporal:7233 \
  --retention 30d \
  --history-archival-state enabled \
  --visibility-archival-state enabled 2>&1 || true

temporal operator namespace list --address temporal:7233 2>&1 || true

echo "Done!"
