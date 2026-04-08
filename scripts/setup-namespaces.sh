#!/bin/sh
# Setup namespaces with retention and archive

export TEMPORAL_CLI_ADDRESS="temporal:7233"

echo "Configuring default namespace with 30-day retention and archive..."

temporal operator namespace update default \
  --retention 30d \
  --history-archival-state enabled \
  --visibility-archival-state enabled 2>&1 || true

temporal operator namespace list 2>&1 || true

echo "Done!"
