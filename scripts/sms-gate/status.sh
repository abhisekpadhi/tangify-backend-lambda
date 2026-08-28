#!/usr/bin/env bash
# Check whether the SMS Gate Android app last checked in (cloud lastSeen).
# Usage:
#   ./status.sh
#   ./status.sh 20    # treat as stale if lastSeen older than N minutes (default 20)
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
MAX_AGE_MIN="${1:-${SMS_GATE_ACTIVE_WITHIN_MINUTES:-20}}"

if [[ -z "$SERVER" || -z "$USERNAME" || -z "$PASSWORD" ]]; then
  echo "SMS_GATE_SERVER, SMS_GATE_USERNAME, and SMS_GATE_PASSWORD are required in .env"
  exit 1
fi
if [[ ! "$MAX_AGE_MIN" =~ ^[0-9]+$ ]]; then
  echo "Active window must be minutes as an integer, got: $MAX_AGE_MIN"
  exit 1
fi

server="${SERVER%/}"
url="${server}/3rdparty/v1/devices"

resp="$(curl -sS -w "\n%{http_code}" \
  -u "${USERNAME}:${PASSWORD}" \
  "${url}")"

http_code="$(printf '%s' "$resp" | tail -n 1)"
body="$(printf '%s' "$resp" | sed '$d')"

if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
  echo "$body"
  echo "HTTP ${http_code}"
  exit 1
fi

python3 -c '
import json, sys
from datetime import datetime, timezone

body = sys.argv[1]
max_age_min = int(sys.argv[2])
devices = json.loads(body)
if not isinstance(devices, list):
    devices = [devices] if devices else []
if not devices:
    print("No devices registered")
    sys.exit(1)

now = datetime.now(timezone.utc)
any_active = False
for d in devices:
    last = d.get("lastSeen") or ""
    age = None
    active = False
    if last:
        ts = datetime.fromisoformat(last.replace("Z", "+00:00"))
        if ts.tzinfo is None:
            ts = ts.replace(tzinfo=timezone.utc)
        age = (now - ts).total_seconds() / 60.0
        active = age <= max_age_min
    if active:
        any_active = True
    state = "active" if active else "stale"
    age_s = "unknown" if age is None else "{:.1f} min ago".format(age)
    dev_id = d.get("id") or "?"
    name = d.get("name") or "-"
    print("\t".join([state, "id=" + str(dev_id), "name=" + str(name), "lastSeen=" + (last or "-"), age_s]))

print(f"window={max_age_min} min (cloud lastSeen can lag ~15 min)")
sys.exit(0 if any_active else 2)
' "$body" "$MAX_AGE_MIN"
