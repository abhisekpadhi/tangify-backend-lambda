# tangify-backend-lambda

Go AWS Lambda backend for Tangify (users/auth + billing + kitchen + plating).

## API documentation

- Main API reference: `api.md`
- Bruno collection: `apidocs-bruno/`

## Project layout

```bash
.
├── api/                         # Lambda handler and domain logic
│   ├── main.go                  # HTTP route handling
│   ├── go.mod
│   ├── users/
│   ├── billing/
│   └── menu/
├── invoice-number-cf-worker/    # Cloudflare Worker for invoice numbers (prod + dev)
├── dynamodb/
│   ├── users/                   # users table definition
│   └── billing/                 # sessions/orders/bills table definitions
├── create-dynamodb-tables.sh    # create required DynamoDB tables
├── deploy.sh                    # sam build + sam deploy
├── template.yaml                # SAM template
└── api.md                       # endpoint docs
```

## Prerequisites

- Go (matches `api/go.mod`, currently `go 1.25.0`)
- AWS CLI configured
- AWS SAM CLI
- Docker (needed by some SAM local workflows)

## Required AWS resources

The API expects these DynamoDB tables:

- `tangify_users`
- `tangify_sessions`
- `tangify_orders`
- `tangify_bills`
- `tangify_bills_with_line_items` (production bills with line items)
- `dev-tangify_bills_with_line_items` (dev bills — used when `X-Tangify-Environment: dev`)

Create them from repo root:

```bash
./create-dynamodb-tables.sh
```

For DynamoDB local:

```bash
ENDPOINT_URL=http://localhost:8000 ./create-dynamodb-tables.sh
```

## Environment variables

Set these in Lambda (or local env):

- `TANGIFY_BOOTSTRAP_SECRET` - required for `POST /api/v1/users/bootstrap`
- `TANGIFY_VENUE_ID` - default venue when request does not pass one (default fallback is `default`)
- `GOOGLE_SHEETS_API_KEY` - required for `GET /api/v1/menu`
- `GOOGLE_SHEET_ID` - required for `GET /api/v1/menu`
- `GOOGLE_SHEET_NAME` - optional for `GET /api/v1/menu`
- `LLM_API_KEY` - required for `POST /api/v1/reviews/generate` (OpenRouter API key)
- `ABLY_KEY` - optional; if set, backend publishes order events to Ably channels
- `INVOICE_NUMBER_WORKER_URL_PROD` - Cloudflare worker URL for production invoice numbers
- `INVOICE_NUMBER_WORKER_URL_DEV` - Cloudflare worker URL for dev invoice numbers (separate sequence)

**Important:** Do not put env vars in `template.yaml` — SAM deploy replaces the whole `Environment` block and wipes console-configured values. `./deploy.sh` runs `scripts/sync-lambda-env.sh` after each deploy using `.env.lambda` (gitignored).

Also ensure SSM contains:

- `tangify.jwt.secret` (used to sign/verify JWTs)

## Realtime events (Ably)

When `ABLY_KEY` is configured, the API publishes:

- `kitchen:{venue_id}`
  - `order.created` on order creation
  - `order.updated` on order updates
- `waiter:{venue_id}`
  - `order.ready` when an order status becomes `ready`

If `ABLY_KEY` is missing, API behavior is unchanged and publish calls are skipped.

## Invoice numbers (Cloudflare Worker)

Bill numbers are allocated by Cloudflare Workers with Durable Objects, not by a separate frontend `/api/bill_no` gist counter.

Flow:

1. Frontend calls Next.js `PUT /api/bills` (houseofodia-menu)
2. Next route proxies to Lambda `PUT /api/v1/billing/bills/with-line-items` with `X-Tangify-Environment`
3. Lambda calls the env-specific Cloudflare invoice worker
4. Worker returns `invoice_number`; Lambda stores bill with that as `id`

Workers live in `invoice-number-cf-worker/`. See that README for deploy and smoke-test commands.

Environment routing:

| `X-Tangify-Environment` | Invoice worker | DynamoDB table |
|-------------------------|----------------|----------------|
| `dev` | `invoice-number-cf-worker-dev` | `dev-tangify_bills_with_line_items` |
| (default / production) | `invoice-number-cf-worker` | `tangify_bills_with_line_items` |

Deploy dev worker:

```bash
cd invoice-number-cf-worker
npm run deploy:dev
```

Then add worker URLs to `.env.lambda` and run `./scripts/sync-lambda-env.sh`.

## Build and test

Build API package:

```bash
cd api
go build ./...
```

Run tests:

```bash
cd api
go test ./...
```

## Local run (SAM)

From repo root:

```bash
sam build
sam local start-api
```

## Deploy

Quick deploy script (builds, deploys, then syncs env vars from `.env.lambda`):

```bash
cp .env.lambda.example .env.lambda   # first time only — fill in values
./deploy.sh
```

Or manually:

```bash
sam build
sam deploy --no-confirm-changeset
./scripts/sync-lambda-env.sh
```
