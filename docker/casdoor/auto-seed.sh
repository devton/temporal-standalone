#!/bin/sh
# Auto-seed Casdoor with Temporal OIDC configuration
# Uses Casdoor's REST API with basic auth (default admin credentials)
set -e

CASDOOR_URL="http://localhost:8000"
ADMIN_USER="admin"
ADMIN_PASS="${CASDOOR_ADMIN_PASSWORD:-123}"

echo "[Seed] Authenticating with Casdoor..."
TOKEN=$(curl -sf "${CASDOOR_URL}/api/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\",\"organization\":\"built-in\",\"application\":\"app-built-in\",\"type\":\"code\"}" 2>/dev/null | jq -r '.data' || echo "")

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "[Seed] WARNING: Could not authenticate with Casdoor. Skipping auto-seed."
  echo "[Seed] You will need to configure Casdoor manually."
  exit 0
fi

AUTH_HEADER="Authorization: Bearer ${TOKEN}"

echo "[Seed] Creating 'temporal' organization..."
ORG_RESP=$(curl -sf "${CASDOOR_URL}/api/add-organization" \
  -H "Content-Type: application/json" \
  -H "${AUTH_HEADER}" \
  -d '{
    "owner": "admin",
    "name": "temporal",
    "displayName": "Temporal",
    "websiteUrl": "http://localhost:8080",
    "passwordType": "plain",
    "passwordOptions": ["AtLeastOneUppercase","AtLeastOneLowercase","AtLeastOneDigit","AtLeastOneSpecialChar"],
    "enableSoftDeletion": false
  }' 2>/dev/null || echo '{"data":"exists"}')

echo "[Seed] Creating 'temporal-ui' application..."
APP_RESP=$(curl -sf "${CASDOOR_URL}/api/add-application" \
  -H "Content-Type: application/json" \
  -H "${AUTH_HEADER}" \
  -d "{
    \"owner\": \"temporal\",
    \"name\": \"temporal-ui\",
    \"displayName\": \"Temporal UI\",
    \"organization\": \"temporal\",
    \"enablePassword\": true,
    \"enableCodeSignin\": false,
    \"enableSamlCompress\": false,
    \"isThirdParty\": false,
    \"tokenFormat\": \"JWT\",
    \"expireInHours\": 168,
    \"refreshExpireInHours\": 720,
    \"grantTypes\": [\"authorization_code\",\"password\",\"refresh_token\"],
    \"redirectUris\": [\"${TEMPORAL_UI_CALLBACK:-http://localhost:8080/auth/sso/callback}\"]
  }" 2>/dev/null || echo '{"data":"exists"}')

# Extract client ID and secret from the response
CLIENT_ID=$(echo "$APP_RESP" | jq -r '.data2.clientId // .data // empty' 2>/dev/null || echo "")
CLIENT_SECRET=$(echo "$APP_RESP" | jq -r '.data2.clientSecret // .data // empty' 2>/dev/null || echo "")

if [ -n "$CLIENT_ID" ] && [ "$CLIENT_ID" != "null" ] && [ "$CLIENT_ID" != "exists" ]; then
  echo "[Seed] Application created successfully!"
  echo "[Seed] CLIENT_ID: ${CLIENT_ID}"
  echo "[Seed] CLIENT_SECRET: ${CLIENT_SECRET}"
  echo "[Seed] Set these as CASDOOR_CLIENT_ID and CASDOOR_CLIENT_SECRET in Railway"
fi

echo "[Seed] Creating test user 'testuser'..."
curl -sf "${CASDOOR_URL}/api/add-user" \
  -H "Content-Type: application/json" \
  -H "${AUTH_HEADER}" \
  -d '{
    "owner": "temporal",
    "name": "testuser",
    "displayName": "Test User",
    "password": "Temporal123!",
    "email": "testuser@temporal.local",
    "emailVerified": true,
    "type": "normal-user",
    "isAdmin": false,
    "isForbidden": false,
    "signupApplication": "temporal-ui"
  }' > /dev/null 2>&1 || echo "[Seed] User may already exist, continuing..."

echo "[Seed] Auto-seed completed!"
echo ""
echo "========================================="
echo "  Casdoor OIDC Configuration"
echo "========================================="
echo "  Issuer:        ${CASDOOR_URL}"
echo "  Client ID:     ${CLIENT_ID:-check Casdoor dashboard}"
echo "  Client Secret: ${CLIENT_SECRET:-check Casdoor dashboard}"
echo "  Callback:      ${TEMPORAL_UI_CALLBACK:-http://localhost:8080/auth/sso/callback}"
echo ""
echo "  Test User: testuser / Temporal123!"
echo "========================================="
