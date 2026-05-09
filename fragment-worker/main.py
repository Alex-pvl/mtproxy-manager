"""Fragment worker — HTTP sidecar that buys Telegram Stars / Premium via Fragment.

Wraps `fragment-sdk` (whicencer/fragment-sdk) so the Go backend never sees the
wallet seed or Fragment session.

Endpoints:
  GET  /health
  GET  /balance                                       → {ton_balance_nano}
  GET  /quote?type=…&quantity=…&recipient=…           → {ton_cost_nano, ton_rub_rate, ton_usd_rate}
  GET  /username/check?u=…&type=stars|premium         → {ok, username, display_name, photo_url, reason?}
  POST /purchase  {order_id, type, recipient, quantity} → {status, tx_hash?, error?}
  GET  /order/{id}                                    → {status, tx_hash?, error?}

Auth: optional Bearer token in `Authorization`.
"""

from __future__ import annotations

import asyncio
import logging
import os
import re
import time
from dataclasses import dataclass, field
from typing import Optional

from aiohttp import web

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
log = logging.getLogger("fragment-worker")

NANO = 1_000_000_000

# ─── Config ──────────────────────────────────────────────────────────────────

TOKEN = os.environ.get("FRAGMENT_WORKER_TOKEN", "")
WALLET_MNEMONIC = os.environ.get("WALLET_MNEMONIC", "")
WALLET_VERSION = os.environ.get("FRAGMENT_WALLET_VERSION", "v5r1")  # v5r1 or v4r2
FRAGMENT_COOKIES = os.environ.get("FRAGMENT_COOKIES", "")
TONAPI_KEY = os.environ.get("TONAPI_KEY", "")
USD_RATE_REFRESH_SEC = int(os.environ.get("USD_RATE_REFRESH_SEC", "300"))


def _have_creds() -> bool:
    return bool(WALLET_MNEMONIC and FRAGMENT_COOKIES and TONAPI_KEY)


# ─── Fragment client (lazy init) ─────────────────────────────────────────────

_fragment = None
_wallet = None
_init_lock = asyncio.Lock()


async def _get_clients():
    """Build wallet + fragment client (cached)."""
    global _fragment, _wallet
    async with _init_lock:
        if _fragment is not None:
            return _fragment, _wallet
        if not _have_creds():
            return None, None
        try:
            from fragment_sdk import FragmentClient
            from fragment_sdk.wallet import WalletClient
        except Exception as e:  # noqa: BLE001
            log.warning("fragment-sdk not available: %s", e)
            return None, None

        try:
            _wallet = WalletClient(
                tonapi_key=TONAPI_KEY,
                mnemonic=WALLET_MNEMONIC,
                version=WALLET_VERSION,
            )
            _fragment = FragmentClient(cookies=FRAGMENT_COOKIES, wallet=_wallet)
            log.info("Fragment client started; wallet=%s", _wallet.address)
        except Exception as e:  # noqa: BLE001
            log.exception("fragment client init failed: %s", e)
            _fragment, _wallet = None, None
            return None, None
        return _fragment, _wallet


# ─── USD rate cache (TON↔RUB comes from Fragment itself) ─────────────────────


@dataclass
class _UsdCache:
    ton_usd: float = 0.0
    fetched_at: float = 0.0


_usd_cache = _UsdCache()


async def _ton_usd_rate() -> float:
    global _usd_cache
    if time.time() - _usd_cache.fetched_at < USD_RATE_REFRESH_SEC and _usd_cache.ton_usd > 0:
        return _usd_cache.ton_usd
    import aiohttp
    url = "https://api.coingecko.com/api/v3/simple/price?ids=the-open-network&vs_currencies=usd"
    try:
        async with aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=10)) as sess:
            async with sess.get(url) as resp:
                data = await resp.json()
        rate = float(data.get("the-open-network", {}).get("usd") or 0)
        if rate > 0:
            _usd_cache = _UsdCache(ton_usd=rate, fetched_at=time.time())
    except Exception as e:  # noqa: BLE001
        log.warning("usd rate fetch failed: %s", e)
    return _usd_cache.ton_usd


# ─── Order state (in-memory) ─────────────────────────────────────────────────


@dataclass
class OrderState:
    status: str = "pending"  # pending | processing | delivered | failed
    tx_hash: str = ""
    error: str = ""
    created_at: float = field(default_factory=time.time)


_orders: dict[int, OrderState] = {}


# ─── Auth ────────────────────────────────────────────────────────────────────


@web.middleware
async def auth_mw(request: web.Request, handler):
    if request.path == "/health":
        return await handler(request)
    if TOKEN:
        header = request.headers.get("Authorization", "")
        if not header.startswith("Bearer ") or header[7:] != TOKEN:
            return web.json_response({"error": "unauthorized"}, status=401)
    return await handler(request)


# ─── Helpers ────────────────────────────────────────────────────────────────

_USERNAME_RE = re.compile(r"^[a-zA-Z][a-zA-Z0-9_]{4,31}$")


def _api_for(client, qtype: str):
    return client.stars if qtype == "stars" else client.premium


def _search_recipient(client, qtype: str, username: str):
    """Returns the SDK's ApiResult."""
    api = _api_for(client, qtype)
    if qtype == "stars":
        return api.search_stars_recipient(username)
    return api.search_premium_recipient(username)


def _init_request(client, qtype: str, recipient_id: str, quantity: int):
    api = _api_for(client, qtype)
    if qtype == "stars":
        return api.init_buy_stars_request(recipient_id, quantity)
    return api.init_gift_premium_request(recipient_id, quantity)


def _get_link(client, qtype: str, account: dict, req_id: str):
    api = _api_for(client, qtype)
    if qtype == "stars":
        return api.get_buy_stars_link(account, req_id)
    return api.get_gift_premium_link(account, req_id)


# ─── Handlers ────────────────────────────────────────────────────────────────


async def health(_: web.Request):
    return web.json_response({
        "ok": True,
        "creds": _have_creds(),
        "wallet_version": WALLET_VERSION,
    })


async def balance(_: web.Request):
    client, wallet = await _get_clients()
    if wallet is None:
        return web.json_response({"error": "fragment client unavailable"}, status=503)
    try:
        ton_balance = await wallet.get_balance()  # in TON (float)
        return web.json_response({"ton_balance_nano": int(ton_balance * NANO)})
    except Exception as e:  # noqa: BLE001
        log.exception("balance error")
        return web.json_response({"error": str(e)}, status=502)


async def quote(request: web.Request):
    qtype = request.query.get("type", "")
    recipient = (request.query.get("recipient") or "").lstrip("@").strip()
    try:
        qty = int(request.query.get("quantity", "0"))
    except ValueError:
        return web.json_response({"error": "invalid quantity"}, status=400)
    if qtype not in ("stars", "premium") or qty <= 0 or not recipient:
        return web.json_response({"error": "type, quantity and recipient required"}, status=400)

    client, _ = await _get_clients()
    if client is None:
        return web.json_response({"error": "fragment client unavailable"}, status=503)

    try:
        search = _search_recipient(client, qtype, recipient)
        if not search:
            return web.json_response({"error": search.error or "recipient not found"}, status=400)
        init = _init_request(client, qtype, search.data.recipient_id, qty)
        if not init:
            return web.json_response({"error": init.error or "quote failed"}, status=502)
        # Fragment returns the price in TON (decimal string, e.g. "0.3087"),
        # not in nanoTON.
        ton_cost_nano = int(round(float(init.data.amount) * NANO))
        ton_rub_rate = float(client._ctx.ton_rate or 0)
        ton_usd_rate = await _ton_usd_rate()
        return web.json_response({
            "ton_cost_nano": ton_cost_nano,
            "ton_rub_rate": ton_rub_rate,
            "ton_usd_rate": ton_usd_rate,
        })
    except Exception as e:  # noqa: BLE001
        log.exception("quote error")
        return web.json_response({"error": str(e)}, status=502)


async def username_check(request: web.Request):
    u = (request.query.get("u") or "").lstrip("@").strip()
    qtype = request.query.get("type", "stars")
    if not _USERNAME_RE.match(u):
        return web.json_response({"ok": False, "username": u, "reason": "invalid username format"})
    if qtype not in ("stars", "premium"):
        qtype = "stars"

    client, _ = await _get_clients()
    if client is None:
        # Fallback: format-valid username, no enrichment
        return web.json_response({"ok": True, "username": u})

    try:
        res = _search_recipient(client, qtype, u)
        if not res:
            return web.json_response({"ok": False, "username": u, "reason": res.error or "user not found"})
        d = res.data
        return web.json_response({
            "ok": True,
            "username": u,
            "display_name": d.name or "",
            "photo_url": d.photo_url or "",
        })
    except Exception as e:  # noqa: BLE001
        log.exception("username check error")
        return web.json_response({"ok": False, "username": u, "reason": str(e)})


async def purchase(request: web.Request):
    body = await request.json()
    order_id = int(body.get("order_id") or 0)
    qtype = body.get("type") or ""
    recipient = (body.get("recipient") or "").lstrip("@")
    qty = int(body.get("quantity") or 0)

    if order_id <= 0 or qtype not in ("stars", "premium") or not recipient or qty <= 0:
        return web.json_response({"error": "invalid params"}, status=400)

    client, wallet = await _get_clients()
    if client is None or wallet is None:
        return web.json_response({"error": "fragment client unavailable"}, status=503)

    state = _orders.setdefault(order_id, OrderState())
    if state.status == "delivered":
        return web.json_response({"status": "delivered", "tx_hash": state.tx_hash})
    state.status = "processing"

    try:
        search = _search_recipient(client, qtype, recipient)
        if not search:
            raise RuntimeError(search.error or "recipient not found")
        init = _init_request(client, qtype, search.data.recipient_id, qty)
        if not init:
            raise RuntimeError(init.error or "init request failed")
        link = _get_link(client, qtype, wallet.get_wallet_data(), init.data.req_id)
        if not link:
            raise RuntimeError(link.error or "get link failed")

        tx_hashes: list[str] = []
        for msg in link.data.messages:
            tx = await wallet.process_transaction(msg.address, msg.amount, msg.payload)
            if tx:
                tx_hashes.append(str(tx))
        state.status = "delivered"
        state.tx_hash = ",".join(tx_hashes)
        return web.json_response({"status": "delivered", "tx_hash": state.tx_hash})
    except Exception as e:  # noqa: BLE001
        log.exception("purchase failed order=%s", order_id)
        state.status = "failed"
        state.error = str(e)
        return web.json_response({"status": "failed", "error": str(e)}, status=502)


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
