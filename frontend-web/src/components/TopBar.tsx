import { useEffect, useState } from "react";
import type { Tab } from "../App";
import { api } from "../api/client";
import { formatTicks } from "../api/types";
import { useAuth } from "../auth/AuthContext";

export function TopBar({
  tab,
  onTab,
  symbol,
  onSymbol,
}: {
  tab: Tab;
  onTab: (t: Tab) => void;
  symbol: string;
  onSymbol: (s: string) => void;
}) {
  const { email, logout } = useAuth();
  const [balance, setBalance] = useState<number | null>(null);
  const [draft, setDraft] = useState(symbol);

  useEffect(() => setDraft(symbol), [symbol]);

  useEffect(() => {
    let active = true;
    const load = () =>
      api.account().then((a) => active && setBalance(a.balanceTicks)).catch(() => {});
    load();
    const id = window.setInterval(load, 3000);
    return () => {
      active = false;
      window.clearInterval(id);
    };
  }, []);

  return (
    <header className="topbar">
      <div className="brand-row">
        <span className="brand">Trading Terminal</span>
        <nav className="tabs">
          <button className={`tab ${tab === "trade" ? "active" : ""}`} onClick={() => onTab("trade")}>
            Trade
          </button>
          <button className={`tab ${tab === "simulate" ? "active" : ""}`} onClick={() => onTab("simulate")}>
            Simulate
          </button>
          <button className={`tab ${tab === "models" ? "active" : ""}`} onClick={() => onTab("models")}>
            Models
          </button>
        </nav>
        <form
          className="symbol-form"
          onSubmit={(e) => {
            e.preventDefault();
            const s = draft.trim().toUpperCase();
            if (s) onSymbol(s);
          }}
        >
          <input aria-label="symbol" value={draft} onChange={(e) => setDraft(e.target.value)} list="symbols" />
          <datalist id="symbols">
            <option value="AAPL" />
            <option value="MSFT" />
            <option value="GOOG" />
            <option value="NVDA" />
            <option value="TSLA" />
          </datalist>
          <button className="btn small" type="submit">Load</button>
        </form>
      </div>

      <div className="topbar-right">
        {tab === "trade" && (
          <span className="balance">
            Balance <strong>${balance != null ? formatTicks(balance) : "—"}</strong>
          </span>
        )}
        <span className="muted email">{email}</span>
        <button className="btn small ghost" onClick={logout}>Sign out</button>
      </div>
    </header>
  );
}
