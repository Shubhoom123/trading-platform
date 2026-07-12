import { useState } from "react";
import { sim, type Impact } from "../../api/sim";
import type { Side } from "../../api/types";

export function ImpactPanel({ symbol }: { symbol: string }) {
  const [side, setSide] = useState<Side>("BUY");
  const [quantity, setQuantity] = useState(250_000);
  const [res, setRes] = useState<Impact | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const run = async () => {
    setBusy(true);
    setErr(null);
    try {
      setRes(await sim.impact({ symbol, side, quantity }));
    } catch {
      setErr("analytics service unavailable");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="card sim-panel">
      <div className="card-head">
        <h2>Market Impact</h2>
        <span className="muted">square-root law</span>
      </div>
      <p className="muted small">
        Estimated slippage of a single large order, calibrated to {symbol}'s real
        volatility and average daily volume.
      </p>

      <div className="sim-controls">
        <div className="seg">
          <button className={`seg-btn buy ${side === "BUY" ? "active" : ""}`} onClick={() => setSide("BUY")}>Buy</button>
          <button className={`seg-btn sell ${side === "SELL" ? "active" : ""}`} onClick={() => setSide("SELL")}>Sell</button>
        </div>
        <label>
          Shares
          <input type="number" min={1} step={1000} value={quantity} onChange={(e) => setQuantity(+e.target.value)} />
        </label>
        <button className="btn primary" onClick={run} disabled={busy}>
          {busy ? "…" : "Estimate impact"}
        </button>
      </div>

      {err && <p className="error small">{err}</p>}

      {res && (
        <>
          <div className="impact-head">
            <div>
              <span className="muted small">Avg fill</span>
              <div className="impact-price">${res.avgFillPrice.toFixed(2)}</div>
              <span className="muted small">ref ${res.refPrice.toFixed(2)}</span>
            </div>
            <div className={`impact-slip ${res.slippageBps >= 0 ? "neg" : "pos"}`}>
              {res.slippageBps} bps
              <span className="muted small">slippage</span>
            </div>
          </div>
          <div className="stat-grid">
            <Stat label="Price move" value={`${res.priceMovePct}%`} />
            <Stat label="Participation" value={`${res.participationPct}% ADV`} />
            <Stat label="Daily vol" value={`${res.dailyVolPct}%`} />
            <Stat label="ADV" value={res.adv.toLocaleString()} />
            <Stat label="Notional" value={`$${res.notional.toLocaleString()}`} />
            <Stat label="Data" value={res.source} />
          </div>
          <p className="muted small">{res.model}</p>
        </>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="stat">
      <span className="stat-label">{label}</span>
      <span className="stat-value">{value}</span>
    </div>
  );
}
