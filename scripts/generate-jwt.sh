#!/bin/bash
# Generate JWT tokens for Temporal authentication testing
# Requires: jq, openssl

set -e

JWT_DIR="$(dirname "$0")/../config/jwt"
PRIVATE_KEY="$JWT_DIR/key.pem"
PUBLIC_KEY="$JWT_DIR/key.pub"

# Create JWT directory if needed
mkdir -p "$JWT_DIR"

# Generate RSA keys if not exist
if [ ! -f "$PRIVATE_KEY" ]; then
    echo "Generating RSA key pair..."
    openssl genrsa -out "$PRIVATE_KEY" 2048
    openssl rsa -in "$PRIVATE_KEY" -pubout -out "$PUBLIC_KEY"
    echo "Keys generated in $JWT_DIR/"
fi

# JWT configuration
ISSUER="temporal-standalone"
AUDIENCE="temporal"
SUBJECT="test-user"
EXPIRATION=$(( $(date +%s) + 86400 * 7 ))  # 7 days

# Base64 encode (URL safe)
base64_encode() {
    echo -n "$1" | openssl base64 -e | tr -d '=' | tr '/+' '_-' | tr -d '\n'
}

# Create JWT header
HEADER='{"alg":"RS256","typ":"JWT"}'

# Create JWT payload
PAYLOAD=$(cat <<EOF
{
  "iss": "$ISSUER",
  "aud": "$AUDIENCE",
  "sub": "$SUBJECT",
  "exp": $EXPIRATION,
  "iat": $(date +%s),
  "name": "Test User",
  "email": "test@example.com"
}
EOF
)

echo ""
echo "=== JWT Token Generator for Temporal ==="
echo ""
echo "Payload:"
echo "$PAYLOAD" | jq .
echo ""

# Encode header and payload
HEADER_B64=$(base64_encode "$HEADER")
PAYLOAD_B64=$(base64_encode "$PAYLOAD")

# Create signature
SIGNATURE=$(echo -n "${HEADER_B64}.${PAYLOAD_B64}" | openssl dgst -sha256 -sign "$PRIVATE_KEY" | openssl base64 | tr -d '=' | tr '/+' '_-' | tr -d '\n')

# Combine into JWT
TOKEN="${HEADER_B64}.${PAYLOAD_B64}.${SIGNATURE}"

echo "=== Generated JWT Token ==="
echo ""
echo "$TOKEN"
echo ""

# Save token to file
TOKEN_FILE="$JWT_DIR/token.txt"
echo "$TOKEN" > "$TOKEN_FILE"
echo "Token saved to: $TOKEN_FILE"
echo ""

# Print usage instructions
echo "=== Usage ==="
echo ""
echo "1. Enable JWT auth in docker-compose.yml (uncomment auth settings)"
echo ""
echo "2. Restart Temporal:"
echo "   docker compose restart temporal temporal-ui"
echo ""
echo "3. Use the token with tctl:"
echo "   export TEMPORAL_CLI_AUTH_TOKEN=\$TOKEN"
echo "   temporal operator namespace list"
echo ""
echo "4. Use the token in your code:"
echo "   client.Dial(client.Options{"
echo "       Auth: &client.AuthOptions{Token: \"\$TOKEN\"}"
echo "   })"
echo ""
