#!/bin/bash
set -e

echo "============================================="
echo "  Temporal Standalone - Railway Monolith"
echo "============================================="

# Parse DATABASE_URL if provided (Railway gives this automatically)
if [ -n "${DATABASE_URL}" ]; then
  echo "[entrypoint] Parsing DATABASE_URL..."
  # Format: postgresql://user:pass@host:port/dbname
  export POSTGRES_SEEDS=$(echo "$DATABASE_URL" | sed -n 's|.*@\(.*\):.*|\1|p')
  export DB_PORT=$(echo "$DATABASE_URL" | sed -n 's|.*:\([0-9]*\)/.*|\1|p')
  export POSTGRES_USER=$(echo "$DATABASE_URL" | sed -n 's|.*://\(.*\):.*@.*|\1|p')
  export POSTGRES_PWD=$(echo "$DATABASE_URL" | sed -n 's|.*://.*:\(.*\)@.*|\1|p')
  export DBNAME=$(echo "$DATABASE_URL" | sed -n 's|.*/\(.*\)|\1|p')
  [ -z "$DBNAME" ] && DBNAME="temporal"
  
  # Also set for psql commands
  export PGHOST="$POSTGRES_SEEDS"
  export PGPORT="$DB_PORT"
  export PGUSER="$POSTGRES_USER"
  export PGPASSWORD="$POSTGRES_PWD"
fi

# Defaults
export DB="${DB:-postgres12}"
export DBNAME="${DBNAME:-temporal}"
export VISIBILITY_DBNAME="${VISIBILITY_DBNAME:-temporal_visibility}"
export POSTGRES_SEEDS="${POSTGRES_SEEDS:-localhost}"
export DB_PORT="${DB_PORT:-5432}"
export POSTGRES_USER="${POSTGRES_USER:-temporal}"
export POSTGRES_PWD="${POSTGRES_PWD:-temporal}"
export ENABLE_ES="${ENABLE_ES:-false}"
export SKIP_DEFAULT_NAMESPACE_CREATION="${SKIP_DEFAULT_NAMESPACE_CREATION:-false}"
export SKIP_ADD_CUSTOM_SEARCH_ATTRIBUTES="${SKIP_ADD_CUSTOM_SEARCH_ATTRIBUTES:-true}"
export DEFAULT_NAMESPACE="${DEFAULT_NAMESPACE:-default}"

# Temporal host for URLs
export TEMPORAL_HOST="${TEMPORAL_HOST:-${RAILWAY_PUBLIC_DOMAIN:-localhost}}"
export CASDOOR_CLIENT_ID="${CASDOOR_CLIENT_ID:-temporal-ui}"
export CASDOOR_CLIENT_SECRET="${CASDOOR_CLIENT_SECRET:-temporal-ui-secret}"

echo "[entrypoint] PostgreSQL: ${POSTGRES_SEEDS}:${DB_PORT}"
echo "[entrypoint] Databases: ${DBNAME}, ${VISIBILITY_DBNAME}"
echo "[entrypoint] Temporal Host: ${TEMPORAL_HOST}"

# ---- Create Casdoor database ----
echo "[entrypoint] Creating casdoor database..."
PGPASSWORD="$POSTGRES_PWD" psql -h "$POSTGRES_SEEDS" -p "$DB_PORT" -U "$POSTGRES_USER" -d postgres \
  -tc "SELECT 1 FROM pg_database WHERE datname = 'casdoor'" | grep -q 1 || \
  PGPASSWORD="$POSTGRES_PWD" psql -h "$POSTGRES_SEEDS" -p "$DB_PORT" -U "$POSTGRES_USER" -d postgres \
  -c "CREATE DATABASE casdoor" 2>/dev/null || true

# ---- Run Temporal auto-setup (schema migration) ----
echo "[entrypoint] Running Temporal schema setup..."
/etc/temporal/auto-setup.sh || echo "[entrypoint] Schema setup completed (some warnings ok)"

# ---- Seed Casdoor (background, waits for casdoor to start) ----
/opt/casdoor/init-casdoor.sh &

# ---- Start everything via supervisord ----
echo "[entrypoint] Starting all services (supervisord)..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
