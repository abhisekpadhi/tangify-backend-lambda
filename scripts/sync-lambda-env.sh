#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${1:-$ROOT/.env.lambda}"
FUNCTION_NAME="${LAMBDA_FUNCTION_NAME:-tangify-backend-lambda}"
AWS_REGION="${AWS_REGION:-ap-south-1}"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "Missing $ENV_FILE"
  echo "Copy .env.lambda.example to .env.lambda and fill in values."
  exit 1
fi

# shellcheck disable=SC1090
set -a
source "$ENV_FILE"
set +a

required=(LLM_API_KEY GOOGLE_SHEETS_API_KEY GOOGLE_SHEET_ID)
for key in "${required[@]}"; do
  if [[ -z "${!key:-}" ]]; then
    echo "Missing required env var in $ENV_FILE: $key"
    exit 1
  fi
done

vars=(
  "LLM_API_KEY=${LLM_API_KEY}"
  "GOOGLE_SHEETS_API_KEY=${GOOGLE_SHEETS_API_KEY}"
  "GOOGLE_SHEET_ID=${GOOGLE_SHEET_ID}"
)

if [[ -n "${GOOGLE_SHEET_NAME:-}" ]]; then
  vars+=("GOOGLE_SHEET_NAME=${GOOGLE_SHEET_NAME}")
fi
if [[ -n "${TANGIFY_BOOTSTRAP_SECRET:-}" ]]; then
  vars+=("TANGIFY_BOOTSTRAP_SECRET=${TANGIFY_BOOTSTRAP_SECRET}")
fi
if [[ -n "${TANGIFY_VENUE_ID:-}" ]]; then
  vars+=("TANGIFY_VENUE_ID=${TANGIFY_VENUE_ID}")
fi
if [[ -n "${CF_SECRET:-}" ]]; then
  vars+=("CF_SECRET=${CF_SECRET}")
fi
if [[ -n "${ABLY_KEY:-}" ]]; then
  vars+=("ABLY_KEY=${ABLY_KEY}")
fi
if [[ -n "${INVOICE_NUMBER_WORKER_URL_PROD:-}" ]]; then
  vars+=("INVOICE_NUMBER_WORKER_URL_PROD=${INVOICE_NUMBER_WORKER_URL_PROD}")
fi
if [[ -n "${INVOICE_NUMBER_WORKER_URL_DEV:-}" ]]; then
  vars+=("INVOICE_NUMBER_WORKER_URL_DEV=${INVOICE_NUMBER_WORKER_URL_DEV}")
fi
if [[ -n "${GUPSHUP_API_KEY:-}" ]]; then
  vars+=("GUPSHUP_API_KEY=${GUPSHUP_API_KEY}")
fi
if [[ -n "${GUPSHUP_SOURCE:-}" ]]; then
  vars+=("GUPSHUP_SOURCE=${GUPSHUP_SOURCE}")
fi
if [[ -n "${GUPSHUP_APP_NAME:-}" ]]; then
  vars+=("GUPSHUP_APP_NAME=${GUPSHUP_APP_NAME}")
fi
if [[ -n "${GUPSHUP_REWARD_POINT_TEMPLATE_ID:-}" ]]; then
  vars+=("GUPSHUP_REWARD_POINT_TEMPLATE_ID=${GUPSHUP_REWARD_POINT_TEMPLATE_ID}")
fi
if [[ -n "${GUPSHUP_POINTS_USED_TEMPLATE_ID:-}" ]]; then
  vars+=("GUPSHUP_POINTS_USED_TEMPLATE_ID=${GUPSHUP_POINTS_USED_TEMPLATE_ID}")
fi
if [[ -n "${GUPSHUP_OTP_TEMPLATE_ID:-}" ]]; then
  vars+=("GUPSHUP_OTP_TEMPLATE_ID=${GUPSHUP_OTP_TEMPLATE_ID}")
fi

joined=$(IFS=, ; echo "${vars[*]}")

echo "Updating Lambda env vars on $FUNCTION_NAME ($AWS_REGION)..."
aws lambda update-function-configuration \
  --function-name "$FUNCTION_NAME" \
  --region "$AWS_REGION" \
  --environment "Variables={$joined}" \
  >/dev/null

echo "Done. Current keys:"
aws lambda get-function-configuration \
  --function-name "$FUNCTION_NAME" \
  --region "$AWS_REGION" \
  --query 'Environment.Variables' \
  --output json
