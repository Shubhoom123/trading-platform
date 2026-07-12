"""Real market data access.

Fetches daily OHLCV from Stooq (free, no API key) and caches it in-memory with
a short TTL. If the fetch fails (offline / rate limited), falls back to a
deterministic synthetic series so the API always responds — clearly flagged as
synthetic so the UI can say so.
"""

from __future__ import annotations

import csv
import io
import math
import random
import time
import urllib.request
from dataclasses import dataclass

STOOQ_URL = "https://stooq.com/q/d/l/?s={ticker}&i=d"
CACHE_TTL_SECONDS = 3600


@dataclass
class Series:
    symbol: str
    dates: list[str]
    open: list[float]
    high: list[float]
    low: list[float]
    close: list[float]
    volume: list[float]
    source: str  # "stooq" or "synthetic"


_cache: dict[str, tuple[float, Series]] = {}


def _stooq_ticker(symbol: str) -> str:
    s = symbol.strip().lower()
    # Stooq expects an exchange suffix; default US equities to ".us".
    return s if "." in s else f"{s}.us"


def _fetch_stooq(symbol: str) -> Series:
    url = STOOQ_URL.format(ticker=_stooq_ticker(symbol))
    req = urllib.request.Request(url, headers={"User-Agent": "trading-platform-sim/1.0"})
    with urllib.request.urlopen(req, timeout=10) as resp:
        text = resp.read().decode("utf-8")

    reader = csv.DictReader(io.StringIO(text))
    dates, o, h, l, c, v = [], [], [], [], [], []
    for row in reader:
        # Stooq returns "N/D" for missing cells and a header we can trust.
        if not row.get("Close") or row["Close"] == "N/D":
            continue
        dates.append(row["Date"])
        o.append(float(row["Open"]))
        h.append(float(row["High"]))
        l.append(float(row["Low"]))
        c.append(float(row["Close"]))
        v.append(float(row.get("Volume") or 0))

    if len(c) < 30:
        raise ValueError("stooq returned too little data")
    return Series(symbol.upper(), dates, o, h, l, c, v, "stooq")


def _synthetic(symbol: str, days: int = 500) -> Series:
    # Deterministic per-symbol geometric random walk with mild drift, so the
    # demo still works offline and is reproducible.
    seed = abs(hash(symbol.upper())) % (2**32)
    rng = random.Random(seed)
    price = 50 + (seed % 150)
    dates, o, h, l, c, v = [], [], [], [], [], []
    t = time.time() - days * 86400
    for _ in range(days):
        drift = 0.0003
        shock = rng.gauss(0, 0.015)
        open_ = price
        price = max(1.0, price * math.exp(drift + shock))
        hi = max(open_, price) * (1 + abs(rng.gauss(0, 0.004)))
        lo = min(open_, price) * (1 - abs(rng.gauss(0, 0.004)))
        dates.append(time.strftime("%Y-%m-%d", time.gmtime(t)))
        o.append(round(open_, 2)); h.append(round(hi, 2)); l.append(round(lo, 2))
        c.append(round(price, 2)); v.append(float(rng.randint(2_000_000, 20_000_000)))
        t += 86400
    return Series(symbol.upper(), dates, o, h, l, c, v, "synthetic")


def get_series(symbol: str, lookback: int = 500) -> Series:
    key = symbol.upper()
    now = time.time()
    cached = _cache.get(key)
    if cached and now - cached[0] < CACHE_TTL_SECONDS:
        series = cached[1]
    else:
        try:
            series = _fetch_stooq(symbol)
        except Exception:
            series = _synthetic(symbol)
        _cache[key] = (now, series)

    # Return only the most recent `lookback` bars.
    if lookback and len(series.close) > lookback:
        s = series
        n = lookback
        series = Series(s.symbol, s.dates[-n:], s.open[-n:], s.high[-n:],
                        s.low[-n:], s.close[-n:], s.volume[-n:], s.source)
    return series
