"""Analytics API — real market data + impact models.

A small FastAPI service that fetches real price history (Stooq) and exposes the
four impact views. It's a separate concern from the live exchange, so it's its
own service; the frontend calls it directly (CORS-enabled).
"""

from __future__ import annotations

import analytics
import ml
from data import get_series
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

app = FastAPI(title="Trading Platform — Simulation & Impact API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
def health() -> dict:
    return {"status": "ok"}


@app.get("/api/sim/history")
def history(symbol: str = "AAPL", lookback: int = 500) -> dict:
    s = get_series(symbol, lookback)
    bars = [
        {"date": d, "open": o, "high": h, "low": l, "close": c, "volume": v}
        for d, o, h, l, c, v in zip(s.dates, s.open, s.high, s.low, s.close, s.volume)
    ]
    return {"symbol": s.symbol, "source": s.source, "bars": bars}


class BacktestReq(BaseModel):
    symbol: str = "AAPL"
    fast: int = 20
    slow: int = 50
    capital: float = 100_000.0
    lookback: int = 500


@app.post("/api/sim/backtest")
def backtest(req: BacktestReq) -> dict:
    s = get_series(req.symbol, req.lookback)
    return analytics.backtest(s, req.fast, req.slow, req.capital)


class ImpactReq(BaseModel):
    symbol: str = "AAPL"
    side: str = "BUY"
    quantity: float = 100_000
    adv: float | None = None
    lookback: int = 500


@app.post("/api/sim/impact")
def impact(req: ImpactReq) -> dict:
    s = get_series(req.symbol, req.lookback)
    return analytics.market_impact(s, req.side, req.quantity, req.adv)


class PositionReq(BaseModel):
    symbol: str = "AAPL"
    side: str = "BUY"
    quantity: float = 100
    entry: float | None = None
    lookback: int = 500


@app.post("/api/sim/position")
def position(req: PositionReq) -> dict:
    s = get_series(req.symbol, req.lookback)
    entry = req.entry if req.entry is not None else s.close[0]
    return analytics.position_pnl(s, entry, req.quantity, req.side)


class ScenarioReq(BaseModel):
    symbol: str = "AAPL"
    side: str = "BUY"
    quantity: float = 100
    entry: float | None = None
    shockPct: float = -10.0
    fromFrac: float = 0.5
    lookback: int = 500


@app.post("/api/sim/scenario")
def scenario(req: ScenarioReq) -> dict:
    s = get_series(req.symbol, req.lookback)
    entry = req.entry if req.entry is not None else s.close[0]
    return analytics.scenario(s, entry, req.quantity, req.side, req.shockPct, req.fromFrac)


# --- ML: train price-direction models and predict -------------------------

@app.get("/api/sim/models")
def models() -> dict:
    return {"models": list(ml.MODELS.keys())}


class TrainReq(BaseModel):
    symbol: str = "AAPL"
    model: str = "gradient_boosting"
    lookback: int = 600


@app.post("/api/sim/train")
def train(req: TrainReq) -> dict:
    s = get_series(req.symbol, req.lookback)
    return ml.train(s, req.model)


class PredictReq(BaseModel):
    symbol: str = "AAPL"
    model: str = "gradient_boosting"
    lookback: int = 600


@app.post("/api/sim/predict")
def predict(req: PredictReq) -> dict:
    s = get_series(req.symbol, req.lookback)
    return {"symbol": s.symbol, "model": req.model, "source": s.source,
            **ml.predict_latest(s, req.model)}
