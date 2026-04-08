#!/bin/sh
# Apply overlays to upstream Temporal UI
# Overlays are files that replace or add to the upstream source

set -e

WORKDIR="${WORKDIR:-/app}"
OVERLAYS_DIR="${OVERLAYS_DIR:-/overlays}"

echo "Applying overlays from $OVERLAYS_DIR to $WORKDIR"

cd "$WORKDIR"

# Copy all overlay files, preserving directory structure
if [ -d "$OVERLAYS_DIR" ]; then
    cp -r "$OVERLAYS_DIR"/* . 2>/dev/null || true
    echo "Overlays applied successfully!"
else
    echo "No overlays directory found, skipping."
fi
