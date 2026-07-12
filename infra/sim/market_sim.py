#!/usr/bin/env python3
"""Market simulator — seeds the exchange with lifelike order flow.

The platform is a real matching engine with no built-in liquidity, so a fresh
book is empty. This bot logs in a few accounts and continuously:

  * random-walks a mid price,
  * posts resting limit orders on both sides around the mid (builds the book),
  * periodically sends an aggressive crossing order (generates trades/fills).

The result: a populated order book, a live fills stream, and moving prices in
the UI. Standard-library only (urllib) so it needs no dependencies.

Run on the host (stack already up):
    python3 infra/sim/market_sim.py
Or as a container:  docker compose -f infra/docker-compose.yml --profile sim up
"""

import json
import os
import random
import sys
import time
import urllib.error
import urllib.request

GATEWAY = os.environ.get("GATEWAY", "http://localhost:8090")
SYMBOL = os.environ.get("SYMBOL", "AAPL")
N_BOTS = int(os.environ.get("BOTS", "3"))
INTERVAL = float(os.environ.get("INTERVAL", "0.4"))
START_PRICE = float(os.environ.get("START_PRICE", "100.00"))


def log(msg: str) -> None:
    print(f"[sim] {msg}", flush=True)


def http(path: str, body=None, token=None, method="GET"):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(GATEWAY + path, data=data, method=method)
    if data is not None:
        req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=10) as r:
            raw = r.read().decode()
            return r.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as e:
        return e.code, None


def ensure_bot(email: str, password: str) -> str:
    """Register the bot, or log in if it already exists. Returns an access token."""
    status, body = http("/api/auth/register", {"email": email, "password": password},
                        method="POST")
    if status == 201 and body:
        return body["accessToken"]
    status, body = http("/api/auth/login", {"email": email, "password": password},
                        method="POST")
    if status == 200 and body:
        return body["accessToken"]
    raise RuntimeError(f"could not authenticate {email} (status {status})")


def place(token: str, side: str, price: float, qty: int) -> None:
    http("/api/orders",
         {"symbol": SYMBOL, "side": side, "type": "LIMIT",
          "price": round(price, 2), "quantity": qty},
         token=token, method="POST")


def main() -> int:
    log(f"gateway={GATEWAY} symbol={SYMBOL} bots={N_BOTS}")
    tokens = []
    for i in range(N_BOTS):
        for attempt in range(30):
            try:
                tokens.append(ensure_bot(f"bot{i}@sim.local", f"sim-bot-password-{i}"))
                break
            except Exception as e:
                if attempt == 29:
                    log(f"giving up on bot{i}: {e}")
                    return 1
                time.sleep(2)  # API may still be starting
    log(f"{len(tokens)} bots ready — seeding {SYMBOL}…")

    mid = START_PRICE
    ticks = 0
    while True:
        # Random walk the mid, gently mean-reverting toward the start price.
        mid += random.uniform(-0.05, 0.05) + (START_PRICE - mid) * 0.01
        mid = max(1.0, mid)
        token = random.choice(tokens)

        if random.random() < 0.30:
            # Aggressive order that crosses the resting book -> a trade.
            if random.random() < 0.5:
                place(token, "BUY", mid + 0.15, random.randint(1, 8))
            else:
                place(token, "SELL", mid - 0.15, random.randint(1, 8))
        else:
            # Passive quote a few ticks off the mid -> builds depth.
            offset = round(random.uniform(0.02, 0.60), 2)
            if random.random() < 0.5:
                place(token, "BUY", mid - offset, random.randint(1, 12))
            else:
                place(token, "SELL", mid + offset, random.randint(1, 12))

        ticks += 1
        if ticks % 50 == 0:
            log(f"{ticks} orders sent, mid≈{mid:.2f}")
        time.sleep(INTERVAL)


if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        sys.exit(0)
