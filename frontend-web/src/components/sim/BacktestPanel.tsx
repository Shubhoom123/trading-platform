import { useState } from "react";
import { sim, type Backtest } from "../../api/sim";
import { LineChart } from "../charts/LineChart";

const money = (v: number) =>
  v.toLocaleString(undefined, { style: "currency", currency: "USD", maximumFractionDigits: 0 });

export function BacktestPanel({ symbol }: { symbol: string }) {
  const [fast, setFast] = useState(20);
  const [slow, setSlow] = useState(50);
  const [capital] = useState(100_000);
  const [res, setRes] = useState<Backtest | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const run = async () => {
    setBusy(true);
    setErr(null);
    try {
      setRes(await sim.backtest({ symbol, fast, slow, capital }));
    } catch {
      setErr("analytics service unavailable");
    } finally {
      setBusy(false);
    }
  };

  const s = res?.stats;
  const beatsHold = s ? s.totalReturnPct > s.buyHoldReturnPct : false;

  return (
    <div className="card sim-panel">
      <div className="card-head">
        <h2>Strategy Backtest</h2>
        <span className="muted">MA crossover</span>
      </div>
      <p className="muted small">
        A moving-average crossover strategy replayed over {symbol}'s real price
        history — the P&amp;L it would have produced.
      </p>

      <div className="sim-controls">
        <label>
          Fast MA
          <input type="number" min={2} value={fast} onChange={(e) => setFast(+e.target.value)} />
        </label>
        <label>
          Slow MA
          <input type="number" min={3} value={slow} onChange={(e) => setSlow(+e.target.value)} />
        </label>
        <button className="btn primary" onClick={run} disabled={busy}>
          {busy ? "…" : "Run backtest"}
        </button>
      </div>

      {err && <p className="error small">{err}</p>}

      {res && s && (
        <>
          <LineChart
            series={[{ values: res.equity.map((p) => p.value), color: "#34d399", label: "Strategy equity", fill: true }]}
            xLabels={[res.equity[0]?.date ?? "", res.equity.at(-1)?.date ?? ""]}
            format={money}
          />
          <div className="stat-grid">
            <Stat label="Strategy return" value={`${s.totalReturnPct}%`} good={s.totalReturnPct >= 0} />
            <Stat label="Buy & hold" value={`${s.buyHoldReturnPct}%`} good={s.buyHoldReturnPct >= 0} />
            <Stat label="Sharpe" value={`${s.sharpe}`} good={s.sharpe >= 1} />
            <Stat label="Max drawdown" value={`${s.maxDrawdownPct}%`} good={s.maxDrawdownPct > -20} />
            <Stat label="Round trips" value={`${s.trades}`} />
            <Stat label="Win rate" value={`${s.winRatePct}%`} good={s.winRatePct >= 50} />
          </div>
          <p className="muted small">
            {beatsHold ? "✓ Beat" : "✗ Trailed"} buy &amp; hold ·{" "}
            {s.exposurePct}% time in market · data: {res.source}
          </p>
        </>
      )}
    </div>
  );
}

function Stat({ label, value, good }: { label: string; value: string; good?: boolean }) {
  return (
    <div className="stat">
      <span className="stat-label">{label}</span>
      <span className={`stat-value ${good === undefined ? "" : good ? "pos" : "neg"}`}>{value}</span>
    </div>
  );
}
