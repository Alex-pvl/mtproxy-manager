# fragment-worker

Python sidecar that fulfills Stars / Premium orders via Fragment for the staytg
backend. The Go backend never sees the wallet seed or Fragment session — it
only talks HTTP to this worker.

## Endpoints

| Method | Path                        | Purpose                                     |
| ------ | --------------------------- | ------------------------------------------- |
| GET    | `/health`                   | Liveness                                    |
| GET    | `/balance`                  | TON wallet balance (nanoTON)                |
| GET    | `/quote?type=…&quantity=…`  | Current Fragment price (nanoTON) + FX rates |
| GET    | `/username/check?u=…`       | Validate Telegram username                  |
| POST   | `/purchase`                 | Buy Stars/Premium for a recipient           |
| GET    | `/order/{id}`               | Status of an in-flight order                |

All endpoints (except `/health`) require `Authorization: Bearer
$FRAGMENT_WORKER_TOKEN` matching the backend.

## Required environment

| Variable               | Description                                                       |
| ---------------------- | ----------------------------------------------------------------- |
| `FRAGMENT_WORKER_TOKEN`| Shared Bearer token (any random string)                           |
| `WALLET_MNEMONIC`      | 24-word TON seed of the verified Fragment account, space-separated|
| `FRAGMENT_COOKIES`     | `Cookie:` header value from a logged-in fragment.com session      |
| `TG_API_ID`            | Telegram API ID from https://my.telegram.org/apps                 |
| `TG_API_HASH`          | Telegram API hash                                                 |
| `PORT`                 | HTTP port (default 8080)                                          |
| `TON_RATE_REFRESH_SEC` | How often to refresh TON↔USD/RUB rates (default 300)              |

## Top-up flow (manual)

The worker does **not** auto-replenish the wallet. The backend pre-flights every
order against `/balance`; when balance falls below `MIN_TON_BALANCE_NANO`, new
orders are rejected and the configured admin Telegram ID is alerted. Top up the
wallet via Fragment as usual.

## Local development

If `fragment-api` is not installed (or credentials are absent) the worker still
starts and returns 503 from `/balance`, `/quote`, and `/purchase` — useful for
working on the frontend without the real Fragment integration.
