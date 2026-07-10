import { useState } from "react";
import { FillsFeed } from "./FillsFeed";
import { OrderBook } from "./OrderBook";
import { OrderForm } from "./OrderForm";
import { OrderHistory } from "./OrderHistory";
import { TopBar } from "./TopBar";

export function TradingView() {
  const [symbol, setSymbol] = useState("AAPL");
  // Bumped whenever an order is placed, to refresh dependent panels immediately.
  const [version, setVersion] = useState(0);
  const bump = () => setVersion((v) => v + 1);

  return (
    <div className="app">
      <TopBar symbol={symbol} onSymbol={setSymbol} version={version} />

      <main className="grid">
        <section className="col-left">
          <OrderBook symbol={symbol} />
        </section>

        <section className="col-mid">
          <OrderForm symbol={symbol} onPlaced={bump} />
          <FillsFeed symbol={symbol} />
        </section>

        <section className="col-right">
          <OrderHistory version={version} />
        </section>
      </main>
    </div>
  );
}
