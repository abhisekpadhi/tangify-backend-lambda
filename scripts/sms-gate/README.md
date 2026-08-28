# Test SMS via SMS Gate cloud

Sends a plain SMS through [SMS Gate](https://docs.sms-gate.app) public cloud (`api.sms-gate.app`). The Android app must be online with Cloud Server enabled.

```bash
cd scripts/sms-gate
cp .env.example .env
chmod +x send.sh status.sh
./send.sh 919439831236
./status.sh
./status.sh 30
```

`./status.sh` calls `GET /3rdparty/v1/devices` and prints `active` / `stale` from `lastSeen`. Exit `0` if any device is within the window, `2` if all are stale. Cloud `lastSeen` can lag about 15 minutes.

`.env` vars: `SMS_GATE_SERVER`, `SMS_GATE_USERNAME`, `SMS_GATE_PASSWORD`, optional `SMS_GATE_ACTIVE_WITHIN_MINUTES`.
