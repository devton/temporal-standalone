#!/bin/bash
# Setup Casdoor for Temporal OIDC
# This script verifies Casdoor is running and shows setup instructions

set -e

# Configuration
CASDOOR_URL="${CASDOOR_URL:-http://192.168.2.68:8000}"
TEMPORAL_UI_URL="${TEMPORAL_UI_URL:-http://192.168.2.68:8080}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Wait for Casdoor to be ready
wait_for_casdoor() {
    log_info "Waiting for Casdoor to be ready..."
    local max_attempts=60
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "$CASDOOR_URL/.well-known/openid-configuration" > /dev/null 2>&1; then
            log_info "Casdoor is ready!"
            return 0
        fi
        attempt=$((attempt + 1))
        echo "  Attempt $attempt/$max_attempts..."
        sleep 2
    done
    
    log_error "Timeout waiting for Casdoor"
    return 1
}

# Main
main() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Casdoor Setup for Temporal OIDC${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    wait_for_casdoor || exit 1
    
    echo ""
    log_info "Casdoor is running at $CASDOOR_URL"
    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}  Manual Setup Required${NC}"
    echo -e "${YELLOW}========================================${NC}"
    echo ""
    echo "The Casdoor API requires browser-based authentication."
    echo "Please follow these steps to configure OIDC:"
    echo ""
    echo -e "${GREEN}Step 1: Access Casdoor Admin${NC}"
    echo "  URL: $CASDOOR_URL"
    echo "  User: admin"
    echo "  Pass: 123"
    echo "  Org:  built-in"
    echo ""
    echo -e "${GREEN}Step 2: Create Organization${NC}"
    echo "  1. Go to Organizations → Add"
    echo "  2. Name: temporal"
    echo "  3. DisplayName: Temporal"
    echo "  4. Website: $TEMPORAL_UI_URL"
    echo "  5. Click Save"
    echo ""
    echo -e "${GREEN}Step 3: Create Application${NC}"
    echo "  1. Go to Applications → Add"
    echo "  2. Organization: temporal"
    echo "  3. Name: temporal-ui"
    echo "  4. Client ID: temporal-ui"
    echo "  5. Client Secret: temporal-ui-secret"
    echo "  6. Redirect URLs:"
    echo "     - $TEMPORAL_UI_URL/auth/callback"
    echo "     - $TEMPORAL_UI_URL"
    echo "  7. Token format: JWT"
    echo "  8. Expire in hours: 168"
    echo "  9. Grant types: authorization_code, refresh_token"
    echo "  10. Click Save"
    echo ""
    echo -e "${GREEN}Step 4: Create Test User${NC}"
    echo "  1. Go to Users → Add"
    echo "  2. Organization: temporal"
    echo "  3. Name: testuser"
    echo "  4. Password: Temporal123!"
    echo "  5. Email: testuser@temporal.local"
    echo "  6. Email verified: true"
    echo "  7. Click Save"
    echo ""
    echo -e "${GREEN}Step 5: Test Login${NC}"
    echo "  1. Access: $TEMPORAL_UI_URL"
    echo "  2. Login with:"
    echo "     - Organization: temporal"
    echo "     - Username: testuser"
    echo "     - Password: Temporal123!"
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "OIDC Configuration Summary:"
    echo "  Issuer:        $CASDOOR_URL"
    echo "  Client ID:     temporal-ui"
    echo "  Client Secret: temporal-ui-secret"
    echo "  Callback:      $TEMPORAL_UI_URL/auth/callback"
    echo ""
}

main "$@"
