import { useEffect, useState } from "react";
import { sim, type MlTrain } from "../api/sim";
import { LineChart } from "./charts/LineChart";

export function ModelsView({ symbol }: { symbol: string }) {
  const [models, setModels] = useState<string[]>([]);
  const [model, setModel] = useState("gradient_boosting");
  const [res, setRes] = useState<MlTrain | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    sim.models().then((m) => setModels(m.models)).catch(() => {});
  }, []);

  const run = async () => {
    setBusy(true);
    setErr(null);
    try {
      setRes(await sim.train({ symbol, model }));
    } catch {
      setErr("analytics service unavailable (is sim-service on :8100?)");
    } finally {
      setBusy(false);
    }
  };

  const m = res?.metrics;
  const beatsBaseline = m ? m.testAccuracyPct > m.baselineAccuracyPct : false;

  return (
    <main className="sim">
      <div className="card">
        <div className="card-head">
          <h2>ML price-direction model — {symbol}</h2>
          {res && (
            <span className={`src-badge ${res.source === "stooq" ? "" : "synthetic"}`}>
              {res.source === "stooq" ? "real data · stooq" : "synthetic data"}
            </span>
          )}
        </div>
        <p className="muted small">
          A classifier trained on {symbol}'s real history to predict the next
          session's direction, then traded out-of-sample. Markets are
          near-efficient, so honest accuracy sits just around the coin-flip
          baseline — the point is a rigorous, no-lookahead pipeline, not a money
          printer.
        </p>

        <div className="sim-controls">
          <label>
            Model
            <select value={model} onChange={(e) => setModel(e.target.value)}>
              {models.map((mm) => (
                <option key={mm} value={mm}>{mm.replace(/_/g, " ")}</option>
              ))}
            </select>
          </label>
          <button className="btn primary" onClick={run} disabled={busy}>
            {busy ? "training…" : "Train & evaluate"}
          </button>
        </div>
        {err && <p className="error small">{err}</p>}
      </div>

      {res && m && (
        <div className="sim-grid">
          <div className="card sim-panel">
            <div className="card-head">
              <h2>Next-session signal</h2>
              <span className="muted">as of {res.latest.asOf}</span>
            </div>
            <div className="signal-big">
              <span className={`signal-dir ${res.latest.direction === "UP" ? "pos" : "neg"}`}>
                {res.latest.direction === "UP" ? "▲ UP" : "▼ DOWN"}
              </span>
              {res.latest.probabilityUp != null && (
                <span className="muted">p(up) = {(res.latest.probabilityUp * 100).toFixed(1)}%</span>
              )}
            </div>
            <div className="stat-grid">
              <Stat label="Test accuracy" value={`${m.testAccuracyPct}%`} good={beatsBaseline} />
              <Stat label="Baseline" value={`${m.baselineAccuracyPct}%`} />
              <Stat label="Edge" value={`${m.edgePct}%`} good={m.edgePct > 0} />
              <Stat label="Train accuracy" value={`${m.trainAccuracyPct}%`} />
              <Stat label="Train / test" value={`${m.nTrain} / ${m.nTest}`} />
              <Stat label="Features" value={`${m.features}`} />
            </div>
            <p className="muted small">
              {m.trainAccuracyPct - m.testAccuracyPct > 15
                ? "Note the train ≫ test gap — the model is overfitting, exactly what out-of-sample testing exposes."
                : "Train and test accuracy are close — little overfitting."}
            </p>
          </div>

          <div className="card sim-panel">
            <div className="card-head">
              <h2>Out-of-sample: trade the signal</h2>
              <span className="muted">test period</span>
            </div>
            <LineChart
              series={[
                { values: res.test.strategyEquity, color: "#34d399", label: `Signal (${res.signal.testReturnPct}%)` },
                { values: res.test.buyHoldEquity, color: "#7c828e", label: `Buy & hold (${res.signal.buyHoldReturnPct}%)` },
              ]}
              xLabels={[res.test.dates[0] ?? "", res.test.dates.at(-1) ?? ""]}
              format={(v) => `${((v - 1) * 100).toFixed(0)}%`}
            />
          </div>

          <div className="card sim-panel">
            <div className="card-head">
              <h2>What the model looks at</h2>
              <span className="muted">feature weight</span>
            </div>
            <div className="feat-list">
              {res.topFeatures.map((f) => (
                <div className="feat" key={f.name}>
                  <span className="feat-name">{f.name.replace(/_/g, " ")}</span>
                  <span className="feat-bar">
                    <span style={{ width: `${Math.min(100, f.weight * 100)}%` }} />
                  </span>
                  <span className="feat-val">{(f.weight * 100).toFixed(0)}%</span>
                </div>
              ))}
            </div>
            <p className="muted small">{res.disclaimer}</p>
          </div>
        </div>
      )}
    </main>
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
