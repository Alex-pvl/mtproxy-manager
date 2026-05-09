# fragment-worker

Python sidecar that fulfills Stars / Premium orders via Fragment for the staytg
backend. Wraps the unofficial [`fragment-sdk`](https://github.com/whicencer/fragment-sdk)
by whicencer. The Go backend never sees the wallet seed or Fragment session —
it only talks HTTP to this worker.

## Endpoints

| Method | Path                                              | Purpose                                  |
| ------ | ------------------------------------------------- | ---------------------------------------- |
| GET    | `/health`                                         | Liveness                                 |
| GET    | `/balance`                                        | Wallet balance (nanoTON)                 |
| GET    | `/quote?type=…&quantity=…&recipient=…`            | Current Fragment price + FX rates        |
| GET    | `/username/check?u=…&type=stars\|premium`         | Validate recipient + display name/photo  |
| POST   | `/purchase`                                       | Buy Stars/Premium for a recipient        |
| GET    | `/order/{id}`                                     | In-flight order status                   |

All endpoints (except `/health`) require `Authorization: Bearer
$FRAGMENT_WORKER_TOKEN` matching the backend.

## Required environment

| Variable                  | Description |
| ------------------------- | ----------- |
| `FRAGMENT_WORKER_TOKEN`   | Shared Bearer token (any random string — `openssl rand -hex 32`) |
| `WALLET_MNEMONIC`         | 24-word TON seed of the wallet linked to the verified Fragment account, space-separated |
| `FRAGMENT_WALLET_VERSION` | `v5r1` (default) or `v4r2` — must match the actual wallet contract |
| `FRAGMENT_COOKIES`        | `Cookie:` header value from a logged-in fragment.com session (see below) |
| `TONAPI_KEY`              | API key from https://tonapi.io (free tier is enough) — used for balance & sending TON |
| `PORT`                    | HTTP port (default 8080) |
| `USD_RATE_REFRESH_SEC`    | Cache TTL for TON→USD rate (default 300) |

### Where to get FRAGMENT_COOKIES

1. Open https://fragment.com in a desktop browser (Chrome/Firefox).
2. Connect your TON wallet (the same one whose mnemonic you put in `WALLET_MNEMONIC`).
3. Make sure your Fragment account is KYC-verified (required for buying Stars/Premium).
4. DevTools → Application (Chrome) / Storage (Firefox) → Cookies → fragment.com.
5. Copy each cookie as `name=value`, joined by `; `. Example:
   ```
   stel_ssid=abcdef0123…; stel_token=xyz…; stel_dt=…
   ```
6. Cookies expire after a few weeks — refresh when the worker starts logging
   `Failed to obtain Stars purchase link` errors.

## Top-up flow (manual)

The worker does **not** auto-replenish. The backend pre-flights every order
against `/balance`; when balance falls below `MIN_TON_BALANCE_NANO`, new orders
are rejected and the configured admin Telegram ID is alerted. Top up the wallet
through Fragment as usual.

## Local development

If `WALLET_MNEMONIC` / `FRAGMENT_COOKIES` / `TONAPI_KEY` aren't set, the worker
still starts and `/health` returns `{"creds": false}` — useful for working on
the frontend without real Fragment credentials. Other endpoints return 503.
