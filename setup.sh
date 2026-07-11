#!/usr/bin/env bash
# setup.sh — Initialize local config files from .example templates.
# Run this after cloning the repo. Safe to re-run (won't overwrite existing files).
set -euo pipefail

cd "$(dirname "$0")"

declare -a TEMPLATES=(
  "docker-compose.yml"
  "docker-compose.override.yml"
  "temporal-server-custom/config_template.yaml"
  "ui-custom/overlays/server/config/docker.yaml"
  ".env"
)

echo "=== Temporal Standalone Setup ==="
echo ""

copied=0
skipped=0

for tmpl in "${TEMPLATES[@]}"; do
  example="${tmpl}.example"
  
  if [ ! -f "$example" ]; then
    # .env uses .env.example
    if [ "$tmpl" = ".env" ] && [ -f ".env.example" ]; then
      example=".env.example"
    else
      echo "  SKIP  $tmpl — no .example found"
      continue
    fi
  fi

  if [ -f "$tmpl" ]; then
    echo "  KEEP  $tmpl (already exists)"
    skipped=$((skipped + 1))
  else
    cp "$example" "$tmpl"
    echo "  COPY  $example → $tmpl"
    copied=$((copied + 1))
  fi
done

echo ""
echo "Done: $copied copied, $skipped kept."

if [ ! -f ".env" ]; then
  echo ""
  echo "⚠️  No .env file found!"
  echo "   Edit .env with your settings (TEMPORAL_HOST, CASDOOR_CLIENT_ID, etc.)"
fi

echo ""
echo "Next steps:"
echo "  1. Edit .env with your configuration"
echo "  2. Review docker-compose.yml (adjust ports/hosts if needed)"
echo "  3. Start services: docker compose up -d"
