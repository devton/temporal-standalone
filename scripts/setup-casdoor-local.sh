#!/bin/bash
# Setup Casdoor for Temporal OIDC
# Run this after Casdoor is running

set -e

CASDOOR_URL="http://192.168.2.68:8000"
PUBLIC_IP="192.168.2.68"

echo "=== Casdoor Setup for Temporal ==="

# Step 1: Login as admin
echo "1. Logging in as admin..."
LOGIN_RESPONSE=$(curl -sf -X POST "$CASDOOR_URL/api/login" \
  -H "Content-Type: "application/json"" \
  -d '{"organization":"built-in","username":"admin","password":"123"}' \
  -c /tmp/casdoor-cookies.txt -b /tmp/casdoor-cookies.txt 2>/dev/null || echo '{"status":"error"}')

if echo "$LOGIN_RESPONSE" | grep -q '"status":"ok"'; then
  echo "   Login successful!"
else
  echo "   Login response: $LOGIN_RESPONSE"
  echo "   Continuing anyway..."
fi

# Step 2: Create temporal organization
echo "2. Creating temporal organization..."
curl -sf -X POST "$CASDOOR_URL/api/add-organization" \
  -H "Content-Type: application/json" \
  -b /tmp/casdoor-cookies.txt \
  -d '{
    "owner": "admin",
    "name": "temporal",
    "displayName": "Temporal",
    "websiteUrl": "http://'"$PUBLIC_IP"':8080",
    "passwordType": "plain",
    "passwordOptions": ["Plain"],
    "enableSoftDeletion": false,
    "isProfilePublic": false
  }' 2>/dev/null || echo "   Organization may already exist"

# Step 3: Create temporal-ui application
echo "3. Creating temporal-ui application..."
curl -sf -X POST "$CASDOOR_URL/api/add-application" \
  -H "Content-Type: application/json" \
  -b /tmp/casdoor-cookies.txt \
  -d '{
    "owner": "temporal",
    "name": "temporal-ui",
    "displayName": "Temporal UI",
    "enablePassword": true,
    "homepageUrl": "http://'"$PUBLIC_IP"':8080",
    "description": "Temporal Workflow Engine UI",
    "redirectUris": ["http://'"$PUBLIC_IP"':8080/auth/callback"],
    "tokenFormat": "JWT",
    "expireInHours": 168,
    "refreshExpireInHours": 168,
    "organization": "temporal",
    "enableSignUp": false,
    "clientId": "temporal-ui",
    "clientSecret": "temporal-ui-secret",
    "grantTypes": ["authorization_code", "refresh_token"],
    "applicationType": "web"
  }' 2>/dev/null || echo "   Application may already exist"

# Step 4: Create test user
echo "4. Creating test user..."
curl -sf -X POST "$CASDOOR_URL/api/add-user" \
  -H "Content-Type: application/json" \
  -b /tmp/casdoor-cookies.txt \
  -d '{
    "owner": "temporal",
    "name": "testuser",
    "type": "normal-user",
    "password": "Temporal123!",
    "displayName": "Test User",
    "email": "testuser@temporal.local",
    "emailVerified": true,
    "isDefaultAvatar": true,
    "isAdmin": false,
    "isForbidden": false,
    "isDeleted": false
  }' 2>/dev/null || echo "   User may already exist"

# Cleanup
rm -f /tmp/casdoor-cookies.txt

echo ""
echo "========================================"
echo "Casdoor Setup Complete!"
echo "========================================"
echo ""
echo "OIDC Configuration:"
echo "  Issuer URL:     $CASDOOR_URL"
echo "  Client ID:      temporal-ui"
echo "  Client Secret:  temporal-ui-secret"
echo "  Callback URL:   http://$PUBLIC_IP:8080/auth/callback"
echo ""
echo "Test User:"
echo "  Username: testuser"
echo "  Password: Temporal123!"
echo "  Organization: temporal"
echo ""
echo "Access URLs:"
echo "  Casdoor Admin:  $CASDOOR_URL"
echo "  Temporal UI:    http://$PUBLIC_IP:8080"
echo ""
