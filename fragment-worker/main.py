"""Fragment worker — HTTP sidecar that buys Telegram Stars / Premium via Fragment.

Endpoints:
  GET  /health
  GET  /balance                       → {ton_balance_nano}
  GET  /quote?type=stars&quantity=100 → {ton_cost_nano, ton_rub_rate, ton_usd_rate}
  GET  /username/check?u=foo          → {ok, username, reason?}
  POST /purchase                      → {status, tx_hash?, error?}
  GET  /order/<id>                    → {status, tx_hash?, error?}

Auth: optional Bearer token in `Authorization` header (FRAGMENT_WORKER_TOKEN).

The worker holds the wallet seed and Fragment session. It is intentionally a
narrow API surface so the Go backend never sees credentials.

Notes
─────
This file targets the unofficial `fragment-api` library (PyPI: fragment-api)
which authenticates against fragment.com via a TON Connect-signed cookie and
sends purchases through the user's TON wallet.

If `fragment-api` is not installed the worker still starts and returns 503 from
the purchase endpoints — useful for local development of the frontend.
"""

from __future__ import annotations

import asyncio
import logging
import os
import re
import time
from dataclasses import dataclass, field
from typing import Any

from aiohttp import web

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("fragment-worker")

# ─── Config ──────────────────────────────────────────────────────────────────

TOKEN = os.environ.get("FRAGMENT_WORKER_TOKEN", "")
WALLET_MNEMONIC = os.environ.get("WALLET_MNEMONIC", "")
FRAGMENT_COOKIES = os.environ.get("FRAGMENT_COOKIES", "")
TG_API_ID = int(os.environ.get("TG_API_ID", "0") or "0")
TG_API_HASH = os.environ.get("TG_API_HASH", "")
TON_RATE_REFRESH_SEC = int(os.environ.get("TON_RATE_REFRESH_SEC", "300"))


def _have_creds() -> bool:
    return bool(WALLET_MNEMONIC and FRAGMENT_COOKIES)


# ─── Fragment client (lazy import, may be missing in dev) ────────────────────

_client = None
_client_lock = asyncio.Lock()


async def _get_client():
    """Lazily build and cache a Fragment client."""
    global _client
    async with _client_lock:
        if _client is not None:
            return _client
        if not _have_creds():
            return None
        try:
            # The third-party `fragment-api` lib exposes an async FragmentAPIClient.
            # If it's missing we report unavailable; integration is optional in dev.
            from fragment_api import FragmentAPIClient  # type: ignore
        except Exception as e:  # noqa: BLE001
            log.warning("fragment-api lib not available: %s", e)
            return None

        client = FragmentAPIClient(
            mnemonics=WALLET_MNEMONIC.split(),
            cookies=FRAGMENT_COOKIES,
        )
        await client.start()
        _client = client
        log.info("Fragment client started")
        return _client


# ─── Rate cache ──────────────────────────────────────────────────────────────


@dataclass
class RateCache:
    ton_rub: float = 0.0
    ton_usd: float = 0.0
    fetched_at: float = 0.0


_rate_cache = RateCache()


async def _refresh_rates():
    """Pull TON→USD and USD→RUB from CoinGecko."""
    global _rate_cache
    import aiohttp

    if time.time() - _rate_cache.fetched_at < TON_RATE_REFRESH_SEC and _rate_cache.ton_rub > 0:
        return _rate_cache
    url = "https://api.coingecko.com/api/v3/simple/price?ids=the-open-network&vs_currencies=usd,rub"
    try:
        async with aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=10)) as sess:
            async with sess.get(url) as resp:
                data = await resp.json()
        ton = data.get("the-open-network", {})
        ton_rub = float(ton.get("rub") or 0)
        ton_usd = float(ton.get("usd") or 0)
        if ton_rub > 0 and ton_usd > 0:
            _rate_cache = RateCache(ton_rub=ton_rub, ton_usd=ton_usd, fetched_at=time.time())
    except Exception as e:  # noqa: BLE001
        log.warning("rate refresh failed: %s", e)
    return _rate_cache


# ─── Order tracking (in-memory; recovery loop on Go side polls /order/{id}) ──


@dataclass
class OrderState:
    status: str = "pending"  # pending | processing | delivered | failed
    tx_hash: str = ""
    error: str = ""
    created_at: float = field(default_factory=time.time)


_orders: dict[int, OrderState] = {}


# ─── Auth middleware ─────────────────────────────────────────────────────────


@web.middleware
async def auth_mw(request: web.Request, handler):
    if request.path == "/health":
        return await handler(request)
    if TOKEN:
        header = request.headers.get("Authorization", "")
        if not header.startswith("Bearer ") or header[7:] != TOKEN:
            return web.json_response({"error": "unauthorized"}, status=401)
    return await handler(request)


# ─── Handlers ────────────────────────────────────────────────────────────────


async def health(_: web.Request):
    return web.json_response(
        {
            "ok": True,
            "creds": _have_creds(),
            "tg_api_configured": bool(TG_API_ID and TG_API_HASH),
        }
    )


async def balance(_: web.Request):
    client = await _get_client()
    if client is None:
        return web.json_response({"error": "fragment client unavailable"}, status=503)
    try:
        # Library exposes wallet balance in nanoTON.
        nano = await client.get_wallet_balance()
        return web.json_response({"ton_balance_nano": int(nano)})
    except Exception as e:  # noqa: BLE001
        log.exception("balance error")
        return web.json_response({"error": str(e)}, status=502)


async def quote(request: web.Request):
    qtype = request.query.get("type", "")
    try:
        qty = int(request.query.get("quantity", "0"))
    except ValueError:
        return web.json_response({"error": "invalid quantity"}, status=400)
    if qtype not in ("stars", "premium") or qty <= 0:
        return web.json_response({"error": "invalid params"}, status=400)

    rates = await _refresh_rates()
    client = await _get_client()
    if client is None:
        return web.json_response({"error": "fragment client unavailable"}, status=503)
    try:
        if qtype == "stars":
            nano = await client.get_stars_price_nano(qty)
        else:
            nano = await client.get_premium_price_nano(months=qty)
    except Exception as e:  # noqa: BLE001
        log.exception("quote error")
        return web.json_response({"error": str(e)}, status=502)

    return web.json_response(
        {
            "ton_cost_nano": int(nano),
            "ton_rub_rate": rates.ton_rub,
            "ton_usd_rate": rates.ton_usd,
        }
    )


_USERNAME_RE = re.compile(r"^[a-zA-Z][a-zA-Z0-9_]{4,31}$")


async def username_check(request: web.Request):
    u = (request.query.get("u") or "").lstrip("@").strip()
    if not _USERNAME_RE.match(u):
        return web.json_response({"ok": False, "username": u, "reason": "invalid username format"})
    client = await _get_client()
    if client is None:
        # Best-effort fallback when library unavailable: accept format-valid usernames.
        return web.json_response({"ok": True, "username": u})
    try:
        ok = await client.check_username(u)
        return web.json_response({"ok": bool(ok), "username": u, "reason": "" if ok else "user not found or cannot receive"})
    except Exception as e:  # noqa: BLE001
        return web.json_response({"ok": False, "username": u, "reason": str(e)})


async def purchase(request: web.Request):
    body = await request.json()
    order_id = int(body.get("order_id") or 0)
    qtype = body.get("type") or ""
    recipient = (body.get("recipient") or "").lstrip("@")
    qty = int(body.get("quantity") or 0)

    if order_id <= 0 or qtype not in ("stars", "premium") or not recipient or qty <= 0:
        return web.json_response({"error": "invalid params"}, status=400)

    client = await _get_client()
    if client is None:
        return web.json_response({"error": "fragment client unavailable"}, status=503)

    state = _orders.setdefault(order_id, OrderState())
    if state.status == "delivered":
        return web.json_response({"status": "delivered", "tx_hash": state.tx_hash})
    state.status = "processing"

    try:
        if qtype == "stars":
            tx_hash = await client.buy_stars(recipient=recipient, quantity=qty)
        else:
            tx_hash = await client.buy_premium(recipient=recipient, months=qty)
    except Exception as e:  # noqa: BLE001
        log.exception("purchase failed order=%s", order_id)
        state.status = "failed"
        state.error = str(e)
        return web.json_response({"status": "failed", "error": str(e)}, status=502)

    state.status = "delivered"
    state.tx_hash = str(tx_hash or "")
    return web.json_response({"status": "delivered", "tx_hash": state.tx_hash})


async def order_status(request: web.Request):
    order_id = int(request.match_info["id"])
    state = _orders.get(order_id)
    if state is None:
        return web.json_response({"status": "unknown"}, status=404)
    return web.json_response({"status": state.status, "tx_hash": state.tx_hash, "error": state.error})


# ─── App factory ─────────────────────────────────────────────────────────────


def make_app() -> web.Application:
    app = web.Application(middlewares=[auth_mw])
    app.router.add_get("/health", health)
    app.router.add_get("/balance", balance)
    app.router.add_get("/quote", quote)
    app.router.add_get("/username/check", username_check)
    app.router.add_post("/purchase", purchase)
    app.router.add_get("/order/{id:\\d+}", order_status)
    return app


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "8080"))
    log.info("Starting fragment-worker on :%s (creds=%s)", port, _have_creds())
    web.run_app(make_app(), port=port)
