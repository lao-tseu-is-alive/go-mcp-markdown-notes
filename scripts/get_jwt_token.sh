#!/usr/bin/env bash
set -euo pipefail

# Helper script to fetch a JWT token from the running auth server
# Usage: ./scripts/get_jwt_token.sh [/path/to/.env]

ENV_FILE="${1:-.env}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Error: environment file '$ENV_FILE' not found." >&2
  exit 1
fi

# Sourcing variables and exporting them
set -o allexport
source "$ENV_FILE"
set +o allexport

ADMIN_USER="${ADMIN_USER:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
AUTH_PORT="${PORT:-9090}"
AUTH_HOST="${AUTH_HOST:-localhost}"

if [ -z "$ADMIN_USER" ] || [ -z "$ADMIN_PASSWORD" ]; then
  echo "Error: ADMIN_USER and ADMIN_PASSWORD must be defined in '$ENV_FILE'." >&2
  exit 1
fi

# Calculate SHA256 of the admin password
PASSWORD_HASH=$(echo -n "$ADMIN_PASSWORD" | sha256sum | awk '{print $1}')

# Call auth server login endpoint
RESPONSE=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"$ADMIN_USER\", \"password_hash\": \"$PASSWORD_HASH\"}" \
  "http://$AUTH_HOST:$AUTH_PORT/login")

# Parse token from JSON response
TOKEN=$(echo "$RESPONSE" | jq -r '.token // empty')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "Error: failed to fetch token from auth server." >&2
  echo "Server response: $RESPONSE" >&2
  exit 1
fi

echo "$TOKEN"
