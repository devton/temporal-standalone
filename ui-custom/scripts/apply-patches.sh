#!/bin/sh
# Apply custom patches to upstream Temporal UI
# This script runs inside the Docker container

set -e

# In Docker: we're at /app with upstream/ and /patches available
WORKDIR="${WORKDIR:-/app}"
PATCHES_DIR="${PATCHES_DIR:-/patches}"

echo "Applying patches from $PATCHES_DIR to $WORKDIR"

cd "$WORKDIR"

# Apply each patch in order using 'patch' instead of 'git apply'
for patch in "$PATCHES_DIR"/*.patch; do
    if [ -f "$patch" ]; then
        echo "Applying $(basename "$patch")..."
        patch -p1 < "$patch" || {
            echo "Failed to apply patch: $patch"
            exit 1
        }
    fi
done

echo "All patches applied successfully!"
