"""Quant analytics over real price series.

Four "impact" views, all built on the same real OHLCV data:
  * backtest      — P&L of a moving-average-crossover strategy
  * market_impact — slippage of a large order (square-root impact law)
  * position_pnl  — mark-to-market P&L of a held position
  * scenario      — the same position under a price shock

The math is intentionally standard and citable (Sharpe, max drawdown, the
Almgren square-root impact law) rather than bespoke.
"""

from __future__ import annotations

import numpy as np

from data import Series

TRADING_DAYS = 252


def _sma(x: np.ndarray, window: int) -> np.ndarray:
    if window <= 1:
        return x.copy()
    out = np.full_like(x, np.nan, dtype=float)
    csum = np.cumsum(np.insert(x, 0, 0.0))
    out[window - 1:] = (csum[window:] - csum[:-window]) / window
    return out


def _sharpe(daily_returns: np.ndarray) -> float:
    r = daily_returns[np.isfinite(daily_returns)]
    if r.size < 2 or r.std() == 0:
        return 0.0
    return float(r.mean() / r.std() * np.sqrt(TRADING_DAYS))


def _max_drawdown_pct(equity: np.ndarray) -> float:
    if equity.size == 0:
        return 0.0
    peak = np.maximum.accumulate(equity)
    dd = (equity - peak) / peak
    return float(dd.min() * 100.0)


def backtest(series: Series, fast: int = 20, slow: int = 50,
             capital: float = 100_000.0) -> dict:
    close = np.asarray(series.close, dtype=float)
    dates = series.dates
    if fast >= slow:
        fast, slow = min(fast, slow) // 2 or 5, max(fast, slow)

    sma_f = _sma(close, fast)
    sma_s = _sma(close, slow)

    # Long when the fast average is above the slow one; position is applied to
    # the NEXT day's return (no lookahead).
    signal = np.where(sma_f > sma_s, 1.0, 0.0)
    signal[np.isnan(sma_s)] = 0.0
    position = np.roll(signal, 1)
    position[0] = 0.0

    daily_ret = np.zeros_like(close)
    daily_ret[1:] = close[1:] / close[:-1] - 1.0
    strat_ret = position * daily_ret

    equity = capital * np.cumprod(1.0 + strat_ret)
    buyhold = capital * np.cumprod(1.0 + daily_ret)

    # Trades: crossings of the position series.
    trades = []
    wins = 0
    entry_price = None
    for i in range(1, len(position)):
        if position[i] == 1.0 and position[i - 1] == 0.0:
            entry_price = close[i]
            trades.append({"date": dates[i], "side": "BUY", "price": round(close[i], 2)})
        elif position[i] == 0.0 and position[i - 1] == 1.0 and entry_price is not None:
            pnl = close[i] - entry_price
            if pnl > 0:
                wins += 1
            trades.append({"date": dates[i], "side": "SELL", "price": round(close[i], 2),
                           "pnlPct": round((close[i] / entry_price - 1) * 100, 2)})
            entry_price = None

    round_trips = sum(1 for t in trades if t["side"] == "SELL")
    equity_curve = [{"date": d, "value": round(float(e), 2)} for d, e in zip(dates, equity)]

    return {
        "symbol": series.symbol,
        "source": series.source,
        "params": {"fast": fast, "slow": slow, "capital": capital},
        "equity": equity_curve,
        "buyHoldFinal": round(float(buyhold[-1]), 2),
        "trades": trades[-40:],
        "stats": {
            "totalReturnPct": round(float(equity[-1] / capital - 1) * 100, 2),
            "buyHoldReturnPct": round(float(buyhold[-1] / capital - 1) * 100, 2),
            "sharpe": round(_sharpe(strat_ret), 2),
            "maxDrawdownPct": round(_max_drawdown_pct(equity), 2),
            "trades": round_trips,
            "winRatePct": round(100.0 * wins / round_trips, 1) if round_trips else 0.0,
            "exposurePct": round(float(position.mean()) * 100, 1),
        },
    }


def market_impact(series: Series, side: str, quantity: float,
                  adv_override: float | None = None) -> dict:
    close = np.asarray(series.close, dtype=float)
    vol = np.asarray(series.volume, dtype=float)
    ref = float(close[-1])

    ret = close[1:] / close[:-1] - 1.0
    daily_vol = float(np.std(ret[-60:])) if ret.size else 0.02
    adv = float(adv_override) if adv_override else float(np.mean(vol[-20:]) or 1.0)
    adv = max(adv, 1.0)

    participation = quantity / adv
    # Almgren square-root impact law: ΔP/P ≈ Y · σ · sqrt(Q / ADV).
    Y = 0.8
    impact_frac = Y * daily_vol * np.sqrt(max(participation, 0.0))
    sign = 1.0 if side.upper() == "BUY" else -1.0

    # Average fill absorbs roughly half of the full price move.
    avg_fill = ref * (1.0 + sign * impact_frac / 2.0)
    final_price = ref * (1.0 + sign * impact_frac)

    return {
        "symbol": series.symbol,
        "source": series.source,
        "side": side.upper(),
        "quantity": quantity,
        "refPrice": round(ref, 2),
        "avgFillPrice": round(avg_fill, 4),
        "finalPrice": round(final_price, 4),
        "slippageBps": round(sign * (avg_fill / ref - 1.0) * 1e4, 1),
        "priceMovePct": round(impact_frac * 100.0, 3),
        "participationPct": round(participation * 100.0, 3),
        "adv": round(adv),
        "dailyVolPct": round(daily_vol * 100.0, 2),
        "notional": round(avg_fill * quantity, 2),
        "model": "square-root (Almgren): dP/P = 0.8 · sigma · sqrt(Q/ADV)",
    }


def _position_curve(close: np.ndarray, dates: list[str], entry: float,
                    qty: float, sign: float) -> tuple[list[dict], float]:
    pnl = sign * qty * (close - entry)
    curve = [{"date": d, "pnl": round(float(p), 2)} for d, p in zip(dates, pnl)]
    return curve, float(pnl[-1])


def position_pnl(series: Series, entry: float, quantity: float, side: str) -> dict:
    close = np.asarray(series.close, dtype=float)
    sign = 1.0 if side.upper() == "BUY" else -1.0
    curve, final = _position_curve(close, series.dates, entry, quantity, sign)
    notional = entry * quantity
    return {
        "symbol": series.symbol,
        "source": series.source,
        "side": side.upper(),
        "entry": entry,
        "quantity": quantity,
        "curve": curve,
        "finalPnl": round(final, 2),
        "returnPct": round(final / notional * 100, 2) if notional else 0.0,
        "maxDrawdownPct": round(_max_drawdown_pct(notional + np.asarray([c["pnl"] for c in curve])), 2),
    }


def scenario(series: Series, entry: float, quantity: float, side: str,
             shock_pct: float, from_frac: float = 0.5) -> dict:
    """Apply a one-off price shock partway through the series and compare the
    held position's P&L before vs. after the shock."""
    close = np.asarray(series.close, dtype=float)
    sign = 1.0 if side.upper() == "BUY" else -1.0
    n = len(close)
    idx = max(1, min(n - 1, int(n * from_frac)))

    shocked = close.copy()
    shocked[idx:] = shocked[idx:] * (1.0 + shock_pct / 100.0)

    base_curve, base_final = _position_curve(close, series.dates, entry, quantity, sign)
    shock_curve, shock_final = _position_curve(shocked, series.dates, entry, quantity, sign)

    return {
        "symbol": series.symbol,
        "source": series.source,
        "shockPct": shock_pct,
        "shockDate": series.dates[idx],
        "base": {"curve": base_curve, "finalPnl": round(base_final, 2)},
        "shocked": {"curve": shock_curve, "finalPnl": round(shock_final, 2)},
        "impactPnl": round(shock_final - base_final, 2),
    }
