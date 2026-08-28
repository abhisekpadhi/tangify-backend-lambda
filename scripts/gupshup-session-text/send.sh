#!/usr/bin/env bash
# Send a free-form WhatsApp session message (not a template).
# Requires the destination to have messaged this WABA within the last 24h.
# Usage:
#   ./send.sh 9876543210
#   ./send.sh 919876543210 "You earned 12 points. Balance 40."
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="$DIR/.env"
if [[ ! -f "$ENV_FILE" && -f "$DIR/../gupshup-reward-point/.env" ]]; then
  ENV_FILE="$DIR/../gupshup-reward-point/.env"
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $DIR/.env"
  echo "Copy .env.example to .env, or reuse ../gupshup-reward-point/.env"
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

API_KEY="${GUPSHUP_API_KEY:-}"
SOURCE="${GUPSHUP_SOURCE:-}"
APP_NAME="${GUPSHUP_APP_NAME:-}"

if [[ -z "$API_KEY" || -z "$SOURCE" ]]; then
  echo "GUPSHUP_API_KEY and GUPSHUP_SOURCE are required in .env"
  exit 1
fi
if [[ -z "$APP_NAME" ]]; then
  echo "GUPSHUP_APP_NAME is required in .env (Gupshup app name / src.name)"
  exit 1
fi

DEST_RAW="${1:-}"
if [[ -z "$DEST_RAW" ]]; then
  echo "Usage: $0 <phone> [message]"
  echo "  phone: 10-digit or 91XXXXXXXXXX"
  echo "  First WhatsApp the business number, then run this within 24h."
  exit 1
fi

TEXT="${2:-House of Odia: session test. If you got this, the 24h window is open.}"

digits="$(printf '%s' "$DEST_RAW" | tr -d '+[:space:]-')"
if [[ "$digits" =~ ^[0-9]{10}$ ]]; then
  digits="91${digits}"
fi
if [[ ! "$digits" =~ ^91[0-9]{10}$ ]]; then
  echo "Phone must be 10 digits or 91 + 10 digits, got: $DEST_RAW"
  exit 1
fi

source_digits="$(printf '%s' "$SOURCE" | tr -d '+[:space:]-')"
message_json="$(python3 -c 'import json,sys; print(json.dumps({"type":"text","text":sys.argv[1]}))' "$TEXT")"

echo "Sending session text -> ${digits}"
echo "If this fails with a session/window error, the user has not messaged ${source_digits} in the last 24h."

resp="$(curl -sS -w "\n%{http_code}" \
  -X POST 'https://api.gupshup.io/wa/api/v1/msg' \
  -H 'Cache-Control: no-cache' \
  -H "apikey: ${API_KEY}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "channel=whatsapp" \
  --data-urlencode "source=${source_digits}" \
  --data-urlencode "destination=${digits}" \
  --data-urlencode "src.name=${APP_NAME}" \
  --data-urlencode "message=${message_json}")"

http_code="$(printf '%s' "$resp" | tail -n 1)"
body="$(printf '%s' "$resp" | sed '$d')"

echo "$body"
if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
  echo "HTTP ${http_code}"
  exit 1
fi
echo "HTTP ${http_code}"
