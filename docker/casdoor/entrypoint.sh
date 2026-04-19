#!/bin/sh
set -e

echo "[Casdoor] Starting Casdoor with auto-configuration..."

# Wait for PostgreSQL to be available
echo "[Casdoor] Waiting for PostgreSQL at ${PGHOST}:${PGPORT:-5432}..."
max_attempts=60
attempt=0
while [ $attempt -lt $max_attempts ]; do
  if pg_isready -h "${PGHOST}" -p "${PGPORT:-5432}" -U "${PGUSER:-temporal}" > /dev/null 2>&1; then
    echo "[Casdoor] PostgreSQL is ready!"
    break
  fi
  attempt=$((attempt + 1))
  echo "  Attempt $attempt/$max_attempts..."
  sleep 2
done

if [ $attempt -eq $max_attempts ]; then
  echo "[Casdoor] ERROR: PostgreSQL not available after $max_attempts attempts"
  exit 1
fi

# Ensure casdoor database and user exist
echo "[Casdoor] Ensuring casdoor database exists..."
PGPASSWORD="${PGPASSWORD}" psql -h "${PGHOST}" -p "${PGPORT:-5432}" -U "${PGUSER:-temporal}" -d postgres > /dev/null 2>&1 <<-EOSQL || true
    DO \$do\$ BEGIN
      IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'casdoor') THEN
        CREATE USER casdoor WITH PASSWORD '${CASDOOR_DB_PASSWORD:-casdoor_secret}';
      END IF;
    END \$do\$;
    SELECT 'CREATE DATABASE casdoor OWNER casdoor' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'casdoor')\gexec
    GRANT ALL PRIVILEGES ON DATABASE casdoor TO casdoor;
EOSQL

echo "[Casdoor] Database ready. Starting Casdoor server..."

# Start Casdoor in background
/bin/entrypoint.sh &
CASDOOR_PID=$!

# Wait for Casdoor to be healthy
echo "[Casdoor] Waiting for Casdoor to be ready..."
attempt=0
while [ $attempt -lt 60 ]; do
  if curl -sf http://localhost:8000/api/get-health > /dev/null 2>&1; then
    echo "[Casdoor] Casdoor is ready!"
    break
  fi
  attempt=$((attempt + 1))
  sleep 2
done

if [ $attempt -eq 60 ]; then
  echo "[Casdoor] ERROR: Casdoor failed to start"
  exit 1
fi

# Auto-seed: configure Casdoor via API
echo "[Casdoor] Running auto-seed..."
sh /auto-seed.sh || echo "[Casdoor] Auto-seed completed with warnings"

# Keep container running
echo "[Casdoor] Casdoor is running."
wait $CASDOOR_PID
