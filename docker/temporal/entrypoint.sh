#!/bin/bash
set -e

echo "[Temporal] Waiting for PostgreSQL at ${POSTGRES_SEEDS}:${DB_PORT:-5432}..."

# Wait for PostgreSQL
max_attempts=60
attempt=0
while [ $attempt -lt $max_attempts ]; do
  if pg_isready -h "${POSTGRES_SEEDS}" -p "${DB_PORT:-5432}" -U "${POSTGRES_USER:-temporal}" > /dev/null 2>&1; then
    echo "[Temporal] PostgreSQL is ready!"
    break
  fi
  attempt=$((attempt + 1))
  echo "  Attempt $attempt/$max_attempts..."
  sleep 2
done

if [ $attempt -eq $max_attempts ]; then
  echo "[Temporal] ERROR: PostgreSQL not available"
  exit 1
fi

# Ensure databases exist
echo "[Temporal] Ensuring 'temporal' and 'temporal_visibility' databases exist..."
PGPASSWORD="${POSTGRES_PWD}" psql -h "${POSTGRES_SEEDS}" -p "${DB_PORT:-5432}" -U "${POSTGRES_USER:-temporal}" -d postgres > /dev/null 2>&1 <<-EOSQL
    SELECT 'CREATE DATABASE temporal' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'temporal')\gexec
    SELECT 'CREATE DATABASE temporal_visibility' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'temporal_visibility')\gexec
EOSQL

echo "[Temporal] Starting auto-setup (schema migration + server)..."

# Run the original auto-setup entrypoint
exec /entrypoint.sh
