# Tangify backend HTTP API

Base URL is your Lambda Function URL or API Gateway stage URL (examples use `https://example.lambda-url.us-east-1.on.aws`).

- **Public (no JWT):** `GET /api/v1/health`, `GET /api/v1/menu`, `POST /api/v1/reviews/generate`, `POST /api/v1/auth/login`, `POST /api/v1/users/bootstrap` (bootstrap requires `X-Bootstrap-Secret` header), `POST /api/v1/users/customer-onboard` (requires `X-CF-Secret`)  
- **Protected:** other routes require header `Authorization: Bearer <JWT>`

Successful responses return **JSON bodies directly** (no `{ "data": ... }` wrapper). Errors use  
`{ "error": "<message>" }` with appropriate HTTP status (`400`, `401`, `500`, etc.).

---

## Health

### `GET /api/v1/health`

**Response** `200`

```json
{ "status": "ok" }
```

```bash
curl -sS "https://EXAMPLE.lambda-url.on.aws/api/v1/health"
```

---

## Menu (Google Sheet)

### `GET /api/v1/menu`

Loads menu rows from Google Sheets (server-side env: `GOOGLE_SHEETS_API_KEY`, `GOOGLE_SHEET_ID`, `GOOGLE_SHEET_NAME`).

**Response** `200` — array of items:

```json
[
  {
    "status": "ON",
    "category": "Mains",
    "name": "Dal Tadka",
    "description": "",
    "is_veg": true,
    "price": "180"
  }
]
```

```bash
curl -sS "https://EXAMPLE.lambda-url.on.aws/api/v1/menu"
```

---

## Reviews (LLM-generated)

### `POST /api/v1/reviews/generate`

Generates a short, casual customer-style review for Tangify using OpenRouter (`openai/gpt-oss-120b:free`). Fetches the live menu from Google Sheets (same env as `GET /api/v1/menu`) and passes ON item names into the prompt so reviews only mention dishes Tangify actually serves. Requires server env `LLM_API_KEY` (OpenRouter API key).

**Request body:**

```json
{ "rating": 4 }
```

- `rating` — integer `1`–`5`

**Response** `200`:

```json
{
  "review": "Aaji lunch re asithili, dalma ta bhala thila. service ta jaldi hela, next time bhi try kariba."
}
```

**Errors:** `400` invalid rating, `503` missing `LLM_API_KEY`, `502` OpenRouter failure.

```bash
curl -sS -X POST -H "Content-Type: application/json" \
  -d '{"rating":5}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/reviews/generate"
```

---

## Users & auth

Accounts live in DynamoDB table `tangify_users`. Create it (with billing tables) via `./create-dynamodb-tables.sh` from the repo root, or use the JSON in `dynamodb/users/tangify_users.json`. Each user has a random `**pw_salt**`; the server bcrypt-hashes a value derived from **password + salt** (see code for the exact derivation and the bcrypt 72-byte limit). Responses never include `pw_hash` or `pw_salt`.

**Roles:** `waiter`, `kitchen`, `admin`, `customer`.

**JWT:** HS256, **24h** TTL. Custom claims: `identity` (user id), `name`, `role`; registered claims include `sub` (same as user id), `exp`, `iat`. Send on protected routes as `Authorization: Bearer <token>`.


| Method  | Path                      | Auth                                 |
| ------- | ------------------------- | ------------------------------------ |
| `POST`  | `/api/v1/auth/login`      | Public                               |
| `POST`  | `/api/v1/users/bootstrap` | Header `X-Bootstrap-Secret` (no JWT) |
| `POST`  | `/api/v1/users/customer-onboard` | Header `X-CF-Secret` (no JWT, server-to-server) |
| `GET`   | `/api/v1/users/me`        | JWT                                  |
| `POST`  | `/api/v1/users`           | JWT **admin**                        |
| `PATCH` | `/api/v1/users/password`  | JWT                                  |


Invalid JSON bodies return `**400`** with `Invalid JSON body` where applicable.

### `POST /api/v1/auth/login`

**Request body** — `LoginRequest`:

```json
{ "login": "user@example.com", "password": "secret" }
```

- If `login` contains `@`, it is treated as **email** (trimmed, lowercased for lookup).
- Otherwise it is treated as **phone**; spaces are removed (`NormalizePhone`).

**Response** `200` — `LoginResponse`:

```json
{
  "token": "<jwt>",
  "user": {
    "id": "…",
    "phone": "+9198…",
    "email": "user@example.com",
    "name": "Ravi",
    "role": "waiter"
  }
}
```

**Errors:** `401` — wrong credentials or missing `login` / `password` (`login and password required`, `invalid credentials`, etc.).

```bash
curl -sS -X POST -H "Content-Type: application/json" \
  -d '{"login":"admin@example.com","password":"your-password"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/auth/login"
```

---

### `POST /api/v1/users/customer-onboard`

Server-to-server endpoint. Validates `X-CF-Secret` against env var `CF_SECRET`, creates/gets a user with role `customer`, and sends a placeholder WhatsApp message via Gupshup.

**Request body**:

```json
{ "phone": "+919876543210", "name": "Customer Name" }
```

**Response** `200` — `UserPublic`.

**Errors:**
- `403` when `CF_SECRET` not configured
- `401` invalid/missing `X-CF-Secret`
- `400` invalid payload
- `502` Gupshup send failure

```bash
curl -sS -X POST -H "Content-Type: application/json" \
  -H "X-CF-Secret: $CF_SECRET" \
  -d '{"phone":"+919876543210","name":"Customer Name"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/users/customer-onboard"
```

---

### `POST /api/v1/users/bootstrap`

Provisions a user when `**TANGIFY_BOOTSTRAP_SECRET**` is set in the Lambda environment. **No JWT.** Header `**X-Bootstrap-Secret`** must match the env value exactly.

If `role` is omitted, it defaults to `**admin**`. Same validation as create user: **either phone or email is required** (you can send both), along with **name** and **password**; **password** at least **8** characters; **role** must be one of `waiter`, `kitchen`, `admin`; provided email/phone values must be unique.

**Request body** — `BootstrapUserRequest`:

```json
{
  "phone": "+919876543210",
  "email": "admin@example.com",
  "name": "Admin",
  "role": "admin",
  "password": "long-secure-password"
}
```

**Response** `200` — `UserPublic` (same shape as `user` in login).

**Errors:** `403` — `Bootstrap is not configured` (env secret empty). `401` — wrong or missing `X-Bootstrap-Secret`. `400` — validation (duplicate email/phone, invalid role, short password, missing fields, etc.).

```bash
curl -sS -X POST -H "Content-Type: application/json" \
  -H "X-Bootstrap-Secret: $TANGIFY_BOOTSTRAP_SECRET" \
  -d '{"phone":"+919876543210","email":"admin@example.com","name":"Admin","role":"admin","password":"securepass123"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/users/bootstrap"
```

---

### `POST /api/v1/users`

**Admin only** (`role` in JWT must be `admin`). Creates a user with the given password.

**Request body** — `CreateUserRequest` (same fields as bootstrap; `**role`** required here — no default):

```json
{
  "phone": "+919876543210",
  "email": "waiter1@example.com",
  "name": "Waiter One",
  "role": "waiter",
  "password": "long-secure-password"
}
```

**Password** minimum length **8**. At least one of `phone` or `email` is required (both are allowed). Any provided email/phone must be unique.

**Response** `200` — `UserPublic`.

**Errors:** `403` — `admin only`. `400` — validation (same messages as bootstrap/create path).

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"phone":"+919876543210","email":"waiter1@example.com","name":"Waiter One","role":"waiter","password":"securepass123"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/users"
```

---

### `GET /api/v1/users/me`

Returns the user for JWT claim `**identity**`.

**Response** `200` — `UserPublic` (same as login `user`).

**Errors:** `404` — `user not found` (id in token missing from DB). `500` — DynamoDB / server error.

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/users/me"
```

---

### `PATCH /api/v1/users/password`

**Who may change whom**

- `**admin`:** may set a new password for **any** user. Send `user_id` and `new_password` only; `**current_password` is not used**.
- **Non-admin:** may change **only their own** password. Send `user_id` equal to your id, plus `**current_password`** and `**new_password**`.

**Request body** — `ChangePasswordRequest`:

```json
{
  "user_id": "<uuid>",
  "current_password": "old",
  "new_password": "new-long-password"
}
```

`new_password` must be at least **8** characters. `user_id` is always required.

**Response** `200` — `{ "status": "ok" }`.

**Errors:** `403` — non-admin trying to change someone else’s password (`forbidden`). `400` — missing `user_id` / `new_password`, user not found, short password, or wrong `current_password` when required (`current password required or invalid`).

```bash
curl -sS -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"user_id":"<uuid>","current_password":"old","new_password":"newsecurepass123"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/users/password"
```

---

## Billing — waiter

Default venue for reads/writes is `TANGIFY_VENUE_ID` env or `"default"`. Sessions and orders store `venue_id` for DynamoDB GSIs.

### Order channels (`channel`)


| Value                     | Description              |
| ------------------------- | ------------------------ |
| `dining_table`            | In-restaurant table      |
| `takeaway`                | Takeaway                 |
| `whatsapp_quickdelivery`  | WhatsApp quick delivery  |
| `whatsapp_normaldelivery` | WhatsApp normal delivery |
| `neighbour_delivery`      | Neighbour delivery       |


### `GET /api/v1/billing/live-orders`

Live or billing sessions with their orders (waiter board).

**Query**


| Param      | Required | Description                      |
| ---------- | -------- | -------------------------------- |
| `venue_id` | No       | Defaults to server default venue |


**Response** `200` — `LiveOrdersGroupedResponse`:

```json
{
  "sessions": [
    {
      "session": {
        "id": "sess_…",
        "table_ids": ["T1", "T2"],
        "status": "live",
        "opened_at": 1710000000000,
        "venue_id": "default"
      },
      "orders": []
    }
  ]
}
```

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/live-orders?venue_id=default"
```

---

### `POST /api/v1/billing/sessions`

Open a table session and place the **first** order (tables become live).

**Request body** — `CreateSessionAndFirstOrderRequest`:

```json
{
  "table_ids": ["T5"],
  "items": [
    { "id": "", "name": "Dal", "quantity": 2, "price": 18000, "status": "" }
  ],
  "channel": "dining_table",
  "customer_id": null,
  "staff_id": null,
  "ordered_at": null
}
```

- `table_ids`: one table or multiple for **joined** tables.  
- Line `id` / `status` may be omitted; server fills defaults (`line_`* ids, `pending`).  
- `price` is in **paise** (integer). Totals use `sum(price * quantity)` per order.
- If any requested `table_id` already belongs to a `live`/`billing` session, API returns `409` and asks you to add orders to the existing session.

**Response** `200` — `SessionWithOrders`:

```json
{
  "session": {
    "id": "sess_…",
    "table_ids": ["T5"],
    "status": "live",
    "opened_at": 1710000000000,
    "updated_at": 1710000000000,
    "venue_id": "default"
  },
  "orders": [
    {
      "id": "ord_…",
      "session_id": "sess_…",
      "venue_id": "default",
      "channel": "dining_table",
      "items": [{ "id": "line_…", "name": "Dal", "quantity": 2, "price": 18000, "status": "pending" }],
      "total_price": 36000,
      "kitchen_status": "pending",
      "ordered_at": 1710000000000,
      "updated_at": 1710000000000
    }
  ]
}
```

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"table_ids":["T5"],"items":[{"name":"Dal","quantity":2,"price":18000}],"channel":"dining_table"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/sessions"
```

---

### `POST /api/v1/billing/orders`

Add another order to an existing session.

**Request body** — `AddOrderToSessionRequest`:

```json
{
  "session_id": "sess_…",
  "items": [{ "name": "Rice", "quantity": 1, "price": 8000 }],
  "channel": "dining_table",
  "source_table_id": null,
  "staff_id": null,
  "ordered_at": null
}
```

**Response** `200` — `Order` (single object).

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"session_id":"sess_xxx","items":[{"name":"Rice","quantity":1,"price":8000}],"channel":"dining_table"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/orders"
```

---

### `PATCH /api/v1/billing/orders`

Update line items and/or kitchen status on an order. Supports soft-removing line items from billing/order totals.

**Request body** — `UpdateOrderRequestV2`:

```json
{
  "order_id": "ord_…",
  "items": [
    {
      "id": "line_…",
      "name": "Dal",
      "quantity": 2,
      "price": 18000,
      "status": "pending"
    }
  ],
  "remove_line_item_ids": ["line_…"],
  "total_price": null,
  "kitchen_status": null
}
```

Omit `items` to leave lines unchanged; set `kitchen_status` to a **KitchenStatus** value if needed (`pending`, `preparing`, `ready`, `served`, `cancelled`).
Use `remove_line_item_ids` to mark specific line items as `removed=true` and `status=cancelled` in that order.

**Response** `200` — `Order`.

```bash
curl -sS -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"order_id":"ord_xxx","kitchen_status":"preparing"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/orders"
```

---

### `GET /api/v1/billing/orders`

List orders either by session or by physical table.

**Query (one required)**


| Param        | Description                                                      |
| ------------ | ---------------------------------------------------------------- |
| `session_id` | All orders for this session                                      |
| `table_id`   | Orders for the **live/billing** session that contains this table |
| `venue_id`   | With `table_id`, optional venue (defaults server-side)           |


**Response** `200` — `Order[]`.

```bash
# By session
curl -sS -H "Authorization: Bearer $TOKEN" \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/orders?session_id=sess_xxx"

# By table
curl -sS -H "Authorization: Bearer $TOKEN" \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/orders?table_id=T5&venue_id=default"
```

---

### `POST /api/v1/billing/bills/start`

Create the bill and move session to **billing**; links orders to the bill and rolls up totals.
This endpoint is idempotent for a session: if a bill already exists (including non-`live` session states), server returns the existing bill instead of creating a duplicate.

**Request body** — `StartBillForSessionRequest`:

```json
{ "session_id": "sess_…", "staff_id": null }
```

**Response** `200` — `Bill`:

```json
{
  "id": "bill_…",
  "session_id": "sess_…",
  "table_ids": ["T5"],
  "payment_method": "cash",
  "payment_status": "pending",
  "created_at": 1710000000000,
  "updated_at": 1710000000000,
  "total_tax_in_paise": 0,
  "total_discount_in_paise": 0,
  "total_amount_in_paise": 44000
}
```

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"session_id":"sess_xxx"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/bills/start"
```

---

### `PATCH /api/v1/billing/bills`

Update payment fields and/or apply billing-time line-item edits across orders in the bill's session.

**Request body** — `UpdateBillRequestV2`:

```json
{
  "bill_id": "bill_…",
  "payment_method": "card",
  "payment_status": "pending",
  "line_item_updates": [
    {
      "order_id": "ord_…",
      "line_item_id": "line_…",
      "user_override": {
        "quantity": 3,
        "price": 17000
      },
      "removed": false
    }
  ]
}
```

`line_item_updates` behavior:

- `user_override.quantity`: optional per-line quantity override (`> 0`)
- `user_override.price`: optional per-line price override (paise)
- `removed`: soft remove from billing totals (`true` also sets line status to `cancelled`)

`total_amount_in_paise` is server-controlled and computed from current non-removed line items (applying any `user_override` values). Client cannot override it.

**Response** `200` — `Bill`.

```bash
curl -sS -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"bill_id":"bill_xxx","payment_method":"card"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/bills"
```

---

### `POST /api/v1/billing/invoice-number`

Generate/fetch invoice number for a bill by calling the invoice worker and persist `invoice_number` on the bill row.

**Request body** — `GenerateInvoiceNumberRequest`:

```json
{ "bill_id": "bill_6b11733c-9f51-4c7d-8e61-012940141d68" }
```

**Response** `200`:

```json
{
  "invoice_number": "2026-000001",
  "bill_id": "bill_6b11733c-9f51-4c7d-8e61-012940141d68",
  "year": 2026,
  "sequence": 1
}
```

**Errors**:

- `400` if `bill_id` missing
- `404` if bill not found
- `502` if invoice worker call fails

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"bill_id":"bill_6b11733c-9f51-4c7d-8e61-012940141d68"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/invoice-number"
```

---

### `POST /api/v1/billing/sessions/close`

Finalize checkout: mark bill paid and session **closed**.

**Request body** — `CloseTableRequest`:

```json
{ "session_id": "sess_…", "bill_id": "bill_…" }
```

**Response** `200`:

```json
{ "status": "closed" }
```

```bash
curl -sS -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"session_id":"sess_xxx","bill_id":"bill_xxx"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/billing/sessions/close"
```

---

### `PUT /api/v1/billing/bills/with-line-items`

Create or update a **bill snapshot** with embedded line items. JWT required.

`X-Tangify-Environment: dev` writes `dev-tangify_bills_with_line_items` + `dev-tangify_points_wallet`; otherwise `tangify_bills_with_line_items` + `tangify_points_wallet`.

**Create** — send `state_key` (no `id`). Idempotent: retries with the same `state_key` return the existing bill without re-deducting points or calling the invoice worker again.

**Update** — send `id` (invoice number from create). Points discount is frozen from create; wallet is not debited again.

**Settle / earn** — send `settled: true` (create or update). Awards `floor((subtotal − discounts) / 50)` points once (`loyalty_points_processed`). Same Dynamo transaction as the bill write.

**Request body** — `UpsertBillWithLineItemsRequest`:

```json
{
  "state_key": "sess_abc::checkout",
  "session_id": "sess_abc",
  "table_ids": ["T5"],
  "customer_id": "919876543210",
  "settled": false,
  "line_items": [
    { "name": "Dal", "quantity": 2, "price": 15000 }
  ],
  "discounts": [
    { "id": "points", "type": "points", "amount": 900 }
  ],
  "taxes": [
    { "id": "gst", "name": "GST", "rate_in_bps": 500, "amount_in_paise": 1500 }
  ]
}
```

Points rules (this API only):
- `customer_id` is `91` + 10-digit phone (legacy 10-digit / `+91` still match).
- `type: "points"` — `amount` is paise to redeem (honored, capped by wallet and remaining subtotal). 1 point = Rs 3 (`300` paise).
- Points cannot stack with other discount types.
- Wallet debit on **create only**. Earn on **`settled: true`** if not already processed.

**Response** `200` — full `BillWithLineItems` document; `id` is the invoice number.

### `GET /api/v1/billing/bills/with-line-items?bill_id=<invoice_number>`

Fetch a bill snapshot by invoice number / `id`. JWT required.

---

## Loyalty

### OTP login (public)

Customer lookup before billing. **No JWT.** User and wallet are created **only after** OTP verify.

#### `POST /api/v1/loyalty/otp/send`

```json
{ "phone": "+919876543210", "name": "optional" }
```

Response:

```json
{ "sent": true }
```

- 4-digit OTP via WhatsApp template `points_at_counter` (`{{1}}` is the code).
- Valid 5 minutes; max 1 send per phone per 60 seconds.

#### `POST /api/v1/loyalty/otp/verify`

```json
{ "phone": "+919876543210", "otp": "1234", "name": "optional" }
```

Response:

```json
{
  "user_id": "…",
  "points_balance": 0,
  "phone": "+919876543210"
}
```

Use `user_id` as `customer_id` on bill checkout. Max 5 failed verify attempts per challenge.

### `GET /api/v1/loyalty/wallet?phone=<91XXXXXXXXXX>`

Staff wallet lookup. **JWT required** (same as bills). Not on the public allowlist.

Creates the customer + empty wallet if missing. Phone is stored as `91` + 10 digits; lookup also matches 10-digit / `+91` / leading `0`.

```json
{
  "phone": "919876543210",
  "user_id": "…",
  "points_balance": 12
}
```

POS should call this via the Next.js `/api/loyalty/wallet` proxy (Clerk + `TANGIFY_BILLING_TOKEN`), not the Function URL.

`X-Tangify-Environment: dev` reads/writes `dev-tangify_points_wallet`; otherwise `tangify_points_wallet`. Same header as bills.

---

### Legacy loyalty (JWT)

Policy (aligned with snapshot bills):
- Earn `1` point per `Rs 50` discounted subtotal (`5000` paise).
- Redeem any whole points. 1 point = Rs 3 (`300` paise).
- Env `LOYALTY_DISCOUNT_PER_100_POINTS_PAISE` still overrides paise **per point** if set.

### `POST /api/v1/loyalty/points/add`

Award points for a bill to a user's wallet.

```json
{ "user_id": "user_xxx", "bill_id": "bill_xxx" }
```

Response:

```json
{
  "user_id": "user_xxx",
  "bill_id": "bill_xxx",
  "points_earned": 20,
  "current_balance": 120
}
```

### `GET /api/v1/loyalty/discount?user_id=<id>`

Get current points and redeemable discount.

```json
{
  "user_id": "user_xxx",
  "current_points": 230,
  "redeemable_points": 200,
  "discount_per_100_points": 25000,
  "redeemable_discount": 50000
}
```

### `POST /api/v1/loyalty/discount/apply`

Apply loyalty discount to a bill (uses multiples of 100 points only).

```json
{ "user_id": "user_xxx", "bill_id": "bill_xxx" }
```

Response:

```json
{
  "user_id": "user_xxx",
  "bill_id": "bill_xxx",
  "points_redeemed": 100,
  "discount_applied": 25000,
  "remaining_points": 130,
  "updated_bill_total": 175000
}
```

---

## Kitchen

### `GET /api/v1/kitchen/item-board`

Expand all venue orders into per-line rows (excludes lines already `served` or `cancelled`).

**Query**


| Param      | Required            |
| ---------- | ------------------- |
| `venue_id` | No (server default) |


**Response** `200` — `KitchenDishCount[]`:

```json
[
  {
    "order_id": "ord_…",
    "line_item_id": "line_…",
    "name": "Dal",
    "quantity": 2,
    "status": "pending"
  }
]
```

```bash
curl -sS -H "Authorization: Bearer $TOKEN" \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/kitchen/item-board?venue_id=default"
```

---

### `PATCH /api/v1/kitchen/line-items/status`

Update **one line item**’s kitchen status.

**Request body** — `PatchLineItemStatusRequest`:

```json
{
  "order_id": "ord_…",
  "line_item_id": "line_…",
  "status": "preparing"
}
```

Line item statuses: `pending`, `preparing`, `ready`, `served`, `cancelled`.

**Response** `200` — full `Order` after update.

```bash
curl -sS -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"order_id":"ord_xxx","line_item_id":"line_xxx","status":"ready"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/kitchen/line-items/status"
```

---

## Plating

### `GET /api/v1/plating/orders`

FIFO-style queue for plating: orders sorted by `ordered_at`, excluding orders whose **order-level** `kitchen_status` is `served`.
If `session_id` and `table_id` are omitted, returns all non-served orders for the given/default `venue_id`.

**Query**


| Param        | Required | Description                                                             |
| ------------ | -------- | ----------------------------------------------------------------------- |
| `session_id` | No       | FIFO for this session (highest priority)                                |
| `table_id`   | No       | Resolve live session for table, then FIFO                               |
| `venue_id`   | No       | Venue scope (default venue); also used for "all non-served orders" mode |
| `limit`      | No       | Max orders (default `100`)                                              |


**Response** `200` — `PlatingQueueOrder[]`:

```json
[
  {
    "order_id": "ord_…",
    "session_id": "sess_…",
    "table_ids": ["T5"],
    "items": [
      {
        "id": "line_…",
        "name": "Dal",
        "quantity": 2,
        "price": 18000,
        "status": "pending"
      }
    ],
    "kitchen_status": "pending",
    "ordered_at": 1710000000000
  }
]
```

```bash
# All non-served orders for venue
curl -sS -H "Authorization: Bearer $TOKEN" \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/plating/orders?venue_id=default&limit=50"

# By table
curl -sS -H "Authorization: Bearer $TOKEN" \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/plating/orders?table_id=T5&venue_id=default&limit=50"
```

---

### `PATCH /api/v1/plating/orders/status`

Update **order-level** kitchen status (plating / expediter).

**Request body** — `PatchOrderKitchenStatusRequestV2`:

```json
{ "order_id": "ord_…", "kitchen_status": "ready" }
```

**Response** `200` — `Order`.

```bash
curl -sS -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"order_id":"ord_xxx","kitchen_status":"served"}' \
  "https://EXAMPLE.lambda-url.on.aws/api/v1/plating/orders/status"
```

---

## JWT for billing, kitchen, and plating

Call `**POST /api/v1/auth/login**` (or use a token from an admin-created user), then pass `**Authorization: Bearer <jwt>**` on routes that are not public. See **Users & auth** above.

---

## Default route

Any unmatched path returns `200` with:

```json
{ "message": "Hello, World!" }
```

(Useful only for smoke tests; prefer explicit routes above.)