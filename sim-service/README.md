# sim-service

A Python (FastAPI) analytics service that turns the platform into a **market
simulator driven by real data**. It fetches real price history and exposes four
"impact" views over it. It's a separate concern from the live exchange, so it's
its own service; the frontend's **Simulate** tab calls it directly.

## Data

Real daily OHLCV from **Stooq** (`stooq.com`) — free, no API key. Results are
cached in-memory (1h TTL). If the fetch fails (offline / rate-limited), it falls
back to a deterministic synthetic series so the API always responds, flagged
`source: "synthetic"` so the UI can say so.

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | liveness |
| GET | `/api/sim/history?symbol=AAPL&lookback=500` | real OHLCV bars |
| POST | `/api/sim/backtest` | MA-crossover strategy P&L (equity curve, Sharpe, drawdown, win rate) |
| POST | `/api/sim/impact` | market impact of a large order (square-root law) |
| POST | `/api/sim/position` | mark-to-market P&L of a held position |
| POST | `/api/sim/scenario` | the same position under a price shock |
| GET | `/api/sim/models` | available ML model types |
| POST | `/api/sim/train` | train a price-direction model + out-of-sample metrics |
| POST | `/api/sim/predict` | latest next-session direction signal |

## The impact models

- **Backtest** — a moving-average-crossover strategy replayed over the real
  series; reports strategy vs. buy-&-hold return, Sharpe, max drawdown, trades.
- **Market impact** — the Almgren square-root law, `ΔP/P ≈ 0.8 · σ · √(Q/ADV)`,
  with `σ` (volatility) and `ADV` estimated from the real data. Returns average
  fill, slippage in bps, price move, and participation rate.
- **Position P&L** — marks a held position to market across the real price path.
- **Scenario** — applies a one-off shock partway through the series and compares
  the position's P&L before vs. after.

## ML price prediction (`ml.py`)

scikit-learn classifiers (logistic regression, random forest, gradient boosting)
predict the sign of the next session's return from technical features (lagged
returns, MA ratios, RSI, momentum, volatility, volume ratio).

The validation is deliberately honest:

- **no lookahead** — each row's features use only data known by that day;
- **time-based split** — train on the older slice, test on the newer one, never
  shuffled;
- **out-of-sample metrics** — test accuracy vs. the majority-class baseline,
  plus a "trade the signal" equity curve vs. buy-&-hold on the test window.

Because equities are near-efficient, test accuracy lands *just around the ~50%
baseline* even when train accuracy is high — the response includes both so the
overfitting gap is visible. This is a feature, not a bug: a model that "predicts
the market" at 90% out-of-sample would be a lookahead bug, not alpha.

The trained signal drives the ML agents in
[`infra/sim/agents.py`](../infra/sim/agents.py).

## Run

```sh
# via compose (recommended — no local Python deps needed):
docker compose -f infra/docker-compose.yml --profile ui up --build -d sim-service
# -> http://localhost:8100/health   ·   docs at /docs

# or locally:
pip install -r requirements.txt
uvicorn app:app --port 8100
```

## Layout

```
data.py       Stooq client + cache + synthetic fallback
analytics.py  backtest / market_impact / position_pnl / scenario (numpy)
app.py        FastAPI wiring + CORS
```

The analytics functions are pure (series in, dict out), so they're easy to test
in isolation — e.g. against the synthetic series, no network required.
