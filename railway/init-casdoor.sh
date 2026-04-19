#!/bin/bash
# Auto-seed Casdoor with Temporal OIDC config
# Runs in background, waits for Casdoor to be healthy, then seeds
set -e

CASDOOR_URL="http://127.0.0.1:8000"
TEMPORAL_UI_URL="http://${TEMPORAL_HOST}:8080"

echo "[casdoor-init] Waiting for Casdoor to be ready..."
max_attempts=120
attempt=0
while [ $attempt -lt $max_attempts ]; do
  if curl -sf "$CASDOOR_URL/.well-known/openid-configuration" > /dev/null 2>&1; then
    echo "[casdoor-init] Casdoor is ready!"
    break
  fi
  attempt=$((attempt + 1))
  sleep 2
done

if [ $attempt -eq $max_attempts ]; then
  echo "[casdoor-init] Timeout waiting for Casdoor, skipping seed"
  exit 0
fi

# Get CSRF token
CSRF_RESP=$(curl -sf -c /tmp/casdoor-cookies "$CASDOOR_URL/api/getcsrf" 2>/dev/null || echo '{}')
CSRF_TOKEN=$(echo "$CSRF_RESP" | jq -r '.token // empty' 2>/dev/null || echo "")

if [ -z "$CSRF_TOKEN" ]; then
  echo "[casdoor-init] Could not get CSRF token, skipping seed"
  exit 0
fi

echo "[casdoor-init] CSRF token obtained, seeding..."

# Login as admin
LOGIN_RESP=$(curl -sf -b /tmp/casdoor-cookies -c /tmp/casdoor-cookies \
  -X POST "$CASDOOR_URL/api/login" \
  -H "Content-Type: application/json" \
  -H "x-csrf-token: $CSRF_TOKEN" \
  -d '{"organization":"built-in","username":"admin","password":"123","type":"login","application":"app-built-in"}' 2>/dev/null || echo '{}')

OWNER=$(echo "$LOGIN_RESP" | jq -r '.owner // empty' 2>/dev/null || echo "")
if [ -z "$OWNER" ]; then
  echo "[casdoor-init] Login failed, skipping seed"
  exit 0
fi

echo "[casdoor-init] Logged in as admin"

# Create temporal organization
curl -sf -b /tmp/casdoor-cookies \
  -X POST "$CASDOOR_URL/api/add-organization" \
  -H "Content-Type: application/json" \
  -H "x-csrf-token: $CSRF_TOKEN" \
  -d '{"owner":"built-in","name":"temporal","displayName":"Temporal","websiteUrl":"'"$TEMPORAL_UI_URL"'"}' 2>/dev/null || echo "[casdoor-init] Org may already exist"

# Create temporal-ui application
APP_SECRET="${CASDOOR_CLIENT_SECRET:-temporal-ui-secret}"
curl -sf -b /tmp/casdoor-cookies \
  -X POST "$CASDOOR_URL/api/add-application" \
  -H "Content-Type: application/json" \
  -H "x-csrf-token: $CSRF_TOKEN" \
  -d '{
    "owner":"temporal",
    "name":"temporal-ui",
    "organization":"temporal",
    "clientId":"'${CASDOOR_CLIENT_ID:-temporal-ui}'",
    "clientSecret":"'"$APP_SECRET"'",
    "redirectUris":["'"$TEMPORAL_UI_URL"'/auth/sso/callback","'"$TEMPORAL_UI_URL"'"],
    "tokenFormat":"JWT",
    "expireInHours":168,
    "grantTypes":["authorization_code","refresh_token"],
    "enablePassword":true,
    "enableCode":true
  }' 2>/dev/null || echo "[casdoor-init] App may already exist"

# Create test user
curl -sf -b /tmp/casdoor-cookies \
  -X POST "$CASDOOR_URL/api/add-user" \
  -H "Content-Type: application/json" \
  -H "x-csrf-token: $CSRF_TOKEN" \
  -d '{
    "owner":"temporal",
    "name":"testuser",
    "displayName":"Test User",
    "password":"Temporal123!",
    "email":"testuser@temporal.local",
    "emailVerified":true,
    "type":"normal-user",
    "signupApplication":"temporal-ui"
  }' 2>/dev/null || echo "[casdoor-init] User may already exist"

echo "[casdoor-init] Seed complete!"
echo "[casdoor-init]   Org: temporal"
echo "[casdoor-init]   App: temporal-ui / $APP_SECRET"
echo "[casdoor-init]   User: testuser / Temporal123!"
