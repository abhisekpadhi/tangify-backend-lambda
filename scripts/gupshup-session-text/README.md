# Test a WhatsApp **session** (free-form) message

Hits Gupshup `POST /wa/api/v1/msg` with a text body. This is **not** a template.

Meta only delivers it if the customer has messaged this business number in the **last 24 hours**. Use this to confirm a QR / click-to-chat inbound opened the window.

## Setup

```bash
cd scripts/gupshup-session-text
cp ../gupshup-reward-point/.env .env
# or: cp .env.example .env  and fill keys
chmod +x send.sh
```

Same env as the template script: `GUPSHUP_API_KEY`, `GUPSHUP_SOURCE`, `GUPSHUP_APP_NAME`. If `.env` is missing here, `send.sh` falls back to `../gupshup-reward-point/.env`.

## Test

1. From the destination phone, WhatsApp `GUPSHUP_SOURCE` (e.g. `https://wa.me/917855074030?text=hi`) and **send**.
2. Then:

```bash
./send.sh 9439831236
./send.sh 919439831236 "You earned 12 points. Balance 40."
```

HTTP 202 with a `messageId` means Gupshup accepted it. Delivery still depends on Meta.

If there is no open session, Gupshup/Meta rejects the send (session/window error). That is the control case: run this **before** the user messages you, then again **after**.
