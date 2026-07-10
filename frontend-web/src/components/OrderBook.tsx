import { formatTicks } from "../api/types";
import { useOrderBook } from "../hooks/useOrderBook";

// Renders bids (green, descending) and asks (red, ascending) with a depth bar
// sized by each level's share of the largest quantity on screen.
export function OrderBook({ symbol }: { symbol: string }) {
  const { book, error } = useOrderBook(symbol);

  const bids = book?.bids ?? [];
  const asks = book?.asks ?? [];
  const maxQty = Math.max(
    1,
    ...bids.map((l) => l.quantity),
    ...asks.map((l) => l.quantity),
  );

  const spread =
    book?.bestBid != null && book?.bestAsk != null
      ? formatTicks(book.bestAsk - book.bestBid)
      : "—";

  return (
    <div className="card book">
      <div className="card-head">
        <h2>Order Book</h2>
        <span className="muted">{symbol}</span>
      </div>

      {error && <p className="error small">{error}</p>}

      <div className="book-cols">
        <div className="book-col">
          <div className="book-row head">
            <span>Price</span>
            <span>Qty</span>
          </div>
          {asks
            .slice()
            .reverse()
            .map((l, i) => (
              <div className="book-row ask" key={`a${i}`}>
                <span
                  className="depth"
                  style={{ width: `${(l.quantity / maxQty) * 100}%` }}
                />
                <span className="price">{formatTicks(l.priceTicks)}</span>
                <span className="qty">{l.quantity}</span>
              </div>
            ))}
          {asks.length === 0 && <div className="book-empty">no asks</div>}
        </div>

        <div className="spread">spread {spread}</div>

        <div className="book-col">
          <div className="book-row head">
            <span>Price</span>
            <span>Qty</span>
          </div>
          {bids.map((l, i) => (
            <div className="book-row bid" key={`b${i}`}>
              <span
                className="depth"
                style={{ width: `${(l.quantity / maxQty) * 100}%` }}
              />
              <span className="price">{formatTicks(l.priceTicks)}</span>
              <span className="qty">{l.quantity}</span>
            </div>
          ))}
          {bids.length === 0 && <div className="book-empty">no bids</div>}
        </div>
      </div>
    </div>
  );
}
