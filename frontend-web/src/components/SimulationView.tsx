import { useEffect, useState } from "react";
import { sim, type History } from "../api/sim";
import { LineChart } from "./charts/LineChart";
import { BacktestPanel } from "./sim/BacktestPanel";
import { ImpactPanel } from "./sim/ImpactPanel";
import { ScenarioPanel } from "./sim/ScenarioPanel";

export function SimulationView({ symbol }: { symbol: string }) {
  const [hist, setHist] = useState<History | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    setHist(null);
    setErr(null);
    sim
      .history(symbol, 500)
      .then((h) => active && setHist(h))
      .catch(() => active && setErr("analytics service unavailable"));
    return () => {
      active = false;
    };
  }, [symbol]);

  const closes = hist?.bars.map((b) => b.close) ?? [];
  const first = hist?.bars[0]?.date ?? "";
  const last = hist?.bars.at(-1)?.date ?? "";
  const changePct =
    closes.length > 1 ? ((closes.at(-1)! / closes[0] - 1) * 100).toFixed(1) : "0";

  return (
    <main className="sim">
      <div className="card price-card">
        <div className="card-head">
          <h2>{symbol} — real price history</h2>
          {hist && (
            <span className={`src-badge ${hist.source}`}>
              {hist.source === "stooq" ? "live data · stooq" : "synthetic (data source offline)"}
            </span>
          )}
        </div>
        {err && <p className="error">{err} — is the analytics service running on :8100?</p>}
        {hist && (
          <>
            <div className="price-summary">
              <span className="price-now">${closes.at(-1)?.toFixed(2)}</span>
              <span className={+changePct >= 0 ? "pos" : "neg"}>
                {+changePct >= 0 ? "▲" : "▼"} {changePct}% over {closes.length} sessions
              </span>
            </div>
            <LineChart
              series={[{ values: closes, color: "#34d399", label: `${symbol} close`, fill: true }]}
              xLabels={[first, last]}
              height={280}
            />
          </>
        )}
        {!hist && !err && <p className="muted">loading {symbol}…</p>}
      </div>

      <div className="sim-grid">
        <BacktestPanel symbol={symbol} />
        <ImpactPanel symbol={symbol} />
        <ScenarioPanel symbol={symbol} />
      </div>
    </main>
  );
}
