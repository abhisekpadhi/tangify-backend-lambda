# invoice-number-cf-worker

Cloudflare Worker that allocates sequential invoice numbers from a `bill_id`.

Used by `tangify-backend-lambda` when saving bills via `PUT /api/v1/billing/bills/with-line-items`. The Lambda does **not** call this worker directly from the frontend; it is invoked server-side during bill creation.

## Workers

| Environment | Worker name | URL |
|-------------|-------------|-----|
| Production | `invoice-number-cf-worker` | `https://invoice-number-cf-worker.subnub.workers.dev/` |
| Dev | `invoice-number-cf-worker-dev` | `https://invoice-number-cf-worker-dev.subnub.workers.dev/` |

Prod and dev use **separate Durable Object namespaces**, so local/testing does not consume production invoice sequence numbers.

## Behavior

- `POST /` with body `{ "bill_id": "bill_123" }`
- Resolves current UTC year (for example, `2026`)
- Routes request to a Durable Object instance keyed by that year
- Durable Object allocates an auto-increment sequence for that year
- Stores both mappings in Durable Object storage in one transaction:
  - `bill:{bill_id}` → invoice payload
  - `inv:{invoice_number}` → invoice payload
- Returns:
  - `invoice_number` (for example, `2026-000001`)
  - `bill_id`
  - `year`
  - `sequence`

If `bill_id` already has an invoice, the worker returns the existing mapping (idempotent by bill).

## Lambda integration

Lambda selects the worker URL from request environment:

- Header `X-Tangify-Environment: dev` → dev worker + `dev-tangify_bills_with_line_items` + `dev-tangify_points_wallet`
- Otherwise → prod worker + `tangify_bills_with_line_items` + `tangify_points_wallet`

Required Lambda env vars:

```bash
INVOICE_NUMBER_WORKER_URL_PROD=https://invoice-number-cf-worker.subnub.workers.dev/
INVOICE_NUMBER_WORKER_URL_DEV=https://invoice-number-cf-worker-dev.subnub.workers.dev/
```

Sync via repo root:

```bash
cp .env.lambda.example .env.lambda   # add invoice worker URLs
./scripts/sync-lambda-env.sh
```

The frontend Next.js route `/api/bills` forwards `X-Tangify-Environment` from `TANGIFY_BILLING_ENV` (`dev` locally, `production` on Vercel).

## Setup

1. Install dependencies:

```bash
npm install
```

2. Run locally (prod config):

```bash
npm run dev
```

3. Run locally (dev config):

```bash
npm run dev:worker
```

## Deploy

Production:

```bash
npm run deploy
# or
./deploy.sh
```

Dev (separate invoice sequence):

```bash
npm run deploy:dev
# or
./deploy-dev.sh
```

## Smoke test

Direct worker test (dev):

```bash
curl -s -X POST https://invoice-number-cf-worker-dev.subnub.workers.dev/ \
  -H 'Content-Type: application/json' \
  -d '{"bill_id":"bill_dev_smoke_001"}'
```

Retry with the same `bill_id` — response should be identical (idempotent).

End-to-end via Lambda (requires JWT):

```bash
API_URL="https://<your-lambda-url>"
TOKEN="<jwt>"

curl -s -X PUT "$API_URL/api/v1/billing/bills/with-line-items" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'X-Tangify-Environment: dev' \
  -d '{
    "state_key": "order-session:test-001::checkout",
    "session_id": "test-session",
    "table_ids": ["T1"],
    "line_items": [{"name": "Saag swag", "quantity": 1, "price": 5900}],
    "taxes": [
      {"id": "cgst", "name": "CGST", "rate_in_bps": 250, "amount_in_paise": 148},
      {"id": "sgst", "name": "SGST", "rate_in_bps": 250, "amount_in_paise": 148}
    ],
    "payment_method": "cash_or_upi",
    "payment_status": "pending"
  }'
```

Expected: response `id` is the allocated invoice number (for example, `2026-000003`).

Bruno requests: `apidocs-bruno/tangify_billing_apis/billing/cloudflare - invoice no.bru` and `cloudflare - invoice no (dev).bru`.

## Project layout

```bash
invoice-number-cf-worker/
├── src/index.ts       # Worker + InvoiceYearCounter Durable Object
├── wrangler.toml      # prod + [env.dev] configs
├── deploy.sh          # deploy production
└── deploy-dev.sh      # deploy dev
```
