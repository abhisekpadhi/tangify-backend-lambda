#!/usr/bin/env bash
# Send the Gupshup WhatsApp utility template `reward_point`.
# Usage:
#   ./send.sh 9876543210
#   ./send.sh 919876543210 12 112
set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="$DIR/.env"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE"
  echo "Copy .env.example to .env and fill GUPSHUP_API_KEY, GUPSHUP_SOURCE, GUPSHUP_APP_NAME."
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

API_KEY="${GUPSHUP_API_KEY:-}"
SOURCE="${GUPSHUP_SOURCE:-}"
APP_NAME="${GUPSHUP_APP_NAME:-}"
TEMPLATE_ID="${GUPSHUP_TEMPLATE_ID:-a8085178-7d66-4223-826d-25d89aa315d0}"

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
  echo "Usage: $0 <phone> [earned_points] [balance_points]"
  echo "  phone: 10-digit or 91XXXXXXXXXX"
  exit 1
fi

EARNED="${2:-12}"
BALANCE="${3:-112}"

digits="$(printf '%s' "$DEST_RAW" | tr -d '+[:space:]-')"
if [[ "$digits" =~ ^[0-9]{10}$ ]]; then
  digits="91${digits}"
fi
if [[ ! "$digits" =~ ^91[0-9]{10}$ ]]; then
  echo "Phone must be 10 digits or 91 + 10 digits, got: $DEST_RAW"
  exit 1
fi

source_digits="$(printf '%s' "$SOURCE" | tr -d '+[:space:]-')"

template_json="$(printf '{"id":"%s","params":["%s","%s"]}' "$TEMPLATE_ID" "$EARNED" "$BALANCE")"

echo "Sending reward_point -> ${digits} earned=${EARNED} balance=${BALANCE}"
echo "Template id ${TEMPLATE_ID}  (must be Approved in Gupshup)"

resp="$(curl -sS -w "\n%{http_code}" \
  -X POST 'https://api.gupshup.io/wa/api/v1/template/msg' \
  -H 'Cache-Control: no-cache' \
  -H "apikey: ${API_KEY}" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "channel=whatsapp" \
  --data-urlencode "source=${source_digits}" \
  --data-urlencode "destination=${digits}" \
  --data-urlencode "src.name=${APP_NAME}" \
  --data-urlencode "template=${template_json}")"

http_code="$(printf '%s' "$resp" | tail -n 1)"
body="$(printf '%s' "$resp" | sed '$d')"

echo "$body"
if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
  echo "HTTP ${http_code}"
  exit 1
fi
echo "HTTP ${http_code}"
