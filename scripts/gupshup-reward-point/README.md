# Test `reward_point` WhatsApp template

Sends the Gupshup **Utility** template:

```
Hello from Tangify.

You earned *{{1}} points* on your recent visit. Your balance is now *{{2}} points*.

Thank you.
```

Template must be **Approved** (not Pending). Pending sends fail.

## Setup

```bash
cd scripts/gupshup-reward-point
cp .env.example .env
```

Fill `.env`:

| Var | What |
| --- | --- |
| `GUPSHUP_API_KEY` | Gupshup apikey header |
| `GUPSHUP_SOURCE` | WhatsApp number, `91XXXXXXXXXX` |
| `GUPSHUP_APP_NAME` | App name in Gupshup (`src.name`) |
| `GUPSHUP_TEMPLATE_ID` | Defaults to `a8085178-7d66-4223-826d-25d89aa315d0` |

## Send

```bash
chmod +x send.sh
./send.sh 9876543210
./send.sh 919876543210 8 40
```

Args: destination phone, optional earned (`{{1}}`), optional balance (`{{2}}`). Defaults 12 and 112.

The destination phone must be able to receive from this WABA. Approved utility templates do not need an open 24h chat session.
