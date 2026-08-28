#!/usr/bin/env bash
# Send an SMS via SMS Gate cloud (https://docs.sms-gate.app).
# Usage:
#   ./send.sh 9876543210
#   ./send.sh 919876543210 "House of Odia: you used 12 points. Balance 40."
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="$DIR/.env"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE"
  echo "Copy .env.example to .env and fill SMS_GATE_SERVER, SMS_GATE_USERNAME, SMS_GATE_PASSWORD."
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

SERVER="${SMS_GATE_SERVER:-}"
USERNAME="${SMS_GATE_USERNAME:-}"
PASSWORD="${SMS_GATE_PASSWORD:-}"

if [[ -z "$SERVER" || -z "$USERNAME" || -z "$PASSWORD" ]]; then
  echo "SMS_GATE_SERVER, SMS_GATE_USERNAME, and SMS_GATE_PASSWORD are required in .env"
  exit 1
fi

DEST_RAW="${1:-}"
if [[ -z "$DEST_RAW" ]]; then
  echo "Usage: $0 <phone> [message]"
  echo "  phone: 10-digit or 91XXXXXXXXXX"
  exit 1
fi

shift || true
MESSAGE="${*:-House of Odia test: you used 12 points and earned 8. Balance 40.}"

digits="$(printf '%s' "$DEST_RAW" | tr -d '+[:space:]-')"
if [[ "$digits" =~ ^[0-9]{10}$ ]]; then
  digits="91${digits}"
fi
if [[ ! "$digits" =~ ^91[0-9]{10}$ ]]; then
  echo "Phone must be 10 digits or 91 + 10 digits, got: $DEST_RAW"
  exit 1
fi

server="${SERVER%/}"
url="${server}/3rdparty/v1/message"

payload="$(python3 -c '
import json, sys
print(json.dumps({
  "textMessage": {"text": sys.argv[1]},
  "phoneNumbers": ["+" + sys.argv[2]],
}))
' "$MESSAGE" "$digits")"

echo "Sending SMS via ${url} -> +${digits}"

resp="$(curl -sS -w "\n%{http_code}" \
  -u "${USERNAME}:${PASSWORD}" \
  -H 'Content-Type: application/json' \
  -d "${payload}" \
  "${url}")"

http_code="$(printf '%s' "$resp" | tail -n 1)"
body="$(printf '%s' "$resp" | sed '$d')"

echo "$body"
if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
  echo "HTTP ${http_code}"
  exit 1
fi
echo "HTTP ${http_code}"
