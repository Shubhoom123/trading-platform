#!/usr/bin/env python3
"""ML-driven market agents.

Bots that trade the live exchange using the ML price-direction model:

  * ML-signal bots — periodically ask the analytics service to predict the
    symbol's next move (a model trained on real history) and trade in that
    direction; conviction scales with the model's probability.
  * Market-maker bots — quote a two-sided book around a mid anchored to the
    real last price, so the book stays populated and realistic.
  * Noise bots — small random orders, the ever-present chatter of a real tape.

The mid drifts with the ML signal, so the platform's price trends the way the
model predicts — a live, real-data-anchored, ML-influenced market the user can
trade alongside. Standard-library only (urllib); runs on host or in a container.

Honest note: the underlying prediction is barely better than a coin flip
(markets are near-efficient), so the bots produce *plausible* flow, not a money
printer. That's the point.
"""

import json
import os
import random
import time
import urllib.error
import urllib.request

GATEWAY = os.environ.get("GATEWAY", "http://localhost:8090")
SIM_SERVICE = os.environ.get("SIM_SERVICE", "http://localhost:8100")
SYMBOL = os.environ.get("SYMBOL", "AAPL")
MODEL = os.environ.get("MODEL", "gradient_boosting")
ML_BOTS = int(os.environ.get("ML_BOTS", "5"))
MM_BOTS = int(os.environ.get("MM_BOTS", "2"))
NOISE_BOTS = int(os.environ.get("NOISE_BOTS", "3"))
INTERVAL = float(os.environ.get("INTERVAL", "0.4"))
PREDICT_REFRESH = float(os.environ.get("PREDICT_REFRESH", "45"))


def log(m: str) -> None:
    print(f"[agents] {m}", flush=True)


def http(base: str, path: str, body=None, token=None, method="GET"):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(base + path, data=data, method=method)
    if data is not None:
        req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            raw = r.read().decode()
            return r.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as e:
        return e.code, None
    except Exception:
        return 0, None


def ensure_bot(i: int) -> str:
    email, pw = f"agent{i}@sim.local", f"sim-agent-password-{i}"
    st, body = http(GATEWAY, "/api/auth/register", {"email": email, "password": pw}, method="POST")
    if st == 201 and body:
        return body["accessToken"]
    st, body = http(GATEWAY, "/api/auth/login", {"email": email, "password": pw}, method="POST")
    if st == 200 and body:
        return body["accessToken"]
    raise RuntimeError(f"auth failed for {email} ({st})")


def place(token: str, side: str, price: float, qty: int) -> None:
    http(GATEWAY, "/api/orders",
         {"symbol": SYMBOL, "side": side, "type": "LIMIT",
          "price": round(max(price, 0.01), 2), "quantity": qty},
         token=token, method="POST")


def reference_price() -> float:
    st, body = http(SIM_SERVICE, f"/api/sim/history?symbol={SYMBOL}&lookback=5")
    if st == 200 and body and body.get("bars"):
        return float(body["bars"][-1]["close"])
    return 100.0


def fetch_prediction() -> tuple[str, float]:
    st, body = http(SIM_SERVICE, "/api/sim/predict",
                    {"symbol": SYMBOL, "model": MODEL}, method="POST")
    if st == 200 and body:
        return body.get("direction", "UP"), float(body.get("probabilityUp") or 0.5)
    return "UP", 0.5


def main() -> int:
    total = ML_BOTS + MM_BOTS + NOISE_BOTS
    log(f"gateway={GATEWAY} sim={SIM_SERVICE} symbol={SYMBOL} model={MODEL} bots={total}")

    tokens = []
    for i in range(total):
        for attempt in range(30):
            try:
                tokens.append(ensure_bot(i))
                break
            except Exception as e:
                if attempt == 29:
                    log(f"giving up on agent{i}: {e}")
                    return 1
                time.sleep(2)
    ml_tokens = tokens[:ML_BOTS]
    mm_tokens = tokens[ML_BOTS:ML_BOTS + MM_BOTS]
    noise_tokens = tokens[ML_BOTS + MM_BOTS:]
    log(f"{len(tokens)} agents ready ({len(ml_tokens)} ML / {len(mm_tokens)} MM / {len(noise_tokens)} noise)")

    mid = reference_price()
    direction, prob_up = fetch_prediction()
    last_predict = time.time()
    log(f"reference≈{mid:.2f}  prediction={direction} p(up)={prob_up:.2f}")

    ticks = 0
    while True:
        # Refresh the ML signal periodically (training-backed, so not every tick).
        if time.time() - last_predict > PREDICT_REFRESH:
            direction, prob_up = fetch_prediction()
            last_predict = time.time()
            log(f"prediction refreshed: {direction} p(up)={prob_up:.2f}  mid≈{mid:.2f}")

        conviction = prob_up - 0.5  # in [-0.5, 0.5]
        # Mid drifts with the signal (bounded) plus a little noise.
        mid *= 1.0 + conviction * 0.004 + random.gauss(0, 0.0015)
        mid = max(1.0, mid)

        roll = random.random()
        if mm_tokens and roll < 0.4:
            # Market maker: quote both sides to keep the book two-sided.
            t = random.choice(mm_tokens)
            spread = random.uniform(0.05, 0.30)
            place(t, "BUY", mid - spread, random.randint(2, 15))
            place(t, "SELL", mid + spread, random.randint(2, 15))
        elif ml_tokens and roll < 0.8:
            # ML trader: lean in the predicted direction, conviction-weighted.
            t = random.choice(ml_tokens)
            aggressive = random.random() < (0.4 + abs(conviction))
            if direction == "UP":
                price = mid + (0.15 if aggressive else -random.uniform(0.05, 0.4))
                place(t, "BUY", price, random.randint(1, 10))
            else:
                price = mid - (0.15 if aggressive else -random.uniform(0.05, 0.4))
                place(t, "SELL", price, random.randint(1, 10))
        elif noise_tokens:
            # Noise trader: small order on a random side near the mid.
            t = random.choice(noise_tokens)
            side = "BUY" if random.random() < 0.5 else "SELL"
            off = random.uniform(-0.25, 0.25)
            place(t, side, mid + off, random.randint(1, 5))

        ticks += 1
        if ticks % 50 == 0:
            log(f"{ticks} orders · mid≈{mid:.2f} · signal={direction}({prob_up:.2f})")
        time.sleep(INTERVAL)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        pass
