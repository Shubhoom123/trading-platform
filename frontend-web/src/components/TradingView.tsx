import { useState } from "react";
import { FillsFeed } from "./FillsFeed";
import { OrderBook } from "./OrderBook";
import { OrderForm } from "./OrderForm";
import { OrderHistory } from "./OrderHistory";

// Live trading against the exchange for the given symbol. The top bar (with the
// symbol picker and Trade/Simulate tabs) lives one level up in App.
export function TradingView({ symbol }: { symbol: string }) {
  // Bumped whenever an order is placed, to refresh dependent panels immediately.
  const [version, setVersion] = useState(0);
  const bump = () => setVersion((v) => v + 1);

  return (
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
  );
}
