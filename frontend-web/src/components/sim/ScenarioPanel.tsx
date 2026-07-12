import { useState } from "react";
import { sim, type Scenario } from "../../api/sim";
import type { Side } from "../../api/types";
import { LineChart } from "../charts/LineChart";

const money = (v: number) =>
  v.toLocaleString(undefined, { style: "currency", currency: "USD", maximumFractionDigits: 0 });

export function ScenarioPanel({ symbol }: { symbol: string }) {
  const [side, setSide] = useState<Side>("BUY");
  const [quantity, setQuantity] = useState(500);
  const [shockPct, setShockPct] = useState(-10);
  const [res, setRes] = useState<Scenario | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const run = async () => {
    setBusy(true);
    setErr(null);
    try {
      setRes(await sim.scenario({ symbol, side, quantity, shockPct, fromFrac: 0.5 }));
    } catch {
      setErr("analytics service unavailable");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="card sim-panel">
      <div className="card-head">
        <h2>Shock Scenario</h2>
        <span className="muted">what-if</span>
      </div>
      <p className="muted small">
        Apply a one-off price shock halfway through {symbol}'s history and see the
        impact on a held {quantity}-share {side.toLowerCase()} position.
      </p>

      <div className="sim-controls">
        <div className="seg">
          <button className={`seg-btn buy ${side === "BUY" ? "active" : ""}`} onClick={() => setSide("BUY")}>Long</button>
          <button className={`seg-btn sell ${side === "SELL" ? "active" : ""}`} onClick={() => setSide("SELL")}>Short</button>
        </div>
        <label>
          Shares
          <input type="number" min={1} value={quantity} onChange={(e) => setQuantity(+e.target.value)} />
        </label>
        <label>
          Shock {shockPct}%
          <input type="range" min={-40} max={40} step={1} value={shockPct} onChange={(e) => setShockPct(+e.target.value)} />
        </label>
        <button className="btn primary" onClick={run} disabled={busy}>
          {busy ? "…" : "Run scenario"}
        </button>
      </div>

      {err && <p className="error small">{err}</p>}

      {res && (
        <>
          <LineChart
            series={[
              { values: res.base.curve.map((p) => p.pnl), color: "#7c828e", label: "Base P&L" },
              { values: res.shocked.curve.map((p) => p.pnl), color: shockPct < 0 ? "#ef6a6a" : "#34d399", label: `Shocked ${shockPct}%` },
            ]}
            xLabels={[res.base.curve[0]?.date ?? "", res.base.curve.at(-1)?.date ?? ""]}
            format={money}
          />
          <div className="stat-grid">
            <Stat label="Base P&L" value={money(res.base.finalPnl)} good={res.base.finalPnl >= 0} />
            <Stat label="Shocked P&L" value={money(res.shocked.finalPnl)} good={res.shocked.finalPnl >= 0} />
            <Stat label="Impact" value={money(res.impactPnl)} good={res.impactPnl >= 0} />
          </div>
          <p className="muted small">shock applied {res.shockDate} · data: {res.source}</p>
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
