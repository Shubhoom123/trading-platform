import { useState } from "react";
import { LoginPage } from "./components/LoginPage";
import { ModelsView } from "./components/ModelsView";
import { SimulationView } from "./components/SimulationView";
import { TopBar } from "./components/TopBar";
import { TradingView } from "./components/TradingView";
import { useAuth } from "./auth/AuthContext";

export type Tab = "trade" | "simulate" | "models";

export default function App() {
  const { authed } = useAuth();
  const [tab, setTab] = useState<Tab>("trade");
  // Symbol is shared across all views, so switching tabs keeps your ticker.
  const [symbol, setSymbol] = useState("AAPL");

  if (!authed) return <LoginPage />;

  return (
    <div className="app">
      <TopBar tab={tab} onTab={setTab} symbol={symbol} onSymbol={setSymbol} />
      {tab === "trade" && <TradingView symbol={symbol} />}
      {tab === "simulate" && <SimulationView symbol={symbol} />}
      {tab === "models" && <ModelsView symbol={symbol} />}
    </div>
  );
}
