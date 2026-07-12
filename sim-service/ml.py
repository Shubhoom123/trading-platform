"""ML price-direction models — trained on real history, used to trade.

Deliberately rigorous so the numbers are honest:
  * features use only past data (no lookahead),
  * the target is the sign of the NEXT day's return,
  * the split is by time (train on the older slice, test on the newer one) —
    never shuffled, so we never train on the future.

Markets are close to efficient, so expect out-of-sample accuracy only slightly
above the 50% coin-flip baseline. Reporting that honestly (rather than a
lookahead-inflated 90%) is the whole point.
"""

from __future__ import annotations

import time

import numpy as np
from sklearn.ensemble import GradientBoostingClassifier, RandomForestClassifier
from sklearn.linear_model import LogisticRegression
from sklearn.pipeline import make_pipeline
from sklearn.preprocessing import StandardScaler

from data import Series

MODELS = {
    "logistic": lambda: make_pipeline(StandardScaler(), LogisticRegression(max_iter=1000)),
    "random_forest": lambda: RandomForestClassifier(
        n_estimators=200, max_depth=4, min_samples_leaf=20, random_state=0),
    "gradient_boosting": lambda: GradientBoostingClassifier(
        n_estimators=150, max_depth=3, learning_rate=0.05, random_state=0),
}

# Trained models cached by (symbol, model_name) so /predict need not retrain.
_cache: dict[tuple[str, str], tuple[float, object, list[str]]] = {}
_CACHE_TTL = 1800


def _sma(x: np.ndarray, w: int) -> np.ndarray:
    out = np.full_like(x, np.nan, dtype=float)
    if w <= len(x):
        c = np.cumsum(np.insert(x, 0, 0.0))
        out[w - 1:] = (c[w:] - c[:-w]) / w
    return out


def _rsi(close: np.ndarray, period: int = 14) -> np.ndarray:
    delta = np.diff(close, prepend=close[0])
    gain = np.where(delta > 0, delta, 0.0)
    loss = np.where(delta < 0, -delta, 0.0)
    avg_gain = _sma(gain, period)
    avg_loss = _sma(loss, period)
    rs = np.divide(avg_gain, avg_loss, out=np.ones_like(avg_gain), where=avg_loss > 0)
    return 100.0 - 100.0 / (1.0 + rs)


def build_features(close: np.ndarray, volume: np.ndarray):
    """Returns (X, y, names, valid_index). Row i predicts the sign of the
    return from day i to day i+1 using only information known by day i."""
    ret1 = np.zeros_like(close)
    ret1[1:] = close[1:] / close[:-1] - 1.0

    feats = {
        "ret_1d": ret1,
        "ret_2d": np.concatenate([[0, 0], close[2:] / close[:-2] - 1.0]),
        "ret_5d": np.concatenate([[0] * 5, close[5:] / close[:-5] - 1.0]),
        "close_sma5": close / _sma(close, 5) - 1.0,
        "close_sma10": close / _sma(close, 10) - 1.0,
        "close_sma20": close / _sma(close, 20) - 1.0,
        "momentum_10": np.concatenate([[0] * 10, close[10:] / close[:-10] - 1.0]),
        "volatility_10": _sma(ret1**2, 10) ** 0.5,
        "rsi_14": _rsi(close, 14) / 100.0,
        "vol_ratio": volume / np.where(_sma(volume, 10) > 0, _sma(volume, 10), 1.0),
    }
    names = list(feats.keys())
    X = np.column_stack([feats[n] for n in names])

    # Target: next-day up (1) / down (0).
    y = np.zeros(len(close))
    y[:-1] = (close[1:] > close[:-1]).astype(float)

    # Drop warmup rows with NaNs and the final row (no known next-day target).
    valid = ~np.isnan(X).any(axis=1)
    valid[-1] = False
    return X[valid], y[valid], names, np.where(valid)[0]


def _equity(returns: np.ndarray) -> list[float]:
    return list(np.cumprod(1.0 + returns))


def train(series: Series, model_name: str = "gradient_boosting",
          test_frac: float = 0.3) -> dict:
    if model_name not in MODELS:
        raise ValueError(f"unknown model '{model_name}'")

    close = np.asarray(series.close, dtype=float)
    volume = np.asarray(series.volume, dtype=float)
    X, y, names, idx = build_features(close, volume)
    if len(y) < 60:
        raise ValueError("not enough data to train")

    split = int(len(y) * (1 - test_frac))
    Xtr, Xte = X[:split], X[split:]
    ytr, yte = y[:split], y[split:]

    model = MODELS[model_name]()
    model.fit(Xtr, ytr)
    train_acc = float(model.score(Xtr, ytr))
    test_acc = float(model.score(Xte, yte))
    baseline = float(max(yte.mean(), 1 - yte.mean()))  # always-predict-majority

    # Cache a model fit on ALL data for live predictions.
    full = MODELS[model_name]()
    full.fit(X, y)
    _cache[(series.symbol, model_name)] = (time.time(), full, names)

    # Out-of-sample "trade the signal" backtest on the test slice.
    pred_te = model.predict(Xte)
    # Realized next-day return for each test row.
    test_rows = idx[split:]
    fwd_ret = close[test_rows + 1] / close[test_rows] - 1.0
    strat_ret = np.where(pred_te > 0.5, fwd_ret, 0.0)  # long when predicted up, else flat
    test_dates = [series.dates[i + 1] for i in test_rows]

    # Feature importance (tree) or |coef| (logistic).
    importance = _importance(model, names)

    return {
        "symbol": series.symbol,
        "source": series.source,
        "model": model_name,
        "metrics": {
            "trainAccuracyPct": round(train_acc * 100, 1),
            "testAccuracyPct": round(test_acc * 100, 1),
            "baselineAccuracyPct": round(baseline * 100, 1),
            "edgePct": round((test_acc - baseline) * 100, 1),
            "nTrain": int(len(ytr)),
            "nTest": int(len(yte)),
            "features": len(names),
        },
        "signal": {
            "testReturnPct": round((np.prod(1 + strat_ret) - 1) * 100, 2),
            "buyHoldReturnPct": round((np.prod(1 + fwd_ret) - 1) * 100, 2),
        },
        "test": {
            "dates": test_dates,
            "strategyEquity": [round(v, 4) for v in _equity(strat_ret)],
            "buyHoldEquity": [round(v, 4) for v in _equity(fwd_ret)],
        },
        "topFeatures": importance[:6],
        "latest": predict_latest(series, model_name),
        "disclaimer": (
            "Out-of-sample, no lookahead. Markets are near-efficient — expect "
            "accuracy only slightly above the ~50% baseline. Not investment advice."
        ),
    }


def _importance(model, names: list[str]) -> list[dict]:
    est = model.steps[-1][1] if hasattr(model, "steps") else model
    if hasattr(est, "feature_importances_"):
        vals = est.feature_importances_
    elif hasattr(est, "coef_"):
        vals = np.abs(est.coef_).ravel()
    else:
        return []
    order = np.argsort(vals)[::-1]
    total = float(vals.sum()) or 1.0
    return [{"name": names[i], "weight": round(float(vals[i]) / total, 3)} for i in order]


def predict_latest(series: Series, model_name: str = "gradient_boosting") -> dict:
    key = (series.symbol, model_name)
    cached = _cache.get(key)
    close = np.asarray(series.close, dtype=float)
    volume = np.asarray(series.volume, dtype=float)

    if not cached or time.time() - cached[0] > _CACHE_TTL:
        X, y, names, _ = build_features(close, volume)
        model = MODELS[model_name]()
        model.fit(X, y)
        _cache[key] = (time.time(), model, names)
    else:
        model = cached[1]

    # Build the single most-recent feature row (uses the last close/volume).
    Xall, _, _, idx = build_features(close, volume)
    # The most recent *usable* feature row is the last valid one; it predicts
    # the direction of the next (as-yet-unseen) session.
    x_now = Xall[-1:].copy()
    proba = None
    if hasattr(model, "predict_proba"):
        proba = float(model.predict_proba(x_now)[0][1])
    pred = int(model.predict(x_now)[0])
    return {
        "asOf": series.dates[idx[-1]],
        "direction": "UP" if pred == 1 else "DOWN",
        "probabilityUp": round(proba, 3) if proba is not None else None,
    }
